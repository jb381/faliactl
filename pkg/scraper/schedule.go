package scraper

import (
	"fmt"
	"io"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// ParseSchedule parses the individual schedule HTML content to extract the courses.
func ParseSchedule(r io.Reader) ([]Course, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, err
	}

	var courses []Course

	// The detailed course info is located within div.event-popover elements
	doc.Find("div.event-popover").Each(func(i int, sel *goquery.Selection) {
		// Header info
		header := sel.Find(".header")
		name := strings.TrimSpace(header.Find("p.title").Text())
		courseType := strings.TrimSpace(header.Find("p.description").Text())

		var dateStr, startTime, endTime, room, groupStr string

		// Content parts
		sel.Find(".content .part").Each(func(j int, part *goquery.Selection) {
			icon, _ := part.Find("img").Attr("src")

			if strings.Contains(icon, "clock") {
				dateStr = strings.TrimSpace(part.Find(".item p.title").Text())
				timeStr := strings.TrimSpace(part.Find(".item p.description").Text())

				// e.g. "08:15 Uhr - 09:45 Uhr"
				timeParts := strings.Split(timeStr, "-")
				if len(timeParts) == 2 {
					startTime = strings.TrimSpace(strings.ReplaceAll(timeParts[0], "Uhr", ""))
					endTime = strings.TrimSpace(strings.ReplaceAll(timeParts[1], "Uhr", ""))
				}
			} else if strings.Contains(icon, "map-marker") {
				room = strings.TrimSpace(part.Find(".item p.title").Text())
			} else if strings.Contains(icon, "group") || strings.Contains(icon, "info-circle") {
				var groups []string
				part.Find(".item").Each(func(k int, item *goquery.Selection) {
					gTitle := strings.TrimSpace(item.Find("p.title").Text())
					groups = append(groups, gTitle)
				})
				groupStr = strings.Join(groups, ", ")
			}
		})

		// Append the course if it has valid time info
		if dateStr != "" && startTime != "" && endTime != "" {
			courses = append(courses, Course{
				Name:      name,
				Type:      courseType,
				DateStr:   dateStr,
				StartTime: startTime,
				EndTime:   endTime,
				Room:      room,
				GroupStr:  groupStr,
			})
		}
	})

	return deduplicateCourses(courses), nil
}

// FetchSchedule downloads and parses the schedule for a given group URL
func (c *Client) FetchSchedule(groupURL string) ([]Course, error) {
	if cachedCourses, ok := readCache(groupURL); ok {
		return cachedCourses, nil
	}

	resp, err := c.Get(groupURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	return ParseSchedule(resp.Body)
}

// deduplicateCourses removes duplicate course entries since the same popover might be listed multiple times if it spans multiple weeks, although usually they have distinct IDs. Adding just in case.
func deduplicateCourses(courses []Course) []Course {
	seen := make(map[string]bool)
	var unique []Course

	for _, c := range courses {
		key := fmt.Sprintf("%s|%s|%s|%s", c.Name, c.DateStr, c.StartTime, c.EndTime)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, c)
		}
	}

	return unique
}

// GetCampusAddress returns the physical street address of an Ostfalia campus based on the room name prefix.
func GetCampusAddress(room string) string {
	// Rooms are sometimes prefixed with cities (e.g. WF-EX-2/127 for Wolfenbüttel Am Exer)
	roomUpper := strings.ToUpper(room)
	if strings.Contains(roomUpper, "SZ") {
		return "Karl-Scharfenberg-Straße 55-57, 38229 Salzgitter"
	} else if strings.Contains(roomUpper, "SUD") {
		return "Herbert-Meyer-Straße 7, 29556 Suderburg"
	} else if strings.Contains(roomUpper, "-EX-") || strings.HasPrefix(roomUpper, "EX") {
		return "Am Exer 2, 38302 Wolfenbüttel"
	}
	// Default to Wolfenbüttel Hauptcampus
	return "Salzdahlumer Str. 46/48, 38302 Wolfenbüttel"
}

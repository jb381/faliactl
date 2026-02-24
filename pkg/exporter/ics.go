package exporter

import (
	"fmt"
	"io"
	"strings"
	"time"

	"faliactl/pkg/scraper"

	ics "github.com/arran4/golang-ical"
)

// GenerateICS creates an ICS file from the slice of courses and writes it to the provided writer
func GenerateICS(courses []scraper.Course, w io.Writer) error {
	cal := ics.NewCalendar()
	cal.SetMethod(ics.MethodPublish)

	// Timezone location for Germany
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		return fmt.Errorf("could not load timezone: %w", err)
	}

	for i, c := range courses {
		// c.DateStr example: "04.03.2026 (Mittwoch)"
		dateParts := strings.Split(c.DateStr, " ")
		if len(dateParts) == 0 {
			continue // Skip malformed dates
		}
		cleanDate := dateParts[0]

		startStr := fmt.Sprintf("%s %s", cleanDate, c.StartTime)
		endStr := fmt.Sprintf("%s %s", cleanDate, c.EndTime)

		layout := "02.01.2006 15:04"

		startTime, err := time.ParseInLocation(layout, startStr, loc)
		if err != nil {
			continue // Skip invalid times
		}

		endTime, err := time.ParseInLocation(layout, endStr, loc)
		if err != nil {
			continue
		}

		event := cal.AddEvent(fmt.Sprintf("%s-%d", startTime.Format("20060102T150405Z"), i))
		event.SetCreatedTime(time.Now())
		event.SetDtStampTime(time.Now())
		event.SetModifiedAt(time.Now())
		event.SetStartAt(startTime)
		event.SetEndAt(endTime)
		event.SetSummary(c.Name)
		event.SetLocation(c.Room)

		description := fmt.Sprintf("Type: %s\nGroup: %s", c.Type, c.GroupStr)
		event.SetDescription(description)
	}

	return cal.SerializeTo(w)
}

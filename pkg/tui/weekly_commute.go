package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"faliactl/pkg/config"
	"faliactl/pkg/scraper"
	"faliactl/pkg/transit"

	"github.com/charmbracelet/huh/spinner"
	"github.com/charmbracelet/lipgloss"
)

// RunWeeklyCommuteTUI generates a 7-day transit itinerary based on Saved Courses
func RunWeeklyCommuteTUI() error {
	cfg, err := config.Load()
	if err != nil || cfg.HomeStationID == "" {
		fmt.Println(errorStyle.Render("Home address is not configured."))
		fmt.Println("Please run 'Settings' from the main menu or 'faliactl config' first.")
		return nil
	}

	if len(cfg.SavedGroupURLs) == 0 || len(cfg.SavedCourses) == 0 {
		fmt.Println(errorStyle.Render("No Saved Courses configured."))
		fmt.Println("Please run 'Settings' -> 'Set Saved Courses' to let Faliactl know what classes you take!")
		return nil
	}

	fmt.Println(accentStyle.Render("Generating Weekly Commute Planner..."))

	client := scraper.NewClient()
	var allCourses []scraper.Course
	var fetchErr error

	// 1. Fetch all schedules from the saved groups (using local cache implicitly)
	_ = spinner.New().
		Title("Checking schedules...").
		Action(func() {
			for _, url := range cfg.SavedGroupURLs {
				groupCourses, cErr := client.FetchSchedule(url)
				if cErr != nil {
					fetchErr = fmt.Errorf("failed to fetch schedule: %w", cErr)
					return
				}
				allCourses = append(allCourses, groupCourses...)
			}
		}).
		Run()

	if fetchErr != nil {
		return fmt.Errorf("failed to fetch schedules: %w", fetchErr)
	}

	// 2. Filter down to only Saved Courses occurring in the next 7 days
	loc, _ := time.LoadLocation("Europe/Berlin")
	now := time.Now().In(loc)
	// Start of today (midnight) so we include courses later today
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	nextWeek := todayStart.AddDate(0, 0, 7)

	existingSavedMap := make(map[string]bool)
	for _, name := range cfg.SavedCourses {
		existingSavedMap[name] = true
	}

	var upcomingCourses []scraper.Course

	for _, c := range allCourses {
		if !existingSavedMap[c.Name] {
			continue
		}

		dateOnly := strings.Split(c.DateStr, " ")[0]
		timeStr := fmt.Sprintf("%s %s", dateOnly, c.StartTime)

		t, parseErr := time.ParseInLocation("02.01.2006 15:04", timeStr, loc)
		// Must be in the future (or later today) AND before next week
		if parseErr == nil && t.After(now) && t.Before(nextWeek) {
			// We only want to commute ONCE per day, to the FIRST class of that day.
			// Let's store them all for now and sort them.
			upcomingCourses = append(upcomingCourses, c)
		}
	}

	if len(upcomingCourses) == 0 {
		fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("\nNo saved classes scheduled for the next 7 days! Enjoy your free time. 🏖️"))
		return nil
	}

	// Sort chronologically
	sort.SliceStable(upcomingCourses, func(i, j int) bool {
		di := strings.Split(upcomingCourses[i].DateStr, " ")[0]
		ti := fmt.Sprintf("%s %s", di, upcomingCourses[i].StartTime)
		t1, _ := time.ParseInLocation("02.01.2006 15:04", ti, loc)

		dj := strings.Split(upcomingCourses[j].DateStr, " ")[0]
		tj := fmt.Sprintf("%s %s", dj, upcomingCourses[j].StartTime)
		t2, _ := time.ParseInLocation("02.01.2006 15:04", tj, loc)

		return t1.Before(t2)
	})

	// Group by Date to only route to the FIRST class each day
	type DailyFirstClass struct {
		Date        string
		Course      scraper.Course
		ArrivalTime time.Time
	}

	var commuteList []DailyFirstClass
	seenDates := make(map[string]bool)

	for _, c := range upcomingCourses {
		dateOnly := strings.Split(c.DateStr, " ")[0]
		if !seenDates[dateOnly] {
			seenDates[dateOnly] = true

			timeStr := fmt.Sprintf("%s %s", dateOnly, c.StartTime)
			arrTime, _ := time.ParseInLocation("02.01.2006 15:04", timeStr, loc)

			commuteList = append(commuteList, DailyFirstClass{
				Date:        dateOnly,
				Course:      c,
				ArrivalTime: arrTime,
			})
		}
	}

	// 3. Calculate all the routes
	type ResolvedCommute struct {
		Date    string
		Course  scraper.Course
		Journey *transit.Journey
		Error   error
	}

	var results []ResolvedCommute
	transitClient := transit.NewClient()

	_ = spinner.New().
		Title("Calculating HAFAS transit routes for the week...").
		Action(func() {
			for _, daily := range commuteList {
				j, jErr := getBestJourney(transitClient, cfg, daily.Course, daily.ArrivalTime)

				results = append(results, ResolvedCommute{
					Date:    daily.Date,
					Course:  daily.Course,
					Journey: j,
					Error:   jErr,
				})
			}
		}).
		Run()

	fmt.Println()

	for _, res := range results {
		dateStr := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Render(res.Date)
		fmt.Printf("--- 📅 %s ---\n", dateStr)
		fmt.Printf("Class: %s (%s @ %s)\n", res.Course.Name, res.Course.Room, res.Course.StartTime)

		if res.Error != nil {
			fmt.Println(errorStyle.Render(fmt.Sprintf("❌ Transit Error: %v\n", res.Error)))
			continue
		}

		firstDepart := res.Journey.Legs[0].Departure
		totalDuration := res.Journey.Legs[len(res.Journey.Legs)-1].Arrival.Sub(firstDepart)

		fmt.Printf("Leave home by: %s\n", errorStyle.Render(firstDepart.Local().Format("15:04")))
		fmt.Printf("Total Travel Time: %d mins\n\n", int(totalDuration.Minutes()))

		for i, leg := range res.Journey.Legs {
			lineName := "Walk🚶"
			if leg.Line != nil {
				lineName = leg.Line.Name
			}

			timeStr := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Render(leg.Departure.Local().Format("15:04"))
			lineStr := lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true).Render(lineName)
			arrStr := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("Arrive: " + leg.Arrival.Local().Format("15:04"))

			fmt.Printf("%d. [%s] %s -> %s (%s)\n", i+1, timeStr, lineStr, leg.Destination.Name, arrStr)
		}
		fmt.Println()
	}

	return nil
}

// Helper to resolve routes
func getBestJourney(client *transit.Client, cfg *config.AppConfig, course scraper.Course, arrivalTime time.Time) (*transit.Journey, error) {
	roomUpper := strings.ToUpper(course.Room)
	var destStationID string

	if strings.HasPrefix(roomUpper, "SZ") {
		destStationID = "991604089" // Salzgitter
	} else if strings.HasPrefix(roomUpper, "SUD") {
		destStationID = "991604106" // Suderburg
	} else if strings.Contains(roomUpper, "EX") {
		destStationID = "891011" // Exer Süd
	} else {
		destStationID = "891097" // Hauptcampus (Salzdahlumer Str.)
	}

	journeys, err := client.FetchJourneysByArrival(cfg.HomeStationID, destStationID, arrivalTime)
	if err != nil {
		return nil, err
	}

	if len(journeys) == 0 {
		return nil, fmt.Errorf("no routes found")
	}

	best := journeys[len(journeys)-1]
	if best.Legs[len(best.Legs)-1].Arrival.After(arrivalTime) {
		best = journeys[0]
	}
	return &best, nil
}

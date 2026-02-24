package tui

import (
	"fmt"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"faliactl/pkg/config"
	"faliactl/pkg/scraper"
	"faliactl/pkg/transit"

	ics "github.com/arran4/golang-ical"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/huh/spinner"
	"github.com/charmbracelet/lipgloss"
)

// ResolvedCommute represents a successfully mapped transit journey for a specific course day
type ResolvedCommute struct {
	Date    string
	Course  scraper.Course
	Journey *transit.Journey
	Error   error
}

// RunWeeklyCommuteTUI generates a transit itinerary based on Saved Courses
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

	var daysStr string
	daysForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("How many days should we plan for?").
				Description("Enter the number of days you want your commute itinerary generated for.").
				Placeholder("7").
				Value(&daysStr).
				Validate(func(v string) error {
					if v == "" {
						return nil // Default to 7
					}
					val, err := strconv.Atoi(v)
					if err != nil || val <= 0 || val > 365 {
						return fmt.Errorf("please enter a valid number between 1 and 365")
					}
					return nil
				}),
		),
	).WithTheme(GetTheme())

	if err := daysForm.Run(); err != nil {
		return err
	}

	days := 7 // default
	if daysStr != "" {
		days, _ = strconv.Atoi(daysStr)
	}

	fmt.Println(accentStyle.Render(fmt.Sprintf("\nGenerating %d-Day Commute Planner...", days)))

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

	// 2. Filter down to only Saved Courses occurring in the given timeframe
	loc, _ := time.LoadLocation("Europe/Berlin")
	now := time.Now().In(loc)
	// Start of today (midnight) so we include courses later today
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	timeHorizon := todayStart.AddDate(0, 0, days)

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
		// Must be in the future (or later today) AND before the horizon
		if parseErr == nil && t.After(now) && t.Before(timeHorizon) {
			// We only want to commute ONCE per day, to the FIRST class of that day.
			// Let's store them all for now and sort them.
			upcomingCourses = append(upcomingCourses, c)
		}
	}

	if len(upcomingCourses) == 0 {
		fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(fmt.Sprintf("\nNo saved classes scheduled for the next %d days! Enjoy your free time. 🏖️", days)))
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

	// 4. Offer to export the generated commute list as an ICS file
	var exportICS bool
	exportForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Export this itinerary to a calendar (.ics) file?").
				Description("Creates a file you can import into Apple Calendar or Google Calendar, complete with tracking links.").
				Value(&exportICS).
				Affirmative("Export").
				Negative("Cancel"),
		),
	).WithTheme(GetTheme())

	if err := exportForm.Run(); err != nil {
		return err
	}

	if exportICS {
		err := exportCommutesToICS(results, cfg.HomeAddress)
		if err != nil {
			fmt.Println(errorStyle.Render(fmt.Sprintf("Failed to export ICS: %v", err)))
		}
	}

	return nil
}

// exportCommutesToICS creates the iCalendar file with Google Maps transit links
func exportCommutesToICS(results []ResolvedCommute, homeAddress string) error {
	cal := ics.NewCalendar()
	cal.SetMethod(ics.MethodPublish)

	for i, res := range results {
		if res.Error != nil {
			continue // skip broken legs
		}

		firstDepart := res.Journey.Legs[0].Departure
		lastArrival := res.Journey.Legs[len(res.Journey.Legs)-1].Arrival

		event := cal.AddEvent(fmt.Sprintf("faliactl-commute-%d", i))
		event.SetCreatedTime(time.Now())
		event.SetDtStampTime(time.Now())
		event.SetModifiedAt(time.Now())
		event.SetStartAt(firstDepart)
		event.SetEndAt(lastArrival)

		event.SetSummary(fmt.Sprintf("🚌 Commute to %s", res.Course.Name))
		event.SetLocation(fmt.Sprintf("Start: %s", res.Journey.Legs[0].Origin.Name))

		// Build a description with all transfers + Map Link
		// Google Maps deep link to generic transit routing for the end destination
		destQuery := url.QueryEscape(res.Journey.Legs[len(res.Journey.Legs)-1].Destination.Name)

		mapsURL := fmt.Sprintf("https://www.google.com/maps/dir/?api=1&origin=%s&destination=%s&travelmode=transit",
			url.QueryEscape(homeAddress),
			destQuery,
		)

		desc := fmt.Sprintf("Live tracking normally available via DB Navigator.\nGoogle Maps Link: %s\n\nJourney Details:\n", mapsURL)
		for j, leg := range res.Journey.Legs {
			lineName := "Walk🚶"
			if leg.Line != nil {
				lineName = leg.Line.Name
			}
			desc += fmt.Sprintf("  %d. [%s] %s -> %s\n", j+1, leg.Departure.Local().Format("15:04"), lineName, leg.Destination.Name)
		}

		desc += fmt.Sprintf("\nArrives at %s in time for %s at %s.", lastArrival.Local().Format("15:04"), res.Course.Name, res.Course.StartTime)
		event.SetDescription(desc)
	}

	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("My_Commutes_%s.ics", timestamp)
	err := os.WriteFile(filename, []byte(cal.Serialize()), 0644)
	if err != nil {
		return fmt.Errorf("could not write ics file: %w", err)
	}

	fmt.Printf("\n✨ Successfully exported commute calendar to: %s\n", filename)
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

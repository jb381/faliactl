package tui

import (
	"fmt"
	"strings"
	"time"

	"faliactl/pkg/config"
	"faliactl/pkg/scraper"
	"faliactl/pkg/transit"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/huh/spinner"
	"github.com/charmbracelet/lipgloss"
)

// RunCourseCommuteTUI launches the interactive experience for routing a specific university course
func RunCourseCommuteTUI() error {
	cfg, err := config.Load()
	if err != nil || cfg.HomeStationID == "" {
		fmt.Println(errorStyle.Render("Home address is not configured."))
		fmt.Println("Please run 'Settings' from the main menu or 'faliactl config' first.")
		return nil
	}

	fmt.Println(accentStyle.Render("Plan Route to Class"))

	client := scraper.NewClient()
	var selectedGroupURLs []string

	var groups []scraper.Group
	_ = spinner.New().
		Title("Fetching available groups...").
		Action(func() {
			groups, err = client.FetchGroups()
		}).
		Run()

	if err != nil {
		return fmt.Errorf("failed to fetch groups: %w", err)
	}

	savedGroupMap := make(map[string]bool)
	if cfg != nil {
		for _, url := range cfg.SavedGroupURLs {
			savedGroupMap[url] = true
		}
	}

	var groupOptions []huh.Option[string]
	for _, g := range groups {
		opt := huh.NewOption(g.Name, g.URL)
		if savedGroupMap[g.URL] {
			opt = opt.Selected(true)
		}
		groupOptions = append(groupOptions, opt)
	}

	groupForm := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select your study group(s)").
				Description("Space = toggle, Enter = confirm. Start typing to filter.").
				Options(groupOptions...).
				Value(&selectedGroupURLs).
				Filterable(true).
				Height(12),
		),
	).WithTheme(GetTheme())

	err = groupForm.Run()
	if err != nil {
		return err
	}

	if len(selectedGroupURLs) == 0 {
		return nil
	}

	var courses []scraper.Course
	var fetchErr error

	_ = spinner.New().
		Title("Fetching schedule...").
		Action(func() {
			for _, url := range selectedGroupURLs {
				groupCourses, cErr := client.FetchSchedule(url)
				if cErr != nil {
					fetchErr = fmt.Errorf("failed to fetch schedule: %w", cErr)
					return
				}
				courses = append(courses, groupCourses...)
			}
		}).
		Run()

	if fetchErr != nil {
		return fmt.Errorf("failed to fetch schedule: %w", fetchErr)
	}

	if len(courses) == 0 {
		fmt.Println(errorStyle.Render("No courses found for this group!"))
		return nil
	}

	// Build the options uniquely identifying each session
	var courseOptions []huh.Option[string]

	// Only show upcoming courses (roughly, since the scraper pulls the current week/semester)
	loc, _ := time.LoadLocation("Europe/Berlin")
	now := time.Now().In(loc)

	existingSavedMap := make(map[string]bool)
	for _, name := range cfg.SavedCourses {
		existingSavedMap[name] = true
	}

	// Since DateStr is "04.03.2026 (Mittwoch)" we need to parse it
	for i, c := range courses {
		// Filter out courses that aren't in the saved list (if the user has a saved list)
		if len(existingSavedMap) > 0 && !existingSavedMap[c.Name] {
			continue
		}

		// "04.03.2026", "08:15"
		dateOnly := strings.Split(c.DateStr, " ")[0]
		timeStr := fmt.Sprintf("%s %s", dateOnly, c.StartTime)

		t, parseErr := time.ParseInLocation("02.01.2006 15:04", timeStr, loc)
		if parseErr == nil && t.After(now) {
			displayStr := fmt.Sprintf("%s - %s @ %s (%s)", dateOnly, c.Name, c.StartTime, c.Room)
			// Store the index as the value so we can retrieve the exact struct
			courseOptions = append(courseOptions, huh.NewOption(displayStr, fmt.Sprintf("%d", i)))
		}
	}

	if len(courseOptions) == 0 {
		fmt.Println(errorStyle.Render("No upcoming courses found in the current schedule window."))
		fmt.Println("This app primarily pulls from the immediate published schedule online.")
		return nil
	}

	var selectedCourseIdxStr string
	courseForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select an upcoming class session").
				Options(courseOptions...).
				Value(&selectedCourseIdxStr).
				Height(12),
		),
	).WithTheme(GetTheme())

	err = courseForm.Run()
	if err != nil {
		return err
	}

	var selectedCourse scraper.Course
	// Find it
	for i, c := range courses {
		if fmt.Sprintf("%d", i) == selectedCourseIdxStr {
			selectedCourse = c
			break
		}
	}

	return calculateRouteToClass(selectedCourse, cfg)
}

func calculateRouteToClass(course scraper.Course, cfg *config.AppConfig) error {
	loc, _ := time.LoadLocation("Europe/Berlin")
	dateOnly := strings.Split(course.DateStr, " ")[0]
	timeStr := fmt.Sprintf("%s %s", dateOnly, course.StartTime)
	arrivalTime, err := time.ParseInLocation("02.01.2006 15:04", timeStr, loc)
	if err != nil {
		return fmt.Errorf("could not parse class start time: %w", err)
	}

	// Determine destination campus based on Room prefix or context
	// WF = Wolfenbuettel, SZ = Salzgitter, SUD = Suderburg
	var destStationID string
	var destName string
	roomUpper := strings.ToUpper(course.Room)

	if strings.HasPrefix(roomUpper, "SZ") {
		destStationID = "991604089" // Salzgitter
		destName = "Ostfalia Salzgitter"
	} else if strings.HasPrefix(roomUpper, "SUD") {
		destStationID = "991604106" // Suderburg
		destName = "Ostfalia Suderburg"
	} else if strings.Contains(roomUpper, "EX") {
		destStationID = "891011" // Exer Süd
		destName = "Ostfalia Am Exer"
	} else {
		// Default to Wolfenbüttel Hauptcampus (Salzdahlumer Straße)
		destStationID = "891097"
		destName = "Ostfalia Hauptcampus (Salzdahlumer Str.)"
	}

	transitClient := transit.NewClient()
	var journeys []transit.Journey
	var fetchErr error

	_ = spinner.New().
		Title(fmt.Sprintf("Calculating route from %s to %s for %s...", cfg.HomeAddress, destName, course.StartTime)).
		Action(func() {
			journeys, fetchErr = transitClient.FetchJourneysByArrival(cfg.HomeStationID, destStationID, arrivalTime)
		}).
		Run()

	if fetchErr != nil {
		return fmt.Errorf("could not calculate journey: %w", fetchErr)
	}

	if len(journeys) == 0 {
		fmt.Println(errorStyle.Render("No valid transit routes found that arrive before class starts!"))
		return nil
	}

	bestJourney := journeys[len(journeys)-1] // HAFAS returns results sorted by arrival time ASCENDING. The last one is the closest purely to the required arrival edge without being late.

	// Check if even the closest one is actually fundamentally broken/late though.
	if bestJourney.Legs[len(bestJourney.Legs)-1].Arrival.After(arrivalTime) {
		// Just pull the first one. (Fallback safety)
		bestJourney = journeys[0]
	}

	fmt.Println(accentStyle.Render(fmt.Sprintf("\n--- 🧭 Commute to %s ---", course.Name)))

	firstDepart := bestJourney.Legs[0].Departure
	totalDuration := bestJourney.Legs[len(bestJourney.Legs)-1].Arrival.Sub(firstDepart)

	fmt.Printf("Leave home by: %s\n", errorStyle.Render(firstDepart.Local().Format("15:04")))
	fmt.Printf("Total Travel Time: %d mins\n\n", int(totalDuration.Minutes()))

	for i, leg := range bestJourney.Legs {
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

	return nil
}

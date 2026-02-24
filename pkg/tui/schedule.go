package tui

import (
	"fmt"
	"os"
	"strings"

	"faliactl/pkg/config"
	"faliactl/pkg/exporter"
	"faliactl/pkg/scraper"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/huh/spinner"
)

// RunScheduleTUI runs the interactive flow for selecting study groups and exporting a timetable
func RunScheduleTUI() error {
	fmt.Println(accentStyle.Render("Welcome to the Faliactl Exporter!"))

	cfg, _ := config.Load()
	var selectedGroupURLs []string

	client := scraper.NewClient()

	var groups []scraper.Group
	var err error

	_ = spinner.New().
		Title("Fetching available study groups from Ostfalia...").
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
		fmt.Println(errorStyle.Render("No groups selected!"))
		return nil
	}

	// Fetch schedule based on selected groups
	var courses []scraper.Course
	seenEvent := make(map[string]bool)
	var fetchErr error

	_ = spinner.New().
		Title("Fetching schedules...").
		Action(func() {
			for _, url := range selectedGroupURLs {
				var groupCourses []scraper.Course
				groupCourses, fetchErr = client.FetchSchedule(url)
				if fetchErr != nil {
					fetchErr = fmt.Errorf("failed to fetch schedule for %s: %w", url, fetchErr)
					return
				}
				for _, c := range groupCourses {
					key := fmt.Sprintf("%s|%s|%s|%s", c.Name, c.DateStr, c.StartTime, c.EndTime)
					if !seenEvent[key] {
						seenEvent[key] = true
						courses = append(courses, c)
					}
				}
			}
		}).
		Run()

	if fetchErr != nil {
		return fetchErr
	}

	if len(courses) == 0 {
		fmt.Println(errorStyle.Render("No courses found for the selected groups!"))
		return nil
	}

	courseNamesMap := make(map[string]bool)
	var courseOptions []huh.Option[string]

	savedCourseMap := make(map[string]bool)
	if cfg != nil {
		for _, name := range cfg.SavedCourses {
			savedCourseMap[name] = true
		}
	}

	for _, c := range courses {
		name := c.Name

		if !courseNamesMap[name] {
			courseNamesMap[name] = true
			opt := huh.NewOption(name, name)

			// If user has saved courses, strictly select only those by default.
			// Otherwise, pre-select all available courses.
			if len(savedCourseMap) > 0 {
				if savedCourseMap[name] {
					opt = opt.Selected(true)
				}
			} else {
				opt = opt.Selected(true)
			}
			courseOptions = append(courseOptions, opt)
		}
	}

	var selectedCourses []string
	var outputFile string

	coursesForm := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select courses to export").
				Description("Space = toggle, Enter = confirm").
				Options(courseOptions...).
				Value(&selectedCourses).
				Filterable(true).
				Height(10),

			huh.NewInput().
				Title("Output file name").
				Value(&outputFile).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("file name cannot be empty")
					}
					return nil
				}),
		),
	).WithTheme(GetTheme())

	// Defaults
	outputFile = "schedule.ics"

	err = coursesForm.Run()
	if err != nil {
		return err
	}

	if !strings.HasSuffix(outputFile, ".ics") {
		outputFile += ".ics"
	}

	// Filter courses based on selection
	selectedMap := make(map[string]bool)
	for _, sc := range selectedCourses {
		selectedMap[sc] = true
	}

	var filteredCourses []scraper.Course
	for _, c := range courses {
		// Match against the base name
		if selectedMap[c.Name] {
			filteredCourses = append(filteredCourses, c)
		}
	}

	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	err = exporter.GenerateICS(filteredCourses, file)
	if err != nil {
		return fmt.Errorf("failed to generate ICS: %w", err)
	}

	fmt.Println(accentStyle.Render(fmt.Sprintf("\nSuccess! Exported %d course events to %s", len(filteredCourses), outputFile)))

	return nil
}

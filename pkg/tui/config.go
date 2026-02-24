package tui

import (
	"fmt"
	"strings"

	"faliactl/pkg/config"
	"faliactl/pkg/scraper"
	"faliactl/pkg/transit"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/huh/spinner"
	"github.com/charmbracelet/lipgloss"
)

// RunConfigTUI launches the interactive experience for managing configurations
func RunConfigTUI() error {
	for {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		var action string

		initialForm := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Configuration Settings").
					Options(
						huh.NewOption("Set Accent Color (Theme)", "theme"),
						huh.NewOption("Set Home Address (For Commutes)", "home"),
						huh.NewOption("Set Default Mensa Campus", "mensa"),
						huh.NewOption("Set Saved Study Groups", "groups"),
						huh.NewOption("Set Saved Courses", "courses"),
						huh.NewOption("View Current Config", "view"),
						huh.NewOption("Back to Main Menu", "back"),
					).
					Value(&action),
			),
		).WithTheme(GetTheme())

		if err := initialForm.Run(); err != nil {
			return err
		}

		if action == "back" {
			return nil
		}

		if action == "theme" {
			err = runSetThemeTUI(cfg)
		} else if action == "home" {
			err = runSetHomeTUI(cfg)
		} else if action == "mensa" {
			err = runSetMensaCampusTUI(cfg)
		} else if action == "groups" {
			err = runSetSavedGroupsTUI(cfg)
		} else if action == "courses" {
			err = runSetSavedCoursesTUI(cfg)
		} else if action == "view" {
			fmt.Println(accentStyle.Render("\n--- Current Configuration (~/.faliactl.json) ---"))
			if cfg.HomeAddress == "" {
				fmt.Println("Home Address: Not set")
			} else {
				fmt.Printf("Home Address: %s\n", cfg.HomeAddress)
			}

			fmt.Printf("Default Mensa: %s\n", cfg.DefaultCampus)
			fmt.Printf("Saved Groups: %d\n", len(cfg.SavedGroupURLs))
			fmt.Printf("Saved Courses: %d\n", len(cfg.SavedCourses))
			fmt.Printf("Accent Color: %s\n", cfg.AccentColor)
			fmt.Println()
		}

		if err != nil {
			return err
		}
	}
}

func runSetMensaCampusTUI(cfg *config.AppConfig) error {
	var selected string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select your default campus for the Mensa Menu").
				Options(
					huh.NewOption("Wolfenbüttel", "wolfenbuettel"),
					huh.NewOption("Braunschweig", "braunschweig"),
					huh.NewOption("Salzgitter", "salzgitter"),
					huh.NewOption("Suderburg", "suderburg"),
				).
				Value(&selected),
		),
	).WithTheme(GetTheme())

	if err := form.Run(); err != nil {
		return err
	}

	cfg.DefaultCampus = selected
	if err := config.Save(cfg); err != nil {
		return err
	}

	fmt.Println(accentStyle.Render(fmt.Sprintf("\n✅ Standard Mensa campus changed to: %s\n", selected)))
	return nil
}

func runSetSavedGroupsTUI(cfg *config.AppConfig) error {
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

	var groupOptions []huh.Option[string]

	// Check which ones are currently selected
	existingMap := make(map[string]bool)
	for _, url := range cfg.SavedGroupURLs {
		existingMap[url] = true
	}

	for _, g := range groups {
		opt := huh.NewOption(g.Name, g.URL)
		if existingMap[g.URL] {
			opt = opt.Selected(true)
		}
		groupOptions = append(groupOptions, opt)
	}

	var selectedURLs []string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select your study group(s)").
				Description("Space = toggle, Enter = confirm. Start typing to filter.").
				Options(groupOptions...).
				Value(&selectedURLs).
				Filterable(true).
				Height(12),
		),
	).WithTheme(GetTheme())

	if err := form.Run(); err != nil {
		return err
	}

	cfg.SavedGroupURLs = selectedURLs
	if err := config.Save(cfg); err != nil {
		return err
	}

	fmt.Println(accentStyle.Render(fmt.Sprintf("\n✅ Successfully saved %d groups.\n", len(selectedURLs))))
	return nil
}

func runSetSavedCoursesTUI(cfg *config.AppConfig) error {
	if len(cfg.SavedGroupURLs) == 0 {
		fmt.Println(errorStyle.Render("You must save at least one Study Group before you can configure Saved Courses!"))
		return nil
	}

	client := scraper.NewClient()
	var allCourses []scraper.Course
	var fetchErr error

	_ = spinner.New().
		Title("Fetching schedules for your saved groups...").
		Action(func() {
			for _, url := range cfg.SavedGroupURLs {
				groupCourses, err := client.FetchSchedule(url)
				if err != nil {
					fetchErr = fmt.Errorf("failed to fetch schedule for a group: %w", err)
					return
				}
				allCourses = append(allCourses, groupCourses...)
			}
		}).
		Run()

	if fetchErr != nil {
		return fetchErr
	}

	if len(allCourses) == 0 {
		fmt.Println(errorStyle.Render("No courses found for your saved groups!"))
		return nil
	}

	// Group chronologically duplicate sessions by their base underlying name
	courseNamesMap := make(map[string]bool)
	var courseOptions []huh.Option[string]

	existingMap := make(map[string]bool)
	for _, name := range cfg.SavedCourses {
		existingMap[name] = true
	}

	for _, c := range allCourses {
		if !courseNamesMap[c.Name] {
			courseNamesMap[c.Name] = true
			opt := huh.NewOption(c.Name, c.Name)
			if existingMap[c.Name] {
				opt = opt.Selected(true)
			}
			courseOptions = append(courseOptions, opt)
		}
	}

	var selectedCourses []string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select your active courses").
				Description("Only these will be shown in your Weekly Commute planner.\nSpace = toggle, Enter = confirm. Start typing to filter.").
				Options(courseOptions...).
				Value(&selectedCourses).
				Filterable(true).
				Height(12),
		),
	).WithTheme(GetTheme())

	if err := form.Run(); err != nil {
		return err
	}

	cfg.SavedCourses = selectedCourses
	if err := config.Save(cfg); err != nil {
		return err
	}

	fmt.Println(accentStyle.Render(fmt.Sprintf("\n✅ Standardized your dashboard down to %d saved courses.\n", len(selectedCourses))))
	return nil
}

func runSetHomeTUI(cfg *config.AppConfig) error {
	var input string

	inputForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Enter your home address or nearest bus stop").
				Description("This will be saved to your local config for fast Transit routing.").
				Placeholder("e.g. Braunschweig Hbf or 123 Musterstraße...").
				Value(&input),
		),
	).WithTheme(GetTheme())

	if err := inputForm.Run(); err != nil {
		return err
	}

	if input == "" {
		fmt.Println("Operation cancelled: No address provided.")
		return nil
	}

	client := transit.NewClient()
	var locations []transit.Location
	var fetchErr error

	_ = spinner.New().
		Title(fmt.Sprintf("Searching transit network for '%s'...", input)).
		Action(func() {
			locations, fetchErr = client.FetchLocations(input)
		}).
		Run()

	if fetchErr != nil {
		return fmt.Errorf("could not lookup address: %w", fetchErr)
	}

	if len(locations) == 0 {
		fmt.Println(errorStyle.Render(fmt.Sprintf("❌ No matching stations or addresses found for '%s'", input)))
		return nil
	}

	// Just grab the best match for now (HAFAS sorts by weight/relevance automatically)
	match := locations[0]
	cfg.HomeAddress = match.Name
	cfg.HomeStationID = match.ID

	if err := config.Save(cfg); err != nil {
		return err
	}

	fmt.Println(accentStyle.Render(fmt.Sprintf("\n✅ Successfully saved home location: %s (ID: %s)\n", match.Name, match.ID)))
	return nil
}

func colorBlock(color string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render("██")
}

func runSetThemeTUI(cfg *config.AppConfig) error {
	var input string

	inputForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Choose an Accent Color for faliactl").
				Description("Select a curated Charm style or choose Custom to enter your own Hex.").
				Options(
					huh.NewOption(fmt.Sprintf("%s Falia Purple", colorBlock("99")), "99"),
					huh.NewOption(fmt.Sprintf("%s Sakura Pink", colorBlock("205")), "205"),
					huh.NewOption(fmt.Sprintf("%s Ocean Blue", colorBlock("86")), "86"),
					huh.NewOption(fmt.Sprintf("%s Matrix Green", colorBlock("42")), "42"),
					huh.NewOption("✨ Custom Hex Code", "custom"),
				).
				Value(&input),
		),
	).WithTheme(GetTheme())

	if err := inputForm.Run(); err != nil {
		return err
	}

	if input == "custom" {
		var hexInput string
		hexForm := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Enter a Hex Color Code").
					Description("Include the `#` symbol. Example: #FF00FF").
					Placeholder("#").
					Value(&hexInput).
					Validate(func(str string) error {
						if len(str) != 7 || !strings.HasPrefix(str, "#") {
							return fmt.Errorf("must be a valid 6-character hex code starting with #")
						}
						return nil
					}),
			),
		).WithTheme(GetTheme())

		if err := hexForm.Run(); err != nil {
			return err
		}
		cfg.AccentColor = hexInput
	} else {
		cfg.AccentColor = input
	}

	if err := config.Save(cfg); err != nil {
		return err
	}

	fmt.Println(accentStyle.Render("\n✅ Beautiful! The theme color is now saved.\n"))
	return nil
}

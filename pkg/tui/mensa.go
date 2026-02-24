package tui

import (
	"fmt"
	"strings"
	"time"

	"faliactl/pkg/config"
	"faliactl/pkg/mensa"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/huh/spinner"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// RunMensaTUI runs the interactive flow for selecting a Mensa and displaying the menu
func RunMensaTUI() error {
	var selectedLocationID int
	var selectedDate string

	client := mensa.NewClient()
	var locations []mensa.Location
	var err error

	cfg, _ := config.Load()
	accentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Bold(true)

	var selectedCampus string
	if cfg != nil && cfg.DefaultCampus != "" {
		selectedCampus = cfg.DefaultCampus
		fmt.Println(accentStyle.Render(fmt.Sprintf("\nChecking Default Campus: %s...", selectedCampus)))
	} else {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Select Mensa Campus").
					Options(
						huh.NewOption("Braunschweig", "braunschweig"),
						huh.NewOption("Wolfenbüttel", "wolfenbuettel"),
						huh.NewOption("Salzgitter", "salzgitter"),
						huh.NewOption("Suderburg", "suderburg"),
					).
					Value(&selectedCampus),
			),
		).WithTheme(GetTheme())

		err = form.Run()
		if err != nil {
			return err
		}
	}

	_ = spinner.New().
		Title("Fetching available Mensa locations...").
		Action(func() {
			locations, err = client.FetchLocations()
		}).
		Run()

	if err != nil {
		return fmt.Errorf("failed to fetch mensa locations: %w", err)
	}

	var campusLocations []mensa.Location
	for _, loc := range locations {
		if strings.EqualFold(loc.Address.City, selectedCampus) {
			campusLocations = append(campusLocations, loc)
		}
	}

	if len(campusLocations) == 0 {
		return fmt.Errorf("no mensa locations found for campus: %s", selectedCampus)
	}

	var locationOptions []huh.Option[int]
	seenLocs := make(map[string]bool)
	for _, loc := range campusLocations {
		if !seenLocs[loc.Name] {
			seenLocs[loc.Name] = true
			locationOptions = append(locationOptions, huh.NewOption(loc.Name, loc.ID))
		}
	}

	// If only one location for the campus, select it automatically
	if len(locationOptions) == 1 {
		selectedLocationID = locationOptions[0].Value
		fmt.Println(accentStyle.Render(fmt.Sprintf("Automatically selected Mensa: %s", locationOptions[0].Key)))
	} else {
		// Otherwise, let the user choose the specific Mensa within the campus
		mensaSelectForm := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[int]().
					Title(fmt.Sprintf("Select Mensa in %s", cases.Title(language.German).String(selectedCampus))).
					Options(locationOptions...).
					Value(&selectedLocationID),
			),
		).WithTheme(GetTheme())

		err = mensaSelectForm.Run()
		if err != nil {
			return err
		}
	}

	today := time.Now().Format("2006-01-02")
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")

	dateOptions := []huh.Option[string]{
		huh.NewOption(fmt.Sprintf("Today (%s)", today), today),
		huh.NewOption(fmt.Sprintf("Tomorrow (%s)", tomorrow), tomorrow),
	}

	dateForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select Date").
				Options(dateOptions...).
				Value(&selectedDate),
		),
	).WithTheme(GetTheme())

	err = dateForm.Run()
	if err != nil {
		return err
	}

	var menu *mensa.MenuResponse

	_ = spinner.New().
		Title(fmt.Sprintf("Fetching menu for %s...", selectedDate)).
		Action(func() {
			menu, err = client.FetchMenu(selectedLocationID, selectedDate)
		}).
		Run()

	if err != nil {
		return fmt.Errorf("failed to fetch mensa menu: %w", err)
	}

	// Reusing same styles as the CLI version
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Bold(true).Padding(1, 0)
	priceStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	laneStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	fmt.Println(titleStyle.Render(fmt.Sprintf("Mensa Menu for %s", selectedDate)))

	if len(menu.Announcements) > 0 {
		warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		for _, a := range menu.Announcements {
			fmt.Println(warnStyle.Render(fmt.Sprintf("\nNOTICE: %s", a.Text)))
			if a.Closed {
				fmt.Println(warnStyle.Render("The Mensa is CLOSED."))
				return nil
			}
		}
	}

	if len(menu.Meals) == 0 {
		fmt.Println("No meals available for this date.")
		return nil
	}

	for _, meal := range menu.Meals {
		vegan := ""
		for _, cat := range meal.Tags.Categories {
			if cat.ID == "VEGA" {
				vegan = lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Render(" [Vegan]")
			}
		}

		extras := []string{}
		for _, a := range meal.Tags.Allergens {
			extras = append(extras, a.Name)
		}
		for _, a := range meal.Tags.Additives {
			extras = append(extras, a.Name)
		}
		for _, a := range meal.Tags.Special {
			extras = append(extras, a.Name)
		}

		extraStr := ""
		if len(extras) > 0 {
			extraStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Italic(true)
			extraStr = extraStyle.Render(fmt.Sprintf("\n  Info: %s", strings.Join(extras, ", ")))
		}

		fmt.Printf("• %s%s\n", meal.Name, vegan)

		prices := fmt.Sprintf("Stud: %s € | Emp: %s € | Guest: %s €",
			priceStyle.Render(meal.Price.Student),
			priceStyle.Render(meal.Price.Employee),
			priceStyle.Render(meal.Price.Guest),
		)

		fmt.Printf("  %s | %s%s\n\n", laneStyle.Render(meal.Lane.Name), prices, extraStr)
	}

	return nil
}

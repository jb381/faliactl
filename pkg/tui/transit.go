package tui

import (
	"fmt"

	"faliactl/pkg/config"
	"faliactl/pkg/transit"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/huh/spinner"
	"github.com/charmbracelet/lipgloss"
)

var (
	transitCampuses = []huh.Option[string]{
		huh.NewOption("Wolfenbüttel (Hauptcampus)", "891097"),
		huh.NewOption("Wolfenbüttel (Am Exer)", "891011"),
		huh.NewOption("Salzgitter (Ostfalia Campus)", "991604089"),
		huh.NewOption("Suderburg (Ostfalia Campus)", "991604106"),
	}
)

// RunTransitTUI launches the interactive experience for public transit
func RunTransitTUI() error {
	var stationID string
	var action string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Which campus are you at?").
				Options(transitCampuses...).
				Value(&stationID),

			huh.NewSelect[string]().
				Title("What do you want to do?").
				Options(
					huh.NewOption("View Next Departures", "departures"),
					huh.NewOption("Route Home", "home"),
				).
				Value(&action),
		),
	).WithTheme(GetTheme())

	if err := form.Run(); err != nil {
		return err
	}

	client := transit.NewClient()

	if action == "departures" {
		return runDeparturesView(client, stationID)
	}

	return runRouteHomeView(client, stationID)
}

func runDeparturesView(client *transit.Client, stationID string) error {
	var deps []transit.Departure
	var err error

	_ = spinner.New().
		Title("Fetching live departures...").
		Action(func() {
			deps, err = client.FetchDepartures(stationID, 60)
		}).
		Run()

	if err != nil {
		return fmt.Errorf("could not fetch departures: %w", err)
	}

	if len(deps) == 0 {
		fmt.Println(errorStyle.Render("No upcoming departures found in the next 60 minutes."))
		return nil
	}

	fmt.Println(accentStyle.Render("\n--- 🚌 Next Departures ---"))

	summary := transit.SummarizeDepartures(deps, 2)

	for _, route := range summary {
		lineStr := lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true).Render(route.LineName)
		fmt.Printf("\n%s -> %s\n", lineStr, route.Direction)

		for _, d := range route.Departures {
			delayStr := ""
			if d.Delay != nil && *d.Delay > 0 {
				delayStr = errorStyle.Render(fmt.Sprintf(" (+%d min delay)", *d.Delay/60))
			}

			timeStr := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Render(d.When.Local().Format("15:04"))

			fmt.Printf("  • [%s]%s\n", timeStr, delayStr)
		}
	}
	fmt.Println()

	return nil
}

func runRouteHomeView(client *transit.Client, stationID string) error {
	cfg, err := config.Load()
	if err != nil || cfg.HomeStationID == "" {
		fmt.Println(errorStyle.Render("Home address is not configured."))
		fmt.Println("Please run 'faliactl config --set-home \"Your Address\"' in your terminal first.")
		return nil
	}

	var journeys []transit.Journey
	var fetchErr error

	_ = spinner.New().
		Title(fmt.Sprintf("Routing trip from campus to %s...", cfg.HomeAddress)).
		Action(func() {
			journeys, fetchErr = client.FetchJourneys(stationID, cfg.HomeStationID)
		}).
		Run()

	if fetchErr != nil {
		return fmt.Errorf("could not route journey: %w", fetchErr)
	}

	if len(journeys) == 0 {
		fmt.Println(errorStyle.Render("No routes could be found. It might be too late at night."))
		return nil
	}

	fmt.Println(accentStyle.Render(fmt.Sprintf("\n--- 🧭 Route Home to %s ---", cfg.HomeAddress)))

	firstJourney := journeys[0]

	for i, leg := range firstJourney.Legs {
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

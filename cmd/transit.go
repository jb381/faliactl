package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"faliactl/pkg/config"
	"faliactl/pkg/transit"

	ics "github.com/arran4/golang-ical"
	"github.com/charmbracelet/huh/spinner"
	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Map standard Ostfalia campus names to their HAFAS Station IDs
var transitCampusMap = map[string]string{
	"wf-haupt":      "891097",    // Wolfenbüttel Hauptcampus
	"wf-exer":       "891011",    // Wolfenbüttel Am Exer
	"wolfenbuettel": "891097",    // Default WF to Hauptcampus
	"salzgitter":    "991604089", // Salzgitter, Ostfalia Hochschule
	"suderburg":     "991604106", // Suderburg, Ostfalia Hochschule
	"braunschweig":  "8000049",   // Braunschweig Hbf (Generic proxy for now)
}

var transitCmd = &cobra.Command{
	Use:   "transit",
	Short: "View live bus and train departures for Ostfalia campuses",
	Long:  "Leverages the public HAFAS API (DB/VRB) to fetch live transit departures or route you directly home from campus.",
	RunE: func(cmd *cobra.Command, args []string) error {
		campusFlag, _ := cmd.Flags().GetString("campus")
		routeHome, _ := cmd.Flags().GetBool("home")
		exportWeek, _ := cmd.Flags().GetBool("export-week")

		if campusFlag == "" {
			return fmt.Errorf("must specify a campus using --campus (e.g., salzgitter, wolfenbuettel)")
		}

		campuses := strings.Split(campusFlag, ",")
		client := transit.NewClient()

		for _, campusName := range campuses {
			campusName = strings.TrimSpace(strings.ToLower(campusName))
			stationID, ok := transitCampusMap[campusName]
			if !ok {
				fmt.Printf("⚠️ Warning: Unknown campus '%s'. Skipping.\n", campusName)
				continue
			}

			if exportWeek {
				if err := exportTransitICS(client, campusName, stationID); err != nil {
					fmt.Printf("❌ Failed to export transit ICS for %s: %v\n", campusName, err)
				}
			} else if routeHome {
				if err := printRouteHome(client, campusName, stationID); err != nil {
					fmt.Printf("❌ Failed to find route home from %s: %v\n", campusName, err)
				}
			} else {
				if err := printDepartures(client, campusName, stationID); err != nil {
					fmt.Printf("❌ Failed to fetch departures for %s: %v\n", campusName, err)
				}
			}
			fmt.Println()
		}

		return nil
	},
}

func printDepartures(client *transit.Client, campusName string, stationID string) error {
	var deps []transit.Departure
	var err error

	_ = spinner.New().
		Title(fmt.Sprintf("Fetching live departures for %s...", campusName)).
		Action(func() {
			deps, err = client.FetchDepartures(stationID, 60)
		}).
		Run()

	if err != nil {
		return err
	}

	fmt.Printf("\n--- 🚌 Next Departures: Ostfalia Campus %s ---\n", cases.Title(language.German).String(campusName))

	if len(deps) == 0 {
		fmt.Println("No upcoming departures found in the next 60 minutes.")
		return nil
	}

	summary := transit.SummarizeDepartures(deps, 2)

	for _, route := range summary {
		fmt.Printf("\n🚐 \033[1m%s\033[0m -> %s\n", route.LineName, route.Direction)

		for _, d := range route.Departures {
			delayStr := ""
			if d.Delay != nil && *d.Delay > 0 {
				delayStr = fmt.Sprintf("\033[31m (+%d min delay)\033[0m", *d.Delay/60)
			}
			fmt.Printf("  • [%s]%s\n",
				d.When.Local().Format("15:04"),
				delayStr,
			)
		}
	}
	return nil
}

func printRouteHome(client *transit.Client, campusName string, fromStationID string) error {
	cfg, err := config.Load()
	if err != nil || cfg.HomeStationID == "" {
		return fmt.Errorf("home address is not configured. Please run 'faliactl config --set-home \"Your Address\"' first")
	}

	var journeys []transit.Journey
	var fetchErr error

	_ = spinner.New().
		Title(fmt.Sprintf("Routing trip from %s to %s...", campusName, cfg.HomeAddress)).
		Action(func() {
			journeys, fetchErr = client.FetchJourneys(fromStationID, cfg.HomeStationID)
		}).
		Run()

	if fetchErr != nil {
		return fetchErr
	}

	fmt.Printf("\n--- 🧭 Route Home %s -> %s ---\n", cases.Title(language.German).String(campusName), cfg.HomeAddress)

	if len(journeys) == 0 {
		fmt.Println("No routes could be found. It might be too late at night.")
		return nil
	}

	// Just print the fastest/closest journey for now
	firstJourney := journeys[0]

	for i, leg := range firstJourney.Legs {
		lineName := "Walk🚶"
		if leg.Line != nil {
			lineName = leg.Line.Name
		}

		fmt.Printf("%d. [%s] %s -> %s (Arrive: %s)\n",
			i+1,
			leg.Departure.Local().Format("15:04"),
			lineName,
			leg.Destination.Name,
			leg.Arrival.Local().Format("15:04"))
	}

	return nil
}

func exportTransitICS(client *transit.Client, campusName string, fromStationID string) error {
	cfg, err := config.Load()
	if err != nil || cfg.HomeStationID == "" {
		return fmt.Errorf("home address is not configured. Please run 'faliactl config --set-home \"Your Address\"' first")
	}

	cal := ics.NewCalendar()
	cal.SetMethod(ics.MethodPublish)

	now := time.Now()

	// Fetch today's fastest route home, then simulate applying it to the next 7 days
	// Note: In an ideal world we iterate date parameters, but this is a solid approximation
	// until we add precise date flags to FetchJourneys.
	var journeys []transit.Journey
	var fetchErr error

	_ = spinner.New().
		Title(fmt.Sprintf("Exporting upcoming 7 days commute from %s to %s...", campusName, cfg.HomeAddress)).
		Action(func() {
			journeys, fetchErr = client.FetchJourneys(fromStationID, cfg.HomeStationID)
		}).
		Run()

	if fetchErr != nil {
		return fetchErr
	}

	if len(journeys) == 0 {
		return fmt.Errorf("no valid transit routes found to export")
	}

	bestJourney := journeys[0]
	// Basic total journey stats from the first leg to the last leg
	journeyStart := bestJourney.Legs[0].Departure
	journeyEnd := bestJourney.Legs[len(bestJourney.Legs)-1].Arrival
	duration := journeyEnd.Sub(journeyStart)

	for i := 0; i < 7; i++ {
		targetDate := now.AddDate(0, 0, i)
		// Skip weekends
		if targetDate.Weekday() == time.Saturday || targetDate.Weekday() == time.Sunday {
			continue
		}

		eventStart := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), journeyStart.Hour(), journeyStart.Minute(), 0, 0, targetDate.Location())
		eventEnd := eventStart.Add(duration)

		event := cal.AddEvent(fmt.Sprintf("%s-commute-%d", campusName, i))
		event.SetCreatedTime(time.Now())
		event.SetDtStampTime(time.Now())
		event.SetModifiedAt(time.Now())
		event.SetStartAt(eventStart)
		event.SetEndAt(eventEnd)
		event.SetSummary(fmt.Sprintf("🚌 Commute Home (%s)", cases.Title(language.German).String(campusName)))
		event.SetLocation(fmt.Sprintf("Start: %s", bestJourney.Legs[0].Origin.Name))

		// Build a description with all transfers
		desc := "Live tracking normally available via DB Navigator.\n\nJourney Details:\n"
		for i, leg := range bestJourney.Legs {
			lineName := "Walk"
			if leg.Line != nil {
				lineName = leg.Line.Name
			}
			desc += fmt.Sprintf("%d. %s -> %s\n", i+1, lineName, leg.Destination.Name)
		}
		event.SetDescription(desc)
	}

	filename := fmt.Sprintf("transit_%s.ics", campusName)
	err = os.WriteFile(filename, []byte(cal.Serialize()), 0644)
	if err != nil {
		return fmt.Errorf("could not write ics file: %w", err)
	}

	fmt.Printf("\n✨ Successfully exported commute calendar to: %s\n", filename)
	return nil
}

func init() {
	rootCmd.AddCommand(transitCmd)
	transitCmd.Flags().StringP("campus", "c", "", "Ostfalia campus (salzgitter, wolfenbuettel, suderburg)")
	transitCmd.Flags().BoolP("home", "r", false, "Route directly from the campus to your saved home address")
	transitCmd.Flags().BoolP("export-week", "e", false, "Export the next 7 days of commutes to an .ics calendar file")
}

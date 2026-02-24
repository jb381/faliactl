package cmd

import (
	"fmt"
	"strings"
	"time"

	"faliactl/pkg/mensa"

	"github.com/charmbracelet/huh/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	campusID int
	dateStr  string
)

// campusMap maps human readable names to API location IDs
var campusMap = map[string]int{
	"wolfenbuettel": 130, // Main Mensa Wolfenbüttel
	"wolfsburg":     112, // Bistro 4U Wolfsburg
	"suderburg":     134, // Mensa Suderburg
	"salzgitter":    200, // Mensa Salzgitter
}

var mensaCmd = &cobra.Command{
	Use:   "mensa",
	Short: "View the Mensa menu for a specific campus",
	Long:  `Fetch and display the daily cafeteria menu for Ostfalia campuses.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		campusName, _ := cmd.Flags().GetString("campus")

		client := mensa.NewClient()

		locID, ok := campusMap[campusName]
		if campusID != 0 {
			locID = campusID
		} else if !ok {
			// Fallback: fetch dynamically and substring match
			var locations []mensa.Location
			var err error
			_ = spinner.New().
				Title("Searching for Mensa location...").
				Action(func() {
					locations, err = client.FetchLocations()
				}).
				Run()

			if err == nil {
				for _, loc := range locations {
					if strings.Contains(strings.ToLower(loc.Name), strings.ToLower(campusName)) {
						locID = loc.ID
						break
					}
				}
			}
			if locID == 0 {
				return fmt.Errorf("could not find a matching Mensa location for: %s", campusName)
			}
		}

		// Default to today if no date provided
		fetchDate := dateStr
		if fetchDate == "" {
			fetchDate = time.Now().Format("2006-01-02")
		}

		var menu *mensa.MenuResponse
		var err error
		_ = spinner.New().
			Title(fmt.Sprintf("Fetching menu for %s...", fetchDate)).
			Action(func() {
				menu, err = client.FetchMenu(locID, fetchDate)
			}).
			Run()

		if err != nil {
			return fmt.Errorf("could not fetch menu: %w", err)
		}

		printMenu(menu, fetchDate)
		return nil
	},
}

func printMenu(menu *mensa.MenuResponse, date string) {
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Bold(true).Padding(1, 0)
	priceStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	laneStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	fmt.Println(titleStyle.Render(fmt.Sprintf("Mensa Menu for %s", date)))

	if len(menu.Announcements) > 0 {
		warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		for _, a := range menu.Announcements {
			fmt.Println(warnStyle.Render(fmt.Sprintf("\nNOTICE: %s", a.Text)))
			if a.Closed {
				fmt.Println(warnStyle.Render("The Mensa is CLOSED."))
				return
			}
		}
	}

	if len(menu.Meals) == 0 {
		fmt.Println("No meals available for this date.")
		return
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
}

func init() {
	rootCmd.AddCommand(mensaCmd)
	mensaCmd.Flags().StringP("campus", "c", "wolfenbuettel", "Campus name (wolfenbuettel, wolfsburg, suderburg, salzgitter)")
	mensaCmd.Flags().IntVar(&campusID, "id", 0, "Direct Mensa Location ID (overrides campus flag)")
	mensaCmd.Flags().StringVarP(&dateStr, "date", "d", "", "Date to fetch (format: YYYY-MM-DD), defaults to today")
}

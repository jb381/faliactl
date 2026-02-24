package cmd

import (
	"faliactl/pkg/config"
	"faliactl/pkg/transit"
	"faliactl/pkg/tui"
	"fmt"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage faliactl configuration",
	Long:  "View or edit your local configuration settings (like home address for transit routing).",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		setHome, _ := cmd.Flags().GetString("set-home")
		if setHome != "" {
			fmt.Printf("Searching HAFAS for address: '%s'...\n", setHome)

			// Use the transit API to lookup the location ID for this address
			client := transit.NewClient()
			locations, err := client.FetchLocations(setHome)
			if err != nil {
				return fmt.Errorf("could not lookup address: %w", err)
			}
			if len(locations) == 0 {
				return fmt.Errorf("no matching stations or addresses found for '%s'", setHome)
			}

			// Snag the first/best match
			match := locations[0]
			cfg.HomeAddress = match.Name
			cfg.HomeStationID = match.ID

			if err := config.Save(cfg); err != nil {
				return err
			}

			fmt.Printf("✅ Home address successfully saved as: %s (ID: %s)\n", match.Name, match.ID)
			return nil
		}

		// If no flags are given, launch the interactive TUI flow
		return tui.RunConfigTUI()
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.Flags().StringP("set-home", "s", "", "Set your home address for transit routing")
}

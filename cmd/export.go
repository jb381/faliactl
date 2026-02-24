package cmd

import (
	"fmt"
	"os"
	"strings"

	"faliactl/pkg/exporter"
	"faliactl/pkg/scraper"

	"github.com/charmbracelet/huh/spinner"
	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Directly export a schedule to an ICS file",
	Long:  `Export a schedule for a specific group to an ICS file without using the interactive TUI.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		group, _ := cmd.Flags().GetString("group")
		output, _ := cmd.Flags().GetString("output")

		// Ensure it has .html suffix
		urlPath := group
		if !strings.HasSuffix(urlPath, ".html") {
			urlPath += ".html"
		}

		client := scraper.NewClient()
		var courses []scraper.Course
		var err error

		_ = spinner.New().
			Title(fmt.Sprintf("Exporting schedule for group %s to %s...", group, output)).
			Action(func() {
				courses, err = client.FetchSchedule(urlPath)
			}).
			Run()

		if err != nil {
			return fmt.Errorf("failed to fetch schedule: %w", err)
		}

		if len(courses) == 0 {
			return fmt.Errorf("no courses found for group %s", group)
		}

		file, err := os.Create(output)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer file.Close()

		err = exporter.GenerateICS(courses, file)
		if err != nil {
			return fmt.Errorf("failed to generate ICS: %w", err)
		}

		fmt.Printf("Successfully exported %d courses to %s\n", len(courses), output)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(exportCmd)

	exportCmd.Flags().StringP("group", "g", "", "Group ID to export (e.g. 161902 or 161902.html)")
	exportCmd.Flags().StringP("output", "o", "schedule.ics", "Output file path")
	exportCmd.MarkFlagRequired("group")
}

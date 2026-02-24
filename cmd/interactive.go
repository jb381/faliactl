package cmd

import (
	"faliactl/pkg/tui"

	"github.com/spf13/cobra"
)

var interactiveCmd = &cobra.Command{
	Use:   "interactive",
	Short: "Launch the interactive TUI",
	Long:  `Launch the Text User Interface to browse groups, filter courses, and export schedules interactively.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return tui.RunTUI()
	},
}

func init() {
	rootCmd.AddCommand(interactiveCmd)
}

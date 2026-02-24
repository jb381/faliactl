package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "faliactl",
	Short: "A CLI and TUI for Ostfalia timetables",
	Long: `faliactl is an application for students at Ostfalia University 
to easily scrape their course schedule and export it to an .ics file.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

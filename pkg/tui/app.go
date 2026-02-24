package tui

import (
	"faliactl/pkg/config"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

var (
	// These act as fallbacks initially, but should ideally be dynamically instantiated by GetTheme()
	accentStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("99"))
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
)

// GetTheme securely loads the user's saved Accent Color and constructs the UI theme.
func GetTheme() *huh.Theme {
	cfg, err := config.Load()
	baseColor := "99" // Default Faliactl Purple

	if err == nil && cfg != nil && cfg.AccentColor != "" {
		baseColor = cfg.AccentColor
	}

	// Update the global lipgloss accent so manual CLI print statements also receive the color
	accentStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(baseColor))

	return GetCustomTheme(baseColor)
}

// GetCustomTheme returns a new huh.Theme instantiated with the provided lipgloss color string.
// This is used for live-previewing styles before they are officially saved.
func GetCustomTheme(baseColor string) *huh.Theme {

	t := huh.ThemeCharm()
	p := lipgloss.Color(baseColor)

	// Inject the dynamic color into the active inputs, cursors, borders, and buttons
	t.Focused.Title = t.Focused.Title.Foreground(p).Bold(true)
	t.Focused.Directory = t.Focused.Directory.Foreground(p)
	t.Focused.File = t.Focused.File.Foreground(p)
	t.Focused.Base = t.Focused.Base.Border(lipgloss.RoundedBorder()).BorderForeground(p).Padding(0, 1)
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(p)
	t.Focused.MultiSelectSelector = t.Focused.MultiSelectSelector.Foreground(p)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(p)
	t.Focused.SelectedPrefix = t.Focused.SelectedPrefix.Foreground(p)
	t.Focused.UnselectedPrefix = t.Focused.UnselectedPrefix.Foreground(lipgloss.AdaptiveColor{Light: "", Dark: "235"})
	t.Focused.TextInput.Cursor = t.Focused.TextInput.Cursor.Foreground(p)
	t.Focused.TextInput.Prompt = t.Focused.TextInput.Prompt.Foreground(p)
	t.Focused.FocusedButton = t.Focused.FocusedButton.Foreground(lipgloss.Color("0")).Background(p)

	// Softer borders for unfocused elements
	t.Blurred.Base = t.Blurred.Base.Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("238")).Padding(0, 1)
	t.Focused.TextInput.Cursor = t.Focused.TextInput.Cursor.Foreground(p)
	t.Focused.TextInput.Prompt = t.Focused.TextInput.Prompt.Foreground(p)
	t.Focused.FocusedButton = t.Focused.FocusedButton.Foreground(lipgloss.Color("0")).Background(p)

	return t
}

// RunTUI launches the main menu interactive form experience
func RunTUI() error {
	var action string

	initialForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("What would you like to do?").
				Options(
					huh.NewOption("📅 Export Timetable", "schedule"),
					huh.NewOption("🍔 View Mensa Menu", "mensa"),
					huh.NewOption("🚆 Check Transit", "transit"),
					huh.NewOption("📍 Plan Course Commute", "commute"),
					huh.NewOption("🗺️ Weekly Commute Planner", "weekly"),
					huh.NewOption("⚙️ Settings", "config"),
				).
				Value(&action),
		),
	).WithTheme(GetTheme())

	if err := initialForm.Run(); err != nil {
		return err
	}

	if action == "mensa" {
		return RunMensaTUI()
	} else if action == "transit" {
		return RunTransitTUI()
	} else if action == "commute" {
		return RunCourseCommuteTUI()
	} else if action == "weekly" {
		return RunWeeklyCommuteTUI()
	} else if action == "config" {
		return RunConfigTUI()
	}

	return RunScheduleTUI()
}

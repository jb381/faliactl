package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// AppConfig holds all user-defined persistent settings
type AppConfig struct {
	HomeAddress    string   `json:"home_address,omitempty"`
	HomeStationID  string   `json:"home_station_id,omitempty"`
	SavedGroupURLs []string `json:"saved_group_urls,omitempty"`
	SavedCourses   []string `json:"saved_courses,omitempty"`
	DefaultCampus  string   `json:"default_campus,omitempty"`
	AccentColor    string   `json:"accent_color,omitempty"`
}

// getConfigPath returns the absolute path to ~/.faliactl.json
func getConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not find user home directory: %w", err)
	}
	return filepath.Join(homeDir, ".faliactl.json"), nil
}

// Load reads the application configuration from disk.
// Returns an empty struct if the file does not exist.
func Load() (*AppConfig, error) {
	path, err := getConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		// If file doesn't exist, just return an empty default configuration
		if os.IsNotExist(err) {
			return &AppConfig{}, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config JSON: %w", err)
	}

	return &cfg, nil
}

// Save writes the application configuration back to disk.
func Save(cfg *AppConfig) error {
	path, err := getConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

package scraper

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// cacheDuration determines how long schedule data is kept before refreshing
const cacheDuration = 12 * time.Hour

// CacheEntry represents the disk data format
type CacheEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Courses   []Course  `json:"courses"`
}

func getCachePath(groupURL string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not find user home directory: %w", err)
	}

	cacheDir := filepath.Join(homeDir, ".faliactl_cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", fmt.Errorf("could not create cache directory: %w", err)
	}

	// Create a safe filesystem name from the URL (e.g., "161902.html" -> "161902_html.json")
	// For simplicity, just use base name
	base := filepath.Base(groupURL)
	return filepath.Join(cacheDir, base+".json"), nil
}

// readCache checks if a valid, unexpired cache exists for this group
func readCache(groupURL string) ([]Course, bool) {
	path, err := getCachePath(groupURL)
	if err != nil {
		return nil, false
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false // File doesn't exist or can't be read
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, false
	}

	// Check expiration
	if time.Since(entry.Timestamp) > cacheDuration {
		return nil, false // Expired
	}

	return entry.Courses, true
}

// writeCache saves the schedule to disk
func writeCache(groupURL string, courses []Course) {
	path, err := getCachePath(groupURL)
	if err != nil {
		return
	}

	entry := CacheEntry{
		Timestamp: time.Now(),
		Courses:   courses,
	}

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return
	}

	_ = os.WriteFile(path, data, 0644)
}

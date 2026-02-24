package mensa

import (
	"testing"
	"time"
)

// TestMensaIntegration actually connects to the api.stw-on.de backend
// If this fails, the API might be down or changed its JSON structure.
func TestMensaIntegration_FetchLocations(t *testing.T) {
	client := NewClient()

	locations, err := client.FetchLocations()
	if err != nil {
		t.Fatalf("Failed to fetch locations: %v", err)
	}

	if len(locations) == 0 {
		t.Fatalf("Expected locations from API, got 0")
	}

	// Make sure Wolfenbüttel is in the list
	foundWolfenbuettel := false
	for _, loc := range locations {
		if loc.ID == 130 { // Main Mensa Wolfenbüttel
			foundWolfenbuettel = true
			if loc.Name == "" {
				t.Errorf("Location ID 130 has no name")
			}
			break
		}
	}

	if !foundWolfenbuettel {
		t.Errorf("Could not find Mensa Wolfenbüttel (ID 130) in the API response.")
	}
}

// TestMensaIntegration_FetchMenu connects to the API to pull a specific day's menu.
func TestMensaIntegration_FetchMenu(t *testing.T) {
	client := NewClient()

	// We'll test grabbing today's menu for Wolfenbüttel (ID 130)
	// Some days (like Sunday) it might be empty, so we mostly check for no HTTP/JSON errors.
	today := time.Now().Format("2006-01-02")
	menu, err := client.FetchMenu(130, today)

	if err != nil {
		// A 404 is technically valid if there are legitimately no meals today (e.g. Sunday/Holiday)
		// but the json parsing shouldn't crash if it returns a 200.
		if err.Error() != "no menu available for this date/location" {
			t.Fatalf("Failed to fetch menu with unexpected error: %v", err)
		}
	} else {
		// If we did get a 200 OK, make sure the structs populated.
		if menu == nil {
			t.Fatalf("Menu was nil despite no errors")
		}
	}
}

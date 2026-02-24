package transit

import (
	"testing"
	"time"
)

func TestSummarizeDepartures(t *testing.T) {
	loc, _ := time.LoadLocation("UTC")
	now := time.Now().In(loc)

	deps := []Departure{
		{
			Line:      Line{Name: "Bus 420"},
			Direction: "Campus",
			When:      now.Add(5 * time.Minute),
		},
		{
			Line:      Line{Name: "Bus 420"},
			Direction: "Campus",
			When:      now.Add(15 * time.Minute),
		},
		{
			Line:      Line{Name: "Tram 1"},
			Direction: "City Center",
			When:      now.Add(2 * time.Minute),
		},
		{
			Line:      Line{Name: "Bus 420"},
			Direction: "Campus",
			When:      now.Add(25 * time.Minute),
		},
		{
			Line:      Line{Name: "Tram 1"},
			Direction: "City Center",
			When:      now.Add(12 * time.Minute),
		},
	}

	// Summarize up to 2 departures per route
	summary := SummarizeDepartures(deps, 2)

	if len(summary) != 2 {
		t.Fatalf("expected 2 unique routes, got %d", len(summary))
	}

	// First route should be Tram 1 because its first departure is sooner (2 min)
	if summary[0].LineName != "Tram 1" {
		t.Errorf("expected first route to be Tram 1 because it's departing sooner, got %s", summary[0].LineName)
	}

	if len(summary[0].Departures) != 2 {
		t.Errorf("expected 2 departures for Tram 1, got %d", len(summary[0].Departures))
	}

	// Second route should be Bus 420
	if summary[1].LineName != "Bus 420" {
		t.Errorf("expected second route to be Bus 420, got %s", summary[1].LineName)
	}

	if len(summary[1].Departures) != 2 {
		t.Errorf("expected exactly 2 departures for Bus 420 (clipping the 3rd), got %d", len(summary[1].Departures))
	}

	// Ensure chronological sorting within the grouped departures
	if summary[1].Departures[0].When.After(summary[1].Departures[1].When) {
		t.Errorf("departures within route are not sorted chronologically")
	}
}

func TestSummarizeDepartures_Empty(t *testing.T) {
	summary := SummarizeDepartures([]Departure{}, 5)
	if len(summary) != 0 {
		t.Errorf("expected empty output for empty input, got %d", len(summary))
	}
}

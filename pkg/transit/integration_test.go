package transit

import (
	"testing"
)

func TestTransitIntegration_FetchLocations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := NewClient()

	locations, err := client.FetchLocations("Wolfenbüttel Fachhochschule")
	if err != nil {
		t.Fatalf("Failed to fetch locations: %v", err)
	}

	if len(locations) == 0 {
		t.Fatal("Expected at least one location, got 0")
	}

	found := false
	for _, loc := range locations {
		if loc.ID == "8000255" || loc.ID == "991604089" || loc.ID == "885208" || loc.Name != "" {
			// Just verify it parses basically
			found = true
			if loc.Name == "" {
				t.Errorf("Location missing name: %+v", loc)
			}
		}
	}

	if !found {
		t.Errorf("Could not find any cleanly parsed locations in: %v", locations)
	}
}

func TestTransitIntegration_FetchDepartures(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := NewClient()

	// 991604089 is Salzgitter Ostfalia Campus (usually has buses all day)
	deps, err := client.FetchDepartures("991604089", 120) // 120 mins
	if err != nil {
		t.Fatalf("Failed to fetch departures: %v", err)
	}

	if len(deps) == 0 {
		t.Logf("Got 0 departures for Salzgitter. Note: this might happen late at night or on weekends.")
	} else {
		for _, dep := range deps {
			if dep.Direction == "" {
				t.Errorf("Departure missing direction: %+v", dep)
			}
			if dep.Line.Name == "" {
				t.Errorf("Departure missing line name: %+v", dep)
			}
			if dep.When.IsZero() {
				t.Errorf("Departure missing timestamp: %+v", dep)
			}
		}
	}
}

func TestTransitIntegration_FetchJourneys(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := NewClient()

	// Wolfenbüttel (8000255) to Braunschweig Hbf (8000049)
	journeys, err := client.FetchJourneys("8000255", "8000049")
	if err != nil {
		t.Fatalf("Failed to fetch journeys: %v", err)
	}

	if len(journeys) == 0 {
		t.Logf("Got 0 journeys between WF and BS. This is unusual but possible late at night.")
	} else {
		for _, j := range journeys {
			if len(j.Legs) == 0 {
				t.Errorf("Journey has no legs: %+v", j)
			}
			for _, leg := range j.Legs {
				if leg.Origin.Name == "" {
					t.Errorf("Leg missing origin name: %+v", leg)
				}
				if leg.Destination.Name == "" {
					t.Errorf("Leg missing destination name: %+v", leg)
				}
			}
		}
	}
}

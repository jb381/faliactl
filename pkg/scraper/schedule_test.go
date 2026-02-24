package scraper

import (
	"os"
	"testing"
)

func TestParseSchedule(t *testing.T) {
	file, err := os.Open("161902_test.html")
	if err != nil {
		t.Skip("161902_test.html not found, skipping test")
	}
	defer file.Close()

	courses, err := ParseSchedule(file)
	if err != nil {
		t.Fatalf("ParseSchedule failed: %v", err)
	}

	if len(courses) == 0 {
		t.Fatalf("Expected to find courses, found 0")
	}

	// Verify the first course which should be Lineare Algebra from the downloaded HTML
	foundLinearAlg := false
	for _, c := range courses {
		if c.Name == "Lineare Algebra" && c.StartTime == "08:15" && c.EndTime == "09:45" {
			foundLinearAlg = true
			if c.Room != "WF-EX-7/3" {
				t.Errorf("Expected room WF-EX-7/3, got %s", c.Room)
			}
			if c.DateStr != "04.03.2026 (Mittwoch)" {
				t.Errorf("Expected date 04.03.2026 (Mittwoch), got %s", c.DateStr)
			}
			break
		}
	}

	if !foundLinearAlg {
		t.Errorf("Failed to find 'Lineare Algebra' at 08:15")
	}
}

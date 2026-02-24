package scraper

import (
	"strings"
	"testing"
)

// TestScraperIntegration actually connects to the Ostfalia web server.
// If this test fails, it means the University changed their HTML structure or the server is down.
func TestScraperIntegration_FetchGroups(t *testing.T) {
	client := NewClient()

	groups, err := client.FetchGroups()
	if err != nil {
		t.Fatalf("Failed to fetch groups from Ostfalia: %v", err)
	}

	if len(groups) == 0 {
		t.Fatalf("Expected to find study groups, but got 0")
	}

	// Verify we got the expected group format
	foundDTI := false
	for _, g := range groups {
		if strings.Contains(strings.ToLower(g.Name), "digital technologies") {
			foundDTI = true
			if g.URL == "" {
				t.Fatalf("Found group but URL was empty")
			}
			break
		}
	}

	if !foundDTI {
		t.Errorf("Could not find 'Digital Technologies' in the group list. Did the university change the program names?")
	}
}

// TestScraperIntegration_FetchSchedule actually connects to a specific Ostfalia schedule page.
func TestScraperIntegration_FetchSchedule(t *testing.T) {
	client := NewClient()

	// 161902.html is historically the "Digital Technologies" schedule endpoint we've used for testing
	// We just want to make sure the endpoint parses *something* without crashing and returning 0 courses.
	courses, err := client.FetchSchedule("161902.html")
	if err != nil {
		t.Fatalf("Failed to fetch schedule from Ostfalia: %v", err)
	}

	// It's possible for a schedule to be legitimately empty if it's out of season,
	// but mostly we are just testing the HTTP connection and parse logic doesn't panic.
	if len(courses) > 0 {
		// Just verify the first course has basic required fields populated
		c := courses[0]
		if c.Name == "" || c.StartTime == "" || c.EndTime == "" {
			t.Errorf("Parsed course is missing critical fields: %+v", c)
		}
	}
}

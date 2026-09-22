package cmd

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"faliactl/pkg/scraper"
)

type stubScheduleFetcher struct {
	courses map[string][]scraper.Course
	errors  map[string]error
}

func (s stubScheduleFetcher) FetchSchedule(groupURL string) ([]scraper.Course, error) {
	if err := s.errors[groupURL]; err != nil {
		return nil, err
	}
	return s.courses[groupURL], nil
}

func TestHandleCalendarRequestRejectsPartialSet(t *testing.T) {
	setsPath := filepath.Join(t.TempDir(), "sets.json")
	setsJSON := `{"combined":{"groups":["working","failing"],"courses":[]}}`
	if err := os.WriteFile(setsPath, []byte(setsJSON), 0600); err != nil {
		t.Fatalf("failed to write sets fixture: %v", err)
	}

	originalSetsPath := setsFilePath
	originalFactory := newScheduleFetcher
	setsFilePath = setsPath
	newScheduleFetcher = func() scheduleFetcher {
		return stubScheduleFetcher{
			courses: map[string][]scraper.Course{
				"working.html": {{
					Name: "Software Engineering", DateStr: "21.09.2026", StartTime: "08:15", EndTime: "09:45",
				}},
			},
			errors: map[string]error{"failing.html": fmt.Errorf("upstream unavailable")},
		}
	}
	t.Cleanup(func() {
		setsFilePath = originalSetsPath
		newScheduleFetcher = originalFactory
	})

	req := httptest.NewRequest(http.MethodGet, "/combined.ics", nil)
	recorder := httptest.NewRecorder()
	handleCalendarRequest(recorder, req)

	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("expected status %d for a partial set, got %d", http.StatusBadGateway, recorder.Code)
	}
}

func TestNewCalendarServerHasTimeouts(t *testing.T) {
	server := newCalendarServer("8080")
	if server.ReadHeaderTimeout <= 0 || server.ReadTimeout <= 0 || server.WriteTimeout <= 0 || server.IdleTimeout <= 0 {
		t.Fatalf("expected all server timeouts to be positive: %+v", server)
	}
	if server.WriteTimeout < 30*time.Second {
		t.Fatalf("write timeout is too short for upstream schedule fetching: %s", server.WriteTimeout)
	}
}

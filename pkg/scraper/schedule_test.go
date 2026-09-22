package scraper

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

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

func TestFetchScheduleWritesAndReusesCache(t *testing.T) {
	cacheHome := t.TempDir()
	t.Setenv("HOME", cacheHome)
	t.Setenv("USERPROFILE", cacheHome)

	const scheduleHTML = `
		<div class="event-popover">
			<div class="header">
				<p class="title">Software Engineering</p>
				<p class="description">Vorlesung</p>
			</div>
			<div class="content">
				<div class="part">
					<img src="clock.svg">
					<div class="item">
						<p class="title">21.09.2026 (Montag)</p>
						<p class="description">08:15 Uhr - 09:45 Uhr</p>
					</div>
				</div>
			</div>
		</div>`

	requests := 0
	client := NewClient()
	client.httpClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(scheduleHTML)),
			Header:     make(http.Header),
		}, nil
	})}

	first, err := client.FetchSchedule("test-group.html")
	if err != nil {
		t.Fatalf("first FetchSchedule failed: %v", err)
	}
	second, err := client.FetchSchedule("test-group.html")
	if err != nil {
		t.Fatalf("cached FetchSchedule failed: %v", err)
	}

	if requests != 1 {
		t.Fatalf("expected one HTTP request followed by a cache hit, got %d requests", requests)
	}
	if len(first) != 1 || len(second) != 1 || second[0].Name != first[0].Name {
		t.Fatalf("cached courses do not match fetched courses: first=%+v second=%+v", first, second)
	}
}

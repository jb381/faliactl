package transit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClient_FetchJourneysByArrival(t *testing.T) {
	// Mock JSON Response representing a typical HAFAS journey payload
	mockJSON := `{
		"journeys": [
			{
				"legs": [
					{
						"origin": {"name": "Home Station"},
						"destination": {"name": "Transfer Station"},
						"departure": "2026-02-25T08:00:00+01:00",
						"arrival": "2026-02-25T08:15:00+01:00",
						"line": {"name": "Bus 420"}
					},
					{
						"origin": {"name": "Transfer Station"},
						"destination": {"name": "Campus"},
						"departure": "2026-02-25T08:20:00+01:00",
						"arrival": "2026-02-25T08:45:00+01:00",
						"line": {"name": "Tram 1"}
					}
				]
			}
		]
	}`

	// Create a local mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify query parameters
		if r.URL.Query().Get("from") != "123" {
			t.Errorf("expected 'from' parameter 123, got %s", r.URL.Query().Get("from"))
		}
		if r.URL.Query().Get("to") != "456" {
			t.Errorf("expected 'to' parameter 456, got %s", r.URL.Query().Get("to"))
		}
		if r.URL.Query().Get("arrival") == "" {
			t.Errorf("expected 'arrival' parameter to be set")
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockJSON))
	}))
	defer server.Close()

	// Initialize the custom client with the mock backend URL
	// We have to temporarily override the global baseURL or inject it.
	// Since baseURL is hardcoded to "https://v6.db.transport.rest", we will inject a custom HTTP Client transport
	// or modify the baseURL string. Let's create an overlay in the struct if possible, or just modify the global var for testing.

	// Temporarily override the unexported global baseURL string
	originalBaseURL := baseURL
	baseURL = server.URL
	defer func() { baseURL = originalBaseURL }()

	client := NewClient()

	loc, _ := time.LoadLocation("Europe/Berlin")
	arrivalTime := time.Date(2026, 2, 25, 9, 0, 0, 0, loc)

	journeys, err := client.FetchJourneysByArrival("123", "456", arrivalTime)
	if err != nil {
		t.Fatalf("unexpected error fetching mocked journeys: %v", err)
	}

	if len(journeys) != 1 {
		t.Fatalf("expected 1 journey, got %d", len(journeys))
	}

	journey := journeys[0]
	if len(journey.Legs) != 2 {
		t.Fatalf("expected 2 legs, got %d", len(journey.Legs))
	}

	if journey.Legs[1].Destination.Name != "Campus" {
		t.Errorf("expected final destination 'Campus', got '%s'", journey.Legs[1].Destination.Name)
	}
}

func TestClient_GetWithRetries_Success(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			// Simulate 503 Gateway Timeout twice
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	client := NewClient()

	// getWithRetries is unexported
	resp, err := client.getWithRetries(server.URL)
	if err != nil {
		t.Fatalf("expected robust retry to succeed on 3rd attempt, got error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d", resp.StatusCode)
	}
	if attempts != 3 {
		t.Errorf("expected exactly 3 attempts, got %d", attempts)
	}
}

func TestClient_GetWithRetries_Fail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always fail
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	client := NewClient()

	_, err := client.getWithRetries(server.URL)
	if err == nil {
		t.Fatalf("expected robust retry to completely fail after 3 attempts, but got nil error")
	}
}

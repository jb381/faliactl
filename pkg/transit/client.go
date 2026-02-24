package transit

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

var baseURL = "https://v6.db.transport.rest"

// Client interacts with the HAFAS DB API
type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// getWithRetries attempts an HTTP GET request up to 3 times for 503/504/timeout errors
func (c *Client) getWithRetries(reqURL string) (*http.Response, error) {
	var lastErr error
	var resp *http.Response

	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequest("GET", reqURL, nil)
		if err != nil {
			return nil, err
		}
		// Public APIs often block default Go user agents
		req.Header.Set("User-Agent", "faliactl-student-project/1.0 (https://github.com/jb381/faliactl)")

		resp, lastErr = c.httpClient.Do(req)

		// If request succeeded but gave a transient error code, also retry
		if lastErr == nil && (resp.StatusCode == 503 || resp.StatusCode == 504 || resp.StatusCode == 502) {
			resp.Body.Close()
			lastErr = fmt.Errorf("transient status code: %d", resp.StatusCode)
		} else if lastErr == nil {
			return resp, nil
		}

		if attempt < 2 {
			fmt.Printf("\r\033[K[Transit API] Network congested, retrying... (Attempt %d/3)\n", attempt+1)
		}

		time.Sleep(time.Duration(attempt+1) * time.Second)
	}

	return nil, fmt.Errorf("failed after 3 attempts: %v", lastErr)
}

// FetchLocations searches for transit stops matching a text query
func (c *Client) FetchLocations(query string) ([]Location, error) {
	// Query parameters
	encodedQuery := url.QueryEscape(query)
	reqURL := fmt.Sprintf("%s/locations?query=%s&results=5", baseURL, encodedQuery)

	resp, err := c.getWithRetries(reqURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch locations: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read location response body: %w", err)
	}

	var locations []Location
	if err := json.Unmarshal(body, &locations); err != nil {
		return nil, fmt.Errorf("failed to decode locations JSON: %w", err)
	}

	// Filter down to just actual stations/stops
	var filtered []Location
	for _, l := range locations {
		if l.Type == "station" || l.Type == "stop" {
			filtered = append(filtered, l)
		}
	}

	return filtered, nil
}

// FetchDepartures gets the next departures for a specific station ID
func (c *Client) FetchDepartures(stationID string, durationMinutes int) ([]Departure, error) {
	reqURL := fmt.Sprintf("%s/stops/%s/departures?duration=%d&results=15", baseURL, stationID, durationMinutes)

	resp, err := c.getWithRetries(reqURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch departures: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read departure response body: %w", err)
	}

	var depResp DepartureResponse
	if err := json.Unmarshal(body, &depResp); err != nil {
		return nil, fmt.Errorf("failed to decode departures JSON: %w", err)
	}

	return depResp.Departures, nil
}

// FetchJourneys plans a trip from a starting station/address ID to a destination ID
func (c *Client) FetchJourneys(fromID string, toID string) ([]Journey, error) {
	reqURL := fmt.Sprintf("%s/journeys?from=%s&to=%s&results=3", baseURL, fromID, toID)

	resp, err := c.getWithRetries(reqURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch journeys: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read journey response body: %w", err)
	}

	var journeyResp JourneyResponse
	if err := json.Unmarshal(body, &journeyResp); err != nil {
		// Log the body if it fails to unmarshal so we can debug the exact JSON shape of the legs
		return nil, fmt.Errorf("failed to decode journey JSON: %w", err)
	}

	return journeyResp.Journeys, nil
}

// FetchJourneysByArrival plans a trip from a starting station ID to a destination ID, arriving before a specific time
func (c *Client) FetchJourneysByArrival(fromID string, toID string, arrival time.Time) ([]Journey, error) {
	encodedArrival := url.QueryEscape(arrival.Format(time.RFC3339))
	reqURL := fmt.Sprintf("%s/journeys?from=%s&to=%s&arrival=%s&results=3", baseURL, fromID, toID, encodedArrival)

	resp, err := c.getWithRetries(reqURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch journeys by arrival: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read journey response body: %w", err)
	}

	var journeyResp JourneyResponse
	if err := json.Unmarshal(body, &journeyResp); err != nil {
		return nil, fmt.Errorf("failed to decode journey JSON: %w", err)
	}

	return journeyResp.Journeys, nil
}

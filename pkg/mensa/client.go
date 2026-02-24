package mensa

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

var baseURL = "https://sls.api.stw-on.de/v1"

// Client handles HTTP requests to the Mensa API
type Client struct {
	httpClient *http.Client
}

// NewClient creates a new API client
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// FetchLocations retrieves all available Mensa locations
func (c *Client) FetchLocations() ([]Location, error) {
	url := fmt.Sprintf("%s/location", baseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "faliactl/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch locations: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var allLocations []Location
	if err := json.NewDecoder(resp.Body).Decode(&allLocations); err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %w", err)
	}

	var validLocations []Location
	for _, loc := range allLocations {
		if len(loc.OpeningHours) > 0 {
			validLocations = append(validLocations, loc)
		}
	}

	return validLocations, nil
}

// FetchMenu retrieves the menu for a given location ID on a specific date (YYYY-MM-DD format)
func (c *Client) FetchMenu(locationID int, date string) (*MenuResponse, error) {
	url := fmt.Sprintf("%s/locations/%d/menu/%s", baseURL, locationID, date)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "faliactl/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch menu: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("no menu available for this date/location")
	} else if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var menuResp MenuResponse
	if err := json.NewDecoder(resp.Body).Decode(&menuResp); err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %w", err)
	}

	return &menuResp, nil
}

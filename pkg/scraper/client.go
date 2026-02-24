package scraper

import (
	"fmt"
	"net/http"
	"time"
)

const baseURL = "https://intranet-i.ostfalia.de/fips/stundenplan"

// Client handles HTTP requests to the Ostfalia schedule website
type Client struct {
	httpClient *http.Client
}

// NewClient creates a new scraper client
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Get fetches the given URL and returns the HTTP response
func (c *Client) Get(path string) (*http.Response, error) {
	url := fmt.Sprintf("%s/%s", baseURL, path)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Add expected headers
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s: %w", url, err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code %d when fetching %s", resp.StatusCode, url)
	}

	return resp, nil
}

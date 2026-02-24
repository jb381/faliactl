package transit

import "time"

// LocationResponse represents the array returned by /locations
type Location struct {
	Type      string  `json:"type"`
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Latitude  float64 `json:"location.latitude"`
	Longitude float64 `json:"location.longitude"`
}

// DepartureResponse represents the object returned by /stops/{id}/departures
type DepartureResponse struct {
	Departures []Departure `json:"departures"`
}

// Departure represents a single transport leaving a station
type Departure struct {
	When      time.Time `json:"when"`
	Direction string    `json:"direction"`
	Line      Line      `json:"line"`
	Delay     *int      `json:"delay"`
	Platform  *string   `json:"platform"`
}

// Line holds the information about the specific bus/train
type Line struct {
	Name    string `json:"name"`
	Product string `json:"productName"` // e.g. "Bus", "RB"
}

// JourneyResponse represents the full route from A to B returned by /journeys
type JourneyResponse struct {
	Journeys []Journey `json:"journeys"`
}

// Journey represents a start-to-finish trip, potentially with transfers
type Journey struct {
	Legs []Leg `json:"legs"`
}

// Leg is a single continuous part of a journey (e.g., walking, or one bus ride)
type Leg struct {
	Origin      Location  `json:"origin"`
	Destination Location  `json:"destination"`
	Departure   time.Time `json:"departure"`
	Arrival     time.Time `json:"arrival"`
	Line        *Line     `json:"line,omitempty"`
	Walking     bool      `json:"walking,omitempty"`
}

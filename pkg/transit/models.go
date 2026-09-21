package transit

import (
	"fmt"
	"time"
)

var berlinLocation = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		// Keep transit output usable on minimal systems without zoneinfo. The
		// Docker image installs tzdata, so this is only a defensive fallback.
		return time.FixedZone("CET", 60*60)
	}
	return loc
}()

// BerlinLocation returns the timezone used by Ostfalia and its transit network.
func BerlinLocation() *time.Location {
	return berlinLocation
}

// InBerlin converts a timestamp to the timezone used by the transit UI.
func InBerlin(t time.Time) time.Time {
	return t.In(berlinLocation)
}

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

// Validate rejects incomplete upstream journey data before presentation code
// indexes into the first or last leg.
func (j Journey) Validate() error {
	if len(j.Legs) == 0 {
		return fmt.Errorf("journey has no legs")
	}

	for i, leg := range j.Legs {
		if leg.Departure.IsZero() {
			return fmt.Errorf("journey leg %d has no departure time", i)
		}
		if leg.Arrival.IsZero() {
			return fmt.Errorf("journey leg %d has no arrival time", i)
		}
		if leg.Origin.Name == "" {
			return fmt.Errorf("journey leg %d has no origin", i)
		}
		if leg.Destination.Name == "" {
			return fmt.Errorf("journey leg %d has no destination", i)
		}
	}

	return nil
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

package transit

import (
	"sort"
)

// SummarizedRoute holds the next few departures for a unique line and destination.
type SummarizedRoute struct {
	LineName   string
	Direction  string
	Departures []Departure
}

// SummarizeDepartures sorts departures by actual time and groups them by Line and Direction,
// limiting the output to maxPerRoute recent departures per unique route.
// This prevents high-frequency routes from spamming the UI out of order.
func SummarizeDepartures(deps []Departure, maxPerRoute int) []SummarizedRoute {
	// Filter out any invalid times just in case
	var valid []Departure
	for _, d := range deps {
		if !d.When.IsZero() {
			valid = append(valid, d)
		}
	}

	// Strictly sort all departures by effective departure time
	sort.Slice(valid, func(i, j int) bool {
		return valid[i].When.Before(valid[j].When)
	})

	// Group them
	routeMap := make(map[string]*SummarizedRoute)
	var routeKeys []string // to maintain order of First Appearance (which is chronological now)

	for _, d := range valid {
		key := d.Line.Name + "|" + d.Direction
		if _, exists := routeMap[key]; !exists {
			routeMap[key] = &SummarizedRoute{
				LineName:  d.Line.Name,
				Direction: d.Direction,
			}
			routeKeys = append(routeKeys, key)
		}

		if len(routeMap[key].Departures) < maxPerRoute {
			routeMap[key].Departures = append(routeMap[key].Departures, d)
		}
	}

	var result []SummarizedRoute
	for _, key := range routeKeys {
		result = append(result, *routeMap[key])
	}

	return result
}

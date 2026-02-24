package exporter

import (
	"bytes"
	"strings"
	"testing"

	"faliactl/pkg/scraper"
)

func TestGenerateICS(t *testing.T) {
	courses := []scraper.Course{
		{
			Name:      "Lineare Algebra",
			Type:      "DT+WI S1",
			DateStr:   "04.03.2026 (Mittwoch)",
			StartTime: "08:15",
			EndTime:   "09:45",
			Room:      "WF-EX-7/3",
			GroupStr:  "DITR 2. Sem., WI 2. Sem.",
		},
	}

	var buf bytes.Buffer
	err := GenerateICS(courses, &buf)
	if err != nil {
		t.Fatalf("GenerateICS failed: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "SUMMARY:Lineare Algebra") {
		t.Errorf("Expected ICS to contain course summary, got: \n%s", output)
	}

	if !strings.Contains(output, "LOCATION:WF-EX-7/3") {
		t.Errorf("Expected ICS to contain room location")
	}

	// 04-Mar-2026 08:15 Berlin time is 07:15 UTC.
	if !strings.Contains(output, "DTSTART:20260304T071500Z") {
		t.Errorf("Expected start time string in ICS (should be UTC), got: \n%s", output)
	}
}

package mensa

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestClient_FetchLocations_Mock(t *testing.T) {
	mockResponse := `[
		{
			"id": 101,
			"name": "Mensa 1 TU Braunschweig",
			"address": {
				"city": "Braunschweig"
			},
			"opening_hours": [
				{"start_day": 1}
			]
		},
		{
			"id": 102,
			"name": "Closed Branch",
			"address": {
				"city": "Wolfenbüttel"
			},
			"opening_hours": []
		}
	]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	originalBaseURL := baseURL
	baseURL = server.URL
	defer func() { baseURL = originalBaseURL }()

	client := NewClient()

	locs, err := client.FetchLocations()
	if err != nil {
		t.Fatalf("unexpected error fetching mocked locations: %v", err)
	}

	// Should only return 1 location because the second one has empty opening_hours
	if len(locs) != 1 {
		t.Fatalf("expected 1 valid location, got %d", len(locs))
	}

	if locs[0].ID != 101 {
		t.Errorf("expected location ID 101, got %d", locs[0].ID)
	}
	if locs[0].Address.City != "Braunschweig" {
		t.Errorf("expected city Braunschweig, got %s", locs[0].Address.City)
	}
}

func TestClient_FetchMenu_Mock(t *testing.T) {
	mockResponse := `{
		"meals": [
			{
				"id": 5001,
				"name": "Vegan Schnitzel",
				"date": "2026-02-25",
				"price": {
					"student": "2.50",
					"employee": "4.00",
					"guest": "5.50"
				},
				"lane": {"name": "Counter 1"},
				"tags": {
					"categories": [{"name": "Vegan"}]
				}
			}
		],
		"announcements": []
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	originalBaseURL := baseURL
	baseURL = server.URL
	defer func() { baseURL = originalBaseURL }()

	client := NewClient()

	menu, err := client.FetchMenu(101, "2026-02-25")
	if err != nil {
		t.Fatalf("unexpected error fetching mocked menu: %v", err)
	}

	if menu == nil || len(menu.Meals) != 1 {
		t.Fatalf("expected 1 meal, got %d", len(menu.Meals))
	}

	meal := menu.Meals[0]
	if meal.Name != "Vegan Schnitzel" {
		t.Errorf("expected meal 'Vegan Schnitzel', got %s", meal.Name)
	}
	if meal.Price.Student != "2.50" {
		t.Errorf("expected student price 2.50, got %s", meal.Price.Student)
	}

	expectedTags := Tags{
		Categories: []Category{{Name: "Vegan"}},
	}
	if !reflect.DeepEqual(meal.Tags.Categories, expectedTags.Categories) {
		t.Errorf("tags mismatch. got: %+v", meal.Tags.Categories)
	}
}

func TestClient_FetchMenu_404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	originalBaseURL := baseURL
	baseURL = server.URL
	defer func() { baseURL = originalBaseURL }()

	client := NewClient()

	_, err := client.FetchMenu(999, "2026-02-25")
	if err == nil || err.Error() != "no menu available for this date/location" {
		t.Fatalf("expected 404 message 'no menu available...', got error: %v", err)
	}
}

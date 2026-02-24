package mensa

// MenuResponse is the top level JSON response from the menu endpoint
type MenuResponse struct {
	Announcements []Announcement `json:"announcements"`
	Meals         []Meal         `json:"meals"`
}

// OpeningHour represents the schedule of a Mensa
type OpeningHour struct {
	Time      string `json:"time"`
	StartDay  int    `json:"start_day"`
	EndDay    int    `json:"end_day"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

// Location represents a Mensa cafeteria
type Location struct {
	ID           int           `json:"id"`
	Name         string        `json:"name"`
	Address      Address       `json:"address"`
	OpeningHours []OpeningHour `json:"opening_hours"`
}

// Address contains physical location data to determine the campus
type Address struct {
	City string `json:"city"`
}

// Announcement alerts students if a mensa is closed
type Announcement struct {
	ID        int    `json:"id"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Text      string `json:"text"`
	Closed    bool   `json:"closed"`
}

// Meal represents a single food item
type Meal struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Date  string `json:"date"`
	Price Price  `json:"price"`
	Lane  Lane   `json:"lane"`
	Tags  Tags   `json:"tags"`
}

// Price holds the cost variants
type Price struct {
	Student  string `json:"student"`
	Employee string `json:"employee"`
	Guest    string `json:"guest"`
}

// Lane represents the counter where the meal is served
type Lane struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Tags hold diet or allergen info
type Tags struct {
	Categories []Category `json:"categories"`
	Allergens  []Category `json:"allergens"`
	Additives  []Category `json:"additives"`
	Special    []Category `json:"special"`
}

// Category represents a dietary classification (e.g. Vegan)
type Category struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

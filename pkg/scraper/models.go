package scraper

// Group represents a study group or program (e.g., "Bachelor of Science Digital Technologies")
type Group struct {
	Name string
	URL  string
}

// Course represents a single course block in a timetable
type Course struct {
	Name      string
	Type      string
	DateStr   string // Raw string e.g. "04.03.2026 (Mittwoch)"
	StartTime string // "08:15"
	EndTime   string // "09:45"
	Room      string // "WF-EX-7/3"
	GroupStr  string // Which groups this course belongs to
}

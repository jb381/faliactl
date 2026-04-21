package cmd

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"faliactl/pkg/exporter"
	"faliactl/pkg/scraper"

	"github.com/spf13/cobra"
)

var setsFilePath string

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start an HTTP server to serve dynamic ICS calendars",
	Long:  `Starts a web server. You can subscribe to dynamic calendars via URL, e.g., http://localhost:8080/161902.ics`,
	RunE: func(cmd *cobra.Command, args []string) error {
		port, _ := cmd.Flags().GetString("port")
		setsFilePath, _ = cmd.Flags().GetString("sets")

		http.HandleFunc("/", handleCalendarRequest)

		fmt.Printf("Starting server on port %s...\n", port)
		fmt.Printf("Using sets file: %s (if exists)\n", setsFilePath)
		fmt.Printf("Subscribe to calendars at http://localhost:%s/<group_or_set>.ics\n", port)
		return http.ListenAndServe(":"+port, nil)
	},
}

func handleCalendarRequest(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/health" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
		return
	}

	// Extract identifier from path
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" || path == "favicon.ico" {
		http.NotFound(w, r)
		return
	}

	identifier := strings.TrimSuffix(path, ".ics")
	log.Printf("Received request for identifier %s from %s", identifier, r.RemoteAddr)

	client := scraper.NewClient()
	var allCourses []scraper.Course

	// Check if identifier matches a subscription set
	sets, err := loadSetsConfig(setsFilePath)
	if err != nil {
		log.Printf("Warning: failed to load sets config: %v", err)
	}

	if set, ok := sets[identifier]; ok {
		// It's a set
		seenEvent := make(map[string]bool)
		for _, group := range set.Groups {
			urlPath := group
			if !strings.HasSuffix(urlPath, ".html") {
				urlPath += ".html"
			}
			groupCourses, fetchErr := client.FetchSchedule(urlPath)
			if fetchErr != nil {
				log.Printf("Error fetching schedule for group %s in set %s: %v", group, identifier, fetchErr)
				continue
			}
			for _, c := range groupCourses {
				key := fmt.Sprintf("%s|%s|%s|%s", c.Name, c.DateStr, c.StartTime, c.EndTime)
				if !seenEvent[key] {
					seenEvent[key] = true
					allCourses = append(allCourses, c)
				}
			}
		}

		// Filter courses if specific ones are defined
		if len(set.Courses) > 0 {
			courseMap := make(map[string]bool)
			for _, name := range set.Courses {
				courseMap[name] = true
			}

			var filtered []scraper.Course
			for _, c := range allCourses {
				if courseMap[c.Name] {
					filtered = append(filtered, c)
				}
			}
			allCourses = filtered
		}
	} else {
		// Fallback: Treat as a single group URL
		urlPath := identifier
		if !strings.HasSuffix(urlPath, ".html") {
			urlPath += ".html"
		}
		var fetchErr error
		allCourses, fetchErr = client.FetchSchedule(urlPath)
		if fetchErr != nil {
			log.Printf("Error fetching schedule for %s: %v\n", identifier, fetchErr)
			http.Error(w, "Failed to fetch schedule", http.StatusInternalServerError)
			return
		}
	}

	if len(allCourses) == 0 {
		http.NotFound(w, r)
		return
	}

	// Set headers for ICS file download / subscription
	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.ics\"", identifier))
	// Encourage caching clients (like Google Calendar) not to over-poll (12 hours)
	w.Header().Set("Cache-Control", "public, max-age=43200")

	err = exporter.GenerateICS(allCourses, w)
	if err != nil {
		log.Printf("Error generating ICS for %s: %v\n", identifier, err)
	} else {
		log.Printf("Successfully served calendar for %s\n", identifier)
	}
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().StringP("port", "p", "8080", "Port to listen on")
	serveCmd.Flags().StringP("sets", "s", "sets.json", "Path to sets configuration file")
}

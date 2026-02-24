package scraper

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestCacheReadWrite(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "faliactl-cache-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Setenv("HOME", tempDir)
	t.Setenv("USERPROFILE", tempDir)

	groupURL := "12345.html"

	// 1. Read non-existent cache
	courses, ok := readCache(groupURL)
	if ok || courses != nil {
		t.Errorf("expected readCache to fail for non-existent cache, but got success")
	}

	// 2. Write cache
	testCourses := []Course{
		{
			Name:      "Testing 101",
			DateStr:   "24.02.2026",
			StartTime: "10:00",
			EndTime:   "11:30",
			Room:      "WF Exer",
		},
	}
	writeCache(groupURL, testCourses)

	// Verify file was created
	expectedPath := filepath.Join(tempDir, ".faliactl_cache", "12345.html.json")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("expected cache file to be created at %s", expectedPath)
	}

	// 3. Read existing valid cache
	loadedCourses, ok := readCache(groupURL)
	if !ok {
		t.Fatalf("expected readCache to succeed for existing cache, but failed")
	}
	if !reflect.DeepEqual(testCourses, loadedCourses) {
		t.Errorf("loaded courses do not match written courses.\nGot: %+v\nExpected: %+v", loadedCourses, testCourses)
	}
}

func TestCacheExpiration(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "faliactl-cache-exp-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Setenv("HOME", tempDir)
	t.Setenv("USERPROFILE", tempDir)

	groupURL := "expired.html"

	// Write cache normally first (so we guarantee directory structure)
	writeCache(groupURL, []Course{})

	// Now manually modify the timestamp in the file to simulate expiration
	cachePath, _ := getCachePath(groupURL)

	entry := CacheEntry{
		Timestamp: time.Now().Add(-24 * time.Hour), // Expired (older than 12h)
		Courses:   []Course{{Name: "Old"}},
	}

	// Open file and overwrite directly ignoring proper locking since tests are serial here
	f, err := os.OpenFile(cachePath, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		t.Fatalf("failed to open cache file for modification: %v", err)
	}

	// Convert to JSON
	importJSON, _ := json.Marshal(entry)
	f.Write(importJSON)
	f.Close()

	// Try reading
	_, ok := readCache(groupURL)
	if ok {
		t.Errorf("expected readCache to reject expired cache (24h old, limit is 12h), but it incorrectly succeeded")
	}
}

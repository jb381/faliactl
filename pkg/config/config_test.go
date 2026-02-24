package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestConfigLoadSave(t *testing.T) {
	// Create a temporary directory to act as the user's home directory
	tempDir, err := os.MkdirTemp("", "faliactl-config-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) // cleanup

	// Override the home directory environment variable for testing
	t.Setenv("HOME", tempDir)
	t.Setenv("USERPROFILE", tempDir) // For Windows compatibility in tests

	// 1. Test Load with no existing file
	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error when loading missing config, got: %v", err)
	}
	if cfg == nil {
		t.Fatalf("expected empty config to be returned, got nil")
	}

	// 2. Modify and Save the config
	cfg.HomeAddress = "Test Address 123"
	cfg.HomeStationID = "12345"
	cfg.SavedGroupURLs = []string{"test_url.html"}
	cfg.SavedCourses = []string{"Course A", "Course B"}
	cfg.DefaultCampus = "wolfenbuettel"

	err = Save(cfg)
	if err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Verify the file was actually created
	configPath := filepath.Join(tempDir, ".faliactl.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("expected config file to be created at %s", configPath)
	}

	// 3. Test Load with existing file
	loadedCfg, err := Load()
	if err != nil {
		t.Fatalf("failed to load existing config: %v", err)
	}

	// Compare loaded config with saved config
	if !reflect.DeepEqual(cfg, loadedCfg) {
		t.Errorf("loaded config does not match saved config.\nGot: %+v\nExpected: %+v", loadedCfg, cfg)
	}
}

func TestConfigParseError(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "faliactl-config-err-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Setenv("HOME", tempDir)
	t.Setenv("USERPROFILE", tempDir)

	// Write invalid JSON to the config file
	configPath := filepath.Join(tempDir, ".faliactl.json")
	err = os.WriteFile(configPath, []byte("invalid json { content"), 0644)
	if err != nil {
		t.Fatalf("failed to write invalid json: %v", err)
	}

	// Attempt to load the invalid JSON
	_, err = Load()
	if err == nil {
		t.Errorf("expected error when loading invalid json, got nil")
	}
}

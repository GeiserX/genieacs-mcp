package config

import (
	"os"
	"testing"
)

func TestLoadACSConfig_Defaults(t *testing.T) {
	// Clear any existing env vars
	os.Unsetenv("ACS_URL")
	os.Unsetenv("ACS_USER")
	os.Unsetenv("ACS_PASS")

	cfg := LoadACSConfig()

	if cfg.BaseURL != "http://localhost:7557" {
		t.Errorf("expected default BaseURL http://localhost:7557, got %s", cfg.BaseURL)
	}
	if cfg.User != "admin" {
		t.Errorf("expected default User admin, got %s", cfg.User)
	}
	if cfg.Pass != "admin" {
		t.Errorf("expected default Pass admin, got %s", cfg.Pass)
	}
}

func TestLoadACSConfig_EnvOverride(t *testing.T) {
	os.Setenv("ACS_URL", "http://acs.example.com:7557")
	os.Setenv("ACS_USER", "myuser")
	os.Setenv("ACS_PASS", "mypass")
	defer func() {
		os.Unsetenv("ACS_URL")
		os.Unsetenv("ACS_USER")
		os.Unsetenv("ACS_PASS")
	}()

	cfg := LoadACSConfig()

	if cfg.BaseURL != "http://acs.example.com:7557" {
		t.Errorf("expected BaseURL http://acs.example.com:7557, got %s", cfg.BaseURL)
	}
	if cfg.User != "myuser" {
		t.Errorf("expected User myuser, got %s", cfg.User)
	}
	if cfg.Pass != "mypass" {
		t.Errorf("expected Pass mypass, got %s", cfg.Pass)
	}
}

func TestGetEnv_Default(t *testing.T) {
	os.Unsetenv("TEST_NONEXISTENT_VAR")
	val := getEnv("TEST_NONEXISTENT_VAR", "default_value")
	if val != "default_value" {
		t.Errorf("expected default_value, got %s", val)
	}
}

func TestGetEnv_Override(t *testing.T) {
	os.Setenv("TEST_EXISTING_VAR", "override_value")
	defer os.Unsetenv("TEST_EXISTING_VAR")

	val := getEnv("TEST_EXISTING_VAR", "default_value")
	if val != "override_value" {
		t.Errorf("expected override_value, got %s", val)
	}
}

func TestGetEnv_EmptyString(t *testing.T) {
	// Empty string should fall back to default
	os.Setenv("TEST_EMPTY_VAR", "")
	defer os.Unsetenv("TEST_EMPTY_VAR")

	val := getEnv("TEST_EMPTY_VAR", "default_value")
	if val != "default_value" {
		t.Errorf("expected default_value for empty env, got %s", val)
	}
}

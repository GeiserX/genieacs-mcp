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
	if cfg.User != "" {
		t.Errorf("expected empty default User, got %s", cfg.User)
	}
	if cfg.Pass != "" {
		t.Errorf("expected empty default Pass, got %s", cfg.Pass)
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

func TestLoadACSConfig_DeviceLimit_Default(t *testing.T) {
	os.Unsetenv("DEVICE_LIMIT")
	cfg := LoadACSConfig()
	if cfg.DeviceLimit != 500 {
		t.Errorf("expected default DeviceLimit 500, got %d", cfg.DeviceLimit)
	}
}

func TestLoadACSConfig_DeviceLimit_Valid(t *testing.T) {
	os.Setenv("DEVICE_LIMIT", "1000")
	defer os.Unsetenv("DEVICE_LIMIT")
	cfg := LoadACSConfig()
	if cfg.DeviceLimit != 1000 {
		t.Errorf("expected DeviceLimit 1000, got %d", cfg.DeviceLimit)
	}
}

func TestLoadACSConfig_DeviceLimit_Invalid(t *testing.T) {
	os.Setenv("DEVICE_LIMIT", "notanumber")
	defer os.Unsetenv("DEVICE_LIMIT")
	cfg := LoadACSConfig()
	if cfg.DeviceLimit != 500 {
		t.Errorf("expected fallback DeviceLimit 500 for invalid input, got %d", cfg.DeviceLimit)
	}
}

func TestLoadACSConfig_DeviceLimit_Negative(t *testing.T) {
	os.Setenv("DEVICE_LIMIT", "-5")
	defer os.Unsetenv("DEVICE_LIMIT")
	cfg := LoadACSConfig()
	if cfg.DeviceLimit != 500 {
		t.Errorf("expected fallback DeviceLimit 500 for negative input, got %d", cfg.DeviceLimit)
	}
}

func TestLoadACSConfig_DeviceLimit_Zero(t *testing.T) {
	os.Setenv("DEVICE_LIMIT", "0")
	defer os.Unsetenv("DEVICE_LIMIT")
	cfg := LoadACSConfig()
	if cfg.DeviceLimit != 500 {
		t.Errorf("expected fallback DeviceLimit 500 for zero input, got %d", cfg.DeviceLimit)
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

func TestGetenv_UnexpandedPlaceholderReadsAsUnset(t *testing.T) {
	cases := map[string]string{
		"${ACS_PASS}":       "",
		"${MCP_AUTH_TOKEN}": "",
		" ${OTHER_NAME} ":   "",
		"s3cret":            "s3cret",
		"pw${ACS_PASS}":     "pw${ACS_PASS}",
		"${not a name}":     "${not a name}",
		"$ACS_PASS":         "$ACS_PASS",
		"":                  "",
	}
	for in, want := range cases {
		t.Setenv("GENIEACS_MCP_TEST_VAR", in)
		if got := Getenv("GENIEACS_MCP_TEST_VAR"); got != want {
			t.Errorf("Getenv with %q = %q, want %q", in, got, want)
		}
	}
}

func TestLoadACSConfig_PlaceholderValuesAreUnset(t *testing.T) {
	for _, k := range []string{"ACS_URL", "ACS_USER", "ACS_PASS", "DEVICE_LIMIT"} {
		t.Setenv(k, "${"+k+"}")
	}

	cfg := LoadACSConfig()
	if cfg.BaseURL != "http://localhost:7557" {
		t.Errorf("BaseURL = %q, want the default", cfg.BaseURL)
	}
	if cfg.User != "" || cfg.Pass != "" {
		t.Errorf("placeholder leaked into config: %+v", cfg)
	}
	if cfg.DeviceLimit != 500 {
		t.Errorf("DeviceLimit = %d, want the default 500", cfg.DeviceLimit)
	}
}

func TestLoadACSConfig_LimitsAndTimeout(t *testing.T) {
	t.Setenv("ACS_MAX_RESPONSE_BYTES", "1048576")
	t.Setenv("ACS_TIMEOUT", "5")
	cfg := LoadACSConfig()
	if cfg.MaxResponseBytes != 1<<20 || cfg.Timeout.Seconds() != 5 {
		t.Errorf("got max %d, timeout %v; want 1048576 and 5s", cfg.MaxResponseBytes, cfg.Timeout)
	}
	t.Setenv("ACS_MAX_RESPONSE_BYTES", "lots")
	t.Setenv("ACS_TIMEOUT", "-3")
	cfg = LoadACSConfig()
	if cfg.MaxResponseBytes != 0 || cfg.Timeout != 0 {
		t.Errorf("invalid values were not ignored: max %d, timeout %v", cfg.MaxResponseBytes, cfg.Timeout)
	}
}

func TestCleartextToRemote(t *testing.T) {
	for base, want := range map[string]bool{
		"http://localhost:7557":      false,
		"http://127.0.0.1:7557":      false,
		"http://[::1]:7557":          false,
		"https://acs.example.com":    false,
		"http://acs.example.com":     true,
		"http://192.168.1.10:7557":   true,
		"HTTP://genieacs:7557/nbi/x": true,
	} {
		if got := cleartextToRemote(base); got != want {
			t.Errorf("cleartextToRemote(%q) = %v, want %v", base, got, want)
		}
	}
}

func TestLoadToolPolicy(t *testing.T) {
	t.Setenv("GENIEACS_READ_ONLY", "")
	t.Setenv("GENIEACS_TOOLS", "")
	if p := LoadToolPolicy(); p.ReadOnly || len(p.Allow) != 0 {
		t.Errorf("empty env gave %+v, want everything allowed", p)
	}
	t.Setenv("GENIEACS_READ_ONLY", "TRUE")
	t.Setenv("GENIEACS_TOOLS", " search_devices, ,get_parameter ")
	p := LoadToolPolicy()
	if !p.ReadOnly || len(p.Allow) != 2 || p.Allow[0] != "search_devices" || p.Allow[1] != "get_parameter" {
		t.Errorf("got %+v", p)
	}
	t.Setenv("GENIEACS_READ_ONLY", "${GENIEACS_READ_ONLY}")
	if LoadToolPolicy().ReadOnly {
		t.Error("an unexpanded placeholder turned read-only mode on")
	}
}

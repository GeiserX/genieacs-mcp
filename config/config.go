package config

import (
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

func init() {
	// Load .env in the working directory; ignore error if the file is absent.
	_ = godotenv.Load()
}

type ACSConfig struct {
	BaseURL     string
	User        string
	Pass        string
	DeviceLimit int
}

func LoadACSConfig() ACSConfig {
	limit := 500
	if v := Getenv("DEVICE_LIMIT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	user := Getenv("ACS_USER")
	pass := Getenv("ACS_PASS")
	if user == "" && pass == "" {
		log.Println("WARNING: ACS_USER and ACS_PASS not set — connecting to GenieACS NBI without authentication")
	}
	return ACSConfig{
		BaseURL:     getEnv("ACS_URL", "http://localhost:7557"),
		User:        user,
		Pass:        pass,
		DeviceLimit: limit,
	}
}

// placeholder matches a value that is nothing but an unexpanded variable reference, like "${ACS_PASS}".
var placeholder = regexp.MustCompile(`^\$\{[A-Za-z_][A-Za-z0-9_]*\}$`)

// Getenv is os.Getenv, except that a value which is only an unexpanded placeholder such as
// "${ACS_PASS}" reads as unset. Some MCP clients write that placeholder into the server's
// environment when an optional field is left blank; taken literally it would be sent to
// GenieACS as a password, or used as the auth token of the HTTP transport.
func Getenv(k string) string {
	v := os.Getenv(k)
	if placeholder.MatchString(strings.TrimSpace(v)) {
		return ""
	}
	return v
}

func getEnv(k, d string) string {
	if v := Getenv(k); v != "" {
		return v
	}
	return d
}

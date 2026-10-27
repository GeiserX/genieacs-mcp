// Package config reads the GenieACS connection settings from the environment,
// after loading a .env file from the working directory when one exists.
package config

import (
	"log"
	"net"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

func init() {
	// Load .env in the working directory; ignore error if the file is absent.
	_ = godotenv.Load()
}

// ACSConfig holds what the server needs to reach GenieACS: the NBI base URL,
// the basic-auth user and password, and the most devices the device list
// resource returns.
type ACSConfig struct {
	BaseURL     string
	User        string
	Pass        string
	DeviceLimit int
	// MaxResponseBytes caps one NBI response (ACS_MAX_RESPONSE_BYTES); 0 keeps the client default.
	MaxResponseBytes int64
	// Timeout bounds one NBI request (ACS_TIMEOUT, in seconds); 0 keeps the client default.
	Timeout time.Duration
}

// LoadACSConfig builds an ACSConfig from ACS_URL (default "http://localhost:7557"),
// ACS_USER, ACS_PASS and DEVICE_LIMIT (default 500; values that are not a positive
// integer are ignored), ACS_MAX_RESPONSE_BYTES and ACS_TIMEOUT. It logs a warning
// when neither credential is set, and when credentials would cross the network
// in cleartext.
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
	base := getEnv("ACS_URL", "http://localhost:7557")
	if (user != "" || pass != "") && cleartextToRemote(base) {
		log.Println("WARNING: ACS_URL is plain http:// to a non-loopback host, so ACS_USER and ACS_PASS cross the network unencrypted; use https://")
	}
	return ACSConfig{
		BaseURL:          base,
		User:             user,
		Pass:             pass,
		DeviceLimit:      limit,
		MaxResponseBytes: int64(positiveInt("ACS_MAX_RESPONSE_BYTES")),
		Timeout:          time.Duration(positiveInt("ACS_TIMEOUT")) * time.Second,
	}
}

// positiveInt reads k as a positive integer, or 0 when it is unset or invalid.
func positiveInt(k string) int {
	v := Getenv(k)
	if v == "" {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil || n <= 0 {
		log.Printf("WARNING: %s=%q is not a positive integer; using the default", k, v)
		return 0
	}
	return n
}

// cleartextToRemote reports whether base is an http:// URL to a host other
// than loopback.
func cleartextToRemote(base string) bool {
	u, err := url.Parse(base)
	if err != nil || !strings.EqualFold(u.Scheme, "http") {
		return false
	}
	host := u.Hostname()
	if host == "localhost" {
		return false
	}
	ip := net.ParseIP(host)
	return ip == nil || !ip.IsLoopback()
}

// ToolPolicy says which tools the server registers: GENIEACS_READ_ONLY=true
// keeps only the tools that never change a device or the ACS, and
// GENIEACS_TOOLS, a comma-separated list of tool names, keeps only those.
// Both may be set; a tool must then pass both.
type ToolPolicy struct {
	ReadOnly bool
	Allow    []string
}

// LoadToolPolicy reads GENIEACS_READ_ONLY and GENIEACS_TOOLS.
func LoadToolPolicy() ToolPolicy {
	var p ToolPolicy
	switch strings.ToLower(strings.TrimSpace(Getenv("GENIEACS_READ_ONLY"))) {
	case "1", "true", "yes", "on":
		p.ReadOnly = true
	}
	for _, name := range strings.Split(Getenv("GENIEACS_TOOLS"), ",") {
		if name = strings.TrimSpace(name); name != "" {
			p.Allow = append(p.Allow, name)
		}
	}
	return p
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

package config

import (
	"os"
	"strconv"

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
	if v := os.Getenv("DEVICE_LIMIT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	return ACSConfig{
		BaseURL:     getEnv("ACS_URL", "http://localhost:7557"),
		User:        getEnv("ACS_USER", "admin"),
		Pass:        getEnv("ACS_PASS", "admin"),
		DeviceLimit: limit,
	}
}

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

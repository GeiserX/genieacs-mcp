package config

import (
	"os"

	"github.com/joho/godotenv"
)

func init() {
	// Load .env in the working directory; ignore error if the file is absent.
	_ = godotenv.Load()
}

type ACSConfig struct {
	BaseURL string
	User    string
	Pass    string
}

func LoadACSConfig() ACSConfig {
	return ACSConfig{
		BaseURL: getEnv("ACS_URL", "http://localhost:7557"),
		User:    getEnv("ACS_USER", "admin"),
		Pass:    getEnv("ACS_PASS", "admin"),
	}
}

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

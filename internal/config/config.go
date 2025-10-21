package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all server configuration
type Config struct {
	Port                 string
	DBBaseDir            string
	CatalogDBPath        string
	CORSOrigins          []string
	DefaultQuotaMB       int64
	ExpiryDays           int
	ExpiryCheckInterval  time.Duration
}

// Load reads configuration from environment variables with sensible defaults
func Load() (*Config, error) {
	cfg := &Config{
		Port:          getEnv("PORT", "8080"),
		DBBaseDir:     getEnv("DB_BASE_DIR", "./data"),
		CatalogDBPath: getEnv("CATALOG_DB_PATH", "./data/catalog.db"),
		CORSOrigins:   parseCORSOrigins(getEnv("CORS_ORIGINS", "*")),
	}

	// Parse DEFAULT_QUOTA_MB
	quotaMB, err := strconv.ParseInt(getEnv("DEFAULT_QUOTA_MB", "100"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid DEFAULT_QUOTA_MB: %w", err)
	}
	if quotaMB <= 0 {
		return nil, fmt.Errorf("DEFAULT_QUOTA_MB must be positive, got %d", quotaMB)
	}
	cfg.DefaultQuotaMB = quotaMB

	// Parse EXPIRY_DAYS
	expiryDays, err := strconv.Atoi(getEnv("EXPIRY_DAYS", "30"))
	if err != nil {
		return nil, fmt.Errorf("invalid EXPIRY_DAYS: %w", err)
	}
	if expiryDays <= 0 {
		return nil, fmt.Errorf("EXPIRY_DAYS must be positive, got %d", expiryDays)
	}
	cfg.ExpiryDays = expiryDays

	// Parse EXPIRY_CHECK_INTERVAL
	intervalStr := getEnv("EXPIRY_CHECK_INTERVAL", "24h")
	interval, err := time.ParseDuration(intervalStr)
	if err != nil {
		return nil, fmt.Errorf("invalid EXPIRY_CHECK_INTERVAL: %w", err)
	}
	if interval <= 0 {
		return nil, fmt.Errorf("EXPIRY_CHECK_INTERVAL must be positive, got %s", intervalStr)
	}
	cfg.ExpiryCheckInterval = interval

	return cfg, nil
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// parseCORSOrigins parses a comma-separated list of CORS origins
func parseCORSOrigins(origins string) []string {
	if origins == "*" {
		return []string{"*"}
	}

	var result []string
	for _, origin := range strings.Split(origins, ",") {
		trimmed := strings.TrimSpace(origin)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	if len(result) == 0 {
		return []string{"*"}
	}

	return result
}

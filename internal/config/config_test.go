package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear all relevant environment variables
	clearEnv()
	defer clearEnv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if cfg.Port != "8080" {
		t.Errorf("Port = %s, want 8080", cfg.Port)
	}
	if cfg.DBBaseDir != "./data" {
		t.Errorf("DBBaseDir = %s, want ./data", cfg.DBBaseDir)
	}
	if cfg.CatalogDBPath != "./data/catalog.db" {
		t.Errorf("CatalogDBPath = %s, want ./data/catalog.db", cfg.CatalogDBPath)
	}
	if len(cfg.CORSOrigins) != 1 || cfg.CORSOrigins[0] != "*" {
		t.Errorf("CORSOrigins = %v, want [*]", cfg.CORSOrigins)
	}
	if cfg.DefaultQuotaMB != 100 {
		t.Errorf("DefaultQuotaMB = %d, want 100", cfg.DefaultQuotaMB)
	}
	if cfg.ExpiryDays != 30 {
		t.Errorf("ExpiryDays = %d, want 30", cfg.ExpiryDays)
	}
	if cfg.ExpiryCheckInterval != 24*time.Hour {
		t.Errorf("ExpiryCheckInterval = %v, want 24h", cfg.ExpiryCheckInterval)
	}
}

func TestLoad_CustomValues(t *testing.T) {
	clearEnv()
	defer clearEnv()

	os.Setenv("PORT", "3000")
	os.Setenv("DB_BASE_DIR", "/var/lib/jsondrop")
	os.Setenv("CATALOG_DB_PATH", "/var/lib/jsondrop/catalog.db")
	os.Setenv("CORS_ORIGINS", "https://example.com,https://app.example.com")
	os.Setenv("DEFAULT_QUOTA_MB", "250")
	os.Setenv("EXPIRY_DAYS", "60")
	os.Setenv("EXPIRY_CHECK_INTERVAL", "12h")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if cfg.Port != "3000" {
		t.Errorf("Port = %s, want 3000", cfg.Port)
	}
	if cfg.DBBaseDir != "/var/lib/jsondrop" {
		t.Errorf("DBBaseDir = %s, want /var/lib/jsondrop", cfg.DBBaseDir)
	}
	if cfg.CatalogDBPath != "/var/lib/jsondrop/catalog.db" {
		t.Errorf("CatalogDBPath = %s, want /var/lib/jsondrop/catalog.db", cfg.CatalogDBPath)
	}
	if len(cfg.CORSOrigins) != 2 {
		t.Errorf("len(CORSOrigins) = %d, want 2", len(cfg.CORSOrigins))
	}
	if cfg.CORSOrigins[0] != "https://example.com" {
		t.Errorf("CORSOrigins[0] = %s, want https://example.com", cfg.CORSOrigins[0])
	}
	if cfg.CORSOrigins[1] != "https://app.example.com" {
		t.Errorf("CORSOrigins[1] = %s, want https://app.example.com", cfg.CORSOrigins[1])
	}
	if cfg.DefaultQuotaMB != 250 {
		t.Errorf("DefaultQuotaMB = %d, want 250", cfg.DefaultQuotaMB)
	}
	if cfg.ExpiryDays != 60 {
		t.Errorf("ExpiryDays = %d, want 60", cfg.ExpiryDays)
	}
	if cfg.ExpiryCheckInterval != 12*time.Hour {
		t.Errorf("ExpiryCheckInterval = %v, want 12h", cfg.ExpiryCheckInterval)
	}
}

func TestLoad_InvalidQuota(t *testing.T) {
	clearEnv()
	defer clearEnv()

	os.Setenv("DEFAULT_QUOTA_MB", "invalid")

	_, err := Load()
	if err == nil {
		t.Error("Load() error = nil, want error for invalid DEFAULT_QUOTA_MB")
	}
}

func TestLoad_NegativeQuota(t *testing.T) {
	clearEnv()
	defer clearEnv()

	os.Setenv("DEFAULT_QUOTA_MB", "-100")

	_, err := Load()
	if err == nil {
		t.Error("Load() error = nil, want error for negative DEFAULT_QUOTA_MB")
	}
}

func TestLoad_ZeroQuota(t *testing.T) {
	clearEnv()
	defer clearEnv()

	os.Setenv("DEFAULT_QUOTA_MB", "0")

	_, err := Load()
	if err == nil {
		t.Error("Load() error = nil, want error for zero DEFAULT_QUOTA_MB")
	}
}

func TestLoad_InvalidExpiryDays(t *testing.T) {
	clearEnv()
	defer clearEnv()

	os.Setenv("EXPIRY_DAYS", "invalid")

	_, err := Load()
	if err == nil {
		t.Error("Load() error = nil, want error for invalid EXPIRY_DAYS")
	}
}

func TestLoad_NegativeExpiryDays(t *testing.T) {
	clearEnv()
	defer clearEnv()

	os.Setenv("EXPIRY_DAYS", "-30")

	_, err := Load()
	if err == nil {
		t.Error("Load() error = nil, want error for negative EXPIRY_DAYS")
	}
}

func TestLoad_InvalidInterval(t *testing.T) {
	clearEnv()
	defer clearEnv()

	os.Setenv("EXPIRY_CHECK_INTERVAL", "invalid")

	_, err := Load()
	if err == nil {
		t.Error("Load() error = nil, want error for invalid EXPIRY_CHECK_INTERVAL")
	}
}

func TestLoad_NegativeInterval(t *testing.T) {
	clearEnv()
	defer clearEnv()

	os.Setenv("EXPIRY_CHECK_INTERVAL", "-24h")

	_, err := Load()
	if err == nil {
		t.Error("Load() error = nil, want error for negative EXPIRY_CHECK_INTERVAL")
	}
}

func TestParseCORSOrigins_Wildcard(t *testing.T) {
	origins := parseCORSOrigins("*")
	if len(origins) != 1 || origins[0] != "*" {
		t.Errorf("parseCORSOrigins(*) = %v, want [*]", origins)
	}
}

func TestParseCORSOrigins_Multiple(t *testing.T) {
	origins := parseCORSOrigins("https://example.com, https://app.example.com , https://api.example.com")
	expected := []string{"https://example.com", "https://app.example.com", "https://api.example.com"}

	if len(origins) != len(expected) {
		t.Fatalf("len(origins) = %d, want %d", len(origins), len(expected))
	}

	for i, origin := range origins {
		if origin != expected[i] {
			t.Errorf("origins[%d] = %s, want %s", i, origin, expected[i])
		}
	}
}

func TestParseCORSOrigins_Empty(t *testing.T) {
	origins := parseCORSOrigins("")
	if len(origins) != 1 || origins[0] != "*" {
		t.Errorf("parseCORSOrigins(\"\") = %v, want [*]", origins)
	}
}

func TestParseCORSOrigins_WithEmptyItems(t *testing.T) {
	origins := parseCORSOrigins("https://example.com,,https://app.example.com,  ,")
	expected := []string{"https://example.com", "https://app.example.com"}

	if len(origins) != len(expected) {
		t.Fatalf("len(origins) = %d, want %d", len(origins), len(expected))
	}

	for i, origin := range origins {
		if origin != expected[i] {
			t.Errorf("origins[%d] = %s, want %s", i, origin, expected[i])
		}
	}
}

func clearEnv() {
	os.Unsetenv("PORT")
	os.Unsetenv("DB_BASE_DIR")
	os.Unsetenv("CATALOG_DB_PATH")
	os.Unsetenv("CORS_ORIGINS")
	os.Unsetenv("DEFAULT_QUOTA_MB")
	os.Unsetenv("EXPIRY_DAYS")
	os.Unsetenv("EXPIRY_CHECK_INTERVAL")
}

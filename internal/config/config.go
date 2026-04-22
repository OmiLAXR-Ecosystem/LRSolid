package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Port        string
	Environment string

	// Storage
	PostgresDSN string

	// xAPI basics
	DefaultTenant string
	MaxPageLimit  int

	// Admin UI
	DefaultOrg string

	// Auth
	JWTSecret string // Required in production. Set via JWT_SECRET env var.

	// SMTP (for invitation emails)
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string

	// App base URL (used in invite links)
	AppURL string

	// Async statement stats rollup
	StatsRollupEnabled   bool
	StatsRollupIntervalS int
	StatsRollupBatchSize int

	// JS plugins directory (optional). Set via PLUGINS_DIR env var.
	PluginsDir string

	// Export defaults (org settings seed)
	ExportDefaultTargetType string
	ExportS3UploadBaseURL   string
	ExportS3PublicBaseURL   string
	ExportS3AuthToken       string
	ExportS3PathPrefix      string
}

func Load() (Config, error) {
	c := Config{
		Port:                    getenv("PORT", "8080"),
		Environment:             getenv("ENV", "dev"),
		PostgresDSN:             getenv("PG_DSN", ""),
		DefaultTenant:           getenv("DEFAULT_TENANT", "default"),
		MaxPageLimit:            getenvInt("MAX_PAGE_LIMIT", 100),
		DefaultOrg:              getenv("DEFAULT_ORG", "default"),
		JWTSecret:               getenv("JWT_SECRET", ""),
		SMTPHost:                getenv("SMTP_HOST", ""),
		SMTPPort:                getenvInt("SMTP_PORT", 587),
		SMTPUser:                getenv("SMTP_USER", ""),
		SMTPPassword:            getenv("SMTP_PASSWORD", ""),
		SMTPFrom:                getenv("SMTP_FROM", "noreply@localhost"),
		AppURL:                  getenv("APP_URL", "http://localhost:5173"),
		StatsRollupEnabled:      getenvBool("STATS_ROLLUP_ENABLED", true),
		StatsRollupIntervalS:    getenvInt("STATS_ROLLUP_INTERVAL_S", 15),
		StatsRollupBatchSize:    getenvInt("STATS_ROLLUP_BATCH_SIZE", 5000),
		PluginsDir:              getenv("PLUGINS_DIR", ""),
		ExportDefaultTargetType: strings.ToLower(getenv("EXPORT_DEFAULT_TARGET_TYPE", "s3")),
		ExportS3UploadBaseURL:   getenv("EXPORT_S3_UPLOAD_BASE_URL", ""),
		ExportS3PublicBaseURL:   getenv("EXPORT_S3_PUBLIC_BASE_URL", ""),
		ExportS3AuthToken:       getenv("EXPORT_S3_AUTH_TOKEN", ""),
		ExportS3PathPrefix:      getenv("EXPORT_S3_PATH_PREFIX", "exports"),
	}

	if c.PostgresDSN == "" {
		return Config{}, fmt.Errorf("PG_DSN is required")
	}
	if c.MaxPageLimit <= 0 || c.MaxPageLimit > 1000 {
		c.MaxPageLimit = 100
	}
	if c.StatsRollupIntervalS <= 0 {
		c.StatsRollupIntervalS = 15
	}
	if c.StatsRollupBatchSize <= 0 {
		c.StatsRollupBatchSize = 5000
	}
	if c.JWTSecret == "" {
		if c.Environment != "dev" {
			return Config{}, fmt.Errorf("JWT_SECRET is required in non-dev environments")
		}
		// Dev-only fallback — never use in production
		c.JWTSecret = "dev-only-insecure-secret-change-me"
	}
	switch c.ExportDefaultTargetType {
	case "local", "s3", "gdrive", "onedrive", "icloud":
	default:
		c.ExportDefaultTargetType = "s3"
	}
	if strings.TrimSpace(c.ExportS3PathPrefix) == "" {
		c.ExportS3PathPrefix = "exports"
	}
	return c, nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func getenvInt(k string, def int) int {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	var n int
	_, err := fmt.Sscanf(v, "%d", &n)
	if err != nil {
		return def
	}
	return n
}

func getenvBool(k string, def bool) bool {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	switch v {
	case "1", "true", "TRUE", "yes", "YES", "on", "ON":
		return true
	case "0", "false", "FALSE", "no", "NO", "off", "OFF":
		return false
	default:
		return def
	}
}

package config

import (
	"fmt"
	"os"
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
}

func Load() (Config, error) {
	c := Config{
		Port:          getenv("PORT", "8080"),
		Environment:   getenv("ENV", "dev"),
		PostgresDSN:   getenv("PG_DSN", ""),
		DefaultTenant: getenv("DEFAULT_TENANT", "default"),
		MaxPageLimit:  getenvInt("MAX_PAGE_LIMIT", 100),
		DefaultOrg:    getenv("DEFAULT_ORG", "default"),
		JWTSecret:     getenv("JWT_SECRET", ""),
	}

	if c.PostgresDSN == "" {
		return Config{}, fmt.Errorf("PG_DSN is required")
	}
	if c.MaxPageLimit <= 0 || c.MaxPageLimit > 1000 {
		c.MaxPageLimit = 100
	}
	if c.JWTSecret == "" {
		if c.Environment != "dev" {
			return Config{}, fmt.Errorf("JWT_SECRET is required in non-dev environments")
		}
		// Dev-only fallback — never use in production
		c.JWTSecret = "dev-only-insecure-secret-change-me"
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

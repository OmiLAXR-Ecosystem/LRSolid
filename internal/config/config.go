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
	PostgresDSN            string
	DataStoreSecretKey     string
	DataStoreSecretKeyFile string

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

	// App base URL (used in invite links and SSO redirect URIs)
	AppURL string

	// Local password login. LocalLoginExplicit is true when LOCAL_LOGIN_ENABLED is set.
	LocalLoginEnabled  bool
	LocalLoginExplicit bool

	// OpenID Connect provider seeded for the default organisation.
	OIDCEnabled              bool
	OIDCSlug                 string
	OIDCDisplayName          string
	OIDCIssuer               string
	OIDCClientID             string
	OIDCClientSecret         string
	OIDCScopes               string
	OIDCAutoProvision        bool
	OIDCDefaultRole          string
	OIDCRequireEmailVerified bool

	// SAML 2.0 / Shibboleth service provider seeded for the default organisation.
	SAMLEnabled        bool
	SAMLSlug           string
	SAMLDisplayName    string
	SAMLIDPMetadataURL string
	SAMLIDPMetadataXML string
	SAMLSPEntityID     string
	SAMLSPCertPEM      string
	SAMLSPKeyPEM       string
	SAMLAutoProvision  bool
	SAMLDefaultRole    string

	// Async statement stats rollup
	StatsRollupEnabled   bool
	StatsRollupIntervalS int
	StatsRollupBatchSize int

	// JS plugins directory (optional). Set via PLUGINS_DIR env var.
	PluginsDir string

	// Export defaults (org settings seed)
	ExportDefaultTargetType string
	ExportS3UploadBaseURL   string
	ExportS3AccessKey       string
	ExportS3SecretKey       string
	ExportS3Region          string
	ExportS3AuthToken       string
	ExportS3PathPrefix      string
}

func Load() (Config, error) {
	c := Config{
		Port:                     getenv("PORT", "8080"),
		Environment:              getenv("ENV", "dev"),
		PostgresDSN:              getenv("PG_DSN", ""),
		DataStoreSecretKey:       getenv("DATA_STORE_SECRET_KEY", ""),
		DataStoreSecretKeyFile:   getenv("DATA_STORE_SECRET_KEY_FILE", ""),
		DefaultTenant:            getenv("DEFAULT_TENANT", "default"),
		MaxPageLimit:             getenvInt("MAX_PAGE_LIMIT", 100),
		DefaultOrg:               getenv("DEFAULT_ORG", "default"),
		JWTSecret:                getenv("JWT_SECRET", ""),
		SMTPHost:                 getenv("SMTP_HOST", ""),
		SMTPPort:                 getenvInt("SMTP_PORT", 587),
		SMTPUser:                 getenv("SMTP_USER", ""),
		SMTPPassword:             getenv("SMTP_PASSWORD", ""),
		SMTPFrom:                 getenv("SMTP_FROM", "noreply@localhost"),
		AppURL:                   getenv("APP_URL", "http://localhost:5173"),
		LocalLoginEnabled:        getenvBool("LOCAL_LOGIN_ENABLED", true),
		LocalLoginExplicit:       os.Getenv("LOCAL_LOGIN_ENABLED") != "",
		OIDCEnabled:              getenvBool("OIDC_ENABLED", false),
		OIDCSlug:                 getenv("OIDC_SLUG", "oidc"),
		OIDCDisplayName:          getenv("OIDC_DISPLAY_NAME", "OpenID Connect"),
		OIDCIssuer:               getenv("OIDC_ISSUER", ""),
		OIDCClientID:             getenv("OIDC_CLIENT_ID", ""),
		OIDCClientSecret:         getenv("OIDC_CLIENT_SECRET", ""),
		OIDCScopes:               getenv("OIDC_SCOPES", "openid profile email"),
		OIDCAutoProvision:        getenvBool("OIDC_AUTO_PROVISION", false),
		OIDCDefaultRole:          getenv("OIDC_DEFAULT_ROLE", "viewer"),
		OIDCRequireEmailVerified: getenvBool("OIDC_REQUIRE_EMAIL_VERIFIED", false),
		SAMLEnabled:              getenvBool("SAML_ENABLED", false),
		SAMLSlug:                 getenv("SAML_SLUG", "shibboleth"),
		SAMLDisplayName:          getenv("SAML_DISPLAY_NAME", "Shibboleth"),
		SAMLIDPMetadataURL:       getenv("SAML_IDP_METADATA_URL", ""),
		SAMLIDPMetadataXML:       getenv("SAML_IDP_METADATA_XML", ""),
		SAMLSPEntityID:           getenv("SAML_SP_ENTITY_ID", ""),
		SAMLAutoProvision:        getenvBool("SAML_AUTO_PROVISION", false),
		SAMLDefaultRole:          getenv("SAML_DEFAULT_ROLE", "viewer"),
		StatsRollupEnabled:       getenvBool("STATS_ROLLUP_ENABLED", true),
		StatsRollupIntervalS:     getenvInt("STATS_ROLLUP_INTERVAL_S", 15),
		StatsRollupBatchSize:     getenvInt("STATS_ROLLUP_BATCH_SIZE", 5000),
		PluginsDir:               getenv("PLUGINS_DIR", ""),
		ExportDefaultTargetType:  strings.ToLower(getenv("EXPORT_DEFAULT_TARGET_TYPE", "s3")),
		ExportS3UploadBaseURL:    getenv("EXPORT_S3_UPLOAD_BASE_URL", ""),
		ExportS3AccessKey:        getenv("EXPORT_S3_ACCESS_KEY", ""),
		ExportS3SecretKey:        getenv("EXPORT_S3_SECRET_KEY", ""),
		ExportS3Region:           getenv("EXPORT_S3_REGION", "us-east-1"),
		ExportS3AuthToken:        getenv("EXPORT_S3_AUTH_TOKEN", ""),
		ExportS3PathPrefix:       getenv("EXPORT_S3_PATH_PREFIX", "exports"),
	}

	if c.PostgresDSN == "" {
		return Config{}, fmt.Errorf("PG_DSN is required")
	}
	if c.DataStoreSecretKey == "" && c.DataStoreSecretKeyFile != "" {
		b, err := os.ReadFile(c.DataStoreSecretKeyFile)
		if err != nil {
			return Config{}, fmt.Errorf("read DATA_STORE_SECRET_KEY_FILE: %w", err)
		}
		c.DataStoreSecretKey = strings.TrimSpace(string(b))
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
	case "local", "s3":
	default:
		c.ExportDefaultTargetType = "s3"
	}
	if strings.TrimSpace(c.ExportS3PathPrefix) == "" {
		c.ExportS3PathPrefix = "exports"
	}
	switch c.OIDCDefaultRole {
	case "owner", "admin", "member", "viewer":
	default:
		c.OIDCDefaultRole = "viewer"
	}
	switch c.SAMLDefaultRole {
	case "owner", "admin", "member", "viewer":
	default:
		c.SAMLDefaultRole = "viewer"
	}
	if pem, err := readPEMFile(getenv("SAML_SP_CERT_FILE", "")); err != nil {
		return Config{}, err
	} else {
		c.SAMLSPCertPEM = pem
	}
	if pem, err := readPEMFile(getenv("SAML_SP_KEY_FILE", "")); err != nil {
		return Config{}, err
	} else {
		c.SAMLSPKeyPEM = pem
	}
	return c, nil
}

func readPEMFile(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	return string(b), nil
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

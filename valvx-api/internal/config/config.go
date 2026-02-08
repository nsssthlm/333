// Package config loads environment variables into a typed configuration struct.
// Follows the existing VALVX_API_* naming convention.
package config

import (
	"os"
	"strconv"
)

// Config holds all API configuration from environment variables.
type Config struct {
	// Server
	BindHost           string
	BindPort           int
	CORSAllowedOrigins string

	// Session
	SessionCookieDomain   string
	SessionCookieSecure   bool
	SessionCookieSameSite string

	// PostgreSQL
	PostgresURL string

	// Blob storage (MinIO/S3)
	BlobstorURL       string
	BlobstorServer    string
	BlobstorBucket    string
	AWSAccessKeyID    string
	AWSSecretAccessKey string

	// Speckle integration
	SpeckleURL         string
	SpeckleInternalURL string
	SpeckleProjectID   string
	SpeckleAPIToken    string
	SpeckleProxyEnabled bool

	// TUS upload
	TUSEnabled  bool
	TUSMaxSize  int64
	TUSChunkSize int64

	// Security
	PasswordPepper string

	// Mailgun
	MailgunAPIKey string

	// Paths
	MigrationsDir string
}

// Load reads all environment variables and returns a Config.
func Load() *Config {
	c := &Config{
		BindHost:           env("VALVX_API_SERVER_BIND_HOST", "127.0.0.1"),
		BindPort:           envInt("VALVX_API_SERVER_BIND_PORT", 4000),
		CORSAllowedOrigins: env("VALVX_API_SERVER_CORS_ALLOWED_ORIGINS", "https://app.valvx.se"),

		SessionCookieDomain:   env("VALVX_API_SERVER_SESSION_COOKIE_DOMAIN", "valvx.se"),
		SessionCookieSecure:   envBool("VALVX_API_SERVER_SESSION_COOKIE_SECURE", true),
		SessionCookieSameSite: env("VALVX_API_SERVER_SESSION_COOKIE_SAME_SITE", "Default"),

		PostgresURL: buildPostgresURL(),

		BlobstorURL:        env("VALVX_API_BLOBSTOR_URL", "s3://?s3ForcePathStyle=true"),
		BlobstorServer:     env("VALVX_API_BLOBSTOR_SERVER", "https://storage.valvx.se"),
		BlobstorBucket:     env("VALVX_API_BLOBSTOR_BUCKET", "valvx"),
		AWSAccessKeyID:     env("AWS_ACCESS_KEY_ID", ""),
		AWSSecretAccessKey: env("AWS_SECRET_ACCESS_KEY", ""),

		SpeckleURL:          env("VALVX_API_SPECKLE_URL", "https://speckle.valvx.se"),
		SpeckleInternalURL:  env("VALVX_API_SPECKLE_INTERNAL_URL", "http://127.0.0.1:8080"),
		SpeckleProjectID:    env("VALVX_API_SPECKLE_PROJECT_ID", ""),
		SpeckleAPIToken:     env("VALVX_API_SPECKLE_API_TOKEN", ""),
		SpeckleProxyEnabled: envBool("VALVX_API_SPECKLE_PROXY_ENABLED", false),

		TUSEnabled:   envBool("VALVX_API_TUS_ENABLED", true),
		TUSMaxSize:   envInt64("VALVX_API_TUS_MAX_SIZE", 5*1024*1024*1024),    // 5 GB
		TUSChunkSize: envInt64("VALVX_API_TUS_CHUNK_SIZE", 5*1024*1024),         // 5 MB

		PasswordPepper: env("VALVX_API_PASSWORD_PEPPER", ""),
		MailgunAPIKey:  env("VALVX_API_MAILGUN_API_KEY", ""),

		MigrationsDir: env("VALVX_API_MIGRATIONS_DIR", "/app/migrations"),
	}

	return c
}

func buildPostgresURL() string {
	explicit := os.Getenv("VALVX_API_POSTGRES_URL")
	if explicit != "" && explicit != "postgres:///?sslmode=disable" {
		return explicit
	}

	host := env("PGHOST", "127.0.0.1")
	port := env("PGPORT", "5432")
	user := env("PGUSER", "valvx")
	pass := env("PGPASSWORD", "")
	dbname := env("PGDATABASE", "valvx")

	return "postgres://" + user + ":" + pass + "@" + host + ":" + port + "/" + dbname + "?sslmode=disable"
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func envInt64(key string, fallback int64) int64 {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i
		}
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

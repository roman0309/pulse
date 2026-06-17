package config

import (
	"os"
	"strings"
	"time"
)

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	Port             string
	PostgresDSN      string
	ClickHouseDSN    string
	JWTSecret        string
	JWTRefreshSecret string
	AccessTTL        time.Duration
	RefreshTTL       time.Duration
	CORSOrigins      []string
	// PublicIngestURL is the externally reachable base URL that remote agents
	// use to push telemetry (e.g. https://pulse.example.com). Surfaced to the
	// Connect page so generated commands target the right host. When empty, the
	// frontend falls back to deriving it from the browser location.
	PublicIngestURL string
	// AgentImage is the published Docker image for the host agent, shown in the
	// Connect page's run command (e.g. ghcr.io/acme/pulse-agent:latest).
	AgentImage string
	// SeedDemo loads demo data (user, project, fake metrics) on first migration.
	SeedDemo bool
}

// Load reads configuration from the environment, applying sensible defaults
// so the service runs out-of-the-box in local development.
func Load() *Config {
	return &Config{
		Port:             getEnv("PORT", "8080"),
		PostgresDSN:      getEnv("POSTGRES_DSN", "postgres://metrics:metrics_secret@localhost:5432/metrics_db?sslmode=disable"),
		ClickHouseDSN:    getEnv("CLICKHOUSE_DSN", "clickhouse://metrics:metrics_secret@localhost:9000/metrics_db"),
		JWTSecret:        getEnv("JWT_SECRET", "super_secret_jwt_key_change_in_production"),
		JWTRefreshSecret: getEnv("JWT_REFRESH_SECRET", "super_secret_refresh_key_change_in_production"),
		AccessTTL:        15 * time.Minute,
		RefreshTTL:       7 * 24 * time.Hour,
		CORSOrigins:      strings.Split(getEnv("CORS_ORIGINS", "http://localhost:3000,http://localhost:5173"), ","),
		PublicIngestURL:  getEnv("PUBLIC_INGEST_URL", ""),
		AgentImage:       getEnv("AGENT_IMAGE", "ghcr.io/roman0309/pulse-agent:latest"),
		SeedDemo:         getEnv("SEED_DEMO", "false") == "true",
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

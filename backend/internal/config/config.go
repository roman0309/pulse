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
	// CredentialsKey encrypts stored SSH credentials and channel secrets at rest.
	// When empty, the server generates and persists a random key (see main).
	CredentialsKey string

	// Self-update: when the Docker socket is mounted, Pulse can pull new images
	// and recreate its own containers via a one-shot watchtower run.
	DockerSocket         string
	WatchtowerImage      string
	SelfUpdateContainers []string
}

// LegacyCredentialsKey is the historical hard-coded key. It is kept ONLY as a
// decryption fallback so secrets written before key hardening still open; new
// data is always encrypted with the resolved (env or generated) key.
const LegacyCredentialsKey = "super_secret_jwt_key_change_in_production"

// Load reads configuration from the environment, applying sensible defaults
// so the service runs out-of-the-box in local development.
func Load() *Config {
	return &Config{
		Port:             getEnv("PORT", "8080"),
		PostgresDSN:      getEnv("POSTGRES_DSN", "postgres://metrics:metrics_secret@localhost:5432/metrics_db?sslmode=disable"),
		ClickHouseDSN:    getEnv("CLICKHOUSE_DSN", "clickhouse://metrics:metrics_secret@localhost:9000/metrics_db"),
		// Secrets default to empty; main resolves them to an env value or a
		// random key persisted in the database (never a shipped constant).
		JWTSecret:        getEnv("JWT_SECRET", ""),
		JWTRefreshSecret: getEnv("JWT_REFRESH_SECRET", ""),
		AccessTTL:        15 * time.Minute,
		RefreshTTL:       7 * 24 * time.Hour,
		CORSOrigins:      strings.Split(getEnv("CORS_ORIGINS", "http://localhost:3000,http://localhost:5173"), ","),
		PublicIngestURL:  getEnv("PUBLIC_INGEST_URL", ""),
		AgentImage:       getEnv("AGENT_IMAGE", "ghcr.io/roman0309/pulse-agent:latest"),
		SeedDemo:         getEnv("SEED_DEMO", "false") == "true",
		CredentialsKey:   getEnv("CREDENTIALS_KEY", ""),

		DockerSocket:         getEnv("DOCKER_SOCKET", "/var/run/docker.sock"),
		WatchtowerImage:      getEnv("WATCHTOWER_IMAGE", "containrrr/watchtower:latest"),
		SelfUpdateContainers: strings.Split(getEnv("SELF_UPDATE_CONTAINERS", "pulse-backend,pulse-frontend"), ","),
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

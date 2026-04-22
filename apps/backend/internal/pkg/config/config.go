package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// HTTPConfig contains HTTP server settings.
type HTTPConfig struct {
	Address           string
	AllowedOrigins    []string
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	ShutdownTimeout   time.Duration
}

// AuthConfig contains authentication settings.
type AuthConfig struct {
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	JWTSigningKey   string
}

// PostgresConfig contains PostgreSQL connection settings.
type PostgresConfig struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// Config contains application configuration.
type Config struct {
	HTTP     HTTPConfig
	Auth     AuthConfig
	Postgres PostgresConfig
}

// Load returns configuration populated from environment variables with
// safe defaults for every optional field. JWT_SIGNING_KEY is required
// and must not equal the placeholder "change-me".
func Load() (Config, error) {
	jwtKey := getEnv("JWT_SIGNING_KEY", "change-me")
	if jwtKey == "change-me" {
		return Config{}, fmt.Errorf("JWT_SIGNING_KEY must be set to a non-default value")
	}

	maxOpenConns, err := getEnvInt("POSTGRES_MAX_OPEN_CONNS", 10)
	if err != nil {
		return Config{}, fmt.Errorf("invalid POSTGRES_MAX_OPEN_CONNS: %w", err)
	}

	maxIdleConns, err := getEnvInt("POSTGRES_MAX_IDLE_CONNS", 5)
	if err != nil {
		return Config{}, fmt.Errorf("invalid POSTGRES_MAX_IDLE_CONNS: %w", err)
	}

	return Config{
		HTTP: HTTPConfig{
			Address:           getEnv("HTTP_ADDRESS", ":8080"),
			AllowedOrigins:    parseOrigins(os.Getenv("ALLOWED_ORIGINS")),
			ReadHeaderTimeout: 5 * time.Second,
			WriteTimeout:      30 * time.Second,
			ShutdownTimeout:   10 * time.Second,
		},
		Auth: AuthConfig{
			AccessTokenTTL:  15 * time.Minute,
			RefreshTokenTTL: 30 * 24 * time.Hour,
			JWTSigningKey:   jwtKey,
		},
		Postgres: PostgresConfig{
			DSN:             getEnv("POSTGRES_DSN", "postgres://postgres:postgres@localhost:5432/office_space_allocation?sslmode=disable"),
			MaxOpenConns:    maxOpenConns,
			MaxIdleConns:    maxIdleConns,
			ConnMaxLifetime: 30 * time.Minute,
		},
	}, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func getEnvInt(key string, fallback int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}

	return parsed, nil
}

func parseOrigins(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			origins = append(origins, trimmed)
		}
	}

	return origins
}

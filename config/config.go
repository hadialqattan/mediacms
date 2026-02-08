package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port             string
	Database         DatabaseConfig
	Redis            RedisConfig
	Outbox           OutboxConfig
	TypesenseAddress string
	TypesenseAPIKey  string
	JWT              JWTConfig
	DefaultAdmin     DefaultAdminConfig
}

type DatabaseConfig struct {
	URL            string
	MaxConnections int
}

type RedisConfig struct {
	Addr            string
	MaxRetries      int
	MinRetryBackoff time.Duration
	MaxRetryBackoff time.Duration
}

type OutboxConfig struct {
	RelayInterval time.Duration
}

type JWTConfig struct {
	Secret          string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

type DefaultAdminConfig struct {
	Email    string
	Password string
}

func LoadCMS() *Config {
	return load("8080")
}

func LoadDiscovery() *Config {
	return load("8081")
}

func LoadOutbox() *Config {
	return load("8082")
}

func LoadSearchIndexer() *Config {
	return load("8083")
}

func load(defaultPort string) *Config {
	return &Config{
		Port: getEnv("PORT", defaultPort),
		Database: DatabaseConfig{
			URL:            getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/mediacms?sslmode=disable"),
			MaxConnections: getIntEnv("DB_MAX_CONNECTIONS", 25),
		},
		Redis: RedisConfig{
			Addr:            getEnv("REDIS_ADDR", "localhost:6379"),
			MaxRetries:      getIntEnv("REDIS_MAX_RETRIES", 3),
			MinRetryBackoff: getDurationEnv("REDIS_MIN_RETRY_BACKOFF", 500*time.Millisecond),
			MaxRetryBackoff: getDurationEnv("REDIS_MAX_RETRY_BACKOFF", time.Second),
		},
		Outbox: OutboxConfig{
			RelayInterval: getDurationEnv("OUTBOX_RELAY_INTERVAL", 5*time.Second),
		},
		TypesenseAddress: getEnv("TYPESENSE_ADDRESS", "http://localhost:8108"),
		TypesenseAPIKey:  getEnv("TYPESENSE_API_KEY", "xyz"),
		JWT: JWTConfig{
			Secret:          getEnv("JWT_SECRET", "secret-key-change-in-production"),
			AccessTokenTTL:  getDurationEnv("JWT_ACCESS_TOKEN_TTL", 15*time.Minute),
			RefreshTokenTTL: getDurationEnv("JWT_REFRESH_TOKEN_TTL", 720*time.Hour),
		},
		DefaultAdmin: DefaultAdminConfig{
			Email:    getEnv("DEFAULT_ADMIN_EMAIL", "admin@mediacms.local"),
			Password: getEnv("DEFAULT_ADMIN_PASSWORD", "changeme"),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}

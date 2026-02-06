package config

import (
	"os"
	"time"
)

type Config struct {
	ServiceName      string
	Port             string
	DatabaseURL      string
	RedisAddr        string
	TypesenseAddress string
	TypesenseAPIKey  string
	JWT              JWTConfig
}

type JWTConfig struct {
	Secret          string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

func Load() *Config {
	return &Config{
		ServiceName:      getEnv("SERVICE_NAME", "thmanyah-cms"),
		Port:             getEnv("PORT", "8080"),
		DatabaseURL:      getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/thmanyah?sslmode=disable"),
		RedisAddr:        getEnv("REDIS_ADDR", "localhost:6379"),
		TypesenseAddress: getEnv("TYPESENSE_ADDRESS", "http://localhost:8108"),
		TypesenseAPIKey:  getEnv("TYPESENSE_API_KEY", "xyz"),
		JWT: JWTConfig{
			Secret:          getEnv("JWT_SECRET", "secret-key-change-in-production"),
			AccessTokenTTL:  getDurationEnv("JWT_ACCESS_TOKEN_TTL", 15*time.Minute),
			RefreshTokenTTL: getDurationEnv("JWT_REFRESH_TOKEN_TTL", 720*time.Hour),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
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

func (c *Config) GetTypesenseClientConfig() (addr, apiKey string) {
	return c.TypesenseAddress, c.TypesenseAPIKey
}

func (c *Config) GetRedisClientConfig() (addr string, retryInterval time.Duration) {
	return c.RedisAddr, 5 * time.Second
}

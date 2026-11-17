// Package config loads orders-api settings from environment variables.
package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all runtime settings.
type Config struct {
	Port              string
	AppVersion        string
	PostgresURL       string
	RedisAddr         string
	RedisPassword     string
	RedisDB           int
	CORSOrigins       []string
	ReconcileInterval time.Duration
	ReconcileAfter    time.Duration
	ShutdownTimeout   time.Duration
}

// Load reads configuration from the environment.
func Load() (Config, error) {
	redisDB, err := intEnv("REDIS_DB", 0)
	if err != nil {
		return Config{}, err
	}
	pgPort, err := intEnv("POSTGRES_PORT", 5432)
	if err != nil {
		return Config{}, err
	}
	reconcileEvery, err := durationEnv("RECONCILE_INTERVAL", 15*time.Second)
	if err != nil {
		return Config{}, err
	}
	reconcileAfter, err := durationEnv("RECONCILE_AFTER", 10*time.Second)
	if err != nil {
		return Config{}, err
	}

	pg := &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(env("POSTGRES_USER", "dhaba"), env("POSTGRES_PASSWORD", "")),
		Host:     fmt.Sprintf("%s:%d", env("POSTGRES_HOST", "localhost"), pgPort),
		Path:     "/" + env("POSTGRES_DB", "dhaba"),
		RawQuery: "sslmode=" + env("POSTGRES_SSLMODE", "disable") + "&application_name=orders-api",
	}

	var origins []string
	for _, o := range strings.Split(env("CORS_ALLOWED_ORIGINS", ""), ",") {
		if o = strings.TrimSpace(o); o != "" {
			origins = append(origins, o)
		}
	}

	return Config{
		Port:              env("PORT", "8081"),
		AppVersion:        env("APP_VERSION", "1.0.0"),
		PostgresURL:       pg.String(),
		RedisAddr:         fmt.Sprintf("%s:%s", env("REDIS_HOST", "localhost"), env("REDIS_PORT", "6379")),
		RedisPassword:     env("REDIS_PASSWORD", ""),
		RedisDB:           redisDB,
		CORSOrigins:       origins,
		ReconcileInterval: reconcileEvery,
		ReconcileAfter:    reconcileAfter,
		ShutdownTimeout:   15 * time.Second,
	}, nil
}

func env(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func intEnv(key string, fallback int) (int, error) {
	v := env(key, "")
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}
	return n, nil
}

func durationEnv(key string, fallback time.Duration) (time.Duration, error) {
	v := env(key, "")
	if v == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s must be a duration like 15s: %w", key, err)
	}
	return d, nil
}

package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadDefaultsAndOverrides(t *testing.T) {
	t.Setenv("POSTGRES_HOST", "db.internal")
	t.Setenv("POSTGRES_PASSWORD", "p@ss word")
	t.Setenv("REDIS_HOST", "cache")
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://a.test, ,http://b.test")
	t.Setenv("RECONCILE_INTERVAL", "30s")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != "8081" || cfg.RedisAddr != "cache:6379" || cfg.ReconcileInterval != 30*time.Second {
		t.Errorf("unexpected config %+v", cfg)
	}
	if !strings.Contains(cfg.PostgresURL, "db.internal:5432/dhaba") || !strings.Contains(cfg.PostgresURL, "p%40ss%20word") {
		t.Errorf("postgres url not built/escaped: %s", cfg.PostgresURL)
	}
	if len(cfg.CORSOrigins) != 2 {
		t.Errorf("origins %v", cfg.CORSOrigins)
	}
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	t.Setenv("REDIS_DB", "zero")
	if _, err := Load(); err == nil {
		t.Error("expected error for REDIS_DB")
	}
	t.Setenv("REDIS_DB", "0")
	t.Setenv("RECONCILE_AFTER", "soon")
	if _, err := Load(); err == nil {
		t.Error("expected error for RECONCILE_AFTER")
	}
}

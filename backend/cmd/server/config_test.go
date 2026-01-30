package main

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfigFromEnvDefaults(t *testing.T) {
	clearConfigEnv(t)
	cfg, err := loadConfigFromEnv()
	if err != nil {
		t.Fatalf("loadConfigFromEnv: %v", err)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Fatalf("expected default addr, got %s", cfg.HTTPAddr)
	}
	if cfg.WSPath != "/ws" {
		t.Fatalf("expected default path, got %s", cfg.WSPath)
	}
	if cfg.Grace != 60*time.Second {
		t.Fatalf("expected default grace, got %s", cfg.Grace)
	}
	if cfg.TurnTimeout != 20*time.Second {
		t.Fatalf("expected default turn timeout, got %s", cfg.TurnTimeout)
	}
	if cfg.PostgresDSN != "" {
		t.Fatalf("expected empty postgres dsn by default")
	}
	if cfg.ReplayAutoMigrate {
		t.Fatalf("expected replay automigrate disabled by default")
	}
}

func TestLoadConfigFromEnvOverrides(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv(envHTTPAddr, ":9090")
	t.Setenv(envWSPath, "/socket")
	t.Setenv(envGrace, "5s")
	t.Setenv(envChallengeTimeout, "7s")
	t.Setenv(envCounterTimeout, "8s")
	t.Setenv(envTurnTimeout, "9s")
	t.Setenv(envPostgresDSN, "postgres://user:pass@localhost:5432/elmakina")
	t.Setenv(envReplayAutoMigrate, "true")

	cfg, err := loadConfigFromEnv()
	if err != nil {
		t.Fatalf("loadConfigFromEnv: %v", err)
	}
	if cfg.HTTPAddr != ":9090" {
		t.Fatalf("expected addr override")
	}
	if cfg.WSPath != "/socket" {
		t.Fatalf("expected path override")
	}
	if cfg.Grace != 5*time.Second {
		t.Fatalf("expected grace override")
	}
	if cfg.ChallengeTimeout != 7*time.Second {
		t.Fatalf("expected challenge timeout override")
	}
	if cfg.CounterTimeout != 8*time.Second {
		t.Fatalf("expected counter timeout override")
	}
	if cfg.TurnTimeout != 9*time.Second {
		t.Fatalf("expected turn timeout override")
	}
	if cfg.PostgresDSN == "" {
		t.Fatalf("expected postgres dsn override")
	}
	if !cfg.ReplayAutoMigrate {
		t.Fatalf("expected replay automigrate override")
	}
}

func TestLoadConfigFromEnvInvalid(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv(envGrace, "bad")
	_, err := loadConfigFromEnv()
	if err == nil {
		t.Fatalf("expected error for invalid grace")
	}
}

func clearConfigEnv(t *testing.T) {
	t.Helper()
	_ = os.Unsetenv(envHTTPAddr)
	_ = os.Unsetenv(envWSPath)
	_ = os.Unsetenv(envGrace)
	_ = os.Unsetenv(envChallengeTimeout)
	_ = os.Unsetenv(envCounterTimeout)
	_ = os.Unsetenv(envTurnTimeout)
	_ = os.Unsetenv(envPostgresDSN)
	_ = os.Unsetenv(envReplayAutoMigrate)
}

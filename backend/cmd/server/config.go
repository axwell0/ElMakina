package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"ElMakina/backend/engine"
	"ElMakina/backend/server/replay"
	"ElMakina/backend/server/ws"
)

const (
	envHTTPAddr          = "ELMAKINA_HTTP_ADDR"
	envWSPath            = "ELMAKINA_WS_PATH"
	envGrace             = "ELMAKINA_GRACE"
	envChallengeTimeout  = "ELMAKINA_CHALLENGE_TIMEOUT"
	envCounterTimeout    = "ELMAKINA_COUNTER_TIMEOUT"
	envTurnTimeout       = "ELMAKINA_TURN_TIMEOUT"
	envPostgresDSN       = "ELMAKINA_POSTGRES_DSN"
	envReplayAutoMigrate = "ELMAKINA_REPLAY_AUTOMIGRATE"
)

type config struct {
	HTTPAddr          string
	WSPath            string
	Grace             time.Duration
	ChallengeTimeout  time.Duration
	CounterTimeout    time.Duration
	TurnTimeout       time.Duration
	PostgresDSN       string
	ReplayAutoMigrate bool
}

// loadConfigFromEnv builds server configuration from environment variables.
// Defaults are tuned for local play and can be overridden per deployment.
func loadConfigFromEnv() (config, error) {
	cfg := config{
		HTTPAddr:          ":8080",
		WSPath:            "/ws",
		Grace:             60 * time.Second,
		ChallengeTimeout:  15 * time.Second,
		CounterTimeout:    15 * time.Second,
		TurnTimeout:       20 * time.Second,
		ReplayAutoMigrate: false,
	}

	if val := os.Getenv(envHTTPAddr); val != "" {
		cfg.HTTPAddr = val
	}
	if val := os.Getenv(envWSPath); val != "" {
		cfg.WSPath = val
	}
	if val := os.Getenv(envGrace); val != "" {
		parsed, err := time.ParseDuration(val)
		if err != nil {
			return cfg, fmt.Errorf("%s: %w", envGrace, err)
		}
		cfg.Grace = parsed
	}
	if val := os.Getenv(envChallengeTimeout); val != "" {
		parsed, err := time.ParseDuration(val)
		if err != nil {
			return cfg, fmt.Errorf("%s: %w", envChallengeTimeout, err)
		}
		cfg.ChallengeTimeout = parsed
	}
	if val := os.Getenv(envCounterTimeout); val != "" {
		parsed, err := time.ParseDuration(val)
		if err != nil {
			return cfg, fmt.Errorf("%s: %w", envCounterTimeout, err)
		}
		cfg.CounterTimeout = parsed
	}
	if val := os.Getenv(envTurnTimeout); val != "" {
		parsed, err := time.ParseDuration(val)
		if err != nil {
			return cfg, fmt.Errorf("%s: %w", envTurnTimeout, err)
		}
		cfg.TurnTimeout = parsed
	}
	if val := os.Getenv(envPostgresDSN); val != "" {
		cfg.PostgresDSN = val
	}
	if val := os.Getenv(envReplayAutoMigrate); val != "" {
		cfg.ReplayAutoMigrate = val == "true" || val == "1" || val == "yes"
	}
	return cfg, nil
}

// loadDotEnv reads key=value pairs into the process environment.
func loadDotEnv(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("invalid .env line %d", i+1)
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		val = strings.Trim(val, `"'`)
		if key == "" {
			return fmt.Errorf("invalid .env line %d", i+1)
		}
		if err := os.Setenv(key, val); err != nil {
			return err
		}
	}
	return nil
}

// buildTurnConfig converts the env config into engine timeouts.
func buildTurnConfig(cfg config) engine.TurnConfig {
	return engine.TurnConfig{
		ChallengeTimeout: cfg.ChallengeTimeout,
		CounterTimeout:   cfg.CounterTimeout,
		TurnTimeout:      cfg.TurnTimeout,
	}
}

// buildLobbyServer wires persistence and orchestration into a WS lobby server.
func buildLobbyServer(cfg config) (*ws.Server, replay.Recorder, error) {
	manager, err := ws.NewLobbyManagerFromEnv(2, 9)
	if err != nil {
		return nil, nil, err
	}
	if err := manager.ResetInGameLobbiesToOpen(); err != nil {
		return nil, nil, err
	}
	// Drop any open lobbies restored from disk that have no online members.
	if _, err := manager.PruneEmptyOpenLobbies(map[string]struct{}{}); err != nil {
		return nil, nil, err
	}
	server := ws.NewServer(manager, engine.RealClock{}, buildTurnConfig(cfg), cfg.Grace)
	var recorder replay.Recorder
	if cfg.PostgresDSN == "" {
		return nil, nil, fmt.Errorf("%s is required", envPostgresDSN)
	}
	db, err := replay.OpenPostgres(context.Background(), cfg.PostgresDSN)
	if err != nil {
		return nil, nil, err
	}
	store := replay.NewStore(db)
	if cfg.ReplayAutoMigrate {
		if err := store.AutoMigrate(context.Background()); err != nil {
			return nil, nil, err
		}
	}
	recorder = replay.NewAsyncRecorder(store, replay.AsyncOptions{})
	server.SetRecorder(recorder)
	return server, recorder, nil
}

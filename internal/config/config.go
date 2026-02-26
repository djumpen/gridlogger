package config

import (
	"errors"
	"os"
	"strconv"
	"time"
)

type Config struct {
	ListenAddr       string
	DatabaseURL      string
	OutageThreshold  time.Duration
	DefaultProjectID int
}

func Load() (Config, error) {
	cfg := Config{
		ListenAddr:       getenv("LISTEN_ADDR", ":8080"),
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		OutageThreshold:  2 * time.Minute,
		DefaultProjectID: 1,
	}

	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}

	if raw := os.Getenv("OUTAGE_THRESHOLD_SECONDS"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v <= 0 {
			return Config{}, errors.New("OUTAGE_THRESHOLD_SECONDS must be a positive integer")
		}
		cfg.OutageThreshold = time.Duration(v) * time.Second
	}

	if raw := os.Getenv("DEFAULT_PROJECT_ID"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v <= 0 {
			return Config{}, errors.New("DEFAULT_PROJECT_ID must be a positive integer")
		}
		cfg.DefaultProjectID = v
	}

	return cfg, nil
}

func getenv(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}

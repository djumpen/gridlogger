package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	ListenAddr       string
	DatabaseURL      string
	OutageThreshold  time.Duration
	DefaultProjectID int

	TelegramBotToken    string
	TelegramBotUsername string
	TelegramAuthTTL     time.Duration
	JWTSecret           string
	JWTIssuer           string
	SessionTTL          time.Duration
	SessionCookieName   string
	SessionCookieSecure bool

	TestEnv string
}

func Load() (Config, error) {
	cfg := Config{
		ListenAddr:        getenv("LISTEN_ADDR", ":8080"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		OutageThreshold:   2 * time.Minute,
		DefaultProjectID:  1,
		TelegramAuthTTL:   24 * time.Hour,
		JWTIssuer:         getenv("JWT_ISSUER", "gridlogger"),
		SessionTTL:        7 * 24 * time.Hour,
		SessionCookieName: getenv("SESSION_COOKIE_NAME", "gridlogger_session"),

		TestEnv: getenv("TEST_ENV", "Not set"),
	}

	cfg.TelegramBotToken = os.Getenv("TELEGRAM_BOT_TOKEN")
	cfg.TelegramBotUsername = os.Getenv("TELEGRAM_BOT_USERNAME")
	cfg.JWTSecret = os.Getenv("JWT_SECRET")

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

	if raw := os.Getenv("TELEGRAM_AUTH_TTL_SECONDS"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v <= 0 {
			return Config{}, errors.New("TELEGRAM_AUTH_TTL_SECONDS must be a positive integer")
		}
		cfg.TelegramAuthTTL = time.Duration(v) * time.Second
	}

	if raw := os.Getenv("SESSION_TTL_SECONDS"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v <= 0 {
			return Config{}, errors.New("SESSION_TTL_SECONDS must be a positive integer")
		}
		cfg.SessionTTL = time.Duration(v) * time.Second
	}

	if raw := os.Getenv("SESSION_COOKIE_SECURE"); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return Config{}, errors.New("SESSION_COOKIE_SECURE must be true/false")
		}
		cfg.SessionCookieSecure = v
	}

	authFieldCount := 0
	if cfg.TelegramBotToken != "" {
		authFieldCount++
	}
	if cfg.TelegramBotUsername != "" {
		authFieldCount++
	}
	if cfg.JWTSecret != "" {
		authFieldCount++
	}

	if authFieldCount != 0 && authFieldCount != 3 {
		return Config{}, errors.New("TELEGRAM_BOT_TOKEN, TELEGRAM_BOT_USERNAME, and JWT_SECRET must be set together")
	}
	if cfg.TelegramAuthEnabled() && len(cfg.JWTSecret) < 32 {
		return Config{}, fmt.Errorf("JWT_SECRET must be at least 32 characters when Telegram auth is enabled")
	}

	return cfg, nil
}

func (c Config) TelegramAuthEnabled() bool {
	return c.TelegramBotToken != "" && c.TelegramBotUsername != "" && c.JWTSecret != ""
}

func getenv(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}

package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	ListenAddr       string
	DatabaseURL      string
	OutageThreshold  time.Duration
	DefaultProjectID int

	TelegramBotToken         string
	TelegramBotUsername      string
	TelegramAuthTTL          time.Duration
	JWTSecret                string
	JWTIssuer                string
	SessionTTL               time.Duration
	SessionCookieName        string
	SessionCookieSecure      bool
	NotificationsEnabled     bool
	NotificationPollInterval time.Duration
	FirmwareBuildEnabled     bool
	FirmwareServiceURL       string
	FirmwareServiceToken     string
	FirmwareServiceTimeout   time.Duration
	FirmwarePingBaseURL      string
	YasnoBaseURL             string
	YasnoTimeout             time.Duration

	TestEnv string
}

func Load() (Config, error) {
	cfg := Config{
		ListenAddr:               getenv("LISTEN_ADDR", ":8080"),
		DatabaseURL:              os.Getenv("DATABASE_URL"),
		OutageThreshold:          4 * time.Minute,
		DefaultProjectID:         1,
		TelegramAuthTTL:          24 * time.Hour,
		JWTIssuer:                getenv("JWT_ISSUER", "gridlogger"),
		SessionTTL:               7 * 24 * time.Hour,
		SessionCookieName:        getenv("SESSION_COOKIE_NAME", "gridlogger_session"),
		NotificationsEnabled:     true,
		NotificationPollInterval: 5 * time.Second,
		FirmwareBuildEnabled:     true,
		FirmwareServiceURL:       getenv("FIRMWARE_SERVICE_URL", "http://firmware:8081"),
		FirmwareServiceToken:     strings.TrimSpace(os.Getenv("FIRMWARE_SERVICE_TOKEN")),
		FirmwareServiceTimeout:   30 * time.Second,
		FirmwarePingBaseURL:      getenv("FIRMWARE_PING_BASE_URL", "https://svitlo.homes"),
		YasnoBaseURL:             getenv("YASNO_BASE_URL", "https://app.yasno.ua/api/blackout-service/public/shutdowns"),
		YasnoTimeout:             15 * time.Second,

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

	if raw := os.Getenv("NOTIFICATIONS_ENABLED"); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return Config{}, errors.New("NOTIFICATIONS_ENABLED must be true/false")
		}
		cfg.NotificationsEnabled = v
	}

	if raw := os.Getenv("NOTIFICATION_POLL_SECONDS"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v <= 0 {
			return Config{}, errors.New("NOTIFICATION_POLL_SECONDS must be a positive integer")
		}
		cfg.NotificationPollInterval = time.Duration(v) * time.Second
	}

	if raw := os.Getenv("FIRMWARE_BUILD_ENABLED"); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return Config{}, errors.New("FIRMWARE_BUILD_ENABLED must be true/false")
		}
		cfg.FirmwareBuildEnabled = v
	}

	if raw := os.Getenv("FIRMWARE_SERVICE_TIMEOUT_SECONDS"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v <= 0 {
			return Config{}, errors.New("FIRMWARE_SERVICE_TIMEOUT_SECONDS must be a positive integer")
		}
		cfg.FirmwareServiceTimeout = time.Duration(v) * time.Second
	}
	if raw := os.Getenv("YASNO_TIMEOUT_SECONDS"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v <= 0 {
			return Config{}, errors.New("YASNO_TIMEOUT_SECONDS must be a positive integer")
		}
		cfg.YasnoTimeout = time.Duration(v) * time.Second
	}
	if strings.TrimSpace(cfg.FirmwareServiceURL) == "" {
		return Config{}, errors.New("FIRMWARE_SERVICE_URL must not be empty")
	}
	if strings.TrimSpace(cfg.FirmwarePingBaseURL) == "" {
		return Config{}, errors.New("FIRMWARE_PING_BASE_URL must not be empty")
	}
	if strings.TrimSpace(cfg.YasnoBaseURL) == "" {
		return Config{}, errors.New("YASNO_BASE_URL must not be empty")
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

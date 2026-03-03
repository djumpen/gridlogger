package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/djumpen/gridlogger/internal/firmwareapi"
	"github.com/djumpen/gridlogger/internal/service"
)

type firmwareConfig struct {
	ListenAddr           string
	AuthToken            string
	FirmwareBuildEnabled bool
	ArduinoCLIPath       string
	BoardFQBN            string
	TemplateDir          string
	WorkDir              string
	BuildTimeout         time.Duration
	JobTTL               time.Duration
}

func main() {
	cfg, err := loadFirmwareConfig()
	if err != nil {
		log.Fatalf("load firmware config: %v", err)
	}

	buildService := service.NewFirmwareBuildService(
		service.NewArduinoFirmwareCompiler(cfg.ArduinoCLIPath, cfg.BoardFQBN, cfg.TemplateDir),
		cfg.FirmwareBuildEnabled,
		cfg.WorkDir,
		cfg.BuildTimeout,
		cfg.JobTTL,
	)
	handler := firmwareapi.NewHandler(buildService, cfg.AuthToken)

	srv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("firmware service listening on %s", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("firmware service listen: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("firmware service shutdown error: %v", err)
	}
}

func loadFirmwareConfig() (firmwareConfig, error) {
	cfg := firmwareConfig{
		ListenAddr:           getenv("LISTEN_ADDR", ":8081"),
		AuthToken:            strings.TrimSpace(os.Getenv("FIRMWARE_SERVICE_TOKEN")),
		FirmwareBuildEnabled: parseEnvBool("FIRMWARE_BUILD_ENABLED", true),
		ArduinoCLIPath:       getenv("FIRMWARE_ARDUINO_CLI_PATH", "arduino-cli"),
		BoardFQBN:            getenv("FIRMWARE_BOARD_FQBN", "esp32:esp32:esp32c3"),
		TemplateDir:          getenv("FIRMWARE_TEMPLATE_DIR", "/home/app/firmware/esp32-c3"),
		WorkDir:              getenv("FIRMWARE_WORK_DIR", "/tmp/gridlogger-firmware"),
		BuildTimeout:         parseEnvDurationSeconds("FIRMWARE_BUILD_TIMEOUT_SECONDS", 300),
		JobTTL:               parseEnvDurationSeconds("FIRMWARE_JOB_TTL_SECONDS", 7200),
	}

	if strings.TrimSpace(cfg.ArduinoCLIPath) == "" {
		return firmwareConfig{}, errors.New("FIRMWARE_ARDUINO_CLI_PATH must not be empty")
	}
	if strings.TrimSpace(cfg.BoardFQBN) == "" {
		return firmwareConfig{}, errors.New("FIRMWARE_BOARD_FQBN must not be empty")
	}
	if strings.TrimSpace(cfg.TemplateDir) == "" {
		return firmwareConfig{}, errors.New("FIRMWARE_TEMPLATE_DIR must not be empty")
	}
	if strings.TrimSpace(cfg.WorkDir) == "" {
		return firmwareConfig{}, errors.New("FIRMWARE_WORK_DIR must not be empty")
	}
	return cfg, nil
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func parseEnvBool(key string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return v
}

func parseEnvDurationSeconds(key string, fallback int) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return time.Duration(fallback) * time.Second
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return time.Duration(fallback) * time.Second
	}
	return time.Duration(v) * time.Second
}

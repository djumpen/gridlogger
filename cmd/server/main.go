package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/djumpen/gridlogger/internal/config"
	"github.com/djumpen/gridlogger/internal/db"
	"github.com/djumpen/gridlogger/internal/httpapi"
	"github.com/djumpen/gridlogger/internal/service"
	_ "github.com/newrelic/go-agent/v3/newrelic"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	log.Println("starting...")

	// app, err := newrelic.NewApplication(
	// 	newrelic.ConfigAppName("gridlogger"),
	// 	newrelic.ConfigLicense("eu01xx73d7b18295cb187b5491a4004aFFFFNRAL"),
	// 	newrelic.ConfigAppLogForwardingEnabled(true),
	// )

	// http.HandleFunc(newrelic.WrapHandleFunc(app, "/users", usersHandler))

	//txn := app.StartTransaction("transaction_name")
	//defer txn.End()

	appCtx, appCancel := context.WithCancel(context.Background())
	defer appCancel()

	pool, err := db.NewPool(appCtx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	if err := db.EnsureSchema(appCtx, pool); err != nil {
		log.Fatalf("ensure schema: %v", err)
	}

	repo := db.NewPingRepository(pool)
	svc := service.NewAvailabilityService(repo, cfg.OutageThreshold)
	projectRepo := db.NewProjectRepository(pool)
	projectCatalog := service.NewProjectCatalogService(projectRepo)
	dtekGroupRepo := db.NewDTEKGroupRepository(pool)
	projectNotificationRepo := db.NewProjectNotificationRepository(pool)
	telegramBot := service.NewTelegramBotService(cfg.TelegramBotToken)
	projectNotifications := service.NewProjectNotificationService(
		projectNotificationRepo,
		projectCatalog,
		repo,
		telegramBot,
		cfg.OutageThreshold,
	)
	firmwareClient := service.NewFirmwareBuildClient(
		cfg.FirmwareServiceURL,
		cfg.FirmwareServiceToken,
		cfg.FirmwareServiceTimeout,
	)
	firmwareBuilds := service.NewFirmwareGatewayService(
		cfg.FirmwareBuildEnabled,
		projectCatalog,
		firmwareClient,
		cfg.FirmwarePingBaseURL,
	)
	yasnoClient := service.NewYasnoClient(cfg.YasnoBaseURL, cfg.YasnoTimeout)
	yasnoSchedules := service.NewYasnoScheduleService(projectCatalog, dtekGroupRepo, yasnoClient)
	telegramRepo := db.NewTelegramAccountRepository(pool)
	telegramAuth := service.NewTelegramAuthService(telegramRepo, cfg.TelegramBotToken, cfg.TelegramAuthTTL)
	sessionAuth := service.NewSessionService(cfg.JWTSecret, cfg.JWTIssuer, cfg.SessionTTL)
	h := httpapi.NewHandler(
		svc,
		projectCatalog,
		projectNotifications,
		firmwareBuilds,
		yasnoSchedules,
		telegramAuth,
		sessionAuth,
		cfg.DefaultProjectID,
		cfg.TelegramBotUsername,
		cfg.SessionCookieName,
		cfg.SessionCookieSecure,
		cfg.SessionTTL,
	)

	if cfg.NotificationsEnabled && cfg.TelegramAuthEnabled() {
		go runProjectNotificationLoop(appCtx, projectNotifications, cfg.NotificationPollInterval)
	} else {
		log.Printf(
			"project notification loop disabled: enabled=%t telegram_auth_enabled=%t",
			cfg.NotificationsEnabled,
			cfg.TelegramAuthEnabled(),
		)
	}

	srv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      h,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("server listening on %s", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	appCancel()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}

func runProjectNotificationLoop(ctx context.Context, notifications *service.ProjectNotificationService, interval time.Duration) {
	if notifications == nil {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		if err := notifications.PollAndNotify(ctx); err != nil {
			log.Printf("project notification poll error: %v", err)
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

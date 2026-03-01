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

	ctx := context.Background()
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	if err := db.EnsureSchema(ctx, pool); err != nil {
		log.Fatalf("ensure schema: %v", err)
	}

	repo := db.NewPingRepository(pool)
	svc := service.NewAvailabilityService(repo, cfg.OutageThreshold)
	telegramRepo := db.NewTelegramAccountRepository(pool)
	telegramAuth := service.NewTelegramAuthService(telegramRepo, cfg.TelegramBotToken, cfg.TelegramAuthTTL)
	sessionAuth := service.NewSessionService(cfg.JWTSecret, cfg.JWTIssuer, cfg.SessionTTL)
	h := httpapi.NewHandler(
		svc,
		telegramAuth,
		sessionAuth,
		cfg.DefaultProjectID,
		cfg.TelegramBotUsername,
		cfg.SessionCookieName,
		cfg.SessionCookieSecure,
		cfg.SessionTTL,
	)

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

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"seruwit-wa/internal/api"
	"seruwit-wa/internal/config"
	"seruwit-wa/internal/database"
	"seruwit-wa/internal/webhook"
	"seruwit-wa/internal/whatsapp"

	waLog "go.mau.fi/whatsmeow/util/log"
)

func main() {
	cfg := config.Load()

	log := waLog.Stdout("Main", cfg.LogLevel, true)
	log.Infof("Starting Seruwit WA Engine (Multi-Tenant, whatsmeow)...")

	// 1. Initialize Database
	db, err := database.Init(cfg.DBDialect, cfg.DBAddress, waLog.Stdout("DB", "WARN", true))
	if err != nil {
		log.Errorf("Database connection failed: %v", err)
		os.Exit(1)
	}
	defer db.Container.Close()
	defer db.DB.Close()
	log.Infof("Connected to database (%s)", cfg.DBDialect)

	// 2. Initialize Webhook Dispatcher
	dispatcher := webhook.NewDispatcher(cfg.CRMWebhookURL, cfg.CRMWebhookSecret, waLog.Stdout("Webhook", "WARN", true))
	if cfg.CRMWebhookURL != "" {
		log.Infof("CRM Webhook configured: %s", cfg.CRMWebhookURL)
	}

	// 3. Initialize WhatsApp Session Manager
	waManager := whatsapp.NewManager(db, dispatcher, log)

	// 4. Auto-reconnect all previously connected sessions
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	if err := waManager.LoadSavedSessions(ctx); err != nil {
		log.Warnf("Error while restoring saved sessions: %v", err)
	}
	cancel()

	// 5. Setup HTTP REST API Server
	app := api.NewServer(cfg, waManager)

	// 6. Graceful Shutdown listener
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		addr := fmt.Sprintf(":%s", cfg.Port)
		log.Infof("Seruwit WA REST API listening on %s", addr)
		if err := app.Listen(addr); err != nil {
			log.Errorf("Server error: %v", err)
		}
	}()

	<-shutdownChan
	log.Infof("Shutting down gracefully...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		log.Errorf("Error during server shutdown: %v", err)
	}

	log.Infof("Server stopped cleanly.")
}

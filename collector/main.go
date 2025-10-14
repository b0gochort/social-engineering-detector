package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"collector/pkg/api"
	"collector/pkg/collector"
	"collector/pkg/config"
	"collector/pkg/storage"
	"collector/pkg/telegram"
)

func main() {
	// Load configuration
	cfgPath := "configs/config.yml"
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		logrus.Fatalf("Failed to load config: %v", err)
	}

	// Apply database migrations
	logrus.Info("Applying database migrations...")
	if err := storage.ApplyMigrations(cfg.Database.URL, "./migrations"); err != nil {
		logrus.Fatalf("Failed to apply migrations: %v", err)
	}
	logrus.Info("Database migrations applied successfully.")

	// Initialize Storage
	dbStorage, err := storage.NewStorage(cfg.Database.URL)
	if err != nil {
		logrus.Fatalf("Failed to create storage: %v", err)
	}
	defer func() {
		if err := dbStorage.Close(); err != nil {
			logrus.Printf("Error closing database: %v", err)
		}
	}()

	// Context for Telegram client and API server
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Initialize Telegram client
	tgClient, err := telegram.NewClient(&cfg.Telegram)
	if err != nil {
		logrus.Fatalf("Failed to create Telegram client: %v", err)
	}

	// Initialize API server
	apiServer := api.NewAPIServer(tgClient)

	// Run API server in a goroutine
	go func() {
		if err := apiServer.Start(":8080"); err != nil {
			logrus.Fatalf("API server failed to start: %v", err)
		}
	}()

	// Run Telegram client and authenticate
	logrus.Info("Starting Telegram client...")
	go func() {
		if err := tgClient.Run(ctx, cfg.Telegram.Phone); err != nil {
			logrus.Fatalf("Telegram client failed to run: %v", err)
		}
	}()

	// Wait for authentication to complete
	select {
	case <-tgClient.AuthCompleted:
		logrus.Info("Telegram authentication completed.")
	case <-ctx.Done():
		logrus.Info("Application interrupted during Telegram client startup.")
		return
	}

	// Initialize and run message collector
	collectionInterval, err := time.ParseDuration(cfg.CollectorInterval)
	if err != nil {
		logrus.Fatalf("Failed to parse collector interval: %v", err)
	}
	msgCollector := collector.NewCollection(tgClient, dbStorage, collectionInterval)
	go msgCollector.Run(ctx)

	<-ctx.Done()
	logrus.Info("Application stopped.")
}

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"time"

	"collector/pkg/api"
	"collector/pkg/collector"
	"collector/pkg/config"
	"collector/pkg/telegram"
	"collector/pkg/vk"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err) // Should not happen in development
	}
	defer func() {
		_ = logger.Sync() // Flushes buffer, if any
	}()

	// Load configuration
	cfgPath := "configs/config.yml"
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	// Context for Telegram client and API server
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Initialize Telegram client
	tgClient, err := telegram.NewClient(&cfg.Telegram)
	if err != nil {
		logger.Fatal("Failed to create Telegram client", zap.Error(err))
	}

	// Initialize VK client and collector (optional)
	var vkCollector *collector.VKCollector
	if cfg.VK.Enabled && cfg.VK.AccessToken != "" {
		vkClient, err := vk.NewClient(&cfg.VK, logger)
		if err != nil {
			logger.Warn("Failed to create VK client, VK collection will be disabled", zap.Error(err))
		} else {
			vkCollector = collector.NewVKCollector(vkClient, logger, 5*time.Minute)
			logger.Info("VK collector initialized successfully")
		}
	} else {
		logger.Info("VK collector is disabled in config")
	}

	// Initialize API server with both Telegram and VK clients
	apiServer := api.NewAPIServer(tgClient, vkCollector, cfg.VK.AppID, cfg.VK.RedirectURI, logger)

	// Run API server in a goroutine
	go func() {
		if err := apiServer.Start(cfg.API.Port); err != nil {
			logger.Fatal("API server failed to start", zap.Error(err))
		}
	}()

	// Run Telegram client and authenticate
	logger.Info("Starting Telegram client...")
	go func() {
		if err := tgClient.Run(ctx, cfg.Telegram.Phone); err != nil {
			logger.Fatal("Telegram client failed to run", zap.Error(err))
		}
	}()

	// Wait for authentication to complete
	select {
	case <-tgClient.AuthCompleted:
		logger.Info("Telegram authentication completed.")
	case <-ctx.Done():
		logger.Info("Application interrupted during Telegram client startup.")
		return
	}

	// The collector will now be triggered via API calls, not run continuously.

	<-ctx.Done()
	logger.Info("Application stopped.")
}

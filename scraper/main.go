package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"scraper/pkg/api"
	"scraper/pkg/config"
	"scraper/pkg/storage"
	"scraper/pkg/telegram"
)

func main() {
	// Load configuration
	cfgPath := "configs/config.yml"
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Apply database migrations
	log.Println("Applying database migrations...")
	if err := storage.ApplyMigrations(cfg.Database.URL, "./migrations"); err != nil {
		log.Fatalf("Failed to apply migrations: %v", err)
	}
	log.Println("Database migrations applied successfully.")

	// Initialize Storage
	dbStorage, err := storage.NewStorage(cfg.Database.URL)
	if err != nil {
		log.Fatalf("Failed to create storage: %v", err)
	}
	defer func() {
		if err := dbStorage.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}()

	// Context for Telegram client and API server
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Initialize Telegram client
	tgClient, err := telegram.NewClient(&cfg.Telegram)
	if err != nil {
		log.Fatalf("Failed to create Telegram client: %v", err)
	}

	// Initialize API server
	apiServer := api.NewAPIServer(tgClient)

	// Run API server in a goroutine
	go func() {
		if err := apiServer.Start(":8080"); err != nil {
			log.Fatalf("API server failed to start: %v", err)
		}
	}()

	// Run Telegram client and authenticate
	log.Println("Starting Telegram client...")
	go func() {
		if err := tgClient.Run(ctx, cfg.Telegram.Phone); err != nil {
			log.Fatalf("Telegram client failed to run: %v", err)
		}
	}()

	// Wait for authentication to complete
	select {
	case <-tgClient.AuthCompleted:
		log.Println("Telegram authentication completed.")
	case <-ctx.Done():
		log.Println("Application interrupted during Telegram client startup.")
		return
	}

	// Fetch messages
	log.Println("Fetching messages...")
	messages, err := tgClient.GetMessages(ctx)
	if err != nil {
		log.Fatalf("Failed to fetch messages: %v", err)
	}

	log.Printf("Fetched %d messages. Saving to database...", len(messages))
	for _, msg := range messages {
		if err := dbStorage.SaveMessage(msg); err != nil {
			log.Printf("Error saving message to database: %v", err)
		}
	}
	log.Println("Messages saved to database.")

	<-ctx.Done()
	log.Println("Application stopped.")
}

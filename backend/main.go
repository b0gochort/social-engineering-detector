package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"backend/internal/annotation_client"
	"backend/internal/collector_client"
	"backend/internal/config"
	"backend/internal/crypto"
	"backend/internal/message_processor"
	"backend/internal/ml_client"
	"backend/internal/models"
	"backend/internal/repository"
	"backend/internal/server"
	"backend/internal/telegram_bot"
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

	// Database connection
	db, err := repository.NewPostgresDB(cfg.Database.URL, logger)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Run migrations
	repository.MigrateDB(db, logger)

	// Initialize KeyManager for encryption/decryption
	keyManager, err := crypto.NewKeyManager()
	if err != nil {
		logger.Fatal("Failed to initialize KeyManager", zap.Error(err))
	}
	logger.Info("KeyManager initialized successfully")

	// Get system user (ID=1) for encrypting messages
	authRepo := repository.NewAuthRepository(db, logger)
	systemUser, err := authRepo.GetUserByUsername("admin")
	if err != nil {
		logger.Warn("System user 'admin' not found - encryption will fail until user is created", zap.Error(err))
		systemUser = &models.User{ID: 1, DKEncrypted: ""} // Placeholder
	}

	// Initialize repositories
	messageRepo := repository.NewMessageRepository(db, logger)
	chatRepo := repository.NewChatRepository(db, logger)
	mlDatasetRepo := repository.NewMLDatasetRepository(db.DB)

	// Initialize collector client
	collectorClient := collector_client.NewClient(cfg.Collector.URL, logger)

	// Initialize ML service client
	mlClient := ml_client.NewClient(cfg.MLService.URL)

	// Initialize annotation service client (optional - для сбора датасета)
	var annotationClient *annotation_client.Client
	if cfg.AnnotationService.Enabled {
		annotationClient = annotation_client.NewClient(cfg.AnnotationService.URL, logger)
		logger.Info("Annotation Service enabled for dataset collection")
	}

	// Initialize message processor
	processor := message_processor.NewProcessor(collectorClient, mlClient, annotationClient, messageRepo, chatRepo, mlDatasetRepo, keyManager, systemUser.ID, systemUser.DKEncrypted, logger, cfg.Collector.PollInterval, cfg.Collector.ChatProcessDelay)

	// Initialize Telegram bot for access control notifications
	accessRequestRepo := repository.NewAccessRequestRepository(db, logger)
	bot, err := telegram_bot.NewBot(cfg, accessRequestRepo, messageRepo, logger)
	if err != nil {
		logger.Warn("Failed to initialize Telegram bot, continuing without it", zap.Error(err))
		bot = nil
	}

	// Context for graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Run Telegram bot in a goroutine (if enabled)
	if bot != nil {
		go func() {
			if err := bot.Start(ctx); err != nil {
				logger.Error("Telegram bot failed", zap.Error(err))
			}
		}()
	}

	// Run message processor in a goroutine
	go processor.Run(ctx)

	// Initialize and run the server
	srv := server.NewServer(db, cfg, logger, bot, collectorClient, keyManager)
	srv.Run(cfg.Server.Port)

	<-ctx.Done()
	logger.Info("Application stopped.")
}

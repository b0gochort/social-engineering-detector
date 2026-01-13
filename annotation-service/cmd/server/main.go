package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"annotation-service/internal/config"
	"annotation-service/internal/gemini"
	"annotation-service/internal/handler"
	"annotation-service/internal/llm"
	"annotation-service/internal/repository"
	"annotation-service/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	// Initialize logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	logger.Info("Starting Annotation Service...")

	// Load configuration
	cfg, err := config.LoadConfig("configs/config.yml")
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	// Initialize LLM client (multi-provider with rate limiting)
	var llmClient service.LLMClient

	// Try to use multi-provider if providers are configured
	if len(cfg.Providers) > 0 {
		multiClient, err := llm.NewMultiProviderClient(llm.MultiProviderConfig{
			Providers:   cfg.Providers,
			MaxFailures: cfg.MaxFailuresBeforeSwitch,
		}, logger)
		if err != nil {
			logger.Warn("Failed to initialize multi-provider client, falling back to single provider",
				zap.Error(err))
		} else {
			llmClient = multiClient
			defer multiClient.Close()
			logger.Info("Multi-provider client initialized",
				zap.Int("provider_count", len(cfg.Providers)))
		}
	}

	// Fallback to single Gemini client if multi-provider failed or not configured
	if llmClient == nil {
		if cfg.Gemini.APIKey == "" || cfg.Gemini.APIKey == "YOUR_API_KEY_HERE" {
			logger.Fatal("Gemini API key not configured. Please set it in configs/config.yml or environment variable")
		}

		geminiClient, err := gemini.NewClient(gemini.Config{
			APIKey:     cfg.Gemini.APIKey,
			ModelName:  cfg.Gemini.ModelName,
			MaxRetries: cfg.Gemini.MaxRetries,
			RetryDelay: 2 * time.Second,
		}, logger)
		if err != nil {
			logger.Fatal("Failed to initialize Gemini client", zap.Error(err))
		}
		defer geminiClient.Close()

		// Wrap with rate limiting
		llmClient = llm.NewRateLimitedProvider(geminiClient, 8, logger)
		logger.Info("Single provider client initialized with rate limiting")
	}

	// Initialize repository
	// Create data directory if not exists
	os.MkdirAll("./data", 0755)

	repo, err := repository.NewAnnotationRepository(cfg.Database.Path, logger)
	if err != nil {
		logger.Fatal("Failed to initialize repository", zap.Error(err))
	}
	defer repo.Close()

	// Initialize service
	annotator := service.NewAnnotator(llmClient, repo, logger)

	// Initialize HTTP handler
	apiHandler := handler.NewHandler(annotator, logger)

	// Setup Gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	// Add CORS middleware
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Register routes
	apiHandler.RegisterRoutes(router)

	// Start server
	serverAddr := fmt.Sprintf(":%s", cfg.Server.Port)
	logger.Info("Server starting", zap.String("address", serverAddr))

	// Graceful shutdown
	srv := &http.Server{
		Addr:    serverAddr,
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Get model info for logging
	modelInfo := llmClient.GetModelInfo()
	modelName := "unknown"
	if m, ok := modelInfo["model"].(string); ok {
		modelName = m
	}

	logger.Info("Annotation Service is running",
		zap.String("port", cfg.Server.Port),
		zap.String("model", modelName))

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
}

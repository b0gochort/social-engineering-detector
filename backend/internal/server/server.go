package server

import (
	"fmt"
	"net/http"

	"backend/internal/collector_client"
	"backend/internal/config"
	"backend/internal/crypto"
	"backend/internal/handler"
	"backend/internal/middleware"
	"backend/internal/repository"
	"backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type Server struct {
	router          *gin.Engine
	db              *sqlx.DB
	cfg             *config.Config
	logger          *zap.Logger
	bot             handler.TelegramBot
	collectorClient *collector_client.Client
	keyManager      *crypto.KeyManager
}

func NewServer(db *sqlx.DB, cfg *config.Config, logger *zap.Logger, bot handler.TelegramBot, collectorClient *collector_client.Client, keyManager *crypto.KeyManager) *Server {
	router := gin.Default()

	// Add CORS middleware
	router.Use(middleware.CORSMiddleware())

	// Initialize server with DB, Config and Logger
	s := &Server{
		router:          router,
		db:              db,
		cfg:             cfg,
		logger:          logger,
		bot:             bot,
		collectorClient: collectorClient,
		keyManager:      keyManager,
	}

	// Setup routes
	s.setupRoutes()

	return s
}

func (s *Server) setupRoutes() {
	// Initialize repositories
	authRepo := repository.NewAuthRepository(s.db, s.logger)
	messageRepo := repository.NewMessageRepository(s.db, s.logger)
	chatRepo := repository.NewChatRepository(s.db, s.logger)
	accessRequestRepo := repository.NewAccessRequestRepository(s.db, s.logger)

	// Initialize services
	authService := service.NewAuthService(authRepo, s.keyManager, s.logger)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService, s.logger)
	incidentHandler := handler.NewIncidentHandler(messageRepo, authRepo, s.cfg, s.logger, s.keyManager)
	chatHandler := handler.NewChatHandler(chatRepo, s.logger)
	vkHandler := handler.NewVKHandler(s.collectorClient, chatRepo, s.logger)
	configHandler := handler.NewConfigHandler(s.cfg, s.collectorClient, s.logger)
	analyticsHandler := handler.NewAnalyticsHandler(messageRepo, chatRepo, s.logger)
	mlDatasetHandler := handler.NewMLDatasetHandler(s.db.DB, s.logger)
	accessRequestHandler := handler.NewAccessRequestHandler(accessRequestRepo, messageRepo, authRepo, s.cfg, s.logger, s.bot)
	settingsHandler := handler.NewSettingsHandler(s.cfg, s.logger)

	// Ping route for health check
	s.router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	// Authentication routes (public)
	authGroup := s.router.Group("/api/auth")
	{
		authGroup.POST("/register", authHandler.RegisterParent)
		authGroup.POST("/login", authHandler.Login)
	}

	// Authenticated routes
	authRequired := s.router.Group("/api")
	authRequired.Use(middleware.AuthMiddleware(s.logger))
	{
		// Auth endpoints
		authRequired.POST("/auth/logout", authHandler.Logout)

		// Incidents endpoints
		authRequired.GET("/events", incidentHandler.GetAllIncidents)
		authRequired.GET("/events/:id", incidentHandler.GetIncidentByID)
		authRequired.PUT("/events/:id/status", incidentHandler.UpdateIncidentStatus)

		// Chats endpoints
		authRequired.GET("/chats", chatHandler.GetAllChats)
		authRequired.GET("/chats/:id", chatHandler.GetChatByID)
		authRequired.PUT("/chats/:id/monitoring", chatHandler.UpdateMonitoringStatus)

		// VK endpoints
		authRequired.GET("/vk/conversations", vkHandler.GetVKConversations)
		authRequired.POST("/vk/chats", vkHandler.AddVKChatToMonitoring)
		authRequired.POST("/vk/chats/:id/collect", vkHandler.CollectVKMessages)

		// Config endpoints
		authRequired.GET("/config/collector", configHandler.GetCollectorConfig)
		authRequired.GET("/config/collector/test", configHandler.TestCollectorConnection)
		authRequired.POST("/config/collector/save", configHandler.SaveCollectorConfig)
		authRequired.POST("/config/collector/restart", configHandler.RestartCollector)
		authRequired.GET("/config/vk/auth-url", configHandler.GetVKAuthURL)
		authRequired.POST("/config/telegram", configHandler.UpdateTelegramConfig)
		authRequired.POST("/config/vk", configHandler.UpdateVKConfig)

		// Settings endpoints
		authRequired.GET("/settings", settingsHandler.GetSettings)
		authRequired.POST("/settings", settingsHandler.UpdateSettings)

		// Analytics endpoints
		authRequired.GET("/analytics/dashboard", analyticsHandler.GetDashboard)

		// ML Dataset endpoints (for training and validation)
		authRequired.GET("/ml-dataset", mlDatasetHandler.GetAllEntries)
		authRequired.POST("/ml-dataset", mlDatasetHandler.CreateEntry)
		authRequired.GET("/ml-dataset/stats", mlDatasetHandler.GetDatasetStats)
		authRequired.GET("/ml-dataset/category/:category_id", mlDatasetHandler.GetEntriesByCategory)
		authRequired.GET("/ml-dataset/validated", mlDatasetHandler.GetValidatedEntries)
		authRequired.GET("/ml-dataset/unvalidated", mlDatasetHandler.GetUnvalidatedEntries)
		authRequired.GET("/ml-dataset/export", mlDatasetHandler.ExportDataset)
		authRequired.POST("/ml-dataset/:id/validate", mlDatasetHandler.ValidateEntry)

		// Access Request endpoints (for access control feature)
		authRequired.POST("/access-requests", accessRequestHandler.CreateAccessRequest)
		authRequired.GET("/access-requests/incident/:id", accessRequestHandler.GetAccessRequestStatus)
		authRequired.GET("/access-requests/pending", accessRequestHandler.GetPendingRequests)
		authRequired.POST("/access-requests/:id/approve", accessRequestHandler.ApproveAccessRequest)
		authRequired.POST("/access-requests/:id/reject", accessRequestHandler.RejectAccessRequest)

		// Protected test endpoint
		authRequired.GET("/protected", func(c *gin.Context) {
			username := c.MustGet("username").(string)
			role := c.MustGet("role").(string)
			c.JSON(http.StatusOK, gin.H{"message": "Welcome to protected area", "username": username, "role": role})
		})
	}
}

func (s *Server) Run(port string) {
	addr := fmt.Sprintf(":%s", port)
	s.logger.Info("Server starting", zap.String("address", addr))
	if err := s.router.Run(addr); err != nil {
		s.logger.Fatal("Server failed to start", zap.Error(err))
	}
}

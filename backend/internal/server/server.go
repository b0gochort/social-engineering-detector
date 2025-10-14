package server

import (
	"net/http"

	"backend/internal/handler"
	"backend/internal/repository"
	"backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

type Server struct {
	router *gin.Engine
	db     *sqlx.DB
	log    *logrus.Logger
}

func NewServer(db *sqlx.DB, log *logrus.Logger) *Server {
	router := gin.Default()

	// Initialize server with DB and Logger
	s := &Server{
		router: router,
		db:     db,
		log:    log,
	}

	// Setup routes
	s.setupRoutes()

	return s
}

func (s *Server) setupRoutes() {
	// Initialize Auth components
	authRepo := repository.NewAuthRepository(s.db, s.log)
	authService := service.NewAuthService(authRepo, s.log)
	authHandler := handler.NewAuthHandler(authService, s.log)

	// Ping route for health check
	s.router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	// Authentication routes
	authGroup := s.router.Group("/api/auth")
	authGroup.POST("/register", authHandler.RegisterParent)
	authGroup.POST("/login", authHandler.Login)

	// Authenticated routes
	authRequired := s.router.Group("/api")
	authRequired.Use(middleware.AuthMiddleware(s.log))
	{
		authRequired.GET("/protected", func(c *gin.Context) {
			username := c.MustGet("username").(string)
			role := c.MustGet("role").(string)
			c.JSON(http.StatusOK, gin.H{"message": "Welcome to protected area", "username": username, "role": role})
		})
		authRequired.POST("/auth/logout", authHandler.Logout)
	}
}

func (s *Server) Run(addr string) {
	s.log.Infof("Server starting on port %s...", addr)
	if err := s.router.Run(addr); err != nil {
		s.log.Fatalf("Server failed to start: %v", err)
	}
}

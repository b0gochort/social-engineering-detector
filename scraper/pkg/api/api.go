package api

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"scraper/pkg/telegram"
)

// APIServer holds the Gin engine and a reference to the Telegram client.
type APIServer struct {
	router *gin.Engine
	tgClient *telegram.Client
}

// NewAPIServer creates a new API server instance.
func NewAPIServer(tgClient *telegram.Client) *APIServer {
	router := gin.Default()
	server := &APIServer{
		router: router,
		tgClient: tgClient,
	}
	server.setupRoutes()
	return server
}

func (s *APIServer) setupRoutes() {
	// Endpoint to submit Telegram authentication code
	// Example: POST /auth/code with JSON body {"code": "12345"}
	s.router.POST("/auth/code", s.handleAuthCode)
}

type authCodeRequest struct {
	Code string `json:"code" binding:"required"`
}

func (s *APIServer) handleAuthCode(c *gin.Context) {
	var req authCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	select {
	case s.tgClient.AuthCode <- req.Code:
		c.JSON(http.StatusOK, gin.H{"message": "Authentication code received."})
	case <-c.Request.Context().Done():
		c.JSON(http.StatusRequestTimeout, gin.H{"error": "Request timed out or cancelled."})
	case <-time.After(5 * time.Second): // Timeout for sending code to channel
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Telegram client not ready to receive code."})
	}
}

// Start runs the API server on the specified address.
func (s *APIServer) Start(addr string) error {
	log.Printf("API server starting on %s", addr)
	return s.router.Run(addr)
}

// Stop gracefully shuts down the API server.
func (s *APIServer) Stop(ctx context.Context) error {
	log.Println("API server stopping...")
	// In Gin, router.Run() is blocking, and there's no direct Stop method.
	// For graceful shutdown, you'd typically use a custom http.Server.
	// For this example, we'll rely on the context cancellation to stop the main app.
	return nil
}

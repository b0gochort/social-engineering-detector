package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"collector/pkg/telegram"
)

// VKCollectorInterface defines methods for VK collection
type VKCollectorInterface interface {
	// Public content (groups/communities)
	CollectWallPosts(ctx context.Context, groupID string, lastPostID int64) (interface{}, error)
	CollectPostComments(ctx context.Context, ownerID int64, postID int64, lastCommentID int64) (interface{}, error)
	GetGroupInfo(ctx context.Context, groupID string) (interface{}, error)
	// Private messages (requires OAuth)
	GetAllConversations(ctx context.Context) (interface{}, error)
	CollectConversationMessages(ctx context.Context, peerID int64, lastMessageID int64) (interface{}, error)
}

// APIServer holds the Gin engine and references to Telegram and VK clients.
type APIServer struct {
	router      *gin.Engine
	tgClient    *telegram.Client
	vkCollector VKCollectorInterface
	vkAppID     int
	vkRedirectURI string
	logger      *zap.Logger
}

// NewAPIServer creates a new API server instance.
func NewAPIServer(tgClient *telegram.Client, vkCollector VKCollectorInterface, vkAppID int, vkRedirectURI string, logger *zap.Logger) *APIServer {
	router := gin.Default()

	// Add CORS middleware
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	server := &APIServer{
		router:        router,
		tgClient:      tgClient,
		vkCollector:   vkCollector,
		vkAppID:       vkAppID,
		vkRedirectURI: vkRedirectURI,
		logger:        logger,
	}
	server.setupRoutes()
	return server
}

func (s *APIServer) setupRoutes() {
	// Telegram endpoints
	tg := s.router.Group("/telegram")
	{
		// Endpoint to submit Telegram authentication code
		tg.POST("/auth/code", s.handleAuthCode)
		// Endpoint to collect messages
		tg.GET("/collect", s.handleCollectMessages)
		// Endpoint to get all available chats
		tg.GET("/chats", s.handleGetChats)
	}

	// VK endpoints
	vk := s.router.Group("/vk")
	{
		// OAuth endpoints
		vk.GET("/auth/url", s.handleGetVKAuthURL)

		// Public content (groups/communities)
		vk.GET("/group/info", s.handleGetVKGroupInfo)
		vk.GET("/wall/posts", s.handleCollectVKWallPosts)
		vk.GET("/wall/comments", s.handleCollectVKPostComments)

		// Private messages (requires OAuth token)
		vk.GET("/conversations", s.handleGetVKConversations)
		vk.GET("/messages/collect", s.handleCollectVKMessages)
	}
}

type authCodeRequest struct {
	Code string `json:"code" binding:"required"`
}

func (s *APIServer) handleAuthCode(c *gin.Context) {
	var req authCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.logger.Error("Failed to bind JSON for auth code", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	select {
	case s.tgClient.AuthCode <- req.Code:
		c.JSON(http.StatusOK, gin.H{"message": "Authentication code received."})
	case <-c.Request.Context().Done():
		s.logger.Warn("Auth code request timed out or cancelled.")
		c.JSON(http.StatusRequestTimeout, gin.H{"error": "Request timed out or cancelled."})
	case <-time.After(5 * time.Second): // Timeout for sending code to channel
		s.logger.Error("Telegram client not ready to receive code.")
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Telegram client not ready to receive code."})
	}
}

func (s *APIServer) handleCollectMessages(c *gin.Context) {
	chatIDStr := c.Query("chat_id")
	if chatIDStr == "" {
		s.logger.Error("chat_id query parameter is required")
		c.JSON(http.StatusBadRequest, gin.H{"error": "chat_id query parameter is required"})
		return
	}

	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		s.logger.Error("Invalid chat_id format", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chat_id format"})
		return
	}

	lastCollectedMessageIDStr := c.Query("last_collected_message_id")
	var lastCollectedMessageID int64 = 0 // Default to 0 if not provided
	if lastCollectedMessageIDStr != "" {
		lastCollectedMessageID, err = strconv.ParseInt(lastCollectedMessageIDStr, 10, 64)
		if err != nil {
			s.logger.Error("Invalid last_collected_message_id format", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid last_collected_message_id format"})
			return
		}
	}

	messages, err := s.tgClient.GetMessages(c.Request.Context(), chatID, lastCollectedMessageID)
	if err != nil {
		s.logger.Error("Failed to collect messages", zap.Error(err), zap.Int64("chat_id", chatID), zap.Int64("last_collected_message_id", lastCollectedMessageID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to collect messages"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"messages": messages})
}

func (s *APIServer) handleGetChats(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second) // Short timeout for chat list
	defer cancel()

	chats, err := s.tgClient.GetAllChatsInfo(ctx)
	if err != nil {
		s.logger.Error("Failed to get chats from Telegram client", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get chats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"chats": chats})
}

// VK handlers

func (s *APIServer) handleGetVKGroupInfo(c *gin.Context) {
	if s.vkCollector == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "VK collector is not enabled"})
		return
	}

	groupID := c.Query("group_id")
	if groupID == "" {
		s.logger.Error("group_id query parameter is required")
		c.JSON(http.StatusBadRequest, gin.H{"error": "group_id query parameter is required"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	groupInfo, err := s.vkCollector.GetGroupInfo(ctx, groupID)
	if err != nil {
		s.logger.Error("Failed to get VK group info", zap.Error(err), zap.String("group_id", groupID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get VK group info"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"group": groupInfo})
}

func (s *APIServer) handleCollectVKWallPosts(c *gin.Context) {
	if s.vkCollector == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "VK collector is not enabled"})
		return
	}

	groupID := c.Query("group_id")
	if groupID == "" {
		s.logger.Error("group_id query parameter is required")
		c.JSON(http.StatusBadRequest, gin.H{"error": "group_id query parameter is required"})
		return
	}

	lastPostIDStr := c.Query("last_post_id")
	var lastPostID int64 = 0
	if lastPostIDStr != "" {
		var err error
		lastPostID, err = strconv.ParseInt(lastPostIDStr, 10, 64)
		if err != nil {
			s.logger.Error("Invalid last_post_id format", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid last_post_id format"})
			return
		}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	posts, err := s.vkCollector.CollectWallPosts(ctx, groupID, lastPostID)
	if err != nil {
		s.logger.Error("Failed to collect VK wall posts", zap.Error(err), zap.String("group_id", groupID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to collect VK wall posts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"posts": posts})
}

func (s *APIServer) handleCollectVKPostComments(c *gin.Context) {
	if s.vkCollector == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "VK collector is not enabled"})
		return
	}

	ownerIDStr := c.Query("owner_id")
	if ownerIDStr == "" {
		s.logger.Error("owner_id query parameter is required")
		c.JSON(http.StatusBadRequest, gin.H{"error": "owner_id query parameter is required"})
		return
	}

	postIDStr := c.Query("post_id")
	if postIDStr == "" {
		s.logger.Error("post_id query parameter is required")
		c.JSON(http.StatusBadRequest, gin.H{"error": "post_id query parameter is required"})
		return
	}

	ownerID, err := strconv.ParseInt(ownerIDStr, 10, 64)
	if err != nil {
		s.logger.Error("Invalid owner_id format", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid owner_id format"})
		return
	}

	postID, err := strconv.ParseInt(postIDStr, 10, 64)
	if err != nil {
		s.logger.Error("Invalid post_id format", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid post_id format"})
		return
	}

	lastCommentIDStr := c.Query("last_comment_id")
	var lastCommentID int64 = 0
	if lastCommentIDStr != "" {
		lastCommentID, err = strconv.ParseInt(lastCommentIDStr, 10, 64)
		if err != nil {
			s.logger.Error("Invalid last_comment_id format", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid last_comment_id format"})
			return
		}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	comments, err := s.vkCollector.CollectPostComments(ctx, ownerID, postID, lastCommentID)
	if err != nil {
		s.logger.Error("Failed to collect VK post comments", zap.Error(err),
			zap.Int64("owner_id", ownerID),
			zap.Int64("post_id", postID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to collect VK post comments"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"comments": comments})
}

func (s *APIServer) handleGetVKAuthURL(c *gin.Context) {
	if s.vkAppID == 0 {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "VK OAuth is not configured"})
		return
	}

	// Import vk package to use GenerateOAuthURL
	authURL := fmt.Sprintf("https://oauth.vk.com/authorize?client_id=%d&redirect_uri=%s&display=page&scope=messages,offline&response_type=token&v=5.131",
		s.vkAppID, s.vkRedirectURI)

	c.JSON(http.StatusOK, gin.H{
		"auth_url": authURL,
		"instructions": "Open this URL in browser, authorize the app, and copy the access_token from the redirect URL",
	})
}

func (s *APIServer) handleGetVKConversations(c *gin.Context) {
	if s.vkCollector == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "VK collector is not enabled"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	conversations, err := s.vkCollector.GetAllConversations(ctx)
	if err != nil {
		s.logger.Error("Failed to get VK conversations", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get VK conversations"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"conversations": conversations})
}

func (s *APIServer) handleCollectVKMessages(c *gin.Context) {
	if s.vkCollector == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "VK collector is not enabled"})
		return
	}

	peerIDStr := c.Query("peer_id")
	if peerIDStr == "" {
		s.logger.Error("peer_id query parameter is required")
		c.JSON(http.StatusBadRequest, gin.H{"error": "peer_id query parameter is required"})
		return
	}

	peerID, err := strconv.ParseInt(peerIDStr, 10, 64)
	if err != nil {
		s.logger.Error("Invalid peer_id format", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid peer_id format"})
		return
	}

	lastMessageIDStr := c.Query("last_message_id")
	var lastMessageID int64 = 0
	if lastMessageIDStr != "" {
		lastMessageID, err = strconv.ParseInt(lastMessageIDStr, 10, 64)
		if err != nil {
			s.logger.Error("Invalid last_message_id format", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid last_message_id format"})
			return
		}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	messages, err := s.vkCollector.CollectConversationMessages(ctx, peerID, lastMessageID)
	if err != nil {
		s.logger.Error("Failed to collect VK messages", zap.Error(err), zap.Int64("peer_id", peerID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to collect VK messages"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"messages": messages})
}

// Start runs the API server on the specified address.
func (s *APIServer) Start(port string) error {
	addr := fmt.Sprintf(":%s", port)
	s.logger.Info("API server starting", zap.String("address", addr))
	return s.router.Run(addr)
}

// Stop gracefully shuts down the API server.
func (s *APIServer) Stop(ctx context.Context) error {
	s.logger.Info("API server stopping...")
	// In Gin, router.Run() is blocking, and there's no direct Stop method.
	// For graceful shutdown, you'd typically use a custom http.Server.
	// For this example, we'll rely on the context cancellation to stop the main app.
	return nil
}

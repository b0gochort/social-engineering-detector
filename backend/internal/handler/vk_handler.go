package handler

import (
	"context"
	"net/http"
	"strconv"

	"backend/internal/collector_client"
	"backend/internal/models"
	"backend/internal/repository"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type VKHandler interface {
	GetVKConversations(c *gin.Context)
	AddVKChatToMonitoring(c *gin.Context)
	CollectVKMessages(c *gin.Context)
}

type vkHandler struct {
	collectorClient *collector_client.Client
	chatRepo        repository.ChatRepository
	logger          *zap.Logger
}

func NewVKHandler(collectorClient *collector_client.Client, chatRepo repository.ChatRepository, logger *zap.Logger) VKHandler {
	return &vkHandler{
		collectorClient: collectorClient,
		chatRepo:        chatRepo,
		logger:          logger,
	}
}

// GetVKConversations handles GET /api/vk/conversations
// Returns list of available VK conversations from the collector
func (h *vkHandler) GetVKConversations(c *gin.Context) {
	ctx := context.Background()

	conversations, err := h.collectorClient.GetVKConversations(ctx)
	if err != nil {
		h.logger.Error("Failed to get VK conversations from collector", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve VK conversations"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"conversations": conversations})
}

// AddVKChatRequest represents the request body for adding a VK chat to monitoring
type AddVKChatRequest struct {
	PeerID  int64  `json:"peer_id" binding:"required"`
	Name    string `json:"name" binding:"required"`
	IsGroup bool   `json:"is_group"`
	Type    string `json:"type"` // "user", "chat", "group"
}

// AddVKChatToMonitoring handles POST /api/vk/chats
// Adds a VK conversation to the monitoring system
func (h *vkHandler) AddVKChatToMonitoring(c *gin.Context) {
	var req AddVKChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind JSON for adding VK chat", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if chat already exists
	existingChat, err := h.chatRepo.GetChatByVKPeerID(req.PeerID)
	if err != nil {
		h.logger.Error("Failed to check existing VK chat", zap.Int64("peer_id", req.PeerID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check existing chat"})
		return
	}

	if existingChat != nil {
		h.logger.Info("VK chat already exists", zap.Int64("peer_id", req.PeerID), zap.Int64("chat_id", existingChat.ID))
		c.JSON(http.StatusConflict, gin.H{
			"error":   "Chat already exists",
			"chat_id": existingChat.ID,
		})
		return
	}

	// Create new chat
	chat := &models.Chat{
		VKPeerID:               &req.PeerID,
		Source:                 "vk",
		Name:                   req.Name,
		IsGroup:                req.IsGroup,
		ChatType:               req.Type,
		MonitoringActive:       true,
		LastCollectedMessageID: 0,
	}

	err = h.chatRepo.CreateChat(chat)
	if err != nil {
		h.logger.Error("Failed to create VK chat", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create chat"})
		return
	}

	h.logger.Info("VK chat added to monitoring", zap.Int64("peer_id", req.PeerID), zap.Int64("chat_id", chat.ID))
	c.JSON(http.StatusCreated, gin.H{
		"message": "VK chat added to monitoring successfully",
		"chat":    chat,
	})
}

// CollectVKMessages handles POST /api/vk/chats/:id/collect
// Triggers collection of messages from a VK conversation
func (h *vkHandler) CollectVKMessages(c *gin.Context) {
	idStr := c.Param("id")
	chatID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.logger.Error("Invalid chat ID", zap.String("id", idStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chat ID"})
		return
	}

	// Get chat from database
	chat, err := h.chatRepo.GetChatByID(chatID)
	if err != nil {
		h.logger.Error("Failed to get chat", zap.Int64("id", chatID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve chat"})
		return
	}

	if chat == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chat not found"})
		return
	}

	// Verify it's a VK chat
	if chat.Source != "vk" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chat is not a VK conversation"})
		return
	}

	if chat.VKPeerID == nil {
		h.logger.Error("VK chat has no peer_id", zap.Int64("chat_id", chatID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid VK chat configuration"})
		return
	}

	// Collect messages from VK
	ctx := context.Background()
	messages, err := h.collectorClient.GetVKMessages(ctx, *chat.VKPeerID, chat.LastCollectedMessageID)
	if err != nil {
		h.logger.Error("Failed to collect VK messages", zap.Int64("peer_id", *chat.VKPeerID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to collect messages from VK"})
		return
	}

	h.logger.Info("Collected VK messages",
		zap.Int64("chat_id", chatID),
		zap.Int64("peer_id", *chat.VKPeerID),
		zap.Int("message_count", len(messages)))

	// TODO: Process messages (save to DB, send to ML service, etc.)
	// This will be implemented in the next step

	c.JSON(http.StatusOK, gin.H{
		"message":        "VK messages collected successfully",
		"message_count":  len(messages),
		"messages":       messages,
	})
}

package handler

import (
	"database/sql"
	"net/http"
	"strconv"

	"backend/internal/repository"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ChatHandler interface {
	GetAllChats(c *gin.Context)
	GetChatByID(c *gin.Context)
	UpdateMonitoringStatus(c *gin.Context)
}

type chatHandler struct {
	chatRepo repository.ChatRepository
	logger   *zap.Logger
}

func NewChatHandler(chatRepo repository.ChatRepository, logger *zap.Logger) ChatHandler {
	return &chatHandler{chatRepo: chatRepo, logger: logger}
}

// GetAllChats handles GET /api/chats
func (h *chatHandler) GetAllChats(c *gin.Context) {
	chats, err := h.chatRepo.GetAllChats()
	if err != nil {
		h.logger.Error("Failed to get chats", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve chats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"chats": chats})
}

// GetChatByID handles GET /api/chats/:id
func (h *chatHandler) GetChatByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.logger.Error("Invalid chat ID", zap.String("id", idStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chat ID"})
		return
	}

	chat, err := h.chatRepo.GetChatByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Chat not found"})
			return
		}
		h.logger.Error("Failed to get chat", zap.Int64("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve chat"})
		return
	}

	if chat == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chat not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"chat": chat})
}

// UpdateMonitoringStatus handles PUT /api/chats/:id/monitoring
type UpdateMonitoringRequest struct {
	Active bool `json:"active"`
}

func (h *chatHandler) UpdateMonitoringStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.logger.Error("Invalid chat ID", zap.String("id", idStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chat ID"})
		return
	}

	var req UpdateMonitoringRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind JSON for monitoring update", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = h.chatRepo.UpdateMonitoringStatus(id, req.Active)
	if err != nil {
		h.logger.Error("Failed to update monitoring status", zap.Int64("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update monitoring status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Monitoring status updated successfully"})
}

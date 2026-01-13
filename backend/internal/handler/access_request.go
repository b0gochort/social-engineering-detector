package handler

import (
	"backend/internal/config"
	"backend/internal/models"
	"backend/internal/repository"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// TelegramBot interface для отправки уведомлений
type TelegramBot interface {
	SendAccessRequestNotification(childTelegramID int64, requestID int64, incidentID int64, threatType string, messageText string) error
}

type AccessRequestHandler interface {
	CreateAccessRequest(c *gin.Context)
	GetAccessRequestStatus(c *gin.Context)
	ApproveAccessRequest(c *gin.Context)
	RejectAccessRequest(c *gin.Context)
	GetPendingRequests(c *gin.Context)
}

type accessRequestHandler struct {
	accessRequestRepo repository.AccessRequestRepository
	messageRepo       repository.MessageRepository
	authRepo          repository.AuthRepository
	cfg               *config.Config
	logger            *zap.Logger
	bot               TelegramBot
}

func NewAccessRequestHandler(
	accessRequestRepo repository.AccessRequestRepository,
	messageRepo repository.MessageRepository,
	authRepo repository.AuthRepository,
	cfg *config.Config,
	logger *zap.Logger,
	bot TelegramBot,
) AccessRequestHandler {
	return &accessRequestHandler{
		accessRequestRepo: accessRequestRepo,
		messageRepo:       messageRepo,
		authRepo:          authRepo,
		cfg:               cfg,
		logger:            logger,
		bot:               bot,
	}
}

// CreateAccessRequest создаёт новый запрос на доступ к инциденту
func (h *accessRequestHandler) CreateAccessRequest(c *gin.Context) {
	// Check if access control is enabled
	if !h.cfg.AccessControl.Enabled {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access control is disabled"})
		return
	}

	var input models.CreateAccessRequestInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get current user (parent) from context
	username, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	parent, err := h.authRepo.GetUserByUsername(username.(string))
	if err != nil {
		h.logger.Error("Failed to get parent user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
		return
	}

	if parent.Role != "parent" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only parents can request access"})
		return
	}

	// Get incident
	incident, err := h.messageRepo.GetIncidentByID(input.IncidentID)
	if err != nil {
		h.logger.Error("Failed to get incident", zap.Int64("incident_id", input.IncidentID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get incident"})
		return
	}

	if incident == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Incident not found"})
		return
	}

	// Check if there's already a pending or approved request
	existingRequest, err := h.accessRequestRepo.GetByIncidentID(input.IncidentID)
	if err != nil {
		h.logger.Error("Failed to check existing request", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check existing request"})
		return
	}

	if existingRequest != nil && (existingRequest.Status == "pending" || existingRequest.Status == "approved") {
		c.JSON(http.StatusConflict, gin.H{"error": "Access request already exists", "request": existingRequest})
		return
	}

	// Find child user (for now, we'll use the first child of the parent)
	// TODO: In production, determine the child from the incident's message/chat relationship
	// For now, create a simple relationship
	// This is a simplified version - in real implementation, you'd need to determine
	// which child the incident belongs to based on the message/chat relationship

	// For MVP, we'll assume there's a child_id stored somewhere or we need to determine it
	// Let's create the request with parent_id and child_id set to parent for now
	// In production, you'd query to find the actual child

	accessRequest := &models.AccessRequest{
		IncidentID:  input.IncidentID,
		ParentID:    parent.ID,
		ChildID:     parent.ID, // TODO: Replace with actual child ID
		Status:      "pending",
		RequestedAt: time.Now(),
	}

	err = h.accessRequestRepo.Create(accessRequest)
	if err != nil {
		h.logger.Error("Failed to create access request", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create access request"})
		return
	}

	// Update incident with the request ID
	err = h.messageRepo.UpdateIncidentAccessGranted(input.IncidentID, false, &accessRequest.ID)
	if err != nil {
		h.logger.Error("Failed to update incident", zap.Error(err))
		// Continue anyway, the request was created
	}

	// Send Telegram notification to child (if bot is available and child has telegram_id)
	// TODO: Get child's telegram_id from database
	// For MVP demo, we'll use a hardcoded Telegram ID (your collector account)
	if h.bot != nil {
		// MVP: Hardcoded Telegram ID for testing
		// TODO: In production, get this from the child user's telegram_id field
		const childTelegramID int64 = 7280888813

		if childTelegramID != 0 {
			h.logger.Info("Sending Telegram notification",
				zap.Int64("incident_id", input.IncidentID),
				zap.String("threat_type", incident.ThreatType),
				zap.String("summary_text", incident.SummaryEncrypted),
			)
			err = h.bot.SendAccessRequestNotification(childTelegramID, accessRequest.ID, input.IncidentID, incident.ThreatType, incident.SummaryEncrypted)
			if err != nil {
				h.logger.Error("Failed to send Telegram notification",
					zap.Error(err),
					zap.Int64("request_id", accessRequest.ID),
					zap.Int64("incident_id", input.IncidentID),
				)
				// Don't fail the request if notification fails
			} else {
				h.logger.Info("Telegram notification sent to child",
					zap.Int64("child_telegram_id", childTelegramID),
					zap.Int64("request_id", accessRequest.ID),
					zap.Int64("incident_id", input.IncidentID),
				)
			}
		} else {
			h.logger.Warn("Telegram notification skipped: childTelegramID is 0 (not configured)")
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Access request created",
		"request": accessRequest,
	})
}

// GetAccessRequestStatus получает статус запроса на доступ для инцидента
func (h *accessRequestHandler) GetAccessRequestStatus(c *gin.Context) {
	incidentIDStr := c.Param("id")
	incidentID, err := strconv.ParseInt(incidentIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid incident ID"})
		return
	}

	request, err := h.accessRequestRepo.GetByIncidentID(incidentID)
	if err != nil {
		h.logger.Error("Failed to get access request", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get access request"})
		return
	}

	if request == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No access request found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"request": request})
}

// ApproveAccessRequest одобряет запрос на доступ (вызывается ребёнком)
func (h *accessRequestHandler) ApproveAccessRequest(c *gin.Context) {
	requestIDStr := c.Param("id")
	requestID, err := strconv.ParseInt(requestIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID"})
		return
	}

	// Get current user (child) from context
	username, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	user, err := h.authRepo.GetUserByUsername(username.(string))
	if err != nil {
		h.logger.Error("Failed to get user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
		return
	}

	// Get the request
	request, err := h.accessRequestRepo.GetByID(requestID)
	if err != nil {
		h.logger.Error("Failed to get access request", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get access request"})
		return
	}

	if request == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Access request not found"})
		return
	}

	// Verify that the user is the child for this request
	if request.ChildID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not authorized to approve this request"})
		return
	}

	if request.Status != "pending" {
		c.JSON(http.StatusConflict, gin.H{"error": "Request is not pending"})
		return
	}

	// Update request status
	now := time.Now()
	err = h.accessRequestRepo.UpdateStatus(requestID, "approved", now)
	if err != nil {
		h.logger.Error("Failed to update request status", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to approve request"})
		return
	}

	// Update incident access_granted
	err = h.messageRepo.UpdateIncidentAccessGranted(request.IncidentID, true, &requestID)
	if err != nil {
		h.logger.Error("Failed to grant access to incident", zap.Error(err))
		// Continue anyway
	}

	// TODO: Send notification to parent

	c.JSON(http.StatusOK, gin.H{"message": "Access request approved"})
}

// RejectAccessRequest отклоняет запрос на доступ (вызывается ребёнком)
func (h *accessRequestHandler) RejectAccessRequest(c *gin.Context) {
	requestIDStr := c.Param("id")
	requestID, err := strconv.ParseInt(requestIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID"})
		return
	}

	// Get current user (child) from context
	username, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	user, err := h.authRepo.GetUserByUsername(username.(string))
	if err != nil {
		h.logger.Error("Failed to get user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
		return
	}

	// Get the request
	request, err := h.accessRequestRepo.GetByID(requestID)
	if err != nil {
		h.logger.Error("Failed to get access request", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get access request"})
		return
	}

	if request == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Access request not found"})
		return
	}

	// Verify that the user is the child for this request
	if request.ChildID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not authorized to reject this request"})
		return
	}

	if request.Status != "pending" {
		c.JSON(http.StatusConflict, gin.H{"error": "Request is not pending"})
		return
	}

	// Update request status
	now := time.Now()
	err = h.accessRequestRepo.UpdateStatus(requestID, "rejected", now)
	if err != nil {
		h.logger.Error("Failed to update request status", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reject request"})
		return
	}

	// TODO: Send notification to parent

	c.JSON(http.StatusOK, gin.H{"message": "Access request rejected"})
}

// GetPendingRequests получает список ожидающих запросов для ребёнка
func (h *accessRequestHandler) GetPendingRequests(c *gin.Context) {
	// Get current user (child) from context
	username, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	user, err := h.authRepo.GetUserByUsername(username.(string))
	if err != nil {
		h.logger.Error("Failed to get user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
		return
	}

	requests, err := h.accessRequestRepo.GetPendingByChildID(user.ID)
	if err != nil {
		h.logger.Error("Failed to get pending requests", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get pending requests"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"requests": requests})
}

package handler

import (
	"database/sql"
	"net/http"
	"strconv"

	"backend/internal/config"
	"backend/internal/crypto"
	"backend/internal/models"
	"backend/internal/repository"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type IncidentHandler interface {
	GetAllIncidents(c *gin.Context)
	GetIncidentByID(c *gin.Context)
	UpdateIncidentStatus(c *gin.Context)
}

type incidentHandler struct {
	messageRepo  repository.MessageRepository
	authRepo     repository.AuthRepository
	cfg          *config.Config
	logger       *zap.Logger
	keyManager   *crypto.KeyManager
}

func NewIncidentHandler(messageRepo repository.MessageRepository, authRepo repository.AuthRepository, cfg *config.Config, logger *zap.Logger, keyManager *crypto.KeyManager) IncidentHandler {
	return &incidentHandler{
		messageRepo:  messageRepo,
		authRepo:     authRepo,
		cfg:          cfg,
		logger:       logger,
		keyManager:   keyManager,
	}
}

// decryptIncidentSummary расшифровывает summary_encrypted для инцидента
func (h *incidentHandler) decryptIncidentSummary(incident *models.Incident) error {
	if incident.SummaryEncrypted == "" {
		return nil
	}

	// Get system user (admin) for decryption
	systemUser, err := h.authRepo.GetUserByUsername("admin")
	if err != nil {
		h.logger.Error("Failed to get system user for decryption", zap.Error(err))
		return err
	}

	// Decrypt the summary
	decrypted, err := h.keyManager.DecryptMessage(incident.SummaryEncrypted, systemUser.ID, systemUser.DKEncrypted)
	if err != nil {
		h.logger.Error("Failed to decrypt incident summary",
			zap.Int64("incident_id", incident.ID),
			zap.Error(err))
		return err
	}

	incident.SummaryEncrypted = decrypted
	return nil
}

// filterIncidentText скрывает текст сообщения если access control включен и доступ не предоставлен
func (h *incidentHandler) filterIncidentText(incident *models.Incident) {
	h.logger.Debug("Filtering incident text",
		zap.Bool("enabled", h.cfg.AccessControl.Enabled),
		zap.Bool("access_granted", incident.AccessGranted),
		zap.Int64("incident_id", incident.ID),
	)
	if h.cfg.AccessControl.Enabled && !incident.AccessGranted {
		h.logger.Info("Hiding incident text", zap.Int64("incident_id", incident.ID))
		incident.SummaryEncrypted = "[Для просмотра текста запросите доступ]"
	}
}

// GetAllIncidents handles GET /api/events
// Query parameters:
// - status: filter by status (optional)
// - threat_type: filter by threat type (optional)
func (h *incidentHandler) GetAllIncidents(c *gin.Context) {
	status := c.Query("status")
	threatType := c.Query("threat_type")

	var incidents []*models.Incident
	var err error

	if status != "" {
		incidents, err = h.messageRepo.GetIncidentsByStatus(status)
	} else if threatType != "" {
		incidents, err = h.messageRepo.GetIncidentsByThreatType(threatType)
	} else {
		incidents, err = h.messageRepo.GetAllIncidents()
	}

	if err != nil {
		h.logger.Error("Failed to get incidents", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve incidents"})
		return
	}

	// Decrypt and filter incidents
	for _, incident := range incidents {
		// First decrypt the summary
		if err := h.decryptIncidentSummary(incident); err != nil {
			h.logger.Warn("Failed to decrypt incident summary, using encrypted value",
				zap.Int64("incident_id", incident.ID),
				zap.Error(err))
			// Continue with encrypted value
		}

		// Then filter based on access control
		if h.cfg.AccessControl.Enabled {
			h.filterIncidentText(incident)
		}
	}

	c.JSON(http.StatusOK, gin.H{"incidents": incidents})
}

// GetIncidentByID handles GET /api/events/:id
func (h *incidentHandler) GetIncidentByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.logger.Error("Invalid incident ID", zap.String("id", idStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid incident ID"})
		return
	}

	incident, err := h.messageRepo.GetIncidentByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Incident not found"})
			return
		}
		h.logger.Error("Failed to get incident", zap.Int64("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve incident"})
		return
	}

	// First decrypt the summary
	if err := h.decryptIncidentSummary(incident); err != nil {
		h.logger.Warn("Failed to decrypt incident summary, using encrypted value",
			zap.Int64("incident_id", incident.ID),
			zap.Error(err))
		// Continue with encrypted value
	}

	// Then filter based on access control
	if h.cfg.AccessControl.Enabled {
		h.filterIncidentText(incident)
	}

	c.JSON(http.StatusOK, gin.H{"incident": incident})
}

// UpdateIncidentStatus handles PUT /api/events/:id/status
type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

func (h *incidentHandler) UpdateIncidentStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.logger.Error("Invalid incident ID", zap.String("id", idStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid incident ID"})
		return
	}

	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind JSON for status update", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate status values
	validStatuses := map[string]bool{
		"new":            true,
		"reviewed":       true,
		"false_positive": true,
	}
	if !validStatuses[req.Status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status. Valid values: new, reviewed, false_positive"})
		return
	}

	err = h.messageRepo.UpdateIncidentStatus(id, req.Status)
	if err != nil {
		h.logger.Error("Failed to update incident status", zap.Int64("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update incident status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Incident status updated successfully"})
}

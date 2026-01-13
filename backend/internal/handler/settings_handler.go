package handler

import (
	"fmt"
	"net/http"
	"os"

	"backend/internal/config"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type SettingsHandler interface {
	GetSettings(c *gin.Context)
	UpdateSettings(c *gin.Context)
}

type settingsHandler struct {
	cfg    *config.Config
	logger *zap.Logger
}

func NewSettingsHandler(cfg *config.Config, logger *zap.Logger) SettingsHandler {
	return &settingsHandler{
		cfg:    cfg,
		logger: logger,
	}
}

// SettingsResponse represents the current system settings
type SettingsResponse struct {
	AccessControl struct {
		RequireAccessRequest bool `json:"requireAccessRequest"`
		AutoApproveAdmins    bool `json:"autoApproveAdmins"`
	} `json:"accessControl"`
	AnnotationService struct {
		Enabled bool `json:"enabled"`
	} `json:"annotationService"`
}

// GetSettings handles GET /api/settings
func (h *settingsHandler) GetSettings(c *gin.Context) {
	response := SettingsResponse{}
	response.AccessControl.RequireAccessRequest = h.cfg.AccessControl.Enabled
	response.AccessControl.AutoApproveAdmins = false // TODO: добавить в конфиг если нужно
	response.AnnotationService.Enabled = h.cfg.AnnotationService.Enabled

	c.JSON(http.StatusOK, response)
}

// UpdateSettingsRequest represents the settings update request
type UpdateSettingsRequest struct {
	AccessControl *struct {
		RequireAccessRequest *bool `json:"requireAccessRequest"`
		AutoApproveAdmins    *bool `json:"autoApproveAdmins"`
	} `json:"accessControl,omitempty"`
	AnnotationService *struct {
		Enabled *bool `json:"enabled"`
	} `json:"annotationService,omitempty"`
}

// UpdateSettings handles POST /api/settings
func (h *settingsHandler) UpdateSettings(c *gin.Context) {
	var req UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind settings request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Path to backend config file
	configPath := "configs/config.yml"

	// Check if running in Docker
	if _, err := os.Stat("/.dockerenv"); err == nil {
		// Running in Docker
		configPath = "/root/configs/config.yml"
	}

	// Read current config file as generic map to preserve structure
	data, err := os.ReadFile(configPath)
	if err != nil {
		h.logger.Error("Failed to read config file", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read config file"})
		return
	}

	// Parse as generic YAML
	var configData map[string]interface{}
	if err := yaml.Unmarshal(data, &configData); err != nil {
		h.logger.Error("Failed to parse config file", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse config file"})
		return
	}

	// Update access_control settings if provided
	if req.AccessControl != nil {
		if configData["access_control"] == nil {
			configData["access_control"] = make(map[string]interface{})
		}
		accessControl := configData["access_control"].(map[string]interface{})

		if req.AccessControl.RequireAccessRequest != nil {
			accessControl["enabled"] = *req.AccessControl.RequireAccessRequest
			// Update in-memory config
			h.cfg.AccessControl.Enabled = *req.AccessControl.RequireAccessRequest
			h.logger.Info("Access control setting updated", zap.Bool("enabled", *req.AccessControl.RequireAccessRequest))
		}
	}

	// Update annotation_service settings if provided
	if req.AnnotationService != nil {
		if configData["annotation_service"] == nil {
			configData["annotation_service"] = make(map[string]interface{})
		}
		annotationService := configData["annotation_service"].(map[string]interface{})

		if req.AnnotationService.Enabled != nil {
			annotationService["enabled"] = *req.AnnotationService.Enabled
			// Update in-memory config
			h.cfg.AnnotationService.Enabled = *req.AnnotationService.Enabled
			h.logger.Info("Annotation service setting updated", zap.Bool("enabled", *req.AnnotationService.Enabled))
		}
	}

	// Marshal back to YAML
	newData, err := yaml.Marshal(&configData)
	if err != nil {
		h.logger.Error("Failed to marshal config", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal config"})
		return
	}

	// Write config file
	if err := os.WriteFile(configPath, newData, 0644); err != nil {
		h.logger.Error("Failed to write config file", zap.Error(err), zap.String("path", configPath))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to write config file: %v", err),
		})
		return
	}

	h.logger.Info("Settings updated successfully", zap.String("path", configPath))

	c.JSON(http.StatusOK, gin.H{
		"message": "Settings updated successfully",
		"restart_required": false, // Settings applied immediately in-memory
	})
}

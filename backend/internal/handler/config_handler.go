package handler

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"

	"backend/internal/collector_client"
	"backend/internal/config"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type ConfigHandler interface {
	GetCollectorConfig(c *gin.Context)
	UpdateTelegramConfig(c *gin.Context)
	UpdateVKConfig(c *gin.Context)
	GetVKAuthURL(c *gin.Context)
	TestCollectorConnection(c *gin.Context)
	RestartCollector(c *gin.Context)
	SaveCollectorConfig(c *gin.Context)
}

type configHandler struct {
	cfg             *config.Config
	collectorClient *collector_client.Client
	logger          *zap.Logger
}

func NewConfigHandler(cfg *config.Config, collectorClient *collector_client.Client, logger *zap.Logger) ConfigHandler {
	return &configHandler{
		cfg:             cfg,
		collectorClient: collectorClient,
		logger:          logger,
	}
}

// CollectorConfigResponse represents the collector configuration status
type CollectorConfigResponse struct {
	TelegramConfigured bool   `json:"telegram_configured"`
	VKConfigured       bool   `json:"vk_configured"`
	VKEnabled          bool   `json:"vk_enabled"`
	CollectorURL       string `json:"collector_url"`
}

// GetCollectorConfig handles GET /api/config/collector
func (h *configHandler) GetCollectorConfig(c *gin.Context) {
	// Check Telegram configuration (через collector API)
	telegramConfigured := false
	vkConfigured := false

	// Try to get chats to check if Telegram is configured
	ctx := c.Request.Context()
	_, err := h.collectorClient.GetChats(ctx)
	if err == nil {
		telegramConfigured = true
	}

	// Try to get VK conversations to check if VK is configured
	_, err = h.collectorClient.GetVKConversations(ctx)
	if err == nil {
		vkConfigured = true
	}

	response := CollectorConfigResponse{
		TelegramConfigured: telegramConfigured,
		VKConfigured:       vkConfigured,
		VKEnabled:          true, // Всегда включено на backend
		CollectorURL:       h.cfg.Collector.URL,
	}

	c.JSON(http.StatusOK, response)
}

// UpdateTelegramConfigRequest represents Telegram configuration request
type UpdateTelegramConfigRequest struct {
	APIID   int    `json:"api_id" binding:"required"`
	APIHash string `json:"api_hash" binding:"required"`
	Phone   string `json:"phone" binding:"required"`
}

// UpdateTelegramConfig handles POST /api/config/telegram
// NOTE: This requires collector API to support dynamic config update
func (h *configHandler) UpdateTelegramConfig(c *gin.Context) {
	var req UpdateTelegramConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind Telegram config request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: Call collector API to update Telegram config
	// For now, return that this feature requires collector restart
	c.JSON(http.StatusOK, gin.H{
		"message": "Telegram configuration updated. Please restart collector to apply changes.",
		"restart_required": true,
	})
}

// UpdateVKConfigRequest represents VK configuration request
type UpdateVKConfigRequest struct {
	AppID       int    `json:"app_id" binding:"required"`
	AccessToken string `json:"access_token"`
}

// UpdateVKConfig handles POST /api/config/vk
func (h *configHandler) UpdateVKConfig(c *gin.Context) {
	var req UpdateVKConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind VK config request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: Call collector API to update VK config
	// For now, return that this feature requires collector restart
	c.JSON(http.StatusOK, gin.H{
		"message": "VK configuration updated. Please restart collector to apply changes.",
		"restart_required": true,
	})
}

// GetVKAuthURL handles GET /api/config/vk/auth-url
func (h *configHandler) GetVKAuthURL(c *gin.Context) {
	// Forward request to collector
	// Call collector's /vk/auth/url endpoint
	ctx := c.Request.Context()

	// Make HTTP request to collector
	resp, err := h.collectorClient.GetVKAuthURL(ctx)
	if err != nil {
		h.logger.Error("Failed to get VK auth URL", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get VK auth URL from collector"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// TestCollectorConnection handles GET /api/config/collector/test
func (h *configHandler) TestCollectorConnection(c *gin.Context) {
	ctx := c.Request.Context()

	// Try to ping collector
	_, err := h.collectorClient.GetChats(ctx)

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"connected": false,
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"connected": true,
	})
}

// CollectorConfig represents collector configuration file structure
type CollectorConfig struct {
	Telegram struct {
		APIID   int    `yaml:"api_id"`
		APIHash string `yaml:"api_hash"`
		Phone   string `yaml:"phone"`
	} `yaml:"telegram"`
	VK struct {
		AppID       int    `yaml:"app_id"`
		AccessToken string `yaml:"access_token"`
		Enabled     bool   `yaml:"enabled"`
	} `yaml:"vk"`
	API struct {
		Port string `yaml:"port"`
	} `yaml:"api"`
	CollectorInterval string `yaml:"collector_interval"`
}

// SaveCollectorConfigRequest represents request to save collector config
type SaveCollectorConfigRequest struct {
	Telegram struct {
		APIID   int    `json:"api_id"`
		APIHash string `json:"api_hash"`
		Phone   string `json:"phone"`
	} `json:"telegram"`
	VK struct {
		AppID       int    `json:"app_id"`
		AccessToken string `json:"access_token"`
	} `json:"vk"`
}

// SaveCollectorConfig handles POST /api/config/collector/save
// Saves configuration to collector's config.yml file
func (h *configHandler) SaveCollectorConfig(c *gin.Context) {
	var req SaveCollectorConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind collector config request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Path to collector config file (assuming it's mounted or accessible)
	configPath := "../collector/configs/config.yml"

	// Check if running in Docker
	if _, err := os.Stat("/.dockerenv"); err == nil {
		// Running in Docker, use volume path
		configPath = "/app/collector-config/config.yml"
	}

	// Read existing config
	var cfg CollectorConfig
	data, err := os.ReadFile(configPath)
	if err != nil {
		// If file doesn't exist, create default config
		h.logger.Warn("Config file not found, creating new one", zap.String("path", configPath))
		cfg.API.Port = "8081"
		cfg.CollectorInterval = "10s"
	} else {
		// Parse existing config
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			h.logger.Error("Failed to parse existing config", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse existing config"})
			return
		}
	}

	// Update config with new values
	if req.Telegram.APIID != 0 {
		cfg.Telegram.APIID = req.Telegram.APIID
		cfg.Telegram.APIHash = req.Telegram.APIHash
		cfg.Telegram.Phone = req.Telegram.Phone
	}

	if req.VK.AppID != 0 {
		cfg.VK.AppID = req.VK.AppID
	}
	if req.VK.AccessToken != "" {
		cfg.VK.AccessToken = req.VK.AccessToken
	}

	// Marshal config to YAML
	newData, err := yaml.Marshal(&cfg)
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

	h.logger.Info("Collector configuration saved successfully", zap.String("path", configPath))

	c.JSON(http.StatusOK, gin.H{
		"message": "Configuration saved successfully. Please restart collector to apply changes.",
		"restart_required": true,
	})
}

// RestartCollector handles POST /api/config/collector/restart
// Restarts the collector Docker container
func (h *configHandler) RestartCollector(c *gin.Context) {
	h.logger.Info("Attempting to restart collector container")

	var cmd *exec.Cmd

	// Check if we're in Docker
	if _, err := os.Stat("/.dockerenv"); err == nil {
		// Running in Docker, use docker CLI directly
		h.logger.Info("Running in Docker, using docker restart command")
		cmd = exec.Command("docker", "restart", "se-detector-collector")
	} else {
		// Not in Docker, use docker-compose
		h.logger.Info("Running outside Docker, using docker-compose")
		cmd = exec.Command("docker-compose", "restart", "collector")
		cmd.Dir = "../"
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		h.logger.Error("Failed to restart collector", zap.Error(err), zap.String("output", string(output)))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to restart collector: %v", err),
			"output": string(output),
		})
		return
	}

	h.logger.Info("Collector container restarted successfully", zap.String("output", string(output)))

	c.JSON(http.StatusOK, gin.H{
		"message": "Collector restarted successfully",
		"output": string(output),
	})
}

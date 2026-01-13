package handler

import (
	"net/http"
	"time"

	"backend/internal/repository"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type AnalyticsHandler interface {
	GetDashboard(c *gin.Context)
}

type analyticsHandler struct {
	messageRepo repository.MessageRepository
	chatRepo    repository.ChatRepository
	logger      *zap.Logger
}

func NewAnalyticsHandler(messageRepo repository.MessageRepository, chatRepo repository.ChatRepository, logger *zap.Logger) AnalyticsHandler {
	return &analyticsHandler{
		messageRepo: messageRepo,
		chatRepo:    chatRepo,
		logger:      logger,
	}
}

// DashboardStats represents the statistics for the dashboard
type DashboardStats struct {
	TotalIncidents       int                       `json:"total_incidents"`
	NewIncidents         int                       `json:"new_incidents"`
	ReviewedIncidents    int                       `json:"reviewed_incidents"`
	FalsePositives       int                       `json:"false_positives"`
	TotalChats           int                       `json:"total_chats"`
	ActiveChats          int                       `json:"active_chats"`
	IncidentsByThreat    map[string]int            `json:"incidents_by_threat"`
	CategoryDistribution map[int]int               `json:"category_distribution"`  // Category ID -> Count
	TotalMessages        int                       `json:"total_messages"`
	DetectionRate        float64                   `json:"detection_rate"`
	Incidents24h         int                       `json:"incidents_24h"`
	RecentIncidents      interface{}               `json:"recent_incidents"`
}

// Threat type to category ID mapping
var threatTypeToCategoryID = map[string]int{
	"Склонение к сексуальным действиям (Груминг)":            1,
	"Угрозы, шантаж, вымогательство":                         2,
	"Физическое насилие/Буллинг":                             3,
	"Склонение к суициду/Самоповреждение":                    4,
	"Склонение к опасным играм/действиям":                    5,
	"Пропаганда запрещенных веществ":                         6,
	"Финансовое мошенничество":                               7,
	"Сбор личных данных (Фишинг)":                            8,
	"Нейтральное общение":                                    9,
}

// GetDashboard handles GET /api/analytics/dashboard
func (h *analyticsHandler) GetDashboard(c *gin.Context) {
	// Get all incidents
	allIncidents, err := h.messageRepo.GetAllIncidents()
	if err != nil {
		h.logger.Error("Failed to get all incidents for dashboard", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve dashboard data"})
		return
	}

	// Get incidents by status
	newIncidents, err := h.messageRepo.GetIncidentsByStatus("new")
	if err != nil {
		h.logger.Error("Failed to get new incidents", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve dashboard data"})
		return
	}

	reviewedIncidents, err := h.messageRepo.GetIncidentsByStatus("reviewed")
	if err != nil {
		h.logger.Error("Failed to get reviewed incidents", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve dashboard data"})
		return
	}

	falsePositives, err := h.messageRepo.GetIncidentsByStatus("false_positive")
	if err != nil {
		h.logger.Error("Failed to get false positive incidents", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve dashboard data"})
		return
	}

	// Get all chats
	allChats, err := h.chatRepo.GetAllChats()
	if err != nil {
		h.logger.Error("Failed to get all chats for dashboard", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve dashboard data"})
		return
	}

	// Count active chats
	activeChatsCount := 0
	totalMessages := 0
	for _, chat := range allChats {
		if chat.MonitoringActive {
			activeChatsCount++
		}
		totalMessages += chat.MessageCount
	}

	// Count incidents by threat type (for backward compatibility) and category ID
	incidentsByThreat := make(map[string]int)
	categoryDistribution := make(map[int]int)
	incidents24h := 0
	now := time.Now()
	twentyFourHoursAgo := now.Add(-24 * time.Hour)

	for _, incident := range allIncidents {
		incidentsByThreat[incident.ThreatType]++

		// Map to category ID
		if categoryID, ok := threatTypeToCategoryID[incident.ThreatType]; ok {
			categoryDistribution[categoryID]++
		}

		// Count incidents in last 24 hours
		if incident.CreatedAt.After(twentyFourHoursAgo) {
			incidents24h++
		}
	}

	// Calculate detection rate
	detectionRate := 0.0
	if totalMessages > 0 {
		detectionRate = float64(len(allIncidents)) / float64(totalMessages)
	}

	// Get recent incidents (last 10)
	recentIncidents := allIncidents
	if len(recentIncidents) > 10 {
		recentIncidents = allIncidents[:10]
	}

	stats := DashboardStats{
		TotalIncidents:       len(allIncidents),
		NewIncidents:         len(newIncidents),
		ReviewedIncidents:    len(reviewedIncidents),
		FalsePositives:       len(falsePositives),
		TotalChats:           len(allChats),
		ActiveChats:          activeChatsCount,
		IncidentsByThreat:    incidentsByThreat,
		CategoryDistribution: categoryDistribution,
		TotalMessages:        totalMessages,
		DetectionRate:        detectionRate,
		Incidents24h:         incidents24h,
		RecentIncidents:      recentIncidents,
	}

	c.JSON(http.StatusOK, stats)
}

package handler

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"backend/internal/models"
	"backend/internal/repository"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// MLDatasetHandler handles ML dataset-related requests.
type MLDatasetHandler struct {
	mlDatasetRepo repository.MLDatasetRepository
	logger        *zap.Logger
}

// NewMLDatasetHandler creates a new ML dataset handler.
func NewMLDatasetHandler(db *sql.DB, logger *zap.Logger) *MLDatasetHandler {
	return &MLDatasetHandler{
		mlDatasetRepo: repository.NewMLDatasetRepository(db),
		logger:        logger,
	}
}

// GetAllEntries returns all ML dataset entries.
// GET /api/ml-dataset
func (h *MLDatasetHandler) GetAllEntries(c *gin.Context) {
	entries, err := h.mlDatasetRepo.GetAllEntries()
	if err != nil {
		h.logger.Error("Failed to get ML dataset entries", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch dataset entries"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"entries": entries,
		"count":   len(entries),
	})
}

// GetEntriesByCategory returns ML dataset entries filtered by category.
// GET /api/ml-dataset/category/:category_id
func (h *MLDatasetHandler) GetEntriesByCategory(c *gin.Context) {
	categoryIDStr := c.Param("category_id")
	categoryID, err := strconv.Atoi(categoryIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
		return
	}

	entries, err := h.mlDatasetRepo.GetEntriesByCategory(categoryID)
	if err != nil {
		h.logger.Error("Failed to get ML dataset entries by category", zap.Error(err), zap.Int("category_id", categoryID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch dataset entries"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"entries":     entries,
		"count":       len(entries),
		"category_id": categoryID,
	})
}

// GetValidatedEntries returns only validated ML dataset entries.
// GET /api/ml-dataset/validated
func (h *MLDatasetHandler) GetValidatedEntries(c *gin.Context) {
	entries, err := h.mlDatasetRepo.GetValidatedEntries()
	if err != nil {
		h.logger.Error("Failed to get validated ML dataset entries", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch validated entries"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"entries": entries,
		"count":   len(entries),
	})
}

// GetUnvalidatedEntries returns only unvalidated ML dataset entries.
// GET /api/ml-dataset/unvalidated
func (h *MLDatasetHandler) GetUnvalidatedEntries(c *gin.Context) {
	entries, err := h.mlDatasetRepo.GetUnvalidatedEntries()
	if err != nil {
		h.logger.Error("Failed to get unvalidated ML dataset entries", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch unvalidated entries"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"entries": entries,
		"count":   len(entries),
	})
}

// ValidateEntry marks an entry as validated.
// POST /api/ml-dataset/:id/validate
func (h *MLDatasetHandler) ValidateEntry(c *gin.Context) {
	entryIDStr := c.Param("id")
	entryID, err := strconv.ParseInt(entryIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid entry ID"})
		return
	}

	// Get user ID from context (set by auth middleware)
	// For now, we'll use a placeholder
	// TODO: Get actual user ID from JWT token
	var validatedBy int64 = 1

	err = h.mlDatasetRepo.ValidateEntry(entryID, validatedBy)
	if err != nil {
		h.logger.Error("Failed to validate ML dataset entry", zap.Error(err), zap.Int64("entry_id", entryID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate entry"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Entry validated successfully",
		"entry_id": entryID,
	})
}

// GetDatasetStats returns statistics about the ML dataset.
// GET /api/ml-dataset/stats
func (h *MLDatasetHandler) GetDatasetStats(c *gin.Context) {
	stats, err := h.mlDatasetRepo.GetDatasetStats()
	if err != nil {
		h.logger.Error("Failed to get ML dataset stats", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch dataset statistics"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// CreateEntry creates a new ML dataset entry from manual testing.
// POST /api/ml-dataset
func (h *MLDatasetHandler) CreateEntry(c *gin.Context) {
	var req struct {
		MessageText   string `json:"message_text" binding:"required"`
		CategoryID    int    `json:"category_id" binding:"required"`
		CategoryName  string `json:"category_name" binding:"required"`
		Justification string `json:"justification"`
		Provider      string `json:"provider"`
		ModelVersion  string `json:"model_version"`
		Source        string `json:"source"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	entry := &models.MLDatasetEntry{
		MessageText:   req.MessageText,
		CategoryID:    req.CategoryID,
		CategoryName:  req.CategoryName,
		Justification: req.Justification,
		Provider:      req.Provider,
		ModelVersion:  req.ModelVersion,
		AnnotatedAt:   time.Now(),
		IsValidated:   false,
		Source:        req.Source,
	}

	if err := h.mlDatasetRepo.SaveEntry(entry); err != nil {
		h.logger.Error("Failed to save ML dataset entry", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save entry"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Entry created successfully",
		"entry_id": entry.ID,
	})
}

// ExportDataset exports the ML dataset in JSON format.
// GET /api/ml-dataset/export
func (h *MLDatasetHandler) ExportDataset(c *gin.Context) {
	// Optional: filter by validated status
	onlyValidated := c.Query("validated") == "true"

	var entries interface{}
	var err error

	if onlyValidated {
		entries, err = h.mlDatasetRepo.GetValidatedEntries()
	} else {
		entries, err = h.mlDatasetRepo.GetAllEntries()
	}

	if err != nil {
		h.logger.Error("Failed to export ML dataset", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to export dataset"})
		return
	}

	// Set headers for file download
	c.Header("Content-Disposition", "attachment; filename=ml_dataset.json")
	c.Header("Content-Type", "application/json")
	c.JSON(http.StatusOK, entries)
}

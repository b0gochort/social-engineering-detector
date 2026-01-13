package handler

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"annotation-service/internal/models"
	"annotation-service/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler handles HTTP requests
type Handler struct {
	annotator *service.Annotator
	logger    *zap.Logger
}

// NewHandler creates a new API handler
func NewHandler(annotator *service.Annotator, logger *zap.Logger) *Handler {
	return &Handler{
		annotator: annotator,
		logger:    logger,
	}
}

// RegisterRoutes registers all API routes
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v1")
	{
		// Annotation endpoints
		api.POST("/annotate/single", h.AnnotateSingle)
		api.POST("/annotate/batch", h.AnnotateBatch)
		api.GET("/annotate/jobs/:id", h.GetJobStatus)

		// Data retrieval
		api.GET("/annotations", h.GetAllAnnotations)
		api.GET("/annotations/category/:id", h.GetAnnotationsByCategory)
		api.GET("/annotations/stats", h.GetStats)

		// Export
		api.GET("/export/csv", h.ExportCSV)
		api.GET("/export/json", h.ExportJSON)
	}

	// Health check
	r.GET("/health", h.HealthCheck)
}

// AnnotateSingle handles single message annotation
func (h *Handler) AnnotateSingle(c *gin.Context) {
	var req models.AnnotationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	annotation, err := h.annotator.AnnotateSingle(c.Request.Context(), req.Text)
	if err != nil {
		h.logger.Error("Failed to annotate", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "annotation failed"})
		return
	}

	c.JSON(http.StatusOK, annotation)
}

// AnnotateBatch handles batch annotation
func (h *Handler) AnnotateBatch(c *gin.Context) {
	var req models.BatchAnnotationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	jobID, err := h.annotator.AnnotateBatch(c.Request.Context(), req.Messages)
	if err != nil {
		h.logger.Error("Failed to start batch job", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start batch job"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"job_id": jobID,
		"status": "pending",
		"message": "Batch annotation started. Check /api/v1/annotate/jobs/" + jobID + " for status",
	})
}

// GetJobStatus returns batch job status
func (h *Handler) GetJobStatus(c *gin.Context) {
	jobID := c.Param("id")

	job, err := h.annotator.GetJobStatus(jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	c.JSON(http.StatusOK, job)
}

// GetAllAnnotations returns all annotations
func (h *Handler) GetAllAnnotations(c *gin.Context) {
	annotations, err := h.annotator.GetAllAnnotations()
	if err != nil {
		h.logger.Error("Failed to get annotations", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get annotations"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"annotations": annotations,
		"total":       len(annotations),
	})
}

// GetAnnotationsByCategory returns annotations by category
func (h *Handler) GetAnnotationsByCategory(c *gin.Context) {
	categoryID, err := strconv.Atoi(c.Param("id"))
	if err != nil || categoryID < 1 || categoryID > 9 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid category ID (must be 1-9)"})
		return
	}

	annotations, err := h.annotator.GetAnnotationsByCategory(categoryID)
	if err != nil {
		h.logger.Error("Failed to get annotations by category", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get annotations"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"annotations": annotations,
		"category_id": categoryID,
		"total":       len(annotations),
	})
}

// GetStats returns annotation statistics
func (h *Handler) GetStats(c *gin.Context) {
	stats, err := h.annotator.GetStats()
	if err != nil {
		h.logger.Error("Failed to get stats", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// ExportCSV exports annotations to CSV
func (h *Handler) ExportCSV(c *gin.Context) {
	annotations, err := h.annotator.GetAllAnnotations()
	if err != nil {
		h.logger.Error("Failed to export CSV", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "export failed"})
		return
	}

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=annotations.csv")

	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	// Write header
	writer.Write([]string{"text", "category_id", "category_name", "justification"})

	// Write data
	for _, ann := range annotations {
		writer.Write([]string{
			ann.Text,
			fmt.Sprintf("%d", ann.Category),
			ann.CategoryName,
			ann.Justification,
		})
	}
}

// ExportJSON exports annotations to JSON
func (h *Handler) ExportJSON(c *gin.Context) {
	annotations, err := h.annotator.GetAllAnnotations()
	if err != nil {
		h.logger.Error("Failed to export JSON", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "export failed"})
		return
	}

	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", "attachment; filename=annotations.json")

	encoder := json.NewEncoder(c.Writer)
	encoder.SetIndent("", "  ")
	encoder.Encode(annotations)
}

// HealthCheck returns service health
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "annotation-service",
		"version": "1.0.0",
	})
}

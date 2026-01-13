package service

import (
	"context"
	"fmt"
	"time"

	"annotation-service/internal/models"
	"annotation-service/internal/repository"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// LLMClient interface for any LLM provider
type LLMClient interface {
	Annotate(ctx context.Context, text string) (*models.AnnotationResponse, error)
	Close() error
	GetModelInfo() map[string]interface{}
}

// Annotator handles annotation business logic
type Annotator struct {
	llmClient LLMClient
	repo      *repository.AnnotationRepository
	logger    *zap.Logger
}

// NewAnnotator creates a new annotator service
func NewAnnotator(
	llmClient LLMClient,
	repo *repository.AnnotationRepository,
	logger *zap.Logger,
) *Annotator {
	return &Annotator{
		llmClient: llmClient,
		repo:      repo,
		logger:    logger,
	}
}

// AnnotateSingle annotates a single message and saves it
func (a *Annotator) AnnotateSingle(ctx context.Context, text string) (*models.Annotation, error) {
	// Call LLM provider
	resp, err := a.llmClient.Annotate(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("llm annotation failed: %w", err)
	}

	// Get provider info
	modelInfo := a.llmClient.GetModelInfo()
	provider := "unknown"
	modelVersion := "unknown"
	if p, ok := modelInfo["provider"].(string); ok {
		provider = p
	}
	if m, ok := modelInfo["model"].(string); ok {
		modelVersion = m
	}

	// Create annotation model
	annotation := &models.Annotation{
		Text:          text,
		Category:      models.ThreatCategory(resp.CategoryID),
		CategoryName:  resp.CategoryName,
		Justification: resp.Justification,
		Confidence:    resp.Confidence,
		AnnotatedAt:   time.Now(),
		Provider:      provider,
		ModelVersion:  modelVersion,
		IsValidated:   false,
	}

	// Save to database
	if err := a.repo.SaveAnnotation(annotation); err != nil {
		return nil, fmt.Errorf("failed to save annotation: %w", err)
	}

	a.logger.Info("Message annotated",
		zap.Int64("id", annotation.ID),
		zap.String("category", annotation.CategoryName))

	return annotation, nil
}

// AnnotateBatch starts async batch annotation
func (a *Annotator) AnnotateBatch(ctx context.Context, messages []models.MessageInput) (string, error) {
	jobID := uuid.New().String()

	job := &models.Job{
		ID:          jobID,
		Status:      "pending",
		TotalCount:  len(messages),
		CreatedAt:   time.Now(),
	}

	if err := a.repo.CreateJob(job); err != nil {
		return "", fmt.Errorf("failed to create job: %w", err)
	}

	// Start async processing
	go a.processBatchJob(jobID, messages)

	return jobID, nil
}

// processBatchJob processes batch annotation job asynchronously
func (a *Annotator) processBatchJob(jobID string, messages []models.MessageInput) {
	ctx := context.Background()

	job, err := a.repo.GetJob(jobID)
	if err != nil {
		a.logger.Error("Failed to get job", zap.String("job_id", jobID), zap.Error(err))
		return
	}

	job.Status = "processing"
	a.repo.UpdateJob(job)

	for i, msg := range messages {
		annotation, err := a.llmClient.Annotate(ctx, msg.Text)
		if err != nil {
			a.logger.Error("Failed to annotate message in batch",
				zap.String("job_id", jobID),
				zap.Int("index", i),
				zap.Error(err))
			job.FailedCount++
		} else {
			// Get provider info
			modelInfo := a.llmClient.GetModelInfo()
			provider := "unknown"
			modelVersion := "unknown"
			if p, ok := modelInfo["provider"].(string); ok {
				provider = p
			}
			if m, ok := modelInfo["model"].(string); ok {
				modelVersion = m
			}

			// Save annotation
			ann := &models.Annotation{
				MessageID:     msg.ID,
				Text:          msg.Text,
				Category:      models.ThreatCategory(annotation.CategoryID),
				CategoryName:  annotation.CategoryName,
				Justification: annotation.Justification,
				Confidence:    annotation.Confidence,
				AnnotatedAt:   time.Now(),
				Provider:      provider,
				ModelVersion:  modelVersion,
				IsValidated:   false,
			}

			if err := a.repo.SaveAnnotation(ann); err != nil {
				a.logger.Error("Failed to save annotation", zap.Error(err))
				job.FailedCount++
			} else {
				job.ProcessedCount++
			}
		}

		// Update job progress
		a.repo.UpdateJob(job)

		// Rate limiting: small delay between requests
		time.Sleep(100 * time.Millisecond)
	}

	// Mark job as completed
	job.Status = "completed"
	completedAt := time.Now()
	job.CompletedAt = &completedAt
	a.repo.UpdateJob(job)

	a.logger.Info("Batch job completed",
		zap.String("job_id", jobID),
		zap.Int("processed", job.ProcessedCount),
		zap.Int("failed", job.FailedCount))
}

// GetJobStatus returns job status
func (a *Annotator) GetJobStatus(jobID string) (*models.Job, error) {
	return a.repo.GetJob(jobID)
}

// GetAllAnnotations returns all annotations
func (a *Annotator) GetAllAnnotations() ([]*models.Annotation, error) {
	return a.repo.GetAllAnnotations()
}

// GetAnnotationsByCategory returns annotations by category
func (a *Annotator) GetAnnotationsByCategory(categoryID int) ([]*models.Annotation, error) {
	return a.repo.GetAnnotationsByCategory(categoryID)
}

// GetStats returns annotation statistics
func (a *Annotator) GetStats() (map[string]interface{}, error) {
	return a.repo.GetStats()
}

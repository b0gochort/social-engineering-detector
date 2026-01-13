package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"annotation-service/internal/models"

	"github.com/google/generative-ai-go/genai"
	"go.uber.org/zap"
	"google.golang.org/api/option"
)

// Client wraps the Gemini API client
type Client struct {
	client       *genai.Client
	model        *genai.GenerativeModel
	logger       *zap.Logger
	modelName    string
	maxRetries   int
	retryDelay   time.Duration
}

// Config for Gemini client
type Config struct {
	APIKey     string
	ModelName  string // Default: "gemini-2.0-flash-exp"
	MaxRetries int
	RetryDelay time.Duration
}

// NewClient creates a new Gemini client
func NewClient(cfg Config, logger *zap.Logger) (*Client, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("gemini API key is required")
	}

	if cfg.ModelName == "" {
		cfg.ModelName = "gemini-2.0-flash-exp" // Fast and free
	}

	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}

	if cfg.RetryDelay == 0 {
		cfg.RetryDelay = 2 * time.Second
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(cfg.APIKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini client: %w", err)
	}

	model := client.GenerativeModel(cfg.ModelName)

	// Set system instruction (from your llm.py)
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(SystemInstruction)},
	}

	// Set response to JSON format
	model.ResponseMIMEType = "application/json"

	// Set generation config
	model.GenerationConfig = genai.GenerationConfig{
		Temperature:     genai.Ptr[float32](0.3), // Lower for consistent classification
		TopP:            genai.Ptr[float32](0.9),
		TopK:            genai.Ptr[int32](40),
		MaxOutputTokens: genai.Ptr[int32](500),
	}

	logger.Info("Gemini client initialized",
		zap.String("model", cfg.ModelName),
		zap.Int("max_retries", cfg.MaxRetries))

	return &Client{
		client:     client,
		model:      model,
		logger:     logger,
		modelName:  cfg.ModelName,
		maxRetries: cfg.MaxRetries,
		retryDelay: cfg.RetryDelay,
	}, nil
}

// Close closes the Gemini client
func (c *Client) Close() error {
	return c.client.Close()
}

// Annotate classifies a single message
func (c *Client) Annotate(ctx context.Context, text string) (*models.AnnotationResponse, error) {
	prompt := BuildPrompt(text)

	var lastErr error
	for attempt := 0; attempt < c.maxRetries; attempt++ {
		if attempt > 0 {
			c.logger.Warn("Retrying Gemini request",
				zap.Int("attempt", attempt+1),
				zap.Int("max_retries", c.maxRetries))
			time.Sleep(c.retryDelay)
		}

		resp, err := c.model.GenerateContent(ctx, genai.Text(prompt))
		if err != nil {
			lastErr = fmt.Errorf("gemini API error: %w", err)
			c.logger.Error("Gemini API error", zap.Error(err), zap.Int("attempt", attempt+1))
			continue
		}

		// Parse response
		if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
			lastErr = fmt.Errorf("empty response from gemini")
			c.logger.Error("Empty response from Gemini", zap.Int("attempt", attempt+1))
			continue
		}

		// Extract text from response
		textPart, ok := resp.Candidates[0].Content.Parts[0].(genai.Text)
		if !ok {
			lastErr = fmt.Errorf("unexpected response type from gemini")
			c.logger.Error("Unexpected response type", zap.Int("attempt", attempt+1))
			continue
		}

		// Parse JSON - strip markdown code blocks if present
		cleanJSON := strings.TrimSpace(string(textPart))
		cleanJSON = strings.TrimPrefix(cleanJSON, "```json")
		cleanJSON = strings.TrimPrefix(cleanJSON, "```")
		cleanJSON = strings.TrimSuffix(cleanJSON, "```")
		cleanJSON = strings.TrimSpace(cleanJSON)

		var result models.AnnotationResponse
		if err := json.Unmarshal([]byte(cleanJSON), &result); err != nil {
			lastErr = fmt.Errorf("failed to parse gemini response: %w", err)
			c.logger.Error("Failed to parse JSON response",
				zap.Error(err),
				zap.String("original_response", string(textPart)),
				zap.String("cleaned_response", cleanJSON),
				zap.Int("attempt", attempt+1))
			continue
		}

		// Validate category ID
		if result.CategoryID < 1 || result.CategoryID > 9 {
			lastErr = fmt.Errorf("invalid category ID: %d", result.CategoryID)
			c.logger.Error("Invalid category ID",
				zap.Int("category_id", result.CategoryID),
				zap.Int("attempt", attempt+1))
			continue
		}

		c.logger.Debug("Successfully annotated message",
			zap.String("category", result.CategoryName),
			zap.Int("category_id", result.CategoryID),
			zap.Int("attempt", attempt+1))

		return &result, nil
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", c.maxRetries, lastErr)
}

// AnnotateBatch classifies multiple messages
func (c *Client) AnnotateBatch(ctx context.Context, messages []string) ([]*models.AnnotationResponse, error) {
	results := make([]*models.AnnotationResponse, len(messages))
	errors := make([]error, len(messages))

	// Simple sequential processing for now
	// TODO: Add concurrent processing with rate limiting
	for i, text := range messages {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("batch annotation cancelled: %w", ctx.Err())
		default:
		}

		result, err := c.Annotate(ctx, text)
		if err != nil {
			c.logger.Error("Failed to annotate message in batch",
				zap.Int("index", i),
				zap.Error(err))
			errors[i] = err
			continue
		}

		results[i] = result
	}

	// Check if any successful
	successCount := 0
	for _, err := range errors {
		if err == nil {
			successCount++
		}
	}

	if successCount == 0 {
		return nil, fmt.Errorf("all batch annotations failed")
	}

	c.logger.Info("Batch annotation completed",
		zap.Int("total", len(messages)),
		zap.Int("successful", successCount),
		zap.Int("failed", len(messages)-successCount))

	return results, nil
}

// GetStats returns usage statistics (for rate limiting)
type Stats struct {
	TotalRequests   int64
	FailedRequests  int64
	AverageLatency  time.Duration
}

// GetModelInfo returns model information
func (c *Client) GetModelInfo() map[string]interface{} {
	return map[string]interface{}{
		"provider":     "gemini",
		"model":        c.modelName,
		"max_retries":  c.maxRetries,
		"retry_delay":  c.retryDelay.String(),
	}
}

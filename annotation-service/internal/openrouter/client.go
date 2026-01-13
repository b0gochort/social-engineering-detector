package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"annotation-service/internal/gemini"
	"annotation-service/internal/models"

	"go.uber.org/zap"
)

// Client represents an OpenRouter API client.
type Client struct {
	apiKey     string
	baseURL    string
	modelName  string
	httpClient *http.Client
	logger     *zap.Logger
	maxRetries int
	retryDelay time.Duration
}

// Config holds configuration for OpenRouter client.
type Config struct {
	APIKey     string
	ModelName  string // e.g., "meta-llama/llama-3.2-3b-instruct:free"
	MaxRetries int
	RetryDelay time.Duration
}

// openRouterRequest represents the request structure for OpenRouter API.
type openRouterRequest struct {
	Model       string                   `json:"model"`
	Messages    []openRouterMessage      `json:"messages"`
	Temperature float64                  `json:"temperature,omitempty"`
	MaxTokens   int                      `json:"max_tokens,omitempty"`
}

type openRouterMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// openRouterResponse represents the response structure from OpenRouter API.
type openRouterResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// NewClient creates a new OpenRouter client.
func NewClient(cfg Config, logger *zap.Logger) (*Client, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("openrouter API key is required")
	}

	if cfg.ModelName == "" {
		cfg.ModelName = "meta-llama/llama-3.2-3b-instruct:free" // Free model
	}

	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 2
	}

	if cfg.RetryDelay == 0 {
		cfg.RetryDelay = 2 * time.Second
	}

	client := &Client{
		apiKey:     cfg.APIKey,
		baseURL:    "https://openrouter.ai/api/v1",
		modelName:  cfg.ModelName,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		logger:     logger,
		maxRetries: cfg.MaxRetries,
		retryDelay: cfg.RetryDelay,
	}

	logger.Info("OpenRouter client initialized",
		zap.String("model", cfg.ModelName),
		zap.Int("max_retries", cfg.MaxRetries))

	return client, nil
}

// Annotate sends a text to OpenRouter for annotation.
func (c *Client) Annotate(ctx context.Context, text string) (*models.AnnotationResponse, error) {
	var lastErr error

	for attempt := 1; attempt <= c.maxRetries; attempt++ {
		result, err := c.annotateOnce(ctx, text, attempt)
		if err == nil {
			return result, nil
		}

		lastErr = err
		c.logger.Warn("OpenRouter API attempt failed",
			zap.Int("attempt", attempt),
			zap.Int("max_retries", c.maxRetries),
			zap.Error(err))

		// Don't retry if context is cancelled
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		// Wait before retry (except on last attempt)
		if attempt < c.maxRetries {
			select {
			case <-time.After(c.retryDelay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", c.maxRetries, lastErr)
}

func (c *Client) annotateOnce(ctx context.Context, text string, attempt int) (*models.AnnotationResponse, error) {
	// Build the prompt using the same system instruction as Gemini
	prompt := gemini.BuildPrompt(text)

	reqBody := openRouterRequest{
		Model: c.modelName,
		Messages: []openRouterMessage{
			{
				Role:    "system",
				Content: gemini.SystemInstruction,
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.3,
		MaxTokens:   500,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://github.com/social-engineering-detector")
	req.Header.Set("X-Title", "Social Engineering Detector")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("OpenRouter API error", zap.Error(err), zap.Int("attempt", attempt))
		return nil, fmt.Errorf("openrouter API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		c.logger.Error("OpenRouter API error",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(body)),
			zap.Int("attempt", attempt))
		return nil, fmt.Errorf("openrouter API returned status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp openRouterResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Check for API error in response
	if apiResp.Error != nil {
		return nil, fmt.Errorf("openrouter API error: %s", apiResp.Error.Message)
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in openrouter response")
	}

	responseText := apiResp.Choices[0].Message.Content

	// Parse JSON - strip markdown code blocks if present
	cleanJSON := strings.TrimSpace(responseText)
	cleanJSON = strings.TrimPrefix(cleanJSON, "```json")
	cleanJSON = strings.TrimPrefix(cleanJSON, "```")
	cleanJSON = strings.TrimSuffix(cleanJSON, "```")
	cleanJSON = strings.TrimSpace(cleanJSON)

	var result models.AnnotationResponse
	if err := json.Unmarshal([]byte(cleanJSON), &result); err != nil {
		c.logger.Error("Failed to parse JSON response",
			zap.Error(err),
			zap.String("original_response", responseText),
			zap.String("cleaned_response", cleanJSON),
			zap.Int("attempt", attempt))
		return nil, fmt.Errorf("failed to parse openrouter response: %w", err)
	}

	// Validate category ID
	if result.CategoryID < 1 || result.CategoryID > 9 {
		c.logger.Error("Invalid category ID",
			zap.Int("category_id", result.CategoryID),
			zap.Int("attempt", attempt))
		return nil, fmt.Errorf("invalid category ID: %d", result.CategoryID)
	}

	// Add provider metadata
	result.Provider = "openrouter"
	result.ModelVersion = c.modelName
	result.AnnotatedAt = time.Now()

	c.logger.Debug("Successfully annotated message with OpenRouter",
		zap.String("category", result.CategoryName),
		zap.Int("category_id", result.CategoryID),
		zap.Int("attempt", attempt))

	return &result, nil
}

// Close closes the client and releases resources.
func (c *Client) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}

// GetModelInfo returns information about the model being used.
func (c *Client) GetModelInfo() map[string]interface{} {
	return map[string]interface{}{
		"provider": "openrouter",
		"model":    c.modelName,
		"base_url": c.baseURL,
	}
}

// cleanMarkdown removes markdown code blocks if present.
func cleanMarkdown(text string) string {
	text = strings.TrimSpace(text)

	// Remove markdown code blocks
	if strings.HasPrefix(text, "```json") {
		text = strings.TrimPrefix(text, "```json")
		text = strings.TrimSuffix(text, "```")
		text = strings.TrimSpace(text)
	} else if strings.HasPrefix(text, "```") {
		text = strings.TrimPrefix(text, "```")
		text = strings.TrimSuffix(text, "```")
		text = strings.TrimSpace(text)
	}

	return text
}

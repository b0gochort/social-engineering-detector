package groq

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

// Client wraps the Groq API client
type Client struct {
	apiKey     string
	baseURL    string
	modelName  string
	httpClient *http.Client
	logger     *zap.Logger
	maxRetries int
	retryDelay time.Duration
}

// Config for Groq client
type Config struct {
	APIKey     string
	ModelName  string // Default: "llama-3.3-70b-versatile"
	MaxRetries int
	RetryDelay time.Duration
}

// groqRequest represents the request to Groq API
type groqRequest struct {
	Model    string          `json:"model"`
	Messages []groqMessage   `json:"messages"`
	Stream   bool            `json:"stream"`
	Temperature float32      `json:"temperature,omitempty"`
}

type groqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// groqResponse represents the response from Groq API
type groqResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Index   int `json:"index"`
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
}

// NewClient creates a new Groq client
func NewClient(cfg Config, logger *zap.Logger) (*Client, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("groq API key is required")
	}

	if cfg.ModelName == "" {
		cfg.ModelName = "llama-3.3-70b-versatile" // Fast and accurate
	}

	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}

	if cfg.RetryDelay == 0 {
		cfg.RetryDelay = 2 * time.Second
	}

	logger.Info("Groq client initialized",
		zap.String("model", cfg.ModelName),
		zap.Int("max_retries", cfg.MaxRetries))

	return &Client{
		apiKey:     cfg.APIKey,
		baseURL:    "https://api.groq.com/openai/v1",
		modelName:  cfg.ModelName,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		logger:     logger,
		maxRetries: cfg.MaxRetries,
		retryDelay: cfg.RetryDelay,
	}, nil
}

// Close closes the Groq client
func (c *Client) Close() error {
	return nil
}

// Annotate classifies a single message
func (c *Client) Annotate(ctx context.Context, text string) (*models.AnnotationResponse, error) {
	prompt := gemini.BuildPrompt(text)

	var lastErr error
	for attempt := 0; attempt < c.maxRetries; attempt++ {
		if attempt > 0 {
			c.logger.Warn("Retrying Groq request",
				zap.Int("attempt", attempt+1),
				zap.Int("max_retries", c.maxRetries))
			time.Sleep(c.retryDelay)
		}

		reqBody := groqRequest{
			Model: c.modelName,
			Messages: []groqMessage{
				{
					Role:    "system",
					Content: gemini.SystemInstruction,
				},
				{
					Role:    "user",
					Content: prompt,
				},
			},
			Stream:      false,
			Temperature: 0.3,
		}

		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			lastErr = fmt.Errorf("failed to marshal request: %w", err)
			continue
		}

		req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewBuffer(jsonData))
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			continue
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.apiKey)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("groq API error: %w", err)
			c.logger.Error("Groq API error", zap.Error(err), zap.Int("attempt", attempt+1))
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("failed to read response: %w", err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("groq API returned status %d: %s", resp.StatusCode, string(body))
			c.logger.Error("Groq API error",
				zap.Int("status", resp.StatusCode),
				zap.String("body", string(body)),
				zap.Int("attempt", attempt+1))
			continue
		}

		var groqResp groqResponse
		if err := json.Unmarshal(body, &groqResp); err != nil {
			lastErr = fmt.Errorf("failed to parse response: %w", err)
			c.logger.Error("Failed to parse JSON response",
				zap.Error(err),
				zap.String("body", string(body)),
				zap.Int("attempt", attempt+1))
			continue
		}

		if len(groqResp.Choices) == 0 {
			lastErr = fmt.Errorf("empty response from groq")
			c.logger.Error("Empty response from Groq", zap.Int("attempt", attempt+1))
			continue
		}

		content := groqResp.Choices[0].Message.Content

		// Parse JSON - strip markdown code blocks if present
		cleanJSON := strings.TrimSpace(content)
		cleanJSON = strings.TrimPrefix(cleanJSON, "```json")
		cleanJSON = strings.TrimPrefix(cleanJSON, "```")
		cleanJSON = strings.TrimSuffix(cleanJSON, "```")
		cleanJSON = strings.TrimSpace(cleanJSON)

		var result models.AnnotationResponse
		if err := json.Unmarshal([]byte(cleanJSON), &result); err != nil {
			lastErr = fmt.Errorf("failed to parse groq response: %w", err)
			c.logger.Error("Failed to parse JSON response",
				zap.Error(err),
				zap.String("original_response", content),
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

		c.logger.Debug("Successfully annotated message with Groq",
			zap.String("category", result.CategoryName),
			zap.Int("category_id", result.CategoryID),
			zap.Int("attempt", attempt+1))

		return &result, nil
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", c.maxRetries, lastErr)
}

// GetModelInfo returns model information
func (c *Client) GetModelInfo() map[string]interface{} {
	return map[string]interface{}{
		"provider":    "groq",
		"model":       c.modelName,
		"max_retries": c.maxRetries,
		"retry_delay": c.retryDelay.String(),
	}
}

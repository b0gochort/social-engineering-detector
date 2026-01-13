package ml_client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is a client for the ML Service API
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// ClassifyRequest represents a single message classification request
type ClassifyRequest struct {
	Text string `json:"text"`
}

// BatchClassifyRequest represents a batch classification request
type BatchClassifyRequest struct {
	Messages []BatchMessage `json:"messages"`
}

// BatchMessage represents a message in batch request
type BatchMessage struct {
	ID   int64  `json:"id"`
	Text string `json:"text"`
}

// ModelPrediction represents prediction from a single model
type ModelPrediction struct {
	Category   string  `json:"category"`
	CategoryID int     `json:"category_id"`
	Confidence float64 `json:"confidence"`
}

// ClassifyResponse represents the classification result
type ClassifyResponse struct {
	Text              string           `json:"text"`
	Category          string           `json:"category"`
	CategoryID        int              `json:"category_id"`
	Confidence        float64          `json:"confidence"`
	IsAttack          bool             `json:"is_attack"`
	ProcessingTimeMs  float64          `json:"processing_time_ms,omitempty"`
	// Legacy dual model fields for backwards compatibility
	V2Prediction      *ModelPrediction `json:"v2_prediction,omitempty"`
	V4Prediction      *ModelPrediction `json:"v4_prediction,omitempty"`
	ModelsAgree       *bool            `json:"models_agree,omitempty"`
	PrimaryCategory   string           `json:"primary_category,omitempty"`
	PrimaryCategoryID int              `json:"primary_category_id,omitempty"`
	PrimaryConfidence float64          `json:"primary_confidence,omitempty"`
}

// BatchClassifyResponse represents batch classification results
type BatchClassifyResponse struct {
	Results          []BatchResult `json:"results"`
	Total            int           `json:"total"`
	ProcessingTimeMs float64       `json:"processing_time_ms"`
}

// BatchResult represents a single result in batch response
type BatchResult struct {
	ID                int64            `json:"id"`
	Text              string           `json:"text"`
	V2Prediction      *ModelPrediction `json:"v2_prediction"`
	V4Prediction      *ModelPrediction `json:"v4_prediction"`
	ModelsAgree       *bool            `json:"models_agree"`
	PrimaryCategory   string           `json:"primary_category"`
	PrimaryCategoryID int              `json:"primary_category_id"`
	PrimaryConfidence float64          `json:"primary_confidence"`
	IsAttack          bool             `json:"is_attack"`
}

// ModelInfo represents model information
type ModelInfo struct {
	ServiceName string                 `json:"service_name"`
	Version     string                 `json:"version"`
	Models      map[string]interface{} `json:"models"`
	NumLabels   int                    `json:"num_labels"`
	Device      string                 `json:"device"`
	MaxLength   int                    `json:"max_length"`
}

// HealthResponse represents health check response
type HealthResponse struct {
	Status        string `json:"status"`
	V2ModelLoaded bool   `json:"v2_model_loaded"`
	V4ModelLoaded bool   `json:"v4_model_loaded"`
	Device        string `json:"device"`
	Message       string `json:"message"`
}

// NewClient creates a new ML Service client
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ClassifySingle classifies a single message with both models
func (c *Client) ClassifySingle(ctx context.Context, text string) (*ClassifyResponse, error) {
	reqBody := ClassifyRequest{
		Text: text,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/v1/classify/single", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ML service returned status %d: %s", resp.StatusCode, string(body))
	}

	var result ClassifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// ClassifyBatch classifies multiple messages in batch with both models
func (c *Client) ClassifyBatch(ctx context.Context, messages []BatchMessage) (*BatchClassifyResponse, error) {
	reqBody := BatchClassifyRequest{
		Messages: messages,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/v1/classify/batch", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ML service returned status %d: %s", resp.StatusCode, string(body))
	}

	var result BatchClassifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// HealthCheck checks if the ML service is healthy
func (c *Client) HealthCheck(ctx context.Context) (*HealthResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/v1/health", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ML service returned status %d: %s", resp.StatusCode, string(body))
	}

	var result HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetModelInfo retrieves information about the loaded models
func (c *Client) GetModelInfo(ctx context.Context) (*ModelInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/v1/model/info", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ML service returned status %d: %s", resp.StatusCode, string(body))
	}

	var result ModelInfo
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

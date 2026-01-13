package annotation_client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Client represents the Annotation Service client
type Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// AnnotateRequest represents the request to annotate a single message
type AnnotateRequest struct {
	Text string `json:"text"`
}

// AnnotationResponse represents the response from annotation service
type AnnotationResponse struct {
	ID             int64     `json:"id"`
	Text           string    `json:"text"`
	CategoryID     int       `json:"category_id"`
	CategoryName   string    `json:"category_name"`
	Justification  string    `json:"justification"`
	AnnotatedAt    time.Time `json:"annotated_at"`
	Provider       string    `json:"provider"`
	ModelVersion   string    `json:"model_version"`
	IsValidated    bool      `json:"is_validated"`
}

// NewClient creates a new Annotation Service client
func NewClient(baseURL string, logger *zap.Logger) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second, // Gemini API can be slow
		},
		logger: logger,
	}
}

// AnnotateSingle sends a single message for annotation
func (c *Client) AnnotateSingle(ctx context.Context, text string) (*AnnotationResponse, error) {
	reqBody := AnnotateRequest{
		Text: text,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/v1/annotate/single", bytes.NewBuffer(jsonData))
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
		return nil, fmt.Errorf("annotation service returned status %d", resp.StatusCode)
	}

	var annotationResp AnnotationResponse
	if err := json.NewDecoder(resp.Body).Decode(&annotationResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &annotationResp, nil
}

// Ping checks if the annotation service is available
func (c *Client) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send health check request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("annotation service health check failed with status %d", resp.StatusCode)
	}

	return nil
}

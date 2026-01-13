package collector_client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Message represents a simplified structure of a message from the collector.
// This should match the structure returned by the collector's /collect endpoint.
type Message struct {
	ID             int64     `json:"id"`
	ChatID         int64     `json:"chat_id"`
	SenderUsername string    `json:"sender_username"`
	Timestamp      time.Time `json:"timestamp"`
	Text           string    `json:"text"`
	Type           string    `json:"type"`           // "message", "post", "comment"
	Source         string    `json:"source"`         // "telegram" or "vk"
}

// Chat represents a simplified structure of a chat from the collector.
type Chat struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	IsGroup bool   `json:"is_group"`
	Type    string `json:"type"`    // "user", "chat", "channel", "group"
	Source  string `json:"source"`  // "telegram" or "vk"
}

// Client for interacting with the Telegram collector service.
type Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// NewClient creates a new Collector API client.
func NewClient(baseURL string, logger *zap.Logger) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second, // Configure appropriate timeout
		},
		logger: logger,
	}
}

// GetMessages fetches messages from the collector service.
func (c *Client) GetMessages(ctx context.Context, chatID int64, lastCollectedMessageID int64) ([]Message, error) {
	url := fmt.Sprintf("%s/telegram/collect?chat_id=%d&last_collected_message_id=%d", c.baseURL, chatID, lastCollectedMessageID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		c.logger.Error("Failed to create request to collector", zap.Error(err))
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to make request to collector", zap.Error(err))
		return nil, fmt.Errorf("failed to make request to collector: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Error("Collector returned non-OK status", zap.Int("status", resp.StatusCode))
		return nil, fmt.Errorf("collector returned status: %d", resp.StatusCode)
	}

	var response struct {
		Messages []Message `json:"messages"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		c.logger.Error("Failed to decode collector response", zap.Error(err))
		return nil, fmt.Errorf("failed to decode collector response: %w", err)
	}

	c.logger.Info("Successfully fetched messages from collector", zap.Int("count", len(response.Messages)))
	return response.Messages, nil
}

// GetChats fetches all available chats from the collector service.
func (c *Client) GetChats(ctx context.Context) ([]Chat, error) {
	url := fmt.Sprintf("%s/telegram/chats", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		c.logger.Error("Failed to create request to collector for chats", zap.Error(err))
		return nil, fmt.Errorf("failed to create request for chats: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to make request to collector for chats", zap.Error(err))
		return nil, fmt.Errorf("failed to make request to collector for chats: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Error("Collector returned non-OK status for chats", zap.Int("status", resp.StatusCode))
		return nil, fmt.Errorf("collector returned status for chats: %d", resp.StatusCode)
	}

	var response struct {
		Chats []Chat `json:"chats"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		c.logger.Error("Failed to decode collector chats response", zap.Error(err))
		return nil, fmt.Errorf("failed to decode collector chats response: %w", err)
	}

	c.logger.Info("Successfully fetched chats from collector", zap.Int("count", len(response.Chats)))
	return response.Chats, nil
}

// ========== VK Methods ==========

// VKAuthURLResponse represents response from /vk/auth/url
type VKAuthURLResponse struct {
	AuthURL      string `json:"auth_url"`
	Instructions string `json:"instructions"`
}

// GetVKAuthURL fetches VK OAuth authorization URL from the collector
func (c *Client) GetVKAuthURL(ctx context.Context) (*VKAuthURLResponse, error) {
	url := fmt.Sprintf("%s/vk/auth/url", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		c.logger.Error("Failed to create request for VK auth URL", zap.Error(err))
		return nil, fmt.Errorf("failed to create request for VK auth URL: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to make request for VK auth URL", zap.Error(err))
		return nil, fmt.Errorf("failed to make request for VK auth URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Error("Collector returned non-OK status for VK auth URL", zap.Int("status", resp.StatusCode))
		return nil, fmt.Errorf("collector returned status for VK auth URL: %d", resp.StatusCode)
	}

	var response VKAuthURLResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		c.logger.Error("Failed to decode VK auth URL response", zap.Error(err))
		return nil, fmt.Errorf("failed to decode VK auth URL response: %w", err)
	}

	return &response, nil
}

// GetVKConversations fetches all available VK conversations from the collector service.
func (c *Client) GetVKConversations(ctx context.Context) ([]Chat, error) {
	url := fmt.Sprintf("%s/vk/conversations", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		c.logger.Error("Failed to create request to collector for VK conversations", zap.Error(err))
		return nil, fmt.Errorf("failed to create request for VK conversations: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to make request to collector for VK conversations", zap.Error(err))
		return nil, fmt.Errorf("failed to make request to collector for VK conversations: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Error("Collector returned non-OK status for VK conversations", zap.Int("status", resp.StatusCode))
		return nil, fmt.Errorf("collector returned status for VK conversations: %d", resp.StatusCode)
	}

	var response struct {
		Conversations []struct {
			ID      int64  `json:"id"`
			Name    string `json:"name"`
			IsGroup bool   `json:"is_group"`
			Type    string `json:"type"`
		} `json:"conversations"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		c.logger.Error("Failed to decode VK conversations response", zap.Error(err))
		return nil, fmt.Errorf("failed to decode VK conversations response: %w", err)
	}

	// Convert to Chat structs with source="vk"
	chats := make([]Chat, len(response.Conversations))
	for i, conv := range response.Conversations {
		chats[i] = Chat{
			ID:      conv.ID,
			Name:    conv.Name,
			IsGroup: conv.IsGroup,
			Type:    conv.Type,
			Source:  "vk",
		}
	}

	c.logger.Info("Successfully fetched VK conversations from collector", zap.Int("count", len(chats)))
	return chats, nil
}

// GetVKMessages fetches messages from a specific VK conversation.
func (c *Client) GetVKMessages(ctx context.Context, peerID int64, lastMessageID int64) ([]Message, error) {
	url := fmt.Sprintf("%s/vk/messages/collect?peer_id=%d&last_message_id=%d", c.baseURL, peerID, lastMessageID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		c.logger.Error("Failed to create request to collector for VK messages", zap.Error(err))
		return nil, fmt.Errorf("failed to create request for VK messages: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to make request to collector for VK messages", zap.Error(err))
		return nil, fmt.Errorf("failed to make request to collector for VK messages: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Error("Collector returned non-OK status for VK messages", zap.Int("status", resp.StatusCode))
		return nil, fmt.Errorf("collector returned status for VK messages: %d", resp.StatusCode)
	}

	var response struct {
		Messages []struct {
			ID             int64     `json:"id"`
			ChatID         int64     `json:"chat_id"`
			SenderUsername string    `json:"sender_username"`
			Timestamp      time.Time `json:"timestamp"`
			Text           string    `json:"text"`
			Type           string    `json:"type"`
		} `json:"messages"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		c.logger.Error("Failed to decode VK messages response", zap.Error(err))
		return nil, fmt.Errorf("failed to decode VK messages response: %w", err)
	}

	// Convert to Message structs with source="vk"
	messages := make([]Message, len(response.Messages))
	for i, msg := range response.Messages {
		messages[i] = Message{
			ID:             msg.ID,
			ChatID:         msg.ChatID,
			SenderUsername: msg.SenderUsername,
			Timestamp:      msg.Timestamp,
			Text:           msg.Text,
			Type:           msg.Type,
			Source:         "vk",
		}
	}

	c.logger.Info("Successfully fetched VK messages from collector", zap.Int("count", len(messages)))
	return messages, nil
}
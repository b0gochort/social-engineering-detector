package vk

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"go.uber.org/zap"

	"collector/pkg/config"
)

// CollectorMessage represents a VK post, comment, or private message to be sent to the backend.
type CollectorMessage struct {
	ID             int64     `json:"id"`
	ChatID         int64     `json:"chat_id"`         // Conversation/Peer ID (for messages) or Group ID (for posts/comments)
	GroupID        int64     `json:"group_id"`        // VK group/community ID (deprecated, use ChatID)
	SenderUsername string    `json:"sender_username"` // Author name
	Timestamp      time.Time `json:"timestamp"`
	Text           string    `json:"text"`
	Type           string    `json:"type"`            // "post", "comment", or "message"
	PostID         *int64    `json:"post_id,omitempty"` // Parent post ID for comments
}

// GroupInfo represents information about a VK group/community.
type GroupInfo struct {
	ID         int64  `json:"id"`
	ScreenName string `json:"screen_name"` // Short name (e.g., "club123" or "public_name")
	Name       string `json:"name"`
	IsActive   bool   `json:"is_active"`
}

// Client encapsulates the VK API client.
type Client struct {
	accessToken string
	apiVersion  string
	logger      *zap.Logger
	httpClient  *http.Client
}

// VK API response structures
type vkResponse struct {
	Response json.RawMessage `json:"response"`
	Error    *vkError        `json:"error"`
}

type vkError struct {
	ErrorCode int    `json:"error_code"`
	ErrorMsg  string `json:"error_msg"`
}

type vkWallGetResponse struct {
	Count int       `json:"count"`
	Items []vkPost  `json:"items"`
}

type vkPost struct {
	ID       int    `json:"id"`
	OwnerID  int64  `json:"owner_id"`
	FromID   int64  `json:"from_id"`
	Date     int64  `json:"date"`
	Text     string `json:"text"`
	Comments *struct {
		Count int `json:"count"`
	} `json:"comments,omitempty"`
}

type vkCommentsGetResponse struct {
	Count int         `json:"count"`
	Items []vkComment `json:"items"`
}

type vkComment struct {
	ID      int    `json:"id"`
	FromID  int64  `json:"from_id"`
	Date    int64  `json:"date"`
	Text    string `json:"text"`
	PostID  int    `json:"post_id,omitempty"`
}

type vkGroupsGetByIdResponse []vkGroup

type vkGroup struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	ScreenName string `json:"screen_name"`
	Type       string `json:"type"`
}

type vkUsersGetResponse []vkUser

type vkUser struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// NewClient creates and initializes a new VK API client.
func NewClient(cfg *config.VKConfig, logger *zap.Logger) (*Client, error) {
	if cfg.AccessToken == "" {
		return nil, fmt.Errorf("VK access token is required")
	}

	return &Client{
		accessToken: cfg.AccessToken,
		apiVersion:  "5.131", // VK API version
		logger:      logger,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// makeAPIRequest performs a VK API request
func (c *Client) makeAPIRequest(ctx context.Context, method string, params url.Values) (json.RawMessage, error) {
	params.Set("access_token", c.accessToken)
	params.Set("v", c.apiVersion)

	apiURL := fmt.Sprintf("https://api.vk.com/method/%s?%s", method, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var vkResp vkResponse
	if err := json.Unmarshal(body, &vkResp); err != nil {
		return nil, fmt.Errorf("failed to parse VK response: %w", err)
	}

	if vkResp.Error != nil {
		return nil, fmt.Errorf("VK API error %d: %s", vkResp.Error.ErrorCode, vkResp.Error.ErrorMsg)
	}

	// VK API rate limit: 3 requests per second
	time.Sleep(350 * time.Millisecond)

	return vkResp.Response, nil
}

// GetGroupInfo fetches information about a VK group by ID or screen name.
func (c *Client) GetGroupInfo(ctx context.Context, groupID string) (*GroupInfo, error) {
	params := url.Values{}
	params.Set("group_id", groupID)

	c.logger.Info("Fetching VK group info", zap.String("group_id", groupID))

	respData, err := c.makeAPIRequest(ctx, "groups.getById", params)
	if err != nil {
		return nil, fmt.Errorf("failed to get group info: %w", err)
	}

	var groups vkGroupsGetByIdResponse
	if err := json.Unmarshal(respData, &groups); err != nil {
		return nil, fmt.Errorf("failed to parse group info response: %w", err)
	}

	if len(groups) == 0 {
		return nil, fmt.Errorf("group not found")
	}

	group := groups[0]
	return &GroupInfo{
		ID:         group.ID,
		ScreenName: group.ScreenName,
		Name:       group.Name,
		IsActive:   true,
	}, nil
}

// GetWallPosts fetches posts from a VK group wall.
// groupID can be numeric ID (e.g., "123456") or screen name (e.g., "apiclub")
// lastPostID is the ID of the last collected post (0 to fetch latest)
func (c *Client) GetWallPosts(ctx context.Context, groupID string, lastPostID int64) ([]CollectorMessage, error) {
	params := url.Values{}

	// Convert group ID to negative owner_id format
	// VK uses negative IDs for groups: group 123 -> owner_id -123
	ownerID := groupID
	if groupID[0] != '-' {
		// Try to get numeric ID
		groupInfo, err := c.GetGroupInfo(ctx, groupID)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve group ID: %w", err)
		}
		ownerID = fmt.Sprintf("-%d", groupInfo.ID)
	}

	params.Set("owner_id", ownerID)
	params.Set("count", "100")
	params.Set("filter", "all")

	c.logger.Info("Fetching VK wall posts", zap.String("owner_id", ownerID), zap.Int64("from_post_id", lastPostID))

	respData, err := c.makeAPIRequest(ctx, "wall.get", params)
	if err != nil {
		return nil, fmt.Errorf("failed to get wall posts: %w", err)
	}

	var wallResp vkWallGetResponse
	if err := json.Unmarshal(respData, &wallResp); err != nil {
		return nil, fmt.Errorf("failed to parse wall response: %w", err)
	}

	var messages []CollectorMessage
	for _, post := range wallResp.Items {
		// Filter posts newer than lastPostID
		if int64(post.ID) > lastPostID {
			authorName, err := c.getUserOrGroupName(ctx, post.FromID)
			if err != nil {
				c.logger.Warn("Failed to get author name", zap.Int64("from_id", post.FromID), zap.Error(err))
				authorName = fmt.Sprintf("ID%d", post.FromID)
			}

			messages = append(messages, CollectorMessage{
				ID:             int64(post.ID),
				GroupID:        post.OwnerID,
				SenderUsername: authorName,
				Timestamp:      time.Unix(post.Date, 0),
				Text:           post.Text,
				Type:           "post",
			})
		}
	}

	c.logger.Info("Fetched VK wall posts", zap.String("owner_id", ownerID), zap.Int("count", len(messages)))
	return messages, nil
}

// GetPostComments fetches comments for a specific post.
func (c *Client) GetPostComments(ctx context.Context, ownerID int64, postID int64, lastCommentID int64) ([]CollectorMessage, error) {
	params := url.Values{}
	params.Set("owner_id", strconv.FormatInt(ownerID, 10))
	params.Set("post_id", strconv.FormatInt(postID, 10))
	params.Set("count", "100")
	params.Set("sort", "asc") // Oldest first

	c.logger.Info("Fetching VK post comments",
		zap.Int64("owner_id", ownerID),
		zap.Int64("post_id", postID),
		zap.Int64("from_comment_id", lastCommentID))

	respData, err := c.makeAPIRequest(ctx, "wall.getComments", params)
	if err != nil {
		return nil, fmt.Errorf("failed to get comments: %w", err)
	}

	var commentsResp vkCommentsGetResponse
	if err := json.Unmarshal(respData, &commentsResp); err != nil {
		return nil, fmt.Errorf("failed to parse comments response: %w", err)
	}

	var messages []CollectorMessage
	for _, comment := range commentsResp.Items {
		// Filter comments newer than lastCommentID
		if int64(comment.ID) > lastCommentID {
			authorName, err := c.getUserOrGroupName(ctx, comment.FromID)
			if err != nil {
				c.logger.Warn("Failed to get commenter name", zap.Int64("from_id", comment.FromID), zap.Error(err))
				authorName = fmt.Sprintf("ID%d", comment.FromID)
			}

			messages = append(messages, CollectorMessage{
				ID:             int64(comment.ID),
				GroupID:        ownerID,
				SenderUsername: authorName,
				Timestamp:      time.Unix(comment.Date, 0),
				Text:           comment.Text,
				Type:           "comment",
				PostID:         &postID,
			})
		}
	}

	c.logger.Info("Fetched VK post comments",
		zap.Int64("post_id", postID),
		zap.Int("count", len(messages)))
	return messages, nil
}

// getUserOrGroupName fetches the name of a user or group by ID.
func (c *Client) getUserOrGroupName(ctx context.Context, id int64) (string, error) {
	if id > 0 {
		// It's a user
		params := url.Values{}
		params.Set("user_ids", strconv.FormatInt(id, 10))

		respData, err := c.makeAPIRequest(ctx, "users.get", params)
		if err != nil {
			return "", err
		}

		var users vkUsersGetResponse
		if err := json.Unmarshal(respData, &users); err != nil {
			return "", fmt.Errorf("failed to parse users response: %w", err)
		}

		if len(users) == 0 {
			return "", fmt.Errorf("user not found")
		}

		return fmt.Sprintf("%s %s", users[0].FirstName, users[0].LastName), nil
	} else if id < 0 {
		// It's a group/community
		params := url.Values{}
		params.Set("group_id", strconv.FormatInt(-id, 10))

		respData, err := c.makeAPIRequest(ctx, "groups.getById", params)
		if err != nil {
			return "", err
		}

		var groups vkGroupsGetByIdResponse
		if err := json.Unmarshal(respData, &groups); err != nil {
			return "", fmt.Errorf("failed to parse groups response: %w", err)
		}

		if len(groups) == 0 {
			return "", fmt.Errorf("group not found")
		}

		return groups[0].Name, nil
	}

	return "Unknown", nil
}

// ConversationInfo represents information about a VK conversation (dialog or chat).
type ConversationInfo struct {
	ID      int64  `json:"id"`       // Peer ID
	Name    string `json:"name"`     // Conversation title or user name
	IsGroup bool   `json:"is_group"` // True for group chats
	Type    string `json:"type"`     // "user", "chat", or "group"
}

// VK Messages API structures
type vkConversationsGetResponse struct {
	Count int                   `json:"count"`
	Items []vkConversationItem  `json:"items"`
	Profiles []vkUser           `json:"profiles"`
	Groups   []vkGroup          `json:"groups"`
}

type vkConversationItem struct {
	Conversation vkConversationData `json:"conversation"`
	LastMessage  *vkMessageData     `json:"last_message,omitempty"`
}

type vkConversationData struct {
	Peer            vkPeerData         `json:"peer"`
	LastMessageID   int                `json:"last_message_id"`
	InRead          int                `json:"in_read"`
	OutRead         int                `json:"out_read"`
	UnreadCount     int                `json:"unread_count"`
	CanWrite        vkCanWrite         `json:"can_write"`
	ChatSettings    *vkChatSettingsData `json:"chat_settings,omitempty"`
}

type vkPeerData struct {
	ID      int64  `json:"id"`
	Type    string `json:"type"` // "user", "chat", "group", "email"
	LocalID int64  `json:"local_id,omitempty"`
}

type vkCanWrite struct {
	Allowed bool `json:"allowed"`
}

type vkChatSettingsData struct {
	Title string `json:"title"`
}

type vkMessageData struct {
	ID                    int    `json:"id"`
	Date                  int64  `json:"date"`
	PeerID                int64  `json:"peer_id"`
	FromID                int64  `json:"from_id"`
	Text                  string `json:"text"`
	ConversationMessageID int    `json:"conversation_message_id"`
}

type vkMessagesHistoryResponse struct {
	Count    int             `json:"count"`
	Items    []vkMessageData `json:"items"`
	Profiles []vkUser        `json:"profiles"`
	Groups   []vkGroup       `json:"groups"`
}

// GetAllConversations fetches all available VK conversations (dialogs and chats).
func (c *Client) GetAllConversations(ctx context.Context) ([]ConversationInfo, error) {
	params := url.Values{}
	params.Set("count", "200")
	params.Set("extended", "1")
	params.Set("filter", "all")

	c.logger.Info("Fetching VK conversations...")

	respData, err := c.makeAPIRequest(ctx, "messages.getConversations", params)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversations: %w", err)
	}

	var convResp vkConversationsGetResponse
	if err := json.Unmarshal(respData, &convResp); err != nil {
		return nil, fmt.Errorf("failed to parse conversations response: %w", err)
	}

	var conversations []ConversationInfo
	for _, item := range convResp.Items {
		conv := ConversationInfo{
			ID:   item.Conversation.Peer.ID,
			Type: item.Conversation.Peer.Type,
		}

		// Determine if it's a group chat
		conv.IsGroup = item.Conversation.Peer.Type == "chat"

		// Set conversation name
		switch item.Conversation.Peer.Type {
		case "user":
			// Find user in profiles
			for _, profile := range convResp.Profiles {
				if profile.ID == item.Conversation.Peer.ID {
					conv.Name = fmt.Sprintf("%s %s", profile.FirstName, profile.LastName)
					break
				}
			}
			if conv.Name == "" {
				conv.Name = fmt.Sprintf("User %d", item.Conversation.Peer.ID)
			}
		case "chat":
			if item.Conversation.ChatSettings != nil {
				conv.Name = item.Conversation.ChatSettings.Title
			} else {
				conv.Name = fmt.Sprintf("Chat %d", item.Conversation.Peer.LocalID)
			}
		case "group":
			// Find group in groups
			groupID := -item.Conversation.Peer.ID // Groups have negative IDs
			for _, group := range convResp.Groups {
				if group.ID == groupID {
					conv.Name = group.Name
					break
				}
			}
			if conv.Name == "" {
				conv.Name = fmt.Sprintf("Group %d", -item.Conversation.Peer.ID)
			}
		default:
			conv.Name = fmt.Sprintf("Conversation %d", item.Conversation.Peer.ID)
		}

		conversations = append(conversations, conv)
	}

	c.logger.Info("Fetched VK conversations", zap.Int("count", len(conversations)))
	return conversations, nil
}

// GetConversationMessages fetches messages from a specific VK conversation.
// peerID is the conversation/peer ID (can be user ID, chat ID, or negative group ID)
// lastMessageID is the ID of the last collected message (0 to fetch latest)
func (c *Client) GetConversationMessages(ctx context.Context, peerID int64, lastMessageID int64) ([]CollectorMessage, error) {
	params := url.Values{}
	params.Set("peer_id", strconv.FormatInt(peerID, 10))
	params.Set("count", "200")
	params.Set("extended", "1")

	c.logger.Info("Fetching VK conversation messages",
		zap.Int64("peer_id", peerID),
		zap.Int64("from_message_id", lastMessageID))

	respData, err := c.makeAPIRequest(ctx, "messages.getHistory", params)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	var msgResp vkMessagesHistoryResponse
	if err := json.Unmarshal(respData, &msgResp); err != nil {
		return nil, fmt.Errorf("failed to parse messages response: %w", err)
	}

	var messages []CollectorMessage
	for _, msg := range msgResp.Items {
		// Filter messages newer than lastMessageID
		if int64(msg.ID) > lastMessageID {
			senderName := ""

			// Resolve sender name
			if msg.FromID > 0 {
				// It's a user
				for _, profile := range msgResp.Profiles {
					if profile.ID == msg.FromID {
						senderName = fmt.Sprintf("%s %s", profile.FirstName, profile.LastName)
						break
					}
				}
				if senderName == "" {
					senderName = fmt.Sprintf("User %d", msg.FromID)
				}
			} else if msg.FromID < 0 {
				// It's a group
				groupID := -msg.FromID
				for _, group := range msgResp.Groups {
					if group.ID == groupID {
						senderName = group.Name
						break
					}
				}
				if senderName == "" {
					senderName = fmt.Sprintf("Group %d", groupID)
				}
			}

			messages = append(messages, CollectorMessage{
				ID:             int64(msg.ID),
				ChatID:         peerID,
				SenderUsername: senderName,
				Timestamp:      time.Unix(msg.Date, 0),
				Text:           msg.Text,
				Type:           "message",
			})
		}
	}

	c.logger.Info("Fetched VK conversation messages",
		zap.Int64("peer_id", peerID),
		zap.Int("count", len(messages)))
	return messages, nil
}

// GenerateOAuthURL generates VK OAuth authorization URL.
func GenerateOAuthURL(appID int, redirectURI string) string {
	params := url.Values{}
	params.Set("client_id", strconv.Itoa(appID))
	params.Set("redirect_uri", redirectURI)
	params.Set("display", "page")
	params.Set("scope", "messages,offline") // messages - access to messages, offline - token doesn't expire
	params.Set("response_type", "token")
	params.Set("v", "5.131")

	return fmt.Sprintf("https://oauth.vk.com/authorize?%s", params.Encode())
}

package collector

import (
	"context"
	"time"

	"go.uber.org/zap"

	"collector/pkg/vk"
)

// VKCollector is responsible for collecting posts and comments from VK groups.
type VKCollector struct {
	vkClient *vk.Client
	logger   *zap.Logger
	interval time.Duration
}

// NewVKCollector creates a new VKCollector instance.
func NewVKCollector(vkClient *vk.Client, logger *zap.Logger, interval time.Duration) *VKCollector {
	return &VKCollector{
		vkClient: vkClient,
		logger:   logger,
		interval: interval,
	}
}

// CollectWallPosts fetches new posts from a VK group wall.
func (c *VKCollector) CollectWallPosts(ctx context.Context, groupID string, lastPostID int64) (interface{}, error) {
	c.logger.Info("Fetching VK wall posts...", zap.String("group_id", groupID))
	messages, err := c.vkClient.GetWallPosts(ctx, groupID, lastPostID)
	if err != nil {
		c.logger.Error("Error fetching VK wall posts", zap.Error(err))
		return nil, err
	}
	c.logger.Info("Fetched VK wall posts.", zap.Int("count", len(messages)))
	return messages, nil
}

// CollectPostComments fetches comments for a specific post.
func (c *VKCollector) CollectPostComments(ctx context.Context, ownerID int64, postID int64, lastCommentID int64) (interface{}, error) {
	c.logger.Info("Fetching VK post comments...",
		zap.Int64("owner_id", ownerID),
		zap.Int64("post_id", postID))
	messages, err := c.vkClient.GetPostComments(ctx, ownerID, postID, lastCommentID)
	if err != nil {
		c.logger.Error("Error fetching VK post comments", zap.Error(err))
		return nil, err
	}
	c.logger.Info("Fetched VK post comments.", zap.Int("count", len(messages)))
	return messages, nil
}

// GetGroupInfo fetches information about a VK group.
func (c *VKCollector) GetGroupInfo(ctx context.Context, groupID string) (interface{}, error) {
	c.logger.Info("Fetching VK group info...", zap.String("group_id", groupID))
	groupInfo, err := c.vkClient.GetGroupInfo(ctx, groupID)
	if err != nil {
		c.logger.Error("Error fetching VK group info", zap.Error(err))
		return nil, err
	}
	c.logger.Info("Fetched VK group info.", zap.String("name", groupInfo.Name))
	return groupInfo, nil
}

// GetAllConversations fetches all VK conversations (dialogs and chats).
func (c *VKCollector) GetAllConversations(ctx context.Context) (interface{}, error) {
	c.logger.Info("Fetching VK conversations...")
	conversations, err := c.vkClient.GetAllConversations(ctx)
	if err != nil {
		c.logger.Error("Error fetching VK conversations", zap.Error(err))
		return nil, err
	}
	c.logger.Info("Fetched VK conversations.", zap.Int("count", len(conversations)))
	return conversations, nil
}

// CollectConversationMessages fetches messages from a specific VK conversation.
func (c *VKCollector) CollectConversationMessages(ctx context.Context, peerID int64, lastMessageID int64) (interface{}, error) {
	c.logger.Info("Fetching VK conversation messages...",
		zap.Int64("peer_id", peerID),
		zap.Int64("from_message_id", lastMessageID))
	messages, err := c.vkClient.GetConversationMessages(ctx, peerID, lastMessageID)
	if err != nil {
		c.logger.Error("Error fetching VK conversation messages", zap.Error(err))
		return nil, err
	}
	c.logger.Info("Fetched VK conversation messages.", zap.Int("count", len(messages)))
	return messages, nil
}

// Run starts the VK collector process (placeholder for future use).
func (c *VKCollector) Run(ctx context.Context) {
	c.logger.Info("Starting VK collector...")
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("VK collector stopped.")
			return
		case <-ticker.C:
			// VK collector will be triggered via API endpoints
			c.logger.Info("VK collector is ready to collect via API.")
		}
	}
}

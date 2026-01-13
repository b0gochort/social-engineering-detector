package telegram

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/tg"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"collector/pkg/config"
)

// CollectorMessage represents a simplified structure of a message to be sent to the backend.
type CollectorMessage struct {
	ID             int64     `json:"id"`
	ChatID         int64     `json:"chat_id"`
	SenderUsername string    `json:"sender_username"`
	Timestamp      time.Time `json:"timestamp"`
	Text           string    `json:"text"`
}

// ChatInfo represents simplified information about a Telegram chat.
type ChatInfo struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	IsGroup bool   `json:"is_group"`
}

// Client encapsulates the Telegram client.
type Client struct {
	*telegram.Client
	Sender        *message.Sender
	AuthCode      chan string   // Channel to receive authentication code
	AuthCompleted chan struct{} // Channel to signal authentication completion
	logger        *zap.Logger

	cachedUsers []tg.UserClass
	cachedChats []tg.ChatClass
}

// NewClient creates and initializes a new Telegram client.
func NewClient(cfg *config.TelegramConfig) (*Client, error) {
	sessionFile := "session.json"

	// Create a new Zap logger
	logger, err := zap.NewDevelopment(zap.IncreaseLevel(zapcore.InfoLevel))
	if err != nil {
		return nil, fmt.Errorf("failed to create zap logger: %w", err)
	}

	client := telegram.NewClient(cfg.APIID, cfg.APIHash, telegram.Options{
		Logger:         logger, // Use zap as the logger
		SessionStorage: &session.FileStorage{Path: sessionFile},
	})

	return &Client{
		Client:        client,
		Sender:        message.NewSender(client.API()),
		AuthCode:      make(chan string),
		AuthCompleted: make(chan struct{}),
		logger:        logger,
	}, nil
}

// Run starts the Telegram client and handles authentication.
func (c *Client) Run(ctx context.Context, phone string) error {
	return c.Client.Run(ctx, func(ctx context.Context) error {
		if err := c.auth(ctx, phone); err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}
		c.logger.Info("Telegram client started and authenticated.")
		close(c.AuthCompleted) // Signal that authentication is complete

		// Keep the client running until the main context is cancelled
		<-ctx.Done()
		return ctx.Err() // Return the context error (e.g., context.Canceled)
	})
}

func (c *Client) auth(ctx context.Context, phone string) error {
	flow := auth.NewFlow(
		auth.Constant(phone, "", auth.CodeAuthenticatorFunc(func(ctx context.Context, sentCode *tg.AuthSentCode) (string, error) {
			c.logger.Info("Waiting for authentication code via API...")
			select {
			case code := <-c.AuthCode:
				return strings.TrimSpace(code), nil
			case <-ctx.Done():
				return "", ctx.Err()
			}
		})),
		auth.SendCodeOptions{},
	)

	return flow.Run(ctx, c.Client.Auth())
}

// GetAllChatsInfo fetches information about all available chats.
func (c *Client) GetAllChatsInfo(ctx context.Context) ([]ChatInfo, error) {
	var chatsInfo []ChatInfo

	dialogs, err := c.API().MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
		ExcludePinned: false,
		OffsetDate:    0,
		OffsetID:      0,
		OffsetPeer:    &tg.InputPeerEmpty{},
		Limit:         100, // Fetch a reasonable number of dialogs
		Hash:          0,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get dialogs: %w", err)
	}

	var dialogsList []tg.DialogClass
	var users []tg.UserClass
	var chats []tg.ChatClass

	switch d := dialogs.(type) {
	case *tg.MessagesDialogs:
		dialogsList = d.Dialogs
		users = d.Users
		chats = d.Chats
	case *tg.MessagesDialogsSlice:
		dialogsList = d.Dialogs
		users = d.Users
		chats = d.Chats
	default:
		c.logger.Warn("Unknown MessagesDialogsClass type", zap.String("type", fmt.Sprintf("%T", dialogs)))
		return nil, fmt.Errorf("unknown dialogs type: %T", dialogs)
	}

	c.cachedUsers = users
	c.cachedChats = chats

	for _, dialog := range dialogsList {
		dlg, ok := dialog.(*tg.Dialog)
		if !ok {
			continue
		}

		peerID, peerType := getPeerIDAndType(dlg.Peer)
		chatName := ""
		isGroup := false

		switch peerType {
		case "user":
			for _, u := range users {
				if user, ok := u.(*tg.User); ok && user.ID == peerID {
					chatName = user.FirstName + " " + user.LastName
					break
				}
			}
		case "chat":
			for _, ch := range chats {
				if chat, ok := ch.(*tg.Chat); ok && chat.ID == peerID {
					chatName = chat.Title
					isGroup = true
					break
				}
			}
		case "channel":
			for _, ch := range chats {
				if channel, ok := ch.(*tg.Channel); ok && channel.ID == peerID {
					chatName = channel.Title
					isGroup = true
					break
				}
			}
		}

		if chatName != "" {
			chatsInfo = append(chatsInfo, ChatInfo{
				ID:      peerID,
				Name:    chatName,
				IsGroup: isGroup,
			})
		}
	}

	return chatsInfo, nil
}

// GetMessages fetches new messages from a specific chat starting from lastCollectedMessageID.
func (c *Client) GetMessages(ctx context.Context, chatID int64, lastCollectedMessageID int64) ([]CollectorMessage, error) {
	var newMessages []CollectorMessage

	targetInputPeer, err := c.resolveInputPeer(chatID)
	if err != nil {
		c.logger.Error("Target chat not found or could not resolve input peer", zap.Int64("chat_id", chatID), zap.Error(err))
		return nil, fmt.Errorf("chat with ID %d not found or could not resolve input peer: %w", chatID, err)
	}

	c.logger.Info("Fetching messages for chat", zap.Int64("chat_id", chatID), zap.Int64("from_message_id", lastCollectedMessageID))

	// Request message history for this specific dialog
	history, err := c.API().MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
		Peer:  targetInputPeer,
		Limit: 100, // Fetch a reasonable number of messages
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get message history for chat %d: %w", chatID, err)
	}

	var msgs []tg.MessageClass
	var historyUsers []tg.UserClass
	var historyChats []tg.ChatClass

	switch h := history.(type) {
	case *tg.MessagesMessages:
		msgs = h.Messages
		historyUsers = h.Users
		historyChats = h.Chats
	case *tg.MessagesMessagesSlice:
		msgs = h.Messages
		historyUsers = h.Users
		historyChats = h.Chats
	case *tg.MessagesChannelMessages:
		msgs = h.Messages
		historyUsers = h.Users
		historyChats = h.Chats
	default:
		c.logger.Warn("Unknown MessagesMessagesClass type", zap.String("type", fmt.Sprintf("%T", history)))
		return nil, fmt.Errorf("unknown messages type: %T", history)
	}

	// Filter messages to ensure they are newer than lastCollectedMessageID and populate CollectorMessage
	for _, m := range msgs {
		if msg, ok := m.(*tg.Message); ok && msg.ID > int(lastCollectedMessageID) {
			senderUsername := ""
			if msg.FromID != nil {
				senderUsername = resolveSenderUsername(msg.FromID, historyUsers, historyChats)
			}

			newMessages = append(newMessages, CollectorMessage{
				ID:             int64(msg.ID),
				ChatID:         chatID,
				SenderUsername: senderUsername,
				Timestamp:      time.Unix(int64(msg.Date), 0),
				Text:           msg.Message,
			})
		}
	}

	c.logger.Info("Fetched new messages for chat", zap.Int64("chat_id", chatID), zap.Int("count", len(newMessages)))
	return newMessages, nil
}

// resolveSenderUsername tries to find the username of the sender from the provided users and chats.
func resolveSenderUsername(peerID tg.PeerClass, users []tg.UserClass, chats []tg.ChatClass) string {
	switch p := peerID.(type) {
	case *tg.PeerUser:
		for _, u := range users {
			if user, ok := u.(*tg.User); ok && user.ID == p.UserID {
				return user.FirstName + " " + user.LastName // Or user.Username if available
			}
		}
	case *tg.PeerChat:
		for _, ch := range chats {
			if chat, ok := ch.(*tg.Chat); ok && chat.ID == p.ChatID {
				return chat.Title
			}
		}
	case *tg.PeerChannel:
		for _, ch := range chats {
			if channel, ok := ch.(*tg.Channel); ok && channel.ID == p.ChannelID {
				return channel.Title
			}
		}
	}
	return "unknown"
}

func (c *Client) resolveInputPeer(chatID int64) (tg.InputPeerClass, error) {
	// Iterate through cached dialogs to find the target chat
	for _, u := range c.cachedUsers {
		if user, ok := u.(*tg.User); ok && user.ID == chatID {
			return &tg.InputPeerUser{
				UserID:     user.ID,
				AccessHash: user.AccessHash,
			}, nil
		}
	}

	for _, ch := range c.cachedChats {
		if chat, ok := ch.(*tg.Chat); ok && chat.ID == chatID {
			return &tg.InputPeerChat{ChatID: chat.ID}, nil
		}
		if channel, ok := ch.(*tg.Channel); ok && channel.ID == chatID {
			return &tg.InputPeerChannel{
				ChannelID:  channel.ID,
				AccessHash: channel.AccessHash,
			}, nil
		}
	}

	return nil, fmt.Errorf("chat with ID %d not found in cache", chatID)
}

// Helper to get peer ID and type
func getPeerIDAndType(peer tg.PeerClass) (int64, string) {
	switch p := peer.(type) {
	case *tg.PeerUser:
		return p.UserID, "user"
	case *tg.PeerChat:
		return p.ChatID, "chat"
	case *tg.PeerChannel:
		return p.ChannelID, "channel"
	default:
		return 0, "unknown"
	}
}

func findInputPeer(peer tg.PeerClass, users []tg.UserClass, chats []tg.ChatClass) (tg.InputPeerClass, error) {
	switch p := peer.(type) {
	case *tg.PeerUser:
		for _, u := range users {
			if user, ok := u.(*tg.User); ok && user.ID == p.UserID {
				return &tg.InputPeerUser{
					UserID:     user.ID,
					AccessHash: user.AccessHash,
				}, nil
			}
		}
	case *tg.PeerChat:
		for _, ch := range chats {
			if chat, ok := ch.(*tg.Chat); ok && chat.ID == p.ChatID {
				return &tg.InputPeerChat{ChatID: chat.ID}, nil
			}
		}
	case *tg.PeerChannel:
		for _, ch := range chats {
			if channel, ok := ch.(*tg.Channel); ok && channel.ID == p.ChannelID {
				return &tg.InputPeerChannel{
					ChannelID:  channel.ID,
					AccessHash: channel.AccessHash,
				}, nil
			}
		}
	}
	return nil, fmt.Errorf("could not resolve peer: %+v", peer)
}

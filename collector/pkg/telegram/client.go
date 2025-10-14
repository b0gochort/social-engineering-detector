package telegram

import (
	"context"
	"fmt"
	"strings"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/tg"
	"github.com/gotd/td/session"
	"github.com/sirupsen/logrus"

	"collector/pkg/config"
)

// Client encapsulates the Telegram client.
type Client struct {
	*telegram.Client
	Sender *message.Sender
	AuthCode chan string // Channel to receive authentication code
	AuthCompleted chan struct{} // Channel to signal authentication completion
}

// NewClient creates and initializes a new Telegram client.
func NewClient(cfg *config.TelegramConfig) (*Client, error) {
	sessionFile := "session.json"
	
	client := telegram.NewClient(cfg.APIID, cfg.APIHash, telegram.Options{
		Logger: logrus.StandardLogger(), // Use logrus as the logger
		SessionStorage: &session.FileStorage{Path: sessionFile},
	})

	return &Client{
		Client: client,
		Sender: message.NewSender(client.API()),
		AuthCode: make(chan string),
		AuthCompleted: make(chan struct{}),
	}, nil
}

// Run starts the Telegram client and handles authentication.
func (c *Client) Run(ctx context.Context, phone string) error {
	return c.Client.Run(ctx, func(ctx context.Context) error {
		if err := c.auth(ctx, phone); err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}
		logrus.Info("Telegram client started and authenticated.")
		close(c.AuthCompleted) // Signal that authentication is complete

		// Keep the client running until the main context is cancelled
		<-ctx.Done()
		return ctx.Err() // Return the context error (e.g., context.Canceled)
	})
}

func (c *Client) auth(ctx context.Context, phone string) error {
	flow := auth.NewFlow(
		auth.Constant(phone, "", auth.CodeAuthenticatorFunc(func(ctx context.Context, sentCode *tg.AuthSentCode) (string, error) {
			logrus.Info("Waiting for authentication code via API...")
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

// GetMessages fetches new messages from all dialogs.
func (c *Client) GetMessages(ctx context.Context) ([]tg.MessageClass, error) {
	dialogs, err := c.API().MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
		OffsetPeer: &tg.InputPeerEmpty{},
		Limit:      100, // Fetch up to 100 dialogs
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get dialogs: %w", err)
	}

	var allMessages []tg.MessageClass
		if concreteDialogs, ok := dialogs.(*tg.MessagesDialogs); ok {
			allMessages = append(allMessages, concreteDialogs.Messages...)
		} else if concreteDialogsSlice, ok := dialogs.(*tg.MessagesDialogsSlice); ok {
			allMessages = append(allMessages, concreteDialogsSlice.Messages...)
		} else {
			logrus.Warnf("Unknown MessagesDialogsClass type: %T", dialogs)
		}

	// TODO: Iterate through dialogs to get messages from each peer, similar to previous attempt
	// The current implementation only gets messages that are directly part of the dialogs response,
	// not necessarily all messages from each dialog's history.

	return allMessages, nil
}

// Stop is no longer needed as client is stopped by context cancellation.
// func (c *Client) Stop() error {
// 	logrus.Info("Stopping Telegram client...")
// 	return c.Client.Stop()
// }
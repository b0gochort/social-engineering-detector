package collector

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"collector/pkg/storage"
	"collector/pkg/telegram"
)

// Collector is responsible for continuously collecting messages from Telegram.
type Collector struct {
	tgClient *telegram.Client
	dbStorage *storage.Storage
	interval time.Duration
}

// NewCollector creates a new Collector instance.
func NewCollector(tgClient *telegram.Client, dbStorage *storage.Storage, interval time.Duration) *Collector {
	return &Collector{
		tgClient: tgClient,
		dbStorage: dbStorage,
		interval: interval,
	}
}

// Run starts the message collection process.
func (c *Collector) Run(ctx context.Context) {
	logrus.Info("Starting message collector...")
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logrus.Info("Message collector stopped.")
			return
		case <-ticker.C:
			logrus.Info("Fetching messages...")
			messages, err := c.tgClient.GetMessages(ctx)
			if err != nil {
				logrus.Printf("Error fetching messages: %v", err)
				continue
			}

			logrus.Printf("Fetched %d messages. Saving to database...", len(messages))
			for _, msg := range messages {
				if err := c.dbStorage.SaveMessage(msg); err != nil {
					logrus.Printf("Error saving message to database: %v", err)
				}
			}
			logrus.Info("Messages saved to database.")
		}
	}
}

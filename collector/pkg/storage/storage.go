package storage

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/gotd/td/tg"
	"github.com/sirupsen/logrus"
)

// Message represents a Telegram message to be stored in the database.
type Message struct {
	ID              int64
	ChatID          int64
	SenderID        int64
	SenderUsername  sql.NullString
	SenderFirstName sql.NullString
	SenderLastName  sql.NullString
	MessageText     sql.NullString
	MessageDate     time.Time
	IsOutgoing      bool
	IsChannelPost   bool
	IsGroupMessage  bool
}

// Storage manages database operations.
type Storage struct {
	db *sql.DB
}

// NewStorage creates a new Storage instance and initializes the database.
func NewStorage(dataSourceName string) (*Storage, error) {
	db, err := sql.Open("postgres", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return &Storage{db: db}, nil
}

// Close closes the database connection.
func (s *Storage) Close() error {
	return s.db.Close()
}

// ApplyMigrations applies database migrations.
func ApplyMigrations(databaseURL, migrationsPath string) error {
	m, err := migrate.New(
		"file://"+migrationsPath,
		databaseURL,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil {
		if err == migrate.ErrNoChange {
			logrus.Info("No new migrations to apply.")
		} else {
			return fmt.Errorf("failed to apply migrations: %w", err)
		}
	}

	return nil
}

// SaveMessage saves a Telegram message to the database.
func (s *Storage) SaveMessage(msg tg.MessageClass) error {
	// Type assertion to get the concrete message type
	message, ok := msg.(*tg.Message)
	if !ok {
		// Handle other message types if necessary, or skip
		return nil // Skip non-tg.Message types for now
	}

	var ( // Initialize with default values
		chatID          int64
		senderID        int64
		senderUsername  sql.NullString
		senderFirstName sql.NullString
		senderLastName  sql.NullString
		isOutgoing      bool
		isChannelPost   bool
		isGroupMessage  bool
	)

	// Extract chat ID
	if p, ok := message.PeerID.(*tg.PeerChannel); ok {
		chatID = p.ChannelID
		isChannelPost = true
	} else if p, ok := message.PeerID.(*tg.PeerChat); ok {
		chatID = p.ChatID
		isGroupMessage = true
	} else if p, ok := message.PeerID.(*tg.PeerUser); ok {
		chatID = p.UserID
	}

	// Extract sender ID and info
	if p, ok := message.FromID.(*tg.PeerUser); ok {
		senderID = p.UserID
		// TODO: Fetch user details from Telegram API using senderID to get username/first/last name
		// For now, these will remain null unless we have a way to resolve them.
	} else if p, ok := message.FromID.(*tg.PeerChannel); ok {
		senderID = p.ChannelID // Or some other identifier for channel posts
		isChannelPost = true
	} else if p, ok := message.FromID.(*tg.PeerChat); ok {
		senderID = p.ChatID // Or some other identifier for group messages
		isGroupMessage = true
	}

	// Check if the message is outgoing
	isOutgoing = message.Out

	query := `
	INSERT INTO messages (
		id, chat_id, sender_id, sender_username, sender_first_name, sender_last_name,
		message_text, message_date, is_outgoing, is_channel_post, is_group_message
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
	) ON CONFLICT (id) DO NOTHING;
	`

	_, err := s.db.Exec(query,
		message.ID,
		chatID,
		senderID,
		senderUsername,
		senderFirstName,
		senderLastName,
		message.Message,
		time.Unix(int64(message.Date), 0),
		isOutgoing,
		isChannelPost,
		isGroupMessage,
	)

	return err
}
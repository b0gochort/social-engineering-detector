package models

import "time"

// Message represents a message stored in the 'messages' table.
type Message struct {
	ID                int64     `db:"id"`
	ChatID            int64     `db:"chat_id"`
	TelegramMessageID *int64    `db:"telegram_message_id"` // Nullable for VK messages
	VKMessageID       *int64    `db:"vk_message_id"`       // VK message ID
	Source            string    `db:"source"`              // "telegram" or "vk"
	MessageType       string    `db:"message_type"`        // "message", "post", "comment"
	SenderUsername    string    `db:"sender_username"`
	Timestamp         time.Time `db:"timestamp"`
	ContentEncrypted  string    `db:"content_encrypted"`
}

// Incident represents an incident stored in the 'incidents' table.
type Incident struct {
	ID                     int64     `db:"id" json:"id"`
	MessageID              int64     `db:"message_id" json:"message_id"` // References messages.id
	ThreatType             string    `db:"threat_type" json:"threat_type"`
	ModelConfidence        float64   `db:"model_confidence" json:"confidence"`
	Status                 string    `db:"status" json:"status"`
	ChatTitle              string    `db:"chat_title" json:"chat_title"`
	CreatedAt              time.Time `db:"created_at" json:"created_at"`
	SummaryEncrypted       string    `db:"summary_encrypted" json:"message_text"`
	AccessGranted          bool      `db:"access_granted" json:"access_granted"`
	CurrentAccessRequestID *int64    `db:"current_access_request_id" json:"current_access_request_id,omitempty"`
	V2CategoryID           *int      `db:"v2_category_id" json:"v2_category_id,omitempty"`
	V4CategoryID           *int      `db:"v4_category_id" json:"v4_category_id,omitempty"`
	ModelsAgree            *bool     `db:"models_agree" json:"models_agree,omitempty"`
	Source                 string    `db:"source" json:"source"` // "telegram" or "vk"
}
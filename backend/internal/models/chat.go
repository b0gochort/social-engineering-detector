package models

import "time"

type Chat struct {
	ID                     int64      `db:"id" json:"id"`
	TelegramID             *int64     `db:"telegram_id" json:"telegram_id,omitempty"`  // Nullable for VK chats
	VKPeerID               *int64     `db:"vk_peer_id" json:"vk_peer_id,omitempty"`    // VK conversation peer_id
	Source                 string     `db:"source" json:"source"`                       // "telegram" or "vk"
	Name                   string     `db:"name" json:"title"`                          // Frontend expects "title"
	IsGroup                bool       `db:"is_group" json:"is_group"`
	MonitoringActive       bool       `db:"monitoring_active" json:"is_monitored"`      // Frontend expects "is_monitored"
	LastCollectedMessageID int64      `db:"last_collected_message_id" json:"last_collected_message_id"`

	// Statistics fields (computed from joined queries)
	MessageCount     int        `db:"message_count" json:"message_count"`
	MemberCount      *int       `db:"member_count" json:"member_count"`               // Nullable
	LastMessageDate  *time.Time `db:"last_message_date" json:"last_message_date"`    // Nullable
	ChatType         string     `db:"chat_type" json:"chat_type"`                     // user, group, chat, channel
}

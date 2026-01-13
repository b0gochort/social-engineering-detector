package repository

import (
	"backend/internal/models"
	"database/sql"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type ChatRepository interface {
	GetChatByTelegramID(telegramID int64) (*models.Chat, error)
	GetChatByVKPeerID(vkPeerID int64) (*models.Chat, error)
	GetChatByID(id int64) (*models.Chat, error)
	UpdateLastCollectedMessageID(chatID, lastCollectedMessageID int64) error
	UpdateMonitoringStatus(chatID int64, active bool) error
	CreateChat(chat *models.Chat) error
	GetAllChats() ([]*models.Chat, error)
}

type chatRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

func NewChatRepository(db *sqlx.DB, logger *zap.Logger) ChatRepository {
	return &chatRepository{db: db, logger: logger}
}

func (r *chatRepository) GetChatByTelegramID(telegramID int64) (*models.Chat, error) {
	var chat models.Chat
	query := `SELECT id, telegram_id, vk_peer_id, source, name, is_group, monitoring_active, last_collected_message_id, chat_type FROM chats WHERE telegram_id = $1`
	err := r.db.Get(&chat, query, telegramID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Chat not found
		}
		return nil, err
	}
	return &chat, nil
}

func (r *chatRepository) GetChatByVKPeerID(vkPeerID int64) (*models.Chat, error) {
	var chat models.Chat
	query := `SELECT id, telegram_id, vk_peer_id, source, name, is_group, monitoring_active, last_collected_message_id, chat_type FROM chats WHERE vk_peer_id = $1`
	err := r.db.Get(&chat, query, vkPeerID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Chat not found
		}
		return nil, err
	}
	return &chat, nil
}

func (r *chatRepository) UpdateLastCollectedMessageID(chatID, lastCollectedMessageID int64) error {
	query := `UPDATE chats SET last_collected_message_id = $1 WHERE id = $2`
	_, err := r.db.Exec(query, lastCollectedMessageID, chatID)
	return err
}

func (r *chatRepository) CreateChat(chat *models.Chat) error {
	query := `INSERT INTO chats (telegram_id, vk_peer_id, source, name, is_group, monitoring_active, last_collected_message_id, chat_type)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`
	return r.db.QueryRowx(query, chat.TelegramID, chat.VKPeerID, chat.Source, chat.Name, chat.IsGroup,
		chat.MonitoringActive, chat.LastCollectedMessageID, chat.ChatType).StructScan(chat)
}

func (r *chatRepository) GetChatByID(id int64) (*models.Chat, error) {
	var chat models.Chat
	query := `SELECT id, telegram_id, vk_peer_id, source, name, is_group, monitoring_active, last_collected_message_id, chat_type FROM chats WHERE id = $1`
	err := r.db.Get(&chat, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &chat, nil
}

func (r *chatRepository) UpdateMonitoringStatus(chatID int64, active bool) error {
	query := `UPDATE chats SET monitoring_active = $1 WHERE id = $2`
	_, err := r.db.Exec(query, active, chatID)
	return err
}

func (r *chatRepository) GetAllChats() ([]*models.Chat, error) {
	var chats []*models.Chat
	query := `
		SELECT
			c.id,
			c.telegram_id,
			c.vk_peer_id,
			c.source,
			c.name,
			c.is_group,
			c.monitoring_active,
			c.last_collected_message_id,
			COALESCE(COUNT(m.id), 0) as message_count,
			NULL::integer as member_count,
			MAX(m.timestamp) as last_message_date,
			c.chat_type
		FROM chats c
		LEFT JOIN messages m ON c.id = m.chat_id
		GROUP BY c.id, c.telegram_id, c.vk_peer_id, c.source, c.name, c.is_group, c.monitoring_active, c.last_collected_message_id, c.chat_type
		ORDER BY c.id
	`
	err := r.db.Select(&chats, query)
	if err != nil {
		return nil, err
	}
	return chats, nil
}

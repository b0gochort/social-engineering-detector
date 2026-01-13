package repository

import (
	"backend/internal/models"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type MessageRepository interface {
	SaveMessage(msg *models.Message) error
	GetMessageByID(id int64) (*models.Message, error)
	SaveIncident(incident *models.Incident) error
	GetAllIncidents() ([]*models.Incident, error)
	GetIncidentByID(id int64) (*models.Incident, error)
	UpdateIncidentStatus(id int64, status string) error
	GetIncidentsByStatus(status string) ([]*models.Incident, error)
	GetIncidentsByThreatType(threatType string) ([]*models.Incident, error)
	UpdateIncidentAccessGranted(incidentID int64, granted bool, requestID *int64) error
}

type messageRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

func NewMessageRepository(db *sqlx.DB, logger *zap.Logger) MessageRepository {
	return &messageRepository{db: db, logger: logger}
}

func (r *messageRepository) SaveMessage(msg *models.Message) error {
	query := `INSERT INTO messages (chat_id, telegram_message_id, vk_message_id, source, message_type, sender_username, timestamp, content_encrypted)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`
	return r.db.QueryRowx(query, msg.ChatID, msg.TelegramMessageID, msg.VKMessageID, msg.Source,
		msg.MessageType, msg.SenderUsername, msg.Timestamp, msg.ContentEncrypted).StructScan(msg)
}

func (r *messageRepository) GetMessageByID(id int64) (*models.Message, error) {
	var msg models.Message
	query := `SELECT id, chat_id, telegram_message_id, vk_message_id, source, message_type, sender_username, timestamp, content_encrypted FROM messages WHERE id = $1`
	err := r.db.Get(&msg, query, id)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

func (r *messageRepository) SaveIncident(incident *models.Incident) error {
	query := `INSERT INTO incidents (message_id, threat_type, model_confidence, status, summary_encrypted) VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at`
	return r.db.QueryRowx(query, incident.MessageID, incident.ThreatType, incident.ModelConfidence, incident.Status, incident.SummaryEncrypted).StructScan(incident)
}

func (r *messageRepository) GetAllIncidents() ([]*models.Incident, error) {
	var incidents []*models.Incident
	query := `
		SELECT
			i.id,
			i.message_id,
			i.threat_type,
			i.model_confidence,
			i.status,
			COALESCE(c.name, 'Неизвестно') AS chat_title,
			i.created_at,
			i.summary_encrypted,
			i.access_granted,
			i.current_access_request_id,
			m.source
		FROM incidents i
		LEFT JOIN messages m ON i.message_id = m.id
		LEFT JOIN chats c ON m.chat_id = c.id
		ORDER BY i.created_at DESC
	`

	rows, err := r.db.Queryx(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		result := make(map[string]interface{})
		err := rows.MapScan(result)
		if err != nil {
			r.logger.Error("Failed to map scan incident", zap.Error(err))
			continue
		}

		// Debug log to see what we got
		r.logger.Info("DEBUG MapScan result",
			zap.Any("threat_type_type", fmt.Sprintf("%T", result["threat_type"])),
			zap.Any("threat_type_value", result["threat_type"]),
			zap.Any("status_type", fmt.Sprintf("%T", result["status"])),
			zap.Any("status_value", result["status"]))

		incident := &models.Incident{}
		if id, ok := result["id"].(int64); ok {
			incident.ID = id
		}
		if msgID, ok := result["message_id"].(int64); ok {
			incident.MessageID = msgID
		}

		// Handle threat_type
		if val := result["threat_type"]; val != nil {
			if threatType, ok := val.([]byte); ok {
				incident.ThreatType = string(threatType)
			} else if threatType, ok := val.(string); ok {
				incident.ThreatType = threatType
			} else {
				incident.ThreatType = fmt.Sprintf("%v", val)
			}
		}

		if conf, ok := result["model_confidence"].(float64); ok {
			incident.ModelConfidence = conf
		}

		// Handle status
		if val := result["status"]; val != nil {
			if status, ok := val.([]byte); ok {
				incident.Status = string(status)
			} else if status, ok := val.(string); ok {
				incident.Status = status
			} else {
				incident.Status = fmt.Sprintf("%v", val)
			}
		}

		// Handle chat_title
		if val := result["chat_title"]; val != nil {
			if chatTitle, ok := val.([]byte); ok {
				incident.ChatTitle = string(chatTitle)
			} else if chatTitle, ok := val.(string); ok {
				incident.ChatTitle = chatTitle
			} else {
				incident.ChatTitle = fmt.Sprintf("%v", val)
			}
		}

		if createdAt, ok := result["created_at"].(time.Time); ok {
			incident.CreatedAt = createdAt
		}

		// Handle summary_encrypted
		if val := result["summary_encrypted"]; val != nil {
			if summary, ok := val.([]byte); ok {
				incident.SummaryEncrypted = string(summary)
			} else if summary, ok := val.(string); ok {
				incident.SummaryEncrypted = summary
			} else {
				incident.SummaryEncrypted = fmt.Sprintf("%v", val)
			}
		}

		// Handle access_granted
		if accessGranted, ok := result["access_granted"].(bool); ok {
			incident.AccessGranted = accessGranted
		}

		// Handle current_access_request_id
		if requestID, ok := result["current_access_request_id"].(int64); ok {
			incident.CurrentAccessRequestID = &requestID
		}

		// Handle source
		if val := result["source"]; val != nil {
			if source, ok := val.([]byte); ok {
				incident.Source = string(source)
			} else if source, ok := val.(string); ok {
				incident.Source = source
			} else {
				incident.Source = fmt.Sprintf("%v", val)
			}
		}

		incidents = append(incidents, incident)
	}

	return incidents, rows.Err()
}

func (r *messageRepository) GetIncidentByID(id int64) (*models.Incident, error) {
	incident := &models.Incident{}
	query := `
		SELECT
			i.id,
			i.message_id,
			i.threat_type,
			i.model_confidence,
			i.status,
			COALESCE(c.name, 'Неизвестно') as chat_title,
			i.created_at,
			i.summary_encrypted,
			i.access_granted,
			i.current_access_request_id,
			m.source
		FROM incidents i
		LEFT JOIN messages m ON i.message_id = m.id
		LEFT JOIN chats c ON m.chat_id = c.id
		WHERE i.id = $1
	`

	err := r.db.Get(incident, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return incident, nil
}

func (r *messageRepository) UpdateIncidentStatus(id int64, status string) error {
	query := `UPDATE incidents SET status = $1 WHERE id = $2`
	_, err := r.db.Exec(query, status, id)
	return err
}

func (r *messageRepository) GetIncidentsByStatus(status string) ([]*models.Incident, error) {
	var incidents []*models.Incident
	query := `
		SELECT
			i.id,
			i.message_id,
			i.threat_type,
			i.model_confidence,
			i.status,
			COALESCE(c.name, 'Неизвестно') as chat_title,
			i.created_at,
			i.summary_encrypted
		FROM incidents i
		LEFT JOIN messages m ON i.message_id = m.id
		LEFT JOIN chats c ON m.chat_id = c.id
		WHERE i.status = $1
		ORDER BY i.created_at DESC
	`

	rows, err := r.db.Queryx(query, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		result := make(map[string]interface{})
		err := rows.MapScan(result)
		if err != nil {
			r.logger.Error("Failed to map scan incident", zap.Error(err))
			continue
		}

		incident := &models.Incident{}
		if id, ok := result["id"].(int64); ok {
			incident.ID = id
		}
		if msgID, ok := result["message_id"].(int64); ok {
			incident.MessageID = msgID
		}
		if threatType, ok := result["threat_type"].([]byte); ok {
			incident.ThreatType = string(threatType)
		}
		if conf, ok := result["model_confidence"].(float64); ok {
			incident.ModelConfidence = conf
		}
		if statusVal, ok := result["status"].([]byte); ok {
			incident.Status = string(statusVal)
		}
		if chatTitle, ok := result["chat_title"].([]byte); ok {
			incident.ChatTitle = string(chatTitle)
		} else if chatTitle, ok := result["chat_title"].(string); ok {
			incident.ChatTitle = chatTitle
		}
		if createdAt, ok := result["created_at"].(time.Time); ok {
			incident.CreatedAt = createdAt
		}
		if summary, ok := result["summary_encrypted"].([]byte); ok {
			incident.SummaryEncrypted = string(summary)
		}

		incidents = append(incidents, incident)
	}

	return incidents, rows.Err()
}

func (r *messageRepository) GetIncidentsByThreatType(threatType string) ([]*models.Incident, error) {
	var incidents []*models.Incident
	query := `
		SELECT
			i.id,
			i.message_id,
			i.threat_type,
			i.model_confidence,
			i.status,
			COALESCE(c.name, 'Неизвестно') as chat_title,
			i.created_at,
			i.summary_encrypted
		FROM incidents i
		LEFT JOIN messages m ON i.message_id = m.id
		LEFT JOIN chats c ON m.chat_id = c.id
		WHERE i.threat_type = $1
		ORDER BY i.created_at DESC
	`

	rows, err := r.db.Queryx(query, threatType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		result := make(map[string]interface{})
		err := rows.MapScan(result)
		if err != nil {
			r.logger.Error("Failed to map scan incident", zap.Error(err))
			continue
		}

		incident := &models.Incident{}
		if id, ok := result["id"].(int64); ok {
			incident.ID = id
		}
		if msgID, ok := result["message_id"].(int64); ok {
			incident.MessageID = msgID
		}
		if threatTypeVal, ok := result["threat_type"].([]byte); ok {
			incident.ThreatType = string(threatTypeVal)
		}
		if conf, ok := result["model_confidence"].(float64); ok {
			incident.ModelConfidence = conf
		}
		if status, ok := result["status"].([]byte); ok {
			incident.Status = string(status)
		}
		if chatTitle, ok := result["chat_title"].([]byte); ok {
			incident.ChatTitle = string(chatTitle)
		} else if chatTitle, ok := result["chat_title"].(string); ok {
			incident.ChatTitle = chatTitle
		}
		if createdAt, ok := result["created_at"].(time.Time); ok {
			incident.CreatedAt = createdAt
		}
		if summary, ok := result["summary_encrypted"].([]byte); ok {
			incident.SummaryEncrypted = string(summary)
		}

		incidents = append(incidents, incident)
	}

	return incidents, rows.Err()
}

func (r *messageRepository) UpdateIncidentAccessGranted(incidentID int64, granted bool, requestID *int64) error {
	query := `
		UPDATE incidents
		SET access_granted = $1, current_access_request_id = $2
		WHERE id = $3
	`

	result, err := r.db.Exec(query, granted, requestID, incidentID)
	if err != nil {
		r.logger.Error("Failed to update incident access granted",
			zap.Int64("incident_id", incidentID),
			zap.Bool("granted", granted),
			zap.Error(err))
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("incident not found: %d", incidentID)
	}

	return nil
}
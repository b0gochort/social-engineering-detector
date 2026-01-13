package repository

import (
	"backend/internal/models"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// AccessRequestRepository defines the interface for access request operations
type AccessRequestRepository interface {
	Create(req *models.AccessRequest) error
	GetByID(id int64) (*models.AccessRequest, error)
	GetByIncidentID(incidentID int64) (*models.AccessRequest, error)
	GetPendingByChildID(childID int64) ([]*models.AccessRequest, error)
	UpdateStatus(id int64, status string, respondedAt time.Time) error
}

type accessRequestRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewAccessRequestRepository creates a new access request repository
func NewAccessRequestRepository(db *sqlx.DB, logger *zap.Logger) AccessRequestRepository {
	return &accessRequestRepository{
		db:     db,
		logger: logger,
	}
}

func (r *accessRequestRepository) Create(req *models.AccessRequest) error {
	query := `
		INSERT INTO access_requests (incident_id, parent_id, child_id, status, requested_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(
		query,
		req.IncidentID,
		req.ParentID,
		req.ChildID,
		req.Status,
		req.RequestedAt,
	).Scan(&req.ID, &req.CreatedAt, &req.UpdatedAt)

	if err != nil {
		r.logger.Error("Failed to create access request", zap.Error(err))
		return err
	}

	return nil
}

func (r *accessRequestRepository) GetByID(id int64) (*models.AccessRequest, error) {
	var req models.AccessRequest
	query := `
		SELECT id, incident_id, parent_id, child_id, status, requested_at, responded_at, created_at, updated_at
		FROM access_requests
		WHERE id = $1
	`

	err := r.db.Get(&req, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.logger.Error("Failed to get access request by ID", zap.Int64("id", id), zap.Error(err))
		return nil, err
	}

	return &req, nil
}

func (r *accessRequestRepository) GetByIncidentID(incidentID int64) (*models.AccessRequest, error) {
	var req models.AccessRequest
	query := `
		SELECT id, incident_id, parent_id, child_id, status, requested_at, responded_at, created_at, updated_at
		FROM access_requests
		WHERE incident_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	err := r.db.Get(&req, query, incidentID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.logger.Error("Failed to get access request by incident ID", zap.Int64("incident_id", incidentID), zap.Error(err))
		return nil, err
	}

	return &req, nil
}

func (r *accessRequestRepository) GetPendingByChildID(childID int64) ([]*models.AccessRequest, error) {
	var requests []*models.AccessRequest
	query := `
		SELECT id, incident_id, parent_id, child_id, status, requested_at, responded_at, created_at, updated_at
		FROM access_requests
		WHERE child_id = $1 AND status = 'pending'
		ORDER BY created_at DESC
	`

	err := r.db.Select(&requests, query, childID)
	if err != nil {
		r.logger.Error("Failed to get pending access requests", zap.Int64("child_id", childID), zap.Error(err))
		return nil, err
	}

	return requests, nil
}

func (r *accessRequestRepository) UpdateStatus(id int64, status string, respondedAt time.Time) error {
	query := `
		UPDATE access_requests
		SET status = $1, responded_at = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
	`

	result, err := r.db.Exec(query, status, respondedAt, id)
	if err != nil {
		r.logger.Error("Failed to update access request status", zap.Int64("id", id), zap.String("status", status), zap.Error(err))
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

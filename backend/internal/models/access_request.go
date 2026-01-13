package models

import "time"

// AccessRequest represents a request from a parent to access incident message content
type AccessRequest struct {
	ID          int64     `db:"id" json:"id"`
	IncidentID  int64     `db:"incident_id" json:"incident_id"`
	ParentID    int64     `db:"parent_id" json:"parent_id"`
	ChildID     int64     `db:"child_id" json:"child_id"`
	Status      string    `db:"status" json:"status"` // pending, approved, rejected
	RequestedAt time.Time `db:"requested_at" json:"requested_at"`
	RespondedAt *time.Time `db:"responded_at" json:"responded_at,omitempty"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

// CreateAccessRequestInput represents input for creating an access request
type CreateAccessRequestInput struct {
	IncidentID int64 `json:"incident_id" binding:"required"`
}

// RespondToAccessRequestInput represents input for responding to an access request
type RespondToAccessRequestInput struct {
	Action string `json:"action" binding:"required,oneof=approve reject"` // approve or reject
}

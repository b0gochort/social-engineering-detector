package models

import "time"

// MLDatasetEntry represents a single entry in the ML dataset table.
// This table stores ALL messages (neutral + threats) in plain text for ML training.
// NOTE: This data is NOT encrypted!
type MLDatasetEntry struct {
	ID        int64     `db:"id" json:"id"`

	// Message content (plain text)
	MessageText string `db:"message_text" json:"message_text"`

	// LLM Annotation
	CategoryID   int     `db:"category_id" json:"category_id"`
	CategoryName string  `db:"category_name" json:"category_name"`
	Justification string `db:"justification" json:"justification"`

	// Model metadata
	Provider      string    `db:"provider" json:"provider"`           // groq, gemini, etc.
	ModelVersion  string    `db:"model_version" json:"model_version"` // llama-3.3-70b-versatile, etc.
	AnnotatedAt   time.Time `db:"annotated_at" json:"annotated_at"`

	// Optional reference to original encrypted message
	OriginalMessageID *int64 `db:"original_message_id" json:"original_message_id,omitempty"`

	// Validation
	IsValidated  bool       `db:"is_validated" json:"is_validated"`
	ValidatedBy  *int64     `db:"validated_by" json:"validated_by,omitempty"`
	ValidatedAt  *time.Time `db:"validated_at" json:"validated_at,omitempty"`

	// Metadata
	Source    string    `db:"source" json:"source"` // telegram, manual, synthetic
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

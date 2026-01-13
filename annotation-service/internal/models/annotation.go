package models

import "time"

// ThreatCategory represents the 9 categories from your llm.py
type ThreatCategory int

const (
	Grooming            ThreatCategory = 1 // Склонение к сексуальным действиям (Груминг)
	Blackmail           ThreatCategory = 2 // Угрозы, шантаж, вымогательство
	Bullying            ThreatCategory = 3 // Физическое насилие/Буллинг
	SuicideEncouragement ThreatCategory = 4 // Склонение к суициду/Самоповреждению
	DangerousActivities ThreatCategory = 5 // Склонение к опасным играм/действиям
	DrugPropaganda      ThreatCategory = 6 // Пропаганда запрещенных веществ
	FinancialFraud      ThreatCategory = 7 // Финансовое мошенничество
	Phishing            ThreatCategory = 8 // Сбор личных данных (Фишинг)
	Neutral             ThreatCategory = 9 // Нейтральное общение
)

// CategoryNames maps category IDs to their Russian names
var CategoryNames = map[ThreatCategory]string{
	Grooming:            "Склонение к сексуальным действиям (Груминг)",
	Blackmail:           "Угрозы, шантаж, вымогательство",
	Bullying:            "Физическое насилие/Буллинг",
	SuicideEncouragement: "Склонение к суициду/Самоповреждению",
	DangerousActivities: "Склонение к опасным играм/действиям",
	DrugPropaganda:      "Пропаганда запрещенных веществ",
	FinancialFraud:      "Финансовое мошенничество",
	Phishing:            "Сбор личных данных (Фишинг)",
	Neutral:             "Нейтральное общение",
}

// Annotation represents a labeled message
type Annotation struct {
	ID             int64          `json:"id" db:"id"`
	MessageID      *int64         `json:"message_id,omitempty" db:"message_id"` // Optional link to original message
	Text           string         `json:"text" db:"text"`
	Category       ThreatCategory `json:"category_id" db:"category_id"`
	CategoryName   string         `json:"category_name" db:"category_name"`
	Justification  string         `json:"justification" db:"justification"`
	Confidence     float64        `json:"confidence,omitempty" db:"confidence"` // If LLM provides confidence
	AnnotatedAt    time.Time      `json:"annotated_at" db:"annotated_at"`
	Provider       string         `json:"provider" db:"provider"` // "gemini", "manual", etc.
	ModelVersion   string         `json:"model_version,omitempty" db:"model_version"`
	IsValidated    bool           `json:"is_validated" db:"is_validated"` // Manual validation flag
}

// AnnotationRequest for single message annotation
type AnnotationRequest struct {
	Text string `json:"text" binding:"required"`
}

// BatchAnnotationRequest for multiple messages
type BatchAnnotationRequest struct {
	Messages []MessageInput `json:"messages" binding:"required,min=1"`
}

// MessageInput represents input message for annotation
type MessageInput struct {
	ID   *int64 `json:"id,omitempty"` // Optional external ID
	Text string `json:"text" binding:"required"`
}

// AnnotationResponse returned by LLM providers (Gemini, Groq, OpenRouter, etc.)
type AnnotationResponse struct {
	CategoryName  string    `json:"category_name"`
	CategoryID    int       `json:"category_id"`
	Justification string    `json:"justification"`
	Confidence    float64   `json:"confidence,omitempty"` // Optional
	Provider      string    `json:"provider"`              // groq, gemini, openrouter
	ModelVersion  string    `json:"model_version"`         // Model name/version
	AnnotatedAt   time.Time `json:"annotated_at"`          // Timestamp
	IsValidated   bool      `json:"is_validated"`          // Manual validation flag
}

// Job represents an async annotation job
type Job struct {
	ID          string    `json:"id" db:"id"`
	Status      string    `json:"status" db:"status"` // "pending", "processing", "completed", "failed"
	TotalCount  int       `json:"total_count" db:"total_count"`
	ProcessedCount int    `json:"processed_count" db:"processed_count"`
	FailedCount int       `json:"failed_count" db:"failed_count"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	ErrorMessage string   `json:"error_message,omitempty" db:"error_message"`
}

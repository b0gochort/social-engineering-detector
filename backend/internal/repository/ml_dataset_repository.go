package repository

import (
	"backend/internal/models"
	"database/sql"
)

// MLDatasetRepository handles database operations for the ML dataset table.
type MLDatasetRepository interface {
	SaveEntry(entry *models.MLDatasetEntry) error
	GetAllEntries() ([]*models.MLDatasetEntry, error)
	GetEntriesByCategory(categoryID int) ([]*models.MLDatasetEntry, error)
	GetValidatedEntries() ([]*models.MLDatasetEntry, error)
	GetUnvalidatedEntries() ([]*models.MLDatasetEntry, error)
	ValidateEntry(entryID int64, validatedBy int64) error
	GetDatasetStats() (map[string]interface{}, error)
}

type mlDatasetRepository struct {
	db *sql.DB
}

// NewMLDatasetRepository creates a new ML dataset repository.
func NewMLDatasetRepository(db *sql.DB) MLDatasetRepository {
	return &mlDatasetRepository{db: db}
}

// SaveEntry saves a new ML dataset entry to the database.
func (r *mlDatasetRepository) SaveEntry(entry *models.MLDatasetEntry) error {
	query := `
		INSERT INTO ml_dataset (
			message_text, category_id, category_name, justification,
			provider, model_version, annotated_at,
			original_message_id, is_validated, source
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at
	`
	return r.db.QueryRow(
		query,
		entry.MessageText,
		entry.CategoryID,
		entry.CategoryName,
		entry.Justification,
		entry.Provider,
		entry.ModelVersion,
		entry.AnnotatedAt,
		entry.OriginalMessageID,
		entry.IsValidated,
		entry.Source,
	).Scan(&entry.ID, &entry.CreatedAt)
}

// GetAllEntries returns all ML dataset entries.
func (r *mlDatasetRepository) GetAllEntries() ([]*models.MLDatasetEntry, error) {
	query := `
		SELECT id, message_text, category_id, category_name, justification,
		       provider, model_version, annotated_at,
		       original_message_id, is_validated, validated_by, validated_at,
		       source, created_at
		FROM ml_dataset
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*models.MLDatasetEntry
	for rows.Next() {
		entry := &models.MLDatasetEntry{}
		err := rows.Scan(
			&entry.ID, &entry.MessageText, &entry.CategoryID, &entry.CategoryName,
			&entry.Justification, &entry.Provider, &entry.ModelVersion, &entry.AnnotatedAt,
			&entry.OriginalMessageID, &entry.IsValidated, &entry.ValidatedBy,
			&entry.ValidatedAt, &entry.Source, &entry.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

// GetEntriesByCategory returns ML dataset entries filtered by category.
func (r *mlDatasetRepository) GetEntriesByCategory(categoryID int) ([]*models.MLDatasetEntry, error) {
	query := `
		SELECT id, message_text, category_id, category_name, justification,
		       provider, model_version, annotated_at,
		       original_message_id, is_validated, validated_by, validated_at,
		       source, created_at
		FROM ml_dataset
		WHERE category_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(query, categoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*models.MLDatasetEntry
	for rows.Next() {
		entry := &models.MLDatasetEntry{}
		err := rows.Scan(
			&entry.ID, &entry.MessageText, &entry.CategoryID, &entry.CategoryName,
			&entry.Justification, &entry.Provider, &entry.ModelVersion, &entry.AnnotatedAt,
			&entry.OriginalMessageID, &entry.IsValidated, &entry.ValidatedBy,
			&entry.ValidatedAt, &entry.Source, &entry.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

// GetValidatedEntries returns only validated ML dataset entries.
func (r *mlDatasetRepository) GetValidatedEntries() ([]*models.MLDatasetEntry, error) {
	query := `
		SELECT id, message_text, category_id, category_name, justification,
		       provider, model_version, annotated_at,
		       original_message_id, is_validated, validated_by, validated_at,
		       source, created_at
		FROM ml_dataset
		WHERE is_validated = TRUE
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*models.MLDatasetEntry
	for rows.Next() {
		entry := &models.MLDatasetEntry{}
		err := rows.Scan(
			&entry.ID, &entry.MessageText, &entry.CategoryID, &entry.CategoryName,
			&entry.Justification, &entry.Provider, &entry.ModelVersion, &entry.AnnotatedAt,
			&entry.OriginalMessageID, &entry.IsValidated, &entry.ValidatedBy,
			&entry.ValidatedAt, &entry.Source, &entry.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

// GetUnvalidatedEntries returns only unvalidated ML dataset entries.
func (r *mlDatasetRepository) GetUnvalidatedEntries() ([]*models.MLDatasetEntry, error) {
	query := `
		SELECT id, message_text, category_id, category_name, justification,
		       provider, model_version, annotated_at,
		       original_message_id, is_validated, validated_by, validated_at,
		       source, created_at
		FROM ml_dataset
		WHERE is_validated = FALSE
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*models.MLDatasetEntry
	for rows.Next() {
		entry := &models.MLDatasetEntry{}
		err := rows.Scan(
			&entry.ID, &entry.MessageText, &entry.CategoryID, &entry.CategoryName,
			&entry.Justification, &entry.Provider, &entry.ModelVersion, &entry.AnnotatedAt,
			&entry.OriginalMessageID, &entry.IsValidated, &entry.ValidatedBy,
			&entry.ValidatedAt, &entry.Source, &entry.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

// ValidateEntry marks an entry as validated by a user.
func (r *mlDatasetRepository) ValidateEntry(entryID int64, validatedBy int64) error {
	query := `
		UPDATE ml_dataset
		SET is_validated = TRUE,
		    validated_by = $1,
		    validated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`
	_, err := r.db.Exec(query, validatedBy, entryID)
	return err
}

// GetDatasetStats returns statistics about the ML dataset.
func (r *mlDatasetRepository) GetDatasetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total entries
	var totalCount int
	err := r.db.QueryRow("SELECT COUNT(*) FROM ml_dataset").Scan(&totalCount)
	if err != nil {
		return nil, err
	}
	stats["total_entries"] = totalCount

	// Validated vs unvalidated
	var validatedCount int
	err = r.db.QueryRow("SELECT COUNT(*) FROM ml_dataset WHERE is_validated = TRUE").Scan(&validatedCount)
	if err != nil {
		return nil, err
	}
	stats["validated_entries"] = validatedCount
	stats["unvalidated_entries"] = totalCount - validatedCount

	// Count by category
	categoryQuery := `
		SELECT category_id, category_name, COUNT(*) as count
		FROM ml_dataset
		GROUP BY category_id, category_name
		ORDER BY category_id
	`
	rows, err := r.db.Query(categoryQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categoryStats := make(map[string]int)
	for rows.Next() {
		var categoryID int
		var categoryName string
		var count int
		if err := rows.Scan(&categoryID, &categoryName, &count); err != nil {
			return nil, err
		}
		categoryStats[categoryName] = count
	}
	stats["by_category"] = categoryStats

	// Count by provider
	providerQuery := `
		SELECT provider, COUNT(*) as count
		FROM ml_dataset
		GROUP BY provider
	`
	rows, err = r.db.Query(providerQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	providerStats := make(map[string]int)
	for rows.Next() {
		var provider string
		var count int
		if err := rows.Scan(&provider, &count); err != nil {
			return nil, err
		}
		providerStats[provider] = count
	}
	stats["by_provider"] = providerStats

	return stats, nil
}

package repository

import (
	"database/sql"
	"fmt"

	"annotation-service/internal/models"

	_ "modernc.org/sqlite"
	"go.uber.org/zap"
)

// AnnotationRepository handles data storage
type AnnotationRepository struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewAnnotationRepository creates a new repository
func NewAnnotationRepository(dbPath string, logger *zap.Logger) (*AnnotationRepository, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	repo := &AnnotationRepository{
		db:     db,
		logger: logger,
	}

	if err := repo.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	logger.Info("Annotation repository initialized", zap.String("db_path", dbPath))

	return repo, nil
}

// migrate creates tables
func (r *AnnotationRepository) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS annotations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		message_id INTEGER,
		text TEXT NOT NULL,
		category_id INTEGER NOT NULL,
		category_name TEXT NOT NULL,
		justification TEXT,
		confidence REAL,
		annotated_at DATETIME NOT NULL,
		provider TEXT NOT NULL,
		model_version TEXT,
		is_validated BOOLEAN DEFAULT 0
	);

	CREATE INDEX IF NOT EXISTS idx_category_id ON annotations(category_id);
	CREATE INDEX IF NOT EXISTS idx_annotated_at ON annotations(annotated_at);
	CREATE INDEX IF NOT EXISTS idx_is_validated ON annotations(is_validated);

	CREATE TABLE IF NOT EXISTS jobs (
		id TEXT PRIMARY KEY,
		status TEXT NOT NULL,
		total_count INTEGER NOT NULL,
		processed_count INTEGER DEFAULT 0,
		failed_count INTEGER DEFAULT 0,
		created_at DATETIME NOT NULL,
		completed_at DATETIME,
		error_message TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_job_status ON jobs(status);
	`

	_, err := r.db.Exec(schema)
	return err
}

// SaveAnnotation saves a single annotation
func (r *AnnotationRepository) SaveAnnotation(ann *models.Annotation) error {
	query := `
		INSERT INTO annotations (
			message_id, text, category_id, category_name, justification,
			confidence, annotated_at, provider, model_version, is_validated
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.Exec(query,
		ann.MessageID,
		ann.Text,
		ann.Category,
		ann.CategoryName,
		ann.Justification,
		ann.Confidence,
		ann.AnnotatedAt,
		ann.Provider,
		ann.ModelVersion,
		ann.IsValidated,
	)

	if err != nil {
		return fmt.Errorf("failed to save annotation: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	ann.ID = id
	return nil
}

// GetAllAnnotations retrieves all annotations
func (r *AnnotationRepository) GetAllAnnotations() ([]*models.Annotation, error) {
	query := `
		SELECT id, message_id, text, category_id, category_name, justification,
		       confidence, annotated_at, provider, model_version, is_validated
		FROM annotations
		ORDER BY annotated_at DESC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query annotations: %w", err)
	}
	defer rows.Close()

	var annotations []*models.Annotation
	for rows.Next() {
		ann := &models.Annotation{}
		err := rows.Scan(
			&ann.ID,
			&ann.MessageID,
			&ann.Text,
			&ann.Category,
			&ann.CategoryName,
			&ann.Justification,
			&ann.Confidence,
			&ann.AnnotatedAt,
			&ann.Provider,
			&ann.ModelVersion,
			&ann.IsValidated,
		)
		if err != nil {
			r.logger.Error("Failed to scan annotation", zap.Error(err))
			continue
		}
		annotations = append(annotations, ann)
	}

	return annotations, nil
}

// GetAnnotationsByCategory retrieves annotations by category
func (r *AnnotationRepository) GetAnnotationsByCategory(categoryID int) ([]*models.Annotation, error) {
	query := `
		SELECT id, message_id, text, category_id, category_name, justification,
		       confidence, annotated_at, provider, model_version, is_validated
		FROM annotations
		WHERE category_id = ?
		ORDER BY annotated_at DESC
	`

	rows, err := r.db.Query(query, categoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to query annotations by category: %w", err)
	}
	defer rows.Close()

	var annotations []*models.Annotation
	for rows.Next() {
		ann := &models.Annotation{}
		err := rows.Scan(
			&ann.ID,
			&ann.MessageID,
			&ann.Text,
			&ann.Category,
			&ann.CategoryName,
			&ann.Justification,
			&ann.Confidence,
			&ann.AnnotatedAt,
			&ann.Provider,
			&ann.ModelVersion,
			&ann.IsValidated,
		)
		if err != nil {
			r.logger.Error("Failed to scan annotation", zap.Error(err))
			continue
		}
		annotations = append(annotations, ann)
	}

	return annotations, nil
}

// GetStats returns statistics about annotations
func (r *AnnotationRepository) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total count
	var total int
	err := r.db.QueryRow("SELECT COUNT(*) FROM annotations").Scan(&total)
	if err != nil {
		return nil, err
	}
	stats["total"] = total

	// By category
	query := `
		SELECT category_id, category_name, COUNT(*) as count
		FROM annotations
		GROUP BY category_id, category_name
		ORDER BY category_id
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byCategory := make(map[string]int)
	for rows.Next() {
		var catID int
		var catName string
		var count int
		if err := rows.Scan(&catID, &catName, &count); err != nil {
			continue
		}
		byCategory[catName] = count
	}
	stats["by_category"] = byCategory

	return stats, nil
}

// CreateJob creates a new annotation job
func (r *AnnotationRepository) CreateJob(job *models.Job) error {
	query := `
		INSERT INTO jobs (id, status, total_count, created_at)
		VALUES (?, ?, ?, ?)
	`

	_, err := r.db.Exec(query, job.ID, job.Status, job.TotalCount, job.CreatedAt)
	return err
}

// UpdateJob updates job progress
func (r *AnnotationRepository) UpdateJob(job *models.Job) error {
	query := `
		UPDATE jobs
		SET status = ?, processed_count = ?, failed_count = ?, completed_at = ?, error_message = ?
		WHERE id = ?
	`

	_, err := r.db.Exec(query, job.Status, job.ProcessedCount, job.FailedCount, job.CompletedAt, job.ErrorMessage, job.ID)
	return err
}

// GetJob retrieves a job by ID
func (r *AnnotationRepository) GetJob(jobID string) (*models.Job, error) {
	query := `
		SELECT id, status, total_count, processed_count, failed_count, created_at, completed_at, error_message
		FROM jobs
		WHERE id = ?
	`

	job := &models.Job{}
	err := r.db.QueryRow(query, jobID).Scan(
		&job.ID,
		&job.Status,
		&job.TotalCount,
		&job.ProcessedCount,
		&job.FailedCount,
		&job.CreatedAt,
		&job.CompletedAt,
		&job.ErrorMessage,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job not found")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	return job, nil
}

// Close closes the database connection
func (r *AnnotationRepository) Close() error {
	return r.db.Close()
}

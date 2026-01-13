package repository

import (
	"errors"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file" // Required for file source
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // PostgreSQL driver
	"go.uber.org/zap"
)

// NewPostgresDB establishes a new connection to the PostgreSQL database.
func NewPostgresDB(dataSourceName string, logger *zap.Logger) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", dataSourceName)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	logger.Info("Successfully connected to the database!")
	return db, nil
}

// MigrateDB runs database migrations.
func MigrateDB(db *sqlx.DB, logger *zap.Logger) {
	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	if err != nil {
		logger.Fatal("Couldn't get database instance for running migrations", zap.Error(err))
	}

	m, err := migrate.NewWithDatabaseInstance("file://migrations", "social_engineering_detector", driver)
	if err != nil {
		logger.Fatal("Couldn't create migrate instance", zap.Error(err))
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		logger.Fatal("Couldn't run database migration", zap.Error(err))
	}

	logger.Info("Database migration was run successfully")
}

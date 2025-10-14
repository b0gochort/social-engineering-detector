package repository

import (
	"errors"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file" // Required for file source
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/sirupsen/logrus"
)

// NewPostgresDB establishes a new connection to the PostgreSQL database.
func NewPostgresDB(dataSourceName string, log *logrus.Logger) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", dataSourceName)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	log.Info("Successfully connected to the database!")
	return db, nil
}

// MigrateDB runs database migrations.
func MigrateDB(db *sqlx.DB, log *logrus.Logger) {
	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	if err != nil {
		log.Fatalf("Couldn't get database instance for running migrations: %v", err)
	}

	m, err := migrate.NewWithDatabaseInstance("file://migrations", "social_engineering_detector", driver)
	if err != nil {
		log.Fatalf("Couldn't create migrate instance: %v", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatalf("Couldn't run database migration: %v", err)
	}

	log.Info("Database migration was run successfully")
}

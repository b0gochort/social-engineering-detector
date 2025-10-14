package main

import (
	"github.com/sirupsen/logrus"

	"backend/internal/repository"
	"backend/internal/server"
)

func main() {
	log := logrus.New()
	log.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	// Database connection
	db, err := repository.NewPostgresDB("user=postgres password=postgres dbname=social_engineering_detector sslmode=disable", log)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations
	repository.MigrateDB(db, log)

	// Initialize and run the server
	srv := server.NewServer(db, log)
	srv.Run(":8080")
}
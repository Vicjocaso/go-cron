package utils

import (
	"context"
	"database/sql"
	"go-cron/models"
	"log"
	"time"
)

// Global DB handle for connection pooling
var db *sql.DB

func InitDB(config *models.AppConfig) {
	var err error
	// The pgx driver is registered with the name "pgx".
	db, err = sql.Open("pgx", config.Database.DatabaseURI)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}

	// Configure connection pool settings.
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Ping the database to verify the connection.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		log.Fatalf("Database ping failed: %v\n", err)
	}
	log.Println("Database connection pool established successfully.")
}

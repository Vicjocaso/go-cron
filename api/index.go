package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

func EncodeOkReponse(w http.ResponseWriter, i interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(i)
}

func Handler(w http.ResponseWriter, r *http.Request) {
	// --- 1. Security Check ---
	cronSecret := os.Getenv("CRON_SECRET")
	if cronSecret == "" {
		log.Println("CRON_SECRET is not set. Aborting.")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	authHeader := r.Header.Get("authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") || strings.TrimPrefix(authHeader, "Bearer ") != cronSecret {
		log.Println("Unauthorized access attempt.")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	result := fmt.Sprintf("Hello from Go! %s", authHeader)
	fmt.Fprintf(w, "%s", result)
	EncodeOkReponse(w, result)
}

// import (
// 	"context"
// 	"database/sql"
// 	"encoding/json"
// 	"fmt"
// 	"io"
// 	"log"
// 	"net/http"
// 	"os"
// 	"strings"
// 	"time"
// 	// _ "github.com/jackc/pgx/v5/stdlib"
// )

// // DataPayload defines the structure of the data expected from the external API.
// // This should be customized to match the actual JSON response.
// type DataPayload struct {
// 	ID   int    `json:"id"`
// 	Name string `json:"name"`
// 	Data string `json:"data"`
// }

// // Global DB handle for connection pooling
// var db *sql.DB

// func init() {
// 	var err error
// 	dbURL := os.Getenv("DATABASE_URL")
// 	if dbURL == "" {
// 		log.Fatal("DATABASE_URL environment variable is not set")
// 	}

// 	// The pgx driver is registered with the name "pgx".
// 	db, err = sql.Open("pgx", dbURL)
// 	if err != nil {
// 		log.Fatalf("Unable to connect to database: %v\n", err)
// 	}

// 	// Configure connection pool settings.
// 	db.SetMaxOpenConns(5)
// 	db.SetMaxIdleConns(5)
// 	db.SetConnMaxLifetime(5 * time.Minute)

// 	// Ping the database to verify the connection.
// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	defer cancel()

// 	err = db.PingContext(ctx)
// 	if err != nil {
// 		log.Fatalf("Database ping failed: %v\n", err)
// 	}
// 	log.Println("Database connection pool established successfully.")
// }

// // Handler is the main entry point for the Vercel Serverless Function.
// func Handler(w http.ResponseWriter, r *http.Request) {
// 	// --- 1. Security Check ---
// 	cronSecret := os.Getenv("CRON_SECRET")
// 	if cronSecret == "" {
// 		log.Println("CRON_SECRET is not set. Aborting.")
// 		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
// 		return
// 	}

// 	authHeader := r.Header.Get("authorization")
// 	if !strings.HasPrefix(authHeader, "Bearer ") || strings.TrimPrefix(authHeader, "Bearer ") != cronSecret {
// 		log.Println("Unauthorized access attempt.")
// 		http.Error(w, "Unauthorized", http.StatusUnauthorized)
// 		return
// 	}

// 	// --- 2. Data Extraction ---
// 	dataSourceURL := os.Getenv("DATA_SOURCE_URL")
// 	if dataSourceURL == "" {
// 		log.Println("DATA_SOURCE_URL is not set. Aborting.")
// 		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
// 		return
// 	}

// 	client := http.Client{Timeout: 30 * time.Second}
// 	resp, err := client.Get(dataSourceURL)
// 	if err != nil {
// 		log.Printf("Error fetching data from source: %v", err)
// 		http.Error(w, "Failed to fetch data", http.StatusInternalServerError)
// 		return
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode != http.StatusOK {
// 		log.Printf("External API returned non-200 status: %d", resp.StatusCode)
// 		http.Error(w, "External API error", http.StatusBadGateway)
// 		return
// 	}

// 	body, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		log.Printf("Error reading response body: %v", err)
// 		http.Error(w, "Failed to read response", http.StatusInternalServerError)
// 		return
// 	}

// 	var payload DataPayload
// 	if err := json.Unmarshal(body, &payload); err != nil {
// 		log.Printf("Error unmarshaling JSON: %v", err)
// 		http.Error(w, "Invalid data format from source", http.StatusInternalServerError)
// 		return
// 	}

// 	// --- 3. Data Loading ---
// 	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
// 	defer cancel()

// 	// Use a parameterized query to prevent SQL injection.
// 	sqlStatement := `INSERT INTO your_table (id, name, data) VALUES ($1, $2, $3)`
// 	_, err = db.ExecContext(ctx, sqlStatement, payload.ID, payload.Name, payload.Data)
// 	if err != nil {
// 		log.Printf("Error inserting data into database: %v", err)
// 		http.Error(w, "Database insertion failed", http.StatusInternalServerError)
// 		return
// 	}

// 	// --- Success ---
// 	log.Println("Data processed and stored successfully.")
// 	w.WriteHeader(http.StatusOK)
// 	fmt.Fprintln(w, "Data processed successfully.")
// }

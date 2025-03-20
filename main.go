package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/raj-vidya-kender/echohttp/ui"
)

type requestData struct {
	ID        int64       `json:"id"`
	Timestamp time.Time   `json:"timestamp"`
	Data      string      `json:"data"`
	Headers   http.Header `json:"headers"`
}

// echoServer holds the application state
type echoServer struct {
	db *sql.DB
	mu sync.RWMutex
}

func (s *echoServer) initDB() error {
	log.Printf("[INFO] initializing database...")
	// Create the requests table if it doesn't exist
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS requests (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp DATETIME NOT NULL,
			data TEXT NOT NULL,
			headers TEXT NOT NULL
		)
	`)
	if err != nil {
		log.Printf("[ERROR] failed to initialize database: %v", err)
		return err
	}
	return nil
}

func (s *echoServer) handleRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		log.Printf("[ERROR] method not allowed: %s from %s", r.Method, r.RemoteAddr)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.Method == http.MethodGet {
		rows, err := s.db.Query("SELECT id, timestamp, data, headers FROM requests ORDER BY id DESC")
		if err != nil {
			if err == sql.ErrNoRows {
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode([]requestData{}); err != nil {
					log.Printf("[ERROR] failed to encode empty response: %v", err)
					http.Error(w, "Failed to encode response", http.StatusInternalServerError)
				}
				log.Printf("[INFO] get request from %s: no records found", r.RemoteAddr)
				return
			}
			log.Printf("[ERROR] database query failed: %v", err)
			http.Error(w, "Failed to query database", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var requests []requestData
		for rows.Next() {
			var req requestData
			var headersJSON string
			err := rows.Scan(&req.ID, &req.Timestamp, &req.Data, &headersJSON)
			if err != nil {
				log.Printf("[ERROR] failed to scan row: %v", err)
				http.Error(w, "Failed to scan row", http.StatusInternalServerError)
				return
			}

			err = json.Unmarshal([]byte(headersJSON), &req.Headers)
			if err != nil {
				log.Printf("[ERROR] failed to unmarshal headers: %v", err)
				http.Error(w, "Failed to unmarshal headers", http.StatusInternalServerError)
				return
			}

			requests = append(requests, req)
		}

		if err := rows.Err(); err != nil {
			log.Printf("[ERROR] error iterating rows: %v", err)
			http.Error(w, "Error iterating rows", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(requests); err != nil {
			log.Printf("[ERROR] failed to encode response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
		log.Printf("[INFO] get request from %s: retrieved %d requests", r.RemoteAddr, len(requests))
		return
	}

	// Handle POST request
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[ERROR] failed to read request body: %v", err)
		http.Error(w, "body couldn't be read", http.StatusBadRequest)
		return
	}

	// Convert headers to JSON for storage
	headersJSON, err := json.Marshal(r.Header)
	if err != nil {
		log.Printf("[ERROR] failed to marshal headers: %v", err)
		http.Error(w, "Error processing headers", http.StatusInternalServerError)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	_, err = s.db.Exec(`
		INSERT INTO requests (timestamp, data, headers)
		VALUES (?, ?, ?)
	`, time.Now(), string(body), string(headersJSON))
	if err != nil {
		log.Printf("[ERROR] failed to insert into database: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	log.Printf("[INFO] post request from %s: stored request with body size %d bytes", r.RemoteAddr, len(body))
	w.WriteHeader(http.StatusOK)
}

func main() {
	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "8025"
	}

	db, err := sql.Open("sqlite3", "echo.db")
	if err != nil {
		log.Fatalf("[ERROR] error opening database: %v", err)
	}
	defer db.Close()

	server := &echoServer{
		db: db,
	}
	if err := server.initDB(); err != nil {
		log.Fatalf("[ERROR] error initializing database: %v", err)
	}

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%s", port),
		Handler:           nil, // Will be set below
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Set up routes
	http.Handle("/", http.StripPrefix("/", http.FileServer(http.FS(ui.Assets()))))
	http.HandleFunc("/echo", server.handleRequests)

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("[INFO] server starting on port %s", port)
		serverErrors <- srv.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		log.Printf("[ERROR] server error: %v", err)

	case sig := <-shutdown:
		log.Printf("[INFO] received signal %v, initiating graceful shutdown...", sig)

		// Create a context with timeout for graceful shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Attempt graceful shutdown
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("[ERROR] could not stop server gracefully: %v", err)
			panic(err)
		}

		log.Printf("[INFO] server stopped gracefully")
	}
}

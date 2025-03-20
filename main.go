package main

import (
	"context"
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

	"github.com/raj-vidya-kender/echohttp/ui"
)

type requestData struct {
	Timestamp time.Time   `json:"timestamp"`
	Data      any         `json:"data"`
	Headers   http.Header `json:"headers"`
}

// echoServer holds the application state
type echoServer struct {
	requests []requestData
	mu       sync.RWMutex
}

func (s *echoServer) handleRequests(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGet(w, r)
	case http.MethodPost:
		s.handlePost(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *echoServer) handleGet(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return the requests in reverse chronological order
	requests := make([]requestData, len(s.requests))
	copy(requests, s.requests)
	for i, j := 0, len(requests)-1; i < j; i, j = i+1, j-1 {
		requests[i], requests[j] = requests[j], requests[i]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(requests)
}

func (s *echoServer) handlePost(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "body couldn't be read", http.StatusBadRequest)
		return
	}

	reqData := requestData{
		Timestamp: time.Now(),
		Data:      string(body),
		Headers:   r.Header,
	}

	s.mu.Lock()
	s.requests = append(s.requests, reqData)
	s.mu.Unlock()

	w.WriteHeader(http.StatusOK)
}

func main() {
	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "8025"
	}

	server := &echoServer{requests: make([]requestData, 0)}

	// Create a new HTTP server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: nil, // Will be set below
	}

	// Create a channel to listen for errors coming from the server
	serverErrors := make(chan error, 1)

	// Create a channel to listen for OS signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Set up routes
	http.Handle("/", http.StripPrefix("/", http.FileServer(http.FS(ui.Assets()))))
	http.HandleFunc("/echo", server.handleRequests)

	// Start the server in a goroutine
	go func() {
		log.Printf("Server starting on %s", srv.Addr)
		serverErrors <- srv.ListenAndServe()
	}()

	// Blocking select waiting for either a signal or server error
	select {
	case err := <-serverErrors:
		log.Printf("Server error: %v", err)

	case sig := <-shutdown:
		log.Printf("Received signal %v, initiating graceful shutdown...", sig)

		// Create a context with timeout for graceful shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Attempt graceful shutdown
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Could not stop server gracefully: %v", err)
			panic(err)
		}

		log.Println("Server stopped gracefully")
	}
}

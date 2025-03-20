package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
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

	// Serve static files from the ui/dist directory
	http.Handle("/", http.StripPrefix("/", http.FileServer(http.FS(ui.Assets()))))

	// API endpoint for both GET and POST
	http.HandleFunc("/echo", server.handleRequests)

	addr := fmt.Sprintf(":%s", port)
	log.Printf("Server starting on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

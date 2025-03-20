package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/raj-vidya-kender/echohttp/ui"
)

func setupTestServer(t *testing.T) (*echoServer, *sql.DB, func()) {
	// Create a temporary SQLite database for testing
	tmpDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	server := &echoServer{db: tmpDB}
	if err := server.initDB(); err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}

	cleanup := func() {
		tmpDB.Close()
	}

	return server, tmpDB, cleanup
}

func TestHandleGetEmptyDatabase(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/echo", nil)
	w := httptest.NewRecorder()

	server.handleRequests(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response []requestData
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(response) != 0 {
		t.Errorf("Expected empty response, got %d items", len(response))
	}
}

func TestHandlePostSuccess(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	testData := "test data"
	req := httptest.NewRequest(http.MethodPost, "/echo", strings.NewReader(testData))
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("X-Test-Header", "test-value")
	w := httptest.NewRecorder()

	server.handleRequests(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	// Verify the data was stored by making a GET request
	getReq := httptest.NewRequest(http.MethodGet, "/echo", nil)
	getW := httptest.NewRecorder()

	server.handleRequests(getW, getReq)

	var response []requestData
	if err := json.NewDecoder(getW.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(response) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(response))
	}

	if response[0].Data != testData {
		t.Errorf("Expected data %q, got %q", testData, response[0].Data)
	}

	if response[0].Headers.Get("X-Test-Header") != "test-value" {
		t.Errorf("Expected header X-Test-Header to be %q, got %q", "test-value", response[0].Headers.Get("X-Test-Header"))
	}
}

func TestHandlePostInvalidMethod(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPut, "/echo", nil)
	w := httptest.NewRecorder()

	server.handleRequests(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status code %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestUIAssets(t *testing.T) {
	// Create a test server
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Set up routes just like in main()
	mux := http.NewServeMux()
	mux.Handle("/", http.StripPrefix("/", http.FileServer(http.FS(ui.Assets()))))
	mux.HandleFunc("/echo", server.handleRequests)

	// Test cases for UI assets
	testCases := []struct {
		name         string
		path         string
		expectedCode int
		expectedType string
		bodyContains string
	}{
		{
			name:         "Index HTML",
			path:         "/",
			expectedCode: http.StatusOK,
			expectedType: "text/html",
			bodyContains: "<html",
		},
	}

	// Find the actual JS and CSS files in the assets directory
	assets, err := fs.ReadDir(ui.Assets(), "assets")
	if err != nil {
		t.Fatalf("Failed to read assets directory: %v", err)
	}

	for _, asset := range assets {
		name := asset.Name()
		switch {
		case strings.HasSuffix(name, ".js"):
			testCases = append(testCases, struct {
				name         string
				path         string
				expectedCode int
				expectedType string
				bodyContains string
			}{
				name:         "Assets JS",
				path:         "/assets/" + name,
				expectedCode: http.StatusOK,
				expectedType: "javascript",
				bodyContains: "const",
			})
		case strings.HasSuffix(name, ".css"):
			testCases = append(testCases, struct {
				name         string
				path         string
				expectedCode int
				expectedType string
				bodyContains string
			}{
				name:         "Assets CSS",
				path:         "/assets/" + name,
				expectedCode: http.StatusOK,
				expectedType: "text/css",
				bodyContains: "{",
			})
		}
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			if w.Code != tc.expectedCode {
				t.Errorf("Expected status code %d, got %d", tc.expectedCode, w.Code)
				t.Logf("Response body: %s", w.Body.String())
			}

			contentType := w.Header().Get("Content-Type")
			if !strings.Contains(contentType, tc.expectedType) {
				t.Errorf("Expected Content-Type containing %q, got %q", tc.expectedType, contentType)
			}

			body, err := io.ReadAll(w.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}

			if !strings.Contains(string(body), tc.bodyContains) {
				t.Errorf("Expected response body to contain %q", tc.bodyContains)
				t.Logf("Got body: %s", string(body))
			}
		})
	}
}

func TestConcurrentRequests(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Number of concurrent requests to make
	numRequests := 10

	// Create a channel to collect results
	results := make(chan error, numRequests)

	// Start concurrent POST requests
	for i := 0; i < numRequests; i++ {
		go func(i int) {
			data := fmt.Sprintf("test data %d", i)
			req := httptest.NewRequest(http.MethodPost, "/echo", strings.NewReader(data))
			w := httptest.NewRecorder()

			server.handleRequests(w, req)

			if w.Code != http.StatusOK {
				results <- fmt.Errorf("request %d: expected status code %d, got %d", i, http.StatusOK, w.Code)
				return
			}

			results <- nil
		}(i)
	}

	// Collect results
	for i := 0; i < numRequests; i++ {
		if err := <-results; err != nil {
			t.Error(err)
		}
	}

	// Verify all requests were stored
	req := httptest.NewRequest(http.MethodGet, "/echo", nil)
	w := httptest.NewRecorder()

	server.handleRequests(w, req)

	var response []requestData
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(response) != numRequests {
		t.Errorf("Expected %d items, got %d", numRequests, len(response))
	}
}

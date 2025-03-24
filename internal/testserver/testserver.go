// internal/testserver/testserver.go
package testserver

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

// ServerConfig represents the configuration for a test server
type ServerConfig struct {
	ID      string
	Port    int
	Latency struct {
		Min int // Minimum latency in ms
		Max int // Maximum latency in ms
	}
	ErrorRate float64 // 0.0 to 1.0 representing probability of errors
}

// RequestStats tracks request statistics for the server
type RequestStats struct {
	TotalRequests int       `json:"totalRequests"`
	Successes     int       `json:"successes"`
	Failures      int       `json:"failures"`
	LastRequest   time.Time `json:"lastRequest"`
}

// TestServer simulates a backend server for testing
type TestServer struct {
	Config     ServerConfig
	Stats      RequestStats
	statsMutex sync.Mutex
	server     *http.Server
}

// NewTestServer creates a new test server
func NewTestServer(config ServerConfig) *TestServer {
	if config.Latency.Min <= 0 {
		config.Latency.Min = 50
	}
	if config.Latency.Max <= config.Latency.Min {
		config.Latency.Max = config.Latency.Min + 200
	}

	return &TestServer{
		Config: config,
		Stats: RequestStats{
			LastRequest: time.Now(),
		},
	}
}

// Start begins the test server
func (ts *TestServer) Start() error {
	mux := http.NewServeMux()

	// Basic endpoint for handling requests
	mux.HandleFunc("/", ts.handleRequest)

	// Health check endpoint
	mux.HandleFunc("/health", ts.handleHealth)

	// Stats endpoint
	mux.HandleFunc("/stats", ts.handleStats)

	ts.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", ts.Config.Port),
		Handler: mux,
	}

	log.Printf("Starting test server %s on port %d", ts.Config.ID, ts.Config.Port)

	return ts.server.ListenAndServe()
}

// Stop shuts down the server
func (ts *TestServer) Stop() error {
	if ts.server != nil {
		return ts.server.Close()
	}
	return nil
}

// handleRequest is the main handler for all incoming requests
func (ts *TestServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	ts.statsMutex.Lock()
	ts.Stats.TotalRequests++
	ts.Stats.LastRequest = time.Now()
	ts.statsMutex.Unlock()

	// Simulate random latency
	latency := ts.Config.Latency.Min + rand.Intn(ts.Config.Latency.Max-ts.Config.Latency.Min)
	time.Sleep(time.Duration(latency) * time.Millisecond)

	// Check if we should generate an error response
	if rand.Float64() < ts.Config.ErrorRate {
		ts.statsMutex.Lock()
		ts.Stats.Failures++
		ts.statsMutex.Unlock()

		http.Error(w, "Simulated server error", http.StatusInternalServerError)
		return
	}

	ts.statsMutex.Lock()
	ts.Stats.Successes++
	ts.statsMutex.Unlock()

	// Send a success response
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"server":  ts.Config.ID,
		"time":    time.Now().Format(time.RFC3339),
		"latency": latency,
		"path":    r.URL.Path,
	}

	json.NewEncoder(w).Encode(response)
}

// handleHealth serves the health check endpoint
func (ts *TestServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Always return status OK for the health check
	response := map[string]string{
		"status": "ok",
		"server": ts.Config.ID,
	}

	json.NewEncoder(w).Encode(response)
}

// handleStats serves the statistics endpoint
func (ts *TestServer) handleStats(w http.ResponseWriter, r *http.Request) {
	ts.statsMutex.Lock()
	stats := ts.Stats
	ts.statsMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// StartTestServers starts multiple test servers with the given configurations
func StartTestServers(configs []ServerConfig) []*TestServer {
	var servers []*TestServer

	for _, config := range configs {
		server := NewTestServer(config)

		// Start the server in a goroutine
		go func() {
			if err := server.Start(); err != nil && err != http.ErrServerClosed {
				log.Printf("Server %s error: %v", server.Config.ID, err)
			}
		}()

		servers = append(servers, server)
	}

	return servers
}

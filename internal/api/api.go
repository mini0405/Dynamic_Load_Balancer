// internal/api/api.go
package api

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"load-balancer/internal/events"
	"load-balancer/internal/lb"
	"load-balancer/internal/metrics"
	"load-balancer/internal/server"
)

// API handles all the dashboard API endpoints
type API struct {
	ServerManager  *server.Manager
	Balancer       *lb.Balancer
	CircuitBreaker *lb.CircuitBreakerCoordinator
	MetricsManager *metrics.MetricsManager
	EventSystem    *events.EventSystem
}

// Config represents the load balancer configuration that can be updated via API
type Config struct {
	UseIPHash         bool `json:"useIPHash"`
	UseStickySessions bool `json:"useStickySessions"`
}

// ServerToggleResponse is returned when toggling a server's status
type ServerToggleResponse struct {
	ID      string `json:"id"`
	Enabled bool   `json:"enabled"`
}

// TestResponse is returned from the test endpoint
type TestResponse struct {
	Server       string    `json:"server"`
	ResponseTime int       `json:"responseTime"`
	Timestamp    time.Time `json:"timestamp"`
}

// NewAPI creates a new API handler
func NewAPI(mgr *server.Manager, balancer *lb.Balancer, cb *lb.CircuitBreakerCoordinator,
	mm *metrics.MetricsManager, es *events.EventSystem) *API {
	return &API{
		ServerManager:  mgr,
		Balancer:       balancer,
		CircuitBreaker: cb,
		MetricsManager: mm,
		EventSystem:    es,
	}
}

// RegisterHandlers registers all API endpoints with the given mux
func (api *API) RegisterHandlers(mux *http.ServeMux) {
	// Server info endpoints
	mux.HandleFunc("/api/servers", api.getServers)
	mux.HandleFunc("/api/servers/", api.handleServerRequests)

	// Configuration endpoint
	mux.HandleFunc("/api/config", api.updateConfig)

	// Test endpoint
	mux.HandleFunc("/api/test", api.handleTest)

	// Packet flow endpoint
	mux.HandleFunc("/api/packets", api.getPackets)

	// Metrics endpoint (enhanced)
	mux.HandleFunc("/api/metrics", api.MetricsManager.Handler())

	// Server-sent events for realtime updates
	mux.HandleFunc("/api/events", api.handleEvents)
}

// getServers returns information about all servers
func (api *API) getServers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	servers := api.ServerManager.GetAllServers()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(servers)
}

// handleServerRequests manages all endpoints under /api/servers/...
func (api *API) handleServerRequests(w http.ResponseWriter, r *http.Request) {
	// Extract the server ID and action from the URL path
	// URL format: /api/servers/{serverID}/{action}
	path := r.URL.Path[len("/api/servers/"):]

	// Find the first slash after the server ID
	serverID := path
	action := ""

	for i, c := range path {
		if c == '/' {
			serverID = path[:i]
			action = path[i+1:]
			break
		}
	}

	// Validate the server ID
	serverFound := false
	var targetServer *server.Server

	servers := api.ServerManager.GetAllServers()
	for _, srv := range servers {
		if srv.ID == serverID {
			serverFound = true
			targetServer = srv
			break
		}
	}

	if !serverFound {
		http.Error(w, "Server not found", http.StatusNotFound)
		return
	}

	// Handle different actions
	switch action {
	case "toggle":
		api.toggleServer(w, r, targetServer)
	case "reset":
		api.resetServer(w, r, targetServer)
	default:
		http.Error(w, "Unknown action", http.StatusBadRequest)
	}
}

// toggleServer enables or disables a server
func (api *API) toggleServer(w http.ResponseWriter, r *http.Request, srv *server.Server) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	wasEnabled := srv.PingStatus
	srv.PingStatus = !srv.PingStatus

	if srv.PingStatus {
		// Restoring connectivity: close breaker and reset counters
		srv.CircuitBreakerState = server.CBStateClosed
		srv.FailureCount = 0
		srv.TrialSuccessCount = 0
	} else {
		// Disabling the server: trip breaker so it is skipped by balancer
		srv.CircuitBreakerState = server.CBStateOpen
		srv.OpenSince = time.Now()
		srv.FailureCount = 0
		srv.TrialSuccessCount = 0
		for server.GetActiveRequests(srv) > 0 {
			server.EndRequest(srv)
		}
	}

	statusText := "enabled"
	eventType := events.SuccessEvent
	if !srv.PingStatus {
		statusText = "disabled"
		eventType = events.WarningEvent
	}

	if wasEnabled == srv.PingStatus {
		statusText = "unchanged"
		eventType = events.InfoEvent
	}

	api.EventSystem.Publish(eventType, fmt.Sprintf("Server %s %s", srv.ID, statusText))

	response := ServerToggleResponse{
		ID:      srv.ID,
		Enabled: srv.PingStatus,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// resetServer resets a server's circuit breaker state
func (api *API) resetServer(w http.ResponseWriter, r *http.Request, srv *server.Server) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Reset the circuit breaker state to closed
	srv.CircuitBreakerState = server.CBStateClosed
	srv.FailureCount = 0
	srv.TrialSuccessCount = 0
	srv.PingStatus = true
	for server.GetActiveRequests(srv) > 0 {
		server.EndRequest(srv)
	}

	// Send event notification
	api.EventSystem.Publish(events.InfoEvent, fmt.Sprintf("Server %s circuit breaker reset", srv.ID))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "reset",
		"id":     srv.ID,
	})
}

// getPackets returns the recent packet flow events for visualization.
func (api *API) getPackets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limit := 100
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	events := api.MetricsManager.GetPacketHistory(limit)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Events []metrics.PacketEvent `json:"events"`
	}{
		Events: events,
	})
}

// updateConfig updates the load balancer configuration
func (api *API) updateConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var config Config
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update the balancer configuration
	api.Balancer.UseIPHash = config.UseIPHash
	api.Balancer.UseStickySessions = config.UseStickySessions

	// Send event notification
	api.EventSystem.Publish(events.InfoEvent, fmt.Sprintf(
		"Load balancer config updated: IP Hash %s, Sticky Sessions %s",
		boolToString(config.UseIPHash),
		boolToString(config.UseStickySessions)))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

// handleTest simulates a request to test the load balancer
func (api *API) handleTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	totalServers := len(api.ServerManager.GetAllServers())
	if totalServers == 0 {
		http.Error(w, "No server available", http.StatusServiceUnavailable)
		api.EventSystem.Publish(events.ErrorEvent, "Test request failed: no servers registered")
		return
	}

	priority := lb.ExtractPriority(r)
	requestID := api.MetricsManager.GeneratePacketID()
	attempted := make(map[string]bool, totalServers)
	var lastErr error

	for attempt := 0; attempt < totalServers; attempt++ {
		srv := api.Balancer.PickServerWithExclude(r, attempted)
		if srv == nil {
			break
		}
		attempted[srv.ID] = true

		active := server.BeginRequest(srv)
		dispatchEvent := metrics.PacketEvent{
			RequestID:      requestID,
			Attempt:        attempt + 1,
			Priority:       priority,
			ServerID:       srv.ID,
			ServerAddress:  fmt.Sprintf("%s:%d", srv.Address, srv.Port),
			Status:         "dispatch",
			Timestamp:      time.Now(),
			ActiveRequests: active,
		}
		api.MetricsManager.RecordAndBroadcastPacketEvent(api.EventSystem, dispatchEvent)

		if active > lb.BusyThreshold {
			activeAfter := server.EndRequest(srv)
			rerouteEvent := dispatchEvent
			rerouteEvent.Status = "rerouted"
			rerouteEvent.Reason = "busy"
			rerouteEvent.Timestamp = time.Now()
			rerouteEvent.ActiveRequests = activeAfter
			api.MetricsManager.RecordAndBroadcastPacketEvent(api.EventSystem, rerouteEvent)
			continue
		}

		processing := time.Duration(50+rand.Intn(200)) * time.Millisecond
		time.Sleep(processing)
		responseMs := float64(processing.Milliseconds())

		failureChance := 0.05 + srv.ErrorRate
		if failureChance > 0.4 {
			failureChance = 0.4
		}

		if rand.Float64() < failureChance {
			activeAfter := server.EndRequest(srv)
			api.CircuitBreaker.RecordFailure(srv)
			api.MetricsManager.RecordRequest(srv.ID, responseMs, true)

			failureEvent := metrics.PacketEvent{
				RequestID:      requestID,
				Attempt:        attempt + 1,
				Priority:       priority,
				ServerID:       srv.ID,
				ServerAddress:  fmt.Sprintf("%s:%d", srv.Address, srv.Port),
				Status:         "failed",
				Reason:         "simulated failure",
				Timestamp:      time.Now(),
				ResponseTime:   responseMs,
				ActiveRequests: activeAfter,
			}
			api.MetricsManager.RecordAndBroadcastPacketEvent(api.EventSystem, failureEvent)
			api.EventSystem.Publish(events.WarningEvent, fmt.Sprintf("Simulated failure on %s for request %s", srv.ID, requestID))
			lastErr = fmt.Errorf("simulated failure")
			continue
		}

		api.CircuitBreaker.RecordSuccess(srv)
		api.MetricsManager.RecordRequest(srv.ID, responseMs, false)
		activeAfter := server.EndRequest(srv)

		successEvent := metrics.PacketEvent{
			RequestID:      requestID,
			Attempt:        attempt + 1,
			Priority:       priority,
			ServerID:       srv.ID,
			ServerAddress:  fmt.Sprintf("%s:%d", srv.Address, srv.Port),
			Status:         "completed",
			Timestamp:      time.Now(),
			ResponseTime:   responseMs,
			ActiveRequests: activeAfter,
		}
		api.MetricsManager.RecordAndBroadcastPacketEvent(api.EventSystem, successEvent)

		response := TestResponse{
			Server:       srv.ID,
			ResponseTime: int(responseMs),
			Timestamp:    time.Now(),
		}

		api.EventSystem.Publish(events.InfoEvent, fmt.Sprintf("Test request %s served by %s (%s priority)", requestID, srv.ID, priority))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("all servers busy")
	}
	api.EventSystem.Publish(events.ErrorEvent, fmt.Sprintf("Test request %s failed: %v", requestID, lastErr))
	http.Error(w, "No server available", http.StatusServiceUnavailable)
}

// handleEvents sets up a Server-Sent Events connection
func (api *API) handleEvents(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create a channel for this client by subscribing to the event system
	subscriber := api.EventSystem.Subscribe()

	// Make sure to unsubscribe when the client disconnects
	defer api.EventSystem.Unsubscribe(subscriber)

	// Send welcome event
	api.EventSystem.Publish(events.InfoEvent, "Connected to event stream")

	// Create notification channel for client disconnection
	notify := r.Context().Done()

	// Keep connection open
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	for {
		select {
		case <-notify:
			return // Client disconnected
		case msg, ok := <-subscriber:
			if !ok {
				return // Channel closed
			}

			// Write the event to the response
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		case <-time.After(30 * time.Second):
			// Send a keepalive comment
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}

// boolToString converts a boolean to "enabled" or "disabled"
func boolToString(b bool) string {
	if b {
		return "enabled"
	}
	return "disabled"
}

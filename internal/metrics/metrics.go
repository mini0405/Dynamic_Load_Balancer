// internal/metrics/metrics.go
package metrics

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"load-balancer/internal/events"
	"load-balancer/internal/server"
)

// LBMetrics tracks load balancer statistics
type LBMetrics struct {
	TotalRequests       int64                   `json:"totalRequests"`
	RequestsPerServer   map[string]int64        `json:"requestsPerServer"`
	AvgResponseTime     float64                 `json:"avgResponseTime"`
	ResponseTimeHistory []ResponseTimeDataPoint `json:"responseTimeHistory"`
	ErrorRate           float64                 `json:"errorRate"`
	LastErrors          []ErrorEvent            `json:"lastErrors"`
	mutex               sync.RWMutex
}

// ResponseTimeDataPoint represents a data point for response time tracking
type ResponseTimeDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"` // in milliseconds
}

// ErrorEvent represents an error event
type ErrorEvent struct {
	Timestamp time.Time `json:"timestamp"`
	ServerID  string    `json:"serverId"`
	Message   string    `json:"message"`
}

// MetricsManager handles collecting and reporting metrics
type MetricsManager struct {
	Metrics       LBMetrics
	ServerManager *server.Manager
	mutex         sync.RWMutex

	// Circular buffer settings for response time history
	maxHistoryPoints int

	// Packet flow tracking
	packetHistory    []PacketEvent
	maxPacketHistory int
	packetCounter    uint64
}

// NewMetricsManager creates a new metrics manager
func NewMetricsManager(srvMgr *server.Manager) *MetricsManager {
	return &MetricsManager{
		Metrics: LBMetrics{
			RequestsPerServer:   make(map[string]int64),
			ResponseTimeHistory: make([]ResponseTimeDataPoint, 0, 100),
			LastErrors:          make([]ErrorEvent, 0, 10),
		},
		ServerManager:    srvMgr,
		maxHistoryPoints: 100, // Keep last 100 data points for response time
		packetHistory:    make([]PacketEvent, 0, 200),
		maxPacketHistory: 200, // Track last 200 packet events
		packetCounter:    0,
	}
}

// RecordRequest records a request to a server
func (mm *MetricsManager) RecordRequest(serverID string, responseTime float64, isError bool) {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	mm.Metrics.TotalRequests++

	// Update requests per server count
	if _, exists := mm.Metrics.RequestsPerServer[serverID]; !exists {
		mm.Metrics.RequestsPerServer[serverID] = 0
	}
	mm.Metrics.RequestsPerServer[serverID]++

	// Update response time history
	dataPoint := ResponseTimeDataPoint{
		Timestamp: time.Now(),
		Value:     responseTime,
	}

	// Add to history with circular buffer behavior
	if len(mm.Metrics.ResponseTimeHistory) >= mm.maxHistoryPoints {
		// Shift array left (remove oldest)
		mm.Metrics.ResponseTimeHistory = append(mm.Metrics.ResponseTimeHistory[1:], dataPoint)
	} else {
		mm.Metrics.ResponseTimeHistory = append(mm.Metrics.ResponseTimeHistory, dataPoint)
	}

	// Recalculate average response time
	total := 0.0
	for _, dp := range mm.Metrics.ResponseTimeHistory {
		total += dp.Value
	}
	mm.Metrics.AvgResponseTime = total / float64(len(mm.Metrics.ResponseTimeHistory))

	// Track error rate and recent errors
	if isError {
		errorEvent := ErrorEvent{
			Timestamp: time.Now(),
			ServerID:  serverID,
			Message:   "Request failed",
		}

		// Keep only most recent 10 errors
		if len(mm.Metrics.LastErrors) >= 10 {
			mm.Metrics.LastErrors = append(mm.Metrics.LastErrors[1:], errorEvent)
		} else {
			mm.Metrics.LastErrors = append(mm.Metrics.LastErrors, errorEvent)
		}

		// Update error rate
		mm.Metrics.ErrorRate = float64(len(mm.Metrics.LastErrors)) / float64(mm.maxHistoryPoints)
	}
}

// PacketEvent captures the lifecycle of a request being routed through the balancer.
type PacketEvent struct {
	RequestID      string    `json:"requestId"`
	Attempt        int       `json:"attempt"`
	Priority       string    `json:"priority"`
	ServerID       string    `json:"serverId"`
	ServerAddress  string    `json:"serverAddress"`
	Status         string    `json:"status"`
	Reason         string    `json:"reason,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
	ResponseTime   float64   `json:"responseTime,omitempty"`
	ActiveRequests int64     `json:"activeRequests"`
}

// GeneratePacketID returns a unique identifier for a routed request.
func (mm *MetricsManager) GeneratePacketID() string {
	id := atomic.AddUint64(&mm.packetCounter, 1)
	return fmt.Sprintf("pkt-%d", id)
}

// RecordPacketEvent stores a packet event in the rolling history.
func (mm *MetricsManager) RecordPacketEvent(evt PacketEvent) {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	if len(mm.packetHistory) >= mm.maxPacketHistory {
		mm.packetHistory = append(mm.packetHistory[1:], evt)
	} else {
		mm.packetHistory = append(mm.packetHistory, evt)
	}
}

// RecordAndBroadcastPacketEvent stores the event and pushes it to subscribers via the event system.
func (mm *MetricsManager) RecordAndBroadcastPacketEvent(es *events.EventSystem, evt PacketEvent) {
	mm.RecordPacketEvent(evt)

	if es == nil {
		return
	}

	payload, err := json.Marshal(evt)
	if err != nil {
		return
	}
	es.Publish(events.PacketEvent, string(payload))
}

// GetPacketHistory returns the most recent packet events up to the requested limit.
func (mm *MetricsManager) GetPacketHistory(limit int) []PacketEvent {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()

	if limit <= 0 || limit > len(mm.packetHistory) {
		limit = len(mm.packetHistory)
	}

	start := len(mm.packetHistory) - limit
	if start < 0 {
		start = 0
	}

	result := make([]PacketEvent, len(mm.packetHistory[start:]))
	copy(result, mm.packetHistory[start:])
	return result
}

// Handler returns an HTTP handler for serving metrics
func (mm *MetricsManager) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mm.mutex.RLock()
		defer mm.mutex.RUnlock()

		w.Header().Set("Content-Type", "application/json")

		// Combine server metrics with load balancer metrics
		servers := mm.ServerManager.GetAllServers()

		// Create combined response
		response := struct {
			LoadBalancer LBMetrics        `json:"loadBalancer"`
			Servers      []*server.Server `json:"servers"`
		}{
			LoadBalancer: mm.Metrics,
			Servers:      servers,
		}

		// Encode and send
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode metrics", http.StatusInternalServerError)
			return
		}
	}
}

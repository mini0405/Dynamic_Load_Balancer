// internal/metrics/metrics.go
package metrics

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

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

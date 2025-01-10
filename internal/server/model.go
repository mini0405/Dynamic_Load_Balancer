// internal/server/model.go
package server

import "time"

// CBState represents the circuit breaker state for a server.
type CBState int

const (
	CBStateClosed CBState = iota
	CBStateOpen
	CBStateHalfOpen
)

// Server represents a backend server in the pool.
type Server struct {
	ID      string
	Address string
	Port    int

	// Metrics relevant for health checks
	CPUUsage     float64
	MemUsage     float64
	ResponseTime float64
	ErrorRate    float64
	PingStatus   bool

	// Derived from metrics
	HealthScore   float64
	CurrentWeight float64

	// Circuit Breaker fields
	CircuitBreakerState CBState
	FailureCount        int
	TrialSuccessCount   int
	OpenSince           time.Time
}

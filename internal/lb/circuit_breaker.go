// internal/lb/circuit_breaker.go
package lb

import (
	"time"

	"load-balancer/internal/server"
)

// CircuitBreakerSettings defines thresholds and timeouts.
type CircuitBreakerSettings struct {
	FailureThreshold int           // consecutive failures to trip CB
	CooldownPeriod   time.Duration // how long to stay Open
	TrialRequests    int           // requests in HalfOpen before closing
}

// CircuitBreakerCoordinator manages circuit breaker transitions for servers.
type CircuitBreakerCoordinator struct {
	Settings      CircuitBreakerSettings
	ServerManager *server.Manager
}

// NewCircuitBreakerCoordinator creates a new CB coordinator.
func NewCircuitBreakerCoordinator(mgr *server.Manager, settings CircuitBreakerSettings) *CircuitBreakerCoordinator {
	return &CircuitBreakerCoordinator{
		Settings:      settings,
		ServerManager: mgr,
	}
}

// RecordFailure increments failure count and potentially opens the breaker.
func (cbc *CircuitBreakerCoordinator) RecordFailure(srv *server.Server) {
	srv.FailureCount++
	if srv.CircuitBreakerState == server.CBStateClosed &&
		srv.FailureCount >= cbc.Settings.FailureThreshold {
		srv.CircuitBreakerState = server.CBStateOpen
		srv.OpenSince = time.Now()
	} else if srv.CircuitBreakerState == server.CBStateHalfOpen {
		// If in HalfOpen and a failure occurs, go back to Open
		srv.CircuitBreakerState = server.CBStateOpen
		srv.OpenSince = time.Now()
	}
}

// RecordSuccess resets the failure count. Also transitions from HalfOpen -> Closed
// if enough success requests have been made.
func (cbc *CircuitBreakerCoordinator) RecordSuccess(srv *server.Server) {
	if srv.CircuitBreakerState == server.CBStateClosed {
		srv.FailureCount = 0
		return
	}

	if srv.CircuitBreakerState == server.CBStateHalfOpen {
		srv.TrialSuccessCount++
		if srv.TrialSuccessCount >= cbc.Settings.TrialRequests {
			// Move to closed
			srv.CircuitBreakerState = server.CBStateClosed
			srv.FailureCount = 0
			srv.TrialSuccessCount = 0
		}
	}
}

// MonitorServers runs periodically to move servers from Open -> HalfOpen after cooldown.
func (cbc *CircuitBreakerCoordinator) MonitorServers() {
	for {
		servers := cbc.ServerManager.GetAllServers()
		for _, srv := range servers {
			if srv.CircuitBreakerState == server.CBStateOpen {
				if time.Since(srv.OpenSince) >= cbc.Settings.CooldownPeriod {
					srv.CircuitBreakerState = server.CBStateHalfOpen
					srv.TrialSuccessCount = 0
				}
			}
		}
		time.Sleep(1 * time.Second)
	}
}

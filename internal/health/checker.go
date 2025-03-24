package health

import (
	"context"
	"time"

	"load-balancer/internal/server"
)

// Checker periodically pulls metrics from servers
// and recalculates each server's health score & weight.
type Checker struct {
	Interval      time.Duration
	ServerManager *server.Manager
	doneCh        chan bool
}

// NewChecker creates a new health checker.
func NewChecker(interval time.Duration, mgr *server.Manager) *Checker {
	return &Checker{
		Interval:      interval,
		ServerManager: mgr,
		doneCh:        make(chan bool),
	}
}

// Start begins the periodic health-check in a goroutine.
func (hc *Checker) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(hc.Interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				hc.checkServers()
			case <-ctx.Done():
				// Context canceled or timed out
				return
			case <-hc.doneCh:
				// Manually stopped
				return
			}
		}
	}()
}

// Stop terminates the health checker goroutine.
func (hc *Checker) Stop() {
	close(hc.doneCh)
}

// checkServers pulls updated metrics and recalculates weights.
func (hc *Checker) checkServers() {
	servers := hc.ServerManager.GetAllServers()

	// Weighted formula coefficients (example weights for each metric)
	alpha := 0.25   // CPU usage importance
	beta := 0.20    // Memory usage importance
	gamma := 0.25   // Response time importance
	delta := 0.25   // Error rate importance
	epsilon := 0.05 // Ping status importance

	// 1) Fetch updated metrics and calculate health scores
	for _, srv := range servers {
		server.FetchMetrics(srv) // Fetch all metrics at once

		// Calculate health score:
		// H = α(1 - CPU) + β(1 - MEM) + γ(1 - Resp) + δ(1 - Error) + ε*Ping
		// Assumes CPU, MEM, Resp, Error are normalized in [0..1]
		H := alpha*(1-srv.CPUUsage) +
			beta*(1-srv.MemUsage) +
			gamma*(1-srv.ResponseTime/500.0) + // Normalizing response time (max 500ms)
			delta*(1-srv.ErrorRate) +
			epsilon*boolToFloat64(srv.PingStatus)

		srv.HealthScore = H
	}

	// 2) Normalize health scores into weights
	totalHealth := 0.0
	for _, srv := range servers {
		totalHealth += srv.HealthScore
	}

	if totalHealth > 0 {
		for _, srv := range servers {
			srv.CurrentWeight = srv.HealthScore / totalHealth
		}
	} else {
		// Edge case: If total health <= 0, set all weights to 0
		for _, srv := range servers {
			srv.CurrentWeight = 0
		}
	}

	// 3) Update the server manager with recalculated weights
	hc.ServerManager.UpdateServers(servers)
}

// boolToFloat64 converts a boolean value to float64 (1 for true, 0 for false).
func boolToFloat64(value bool) float64 {
	if value {
		return 1.0
	}
	return 0.0
}

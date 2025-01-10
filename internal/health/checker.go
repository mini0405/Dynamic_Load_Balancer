// internal/health/checker.go
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

// checkServers pulls updated metrics and recalculates weights
func (hc *Checker) checkServers() {
	servers := hc.ServerManager.GetAllServers()

	// Weighted formula coefficients (example)
	alpha := 0.25   // CPU usage importance
	beta := 0.20    // Memory usage importance
	gamma := 0.25   // Response time importance
	delta := 0.25   // Error rate importance
	epsilon := 0.05 // Ping status importance

	// 1) Get updated metrics for each server
	for _, srv := range servers {
		cpuUsage := fetchCPUUsage(srv)
		memUsage := fetchMemoryUsage(srv)
		respTime := fetchResponseTime(srv)
		errorRate := fetchErrorRate(srv)
		pingStatus := fetchPingStatus(srv)

		// 2) Calculate health score:
		//    H = α(1 - CPU) + β(1 - MEM) + γ(1 - Resp) + δ(1 - Error) + ε*Ping
		//    (assuming CPU/MEM/Resp/Error are normalized in [0..1])
		H := alpha*(1-cpuUsage) +
			beta*(1-memUsage) +
			gamma*(1-respTime) +
			delta*(1-errorRate) +
			epsilon*pingStatus

		srv.HealthScore = H
	}

	// 3) Convert each HealthScore to CurrentWeight = H / sum(H)
	sumH := 0.0
	for _, srv := range servers {
		sumH += srv.HealthScore
	}
	if sumH > 0 {
		for _, srv := range servers {
			srv.CurrentWeight = srv.HealthScore / sumH
		}
	} else {
		// Edge case: if sumH <= 0, set all weights to 0
		for _, srv := range servers {
			srv.CurrentWeight = 0
		}
	}

	// 4) Update the manager
	hc.ServerManager.UpdateServers(servers)
}

// The below functions emulate fetching real metrics (e.g., via HTTP calls).
// Replace them with actual logic in production.

func fetchCPUUsage(srv *server.Server) float64 {
	return srv.CPUUsage
}

func fetchMemoryUsage(srv *server.Server) float64 {
	return srv.MemUsage
}

func fetchResponseTime(srv *server.Server) float64 {
	return srv.ResponseTime
}

func fetchErrorRate(srv *server.Server) float64 {
	return srv.ErrorRate
}

func fetchPingStatus(srv *server.Server) float64 {
	if srv.PingStatus {
		return 1
	}
	return 0
}

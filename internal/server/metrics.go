package server

import (
	"math/rand"
)

// NormalizeCPUUsage converts raw CPU usage to a [0..1] scale.
func NormalizeCPUUsage(rawCPU float64) float64 {
	return rawCPU / 100.0
}

// NormalizeMemoryUsage converts raw memory usage to a [0..1] scale.
func NormalizeMemoryUsage(rawMem float64) float64 {
	return rawMem / 100.0
}

// SimulateResponseTime generates a simulated response time for a server.
func SimulateResponseTime() float64 {
	// Randomized response time in milliseconds (example range: 50ms to 500ms)
	return float64(50 + rand.Intn(450))
}

// SimulatePingStatus checks if the server is reachable (1 for success, 0 for failure).
func SimulatePingStatus() float64 {
	// Simulate a 95% chance of success
	if rand.Float64() < 0.95 {
		return 1
	}
	return 0
}

// SimulateErrorRate generates a simulated error rate for the server.
func SimulateErrorRate() float64 {
	// Simulated error rate (example range: 0% to 5%)
	return rand.Float64() * 0.05
}

// FetchMetrics updates all metrics for a given server.
func FetchMetrics(srv *Server) {
	// Simulate fetching metrics and updating the server object
	srv.CPUUsage = NormalizeCPUUsage(50 + rand.Float64()*50)    // Simulated CPU usage: 50% - 100%
	srv.MemUsage = NormalizeMemoryUsage(30 + rand.Float64()*70) // Simulated memory usage: 30% - 100%
	srv.ResponseTime = SimulateResponseTime()                   // Random response time
	srv.PingStatus = SimulatePingStatus() == 1                  // Random ping status
	srv.ErrorRate = SimulateErrorRate()                         // Random error rate
}

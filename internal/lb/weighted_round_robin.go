// internal/lb/weighted_round_robin.go
package lb

import (
	"math/rand"
	"sync"
	"time"

	"load-balancer/internal/server"
)

// WeightedRoundRobin implements weighted RR to pick a server based on CurrentWeight.
type WeightedRoundRobin struct {
	mu            sync.Mutex
	ServerManager *server.Manager
	randSource    *rand.Rand
}

// NewWeightedRoundRobin creates a WeightedRoundRobin instance.
func NewWeightedRoundRobin(mgr *server.Manager) *WeightedRoundRobin {
	return &WeightedRoundRobin{
		ServerManager: mgr,
		randSource:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// PickServer picks a server based on their CurrentWeight, skipping those in Open CB state.
func (w *WeightedRoundRobin) PickServer() *server.Server {
	w.mu.Lock()
	defer w.mu.Unlock()

	servers := w.ServerManager.GetAllServers()
	if len(servers) == 0 {
		return nil // no servers
	}

	// Calculate total weight among servers that are not in Open state
	totalWeight := 0.0
	for _, srv := range servers {
		if srv.CircuitBreakerState != server.CBStateOpen {
			totalWeight += srv.CurrentWeight
		}
	}

	if totalWeight <= 0 {
		return nil // no server has a positive weight
	}

	randVal := w.randSource.Float64() * totalWeight
	runningSum := 0.0
	for _, srv := range servers {
		if srv.CircuitBreakerState == server.CBStateOpen {
			continue
		}
		runningSum += srv.CurrentWeight
		if randVal < runningSum {
			return srv
		}
	}

	// Fallback if something unexpected happens
	return nil
}

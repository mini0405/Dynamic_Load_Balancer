// internal/lb/weighted_round_robin.go
package lb

import (
	"sync"

	"load-balancer/internal/server"
)

// WeightedRoundRobin implements weighted RR to pick a server based on CurrentWeight.
type WeightedRoundRobin struct {
	mu             sync.Mutex
	ServerManager  *server.Manager
	currentWeights map[string]float64
	fallbackIndex  int
}

// NewWeightedRoundRobin creates a WeightedRoundRobin instance.
func NewWeightedRoundRobin(mgr *server.Manager) *WeightedRoundRobin {
	return &WeightedRoundRobin{
		ServerManager:  mgr,
		currentWeights: make(map[string]float64),
	}
}

// PickServer picks a server using a smooth weighted round robin strategy.
func (w *WeightedRoundRobin) PickServer(exclude map[string]bool) *server.Server {
	w.mu.Lock()
	defer w.mu.Unlock()

	servers := w.ServerManager.GetAllServers()
	if len(servers) == 0 {
		return nil
	}

	type candidate struct {
		srv    *server.Server
		weight float64
	}

	candidates := make([]candidate, 0, len(servers))
	totalWeight := 0.0

	// Track which servers currently exist so we can prune removed entries.
	existing := make(map[string]struct{}, len(servers))

	for _, srv := range servers {
		existing[srv.ID] = struct{}{}

		if exclude != nil && exclude[srv.ID] {
			w.currentWeights[srv.ID] = 0
			continue
		}

		if !srv.PingStatus {
			delete(w.currentWeights, srv.ID)
			continue
		}

		if srv.CircuitBreakerState != server.CBStateClosed {
			delete(w.currentWeights, srv.ID)
			continue
		}

		weight := srv.CurrentWeight
		if weight < 0 {
			weight = 0
		}

		if weight == 0 {
			w.currentWeights[srv.ID] = 0
		} else {
			totalWeight += weight
		}

		candidates = append(candidates, candidate{srv: srv, weight: weight})
	}

	// Remove bookkeeping for servers that no longer exist.
	for id := range w.currentWeights {
		if _, ok := existing[id]; !ok {
			delete(w.currentWeights, id)
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	// If total weight collapsed to zero (all servers reported zero weight),
	// fall back to a simple round robin over the active set.
	if totalWeight <= 0 {
		for _, cand := range candidates {
			w.currentWeights[cand.srv.ID] = 0
		}

		server := candidates[w.fallbackIndex%len(candidates)].srv
		w.fallbackIndex++
		return server
	}

	var chosen *server.Server
	var maxWeight float64
	seenAny := false

	for _, cand := range candidates {
		w.currentWeights[cand.srv.ID] += cand.weight

		if !seenAny || w.currentWeights[cand.srv.ID] > maxWeight {
			maxWeight = w.currentWeights[cand.srv.ID]
			chosen = cand.srv
			seenAny = true
		}
	}

	if chosen == nil {
		return nil
	}

	w.currentWeights[chosen.ID] -= totalWeight
	w.fallbackIndex = 0

	return chosen
}

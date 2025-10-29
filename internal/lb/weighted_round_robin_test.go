package lb

import (
	"testing"

	"load-balancer/internal/server"
)

func newTestManagerWithWeights(weights ...float64) *server.Manager {
	servers := make([]*server.Server, 0, len(weights))
	for i, w := range weights {
		servers = append(servers, &server.Server{
			ID:                  "srv-" + string(rune('A'+i)),
			CurrentWeight:       w,
			PingStatus:          true,
			CircuitBreakerState: server.CBStateClosed,
		})
	}
	return server.NewManager(servers)
}

func TestWeightedRoundRobin_DistributesEvenly(t *testing.T) {
	mgr := newTestManagerWithWeights(0.33, 0.33, 0.34)
	wrr := NewWeightedRoundRobin(mgr)

	counts := map[string]int{}
	for i := 0; i < 6; i++ {
		srv := wrr.PickServer(nil)
		if srv == nil {
			t.Fatalf("expected a server on iteration %d", i)
		}
		counts[srv.ID]++
	}

	for id, want := range map[string]int{"srv-A": 2, "srv-B": 2, "srv-C": 2} {
		if got := counts[id]; got != want {
			t.Fatalf("expected %s to receive %d picks, got %d (counts=%v)", id, want, got, counts)
		}
	}
}

func TestWeightedRoundRobin_HonorsRelativeWeights(t *testing.T) {
	mgr := newTestManagerWithWeights(0.5, 0.3, 0.2)
	wrr := NewWeightedRoundRobin(mgr)

	counts := map[string]int{}
	for i := 0; i < 10; i++ {
		srv := wrr.PickServer(nil)
		if srv == nil {
			t.Fatalf("expected a server on iteration %d", i)
		}
		counts[srv.ID]++
	}

	wantCounts := map[string]int{"srv-A": 5, "srv-B": 3, "srv-C": 2}
	for id, want := range wantCounts {
		if counts[id] != want {
			t.Fatalf("expected %s to receive %d picks, got %d (counts=%v)", id, want, counts[id], counts)
		}
	}
}

func TestWeightedRoundRobin_FallbackRoundRobinWhenNoWeights(t *testing.T) {
	mgr := newTestManagerWithWeights(0, 0, 0)
	wrr := NewWeightedRoundRobin(mgr)

	counts := map[string]int{}
	for i := 0; i < 6; i++ {
		srv := wrr.PickServer(nil)
		if srv == nil {
			t.Fatalf("expected a server on iteration %d", i)
		}
		counts[srv.ID]++
	}

	for id, want := range map[string]int{"srv-A": 2, "srv-B": 2, "srv-C": 2} {
		if got := counts[id]; got != want {
			t.Fatalf("expected %s to receive %d picks, got %d (counts=%v)", id, want, got, counts)
		}
	}
}

func TestWeightedRoundRobin_Exclude(t *testing.T) {
	mgr := newTestManagerWithWeights(0.6, 0.4)
	wrr := NewWeightedRoundRobin(mgr)

	exclude := map[string]bool{"srv-A": true}
	srv := wrr.PickServer(exclude)
	if srv == nil {
		t.Fatalf("expected a server even with exclusions")
	}
	if srv.ID != "srv-B" {
		t.Fatalf("expected srv-B to be selected when srv-A excluded, got %s", srv.ID)
	}
}

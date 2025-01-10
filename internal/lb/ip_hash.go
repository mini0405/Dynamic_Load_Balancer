// internal/lb/ip_hash.go
package lb

import (
	"hash/crc32"
	"sync"

	"load-balancer/internal/server"
)

// IPHash implements basic IP-based hashing to pick a consistent server.
type IPHash struct {
	mu            sync.Mutex
	ServerManager *server.Manager
}

// NewIPHash returns a new IPHash struct.
func NewIPHash(mgr *server.Manager) *IPHash {
	return &IPHash{ServerManager: mgr}
}

// GetServerForIP returns a server for a given IP based on crc32 hashing.
func (ih *IPHash) GetServerForIP(ip string) *server.Server {
	ih.mu.Lock()
	defer ih.mu.Unlock()

	servers := ih.ServerManager.GetAllServers()
	if len(servers) == 0 {
		return nil
	}

	hashVal := crc32.ChecksumIEEE([]byte(ip))
	index := int(hashVal) % len(servers)
	chosen := servers[index]
	if chosen.CircuitBreakerState != server.CBStateClosed {
		return nil
	}
	return chosen
}

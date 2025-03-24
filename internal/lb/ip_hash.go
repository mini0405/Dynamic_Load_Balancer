// internal/lb/ip_hash.go
package lb

import (
	"hash/crc32"
	"sync"

	"load-balancer/internal/server"
)

// IPHash implements basic IP-based hashing to pick a consistent server.
// using mutex will help me prevent the race condition on that particular serverManager
// Thread safe is absolutely required in this scenario

type IPHash struct {
	mu            sync.Mutex
	ServerManager *server.Manager
}

// NewIPHash returns a new IPHash struct
// takes a reference to the servermanager
//mgr *server.Manager is a pointer to the server manager 
// Since the server manager is very large, it is better to pass it by reference
// return type is a new IpHash with the address of the new struct
func NewIPHash(mgr *server.Manager) *IPHash {
	return &IPHash{ServerManager: mgr}
}

// GetServerForIP returns a server for a given IP based on crc32 hashing.
// this is the core logic
// Taking a new server out for each new IP as per client
// using a standardized CRC to measure the hash
func (ih *IPHash) GetServerForIP(ip string) *server.Server {
	ih.mu.Lock() // initiated a thread safe lock
	defer ih.mu.Unlock() 
	// Defering to make sure that the lock is always released to avoid the deadlock state

	servers := ih.ServerManager.GetAllServers() // fetching all the servers
	if len(servers) == 0 {
		return nil // if there are no servers, we return nil
	}

	hashVal := crc32.ChecksumIEEE([]byte(ip)) // creating a hashvalue for a particular IP
	index := int(hashVal) % len(servers) // selected a server index based on the hashvalue, since using modulus we wont go out of bounds
	chosen := servers[index] // once we got the index we select the server and return it as chosen
	if chosen.CircuitBreakerState != server.CBStateClosed {
		return nil // if the server is not closed, we return nil
	}
	return chosen
}

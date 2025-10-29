// internal/lb/balancer.go
package lb

import (
	"load-balancer/internal/server"
	"net/http"
	"strings"
	"sync"
)

const BusyThreshold int64 = 5

// Balancer orchestrates the load-balancing process.
type Balancer struct {
	mu               sync.Mutex
	ServerManager    *server.Manager
	WRR              *WeightedRoundRobin
	IPHasher         *IPHash
	StickySessionMgr *StickySessions

	UseStickySessions bool
	UseIPHash         bool
}

// NewBalancer creates a new Balancer instance.
func NewBalancer(mgr *server.Manager, wrr *WeightedRoundRobin, ipHash *IPHash, sticky *StickySessions) *Balancer {
	return &Balancer{
		ServerManager:    mgr,
		WRR:              wrr,
		IPHasher:         ipHash,
		StickySessionMgr: sticky,
	}
}

// PickServer chooses which server should handle the request.
func (b *Balancer) PickServer(r *http.Request) *server.Server {
	return b.pickServerInternal(r, nil)
}

// PickServerWithExclude chooses a server while excluding any IDs present in the provided map.
func (b *Balancer) PickServerWithExclude(r *http.Request, exclude map[string]bool) *server.Server {
	return b.pickServerInternal(r, exclude)
}

func (b *Balancer) pickServerInternal(r *http.Request, exclude map[string]bool) *server.Server {
	b.mu.Lock()
	defer b.mu.Unlock()

	// 1. Check sticky session
	sessionID := extractSessionID(r)
	if b.UseStickySessions && sessionID != "" {
		if srv := b.StickySessionMgr.GetServerForSession(sessionID); srv != nil {
			// If the sticky server is healthy (Closed), return it.
			if srv.CircuitBreakerState == server.CBStateClosed {
				if exclude == nil || !exclude[srv.ID] {
					return srv
				}
			}
		}
	}

	// 2. If IP Hash is enabled, pick server based on IP.
	if b.UseIPHash {
		clientIP := extractClientIP(r)
		if srv := b.IPHasher.GetServerForIP(clientIP); srv != nil {
			// If the IP-hashed server is healthy, bind session (if using sticky)
			if srv.CircuitBreakerState == server.CBStateClosed {
				if exclude == nil || !exclude[srv.ID] {
					if b.UseStickySessions && sessionID != "" {
						b.StickySessionMgr.BindSessionToServer(sessionID, srv)
					}
					return srv
				}
			}
		}
	}

	// 3. Fallback to Weighted Round Robin
	chosen := b.WRR.PickServer(exclude)
	if chosen == nil {
		// All servers might be in Open state or no servers exist
		return nil
	}

	// If using sticky sessions, bind the session
	if b.UseStickySessions && sessionID != "" {
		b.StickySessionMgr.BindSessionToServer(sessionID, chosen)
	}

	return chosen
}

// extractSessionID is a simple example to read session ID from a cookie.
func extractSessionID(r *http.Request) string {
	cookie, err := r.Cookie("session_id")
	if err == nil {
		return cookie.Value
	}
	return ""
}

// extractClientIP retrieves the client's IP address.
func extractClientIP(r *http.Request) string {
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		// Fallback to r.RemoteAddr (which includes port)
		ip = r.RemoteAddr
		if idx := strings.LastIndex(ip, ":"); idx != -1 {
			ip = ip[:idx]
		}
	}
	return ip
}

// internal/lb/sticky_sessions.go
package lb

import (
	"sync"

	"load-balancer/internal/server"
)

// StickySessions maintains a mapping of sessionID -> *Server
// This is a simple in-memory approach.
type StickySessions struct {
	mu            sync.Mutex
	sessionToSrv  map[string]*server.Server
	ServerManager *server.Manager
}

// NewStickySessions creates a StickySessions instance.
func NewStickySessions(mgr *server.Manager) *StickySessions {
	return &StickySessions{
		sessionToSrv:  make(map[string]*server.Server),
		ServerManager: mgr,
	}
}

// GetServerForSession returns the server for a given session, if healthy.
func (ss *StickySessions) GetServerForSession(sessionID string) *server.Server {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	srv, exists := ss.sessionToSrv[sessionID]
	if !exists {
		return nil
	}
	// Check if still healthy
	if srv.CircuitBreakerState != server.CBStateClosed {
		return nil
	}
	return srv
}

// BindSessionToServer maps a session ID to a particular server.
func (ss *StickySessions) BindSessionToServer(sessionID string, srv *server.Server) {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	ss.sessionToSrv[sessionID] = srv
}

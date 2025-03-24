// internal/lb/sticky_sessions.go
package lb

import (
	"sync"

	"load-balancer/internal/server"
)

// StickySessions maintains a mapping of sessionID -> *Server
// This is a simple in-memory approach.
type StickySessions struct {
	mu            sync.Mutex // ensuring thread safe actions
	sessionToSrv  map[string]*server.Server // creating a map for future mapping between session ids and servers
	ServerManager *server.Manager // a reference to the server manager
}

// NewStickySessions creates a StickySessions instance.
func NewStickySessions(mgr *server.Manager) *StickySessions { 
	return &StickySessions{
		sessionToSrv:  make(map[string]*server.Server), // empty session to server map
		ServerManager: mgr, // Links the ServerManager to allow access to backend server details.
	}
}

// GetServerForSession returns the server for a given session, if healthy.
// Core logic for the sticky session
// Asks for a session ID 
// retrieves the server assigned to a particular session ID
func (ss *StickySessions) GetServerForSession(sessionID string) *server.Server {
	ss.mu.Lock()
	defer ss.mu.Unlock() // to avoid deadlock
	// defer occurs after the lock is removed to avoid race condition

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

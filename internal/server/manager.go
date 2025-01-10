// internal/server/manager.go
package server

import "sync"

// Manager holds the list of servers and provides concurrency-safe access.
type Manager struct {
	mu      sync.RWMutex
	servers []*Server
}

// NewManager creates a new Manager instance.
func NewManager(servers []*Server) *Manager {
	return &Manager{servers: servers}
}

// GetAllServers returns a copy of the current slice of servers (thread-safe).
func (m *Manager) GetAllServers() []*Server {
	m.mu.RLock()
	defer m.mu.RUnlock()

	serversCopy := make([]*Server, len(m.servers))
	copy(serversCopy, m.servers)
	return serversCopy
}

// UpdateServers updates the entire server list (thread-safe).
func (m *Manager) UpdateServers(updated []*Server) {
	m.mu.Lock()
	defer m.mu.Unlock()

	newServers := make([]*Server, len(updated))
	copy(newServers, updated)
	m.servers = newServers
}

// AddServer dynamically adds a new server to the pool.
func (m *Manager) AddServer(s *Server) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.servers = append(m.servers, s)
}

// RemoveServer removes a server from the pool by ID.
func (m *Manager) RemoveServer(serverID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var newServers []*Server
	for _, srv := range m.servers {
		if srv.ID != serverID {
			newServers = append(newServers, srv)
		}
	}
	m.servers = newServers
}

package server

import "sync/atomic"

// BeginRequest increments the active request counter for a server and
// returns the current number of in-flight requests.
func BeginRequest(srv *Server) int64 {
	if srv == nil {
		return 0
	}
	return atomic.AddInt64(&srv.ActiveRequests, 1)
}

// EndRequest decrements the active request counter for a server and returns
// the updated number of in-flight requests.
func EndRequest(srv *Server) int64 {
	if srv == nil {
		return 0
	}
	return atomic.AddInt64(&srv.ActiveRequests, -1)
}

// GetActiveRequests returns the current number of in-flight requests
// for the provided server.
func GetActiveRequests(srv *Server) int64 {
	if srv == nil {
		return 0
	}
	return atomic.LoadInt64(&srv.ActiveRequests)
}

package ratelimiter

import (
	"sync"
	"time"
)

type limiter struct {
	mu              sync.Mutex
	rateLimit       int
	lastRequestTime time.Time
}

// NewLimiter creates a new rate limiter with the given rate limit.
func NewLimiter(rateLimit int) *limiter {
	return &limiter{
		rateLimit: rateLimit,
	}
}

// Allow checks if a request is allowed based on the rate limit.
func (l *limiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.lastRequestTime.IsZero() {
		l.lastRequestTime = time.Now()
		return true
	}

	if time.Since(l.lastRequestTime) >= time.Second {
		l.lastRequestTime = time.Now()
		return true
	}

	return false
}

// SetRateLimit sets a new rate limit for the limiter.
func (l *limiter) SetRateLimit(rateLimit int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.rateLimit = rateLimit
}

// GetRateLimit returns the current rate limit of the limiter.
func (l *limiter) GetRateLimit() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.rateLimit == 0 {
		return 1
	}
	return l.rateLimit
}

// GetLastRequestTime returns the time of the last request.
func (l *limiter) GetLastRequestTime() time.Time {
	l.mu.Lock()
	defer l.mu.Unlock()
	for l.lastRequestTime.IsZero() {
		return time.Time{}
	}
	return l.lastRequestTime
	
}


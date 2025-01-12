// internal/config/config.go
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds the entire LB configuration
type Config struct {
	LBPort              int
	Servers             []ServerConfig
	HealthCheckInterval time.Duration
	UseIPHash           bool
	UseStickySessions   bool
	CircuitBreaker      CircuitBreakerConfig
}

// ServerConfig represents each backend server’s config
type ServerConfig struct {
	ID      string
	Address string
	Port    int
}

// CircuitBreakerConfig for controlling circuit breaker thresholds
type CircuitBreakerConfig struct {
	FailureThreshold int
	CooldownPeriod   time.Duration
	TrialRequests    int
}

// LoadConfig loads config from environment variables or from defaults
func LoadConfig() (*Config, error) {
	// Example: read LB_PORT from env
	lbPortStr := os.Getenv("LB_PORT")
	lbPort, err := strconv.Atoi(lbPortStr)
	if err != nil || lbPort == 0 {
		lbPort = 8080 // default if not set
	}

	cfg := &Config{
		LBPort:              lbPort,
		HealthCheckInterval: 5 * time.Second, // default 5 seconds
		UseIPHash:           false,
		UseStickySessions:   false,
		CircuitBreaker: CircuitBreakerConfig{
			FailureThreshold: 3,
			CooldownPeriod:   10 * time.Second,
			TrialRequests:    2,
		},
		Servers: []ServerConfig{
			{
				ID:      "server-1",
				Address: "localhost",
				Port:    9001,
			},
			{
				ID:      "server-2",
				Address: "localhost",
				Port:    9002,
			},
			// We can add more servers if needed
		},
	}

	fmt.Printf("[CONFIG] LB Port=%d, Servers=%v\n", cfg.LBPort, cfg.Servers)
	return cfg, nil
}

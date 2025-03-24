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
	StartTestServers    bool // Whether to start test servers
}

// ServerConfig represents each backend server's config
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
	// Read LB_PORT from env
	lbPortStr := os.Getenv("LB_PORT")
	lbPort, err := strconv.Atoi(lbPortStr)
	if err != nil || lbPort == 0 {
		lbPort = 8080 // default if not set
	}

	// Read USE_IP_HASH from env
	useIPHashStr := os.Getenv("USE_IP_HASH")
	useIPHash := false
	if useIPHashStr == "true" || useIPHashStr == "1" {
		useIPHash = true
	}

	// Read USE_STICKY_SESSIONS from env
	useStickySessStr := os.Getenv("USE_STICKY_SESSIONS")
	useStickySessions := true // default to true
	if useStickySessStr == "false" || useStickySessStr == "0" {
		useStickySessions = false
	}

	// Read START_TEST_SERVERS from env
	startTestServersStr := os.Getenv("START_TEST_SERVERS")
	startTestServers := true // default to true for easy testing
	if startTestServersStr == "false" || startTestServersStr == "0" {
		startTestServers = false
	}

	// Read FAILURE_THRESHOLD from env
	failureThresholdStr := os.Getenv("FAILURE_THRESHOLD")
	failureThreshold, err := strconv.Atoi(failureThresholdStr)
	if err != nil || failureThreshold == 0 {
		failureThreshold = 3 // default
	}

	// Read COOLDOWN_PERIOD from env
	cooldownPeriodStr := os.Getenv("COOLDOWN_PERIOD")
	cooldownPeriod, err := strconv.Atoi(cooldownPeriodStr)
	if err != nil || cooldownPeriod == 0 {
		cooldownPeriod = 10 // default 10 seconds
	}

	// Read TRIAL_REQUESTS from env
	trialRequestsStr := os.Getenv("TRIAL_REQUESTS")
	trialRequests, err := strconv.Atoi(trialRequestsStr)
	if err != nil || trialRequests == 0 {
		trialRequests = 2 // default
	}

	// Read HEALTH_CHECK_INTERVAL from env
	healthCheckIntervalStr := os.Getenv("HEALTH_CHECK_INTERVAL")
	healthCheckInterval, err := strconv.Atoi(healthCheckIntervalStr)
	if err != nil || healthCheckInterval == 0 {
		healthCheckInterval = 5 // default 5 seconds
	}

	cfg := &Config{
		LBPort:              lbPort,
		HealthCheckInterval: time.Duration(healthCheckInterval) * time.Second,
		UseIPHash:           useIPHash,
		UseStickySessions:   useStickySessions,
		StartTestServers:    startTestServers,
		CircuitBreaker: CircuitBreakerConfig{
			FailureThreshold: failureThreshold,
			CooldownPeriod:   time.Duration(cooldownPeriod) * time.Second,
			TrialRequests:    trialRequests,
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
			// Add more servers if needed
		},
	}

	fmt.Printf("[CONFIG] Load Balancer Port: %d\n", cfg.LBPort)
	fmt.Printf("[CONFIG] IP Hash: %v\n", cfg.UseIPHash)
	fmt.Printf("[CONFIG] Sticky Sessions: %v\n", cfg.UseStickySessions)
	fmt.Printf("[CONFIG] Start Test Servers: %v\n", cfg.StartTestServers)
	fmt.Printf("[CONFIG] Health Check Interval: %v\n", cfg.HealthCheckInterval)
	fmt.Printf("[CONFIG] Circuit Breaker: Failure Threshold=%d, Cooldown=%v, Trial Requests=%d\n",
		cfg.CircuitBreaker.FailureThreshold,
		cfg.CircuitBreaker.CooldownPeriod,
		cfg.CircuitBreaker.TrialRequests)

	return cfg, nil
}

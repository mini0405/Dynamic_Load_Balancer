package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"load-balancer/internal/api"
	"load-balancer/internal/config"
	"load-balancer/internal/dashboard"
	"load-balancer/internal/events"
	"load-balancer/internal/health"
	"load-balancer/internal/lb"
	"load-balancer/internal/metrics"
	"load-balancer/internal/server"
	"load-balancer/internal/testserver"
)

func main() {
	// 1. Loading configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Unable to load config: %v", err)
	}

	// 2. Initializing the server manager
	srvMgr := initServerManager(cfg)

	// 3. Create event system for real-time notifications
	eventSystem := events.NewEventSystem(100) // Keep last 100 events

	// 4. Create metrics manager for tracking load balancer performance
	metricsManager := metrics.NewMetricsManager(srvMgr)

	// 5. Created Weighted Round Robin, IP hash, sticky sessions
	wrr := lb.NewWeightedRoundRobin(srvMgr)
	ipHash := lb.NewIPHash(srvMgr)
	stickyMgr := lb.NewStickySessions(srvMgr)

	// 6. Created Balancer
	balancer := lb.NewBalancer(srvMgr, wrr, ipHash, stickyMgr)
	balancer.UseIPHash = cfg.UseIPHash
	balancer.UseStickySessions = cfg.UseStickySessions

	// 7. Setting up the circuit breaker
	cbSettings := lb.CircuitBreakerSettings{
		FailureThreshold: cfg.CircuitBreaker.FailureThreshold,
		CooldownPeriod:   cfg.CircuitBreaker.CooldownPeriod,
		TrialRequests:    cfg.CircuitBreaker.TrialRequests,
	}
	cbCoordinator := lb.NewCircuitBreakerCoordinator(srvMgr, cbSettings)

	// Starting a background goroutine to monitor open circuits
	go cbCoordinator.MonitorServers()

	// 8. Starting the health checker
	checker := health.NewChecker(cfg.HealthCheckInterval, srvMgr)
	healthCtx, healthCancel := context.WithCancel(context.Background())
	checker.Start(healthCtx)

	// Log startup information
	eventSystem.Publish(events.InfoEvent, "Load balancer starting up")
	eventSystem.Publish(events.InfoEvent, fmt.Sprintf("Using IP Hash: %v, Sticky Sessions: %v",
		cfg.UseIPHash, cfg.UseStickySessions))

	// 9. Setup HTTP server to handle incoming requests
	mux := http.NewServeMux()

	// 9a. Load balancer endpoint
	mux.HandleFunc("/lb/", func(w http.ResponseWriter, r *http.Request) {
		// Strip the /lb prefix from the URL
		r.URL.Path = r.URL.Path[3:]
		if r.URL.Path == "" {
			r.URL.Path = "/"
		}

		// Picking up a server via balancer
		chosenSrv := balancer.PickServer(r)
		if chosenSrv == nil {
			// No available server
			eventSystem.Publish(events.ErrorEvent, "Request failed: No healthy servers available")
			http.Error(w, "Service Unavailable (no healthy servers)", http.StatusServiceUnavailable)
			return
		}

		// Forward request to chosen server
		proxyRequest(chosenSrv, w, r, cbCoordinator, metricsManager, eventSystem)
	})

	// 9b. Setup the dashboard API endpoints
	apiHandler := api.NewAPI(srvMgr, balancer, cbCoordinator, metricsManager, eventSystem)
	apiHandler.RegisterHandlers(mux)

	// 9c. Setup the dashboard UI
	mux.HandleFunc("/", dashboard.Handler(srvMgr))

	// 10. Start test servers if enabled
	var testServers []*testserver.TestServer
	if cfg.StartTestServers {
		testServerConfigs := []testserver.ServerConfig{
			{
				ID:        "server-1",
				Port:      9001,
				Latency:   struct{ Min, Max int }{Min: 50, Max: 150},
				ErrorRate: 0.01,
			},
			{
				ID:        "server-2",
				Port:      9002,
				Latency:   struct{ Min, Max int }{Min: 100, Max: 300},
				ErrorRate: 0.05,
			},
		}

		testServers = testserver.StartTestServers(testServerConfigs)
		log.Println("Started test servers")
		eventSystem.Publish(events.SuccessEvent, "Test servers started successfully")
	}

	// 11. Create and start the HTTP server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.LBPort),
		Handler: mux,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Load Balancer listening on port %d...", cfg.LBPort)
		log.Printf("Dashboard available at http://localhost:%d/", cfg.LBPort)
		eventSystem.Publish(events.SuccessEvent, fmt.Sprintf("Load balancer listening on port %d", cfg.LBPort))

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// 12. Wait for interrupt signal to gracefully shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down load balancer...")
	eventSystem.Publish(events.InfoEvent, "Load balancer shutting down...")

	healthCancel() // stop the health checker

	ctxTimeout, cancelTimeout := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelTimeout()

	if err := srv.Shutdown(ctxTimeout); err != nil {
		log.Fatalf("HTTP server Shutdown error: %v", err)
	}

	// Stop test servers if they were started
	for _, ts := range testServers {
		if err := ts.Stop(); err != nil {
			log.Printf("Error stopping test server %s: %v", ts.Config.ID, err)
		}
	}

	log.Println("Load balancer stopped.")
	eventSystem.Publish(events.InfoEvent, "Load balancer stopped")
}

// initServerManager initializes the server manager with servers read from config.
func initServerManager(cfg *config.Config) *server.Manager {
	var servers []*server.Server
	for _, s := range cfg.Servers {
		servers = append(servers, &server.Server{
			ID:                  s.ID,
			Address:             s.Address,
			Port:                s.Port,
			CircuitBreakerState: server.CBStateClosed,
			// Initial placeholders for metrics:
			CPUUsage:     0.1,
			MemUsage:     0.1,
			ResponseTime: 0.1,
			ErrorRate:    0.0,
			PingStatus:   true,
		})
	}
	return server.NewManager(servers)
}

// proxyRequest forwards a request to the chosen server.
func proxyRequest(srv *server.Server, w http.ResponseWriter, r *http.Request,
	cbc *lb.CircuitBreakerCoordinator, mm *metrics.MetricsManager, es *events.EventSystem) {

	startTime := time.Now()

	// Building the destination URL
	url := fmt.Sprintf("http://%s:%d%s", srv.Address, srv.Port, r.URL.Path)

	// Create a new HTTP request
	req, err := http.NewRequest(r.Method, url, r.Body)
	if err != nil {
		cbc.RecordFailure(srv)
		mm.RecordRequest(srv.ID, 0, true)
		es.Publish(events.ErrorEvent, fmt.Sprintf("Bad request to backend server %s: %v", srv.ID, err))
		http.Error(w, "Bad request to backend", http.StatusBadGateway)
		return
	}

	// Copy headers from the original request
	for k, vv := range r.Header {
		for _, v := range vv {
			req.Header.Add(k, v)
		}
	}

	// Create an HTTP client with timeout
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		cbc.RecordFailure(srv)
		responseTime := time.Since(startTime).Milliseconds()
		mm.RecordRequest(srv.ID, float64(responseTime), true)
		es.Publish(events.ErrorEvent, fmt.Sprintf("Request to %s failed: %v", srv.ID, err))
		http.Error(w, "Backend request failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Mark success in circuit breaker
	cbc.RecordSuccess(srv)

	// Record metrics
	responseTime := time.Since(startTime).Milliseconds()
	mm.RecordRequest(srv.ID, float64(responseTime), resp.StatusCode >= 500)

	// Log successful request
	if resp.StatusCode < 400 {
		es.Publish(events.InfoEvent, fmt.Sprintf("Request to %s completed in %dms",
			srv.ID, responseTime))
	} else {
		es.Publish(events.WarningEvent, fmt.Sprintf("Request to %s returned status %d",
			srv.ID, resp.StatusCode))
	}

	// Copy response headers
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)

	// Copy response body
	if _, copyErr := io.Copy(w, resp.Body); copyErr != nil {
		log.Printf("Error copying response body: %v", copyErr)
	}
}

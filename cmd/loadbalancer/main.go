package main

import (
	"bytes"
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

		handleLoadBalancedRequest(balancer, w, r, cbCoordinator, metricsManager, eventSystem)
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

func handleLoadBalancedRequest(balancer *lb.Balancer, w http.ResponseWriter, r *http.Request,
	cbc *lb.CircuitBreakerCoordinator, mm *metrics.MetricsManager, es *events.EventSystem) {

	totalServers := len(balancer.ServerManager.GetAllServers())
	if totalServers == 0 {
		es.Publish(events.ErrorEvent, "Request failed: No backend servers registered")
		http.Error(w, "Service Unavailable (no backend servers)", http.StatusServiceUnavailable)
		return
	}

	priority := lb.ExtractPriority(r)
	requestID := mm.GeneratePacketID()
	attempted := make(map[string]bool, totalServers)

	var bodyBytes []byte
	if r.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(r.Body)
		if err != nil {
			es.Publish(events.ErrorEvent, fmt.Sprintf("Failed to read request body: %v", err))
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	var lastErr error

	for attempt := 0; attempt < totalServers; attempt++ {
		srv := balancer.PickServerWithExclude(r, attempted)
		if srv == nil {
			break
		}
		attempted[srv.ID] = true

		active := server.BeginRequest(srv)
		dispatchEvent := metrics.PacketEvent{
			RequestID:      requestID,
			Attempt:        attempt + 1,
			Priority:       priority,
			ServerID:       srv.ID,
			ServerAddress:  fmt.Sprintf("%s:%d", srv.Address, srv.Port),
			Status:         "dispatch",
			Timestamp:      time.Now(),
			ActiveRequests: active,
		}
		mm.RecordAndBroadcastPacketEvent(es, dispatchEvent)

		if active > lb.BusyThreshold {
			activeAfter := server.EndRequest(srv)
			rerouteEvent := dispatchEvent
			rerouteEvent.Status = "rerouted"
			rerouteEvent.Reason = "busy"
			rerouteEvent.Timestamp = time.Now()
			rerouteEvent.ActiveRequests = activeAfter
			mm.RecordAndBroadcastPacketEvent(es, rerouteEvent)

			es.Publish(events.WarningEvent, fmt.Sprintf("Server %s busy; rerouting request %s", srv.ID, requestID))
			continue
		}

		start := time.Now()
		result, err := forwardToBackend(srv, r, bodyBytes)
		duration := time.Since(start)
		responseMs := float64(duration.Milliseconds())

		if err != nil {
			activeAfter := server.EndRequest(srv)
			cbc.RecordFailure(srv)
			mm.RecordRequest(srv.ID, responseMs, true)

			failureEvent := metrics.PacketEvent{
				RequestID:      requestID,
				Attempt:        attempt + 1,
				Priority:       priority,
				ServerID:       srv.ID,
				ServerAddress:  fmt.Sprintf("%s:%d", srv.Address, srv.Port),
				Status:         "failed",
				Reason:         err.Error(),
				Timestamp:      time.Now(),
				ResponseTime:   responseMs,
				ActiveRequests: activeAfter,
			}
			mm.RecordAndBroadcastPacketEvent(es, failureEvent)
			es.Publish(events.ErrorEvent, fmt.Sprintf("Request to %s failed: %v", srv.ID, err))

			lastErr = err
			continue
		}

		if result.StatusCode >= http.StatusInternalServerError {
			activeAfter := server.EndRequest(srv)
			cbc.RecordFailure(srv)
			mm.RecordRequest(srv.ID, responseMs, true)

			failureEvent := metrics.PacketEvent{
				RequestID:      requestID,
				Attempt:        attempt + 1,
				Priority:       priority,
				ServerID:       srv.ID,
				ServerAddress:  fmt.Sprintf("%s:%d", srv.Address, srv.Port),
				Status:         "failed",
				Reason:         fmt.Sprintf("status %d", result.StatusCode),
				Timestamp:      time.Now(),
				ResponseTime:   responseMs,
				ActiveRequests: activeAfter,
			}
			mm.RecordAndBroadcastPacketEvent(es, failureEvent)
			es.Publish(events.WarningEvent, fmt.Sprintf("Request to %s returned status %d", srv.ID, result.StatusCode))

			lastErr = fmt.Errorf("backend status %d", result.StatusCode)
			continue
		}

		cbc.RecordSuccess(srv)
		mm.RecordRequest(srv.ID, responseMs, false)
		activeAfter := server.EndRequest(srv)

		successEvent := metrics.PacketEvent{
			RequestID:      requestID,
			Attempt:        attempt + 1,
			Priority:       priority,
			ServerID:       srv.ID,
			ServerAddress:  fmt.Sprintf("%s:%d", srv.Address, srv.Port),
			Status:         "completed",
			Timestamp:      time.Now(),
			ResponseTime:   responseMs,
			ActiveRequests: activeAfter,
		}
		mm.RecordAndBroadcastPacketEvent(es, successEvent)

		es.Publish(events.InfoEvent, fmt.Sprintf("Request %s served by %s in %.0fms", requestID, srv.ID, responseMs))

		writeBackendResponse(w, result)
		return
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("no healthy downstream servers")
	}
	es.Publish(events.ErrorEvent, fmt.Sprintf("Request %s failed: %v", requestID, lastErr))
	http.Error(w, "Service Unavailable (no healthy servers)", http.StatusServiceUnavailable)
}

type backendResult struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

func forwardToBackend(srv *server.Server, original *http.Request, body []byte) (*backendResult, error) {
	url := fmt.Sprintf("http://%s:%d%s", srv.Address, srv.Port, original.URL.Path)
	if raw := original.URL.RawQuery; raw != "" {
		url = url + "?" + raw
	}

	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(original.Context(), original.Method, url, reqBody)
	if err != nil {
		return nil, err
	}

	for k, vv := range original.Header {
		for _, v := range vv {
			req.Header.Set(k, v)
		}
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return &backendResult{
		StatusCode: resp.StatusCode,
		Header:     resp.Header.Clone(),
		Body:       respBody,
	}, nil
}

func writeBackendResponse(w http.ResponseWriter, result *backendResult) {
	for k, vv := range result.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}

	w.WriteHeader(result.StatusCode)
	if len(result.Body) == 0 {
		return
	}
	if _, err := w.Write(result.Body); err != nil {
		log.Printf("Error writing response body: %v", err)
	}
}

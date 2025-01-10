// cmd/loadbalancer/main.go
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

	"load-balancer/internal/config"

	"load-balancer/internal/health"

	"load-balancer/internal/lb"

	"load-balancer/internal/server"
)

func main() {
	// 1. Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Unable to load config: %v", err)
	}

	// 2. Initialize the server manager
	srvMgr := initServerManager(cfg)

	// 3. Create Weighted Round Robin, IP hash, sticky sessions
	wrr := lb.NewWeightedRoundRobin(srvMgr)
	ipHash := lb.NewIPHash(srvMgr)
	stickyMgr := lb.NewStickySessions(srvMgr)

	// 4. Create Balancer
	balancer := lb.NewBalancer(srvMgr, wrr, ipHash, stickyMgr)
	balancer.UseIPHash = cfg.UseIPHash
	balancer.UseStickySessions = cfg.UseStickySessions

	// 5. Setup circuit breaker
	cbSettings := lb.CircuitBreakerSettings{
		FailureThreshold: cfg.CircuitBreaker.FailureThreshold,
		CooldownPeriod:   cfg.CircuitBreaker.CooldownPeriod,
		TrialRequests:    cfg.CircuitBreaker.TrialRequests,
	}
	cbCoordinator := lb.NewCircuitBreakerCoordinator(srvMgr, cbSettings)
	// Start a background goroutine to monitor open circuits
	go cbCoordinator.MonitorServers()

	// 6. Start health checker
	checker := health.NewChecker(cfg.HealthCheckInterval, srvMgr)
	ctx, cancel := context.WithCancel(context.Background())
	checker.Start(ctx)

	// 7. Setup HTTP server to handle incoming requests
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// 7a. Pick a server via balancer
		chosenSrv := balancer.PickServer(r)
		if chosenSrv == nil {
			// No available server
			http.Error(w, "Service Unavailable (no healthy servers)", http.StatusServiceUnavailable)
			return
		}

		// 7b. Forward request to chosen server (simplistic example)
		proxyRequest(chosenSrv, w, r, cbCoordinator)
	})

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.LBPort),
		Handler: mux,
	}

	// 8. Start server in a goroutine
	go func() {
		log.Printf("Load Balancer listening on port %d...", cfg.LBPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// 9. Wait for interrupt signal to gracefully shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down load balancer...")
	cancel() // stop the health checker
	ctxTimeout, cancelTimeout := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelTimeout()

	if err := srv.Shutdown(ctxTimeout); err != nil {
		log.Fatalf("HTTP server Shutdown error: %v", err)
	}
	log.Println("Load balancer stopped.")
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

// proxyRequest is a simplistic example of forwarding a request to the chosen server.
// In a real LB, you'd do more robust logic, possibly using httputil.ReverseProxy.
func proxyRequest(srv *server.Server, w http.ResponseWriter, r *http.Request, cbc *lb.CircuitBreakerCoordinator) {
	// Build the destination URL
	url := fmt.Sprintf("http://%s:%d%s", srv.Address, srv.Port, r.URL.Path)

	req, err := http.NewRequest(r.Method, url, r.Body)
	if err != nil {
		cbc.RecordFailure(srv)
		http.Error(w, "Bad request to backend", http.StatusBadGateway)
		return
	}
	// Copy headers
	for k, vv := range r.Header {
		for _, v := range vv {
			req.Header.Add(k, v)
		}
	}

	// If the request had a body, the Body was already consumed by NewRequest.
	// If you need to read it again or pass it directly, handle carefully.

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		cbc.RecordFailure(srv)
		http.Error(w, "Backend request failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Mark success in circuit breaker
	cbc.RecordSuccess(srv)

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

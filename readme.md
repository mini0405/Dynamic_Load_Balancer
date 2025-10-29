# Dynamic Load Balancer

A production-style traffic director written in Go with a neon-themed dashboard that lets you watch, stress, and recover the service in real time.

Minentle Stuurman · Student #222392436  
Module: Industrial Computing Design Project  
Supervisor: Vusumuzi Moyo

---

## Why It Works

The balancer combines four cooperating subsystems:

1. **Smooth Weighted Round Robin**  
   - Located in `internal/lb/weighted_round_robin.go`.  
   - Every health check produces a `CurrentWeight` for each server.  
   - On every request the algorithm increments each server’s running total by its weight, picks the largest value, then subtracts the total weight (classic smooth WRR).  
   - Servers marked `PingStatus = false`, or whose circuit breaker is Open, are skipped.

2. **Health & Weight Engine**  
   - `internal/health/checker.go` calls `internal/server/metrics.go` to simulate metrics and derive a `HealthScore`.  
   - Health scores are normalised into weights and written back through the thread-safe `server.Manager`.

3. **Circuit Breaker Coordinator**  
   - `internal/lb/circuit_breaker.go` counts consecutive failures, trips servers Open, lets them cool down, and moves them to Half-Open for trial requests.  
   - The balancer checks these states, so unhealthy nodes are automatically avoided.

4. **Concurrency & Telemetry**  
   - `cmd/loadbalancer/main.go` wraps every proxied request in `server.BeginRequest` / `EndRequest`.  
   - `internal/metrics/metrics.go` emits `PacketEvent`s (dispatch, rerouted, failed, completed) to SSE clients and keeps per-server request counts, response times, and error history.

Together they guarantee that healthy nodes with strong weights carry most of the traffic, unhealthy ones get rotated out, and the dashboards can visualise the entire flow.

---

## Code Map

| Path | Role |
|------|------|
| `cmd/loadbalancer/main.go` | Boots the balancer, HTTP API, dashboards, test servers, and routes requests through the orchestrator. |
| `internal/lb/balancer.go` | Checks sticky sessions and IP hash, then delegates to WRR; binds sticky sessions. |
| `internal/lb/weighted_round_robin.go` | Smooth WRR implementation with exclusion support. |
| `internal/health/checker.go` | Periodic health check & weight normalisation. |
| `internal/lb/circuit_breaker.go` | Tracks failure thresholds and cooldowns. |
| `internal/server/concurrency.go` | Atomic counters for in-flight requests per server. |
| `internal/metrics/metrics.go` | Tracks LB metrics, emits packet events, exposes `/api/metrics` and `/api/packets`. |
| `internal/api/api.go` | Dashboard/back-office API: server list, toggle/reset, config updates, `/api/test` simulator, SSE events. |
| `internal/dashboard/templates/` + `static/` | The Go-served neon dashboard (works without the React build). |
| `frontend/` | React single-page dashboard with the Flow Mapper, packet stream, control deck, and charts. |

---

## Request Lifecycle

1. UI or external client hits `GET /lb/<path>`.
2. `cmd/loadbalancer/main.go` calls `balancer.PickServerWithExclude`.
3. Balancer checks:
   - Sticky-session map → healthy server? return it.
   - IP-hash map → healthy server? return it.
   - Otherwise calls `weightedRoundRobin.PickServer`.
4. Smooth WRR skips disabled or circuit-open nodes, executes weighted pick.
5. Proxy forwards request, measures time, updates circuit breaker and metrics.
6. `metrics.Manager` records the request and emits packet events for the dashboards.

Busy threshold and retries in `handleLoadBalancedRequest` ensure traffic shifts automatically when a node is saturated.

---

## Dashboards

### Go Dashboard (`internal/dashboard/...`)
- **Traffic Driver** – Start/Stop buttons + RPS slider.
- **Scenario Lab** – Failure Drill, Heavy Load, Priority Spike, Recovery Sweep.
- **Flow Visual** – balancer-to-server animation powered by SSE packet events.
- **Server cards** – disable/enable, reset, and metrics per node.
- **Distribution & Response charts** – Chart.js with neon styling.

### React Dashboard (`frontend/src/App.jsx`)
- Mirrors the traffic driver & scenario lab.
- `TrafficVisualizer` animates the same packet events.
- `PacketStream` groups attempts per request ID with status tags.
- Control panel manages config toggles, traffic generator, scenarios, and server enrolment.

Both dashboards talk to the same REST + SSE endpoints (`internal/api/api.go`).

---

## Running Locally

```bash
# start Go balancer + dashboards + sample backend servers
go run cmd/loadbalancer/main.go

# (optional) run React dashboard in dev mode
cd frontend
npm install
npm run dev
```

Visit:

- Go dashboard: `http://localhost:8080/`
- React dashboard dev server: `http://localhost:5173/`
- External load-balanced endpoint: `http://localhost:8080/lb/...`

---

## Exercising the System

1. Open the dashboard’s Traffic Driver and start traffic at ~20 rps.
2. Watch the Flow Visual – pulses show dispatch, reroute, failure, completion.
3. Use Scenario Lab:
   - Failure Drill: disables the first healthy server.
   - Heavy Load: burst of `/api/test` calls with current priority focus.
   - Priority Spike: mix of critical and medium priority traffic.
   - Recovery Sweep: resets breakers and re-enables offline servers.
4. Toggle or reset individual servers via the Server Fabric table/cards.
5. Observe metrics export: `curl http://localhost:8080/api/metrics`.

---

## Customising

- Adjust `BusyThreshold` or circuit breaker settings in `internal/lb/balancer.go` and `internal/lb/circuit_breaker.go`.
- Replace simulated metrics with real probes in `internal/server/metrics.go`.
- Add new scenarios by wiring buttons → API handlers → `handleLoadBalancedRequest`.

---

## Summary

This project showcases a complete load-balancing cycle:

- Automatic weighting from health metrics.
- Smooth weighted round robin written from scratch.
- Circuit breaker protection and traffic re-routing.
- Concurrency-aware telemetry, streamed live to dashboards.
- Interactive control panel for stress-testing and demos.

Everything is transparent—from Go’s internals to the neon UI—so you can understand, tweak, and present a state-of-the-art load balancer tailored for the Industrial Computing Design Project.

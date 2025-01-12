
# **THANOS: The Ultimate Load Balancer**

![THANOS Logo](https://via.placeholder.com/600x200?text=THANOS+Load+Balancer)

**THANOS** is a powerful, feature-rich load balancer built with Golang that uses the **five infinity stones** (metrics) to deliver optimal and dynamic traffic distribution. Inspired by the balance of the universe, THANOS ensures fairness, reliability, and resilience in your distributed systems.

---

## 🌟 **Features**

- **Weighted Round Robin Load Balancing**:
  - Dynamically distributes traffic based on server health and custom metrics.
  
- **Infinity Stone Metrics**:
  - **CPU Utilization** (Power Stone)
  - **Memory Usage** (Reality Stone)
  - **Response Time** (Time Stone)
  - **Error Rate** (Soul Stone)
  - **Ping Status** (Space Stone)

- **Sticky Sessions**:
  - Ensures session affinity for consistent user experience.

- **IP Hashing**:
  - Maps clients to servers based on their IP addresses for predictable routing.

- **Health Checks**:
  - Periodically assesses server health using the five metrics.

- **Circuit Breaker**:
  - Protects servers by dynamically routing traffic away from unhealthy ones.

- **Metrics Dashboard**:
  - Visualize server health, weights, and statuses in real time.

---

## 🛠 **Installation**

### Prerequisites
- **Go**: Version 1.18 or higher
- **Git**: Version 2.25 or higher

### Clone the Repository
```bash
git clone https://github.com//thanos-load-balancer.git
cd thanos-load-balancer
```

### Build and Run
1. **Build the project**:
   ```bash
   go build -o thanos cmd/loadbalancer/main.go
   ```

2. **Run the load balancer**:
   ```bash
   ./thanos
   ```

3. **Configure your servers**:
   - Edit the configuration file `internal/config/config.go` to include your backend servers.

---

## ⚙️ **Configuration**

Modify the `config.go` file to set up your environment:
```go
LBPort:              8080, // Load balancer port
HealthCheckInterval: 5 * time.Second, // Health check interval
UseIPHash:           true, // Enable IP hashing
UseStickySessions:   true, // Enable sticky sessions
CircuitBreaker: {
    FailureThreshold: 3,
    CooldownPeriod:   10 * time.Second,
    TrialRequests:    2,
},
Servers: [
    {ID: "server-1", Address: "localhost", Port: 9001},
    {ID: "server-2", Address: "localhost", Port: 9002},
]
```

---

## 🚀 **Usage**

### Start the Load Balancer
```bash
./thanos
```

### Endpoints
1. **Forward Traffic**:
   - Send client requests to the load balancer:
     ```bash
     curl http://localhost:8080/
     ```

2. **Metrics Dashboard**:
   - Access real-time server metrics:
     ```bash
     http://localhost:8080/metrics
     ```

---

## 📊 **Metrics Dashboard**

The dashboard visualizes the health of servers based on the infinity stone metrics. It displays:
- **Server Health Scores**
- **Current Weights**
- **Circuit Breaker Status (Closed/Open/Half-Open)**
- **CPU and Memory Usage**
- **Response Times and Error Rates**

![Dashboard Preview](https://via.placeholder.com/800x400?text=Dashboard+Preview)

---

## 🧩 **How It Works**

### Infinity Stones (Metrics)
1. **Power Stone (CPU Utilization)**: Measures server processing power usage.
2. **Reality Stone (Memory Usage)**: Tracks server memory consumption.
3. **Time Stone (Response Time)**: Ensures servers respond within acceptable limits.
4. **Soul Stone (Error Rate)**: Monitors failure rates for server reliability.
5. **Space Stone (Ping Status)**: Verifies server reachability.

### Balancing Algorithm
1. **Weighted Round Robin**:
   - Dynamically assigns traffic based on the combined weights of all five metrics.
2. **Circuit Breaker**:
   - Routes traffic away from unhealthy servers.
3. **Sticky Sessions & IP Hashing**:
   - Ensures consistent routing for repeat users or sessions.

---

## 🧪 **Testing**

### Run the Load Balancer with Test Servers
1. Start mock servers (9001 and 9002).
2. Send continuous requests to the load balancer:
   ```bash
   while true; do curl -s http://localhost:8080/ -o /dev/null & done
   ```

3. Simulate failures by introducing delays or increasing CPU/memory usage on servers.

### View Metrics
- Use `/metrics` or the dashboard to monitor server health and load balancing in action.

---

## 🤝 **Contributing**

Contributions are welcome! Here's how you can get involved:
1. Fork the repository.
2. Create a new branch:
   ```bash
   git checkout -b feature/my-feature
   ```
3. Commit your changes:
   ```bash
   git commit -m "Add new feature"
   ```
4. Push to the branch:
   ```bash
   git push origin feature/my-feature
   ```
5. Submit a pull request.

---

## 🛡 **License**

This project is licensed under the [MIT License](LICENSE).

---

## 💬 **Feedback**

Have ideas or suggestions? Open an issue or reach out to us at [your_email@example.com](mailto:your_email@example.com).

---

## 🌌 **THANOS: Bringing Balance to Distributed Systems**

Ensure your applications are served with fairness, reliability, and efficiency with **THANOS**, the load balancer that wields the power of the **five infinity stones**.
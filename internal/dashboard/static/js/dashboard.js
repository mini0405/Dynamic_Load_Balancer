// internal/dashboard/static/js/dashboard.js

document.addEventListener('DOMContentLoaded', function () {
    // DOM elements
    const serversContainer = document.getElementById('servers-container');
    const eventsContainer = document.getElementById('events-container');
    const toggleIpHash = document.getElementById('toggle-ip-hash');
    const toggleStickySessions = document.getElementById('toggle-sticky-sessions');
    const generateTrafficBtn = document.getElementById('btn-generate-traffic');
    const stopTrafficBtn = document.getElementById('btn-stop-traffic');
    const requestsPerSecond = document.getElementById('requests-per-second');
    const rpsValue = document.getElementById('rps-value');
    const refreshServersBtn = document.getElementById('refresh-servers');
    const clearEventsBtn = document.getElementById('clear-events');
    const totalRequestsEl = document.getElementById('total-requests');
    const avgResponseTimeEl = document.getElementById('avg-response-time');

    // Charts
    let distributionChart;
    let responseTimeChart;

    // State
    let trafficInterval = null;
    let servers = [];
    let serverRequestCounts = {};
    let responseTimes = [];
    let totalRequests = 0;
    let isFirstLoad = true;

    // Initialize dashboard
    console.log("Dashboard initializing...");
    fetchServers();
    setupEventListeners();
    connectEventSource();

    // Fetch servers data every 3 seconds
    setInterval(fetchServers, 3000);

    function fetchServers() {
        console.log("Fetching servers data...");

        fetch('/api/servers')
            .then(response => {
                if (!response.ok) {
                    throw new Error(`Server responded with status: ${response.status}`);
                }
                return response.json();
            })
            .then(data => {
                console.log("Received servers data:", data);
                servers = data;
                renderServers();

                if (isFirstLoad) {
                    initCharts();
                    isFirstLoad = false;
                }

                updateDistributionChart();
            })
            .catch(error => {
                console.error("Error fetching server data:", error);
                logEvent('Error fetching server data: ' + error.message, 'error');

                // Show offline status
                document.getElementById('lb-status-indicator').className = 'status-indicator offline';
                document.getElementById('lb-status-text').textContent = 'Offline';
            });
    }

    function renderServers() {
        if (!serversContainer) {
            console.error("servers-container element not found!");
            return;
        }

        serversContainer.innerHTML = '';

        if (servers.length === 0) {
            serversContainer.innerHTML = '<div class="no-servers">No servers available</div>';
            return;
        }

        servers.forEach(server => {
            const serverHealth = getServerHealthClass(server);
            const cbState = getCircuitBreakerState(server.CircuitBreakerState);
            let cbIcon = '<i class="fas fa-check-circle" style="color: #2ecc71;"></i>';

            if (cbState === "Open") {
                cbIcon = '<i class="fas fa-times-circle" style="color: #e74c3c;"></i>';
            } else if (cbState === "Half-Open") {
                cbIcon = '<i class="fas fa-exclamation-circle" style="color: #f39c12;"></i>';
            }

            const serverCard = document.createElement('div');
            serverCard.className = `server-card ${serverHealth}`;

            serverCard.innerHTML = `
                <div class="server-header">
                    <span class="server-id">${server.ID}</span>
                    <div class="server-actions">
                        <button class="action-button toggle-server" data-id="${server.ID}">
                            ${server.PingStatus ? 'Disable' : 'Enable'}
                        </button>
                        <button class="action-button reset-server" data-id="${server.ID}">Reset</button>
                    </div>
                </div>
                <div class="server-details">
                    <div>${server.Address}:${server.Port}</div>
                    <div>Circuit Breaker: ${cbIcon} ${cbState}</div>
                </div>
                <div class="server-metrics">
                    <div class="metric-row">
                        <span class="metric-label">CPU:</span>
                        <span>${Math.round(server.CPUUsage * 100)}%</span>
                    </div>
                    <div class="metric-row">
                        <span class="metric-label">Memory:</span>
                        <span>${Math.round(server.MemUsage * 100)}%</span>
                    </div>
                    <div class="metric-row">
                        <span class="metric-label">Response Time:</span>
                        <span>${Math.round(server.ResponseTime)}ms</span>
                    </div>
                    <div class="metric-row">
                        <span class="metric-label">Error Rate:</span>
                        <span>${(server.ErrorRate * 100).toFixed(1)}%</span>
                    </div>
                    <div class="metric-row">
                        <span class="metric-label">Health Score:</span>
                        <span>${server.HealthScore.toFixed(2)}</span>
                    </div>
                    <div class="metric-row">
                        <span class="metric-label">Weight:</span>
                        <span>${(server.CurrentWeight * 100).toFixed(1)}%</span>
                    </div>
                </div>
            `;

            serversContainer.appendChild(serverCard);
        });

        // Add event listeners to server buttons
        document.querySelectorAll('.toggle-server').forEach(button => {
            button.addEventListener('click', function () {
                toggleServerStatus(this.dataset.id);
            });
        });

        document.querySelectorAll('.reset-server').forEach(button => {
            button.addEventListener('click', function () {
                resetServer(this.dataset.id);
            });
        });
    }

    function getServerHealthClass(server) {
        if (server.CircuitBreakerState !== 0) {
            return 'unhealthy';
        }

        if (!server.PingStatus || server.ErrorRate > 0.05) {
            return 'degraded';
        }

        return 'healthy';
    }

    function getCircuitBreakerState(state) {
        switch (state) {
            case 0: return 'Closed';
            case 1: return 'Open';
            case 2: return 'Half-Open';
            default: return 'Unknown';
        }
    }

    function toggleServerStatus(serverId) {
        fetch(`/api/servers/${serverId}/toggle`, {
            method: 'POST'
        })
            .then(response => {
                if (!response.ok) {
                    throw new Error(`Server responded with status: ${response.status}`);
                }
                return response.json();
            })
            .then(data => {
                logEvent(`Server ${serverId} ${data.enabled ? 'enabled' : 'disabled'}`,
                    data.enabled ? 'success' : 'warning');
                fetchServers();
            })
            .catch(error => {
                console.error("Error toggling server:", error);
                logEvent('Error toggling server: ' + error.message, 'error');
            });
    }

    function resetServer(serverId) {
        fetch(`/api/servers/${serverId}/reset`, {
            method: 'POST'
        })
            .then(response => {
                if (!response.ok) {
                    throw new Error(`Server responded with status: ${response.status}`);
                }
                return response.json();
            })
            .then(data => {
                logEvent(`Server ${serverId} circuit breaker reset`, 'info');
                fetchServers();
            })
            .catch(error => {
                console.error("Error resetting server:", error);
                logEvent('Error resetting server: ' + error.message, 'error');
            });
    }

    function setupEventListeners() {
        toggleIpHash.addEventListener('change', function () {
            updateLoadBalancerConfig();
        });

        toggleStickySessions.addEventListener('change', function () {
            updateLoadBalancerConfig();
        });

        generateTrafficBtn.addEventListener('click', function () {
            startGeneratingTraffic();
        });

        stopTrafficBtn.addEventListener('click', function () {
            stopGeneratingTraffic();
        });

        requestsPerSecond.addEventListener('input', function () {
            rpsValue.textContent = this.value;

            if (trafficInterval) {
                stopGeneratingTraffic();
                startGeneratingTraffic();
            }
        });

        refreshServersBtn.addEventListener('click', function () {
            this.classList.add('rotating');
            fetchServers();
            setTimeout(() => {
                this.classList.remove('rotating');
            }, 1000);
        });

        clearEventsBtn.addEventListener('click', function () {
            eventsContainer.innerHTML = '';
            logEvent('Event log cleared', 'info');
        });
    }

    function updateLoadBalancerConfig() {
        fetch('/api/config', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                useIPHash: toggleIpHash.checked,
                useStickySessions: toggleStickySessions.checked
            })
        })
            .then(response => {
                if (!response.ok) {
                    throw new Error(`Server responded with status: ${response.status}`);
                }
                return response.json();
            })
            .then(data => {
                let message = `Load balancer config updated:`;
                if (data.useIPHash) {
                    message += ' IP Hash enabled,';
                } else {
                    message += ' IP Hash disabled,';
                }

                if (data.useStickySessions) {
                    message += ' Sticky Sessions enabled';
                } else {
                    message += ' Sticky Sessions disabled';
                }

                logEvent(message, 'info');
            })
            .catch(error => {
                console.error("Error updating config:", error);
                logEvent('Error updating config: ' + error.message, 'error');
            });
    }

    function startGeneratingTraffic() {
        const rps = parseInt(requestsPerSecond.value);
        const interval = Math.floor(1000 / rps);

        generateTrafficBtn.disabled = true;
        stopTrafficBtn.disabled = false;

        logEvent(`Starting traffic generation: ${rps} requests per second`, 'info');

        clearServerRequestCounts();

        trafficInterval = setInterval(() => {
            sendTestRequest();
        }, interval);
    }

    function stopGeneratingTraffic() {
        if (trafficInterval) {
            clearInterval(trafficInterval);
            trafficInterval = null;

            generateTrafficBtn.disabled = false;
            stopTrafficBtn.disabled = true;

            logEvent('Traffic generation stopped', 'info');
        }
    }

    function clearServerRequestCounts() {
        serverRequestCounts = {};
        servers.forEach(server => {
            serverRequestCounts[server.ID] = 0;
        });
    }

    function sendTestRequest() {
        // Create a cookie with a random session ID (for sticky session testing)
        if (toggleStickySessions.checked && !document.cookie.includes('session_id')) {
            const sessionId = 'session-' + Math.floor(Math.random() * 1000000);
            document.cookie = `session_id=${sessionId}; path=/;`;
        }

        const startTime = performance.now();

        fetch('/api/test')
            .then(response => {
                if (!response.ok) {
                    throw new Error(`Test request failed with status: ${response.status}`);
                }
                return response.json();
            })
            .then(data => {
                const endTime = performance.now();
                const responseTime = endTime - startTime;

                // Record which server handled the request
                if (data.server && serverRequestCounts.hasOwnProperty(data.server)) {
                    serverRequestCounts[data.server]++;
                }

                // Record response time
                responseTimes.push({
                    time: new Date(),
                    value: responseTime
                });

                if (responseTimes.length > 50) {
                    responseTimes.shift();
                }

                // Update total request count
                totalRequests++;
                totalRequestsEl.textContent = `Total Requests: ${totalRequests}`;

                // Update average response time
                const avgResponseTime = responseTimes.reduce((sum, item) => sum + item.value, 0) / responseTimes.length;
                avgResponseTimeEl.textContent = `Avg Response: ${Math.round(avgResponseTime)}ms`;

                updateCharts();
            })
            .catch(error => {
                console.error("Test request failed:", error);
                logEvent('Test request failed: ' + error.message, 'error');
            });
    }

    function initCharts() {
        console.log("Initializing charts...");

        // Check if canvas elements exist
        const distributionCanvasEl = document.getElementById('distribution-chart');
        const responseCanvasEl = document.getElementById('response-chart');

        if (!distributionCanvasEl || !responseCanvasEl) {
            console.error("Chart canvas elements not found!");
            return;
        }

        // Destroy existing charts if they exist
        if (distributionChart) {
            distributionChart.destroy();
        }

        if (responseTimeChart) {
            responseTimeChart.destroy();
        }

        // Initialize server request counts
        clearServerRequestCounts();

        // Configuration for both charts
        const chartConfig = {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: {
                    display: true,
                    position: 'top',
                    labels: {
                        boxWidth: 12,
                        padding: 10,
                        font: {
                            size: 12
                        }
                    }
                },
                tooltip: {
                    mode: 'index',
                    intersect: false,
                    titleFont: {
                        size: 12
                    },
                    bodyFont: {
                        size: 12
                    },
                    padding: 10
                }
            },
            animation: {
                duration: 1000
            }
        };

        // Distribution chart
        const distributionCtx = distributionCanvasEl.getContext('2d');
        distributionChart = new Chart(distributionCtx, {
            type: 'bar',
            data: {
                labels: servers.map(server => server.ID),
                datasets: [{
                    label: 'Request Distribution',
                    data: Object.values(serverRequestCounts),
                    backgroundColor: 'rgba(52, 152, 219, 0.7)',
                    borderColor: 'rgba(52, 152, 219, 1)',
                    borderWidth: 1
                }]
            },
            options: {
                ...chartConfig,
                scales: {
                    y: {
                        beginAtZero: true,
                        title: {
                            display: true,
                            text: 'Number of Requests',
                            font: {
                                size: 12
                            }
                        }
                    },
                    x: {
                        title: {
                            display: true,
                            text: 'Servers',
                            font: {
                                size: 12
                            }
                        }
                    }
                }
            }
        });

        // Response time chart
        const responseCtx = responseCanvasEl.getContext('2d');
        responseTimeChart = new Chart(responseCtx, {
            type: 'line',
            data: {
                labels: [],
                datasets: [{
                    label: 'Response Time (ms)',
                    data: [],
                    backgroundColor: 'rgba(46, 204, 113, 0.2)',
                    borderColor: 'rgba(46, 204, 113, 1)',
                    borderWidth: 2,
                    pointRadius: 3,
                    pointBackgroundColor: 'rgba(46, 204, 113, 1)',
                    tension: 0.3,
                    fill: true
                }]
            },
            options: {
                ...chartConfig,
                scales: {
                    y: {
                        beginAtZero: true,
                        title: {
                            display: true,
                            text: 'Response Time (ms)',
                            font: {
                                size: 12
                            }
                        }
                    },
                    x: {
                        title: {
                            display: true,
                            text: 'Time',
                            font: {
                                size: 12
                            }
                        }
                    }
                }
            }
        });

        console.log("Charts initialized successfully");
    }

    function updateCharts() {
        updateDistributionChart();
        updateResponseTimeChart();
    }

    function updateDistributionChart() {
        if (!distributionChart) return;

        distributionChart.data.labels = Object.keys(serverRequestCounts);
        distributionChart.data.datasets[0].data = Object.values(serverRequestCounts);
        distributionChart.update('none'); // Update without animation for smoother updates
    }

    function updateResponseTimeChart() {
        if (!responseTimeChart || responseTimes.length === 0) return;

        const labels = responseTimes.map(item => {
            const time = item.time;
            return `${time.getHours()}:${String(time.getMinutes()).padStart(2, '0')}:${String(time.getSeconds()).padStart(2, '0')}`;
        });

        const data = responseTimes.map(item => item.value);

        responseTimeChart.data.labels = labels;
        responseTimeChart.data.datasets[0].data = data;
        responseTimeChart.update('none'); // Update without animation for smoother updates
    }

    function connectEventSource() {
        const eventSource = new EventSource('/api/events');

        eventSource.onmessage = function (event) {
            try {
                const eventData = JSON.parse(event.data);
                logEvent(eventData.message, eventData.type);
            } catch (error) {
                console.error("Error parsing event data:", error);
            }
        };

        eventSource.onerror = function () {
            console.error("EventSource connection error");
            eventSource.close();
            document.getElementById('lb-status-indicator').className = 'status-indicator offline';
            document.getElementById('lb-status-text').textContent = 'Offline';

            // Try to reconnect after 5 seconds
            setTimeout(connectEventSource, 5000);
        };

        eventSource.onopen = function () {
            console.log("EventSource connected");
            document.getElementById('lb-status-indicator').className = 'status-indicator online';
            document.getElementById('lb-status-text').textContent = 'Online';
        };
    }

    function logEvent(message, type = 'info') {
        if (!eventsContainer) {
            console.error("events-container element not found!");
            return;
        }

        const timestamp = new Date().toLocaleTimeString();

        const eventEntry = document.createElement('div');
        eventEntry.className = `event-entry event-type-${type}`;

        let icon = '';
        switch (type) {
            case 'info':
                icon = '<i class="fas fa-info-circle event-icon"></i>';
                break;
            case 'success':
                icon = '<i class="fas fa-check-circle event-icon"></i>';
                break;
            case 'warning':
                icon = '<i class="fas fa-exclamation-triangle event-icon"></i>';
                break;
            case 'error':
                icon = '<i class="fas fa-times-circle event-icon"></i>';
                break;
        }

        eventEntry.innerHTML = `
            <span class="event-timestamp">[${timestamp}]</span>
            ${icon}
            <span class="event-message">${message}</span>
        `;

        eventsContainer.insertBefore(eventEntry, eventsContainer.firstChild);

        // Limit to 100 events
        if (eventsContainer.children.length > 100) {
            eventsContainer.removeChild(eventsContainer.lastChild);
        }
    }

    // Add rotating animation for refresh button
    document.head.insertAdjacentHTML('beforeend', `
        <style>
            @keyframes rotating {
                from { transform: rotate(0deg); }
                to { transform: rotate(360deg); }
            }
            .rotating {
                animation: rotating 1s linear;
            }
        </style>
    `);
});
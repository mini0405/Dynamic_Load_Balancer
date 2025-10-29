// internal/dashboard/static/js/dashboard.js

document.addEventListener('DOMContentLoaded', function () {
    // DOM elements
    const serversContainer = document.getElementById('servers-container');
    const eventsContainer = document.getElementById('events-container');
    const toggleIpHash = document.getElementById('toggle-ip-hash');
    const toggleStickySessions = document.getElementById('toggle-sticky-sessions');
    const requestsPerSecond = document.getElementById('requests-per-second');
    const rpsValue = document.getElementById('rps-value');
    const startTrafficBtn = document.getElementById('btn-start-traffic');
    const stopTrafficBtn = document.getElementById('btn-stop-traffic');
    const refreshServersBtn = document.getElementById('refresh-servers');
    const clearEventsBtn = document.getElementById('clear-events');
    const totalRequestsEl = document.getElementById('total-requests');
    const avgResponseTimeEl = document.getElementById('avg-response-time');
    const heroStatusMessage = document.getElementById('hero-status-message');
    const scenarioFailureBtn = document.getElementById('scenario-failure');
    const scenarioHeavyBtn = document.getElementById('scenario-heavy');
    const scenarioPriorityBtn = document.getElementById('scenario-priority');
    const scenarioRecoveryBtn = document.getElementById('scenario-recovery');
    const flowVisual = document.getElementById('flow-visual');
    const flowServersContainer = document.getElementById('flow-servers');
    const flowParticlesContainer = document.getElementById('flow-particles');
    const flowBalancerNode = flowVisual ? flowVisual.querySelector('.flow-node') : null;
    const scenarioButtonsList = [scenarioFailureBtn, scenarioHeavyBtn, scenarioPriorityBtn, scenarioRecoveryBtn].filter(Boolean);

    // Charts
    let distributionChart;
    let responseTimeChart;

    // State
    let servers = [];
    let serverRequestCounts = {};
    let responseTimes = [];
    let totalRequests = 0;
    let isFirstLoad = true;
    let scenarioBusy = false;
    let trafficTimer = null;
    if (startTrafficBtn) {
        startTrafficBtn.disabled = true;
    }
    if (stopTrafficBtn) {
        stopTrafficBtn.disabled = true;
    }



    function setStatusMessage(message) {
        if (heroStatusMessage) {
            heroStatusMessage.textContent = message;
        }
    }
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
                setStatusMessage('Monitoring live traffic');

                if (isFirstLoad) {
                    initCharts();
                    isFirstLoad = false;
                }

                updateDistributionChart();
            })
            .catch(error => {
                console.error("Error fetching server data:", error);
                logEvent('Error fetching server data: ' + error.message, 'error');
                setStatusMessage('Unable to reach load balancer');

                servers = [];
                renderServers();

                if (trafficTimer) {
                    clearInterval(trafficTimer);
                    trafficTimer = null;
                }
                if (startTrafficBtn) {
                    startTrafficBtn.disabled = true;
                }
                if (stopTrafficBtn) {
                    stopTrafficBtn.disabled = true;
                }

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
        if (flowServersContainer) {
            flowServersContainer.innerHTML = '';
        }

        if (servers.length === 0) {
            serversContainer.innerHTML = '<div class="no-servers">No servers available</div>';
            if (flowServersContainer) {
                flowServersContainer.innerHTML = '<div class="flow-empty">No servers registered</div>';
            }
            if (trafficTimer) {
                clearInterval(trafficTimer);
                trafficTimer = null;
            }
            if (startTrafficBtn) {
                startTrafficBtn.disabled = true;
            }
            if (stopTrafficBtn) {
                stopTrafficBtn.disabled = true;
            }
            return;
        }

        servers.forEach((server, index) => {
            if (!serverRequestCounts.hasOwnProperty(server.ID)) {
                serverRequestCounts[server.ID] = 0;
            }
            const serverHealth = getServerHealthClass(server);
            const cbState = getCircuitBreakerState(server.CircuitBreakerState);
            let cbIcon = '<i class="fas fa-check-circle" style="color: #46ffb9;"></i>';

            if (cbState === "Open") {
                cbIcon = '<i class="fas fa-times-circle" style="color: #ff3b6b;"></i>';
            } else if (cbState === "Half-Open") {
                cbIcon = '<i class="fas fa-exclamation-circle" style="color: #fcee0b;"></i>';
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
                        <span class="metric-label">Active Requests:</span>
                        <span>${server.ActiveRequests || 0}</span>
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

            if (flowServersContainer) {
                const flowNode = document.createElement('div');
                flowNode.className = 'server-node ' + (server.PingStatus ? 'online' : 'offline');
                flowNode.dataset.serverId = server.ID;
                flowNode.innerHTML = `
                    <div class="node-core">${index + 1}</div>
                    <span class="node-label">${server.ID.slice(0, 8)}</span>
                    <span class="node-meta">${server.PingStatus ? 'Online' : 'Offline'} · ${cbState}</span>
                `;
                flowServersContainer.appendChild(flowNode);
            }
        });

        if (startTrafficBtn) {
            startTrafficBtn.disabled = trafficTimer ? true : servers.length === 0;
        }
        if (stopTrafficBtn) {
            stopTrafficBtn.disabled = !trafficTimer;
        }

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

        updateScenarioAvailability();
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
        setStatusMessage(`Updating ${serverId} state...`);
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
                setStatusMessage(`Server ${serverId} ${data.enabled ? 'enabled' : 'disabled'}`);
                fetchServers();
            })
            .catch(error => {
                console.error("Error toggling server:", error);
                logEvent('Error toggling server: ' + error.message, 'error');
                setStatusMessage('Failed to toggle server');
            });
    }

    function resetServer(serverId) {
        setStatusMessage(`Resetting ${serverId}...`);
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
                setStatusMessage(`Server ${serverId} reset complete`);
                fetchServers();
            })
            .catch(error => {
                console.error("Error resetting server:", error);
                logEvent('Error resetting server: ' + error.message, 'error');
                setStatusMessage('Reset request failed');
            });
    }

    function triggerScenario(button, handler) {
        if (scenarioBusy) {
            return;
        }
        scenarioBusy = true;
        setScenarioBusyState(true, button);
        Promise.resolve(handler())
            .catch(error => {
                console.error('Scenario error:', error);
                logEvent('Scenario failed: ' + error.message, 'error');
                setStatusMessage('Scenario failed');
            })
            .finally(() => {
                scenarioBusy = false;
                setScenarioBusyState(false);
                updateScenarioAvailability();
            });
    }

    function setScenarioBusyState(isBusy, activeButton) {
        const hasOnline = servers.some(server => server.PingStatus);
        scenarioButtonsList.forEach(btn => {
            if (!btn) {
                return;
            }
            const disableFailure = btn === scenarioFailureBtn && !hasOnline;
            btn.disabled = isBusy || disableFailure;
            if (isBusy && btn === activeButton) {
                btn.classList.add('running');
            } else {
                btn.classList.remove('running');
            }
        });
    }

    function updateScenarioAvailability() {
        setScenarioBusyState(scenarioBusy, null);
    }

    async function runFailureScenario() {
        const target = servers.find(server => server.PingStatus);
        if (!target) {
            setStatusMessage('No active servers available for failure drill');
            logEvent('Failure scenario skipped: no active servers', 'warning');
            return;
        }
        await fetch(`/api/servers/${target.ID}/toggle`, { method: 'POST' });
        logEvent(`Failure drill toggled ${target.ID}`, 'warning');
        setStatusMessage(`Server ${target.ID} disabled for failure drill`);
        fetchServers();
    }

    async function runHeavyScenario() {
        const burst = requestsPerSecond ? parseInt(requestsPerSecond.value, 10) : 10;
        const clamped = Math.min(Math.max(burst, 5), 60);
        await runTrafficBurst(clamped, 'high');
        setStatusMessage(`Heavy load burst dispatched (${clamped} requests)`);
        logEvent(`Heavy load burst dispatched (${clamped} requests)`, 'info');
    }

    async function runPriorityScenario() {
        await runTrafficBurst(8, 'critical');
        await runTrafficBurst(6, 'medium');
        setStatusMessage('Priority spike executed');
        logEvent('Priority spike executed', 'success');
    }

    async function runRecoveryScenario() {
        const actions = servers.map(async server => {
            if (!server.PingStatus) {
                await fetch(`/api/servers/${server.ID}/toggle`, { method: 'POST' });
            }
            await fetch(`/api/servers/${server.ID}/reset`, { method: 'POST' });
        });
        await Promise.allSettled(actions);
        setStatusMessage('Recovery sweep complete');
        logEvent('Recovery scenario executed', 'success');
        fetchServers();
    }

    function runTrafficBurst(count, priority) {
        const tasks = [];
        for (let i = 0; i < count; i++) {
            tasks.push(sendTestRequest({ priority, burstId: `${priority || 'normal'}-${Date.now()}-${i}` }));
        }
        return Promise.allSettled(tasks);
    }

    function handlePacketEvent(packet) {
        if (!packet || !flowVisual || !flowParticlesContainer || !flowBalancerNode) {
            return;
        }

        const selector = packet.serverId ? '[data-server-id="' + packet.serverId + '"]' : null;
        const targetNode = selector && flowServersContainer ? flowServersContainer.querySelector(selector) : null;
        if (!targetNode) {
            return;
        }

        const containerRect = flowVisual.getBoundingClientRect();
        const originRect = flowBalancerNode.getBoundingClientRect();
        const targetRect = targetNode.getBoundingClientRect();

        const originX = originRect.left + originRect.width / 2 - containerRect.left;
        const originY = originRect.top + originRect.height / 2 - containerRect.top;
        const targetX = targetRect.left + targetRect.width / 2 - containerRect.left;
        const targetY = targetRect.top + targetRect.height / 2 - containerRect.top;

        const particle = document.createElement('span');
        const statusName = packet.status ? 'status-' + packet.status : 'status-dispatch';
        particle.className = 'flow-particle ' + statusName;
        particle.style.setProperty('--dx', (targetX - originX) + 'px');
        particle.style.setProperty('--dy', (targetY - originY) + 'px');
        particle.style.left = originX + 'px';
        particle.style.top = originY + 'px';

        flowParticlesContainer.appendChild(particle);
        setTimeout(() => particle.remove(), 1100);
    }
      function setupEventListeners() {
        if (toggleIpHash) {
            toggleIpHash.addEventListener('change', updateLoadBalancerConfig);
        }

        if (toggleStickySessions) {
            toggleStickySessions.addEventListener('change', updateLoadBalancerConfig);
        }

        if (requestsPerSecond && rpsValue) {
            requestsPerSecond.addEventListener('input', function () {
                rpsValue.textContent = this.value;
                if (trafficTimer) {
                    stopTrafficLoop();
                    startTrafficLoop();
                }
            });
        }

        if (startTrafficBtn) {
            startTrafficBtn.addEventListener('click', startTrafficLoop);
        }

        if (stopTrafficBtn) {
            stopTrafficBtn.addEventListener('click', stopTrafficLoop);
        }

        if (refreshServersBtn) {
            refreshServersBtn.addEventListener('click', function () {
                this.classList.add('rotating');
                fetchServers();
                setTimeout(() => {
                    this.classList.remove('rotating');
                }, 1000);
            });
        }

        if (clearEventsBtn && eventsContainer) {
            clearEventsBtn.addEventListener('click', function () {
                eventsContainer.innerHTML = '';
                logEvent('Event log cleared', 'info');
            });
        }

        const scenarioButtons = [
            { element: scenarioFailureBtn, handler: runFailureScenario },
            { element: scenarioHeavyBtn, handler: runHeavyScenario },
            { element: scenarioPriorityBtn, handler: runPriorityScenario },
            { element: scenarioRecoveryBtn, handler: runRecoveryScenario }
        ];

        scenarioButtons.forEach(({ element, handler }) => {
            if (!element || !handler) {
                return;
            }
            element.addEventListener('click', () => triggerScenario(element, handler));
        });

        updateScenarioAvailability();
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

    function clearServerRequestCounts() {
        serverRequestCounts = {};
        servers.forEach(server => {
            serverRequestCounts[server.ID] = 0;
        });
    }

    function sendTestRequest(options = {}) {
        const { priority, burstId } = options;

        if (toggleStickySessions && toggleStickySessions.checked && !document.cookie.includes('session_id')) {
            const sessionId = 'session-' + Math.floor(Math.random() * 1000000);
            document.cookie = `session_id=${sessionId}; path=/;`;
        }

        const params = new URLSearchParams();
        if (priority) {
            params.append('priority', priority);
        }
        if (typeof burstId !== 'undefined') {
            params.append('burst', burstId);
        }
        const url = params.toString() ? `/api/test?${params.toString()}` : '/api/test';
        const startTime = performance.now();

        return fetch(url)
            .then(response => {
                if (!response.ok) {
                    throw new Error(`Test request failed with status: ${response.status}`);
                }
                return response.json();
            })
            .then(data => {
                const endTime = performance.now();
                const responseTime = endTime - startTime;

                if (data.server) {
                    if (!serverRequestCounts.hasOwnProperty(data.server)) {
                        serverRequestCounts[data.server] = 0;
                    }
                    serverRequestCounts[data.server]++;
                }

                responseTimes.push({
                    time: new Date(),
                    value: responseTime
                });

                if (responseTimes.length > 50) {
                    responseTimes.shift();
                }

                totalRequests++;
                if (totalRequestsEl) {
                    totalRequestsEl.textContent = `Total Requests: ${totalRequests}`;
                }

                if (avgResponseTimeEl && responseTimes.length > 0) {
                    const avgResponseTime = responseTimes.reduce((sum, item) => sum + item.value, 0) / responseTimes.length;
                    avgResponseTimeEl.textContent = `Avg Response: ${Math.round(avgResponseTime)}ms`;
                }

                updateCharts();
                return data;
            })
            .catch(error => {
                console.error("Test request failed:", error);
                logEvent('Test request failed: ' + error.message, 'error');
                throw error;
            });
    }

    function startTrafficLoop() {
        if (trafficTimer) {
            return;
        }

        if (!servers || servers.length === 0) {
            setStatusMessage('No servers available for traffic generator');
            logEvent('Traffic generator skipped: no servers online', 'warning');
            return;
        }

        const requested = requestsPerSecond ? parseInt(requestsPerSecond.value || '10', 10) : 10;
        const clamped = Math.max(1, requested);
        const interval = Math.max(1000 / clamped, 75);

        trafficTimer = setInterval(() => {
            sendTestRequest().catch(() => {});
        }, interval);

        if (startTrafficBtn) {
            startTrafficBtn.disabled = true;
        }
        if (stopTrafficBtn) {
            stopTrafficBtn.disabled = false;
        }

        setStatusMessage(`Traffic generator running (${clamped} rps)`);
        logEvent(`Traffic generator running (${clamped} rps)`, 'info');
    }

    function stopTrafficLoop() {
        if (!trafficTimer) {
            return;
        }

        clearInterval(trafficTimer);
        trafficTimer = null;

        if (startTrafficBtn) {
            startTrafficBtn.disabled = false;
        }
        if (stopTrafficBtn) {
            stopTrafficBtn.disabled = true;
        }

        setStatusMessage('Traffic generator stopped');
        logEvent('Traffic generator stopped', 'info');
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
                            size: 12,
                            family: 'Orbitron'
                        },
                        color: '#e2f7ff'
                    }
                },
                tooltip: {
                    mode: 'index',
                    intersect: false,
                    backgroundColor: 'rgba(5, 5, 25, 0.9)',
                    borderColor: 'rgba(0, 255, 240, 0.3)',
                    borderWidth: 1,
                    titleColor: '#00faff',
                    bodyColor: '#e2f7ff',
                    titleFont: {
                        size: 12,
                        family: 'Orbitron'
                    },
                    bodyFont: {
                        size: 12,
                        family: 'Share Tech Mono'
                    },
                    padding: 10
                }
            },
            animation: {
                duration: 1000
            },
            layout: {
                padding: 12
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
                    backgroundColor: 'rgba(0, 250, 255, 0.35)',
                    borderColor: 'rgba(0, 250, 255, 0.85)',
                    hoverBackgroundColor: 'rgba(255, 0, 245, 0.4)',
                    borderWidth: 2,
                    borderRadius: 6
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
                        },
                        ticks: {
                            color: '#7dddfc',
                            font: {
                                family: 'Share Tech Mono'
                            }
                        },
                        grid: {
                            color: 'rgba(0, 255, 240, 0.12)'
                        }
                    },
                    x: {
                        title: {
                            display: true,
                            text: 'Servers',
                            font: {
                                size: 12
                            }
                        },
                        ticks: {
                            color: '#7dddfc',
                            font: {
                                family: 'Share Tech Mono'
                            }
                        },
                        grid: {
                            color: 'rgba(255, 0, 245, 0.12)'
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
                    backgroundColor: 'rgba(255, 0, 245, 0.18)',
                    borderColor: 'rgba(255, 0, 245, 0.85)',
                    borderWidth: 3,
                    pointRadius: 4,
                    pointBackgroundColor: '#fcee0b',
                    pointBorderColor: 'rgba(255, 0, 245, 0.85)',
                    tension: 0.25,
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
                        },
                        ticks: {
                            color: '#7dddfc',
                            font: {
                                family: 'Share Tech Mono'
                            }
                        },
                        grid: {
                            color: 'rgba(0, 255, 240, 0.12)'
                        }
                    },
                    x: {
                        title: {
                            display: true,
                            text: 'Time',
                            font: {
                                size: 12
                            }
                        },
                        ticks: {
                            color: '#7dddfc',
                            font: {
                                family: 'Share Tech Mono'
                            }
                        },
                        grid: {
                            color: 'rgba(255, 0, 245, 0.12)'
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
                if (eventData.type === 'packet') {
                    if (eventData.message) {
                        try {
                            const packet = JSON.parse(eventData.message);
                            handlePacketEvent(packet);
                        } catch (parseError) {
                            console.error('Failed to parse packet payload:', parseError);
                        }
                    }
                    return;
                }
                logEvent(eventData.message, eventData.type);
                setStatusMessage(eventData.message);
            } catch (error) {
                console.error('Error parsing event data:', error);
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



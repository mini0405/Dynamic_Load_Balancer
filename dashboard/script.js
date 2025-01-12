// Fetch metrics periodically and update the table
async function fetchMetrics() {
    try {
        const response = await fetch('http://localhost:8080/metrics');
        if (!response.ok) {
            throw new Error(`Failed to fetch metrics: ${response.statusText}`);
        }
        const metrics = await response.json();
        console.log('Fetched Metrics:', metrics); // Debug log
        updateTable(metrics);
    } catch (error) {
        console.error('Error fetching metrics:', error);
    }
}

// Update the table with the server metrics
function updateTable(metrics) {
    const tableBody = document.querySelector('#serverTable tbody');
    tableBody.innerHTML = ''; // Clear existing rows

    if (!metrics || metrics.length === 0) {
        console.warn('No metrics data available to display.');
        return;
    }

    metrics.forEach(server => {
        const row = document.createElement('tr');

        row.innerHTML = `
            <td>${server.ID}</td>
            <td>${(server.CPUUsage * 100).toFixed(2)}%</td>
            <td>${(server.MemUsage * 100).toFixed(2)}%</td>
            <td>${server.ResponseTime} ms</td>
            <td>${(server.ErrorRate * 100).toFixed(2)}%</td>
            <td>${(server.HealthScore * 100).toFixed(2)}</td>
            <td>${(server.CurrentWeight * 100).toFixed(2)}%</td>
            <td class="${getCircuitBreakerClass(server.CircuitBreakerState)}">
                ${getCircuitBreakerState(server.CircuitBreakerState)}
            </td>
        `;

        tableBody.appendChild(row);
    });
}

// Get CSS class based on circuit breaker state
function getCircuitBreakerClass(state) {
    switch (state) {
        case 0: return 'circuit-closed'; // Closed
        case 1: return 'circuit-open';   // Open
        case 2: return 'circuit-halfopen'; // Half Open
        default: return '';
    }
}

// Get readable circuit breaker state
function getCircuitBreakerState(state) {
    switch (state) {
        case 0: return 'Closed';
        case 1: return 'Open';
        case 2: return 'Half-Open';
        default: return 'Unknown';
    }
}

// Fetch metrics every second
setInterval(fetchMetrics, 1000);

// Initial fetch to populate the table on page load
fetchMetrics();

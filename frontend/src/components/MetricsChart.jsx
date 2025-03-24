import React, { useEffect } from 'react';
import { Line } from 'react-chartjs-2';
import Chart from 'chart.js/auto';

const MetricsChart = ({ servers }) => {
    const [chartData, setChartData] = useState({
        labels: [],
        datasets: []
    });

    useEffect(() => {
        const labels = servers.map(s => `${s.Address}:${s.Port}`);
        
        const datasets = [{
            label: 'CPU Usage',
            data: servers.map(s => s.CPUUsage * 100),
            borderColor: 'rgb(255, 99, 132)',
            tension: 0.1
        }, {
            label: 'Memory Usage',
            data: servers.map(s => s.MemUsage * 100),
            borderColor: 'rgb(54, 162, 235)',
            tension: 0.1
        }, {
            label: 'Response Time (ms)',
            data: servers.map(s => s.ResponseTime),
            borderColor: 'rgb(75, 192, 192)',
            tension: 0.1
        }];

        setChartData({ labels, datasets });
    }, [servers]);

    return (
        <div className="metrics-chart">
            <h2>Server Metrics</h2>
            <Line 
                data={chartData}
                options={{
                    responsive: true,
                    scales: {
                        y: {
                            beginAtZero: true
                        }
                    }
                }}
            />
        </div>
    );
};

export default MetricsChart;
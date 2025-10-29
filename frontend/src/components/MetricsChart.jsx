import React, { useEffect, useState } from 'react';
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
            borderColor: 'rgba(0, 250, 255, 0.85)',
            backgroundColor: 'rgba(0, 250, 255, 0.15)',
            tension: 0.25,
            fill: true
        }, {
            label: 'Memory Usage',
            data: servers.map(s => s.MemUsage * 100),
            borderColor: 'rgba(255, 0, 245, 0.85)',
            backgroundColor: 'rgba(255, 0, 245, 0.12)',
            tension: 0.25,
            fill: true
        }, {
            label: 'Response Time (ms)',
            data: servers.map(s => s.ResponseTime),
            borderColor: 'rgba(252, 238, 11, 0.85)',
            backgroundColor: 'rgba(252, 238, 11, 0.12)',
            tension: 0.25,
            fill: true
        }];

        setChartData({ labels, datasets });
    }, [servers]);

    return (
        <div className="metrics-chart panel">
            <h2>Server Metrics</h2>
            <Line 
                data={chartData}
                options={{
                    responsive: true,
                    maintainAspectRatio: false,
                    scales: {
                        x: {
                            ticks: {
                                color: '#7dddfc',
                                font: {
                                    family: 'Share Tech Mono'
                                }
                            },
                            grid: {
                                color: 'rgba(0, 255, 240, 0.08)'
                            }
                        },
                        y: {
                            beginAtZero: true,
                            ticks: {
                                color: '#7dddfc',
                                font: {
                                    family: 'Share Tech Mono'
                                }
                            },
                            grid: {
                                color: 'rgba(255, 0, 245, 0.08)'
                            }
                        }
                    },
                    elements: {
                        point: {
                            radius: 5,
                            hoverRadius: 7
                        },
                        line: {
                            borderWidth: 3
                        }
                    },
                    plugins: {
                        legend: {
                            labels: {
                                color: '#e2f7ff',
                                font: {
                                    family: 'Orbitron',
                                    size: 12
                                },
                                padding: 20
                            }
                        }
                    }
                }}
            />
        </div>
    );
};

export default MetricsChart;

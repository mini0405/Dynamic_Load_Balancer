import React, { useEffect, useState } from 'react';
import ServerList from './components/ServerList';
import MetricsChart from './components/MetricsChart';
import PacketStream from './components/PacketStream';
import ControlPanel from './components/ControlPanel';
import TrafficVisualizer from './components/TrafficVisualizer';
import './App.css';

function App() {
    const [servers, setServers] = useState([]);
    const [isLoading, setIsLoading] = useState(false);
    const [statusMessage, setStatusMessage] = useState('Monitoring live traffic');

    const fetchServers = async () => {
        try {
            setIsLoading(true);
            const response = await fetch('/api/servers');
            const data = await response.json();
            setServers(Array.isArray(data) ? data : []);
        } catch (error) {
            console.error('Error fetching servers:', error);
        } finally {
            setIsLoading(false);
        }
    };

    useEffect(() => {
        fetchServers();
        const interval = setInterval(fetchServers, 2500);
        return () => clearInterval(interval);
    }, []);

    const toggleServer = async (server) => {
        if (!server) return;
        try {
            await fetch(`/api/servers/${server.ID}/toggle`, { method: 'POST' });
            setStatusMessage(`${server.ID} ${server.PingStatus ? 'disabled' : 'enabled'}`);
            fetchServers();
        } catch (error) {
            console.error('Failed to toggle server', error);
        }
    };

    const resetServer = async (server) => {
        if (!server) return;
        try {
            await fetch(`/api/servers/${server.ID}/reset`, { method: 'POST' });
            setStatusMessage(`${server.ID} reset initiated`);
            fetchServers();
        } catch (error) {
            console.error('Failed to reset server', error);
        }
    };

    const handleScenarioComplete = (scenario) => {
        const labels = {
            failure: 'Failure scenario executed',
            heavy: 'Heavy load burst sent',
            priority: 'Priority traffic generated',
            recovery: 'Recovery sweep dispatched'
        };
        setStatusMessage(labels[scenario] || 'Scenario executed');
        fetchServers();
    };

    return (
        <div className="dashboard">
            <header className="hero-banner">
                <div className="identity">
                    <h1>Minentle Stuurman</h1>
                    <p>Student #222392436 · Industrial Computing Design Project</p>
                    <span className="supervisor">Supervisor: Vusumuzi Moyo</span>
                </div>
                <div className="hero-status">
                    <span className="status-pill">{isLoading ? 'Updating topology…' : 'Live'}</span>
                    <p>{statusMessage}</p>
                </div>
            </header>

            <main className="dashboard-grid">
                <aside className="grid-left">
                    <ControlPanel
                        servers={servers}
                        onScenarioComplete={handleScenarioComplete}
                        onServerAdded={fetchServers}
                        onRefreshConfig={fetchServers}
                        onStatusChange={setStatusMessage}
                    />
                </aside>
                <section className="grid-center">
                    <TrafficVisualizer servers={servers} />
                    <PacketStream />
                </section>
                <aside className="grid-right">
                    <ServerList
                        servers={servers}
                        onToggleServer={toggleServer}
                        onResetServer={resetServer}
                    />
                    <MetricsChart servers={servers} />
                </aside>
            </main>
        </div>
    );
}

export default App;


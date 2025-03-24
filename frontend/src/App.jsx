import React, { useState, useEffect } from 'react';
import ServerList from './components/ServerList';
import AddServerForm from './components/AddServerForm';
import MetricsChart from './components/MetricsChart';
import './App.jsx';

function App() {
    const [servers, setServers] = useState([]);
    const [refresh, setRefresh] = useState(false);

    useEffect(() => {
        const fetchServers = async () => {
            try {
                const response = await fetch('http://localhost:8080/api/servers');
                const data = await response.json();
                setServers(data);
            } catch (error) {
                console.error('Error fetching servers:', error);
            }
        };

        fetchServers();
        const interval = setInterval(fetchServers, 2000);
        return () => clearInterval(interval);
    }, [refresh]);

    return (
        <div className="dashboard">
            <h1>Load Balancer Dashboard</h1>
            <div className="row">
                <div className="col">
                    <AddServerForm onRefresh={() => setRefresh(!refresh)} />
                    <ServerList servers={servers} />
                </div>
                <div className="col">
                    <MetricsChart servers={servers} />
                </div>
            </div>
        </div>
    );
}

export default App;
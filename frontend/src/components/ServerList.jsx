import React from 'react';

const ServerList = ({ servers }) => {
    const simulateFailure = async (serverId) => {
        try {
            await fetch('http://localhost:8080/api/servers/simulate-failure', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ server_id: serverId }),
            });
        } catch (error) {
            console.error('Error simulating failure:', error);
        }
    };

    return (
        <div className="server-list">
            <h2>Servers</h2>
            <table>
                <thead>
                    <tr>
                        <th>ID</th>
                        <th>Address</th>
                        <th>Status</th>
                        <th>Failures</th>
                        <th>Actions</th>
                    </tr>
                </thead>
                <tbody>
                    {servers.map(server => (
                        <tr key={server.ID}>
                            <td>{server.ID.slice(0, 6)}</td>
                            <td>{server.Address}:{server.Port}</td>
                            <td>
                                <span className={`status ${server.CircuitBreakerState === 0 ? 'healthy' : 'unhealthy'}`}>
                                    {['Closed', 'Open', 'Half-Open'][server.CircuitBreakerState]}
                                </span>
                            </td>
                            <td>{server.FailureCount}</td>
                            <td>
                                <button 
                                    onClick={() => simulateFailure(server.ID)}
                                    disabled={server.CircuitBreakerState !== 0}
                                >
                                    Simulate Failure
                                </button>
                            </td>
                        </tr>
                    ))}
                </tbody>
            </table>
        </div>
    );
};

export default ServerList;
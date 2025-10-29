import React from 'react';

const ServerList = ({ servers, onToggleServer, onResetServer }) => {
    return (
        <div className="server-list panel">
            <h2>Server Fabric</h2>
            <table>
                <thead>
                    <tr>
                        <th>ID</th>
                        <th>Endpoint</th>
                        <th>Health</th>
                        <th>Active</th>
                        <th>Breaker</th>
                        <th></th>
                    </tr>
                </thead>
                <tbody>
                    {servers.length === 0 && (
                        <tr className="empty-row">
                            <td colSpan="6">No servers registered</td>
                        </tr>
                    )}
                    {servers.map(server => (
                        <tr key={server.ID}>
                            <td>{server.ID.slice(0, 8)}</td>
                            <td className="server-endpoint">{server.Address}:{server.Port}</td>
                            <td>
                                <span className={`status-chip ${server.PingStatus ? 'online' : 'offline'}`}>
                                    {server.PingStatus ? 'Online' : 'Offline'}
                                </span>
                            </td>
                            <td>{server.ActiveRequests || 0}</td>
                            <td>
                                <span className={`status ${server.CircuitBreakerState === 0 ? 'healthy' : 'unhealthy'}`}>
                                    {['Closed', 'Open', 'Half-Open'][server.CircuitBreakerState]}
                                </span>
                            </td>
                            <td>
                                <div className="server-actions-inline">
                                    <button
                                        className="cyber-button ghost"
                                        onClick={() => onToggleServer(server)}
                                    >
                                        {server.PingStatus ? 'Disable' : 'Enable'}
                                    </button>
                                    <button
                                        className="cyber-button ghost"
                                        onClick={() => onResetServer(server)}
                                    >
                                        Reset
                                    </button>
                                </div>
                            </td>
                        </tr>
                    ))}
                </tbody>
            </table>
        </div>
    );
};

export default ServerList;

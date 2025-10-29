import React, { useEffect, useRef, useState } from 'react';
import AddServerForm from './AddServerForm';
import RpsDial from './RpsDial';

const SCENARIO_LABELS = {
    failure: 'Trigger Failure',
    heavy: 'Heavy Load',
    priority: 'Priority Spike',
    recovery: 'Recovery Sweep'
};

const PRIORITY_OPTIONS = [
    { value: 'critical', label: 'Critical' },
    { value: 'high', label: 'High' },
    { value: 'medium', label: 'Medium' },
    { value: 'normal', label: 'Normal' }
];

const ControlPanel = ({ servers, onScenarioComplete, onRefreshConfig, onServerAdded, onStatusChange }) => {
    const [useIPHash, setUseIPHash] = useState(false);
    const [useSticky, setUseSticky] = useState(true);
    const [priority, setPriority] = useState('critical');
    const [busyScenario, setBusyScenario] = useState(null);
    const [trafficRate, setTrafficRate] = useState(10);
    const [trafficActive, setTrafficActive] = useState(false);

    const trafficTimerRef = useRef(null);
    const priorityRef = useRef(priority);
    const hasServers = Array.isArray(servers) && servers.length > 0;
    const hasOnlineServers = hasServers && servers.some((srv) => srv.PingStatus);

    useEffect(() => {
        priorityRef.current = priority;
    }, [priority]);

    useEffect(() => () => {
        if (trafficTimerRef.current) {
            clearInterval(trafficTimerRef.current);
        }
    }, []);

    const notifyStatus = (message) => {
        if (onStatusChange) {
            onStatusChange(message);
        }
    };

    const updateConfig = async (nextConfig) => {
        try {
            const body = {
                useIPHash,
                useStickySessions: useSticky,
                ...nextConfig
            };
            setUseIPHash(body.useIPHash);
            setUseSticky(body.useStickySessions);

            await fetch('/api/config', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(body)
            });

            if (onRefreshConfig) {
                onRefreshConfig(body);
            }
        } catch (error) {
            console.error('Failed to update config', error);
        }
    };

    const fireTestRequest = async (priorityOverride = priorityRef.current) => {
        try {
            const params = new URLSearchParams();
            if (priorityOverride) {
                params.append('priority', priorityOverride);
            }
            await fetch(`/api/test?${params.toString()}`, { method: 'GET' });
        } catch (error) {
            console.error('Traffic request failed', error);
        }
    };

    const restartTrafficTimer = (rate) => {
        if (!trafficTimerRef.current) {
            return;
        }
        if (!servers || servers.length === 0 || !servers.some((srv) => srv.PingStatus)) {
            stopTraffic();
            return;
        }
        clearInterval(trafficTimerRef.current);
        const clamped = Math.max(1, rate);
        const interval = Math.max(1000 / clamped, 75);
        trafficTimerRef.current = setInterval(() => {
            fireTestRequest();
        }, interval);
        notifyStatus(`Traffic generator running (${clamped} rps)`);
    };

    const startTraffic = () => {
        if (trafficTimerRef.current) {
            return;
        }
        if (!servers || servers.length === 0) {
            notifyStatus('No servers available for traffic generator');
            return;
        }
        if (!servers.some((srv) => srv.PingStatus)) {
            notifyStatus('No online servers available for traffic generator');
            return;
        }
        const clamped = Math.max(1, trafficRate);
        const interval = Math.max(1000 / clamped, 75);
        trafficTimerRef.current = setInterval(() => {
            fireTestRequest();
        }, interval);
        setTrafficActive(true);
        notifyStatus(`Traffic generator running (${clamped} rps)`);
    };

    const stopTraffic = () => {
        if (!trafficTimerRef.current) {
            return;
        }
        clearInterval(trafficTimerRef.current);
        trafficTimerRef.current = null;
        setTrafficActive(false);
        notifyStatus('Traffic generator stopped');
    };

    const handleRateChange = (value) => {
        setTrafficRate(value);
        restartTrafficTimer(value);
    };

    useEffect(() => {
        if (!trafficActive) {
            return;
        }
        const online = Array.isArray(servers) && servers.some((srv) => srv.PingStatus);
        if (!online) {
            stopTraffic();
            notifyStatus('No online servers available for traffic generator');
        }
    }, [servers, trafficActive]);

    const runScenario = async (scenario) => {
        if (busyScenario) return;

        if (!servers || servers.length === 0) {
            notifyStatus('No servers to test');
            return;
        }
        const onlineServers = servers.filter((srv) => srv.PingStatus);
        if (scenario !== 'recovery' && onlineServers.length === 0) {
            notifyStatus('No online servers available for this scenario');
            return;
        }

        setBusyScenario(scenario);
        const label = SCENARIO_LABELS[scenario] || 'Scenario';
        notifyStatus(`${label} running...`);

        try {
            switch (scenario) {
                case 'failure': {
                    const target = onlineServers[0];
                    await fetch(`/api/servers/${target.ID}/toggle`, { method: 'POST' });
                    break;
                }
                case 'heavy': {
                    const bursts = Array.from({ length: 15 }).map((_, idx) =>
                        fetch(`/api/test?priority=${priority}&burst=${idx}`, { method: 'GET' })
                    );
                    await Promise.allSettled(bursts);
                    break;
                }
                case 'priority': {
                    const priorityBursts = Array.from({ length: 8 }).map(() =>
                        fetch(`/api/test?priority=critical&burst=${Math.random()}`, { method: 'GET' })
                    );
                    await Promise.allSettled(priorityBursts);
                    break;
                }
                case 'recovery': {
                    await Promise.all(
                        servers.map((srv) =>
                            fetch(`/api/servers/${srv.ID}/reset`, { method: 'POST' })
                        )
                    );
                    break;
                }
                default:
                    break;
            }

            if (onScenarioComplete) {
                onScenarioComplete(scenario);
            }
        } catch (error) {
            console.error(`Scenario ${scenario} failed`, error);
            notifyStatus(`${label} failed`);
        } finally {
            setBusyScenario(null);
        }
    };

    return (
        <div className="panel control-deck">
            <div className="deck-section">
                <h3>Traffic Controls</h3>
                <div className="switch-stack">
                    <label className="switch-row">
                        <span className="switch-label">IP Hash Routing</span>
                        <input
                            type="checkbox"
                            checked={useIPHash}
                            onChange={(e) => updateConfig({ useIPHash: e.target.checked })}
                        />
                    </label>
                    <label className="switch-row">
                        <span className="switch-label">Sticky Sessions</span>
                        <input
                            type="checkbox"
                            checked={useSticky}
                            onChange={(e) => updateConfig({ useStickySessions: e.target.checked })}
                        />
                    </label>
                    <label className="switch-row">
                        <span className="switch-label">Priority Focus</span>
                        <select
                            value={priority}
                            onChange={(e) => setPriority(e.target.value)}
                        >
                            {PRIORITY_OPTIONS.map((opt) => (
                                <option key={opt.value} value={opt.value}>
                                    {opt.label}
                                </option>
                            ))}
                        </select>
                    </label>
                </div>
                <div className="dial-stack">
                    <RpsDial value={trafficRate} onChange={handleRateChange} />
                    <div className="dial-readout">
                        <span className="dial-title">Requests / Second</span>
                        <span className="dial-value">{trafficRate}</span>
                    </div>
                </div>
                <div className="scenario-column">
                    <button
                        className={`scenario-button ${trafficActive ? 'running' : ''}`}
                        onClick={startTraffic}
                        disabled={trafficActive || !hasOnlineServers}
                    >
                        Start Traffic
                    </button>
                    <button
                        className="scenario-button"
                        onClick={stopTraffic}
                        disabled={!trafficActive}
                    >
                        Stop Traffic
                    </button>
                </div>
            </div>

            <div className="deck-section">
                <h3>Scenario Lab</h3>
                <div className="scenario-column">
                    {Object.entries(SCENARIO_LABELS).map(([key, label]) => (
                        <button
                            key={key}
                            className={`scenario-button ${busyScenario === key ? 'running' : ''}`}
                            onClick={() => runScenario(key)}
                            disabled={
                                !!busyScenario ||
                                !hasServers ||
                                (key !== 'recovery' && !hasOnlineServers)
                            }
                        >
                            {label}
                        </button>
                    ))}
                </div>
            </div>

            <div className="deck-section">
                <h3>Server Enrollment</h3>
                <AddServerForm onRefresh={onServerAdded} />
            </div>
        </div>
    );
};

export default ControlPanel;



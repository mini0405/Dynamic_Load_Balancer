import React, { useEffect, useMemo, useRef, useState } from 'react';
import usePacketFeed from '../hooks/usePacketFeed';

const STATUS_COLORS = {
    dispatch: '#00faff',
    rerouted: '#fcee0b',
    failed: '#ff3b6b',
    completed: '#46ffb9'
};

function TrafficVisualizer({ servers }) {
    const containerRef = useRef(null);
    const balancerRef = useRef(null);
    const serverRefs = useRef({});
    const [particles, setParticles] = useState([]);
    const { latest } = usePacketFeed(0);

    const activeServers = useMemo(() => servers || [], [servers]);
    useEffect(() => {
        const ids = new Set(activeServers.map((srv) => srv.ID));
        Object.keys(serverRefs.current).forEach((id) => {
            if (!ids.has(id)) {
                delete serverRefs.current[id];
            }
        });
    }, [activeServers]);

    useEffect(() => {
        if (!latest || !latest.serverId) {
            return;
        }

        const status = latest.status || 'dispatch';
        const targetRef = serverRefs.current[latest.serverId];
        const balancerNode = balancerRef.current;
        const container = containerRef.current;

        if (!targetRef || !balancerNode || !container) {
            return;
        }

        const containerRect = container.getBoundingClientRect();
        const balancerRect = balancerNode.getBoundingClientRect();
        const targetRect = targetRef.getBoundingClientRect();

        const originX = balancerRect.left + balancerRect.width / 2 - containerRect.left;
        const originY = balancerRect.top + balancerRect.height / 2 - containerRect.top;

        const targetX = targetRect.left + targetRect.width / 2 - containerRect.left;
        const targetY = targetRect.top + targetRect.height / 2 - containerRect.top;

        const particle = {
            id: `${latest.requestId}-${latest.attempt}-${status}-${Date.now()}`,
            dx: targetX - originX,
            dy: targetY - originY,
            color: STATUS_COLORS[status] || STATUS_COLORS.dispatch,
            offsetX: originX,
            offsetY: originY,
        };

        setParticles((prev) => [...prev, particle]);
    }, [latest]);

    useEffect(() => {
        if (particles.length === 0) {
            return;
        }
        const timeout = setTimeout(() => {
            setParticles((prev) => prev.slice(1));
        }, 1100);
        return () => clearTimeout(timeout);
    }, [particles]);

    return (
        <div className="panel traffic-visual">
            <div className="traffic-header">
                <h2>Flow Mapper</h2>
                <span className="traffic-subtitle">Live packet animation</span>
            </div>
            <div className="traffic-stage" ref={containerRef}>
                <div className="balancer-node" ref={balancerRef}>
                    <div className="node-core">LB</div>
                    <span className="node-label">Balancer Core</span>
                </div>
                <div className="server-column">
                    {activeServers.map((server, index) => (
                        <div
                            key={server.ID}
                            className="server-node"
                            ref={(el) => { serverRefs.current[server.ID] = el; }}
                        >
                            <div className="node-core">{index + 1}</div>
                            <span className="node-label">{server.ID}</span>
                            <span className="node-meta">
                                {server.CircuitBreakerState === 0 ? 'Online' : 'Offline'}
                            </span>
                        </div>
                    ))}
                </div>
                <div className="particle-layer">
                    {particles.map((particle) => (
                        <span
                            key={particle.id}
                            className="traffic-particle"
                            style={{
                                '--dx': `${particle.dx}px`,
                                '--dy': `${particle.dy}px`,
                                left: particle.offsetX,
                                top: particle.offsetY,
                                backgroundColor: particle.color,
                            }}
                        />
                    ))}
                </div>
            </div>
            <div className="traffic-legend">
                <div><span className="legend-swatch" style={{ backgroundColor: STATUS_COLORS.dispatch }} /> Dispatch</div>
                <div><span className="legend-swatch" style={{ backgroundColor: STATUS_COLORS.completed }} /> Completed</div>
                <div><span className="legend-swatch" style={{ backgroundColor: STATUS_COLORS.rerouted }} /> Rerouted</div>
                <div><span className="legend-swatch" style={{ backgroundColor: STATUS_COLORS.failed }} /> Failed</div>
            </div>
        </div>
    );
}

export default TrafficVisualizer;


import React, { useMemo } from 'react';
import usePacketFeed from '../hooks/usePacketFeed';

const STATUS_LABELS = {
    dispatch: 'Dispatched',
    rerouted: 'Rerouted',
    failed: 'Failed',
    completed: 'Completed'
};

const PRIORITY_LABELS = {
    critical: 'Critical',
    high: 'High',
    medium: 'Medium',
    normal: 'Normal',
    low: 'Low'
};

const PRIORITY_ORDER = {
    critical: 0,
    high: 1,
    medium: 2,
    normal: 3,
    low: 4
};

function groupEvents(events) {
    const map = new Map();

    events.forEach(evt => {
        const id = evt.requestId;
        if (!id) {
            return;
        }
        if (!map.has(id)) {
            map.set(id, {
                id,
                priority: evt.priority || 'normal',
                attempts: [],
                lastUpdated: evt.timestamp
            });
        }

        const entry = map.get(id);
        entry.priority = evt.priority || entry.priority;
        entry.lastUpdated = evt.timestamp || entry.lastUpdated;
        entry.attempts.push(evt);
    });

    const list = Array.from(map.values());
    list.forEach(item => {
        item.attempts.sort((a, b) => {
            if (a.attempt === b.attempt) {
                return new Date(a.timestamp) - new Date(b.timestamp);
            }
            return a.attempt - b.attempt;
        });
    });

    list.sort((a, b) => {
        const priorityDelta = (PRIORITY_ORDER[a.priority] ?? 99) - (PRIORITY_ORDER[b.priority] ?? 99);
        if (priorityDelta !== 0) {
            return priorityDelta;
        }
        return new Date(b.lastUpdated).getTime() - new Date(a.lastUpdated).getTime();
    });

    return list;
}

function formatTime(timestamp) {
    if (!timestamp) {
        return '';
    }
    const date = new Date(timestamp);
    if (Number.isNaN(date.getTime())) {
        return '';
    }
    return date.toLocaleTimeString();
}

function PacketStream() {
    const { events } = usePacketFeed(160);

    const requests = useMemo(() => groupEvents(events), [events]);

    return (
        <div className="panel packet-stream">
            <div className="packet-stream-header">
                <h2>Packet Flow</h2>
                <span className="packet-stream-subtitle">
                    Live distribution &amp; reroute telemetry
                </span>
            </div>

            <div className="packet-stream-body">
                {requests.length === 0 ? (
                    <div className="packet-empty">Waiting for traffic...</div>
                ) : (
                    requests.slice(0, 12).map(request => (
                        <div
                            key={request.id}
                            className={`packet-row priority-${request.priority || 'normal'}`}
                        >
                            <div className="packet-row-header">
                                <span className="packet-id">{request.id}</span>
                                <span className={`priority-tag priority-${request.priority || 'normal'}`}>
                                    {PRIORITY_LABELS[request.priority] || 'Normal'} Priority
                                </span>
                                <span className="packet-updated">{formatTime(request.lastUpdated)}</span>
                            </div>

                            <div className="packet-attempts">
                                {request.attempts.map(attempt => (
                                    <div
                                        key={`${request.id}-attempt-${attempt.attempt}-${attempt.status}`}
                                        className={`packet-attempt packet-${attempt.status}`}
                                    >
                                        <div className="packet-attempt-header">
                                            <span className="packet-attempt-label">
                                                Attempt {attempt.attempt}
                                            </span>
                                            <span className="packet-status">
                                                {STATUS_LABELS[attempt.status] || attempt.status}
                                            </span>
                                        </div>
                                        <div className="packet-attempt-meta">
                                            <span className="packet-server">{attempt.serverId}</span>
                                            <span className="packet-address">{attempt.serverAddress}</span>
                                        </div>
                                        <div className="packet-attempt-footer">
                                            <span className="packet-active">
                                                Active: {attempt.activeRequests ?? 0}
                                            </span>
                                            <span className="packet-response">
                                                {attempt.responseTime
                                                    ? `${Math.round(attempt.responseTime)}ms`
                                                    : 'pending'}
                                            </span>
                                        </div>
                                        {attempt.reason && (
                                            <div className="packet-reason">
                                                {attempt.reason}
                                            </div>
                                        )}
                                    </div>
                                ))}
                            </div>
                        </div>
                    ))
                )}
            </div>
        </div>
    );
}

export default PacketStream;

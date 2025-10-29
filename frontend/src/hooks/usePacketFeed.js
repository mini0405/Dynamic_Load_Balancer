import { useEffect, useRef, useState } from 'react';

let eventSource = null;
const subscribers = new Set();

function startEventSource() {
    if (eventSource) {
        return;
    }

    eventSource = new EventSource('/api/events');
    eventSource.onmessage = (event) => {
        try {
            const payload = JSON.parse(event.data);
            if (payload.type !== 'packet') {
                return;
            }
            const packet = JSON.parse(payload.message);
            packet.timestamp = packet.timestamp || payload.timestamp;
            subscribers.forEach((cb) => cb(packet));
        } catch (error) {
            console.error('Failed to parse packet event', error);
        }
    };

    eventSource.onerror = () => {
        if (eventSource) {
            eventSource.close();
            eventSource = null;
        }
        setTimeout(startEventSource, 2000);
    };
}

function stopEventSource() {
    if (eventSource && subscribers.size === 0) {
        eventSource.close();
        eventSource = null;
    }
}

export default function usePacketFeed(limit = 120) {
    const [events, setEvents] = useState([]);
    const [latest, setLatest] = useState(null);
    const limitRef = useRef(limit);
    limitRef.current = limit;

    useEffect(() => {
        let active = true;

        const loadHistory = async () => {
            if (limitRef.current <= 0) {
                return;
            }
            try {
                const response = await fetch(`/api/packets?limit=${limitRef.current}`);
                if (!response.ok) {
                    throw new Error(`Failed to load packet history (${response.status})`);
                }
                const data = await response.json();
                if (!active) return;
                const history = Array.isArray(data.events) ? data.events : [];
                setEvents(history);
            } catch (error) {
                console.error('Error loading packet history:', error);
            }
        };

        loadHistory();

        const listener = (packet) => {
            if (!active) {
                return;
            }
            setEvents((prev) => {
                const next = [...prev, packet];
                if (next.length > limitRef.current) {
                    next.shift();
                }
                return next;
            });
            setLatest(packet);
        };

        subscribers.add(listener);
        startEventSource();

        return () => {
            active = false;
            subscribers.delete(listener);
            stopEventSource();
        };
    }, []);

    return { events, latest };
}

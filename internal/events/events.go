// internal/events/events.go
package events

import (
	"encoding/json"
	"log"
	"sync"
	"time"
)

// EventType defines the type of event
type EventType string

const (
	// Event types
	InfoEvent    EventType = "info"
	SuccessEvent EventType = "success"
	WarningEvent EventType = "warning"
	ErrorEvent   EventType = "error"
)

// Event represents a system event
type Event struct {
	Type      EventType `json:"type"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// Subscriber is a channel that receives event notifications
type Subscriber chan string

// EventSystem manages pub-sub for system events
type EventSystem struct {
	subscribers      map[Subscriber]bool
	subscribersMutex sync.RWMutex
	events           []Event
	eventsMutex      sync.RWMutex
	maxEvents        int
}

// NewEventSystem creates a new event system
func NewEventSystem(maxEvents int) *EventSystem {
	if maxEvents <= 0 {
		maxEvents = 100 // Default to 100 events in history
	}

	return &EventSystem{
		subscribers: make(map[Subscriber]bool),
		events:      make([]Event, 0, maxEvents),
		maxEvents:   maxEvents,
	}
}

// Subscribe registers a new subscriber channel
func (es *EventSystem) Subscribe() Subscriber {
	es.subscribersMutex.Lock()
	defer es.subscribersMutex.Unlock()

	subscriber := make(Subscriber, 10) // Buffer size of 10
	es.subscribers[subscriber] = true

	return subscriber
}

// Unsubscribe removes a subscriber
func (es *EventSystem) Unsubscribe(subscriber Subscriber) {
	es.subscribersMutex.Lock()
	defer es.subscribersMutex.Unlock()

	if _, exists := es.subscribers[subscriber]; exists {
		delete(es.subscribers, subscriber)
		close(subscriber)
	}
}

// Publish broadcasts an event to all subscribers
func (es *EventSystem) Publish(eventType EventType, message string) {
	event := Event{
		Type:      eventType,
		Message:   message,
		Timestamp: time.Now(),
	}

	// Store event in history
	es.eventsMutex.Lock()
	if len(es.events) >= es.maxEvents {
		// Remove oldest event
		es.events = append(es.events[1:], event)
	} else {
		es.events = append(es.events, event)
	}
	es.eventsMutex.Unlock()

	// Marshal event to JSON
	eventJSON, err := json.Marshal(event)
	if err != nil {
		log.Printf("Error marshaling event: %v", err)
		return
	}

	// Send to all subscribers
	es.subscribersMutex.RLock()
	defer es.subscribersMutex.RUnlock()

	for subscriber := range es.subscribers {
		// Non-blocking send
		select {
		case subscriber <- string(eventJSON):
			// Message sent successfully
		default:
			// Channel buffer is full, log and continue
			log.Printf("Subscriber channel full, event dropped")
		}
	}
}

// GetRecentEvents returns recent events from history
func (es *EventSystem) GetRecentEvents(limit int) []Event {
	es.eventsMutex.RLock()
	defer es.eventsMutex.RUnlock()

	if limit <= 0 || limit > len(es.events) {
		limit = len(es.events)
	}

	// Get the most recent events up to the limit
	start := len(es.events) - limit
	if start < 0 {
		start = 0
	}

	// Create a copy of the slice to return
	result := make([]Event, len(es.events[start:]))
	copy(result, es.events[start:])

	return result
}

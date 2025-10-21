package events

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"jsondrop/internal/models"
)

// Broadcaster manages SSE connections and event distribution
type Broadcaster struct {
	mu                  sync.RWMutex
	databaseListeners   map[string]map[*Listener]bool            // dbID -> listeners
	collectionListeners map[string]map[string]map[*Listener]bool // dbID -> collection -> listeners
}

// Listener represents a single SSE connection
type Listener struct {
	ID       string
	Events   chan models.ChangeEvent
	Done     chan bool
	LastPing time.Time
}

// NewBroadcaster creates a new event broadcaster
func NewBroadcaster() *Broadcaster {
	b := &Broadcaster{
		databaseListeners:   make(map[string]map[*Listener]bool),
		collectionListeners: make(map[string]map[string]map[*Listener]bool),
	}

	// Start cleanup goroutine for dead connections
	go b.cleanupRoutine()

	return b
}

// Subscribe adds a listener for database-level events
func (b *Broadcaster) Subscribe(dbID string) *Listener {
	listener := &Listener{
		ID:       generateListenerID(),
		Events:   make(chan models.ChangeEvent, 10),
		Done:     make(chan bool),
		LastPing: time.Now(),
	}

	b.mu.Lock()
	if b.databaseListeners[dbID] == nil {
		b.databaseListeners[dbID] = make(map[*Listener]bool)
	}
	b.databaseListeners[dbID][listener] = true
	b.mu.Unlock()

	return listener
}

// Unsubscribe removes a listener
func (b *Broadcaster) Unsubscribe(dbID string, listener *Listener) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if listeners, exists := b.databaseListeners[dbID]; exists {
		delete(listeners, listener)
		if len(listeners) == 0 {
			delete(b.databaseListeners, dbID)
		}
	}

	close(listener.Done)
}

// SubscribeCollection adds a listener for collection-specific events
func (b *Broadcaster) SubscribeCollection(dbID string, collection string) *Listener {
	listener := &Listener{
		ID:       generateListenerID(),
		Events:   make(chan models.ChangeEvent, 10),
		Done:     make(chan bool),
		LastPing: time.Now(),
	}

	b.mu.Lock()
	if b.collectionListeners[dbID] == nil {
		b.collectionListeners[dbID] = make(map[string]map[*Listener]bool)
	}
	if b.collectionListeners[dbID][collection] == nil {
		b.collectionListeners[dbID][collection] = make(map[*Listener]bool)
	}
	b.collectionListeners[dbID][collection][listener] = true
	b.mu.Unlock()

	return listener
}

// UnsubscribeCollection removes a collection listener
func (b *Broadcaster) UnsubscribeCollection(dbID string, collection string, listener *Listener) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if collections, exists := b.collectionListeners[dbID]; exists {
		if listeners, exists := collections[collection]; exists {
			delete(listeners, listener)
			if len(listeners) == 0 {
				delete(collections, collection)
			}
		}
		if len(collections) == 0 {
			delete(b.collectionListeners, dbID)
		}
	}

	close(listener.Done)
}

// Broadcast sends an event to all listeners for a database and specific collection
func (b *Broadcaster) Broadcast(dbID string, event models.ChangeEvent) {
	b.mu.RLock()
	databaseListeners := b.databaseListeners[dbID]
	var collectionListeners map[*Listener]bool
	if collections, exists := b.collectionListeners[dbID]; exists {
		collectionListeners = collections[event.Collection]
	}
	b.mu.RUnlock()

	// Send to database-level listeners
	for listener := range databaseListeners {
		select {
		case listener.Events <- event:
			// Event sent successfully
		default:
			// Channel full, skip this listener
			// TODO: Add logging
		}
	}

	// Send to collection-specific listeners
	for listener := range collectionListeners {
		select {
		case listener.Events <- event:
			// Event sent successfully
		default:
			// Channel full, skip this listener
			// TODO: Add logging
		}
	}
}

// GetListenerCount returns the number of active listeners for a database
func (b *Broadcaster) GetListenerCount(dbID string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if listeners, exists := b.databaseListeners[dbID]; exists {
		return len(listeners)
	}
	return 0
}

// cleanupRoutine periodically removes stale connections
func (b *Broadcaster) cleanupRoutine() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		b.mu.Lock()

		// Cleanup database-level listeners
		for dbID, listeners := range b.databaseListeners {
			for listener := range listeners {
				// Remove listeners that haven't been pinged in 2 minutes
				if time.Since(listener.LastPing) > 2*time.Minute {
					delete(listeners, listener)
					close(listener.Done)
				}
			}
			// Clean up empty database entries
			if len(listeners) == 0 {
				delete(b.databaseListeners, dbID)
			}
		}

		// Cleanup collection-level listeners
		for dbID, collections := range b.collectionListeners {
			for collection, listeners := range collections {
				for listener := range listeners {
					// Remove listeners that haven't been pinged in 2 minutes
					if time.Since(listener.LastPing) > 2*time.Minute {
						delete(listeners, listener)
						close(listener.Done)
					}
				}
				// Clean up empty collection entries
				if len(listeners) == 0 {
					delete(collections, collection)
				}
			}
			// Clean up empty database entries
			if len(collections) == 0 {
				delete(b.collectionListeners, dbID)
			}
		}

		b.mu.Unlock()
	}
}

// UpdatePing updates the last ping time for a listener
func (b *Broadcaster) UpdatePing(listener *Listener) {
	listener.LastPing = time.Now()
}

// FormatSSE formats an event as Server-Sent Events format
func FormatSSE(event models.ChangeEvent) string {
	data, _ := json.Marshal(event)
	return fmt.Sprintf("event: change\ndata: %s\n\n", string(data))
}

// FormatPing formats a ping/heartbeat message
func FormatPing() string {
	return ": ping\n\n"
}

// generateListenerID generates a unique listener ID
func generateListenerID() string {
	return fmt.Sprintf("listener_%d", time.Now().UnixNano())
}

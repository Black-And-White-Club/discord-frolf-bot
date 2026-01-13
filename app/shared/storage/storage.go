package storage

import (
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
)

// ISInterface defines the behavior for the interaction store
type ISInterface interface {
	Set(correlationID string, interaction interface{}, ttl time.Duration) error
	Delete(correlationID string)
	Get(correlationID string) (interface{}, bool)
}

// interactionItem holds the data and the active timer for a specific key
type interactionItem struct {
	value interface{}
	timer *time.Timer
}

// InteractionStore manages short-lived interaction tokens
type InteractionStore struct {
	store map[string]*interactionItem
	mu    sync.RWMutex // Protects the store map
}

// NewInteractionStore initializes a new InteractionStore
func NewInteractionStore() ISInterface {
	return &InteractionStore{
		store: make(map[string]*interactionItem),
	}
}

// Set stores an interaction token with an auto-expiring timer
func (ts *InteractionStore) Set(correlationID string, interaction interface{}, ttl time.Duration) error {
	slog.Info("InteractionStore: Storing interaction", attr.String("correlation_id", correlationID))
	if correlationID == "" {
		return errors.New("correlation ID is empty")
	}
	if interaction == nil {
		return errors.New("interaction is nil")
	}
	if ttl <= 0 {
		return errors.New("TTL must be greater than 0")
	}

	ts.mu.Lock()
	defer ts.mu.Unlock()

	// 1. If an entry already exists, stop its old timer to prevent leaks
	if existing, exists := ts.store[correlationID]; exists {
		if existing.timer != nil {
			existing.timer.Stop()
		}
	}

	// 2. Define the cleanup function
	// We wrap the cleanup logic so it acquires its own lock later
	cleanupAction := func() {
		ts.mu.Lock()
		defer ts.mu.Unlock()

		// Verify the item still exists and hasn't been replaced
		// (Though simpler here because we replaced the timer above, extra safety doesn't hurt)
		if _, exists := ts.store[correlationID]; exists {
			delete(ts.store, correlationID)
			slog.Info("InteractionStore: Interaction deleted due to TTL expiration", attr.String("correlation_id", correlationID))
		}
	}

	// 3. Create the new item with a new timer
	ts.store[correlationID] = &interactionItem{
		value: interaction,
		timer: time.AfterFunc(ttl, cleanupAction),
	}

	return nil
}

// Delete removes the token associated with the given correlation ID.
func (ts *InteractionStore) Delete(correlationID string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	slog.Info("InteractionStore: Deleting interaction", attr.String("correlation_id", correlationID))

	if item, exists := ts.store[correlationID]; exists {
		// Stop the timer immediately so it doesn't fire later
		if item.timer != nil {
			item.timer.Stop()
		}
		delete(ts.store, correlationID)
		slog.Info("InteractionStore: Interaction deleted successfully", attr.String("correlation_id", correlationID))
	} else {
		slog.Info("InteractionStore: Interaction not found", attr.String("correlation_id", correlationID))
	}
}

// Get retrieves the token
func (ts *InteractionStore) Get(correlationID string) (interface{}, bool) {
	ts.mu.RLock() // Use Read Lock for concurrent access
	defer ts.mu.RUnlock()

	slog.Info("InteractionStore: Retrieving interaction", attr.String("correlation_id", correlationID))

	item, exists := ts.store[correlationID]
	if !exists {
		slog.Info("InteractionStore: Interaction not found", attr.String("correlation_id", correlationID))
		return nil, false
	}
	return item.value, true
}

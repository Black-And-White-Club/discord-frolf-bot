package storage

import (
	"log/slog"
	"sync"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
)

// InteractionStore manages short-lived interaction tokens
type InteractionStore struct {
	store sync.Map
	mu    sync.Mutex // Mutex to protect access to the store
}

// NewInteractionStore initializes a new InteractionStore
func NewInteractionStore() *InteractionStore {
	return &InteractionStore{}
}

// Set stores an interaction token with an auto-expiring timer
func (ts *InteractionStore) Set(correlationID string, interaction interface{}, ttl time.Duration) {
	slog.Info("InteractionStore: Storing interaction", attr.String("correlation_id", correlationID))
	ts.store.Store(correlationID, interaction)

	// 🔥 Auto-remove token after TTL expires
	time.AfterFunc(ttl, func() {
		slog.Info("InteractionStore: Deleting interaction (TTL expired)", attr.String("correlation_id", correlationID))
		ts.mu.Lock()
		defer ts.mu.Unlock()
		if _, exists := ts.store.Load(correlationID); exists {
			ts.store.Delete(correlationID)
			slog.Info("InteractionStore: Interaction deleted due to TTL expiration", attr.String("correlation_id", correlationID))
		} else {
			slog.Info("InteractionStore: Interaction already deleted", attr.String("correlation_id", correlationID))
		}
	})
}

// Delete removes the token associated with the given correlation ID.
func (ts *InteractionStore) Delete(correlationID string) {
	ts.mu.Lock() // Lock to ensure safe access
	defer ts.mu.Unlock()
	slog.Info("InteractionStore: Deleting interaction", attr.String("correlation_id", correlationID))
	if _, exists := ts.store.Load(correlationID); !exists {
		slog.Info("InteractionStore: Interaction not found", attr.String("correlation_id", correlationID))
		return // Token not found, nothing to delete
	}
	ts.store.Delete(correlationID)
	slog.Info("InteractionStore: Interaction deleted successfully", attr.String("correlation_id", correlationID))
}

// Get retrieves the token (one-time use)
func (ts *InteractionStore) Get(correlationID string) (interface{}, bool) {
	ts.mu.Lock() // Lock to ensure safe access
	defer ts.mu.Unlock()
	slog.Info("InteractionStore: Retrieving interaction", attr.String("correlation_id", correlationID))
	value, exists := ts.store.Load(correlationID)
	if !exists {
		slog.Info("InteractionStore: Interaction not found", attr.String("correlation_id", correlationID))
		return nil, false
	}
	return value, true
}

package storage

import (
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
)

type ISInterface interface {
	Set(correlationID string, interaction interface{}, ttl time.Duration) error
	Delete(correlationID string)
	Get(correlationID string) (interface{}, bool)
}

// InteractionStore manages short-lived interaction tokens
type InteractionStore struct {
	store sync.Map
	mu    sync.Mutex // Mutex to protect access to the store
}

// NewInteractionStore initializes a new InteractionStore
func NewInteractionStore() ISInterface {
	return &InteractionStore{}
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
	return nil
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

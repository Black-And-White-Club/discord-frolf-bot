package storage

import (
	"log/slog"
	"sync"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
)

// TokenStore manages short-lived interaction tokens
type TokenStore struct {
	store sync.Map
}

// NewTokenStore initializes a new TokenStore
func NewTokenStore() *TokenStore {
	return &TokenStore{}
}

// Set stores an interaction token with an auto-expiring timer
func (ts *TokenStore) Set(interactionID, token string, ttl time.Duration) {
	slog.Info("TokenStore: Storing token", attr.String("interaction_id", interactionID))
	ts.store.Store(interactionID, token)

	// 🔥 Auto-remove token after TTL expires
	time.AfterFunc(ttl, func() {
		slog.Info("TokenStore: Deleting token (TTL expired)", attr.String("interaction_id", interactionID))
		ts.store.Delete(interactionID)
	})
}

// Get retrieves and deletes the token (one-time use)
func (ts *TokenStore) Get(interactionID string) (string, bool) {
	slog.Info("TokenStore: Retrieving token", attr.String("interaction_id", interactionID))
	value, exists := ts.store.Load(interactionID)
	if !exists {
		slog.Info("TokenStore: Token not found", attr.String("interaction_id", interactionID))
		return "", false
	}
	ts.store.Delete(interactionID) // 🔥 Remove token after retrieval (one-time use)
	slog.Info("TokenStore: Token retrieved and deleted", attr.String("interaction_id", interactionID))
	return value.(string), true
}

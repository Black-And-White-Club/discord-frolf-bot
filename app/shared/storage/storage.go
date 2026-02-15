package storage

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
)

// ISInterface defines the behavior for the interaction store using Generics [T]
type ISInterface[T any] interface {
	Set(ctx context.Context, correlationID string, interaction T) error // Removed TTL from here if you prefer a fixed default, or keep it if you want dynamic
	Delete(ctx context.Context, correlationID string)
	Get(ctx context.Context, correlationID string) (T, error)
}

// interactionItem holds the data and the expiration timestamp
type interactionItem[T any] struct {
	value      T
	expiryTime int64 // UnixNano for high-performance comparison
}

// InteractionStore manages short-lived interaction tokens
type InteractionStore[T any] struct {
	store map[string]interactionItem[T]
	mu    sync.RWMutex
	ttl   time.Duration // Added a default TTL for this store instance
}

// Stores hub remains the same
type Stores struct {
	InteractionStore ISInterface[any]
	GuildConfigCache ISInterface[GuildConfig]
}

const (
	defaultInteractionStoreTTL   = 1 * time.Hour
	defaultGuildConfigCacheSize  = 10000
	defaultGuildConfigCacheTTL   = 24 * time.Hour
	defaultGuildConfigRefreshTTL = 2 * time.Hour
)

func NewStores(ctx context.Context) *Stores {
	guildConfigCache := NewGuildConfigCache(
		ctx,
		defaultGuildConfigCacheSize,
		defaultGuildConfigCacheTTL,
		defaultGuildConfigRefreshTTL,
	)

	return &Stores{
		InteractionStore: NewInteractionStore[any](ctx, defaultInteractionStoreTTL),
		GuildConfigCache: newGuildConfigStoreAdapter(guildConfigCache),
	}
}

type guildConfigStoreAdapter struct {
	cache GuildConfigCacheInterface
}

func newGuildConfigStoreAdapter(cache GuildConfigCacheInterface) ISInterface[GuildConfig] {
	return &guildConfigStoreAdapter{cache: cache}
}

func (a *guildConfigStoreAdapter) Set(_ context.Context, correlationID string, interaction GuildConfig) error {
	return a.cache.Set(correlationID, &interaction)
}

func (a *guildConfigStoreAdapter) Delete(_ context.Context, correlationID string) {
	a.cache.Delete(correlationID)
}

func (a *guildConfigStoreAdapter) Get(_ context.Context, correlationID string) (GuildConfig, error) {
	config, ok := a.cache.Get(correlationID)
	if !ok || config == nil {
		return GuildConfig{}, errors.New("item not found or expired")
	}
	return *config, nil
}

func NewInteractionStore[T any](ctx context.Context, ttl time.Duration) ISInterface[T] {
	is := &InteractionStore[T]{
		store: make(map[string]interactionItem[T]),
		ttl:   ttl,
	}
	// The cleanup interval can be a fraction of the TTL or a fixed 1m
	go is.startJanitor(ctx, 1*time.Minute)
	return is
}

// Set now matches the interface with context
func (ts *InteractionStore[T]) Set(ctx context.Context, correlationID string, interaction T) error {
	if correlationID == "" {
		return errors.New("correlation ID is empty")
	}

	ts.mu.Lock()
	defer ts.mu.Unlock()

	ts.store[correlationID] = interactionItem[T]{
		value:      interaction,
		expiryTime: time.Now().Add(ts.ttl).UnixNano(),
	}

	slog.DebugContext(ctx, "InteractionStore: Stored item", attr.String("correlation_id", correlationID))
	return nil
}

// Get now returns (T, error) and accepts context
func (ts *InteractionStore[T]) Get(ctx context.Context, correlationID string) (T, error) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	item, exists := ts.store[correlationID]
	if !exists || time.Now().UnixNano() > item.expiryTime {
		var zero T
		return zero, errors.New("item not found or expired")
	}

	return item.value, nil
}

// Delete now accepts context
func (ts *InteractionStore[T]) Delete(ctx context.Context, correlationID string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	delete(ts.store, correlationID)
}

// startJanitor runs in the background and removes expired keys at a fixed interval
func (ts *InteractionStore[T]) startJanitor(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	slog.Info("🧹 InteractionStore janitor started", attr.Duration("interval", interval))

	for {
		select {
		case <-ctx.Done():
			slog.Info("🧹 InteractionStore janitor stopping")
			return
		case <-ticker.C:
			ts.performCleanup()
		}
	}
}

func (ts *InteractionStore[T]) performCleanup() {
	now := time.Now().UnixNano()

	ts.mu.Lock()
	defer ts.mu.Unlock()

	initialSize := len(ts.store)
	for id, item := range ts.store {
		if now > item.expiryTime {
			delete(ts.store, id)
		}
	}

	removed := initialSize - len(ts.store)
	if removed > 0 {
		slog.Debug("InteractionStore: Cleanup complete",
			attr.Int("removed_count", removed),
			attr.Int("remaining_count", len(ts.store)))
	}
}

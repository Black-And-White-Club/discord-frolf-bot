package storage

import (
	"container/list"
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
)

// CacheMetrics interface for monitoring cache performance
type CacheMetrics interface {
	RecordHit()
	RecordMiss()
	RecordEviction()
	RecordPendingRequest()
	RecordPendingCleared()
	GetStats() CacheStats
}

// CacheStats contains cache performance statistics
type CacheStats struct {
	Hits            int64
	Misses          int64
	Evictions       int64
	PendingRequests int64
	PendingCleared  int64
	HitRatio        float64
}

// DefaultCacheMetrics implements CacheMetrics with atomic counters
type DefaultCacheMetrics struct {
	hits            int64
	misses          int64
	evictions       int64
	pendingRequests int64
	pendingCleared  int64
}

func NewCacheMetrics() CacheMetrics {
	return &DefaultCacheMetrics{}
}

func (m *DefaultCacheMetrics) RecordHit() {
	atomic.AddInt64(&m.hits, 1)
}

func (m *DefaultCacheMetrics) RecordMiss() {
	atomic.AddInt64(&m.misses, 1)
}

func (m *DefaultCacheMetrics) RecordEviction() {
	atomic.AddInt64(&m.evictions, 1)
}

func (m *DefaultCacheMetrics) RecordPendingRequest() {
	atomic.AddInt64(&m.pendingRequests, 1)
}

func (m *DefaultCacheMetrics) RecordPendingCleared() {
	atomic.AddInt64(&m.pendingCleared, 1)
}

func (m *DefaultCacheMetrics) GetStats() CacheStats {
	hits := atomic.LoadInt64(&m.hits)
	misses := atomic.LoadInt64(&m.misses)
	total := hits + misses

	var hitRatio float64
	if total > 0 {
		hitRatio = float64(hits) / float64(total)
	}

	return CacheStats{
		Hits:            hits,
		Misses:          misses,
		Evictions:       atomic.LoadInt64(&m.evictions),
		PendingRequests: atomic.LoadInt64(&m.pendingRequests),
		PendingCleared:  atomic.LoadInt64(&m.pendingCleared),
		HitRatio:        hitRatio,
	}
}

// GuildConfig represents a cached guild configuration
type GuildConfig struct {
	GuildID              string            `json:"guild_id"`
	SignupChannelID      string            `json:"signup_channel_id"`
	SignupMessageID      string            `json:"signup_message_id"`
	SignupEmoji          string            `json:"signup_emoji"`
	EventChannelID       string            `json:"event_channel_id"`
	LeaderboardChannelID string            `json:"leaderboard_channel_id"`
	RegisteredRoleID     string            `json:"registered_role_id"`
	EditorRoleID         string            `json:"editor_role_id"`
	AdminRoleID          string            `json:"admin_role_id"`
	RoleMappings         map[string]string `json:"role_mappings"`
	CachedAt             time.Time         `json:"cached_at"`
	RefreshedAt          time.Time         `json:"refreshed_at"`       // New: soft refresh timestamp
	IsPlaceholder        bool              `json:"is_placeholder"`     // True if this is just a marker that guild is configured
	IsRequestPending     bool              `json:"is_request_pending"` // True if a config request is currently in flight
}

type GuildConfigCacheInterface interface {
	Get(guildID string) (*GuildConfig, bool)
	Set(guildID string, config *GuildConfig) error
	Delete(guildID string)
	Size() int
	Clear()
	MarkRequestPending(guildID string) bool           // Returns true if request was marked, false if already pending
	ClearRequestPending(guildID string)               // Clears the pending request flag
	GetMetrics() CacheStats                           // Get cache performance metrics
	SetRefreshCallback(callback func(guildID string)) // Set callback for background refresh
}

// GuildConfigCache manages long-lived guild configurations with memory optimization
// Uses LRU eviction with O(1) operations via doubly-linked list + map
type GuildConfigCache struct {
	store           sync.Map
	lruList         *list.List               // Doubly-linked list for O(1) LRU operations
	lruMap          map[string]*list.Element // Map guildID -> list element for O(1) access
	mu              sync.RWMutex
	maxSize         int           // Maximum number of cached configs
	cacheTTL        time.Duration // How long configs stay cached (hard expiry)
	refreshTTL      time.Duration // How long before background refresh is triggered
	metrics         CacheMetrics
	refreshCallback func(guildID string) // Callback for triggering background refresh
}

// lruEntry represents an entry in the LRU list
type lruEntry struct {
	guildID   string
	timestamp time.Time
}

// NewGuildConfigCache creates a new optimized guild config cache with LRU eviction and background refresh
func NewGuildConfigCache(ctx context.Context, maxSize int, cacheTTL, refreshTTL time.Duration) GuildConfigCacheInterface {
	cache := &GuildConfigCache{
		maxSize:    maxSize,
		cacheTTL:   cacheTTL,
		refreshTTL: refreshTTL,
		metrics:    NewCacheMetrics(),
		lruList:    list.New(),
		lruMap:     make(map[string]*list.Element),
	}

	// Start cleanup goroutine for expired entries
	go cache.cleanupExpiredEntries(ctx)

	return cache
}

// Get retrieves a guild config from cache and updates LRU order
func (gc *GuildConfigCache) Get(guildID string) (*GuildConfig, bool) {
	value, exists := gc.store.Load(guildID)
	if !exists {
		gc.metrics.RecordMiss()
		return nil, false
	}

	config := value.(*GuildConfig)

	// Hard expiry: remove from cache if cacheTTL exceeded
	if time.Since(config.CachedAt) > gc.cacheTTL {
		slog.Debug("Guild config cache expired, removing",
			attr.String("guild_id", guildID))
		gc.Delete(guildID) // Use Delete method to properly clean up LRU tracking
		gc.metrics.RecordMiss()
		return nil, false
	}

	// Background refresh trigger: if refreshTTL exceeded and not already pending
	if time.Since(config.RefreshedAt) > gc.refreshTTL && !config.IsRequestPending {
		slog.Debug("Guild config is stale — triggering background refresh",
			attr.String("guild_id", guildID))

		go func() {
			if gc.MarkRequestPending(guildID) {
				// Trigger refresh using callback if available
				gc.mu.RLock()
				callback := gc.refreshCallback
				gc.mu.RUnlock()

				if callback != nil {
					callback(guildID)
				} else {
					slog.Info("Triggering async config refresh", attr.String("guild_id", guildID))
				}
			}
		}()
	}

	// Update LRU order: move to front (most recently used)
	gc.updateLRUOrder(guildID)

	gc.metrics.RecordHit()
	slog.Debug("Guild config cache hit", attr.String("guild_id", guildID))
	return config, true
}

// Set stores a guild config with LRU-style eviction if needed
func (gc *GuildConfigCache) Set(guildID string, config *GuildConfig) error {
	if config == nil {
		return errors.New("config cannot be nil")
	}

	// Set cache and refresh timestamps
	now := time.Now()
	config.CachedAt = now
	config.RefreshedAt = now

	// Check if we need to evict entries before adding new one
	if gc.Size() >= gc.maxSize {
		gc.evictOldestEntry()
	}

	// Store the config
	gc.store.Store(guildID, config)

	// Update LRU order: move to front (most recently used)
	gc.updateLRUOrder(guildID)

	slog.Debug("Guild config cached",
		attr.String("guild_id", guildID),
		attr.Int("cache_size", gc.Size()))

	return nil
}

// Delete removes a guild config from cache and LRU tracking
func (gc *GuildConfigCache) Delete(guildID string) {
	gc.store.Delete(guildID)

	// Remove from LRU tracking
	gc.mu.Lock()
	if element, exists := gc.lruMap[guildID]; exists {
		gc.lruList.Remove(element)
		delete(gc.lruMap, guildID)
	}
	gc.mu.Unlock()

	slog.Debug("Guild config removed from cache", attr.String("guild_id", guildID))
}

// Size returns the current number of cached configs
func (gc *GuildConfigCache) Size() int {
	count := 0
	gc.store.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	return count
}

// Clear removes all cached configs
func (gc *GuildConfigCache) Clear() {
	gc.store.Range(func(key, _ interface{}) bool {
		gc.store.Delete(key)
		return true
	})
	slog.Info("Guild config cache cleared")
}

// evictOldestEntry removes the least recently used entry in O(1) time
func (gc *GuildConfigCache) evictOldestEntry() {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	// Get the least recently used item (tail of list)
	oldest := gc.lruList.Back()
	if oldest == nil {
		return // Empty list
	}

	// Remove from both the LRU list and the cache
	entry := oldest.Value.(*lruEntry)
	gc.lruList.Remove(oldest)
	delete(gc.lruMap, entry.guildID)
	gc.store.Delete(entry.guildID)

	gc.metrics.RecordEviction()
	slog.Debug("Evicted least recently used guild config from cache",
		attr.String("guild_id", entry.guildID))
}

// cleanupExpiredEntries runs periodically to remove expired entries
func (gc *GuildConfigCache) cleanupExpiredEntries(ctx context.Context) {
	ticker := time.NewTicker(gc.cacheTTL / 4) // Cleanup 4x more frequently than TTL
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("🧹 Cache cleanup stopped")
			return
		case <-ticker.C:
			now := time.Now()
			var keysToDelete []interface{}

			// Collect expired keys
			gc.store.Range(func(key, value interface{}) bool {
				config := value.(*GuildConfig)
				if now.Sub(config.CachedAt) > gc.cacheTTL {
					keysToDelete = append(keysToDelete, key)
				}
				return true
			})

			// Delete expired entries
			for _, key := range keysToDelete {
				gc.Delete(key.(string)) // Use Delete method to properly clean up LRU tracking
				slog.Debug("Cleaned up expired guild config",
					attr.String("guild_id", key.(string)))
			}

			if len(keysToDelete) > 0 {
				slog.Debug("Cleaned up expired guild configs",
					attr.Int("removed_count", len(keysToDelete)),
					attr.Int("remaining_count", gc.Size()))
			}
		}
	}
}

// IsConfigured returns true if this guild config represents a fully configured guild
// A guild is considered configured if it has essential channels and roles set up
func (gc *GuildConfig) IsConfigured() bool {
	// If this is a placeholder config, it means the guild is marked as configured
	if gc.IsPlaceholder {
		return true
	}

	// Normal validation for real configs
	return gc.GuildID != "" &&
		gc.SignupChannelID != "" &&
		gc.EventChannelID != "" &&
		gc.LeaderboardChannelID != "" &&
		gc.RegisteredRoleID != ""
}

// MarkRequestPending marks a guild as having a pending config request
// Returns true if the request was marked as pending, false if already pending
// Uses atomic CAS operations with retry loop to prevent race conditions
func (gc *GuildConfigCache) MarkRequestPending(guildID string) bool {
	// Retry loop for CAS operations to avoid stack overflow under high contention
	for retries := 0; retries < 10; retries++ {
		if value, found := gc.store.Load(guildID); found {
			config := value.(*GuildConfig)

			if config.IsRequestPending {
				return false // Already pending
			}

			// Create new config with pending flag set
			newConfig := *config // Copy struct
			newConfig.IsRequestPending = true

			// Atomic compare-and-swap: only store if value hasn't changed
			if gc.store.CompareAndSwap(guildID, value, &newConfig) {
				gc.metrics.RecordPendingRequest()
				return true // Successfully marked as pending
			}
			// If CAS failed, another goroutine modified it, retry
			continue
		}

		// Create a placeholder with pending request
		now := time.Now()
		pendingConfig := &GuildConfig{
			GuildID:          guildID,
			CachedAt:         now,
			RefreshedAt:      now,
			IsPlaceholder:    true,
			IsRequestPending: true,
		}

		// Use LoadOrStore for atomic "create if not exists"
		_, loaded := gc.store.LoadOrStore(guildID, pendingConfig)
		if !loaded {
			// We successfully created the pending entry
			gc.metrics.RecordPendingRequest()
			return true
		}
		// Another goroutine created it first, retry to check if pending
	}

	slog.Debug("Failed to mark request as pending after retries",
		attr.String("guild_id", guildID))
	return false
}

// ClearRequestPending clears the pending request flag for a guild
// Uses atomic CAS operations with retry loop to prevent race conditions
func (gc *GuildConfigCache) ClearRequestPending(guildID string) {
	// Retry loop for CAS operations to avoid stack overflow under high contention
	for retries := 0; retries < 10; retries++ {
		value, found := gc.store.Load(guildID)
		if !found {
			return // Entry doesn't exist
		}

		config := value.(*GuildConfig)
		if !config.IsRequestPending {
			return // Already not pending
		}

		// Create new config with pending flag cleared
		newConfig := *config // Copy struct
		newConfig.IsRequestPending = false

		// Atomic compare-and-swap: only store if value hasn't changed
		if gc.store.CompareAndSwap(guildID, value, &newConfig) {
			gc.metrics.RecordPendingCleared()
			return // Successfully cleared pending flag
		}
		// If CAS failed, another goroutine modified it, retry
	}

	slog.Debug("Failed to clear pending request after retries",
		attr.String("guild_id", guildID))
}

// GetMetrics returns cache performance statistics
func (gc *GuildConfigCache) GetMetrics() CacheStats {
	return gc.metrics.GetStats()
}

// SetRefreshCallback sets the callback function for triggering background refresh
func (gc *GuildConfigCache) SetRefreshCallback(callback func(guildID string)) {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	gc.refreshCallback = callback
}

// updateLRUOrder moves the specified guildID to the front of the LRU list (most recently used)
func (gc *GuildConfigCache) updateLRUOrder(guildID string) {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	now := time.Now()

	// Check if already in LRU tracking
	if element, exists := gc.lruMap[guildID]; exists {
		// Move to front (most recently used)
		entry := element.Value.(*lruEntry)
		entry.timestamp = now
		gc.lruList.MoveToFront(element)
	} else {
		// Add new entry to front
		entry := &lruEntry{
			guildID:   guildID,
			timestamp: now,
		}
		element := gc.lruList.PushFront(entry)
		gc.lruMap[guildID] = element
	}
}

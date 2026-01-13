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
	store           map[string]*list.Element // Direct map to list elements
	lruList         *list.List               // Doubly-linked list for O(1) LRU operations
	mu              sync.RWMutex             // Protects both store and lruList
	maxSize         int                      // Maximum number of cached configs
	cacheTTL        time.Duration            // How long configs stay cached (hard expiry)
	refreshTTL      time.Duration            // How long before background refresh is triggered
	metrics         CacheMetrics
	refreshCallback func(guildID string) // Callback for triggering background refresh
}

// lruEntry represents an entry in the LRU list
// The actual Config data lives here inside the list element
type lruEntry struct {
	guildID string
	config  *GuildConfig
}

// NewGuildConfigCache creates a new optimized guild config cache
func NewGuildConfigCache(ctx context.Context, maxSize int, cacheTTL, refreshTTL time.Duration) GuildConfigCacheInterface {
	cache := &GuildConfigCache{
		store:      make(map[string]*list.Element),
		lruList:    list.New(),
		maxSize:    maxSize,
		cacheTTL:   cacheTTL,
		refreshTTL: refreshTTL,
		metrics:    NewCacheMetrics(),
	}

	// Start cleanup goroutine for expired entries
	go cache.cleanupExpiredEntries(ctx)

	return cache
}

// Get retrieves a guild config from cache and updates LRU order
func (gc *GuildConfigCache) Get(guildID string) (*GuildConfig, bool) {
	// We need a Write Lock because we are modifying the LRU list order
	gc.mu.Lock()
	defer gc.mu.Unlock()

	element, exists := gc.store[guildID]
	if !exists {
		gc.metrics.RecordMiss()
		return nil, false
	}

	entry := element.Value.(*lruEntry)
	config := entry.config

	// Hard expiry: remove from cache if cacheTTL exceeded
	if time.Since(config.CachedAt) > gc.cacheTTL {
		slog.Debug("Guild config cache expired, removing", attr.String("guild_id", guildID))

		// Inline eviction logic for efficiency
		gc.lruList.Remove(element)
		delete(gc.store, guildID)

		gc.metrics.RecordMiss()
		return nil, false
	}

	// Update LRU order: move to front (most recently used)
	gc.lruList.MoveToFront(element)

	// Background refresh trigger: if refreshTTL exceeded, check pending flag
	if time.Since(config.RefreshedAt) > gc.refreshTTL && !config.IsRequestPending {
		slog.Debug("Guild config is stale — attempting to trigger background refresh", attr.String("guild_id", guildID))

		// We trigger the refresh logic in a separate goroutine to avoid blocking the Get return.
		// We pass the guildID to a helper function.
		go gc.triggerAsyncRefresh(guildID)
	}

	gc.metrics.RecordHit()
	slog.Debug("Guild config cache hit", attr.String("guild_id", guildID))
	return config, true
}

// Set stores a guild config with LRU-style eviction if needed
func (gc *GuildConfigCache) Set(guildID string, config *GuildConfig) error {
	if config == nil {
		return errors.New("config cannot be nil")
	}

	gc.mu.Lock()
	defer gc.mu.Unlock()

	now := time.Now()
	config.CachedAt = now
	config.RefreshedAt = now

	// Check if update existing
	if element, exists := gc.store[guildID]; exists {
		entry := element.Value.(*lruEntry)
		entry.config = config
		gc.lruList.MoveToFront(element)
		return nil
	}

	// Check capacity for new entry
	if gc.lruList.Len() >= gc.maxSize {
		gc.evictOldestEntry()
	}

	// Add new entry
	newEntry := &lruEntry{
		guildID: guildID,
		config:  config,
	}
	element := gc.lruList.PushFront(newEntry)
	gc.store[guildID] = element

	slog.Debug("Guild config cached",
		attr.String("guild_id", guildID),
		attr.Int("cache_size", gc.lruList.Len()))

	return nil
}

// Delete removes a guild config from cache and LRU tracking
func (gc *GuildConfigCache) Delete(guildID string) {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	if element, exists := gc.store[guildID]; exists {
		gc.lruList.Remove(element)
		delete(gc.store, guildID)
		slog.Debug("Guild config removed from cache", attr.String("guild_id", guildID))
	}
}

// Size returns the current number of cached configs
func (gc *GuildConfigCache) Size() int {
	gc.mu.RLock()
	defer gc.mu.RUnlock()
	return len(gc.store)
}

// Clear removes all cached configs
func (gc *GuildConfigCache) Clear() {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	gc.store = make(map[string]*list.Element)
	gc.lruList.Init() // Clears the list
	slog.Info("Guild config cache cleared")
}

// evictOldestEntry removes the least recently used entry
// Caller must hold the lock
func (gc *GuildConfigCache) evictOldestEntry() {
	oldest := gc.lruList.Back()
	if oldest == nil {
		return
	}

	entry := oldest.Value.(*lruEntry)
	gc.lruList.Remove(oldest)
	delete(gc.store, entry.guildID)

	gc.metrics.RecordEviction()
	slog.Debug("Evicted least recently used guild config from cache", attr.String("guild_id", entry.guildID))
}

// cleanupExpiredEntries runs periodically to remove expired entries
func (gc *GuildConfigCache) cleanupExpiredEntries(ctx context.Context) {
	ticker := time.NewTicker(gc.cacheTTL / 4)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("🧹 Cache cleanup stopped")
			return
		case <-ticker.C:
			gc.performCleanup()
		}
	}
}

// performCleanup handles the locking and deletion of expired keys
func (gc *GuildConfigCache) performCleanup() {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	now := time.Now()
	var expiredElements []*list.Element

	// Iterate through the store to find expired items
	// Note: We scan the map because LRU order != Time order necessarily
	for _, element := range gc.store {
		entry := element.Value.(*lruEntry)
		if now.Sub(entry.config.CachedAt) > gc.cacheTTL {
			expiredElements = append(expiredElements, element)
		}
	}

	for _, element := range expiredElements {
		entry := element.Value.(*lruEntry)
		gc.lruList.Remove(element)
		delete(gc.store, entry.guildID)
		slog.Debug("Cleaned up expired guild config", attr.String("guild_id", entry.guildID))
	}

	if len(expiredElements) > 0 {
		slog.Debug("Cleaned up expired guild configs",
			attr.Int("removed_count", len(expiredElements)),
			attr.Int("remaining_count", len(gc.store)))
	}
}

// IsConfigured returns true if this guild config represents a fully configured guild
func (gc *GuildConfig) IsConfigured() bool {
	if gc.IsPlaceholder {
		return true
	}
	return gc.GuildID != "" &&
		gc.SignupChannelID != "" &&
		gc.EventChannelID != "" &&
		gc.LeaderboardChannelID != "" &&
		gc.RegisteredRoleID != ""
}

// MarkRequestPending marks a guild as having a pending config request
func (gc *GuildConfigCache) MarkRequestPending(guildID string) bool {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	// Check if exists
	element, exists := gc.store[guildID]
	if !exists {
		// Create a placeholder with pending request
		now := time.Now()
		pendingConfig := &GuildConfig{
			GuildID:          guildID,
			CachedAt:         now,
			RefreshedAt:      now,
			IsPlaceholder:    true,
			IsRequestPending: true,
		}

		entry := &lruEntry{guildID: guildID, config: pendingConfig}
		// Check capacity before adding placeholder
		if gc.lruList.Len() >= gc.maxSize {
			gc.evictOldestEntry()
		}

		newElem := gc.lruList.PushFront(entry)
		gc.store[guildID] = newElem
		gc.metrics.RecordPendingRequest()
		return true
	}

	// Update existing
	entry := element.Value.(*lruEntry)
	if entry.config.IsRequestPending {
		return false // Already pending
	}

	entry.config.IsRequestPending = true
	gc.metrics.RecordPendingRequest()
	return true
}

// ClearRequestPending clears the pending request flag for a guild
func (gc *GuildConfigCache) ClearRequestPending(guildID string) {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	element, exists := gc.store[guildID]
	if !exists {
		return
	}

	entry := element.Value.(*lruEntry)
	if entry.config.IsRequestPending {
		entry.config.IsRequestPending = false
		gc.metrics.RecordPendingCleared()
	}
}

// triggerAsyncRefresh handles the locking needed to call the callback safely
func (gc *GuildConfigCache) triggerAsyncRefresh(guildID string) {
	// Attempt to mark as pending. This handles the concurrency check.
	// If another routine beat us to it, MarkRequestPending returns false.
	if gc.MarkRequestPending(guildID) {

		// Retrieve callback safely
		gc.mu.RLock()
		callback := gc.refreshCallback
		gc.mu.RUnlock()

		if callback != nil {
			callback(guildID)
		} else {
			slog.Info("Triggering async config refresh (no callback set)", attr.String("guild_id", guildID))
		}
	}
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

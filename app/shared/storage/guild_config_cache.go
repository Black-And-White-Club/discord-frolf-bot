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
	"golang.org/x/sync/singleflight"
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

func (m *DefaultCacheMetrics) RecordHit()            { atomic.AddInt64(&m.hits, 1) }
func (m *DefaultCacheMetrics) RecordMiss()           { atomic.AddInt64(&m.misses, 1) }
func (m *DefaultCacheMetrics) RecordEviction()       { atomic.AddInt64(&m.evictions, 1) }
func (m *DefaultCacheMetrics) RecordPendingRequest() { atomic.AddInt64(&m.pendingRequests, 1) }
func (m *DefaultCacheMetrics) RecordPendingCleared() { atomic.AddInt64(&m.pendingCleared, 1) }

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
	RefreshedAt          time.Time         `json:"refreshed_at"`
	IsPlaceholder        bool              `json:"is_placeholder"`
	IsRequestPending     bool              `json:"is_request_pending"`
}

type GuildConfigCacheInterface interface {
	Get(guildID string) (*GuildConfig, bool)
	Set(guildID string, config *GuildConfig) error
	Delete(guildID string)
	Size() int
	Clear()
	MarkRequestPending(guildID string) bool
	ClearRequestPending(guildID string)
	GetMetrics() CacheStats
	SetRefreshCallback(callback func(guildID string))
}

type GuildConfigCache struct {
	store           map[string]*list.Element
	lruList         *list.List
	mu              sync.RWMutex
	sfGroup         singleflight.Group // Prevents redundant concurrent refreshes
	maxSize         int
	cacheTTL        time.Duration
	refreshTTL      time.Duration
	metrics         CacheMetrics
	refreshCallback func(guildID string)
}

type lruEntry struct {
	guildID string
	config  *GuildConfig
}

func NewGuildConfigCache(ctx context.Context, maxSize int, cacheTTL, refreshTTL time.Duration) GuildConfigCacheInterface {
	cache := &GuildConfigCache{
		store:      make(map[string]*list.Element),
		lruList:    list.New(),
		maxSize:    maxSize,
		cacheTTL:   cacheTTL,
		refreshTTL: refreshTTL,
		metrics:    NewCacheMetrics(),
	}

	go cache.cleanupExpiredEntries(ctx)

	return cache
}

func (gc *GuildConfigCache) Get(guildID string) (*GuildConfig, bool) {
	gc.mu.Lock()
	element, exists := gc.store[guildID]
	if !exists {
		gc.mu.Unlock()
		gc.metrics.RecordMiss()
		return nil, false
	}

	entry := element.Value.(*lruEntry)
	config := entry.config

	// Hard expiry check
	if time.Since(config.CachedAt) > gc.cacheTTL {
		slog.Debug("Guild config cache expired, removing", attr.String("guild_id", guildID))
		gc.lruList.Remove(element)
		delete(gc.store, guildID)
		gc.mu.Unlock()
		gc.metrics.RecordMiss()
		return nil, false
	}

	// Update LRU position
	gc.lruList.MoveToFront(element)
	gc.mu.Unlock()

	// Check if soft refresh is needed
	if time.Since(config.RefreshedAt) > gc.refreshTTL && !config.IsRequestPending {
		// Use singleflight to ensure only one refresh runs for this guildID
		go gc.sfGroup.Do(guildID, func() (interface{}, error) {
			slog.Debug("Triggering deduplicated background refresh", attr.String("guild_id", guildID))
			gc.triggerAsyncRefresh(guildID)
			return nil, nil
		})
	}

	gc.metrics.RecordHit()
	return config, true
}

func (gc *GuildConfigCache) Set(guildID string, config *GuildConfig) error {
	if config == nil {
		return errors.New("config cannot be nil")
	}

	gc.mu.Lock()
	defer gc.mu.Unlock()

	now := time.Now()
	config.CachedAt = now
	config.RefreshedAt = now
	config.IsRequestPending = false // Reset pending flag on successful set

	if element, exists := gc.store[guildID]; exists {
		entry := element.Value.(*lruEntry)
		entry.config = config
		gc.lruList.MoveToFront(element)
		return nil
	}

	if gc.lruList.Len() >= gc.maxSize {
		gc.evictOldestEntry()
	}

	newEntry := &lruEntry{guildID: guildID, config: config}
	element := gc.lruList.PushFront(newEntry)
	gc.store[guildID] = element

	return nil
}

func (gc *GuildConfigCache) Delete(guildID string) {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	if element, exists := gc.store[guildID]; exists {
		gc.lruList.Remove(element)
		delete(gc.store, guildID)
	}
}

func (gc *GuildConfigCache) Size() int {
	gc.mu.RLock()
	defer gc.mu.RUnlock()
	return len(gc.store)
}

func (gc *GuildConfigCache) Clear() {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	gc.store = make(map[string]*list.Element)
	gc.lruList.Init()
}

func (gc *GuildConfigCache) evictOldestEntry() {
	oldest := gc.lruList.Back()
	if oldest == nil {
		return
	}
	entry := oldest.Value.(*lruEntry)
	gc.lruList.Remove(oldest)
	delete(gc.store, entry.guildID)
	gc.metrics.RecordEviction()
}

func (gc *GuildConfigCache) cleanupExpiredEntries(ctx context.Context) {
	ticker := time.NewTicker(gc.cacheTTL / 4)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			gc.performCleanup()
		}
	}
}

func (gc *GuildConfigCache) performCleanup() {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	now := time.Now()
	for guildID, element := range gc.store {
		entry := element.Value.(*lruEntry)
		if now.Sub(entry.config.CachedAt) > gc.cacheTTL {
			gc.lruList.Remove(element)
			delete(gc.store, guildID)
		}
	}
}

func (gc *GuildConfigCache) MarkRequestPending(guildID string) bool {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	element, exists := gc.store[guildID]
	if !exists {
		now := time.Now()
		pendingConfig := &GuildConfig{
			GuildID:          guildID,
			CachedAt:         now,
			RefreshedAt:      now,
			IsPlaceholder:    true,
			IsRequestPending: true,
		}
		if gc.lruList.Len() >= gc.maxSize {
			gc.evictOldestEntry()
		}
		gc.store[guildID] = gc.lruList.PushFront(&lruEntry{guildID: guildID, config: pendingConfig})
		gc.metrics.RecordPendingRequest()
		return true
	}

	entry := element.Value.(*lruEntry)
	if entry.config.IsRequestPending {
		return false
	}
	entry.config.IsRequestPending = true
	gc.metrics.RecordPendingRequest()
	return true
}

func (gc *GuildConfigCache) ClearRequestPending(guildID string) {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	if element, exists := gc.store[guildID]; exists {
		entry := element.Value.(*lruEntry)
		if entry.config.IsRequestPending {
			entry.config.IsRequestPending = false
			gc.metrics.RecordPendingCleared()
		}
	}
}

func (gc *GuildConfigCache) triggerAsyncRefresh(guildID string) {
	if gc.MarkRequestPending(guildID) {
		gc.mu.RLock()
		callback := gc.refreshCallback
		gc.mu.RUnlock()

		if callback != nil {
			callback(guildID)
		}
	}
}

func (gc *GuildConfigCache) GetMetrics() CacheStats {
	return gc.metrics.GetStats()
}

func (gc *GuildConfigCache) SetRefreshCallback(callback func(guildID string)) {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	gc.refreshCallback = callback
}

func (gc *GuildConfig) IsConfigured() bool {
	if gc.IsPlaceholder {
		return true
	}
	return gc.GuildID != "" && gc.SignupChannelID != "" && gc.EventChannelID != "" &&
		gc.LeaderboardChannelID != "" && gc.RegisteredRoleID != ""
}

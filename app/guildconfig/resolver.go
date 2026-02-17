package guildconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// ConfigErrorMetrics tracks guild config resolution errors
type ConfigErrorMetrics struct {
	ErrorsByType  map[string]int64 `json:"errors_by_type"`
	ErrorsByGuild map[string]int64 `json:"errors_by_guild"`
	LastErrorTime time.Time        `json:"last_error_time"`
	TotalErrors   int64            `json:"total_errors"`
}

// configRequest represents an in-flight request for a guild config
type configRequest struct {
	ready  chan struct{}        // Closed when the request completes
	config *storage.GuildConfig // The resulting config (may be nil)
	err    error                // Any error that occurred
}

// Resolver coordinates guild configuration retrieval with local caching and backend events.
type Resolver struct {
	eventBus         eventbus.EventBus
	cache            storage.ISInterface[storage.GuildConfig] // Local cache for high-performance lookups
	inflightRequests sync.Map                                 // map[string]*configRequest
	config           *ResolverConfig
	errorMetrics     *ConfigErrorMetrics
	errorMutex       sync.RWMutex
	retryMutex       sync.Mutex
	lastRetry        map[string]time.Time
}

// NewResolver creates a resolver and validates its configuration.
func NewResolver(ctx context.Context, eventBus eventbus.EventBus, cache storage.ISInterface[storage.GuildConfig], config *ResolverConfig) (*Resolver, error) {
	if config == nil {
		config = DefaultResolverConfig()
	}

	if err := config.Validate(); err != nil {
		slog.ErrorContext(ctx, "Invalid guild config resolver configuration", attr.Error(err))
		return nil, fmt.Errorf("guild config resolver configuration validation failed: %w", err)
	}

	return &Resolver{
		eventBus:     eventBus,
		cache:        cache,
		config:       config,
		errorMetrics: &ConfigErrorMetrics{ErrorsByType: make(map[string]int64), ErrorsByGuild: make(map[string]int64)},
		lastRetry:    make(map[string]time.Time),
	}, nil
}

// GetGuildConfig retrieves guild config using a background context.
func (r *Resolver) GetGuildConfig(guildID string) (*storage.GuildConfig, error) {
	return r.GetGuildConfigWithContext(context.Background(), guildID)
}

// GetGuildConfigWithContext first checks the local cache; if a miss, it coordinates a backend request.
func (r *Resolver) GetGuildConfigWithContext(ctx context.Context, guildID string) (*storage.GuildConfig, error) {
	// 1. Check local cache first (Immediate return on hit)
	cached, err := r.cache.Get(ctx, guildID)
	if err == nil {
		slog.DebugContext(ctx, "Guild config cache hit", attr.String("guild_id", guildID))
		// Create a copy on the heap to return a pointer safely
		result := cached
		return &result, nil
	}
	slog.InfoContext(ctx, "Guild config cache miss, requesting from backend", attr.String("guild_id", guildID))

	select {
	case <-ctx.Done():
		return nil, &ConfigTemporaryError{GuildID: guildID, Reason: "request cancelled", Cause: ctx.Err()}
	default:
	}

	// 2. Coalesce concurrent requests for the same guild ID
	req := &configRequest{ready: make(chan struct{})}
	actual, loaded := r.inflightRequests.LoadOrStore(guildID, req)
	actualReq := actual.(*configRequest)

	if !loaded {
		// Note: coordinateConfigRequest handles its own timeout/cancellation logic
		go r.coordinateConfigRequest(ctx, guildID, actualReq)
	}

	select {
	case <-actualReq.ready:
		if actualReq.err != nil {
			return nil, actualReq.err
		}
		return actualReq.config, nil

	case <-ctx.Done():
		return nil, &ConfigTemporaryError{GuildID: guildID, Reason: "request cancelled", Cause: ctx.Err()}
	}
}

// RequestGuildConfigAsync triggers a config retrieval request without blocking on the response.
func (r *Resolver) RequestGuildConfigAsync(ctx context.Context, guildID string) {
	if guildID == "" {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if _, err := r.cache.Get(ctx, guildID); err == nil {
		return
	}

	req := &configRequest{ready: make(chan struct{})}
	actual, loaded := r.inflightRequests.LoadOrStore(guildID, req)
	actualReq := actual.(*configRequest)
	if loaded {
		return
	}

	go r.coordinateConfigRequest(ctx, guildID, actualReq)
}

// coordinateConfigRequest handles the mechanics of publishing the event and waiting for the async response.
func (r *Resolver) coordinateConfigRequest(ctx context.Context, guildID string, req *configRequest) {
	defer func() {
		if reqInterface, exists := r.inflightRequests.LoadAndDelete(guildID); exists {
			if existingReq, ok := reqInterface.(*configRequest); ok && existingReq == req {
				defer func() { recover() }() // Safe close
				close(req.ready)
			}
		}
	}()

	requestCtx, cancel := context.WithTimeout(ctx, r.config.RequestTimeout)
	defer cancel()

	payload := guildevents.GuildConfigRetrievalRequestedPayloadV1{
		GuildID: sharedtypes.GuildID(guildID),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		req.err = &ConfigTemporaryError{GuildID: guildID, Reason: "serialization failed", Cause: err}
		return
	}

	msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
	msg.SetContext(requestCtx)
	msg.Metadata.Set("type", guildevents.GuildConfigRetrievalRequestedV1)

	if err := r.eventBus.Publish(guildevents.GuildConfigRetrievalRequestedV1, msg); err != nil {
		req.err = &ConfigTemporaryError{GuildID: guildID, Reason: "publish failed", Cause: err}
		return
	}

	// Wait for HandleGuildConfigReceived or Timeout
	// Use independent context for response wait to get full ResponseTimeout
	// (not limited by requestCtx's shorter deadline)
	responseCtx, responseCancel := context.WithTimeout(context.Background(), r.config.ResponseTimeout)
	defer responseCancel()

	select {
	case <-req.ready:
		return
	case <-responseCtx.Done():
		req.err = &ConfigLoadingError{GuildID: guildID}
		slog.WarnContext(responseCtx, "Backend response timeout", attr.String("guild_id", guildID))
		r.scheduleRetry(responseCtx, guildID)
	}
}

// HandleGuildConfigReceived populates the cache and unblocks any waiting requests.
func (r *Resolver) HandleGuildConfigReceived(ctx context.Context, guildID string, config *storage.GuildConfig) {
	slog.InfoContext(ctx, "HandleGuildConfigReceived called",
		attr.String("guild_id", guildID),
		attr.Bool("config_nil", config == nil))

	if config != nil {
		// Populate the local cache for future Get calls
		r.cache.Set(ctx, guildID, *config)
		slog.InfoContext(ctx, "Guild config cached successfully",
			attr.String("guild_id", guildID),
			attr.Bool("is_placeholder", config.IsPlaceholder))
	}

	if reqInterface, exists := r.inflightRequests.LoadAndDelete(guildID); exists {
		if req, ok := reqInterface.(*configRequest); ok {
			req.config = config
			close(req.ready)
		}
	}
}

// HandleGuildConfigRetrievalFailed handles errors and notifies waiting goroutines.
func (r *Resolver) HandleGuildConfigRetrievalFailed(ctx context.Context, guildID string, reason string, isPermanent bool) {
	var resultErr error
	if isPermanent {
		resultErr = &ConfigNotFoundError{GuildID: guildID, Reason: reason}
		r.recordError(ctx, guildID, "permanent")
		r.cache.Set(ctx, guildID, storage.GuildConfig{
			GuildID:       guildID,
			IsPlaceholder: true,
			CachedAt:      time.Now(),
			RefreshedAt:   time.Now(),
		})
	} else {
		resultErr = &ConfigTemporaryError{GuildID: guildID, Reason: reason}
		r.recordError(ctx, guildID, "temporary")
		r.scheduleRetry(ctx, guildID)
	}

	if reqInterface, exists := r.inflightRequests.LoadAndDelete(guildID); exists {
		if req, ok := reqInterface.(*configRequest); ok {
			req.err = resultErr
			close(req.ready)
		}
	}
}

func (r *Resolver) scheduleRetry(ctx context.Context, guildID string) {
	const retryDelay = 5 * time.Second

	if guildID == "" {
		return
	}
	if !r.markRetry(guildID, retryDelay) {
		return
	}
	if ctx == nil || ctx.Err() != nil {
		ctx = context.Background()
	}

	go func() {
		timer := time.NewTimer(retryDelay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}

		r.RequestGuildConfigAsync(context.Background(), guildID)
	}()
}

func (r *Resolver) markRetry(guildID string, retryDelay time.Duration) bool {
	now := time.Now()

	r.retryMutex.Lock()
	defer r.retryMutex.Unlock()

	last, exists := r.lastRetry[guildID]
	if exists && now.Sub(last) < retryDelay {
		return false
	}

	r.lastRetry[guildID] = now
	return true
}

// ClearInflightRequest invalidates both the inflight status and the local cache entry.
func (r *Resolver) ClearInflightRequest(ctx context.Context, guildID string) {
	r.cache.Delete(ctx, guildID)
	if reqInterface, existed := r.inflightRequests.LoadAndDelete(guildID); existed {
		if req, ok := reqInterface.(*configRequest); ok {
			select {
			case <-req.ready:
			default:
				req.err = &ConfigNotFoundError{GuildID: guildID, Reason: "config deleted/cleared"}
				close(req.ready)
			}
		}
	}
}

// IsGuildSetupComplete checks configuration status.
func (r *Resolver) IsGuildSetupComplete(guildID string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	config, err := r.GetGuildConfigWithContext(ctx, guildID)
	if err != nil || config == nil {
		return false
	}
	return config.IsConfigured()
}

func (r *Resolver) recordError(ctx context.Context, guildID string, errorType string) {
	r.errorMutex.Lock()
	defer r.errorMutex.Unlock()
	r.errorMetrics.ErrorsByType[errorType]++
	r.errorMetrics.ErrorsByGuild[guildID]++
	r.errorMetrics.TotalErrors++
	r.errorMetrics.LastErrorTime = time.Now()
}

func (r *Resolver) HandleBackendError(ctx context.Context, guildID string, err error) {
	if err == nil {
		return
	}
	isPermanent, reason := ClassifyBackendError(err, guildID)
	r.HandleGuildConfigRetrievalFailed(ctx, guildID, reason, isPermanent)
}

// GetErrorMetrics returns a defensive copy of current metrics.
func (r *Resolver) GetErrorMetrics() ConfigErrorMetrics {
	r.errorMutex.RLock()
	defer r.errorMutex.RUnlock()

	// Create a copy to avoid race conditions
	errorsByType := make(map[string]int64)
	for k, v := range r.errorMetrics.ErrorsByType {
		errorsByType[k] = v
	}

	errorsByGuild := make(map[string]int64)
	for k, v := range r.errorMetrics.ErrorsByGuild {
		errorsByGuild[k] = v
	}

	return ConfigErrorMetrics{
		ErrorsByType:  errorsByType,
		ErrorsByGuild: errorsByGuild,
		LastErrorTime: r.errorMetrics.LastErrorTime,
		TotalErrors:   r.errorMetrics.TotalErrors,
	}
}

// ResetErrorMetrics clears counters (testing/maintenance helper).
func (r *Resolver) ResetErrorMetrics() {
	r.errorMutex.Lock()
	defer r.errorMutex.Unlock()

	r.errorMetrics.ErrorsByType = make(map[string]int64)
	r.errorMetrics.ErrorsByGuild = make(map[string]int64)
	r.errorMetrics.TotalErrors = 0
	r.errorMetrics.LastErrorTime = time.Time{}
}

// LogConfigEvent emits a structured log entry for config-related events.
func (r *Resolver) LogConfigEvent(ctx context.Context, eventType, guildID string, attrs ...slog.Attr) {
	allAttrs := []slog.Attr{
		attr.String("component", "guildconfig_resolver"),
		attr.String("event_type", eventType),
		attr.String("guild_id", guildID),
	}
	allAttrs = append(allAttrs, attrs...)

	slog.InfoContext(ctx, "Guild config resolver event", convertAttrsToAny(allAttrs)...)
}

// convertAttrsToAny adapts slog.Attr slice to variadic any.
func convertAttrsToAny(attrs []slog.Attr) []any {
	result := make([]any, len(attrs))
	for i, attr := range attrs {
		result[i] = attr
	}
	return result
}

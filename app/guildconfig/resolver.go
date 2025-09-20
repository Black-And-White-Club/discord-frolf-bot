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
// This prevents duplicate requests by coordinating multiple concurrent requests for the same guild
type configRequest struct {
	ready  chan struct{}        // Closed when the request completes
	config *storage.GuildConfig // The resulting config (may be nil)
	err    error                // Any error that occurred
}

// Resolver requests guild configuration from the backend each time (no caching layer).
type Resolver struct {
	eventBus         eventbus.EventBus   // For requesting configs from backend
	inflightRequests sync.Map            // map[string]*configRequest - prevents duplicate requests
	config           *ResolverConfig     // Resolver configuration
	errorMetrics     *ConfigErrorMetrics // Error tracking for observability
	errorMutex       sync.RWMutex        // Protects error metrics
}

// NewResolver creates a resolver validating config (panics on invalid config for fast fail in startup paths).
func NewResolver(ctx context.Context, eventBus eventbus.EventBus, config *ResolverConfig) *Resolver {
	if config == nil {
		config = DefaultResolverConfig()
	}

	// Validate configuration before proceeding
	if err := config.Validate(); err != nil {
		slog.ErrorContext(ctx, "Invalid guild config resolver configuration", attr.Error(err))
		panic(fmt.Errorf("guild config resolver configuration validation failed: %w", err))
	}

	resolver := &Resolver{
		eventBus:     eventBus,
		config:       config,
		errorMetrics: &ConfigErrorMetrics{ErrorsByType: make(map[string]int64), ErrorsByGuild: make(map[string]int64)},
	}

	// No need for cache refresh callbacks since we're not caching

	return resolver
}

// NewResolverWithDefaults creates a resolver using DefaultResolverConfig.
func NewResolverWithDefaults(ctx context.Context, eventBus eventbus.EventBus) *Resolver {
	return NewResolver(ctx, eventBus, nil) // nil will use DefaultResolverConfig()
}

// GetGuildConfig retrieves guild config with a background context.
func (r *Resolver) GetGuildConfig(guildID string) (*storage.GuildConfig, error) {
	return r.GetGuildConfigWithContext(context.Background(), guildID)
}

// GetGuildConfigWithContext always publishes a retrieval request; concurrent calls for the same guild share a single in-flight request.
func (r *Resolver) GetGuildConfigWithContext(ctx context.Context, guildID string) (*storage.GuildConfig, error) {
	slog.InfoContext(ctx, "Requesting guild config from backend",
		attr.String("guild_id", guildID))

	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return nil, &ConfigTemporaryError{
			GuildID: guildID,
			Reason:  "request cancelled",
			Cause:   ctx.Err(),
		}
	default:
	}

	// Create a new request or get existing one atomically to prevent duplicate requests
	req := &configRequest{
		ready: make(chan struct{}),
	}

	// Use LoadOrStore to atomically check for existing request or store new one
	actual, loaded := r.inflightRequests.LoadOrStore(guildID, req)
	actualReq := actual.(*configRequest)

	if !loaded {
		// We're the leader - initiate the backend request
		go r.coordinateConfigRequest(ctx, guildID, actualReq)
		slog.DebugContext(ctx, "Sending guild config request to backend",
			attr.String("guild_id", guildID))
	} else {
		slog.DebugContext(ctx, "Joining existing guild config request",
			attr.String("guild_id", guildID))
	}

	// Wait for the request to complete or context to be cancelled
	select {
	case <-actualReq.ready:
		// Request completed - return the result
		if actualReq.err != nil {
			return nil, actualReq.err
		}
		return actualReq.config, nil

	case <-ctx.Done():
		// Clean up the request since it was cancelled
		r.inflightRequests.Delete(guildID)
		return nil, &ConfigTemporaryError{
			GuildID: guildID,
			Reason:  "request cancelled",
			Cause:   ctx.Err(),
		}
	}
}

// coordinateConfigRequest publishes the retrieval request and waits for completion notification or timeout.
func (r *Resolver) coordinateConfigRequest(ctx context.Context, guildID string, req *configRequest) {
	defer func() {
		// Clean up only if the request hasn't been handled by HandleGuildConfigReceived/Failed
		// We check if the request is still in the map - if not, it was already handled
		if reqInterface, exists := r.inflightRequests.LoadAndDelete(guildID); exists {
			if existingReq, ok := reqInterface.(*configRequest); ok && existingReq == req {
				// Check if channel is already closed by trying to close it safely
				defer func() {
					if recover() != nil {
						// Channel was already closed, which is fine
					}
				}()
				close(req.ready)
			}
		}
	}()

	// Create timeout context for the request
	requestCtx, cancel := context.WithTimeout(ctx, r.config.RequestTimeout)
	defer cancel()

	// Check if context is cancelled before proceeding
	if requestCtx.Err() != nil {
		req.err = &ConfigTemporaryError{
			GuildID: guildID,
			Reason:  "request timeout",
			Cause:   requestCtx.Err(),
		}
		r.recordError(requestCtx, guildID, "timeout")
		slog.ErrorContext(requestCtx, "Request timeout while requesting guild config",
			attr.String("guild_id", guildID),
			attr.Error(requestCtx.Err()))
		return
	}

	// Try cache again (another goroutine might have populated it while we were setting up)
	// NO CACHE - Always send request to backend

	// Create guild configuration request payload using shared events
	payload := guildevents.GuildConfigRetrievalRequestedPayload{
		GuildID: sharedtypes.GuildID(guildID),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		req.err = &ConfigTemporaryError{
			GuildID: guildID,
			Reason:  "failed to serialize request",
			Cause:   err,
		}
		r.recordError(requestCtx, guildID, "serialization")
		slog.ErrorContext(requestCtx, "Failed to serialize guild config request",
			attr.String("guild_id", guildID),
			attr.Error(err))
		return
	}

	// Create message with structured payload
	msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
	msg.SetContext(requestCtx)                                          // Attach context to the message
	msg.Metadata.Set("type", guildevents.GuildConfigRetrievalRequested) // Use proper event topic

	// Publish guild configuration retrieval request event to backend
	publishErr := r.eventBus.Publish(guildevents.GuildConfigRetrievalRequested, msg)
	if publishErr != nil {
		req.err = &ConfigTemporaryError{
			GuildID: guildID,
			Reason:  "failed to publish request to backend",
			Cause:   publishErr,
		}
		r.recordError(requestCtx, guildID, "publish")
		slog.ErrorContext(requestCtx, "Failed to publish guild configuration request from resolver",
			attr.String("guild_id", guildID),
			attr.Error(publishErr))
		return
	}

	slog.InfoContext(requestCtx, "Published coordinated guild configuration request from resolver",
		attr.String("guild_id", guildID))

	// Set up a timeout for the backend response
	responseTimeout := r.config.ResponseTimeout
	responseCtx, responseCancel := context.WithTimeout(requestCtx, responseTimeout)
	defer responseCancel()

	// Wait for either the response or timeout
	// Note: The actual response handling is done in HandleGuildConfigReceived/HandleGuildConfigRetrievalFailed
	// This goroutine will be notified when those methods close the req.ready channel
	select {
	case <-req.ready:
		// Response received successfully, req.config or req.err is set
		return
	case <-responseCtx.Done():
		// Timeout occurred
		break
	}

	// Check cache one more time in case the response arrived just as we timed out
	// NO CACHE - If timeout occurred, just return loading error
	req.err = &ConfigLoadingError{GuildID: guildID}
	slog.InfoContext(responseCtx, "Backend response timeout for guild config - returning loading state",
		attr.String("guild_id", guildID),
		attr.Duration("timeout", responseTimeout))
}

// HandleGuildConfigReceived completes any waiting requests with the provided config.
func (r *Resolver) HandleGuildConfigReceived(ctx context.Context, guildID string, config *storage.GuildConfig) {
	// Debug log: show config contents
	if config != nil {
		slog.InfoContext(ctx, "Guild config received from backend",
			attr.String("guild_id", guildID),
			attr.String("signup_channel_id", config.SignupChannelID),
			attr.String("signup_emoji", config.SignupEmoji),
		)
	} else {
		slog.WarnContext(ctx, "Guild config received is nil",
			attr.String("guild_id", guildID))
	}

	// Notify any coordinated waiters by completing inflight requests
	if reqInterface, exists := r.inflightRequests.LoadAndDelete(guildID); exists {
		if req, ok := reqInterface.(*configRequest); ok {
			req.config = config
			close(req.ready) // Signal that the request is complete
		}
	}

	r.LogConfigEvent(ctx, "config_received", guildID, attr.Bool("has_config", config != nil))
	slog.InfoContext(ctx, "Guild config received and provided to waiter",
		attr.String("guild_id", guildID))
}

// HandleGuildConfigRetrievalFailed completes waiters with an error classified as permanent or temporary.
func (r *Resolver) HandleGuildConfigRetrievalFailed(ctx context.Context, guildID string, reason string, isPermanent bool) {
	var resultErr error

	if isPermanent {
		// For permanent failures (guild doesn't exist, not configured, etc.)
		// Log as warning since this affects user experience but isn't a system error
		resultErr = &ConfigNotFoundError{
			GuildID: guildID,
			Reason:  reason,
		}
		r.recordError(ctx, guildID, "permanent")
		slog.WarnContext(ctx, "Guild config permanently unavailable",
			attr.String("guild_id", guildID),
			attr.String("reason", reason),
			attr.String("error_type", "permanent"))

		// Don't store anything for permanent failures to allow setup to work
		// The next request will trigger a fresh backend check
	} else {
		// For temporary failures (network issues, database timeouts, etc.)
		// Log as info since these are expected to resolve automatically
		resultErr = &ConfigTemporaryError{
			GuildID: guildID,
			Reason:  reason,
		}
		r.recordError(ctx, guildID, "temporary")
		slog.InfoContext(ctx, "Guild config temporarily unavailable, will retry later",
			attr.String("guild_id", guildID),
			attr.String("reason", reason),
			attr.String("error_type", "temporary"))
	}

	// Notify any coordinated waiters by completing inflight requests
	if reqInterface, exists := r.inflightRequests.LoadAndDelete(guildID); exists {
		if req, ok := reqInterface.(*configRequest); ok {
			req.err = resultErr
			close(req.ready) // Signal that the request is complete
		}
	}

	// Note: We don't use a circuit breaker here because Watermill/JetStream already provides
	// sophisticated retry/backoff mechanisms and dead letter queues for handling failures
}

// HandleBackendError maps a raw backend error into permanent/temporary classification then delegates.
func (r *Resolver) HandleBackendError(ctx context.Context, guildID string, err error) {
	if err == nil {
		return
	}

	isPermanent, reason := ClassifyBackendError(err, guildID)
	r.HandleGuildConfigRetrievalFailed(ctx, guildID, reason, isPermanent)
}

// Cache-related methods were removed; resolver always queries backend.

// IsGuildSetupComplete fetches config with a short timeout and evaluates required fields.
func (r *Resolver) IsGuildSetupComplete(guildID string) bool {
	// Create a short-timeout context for quick setup checks
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Try to get guild config from backend
	config, err := r.GetGuildConfigWithContext(ctx, guildID)
	if err != nil {
		// If we can't fetch config or it times out, assume not set up
		slog.DebugContext(ctx, "Guild setup check failed - assuming not set up",
			attr.String("guild_id", guildID),
			attr.Error(err))
		return false
	}

	if config == nil {
		slog.DebugContext(ctx, "Guild has no config - not set up",
			attr.String("guild_id", guildID))
		return false
	}

	// Use the IsConfigured method to check if all required fields are present
	isConfigured := config.IsConfigured()
	slog.DebugContext(ctx, "Guild setup check completed",
		attr.String("guild_id", guildID),
		attr.Bool("is_configured", isConfigured))

	return isConfigured
}

// Observability and metrics

// recordError updates counters for observability.
func (r *Resolver) recordError(ctx context.Context, guildID string, errorType string) {
	r.errorMutex.Lock()
	defer r.errorMutex.Unlock()

	r.errorMetrics.ErrorsByType[errorType]++
	r.errorMetrics.ErrorsByGuild[guildID]++
	r.errorMetrics.TotalErrors++
	r.errorMetrics.LastErrorTime = time.Now()

	// Log error pattern for observability
	slog.InfoContext(ctx, "Guild config resolver error recorded",
		attr.String("guild_id", guildID),
		attr.String("error_type", errorType),
		attr.Int64("total_errors", r.errorMetrics.TotalErrors),
		attr.Int64("guild_errors", r.errorMetrics.ErrorsByGuild[guildID]))
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

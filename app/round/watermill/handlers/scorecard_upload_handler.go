package roundhandlers

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

const (
	maxScorecardPayloadBytes   = 5_242_880 // ~5MB payload guardrail to drop obviously bad inputs
	maxEventsPerGuildPerMinute = 20        // lightweight rate limit to reduce spam from a single guild
)

var (
	// guildRateLimiter is a best-effort, per-process rate limiter.
	//
	// Note: This bot can run multiple instances (e.g., in Kubernetes). In that case the limit
	// is enforced per instance only; a true global limit would require a shared store (e.g., Redis).
	guildRateLimiter     = make(map[string][]time.Time)
	guildRateLimiterLock sync.Mutex
)

// allowGuildEvent returns false if the guild exceeds the per-minute budget.
func allowGuildEvent(guildID sharedtypes.GuildID) bool {
	now := time.Now()
	cutoff := now.Add(-1 * time.Minute)

	guildRateLimiterLock.Lock()
	defer guildRateLimiterLock.Unlock()

	key := string(guildID)
	times := guildRateLimiter[key]
	// drop stale entries
	pruned := times[:0]
	for _, t := range times {
		if t.After(cutoff) {
			pruned = append(pruned, t)
		}
	}
	if len(pruned) >= maxEventsPerGuildPerMinute {
		guildRateLimiter[key] = pruned
		return false
	}
	pruned = append(pruned, now)
	guildRateLimiter[key] = pruned
	return true
}

// HandleScorecardUploaded handles scorecard uploaded events.
func (rh *RoundHandlers) HandleScorecardUploaded(ctx context.Context, payload *roundevents.ScorecardUploadedPayloadV1) ([]handlerwrapper.Result, error) {
	if len(payload.FileData) > maxScorecardPayloadBytes {
		return nil, fmt.Errorf("payload too large: %d bytes", len(payload.FileData))
	}

	// Basic required-field validation to reject obviously bad payloads early.
	if payload.ImportID == "" || payload.GuildID == "" || payload.RoundID.String() == "" {
		return nil, fmt.Errorf("invalid payload: missing required identifiers")
	}

	// Basic extension check (applies when using URL-based upload).
	if payload.UDiscURL != "" {
		allowedExt := map[string]struct{}{".csv": {}, ".xlsx": {}}
		if u, err := url.Parse(payload.UDiscURL); err == nil {
			ext := strings.ToLower(filepath.Ext(u.Path))
			if _, ok := allowedExt[ext]; ext != "" && !ok {
				return nil, fmt.Errorf("unsupported scorecard extension: %s", ext)
			}
		}
	}

	if !allowGuildEvent(payload.GuildID) {
		return nil, fmt.Errorf("rate limit exceeded for guild %s", payload.GuildID)
	}

	// TODO: Implement scorecard parsing and player matching
	// This is handled by the frolf-bot backend service

	return nil, nil
}

// HandleScorecardParseFailed handles scorecard parse failed events.
func (rh *RoundHandlers) HandleScorecardParseFailed(ctx context.Context, payload *roundevents.ScorecardParseFailedPayloadV1) ([]handlerwrapper.Result, error) {
	// Notify user in Discord about parsing failure
	if payload.ChannelID != "" {
		err := rh.service.GetScorecardUploadManager().SendUploadError(ctx, payload.ChannelID, payload.Error)
		if err != nil {
			return nil, fmt.Errorf("failed to notify user of parsing failure: %w", err)
		}
	}

	return nil, nil
}

// HandleImportFailed handles import failed events.
func (rh *RoundHandlers) HandleImportFailed(ctx context.Context, payload *roundevents.ImportFailedPayloadV1) ([]handlerwrapper.Result, error) {
	// Notify user in Discord about import failure
	if payload.ChannelID != "" {
		err := rh.service.GetScorecardUploadManager().SendUploadError(ctx, payload.ChannelID, payload.Error)
		if err != nil {
			return nil, fmt.Errorf("failed to notify user of import failure: %w", err)
		}
	}

	return nil, nil
}

// HandleScorecardURLRequested handles scorecard URL requested events.
func (rh *RoundHandlers) HandleScorecardURLRequested(ctx context.Context, payload *roundevents.ScorecardURLRequestedPayloadV1) ([]handlerwrapper.Result, error) {
	// TODO: Respond with scorecard URL in Discord
	return nil, nil
}

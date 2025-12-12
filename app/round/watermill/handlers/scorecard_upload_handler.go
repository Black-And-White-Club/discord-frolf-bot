package roundhandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
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

// ScorecardUploadedEvent processes scorecard upload events from the backend.
func (rh *RoundHandlers) ScorecardUploadedEvent(msg *message.Message) error {
	ctx, span := rh.Tracer.Start(context.Background(), "scorecard_uploaded_event")
	defer span.End()

	if len(msg.Payload) > maxScorecardPayloadBytes {
		err := fmt.Errorf("payload too large: %d bytes", len(msg.Payload))
		rh.Logger.WarnContext(ctx, "Dropping scorecard upload: payload too large",
			attr.Int("payload_bytes", len(msg.Payload)),
		)
		span.RecordError(err)
		return err
	}

	var payload roundevents.ScorecardUploadedPayload
	err := json.Unmarshal(msg.Payload, &payload)
	if err != nil {
		rh.Logger.ErrorContext(ctx, "Failed to unmarshal scorecard uploaded event",
			attr.Error(err),
			attr.String("message_id", msg.UUID),
		)
		span.RecordError(err)
		return err
	}

	rh.Logger.InfoContext(ctx, "Processing scorecard uploaded event",
		attr.String("import_id", payload.ImportID),
		attr.String("guild_id", string(payload.GuildID)),
		attr.String("round_id", payload.RoundID.String()),
		attr.String("udisc_url", payload.UDiscURL),
	)

	// Basic required-field validation to reject obviously bad payloads early.
	if payload.ImportID == "" || payload.GuildID == "" || payload.RoundID.String() == "" {
		err := fmt.Errorf("invalid payload: missing required identifiers")
		rh.Logger.WarnContext(ctx, "Dropping scorecard upload: missing identifiers",
			attr.String("import_id", payload.ImportID),
			attr.String("guild_id", string(payload.GuildID)),
			attr.String("round_id", payload.RoundID.String()),
		)
		span.RecordError(err)
		return err
	}

	// Basic extension check (applies when using URL-based upload).
	if payload.UDiscURL != "" {
		allowedExt := map[string]struct{}{".csv": {}, ".xlsx": {}}
		if u, err := url.Parse(payload.UDiscURL); err == nil {
			ext := strings.ToLower(filepath.Ext(u.Path))
			if _, ok := allowedExt[ext]; ext != "" && !ok {
				err := fmt.Errorf("unsupported scorecard extension: %s", ext)
				rh.Logger.WarnContext(ctx, "Dropping scorecard upload: unsupported extension",
					attr.String("guild_id", string(payload.GuildID)),
					attr.String("extension", ext),
				)
				span.RecordError(err)
				return err
			}
		}
	}

	if !allowGuildEvent(payload.GuildID) {
		err := fmt.Errorf("rate limit exceeded for guild %s", payload.GuildID)
		rh.Logger.WarnContext(ctx, "Dropping scorecard upload: guild rate limit",
			attr.String("guild_id", string(payload.GuildID)),
		)
		span.RecordError(err)
		return err
	}

	// TODO: Implement scorecard parsing and player matching
	// This is handled by the frolf-bot backend service

	msg.Ack()
	return nil
}

// ScorecardParseFailedEvent processes scorecard parse failed events.
func (rh *RoundHandlers) ScorecardParseFailedEvent(msg *message.Message) error {
	ctx, span := rh.Tracer.Start(context.Background(), "scorecard_parse_failed_event")
	defer span.End()

	var payload roundevents.ScorecardParseFailedPayload
	err := json.Unmarshal(msg.Payload, &payload)
	if err != nil {
		rh.Logger.ErrorContext(ctx, "Failed to unmarshal scorecard parse failed event",
			attr.Error(err),
			attr.String("message_id", msg.UUID),
		)
		span.RecordError(err)
		return err
	}

	rh.Logger.ErrorContext(ctx, "Scorecard parsing failed",
		attr.String("import_id", payload.ImportID),
		attr.String("guild_id", string(payload.GuildID)),
		attr.String("error", payload.Error),
	)

	// Notify user in Discord about parsing failure
	if payload.ChannelID != "" {
		err := rh.RoundDiscord.GetScorecardUploadManager().SendUploadError(ctx, payload.ChannelID, payload.Error)
		if err != nil {
			rh.Logger.ErrorContext(ctx, "Failed to notify user of parsing failure", attr.Error(err))
		}
	}

	msg.Ack()
	return nil
}

// ImportFailedEvent processes import failed events.
func (rh *RoundHandlers) ImportFailedEvent(msg *message.Message) error {
	ctx, span := rh.Tracer.Start(context.Background(), "import_failed_event")
	defer span.End()

	var payload roundevents.ImportFailedPayload
	err := json.Unmarshal(msg.Payload, &payload)
	if err != nil {
		rh.Logger.ErrorContext(ctx, "Failed to unmarshal import failed event",
			attr.Error(err),
			attr.String("message_id", msg.UUID),
		)
		span.RecordError(err)
		return err
	}

	rh.Logger.ErrorContext(ctx, "Scorecard import failed",
		attr.String("import_id", payload.ImportID),
		attr.String("guild_id", string(payload.GuildID)),
		attr.String("error", payload.Error),
	)

	// Notify user in Discord about import failure
	if payload.ChannelID != "" {
		err := rh.RoundDiscord.GetScorecardUploadManager().SendUploadError(ctx, payload.ChannelID, payload.Error)
		if err != nil {
			rh.Logger.ErrorContext(ctx, "Failed to notify user of import failure", attr.Error(err))
		}
	}

	msg.Ack()
	return nil
}

// ScorecardURLRequestedEvent processes requests for scorecard URLs.
func (rh *RoundHandlers) ScorecardURLRequestedEvent(msg *message.Message) error {
	ctx, span := rh.Tracer.Start(context.Background(), "scorecard_url_requested_event")
	defer span.End()

	var payload roundevents.ScorecardURLRequestedPayload
	err := json.Unmarshal(msg.Payload, &payload)
	if err != nil {
		rh.Logger.ErrorContext(ctx, "Failed to unmarshal scorecard URL requested event",
			attr.Error(err),
			attr.String("message_id", msg.UUID),
		)
		span.RecordError(err)
		return err
	}

	rh.Logger.InfoContext(ctx, "Scorecard URL requested",
		attr.String("round_id", payload.RoundID.String()),
		attr.String("guild_id", string(payload.GuildID)),
	)

	// TODO: Respond with scorecard URL in Discord

	msg.Ack()
	return nil
}

// HandleScorecardUploaded handles scorecard uploaded events.
func (rh *RoundHandlers) HandleScorecardUploaded(msg *message.Message) ([]*message.Message, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := rh.ScorecardUploadedEvent(msg)
	if err != nil {
		rh.Logger.ErrorContext(ctx, "Failed to handle scorecard uploaded event", attr.Error(err))
		return nil, err
	}

	return nil, nil
}

// HandleScorecardParseFailed handles scorecard parse failed events.
func (rh *RoundHandlers) HandleScorecardParseFailed(msg *message.Message) ([]*message.Message, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := rh.ScorecardParseFailedEvent(msg)
	if err != nil {
		rh.Logger.ErrorContext(ctx, "Failed to handle scorecard parse failed event", attr.Error(err))
		return nil, err
	}

	return nil, nil
}

// HandleImportFailed handles import failed events.
func (rh *RoundHandlers) HandleImportFailed(msg *message.Message) ([]*message.Message, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := rh.ImportFailedEvent(msg)
	if err != nil {
		rh.Logger.ErrorContext(ctx, "Failed to handle import failed event", attr.Error(err))
		return nil, err
	}

	return nil, nil
}

// HandleScorecardURLRequested handles scorecard URL requested events.
func (rh *RoundHandlers) HandleScorecardURLRequested(msg *message.Message) ([]*message.Message, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := rh.ScorecardURLRequestedEvent(msg)
	if err != nil {
		rh.Logger.ErrorContext(ctx, "Failed to handle scorecard URL requested event", attr.Error(err))
		return nil, err
	}

	return nil, nil
}

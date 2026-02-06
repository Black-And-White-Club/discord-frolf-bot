package gateway

import (
	"log/slog"
	"regexp"
	"strings"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	rounddiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

// roundIDPattern matches the RoundID footer embedded in native event descriptions.
// Format: \n---\nRoundID: <uuid>
var roundIDPattern = regexp.MustCompile(`(?m)^RoundID:\s*([0-9a-fA-F-]{36})$`)

// ReconcileNativeEventMap populates the NativeEventMap from active Guild Scheduled Events.
// This should be called on bot startup (after the Discord session is ready) to ensure
// the RSVP gateway listeners can resolve events immediately after restart.
func ReconcileNativeEventMap(
	session discord.Session,
	nativeEventMap rounddiscord.NativeEventMap,
	guilds []*discordgo.Guild,
	logger *slog.Logger,
) {
	if nativeEventMap == nil {
		logger.Warn("NativeEventMap is nil, skipping reconciliation")
		return
	}

	reconciled := 0
	for _, g := range guilds {
		if g == nil || g.ID == "" {
			continue
		}

		events, err := session.GuildScheduledEvents(g.ID, false)
		if err != nil {
			logger.Warn("Failed to fetch scheduled events for guild during reconciliation",
				attr.String("guild_id", g.ID),
				attr.Error(err))
			continue
		}

		for _, event := range events {
			if event.Status == discordgo.GuildScheduledEventStatusCompleted ||
				event.Status == discordgo.GuildScheduledEventStatusCanceled {
				continue
			}

			roundID, ok := parseRoundIDFromDescription(event.Description)
			if !ok {
				continue
			}

			nativeEventMap.Store(event.ID, roundID, sharedtypes.GuildID(g.ID))
			reconciled++

			logger.Debug("Reconciled native event mapping",
				attr.String("guild_id", g.ID),
				attr.String("discord_event_id", event.ID),
				attr.String("round_id", roundID.String()))
		}
	}

	logger.Info("Native event map reconciliation complete",
		attr.Int("reconciled_count", reconciled),
		attr.Int("guild_count", len(guilds)))
}

// CleanupOrphanedNativeEvents cancels Discord Guild Scheduled Events that have a
// RoundID in their description but are not tracked in the NativeEventMap.
// This handles the edge case where a round is deleted while native event creation
// is still in-flight, leaving an orphaned Discord event.
//
// This is best-effort: failures to cancel individual events are logged but do not
// stop processing. Orphaned events without a parseable RoundID are ignored (they
// were not created by this bot, or will expire naturally via ScheduledEndTime).
func CleanupOrphanedNativeEvents(
	session discord.Session,
	nativeEventMap rounddiscord.NativeEventMap,
	guilds []*discordgo.Guild,
	logger *slog.Logger,
) {
	if nativeEventMap == nil {
		return
	}

	canceled := 0
	for _, g := range guilds {
		if g == nil || g.ID == "" {
			continue
		}

		events, err := session.GuildScheduledEvents(g.ID, false)
		if err != nil {
			logger.Warn("Failed to fetch scheduled events for orphan cleanup",
				attr.String("guild_id", g.ID),
				attr.Error(err))
			continue
		}

		for _, event := range events {
			if event.Status == discordgo.GuildScheduledEventStatusCompleted ||
				event.Status == discordgo.GuildScheduledEventStatusCanceled {
				continue
			}

			roundID, ok := parseRoundIDFromDescription(event.Description)
			if !ok {
				// Not our event, or no RoundID footer — skip
				continue
			}

			// Check if this event is tracked in the NativeEventMap
			_, mapOK := nativeEventMap.LookupByRoundID(roundID)
			if mapOK {
				continue // Known event, not orphaned
			}

			// Orphaned: the round no longer exists but the Discord event does
			logger.Warn("Canceling orphaned native event",
				attr.String("guild_id", g.ID),
				attr.String("discord_event_id", event.ID),
				attr.String("round_id", roundID.String()))

			if err := session.GuildScheduledEventDelete(g.ID, event.ID); err != nil {
				logger.Warn("Failed to cancel orphaned native event",
					attr.String("guild_id", g.ID),
					attr.String("discord_event_id", event.ID),
					attr.Error(err))
				continue
			}
			canceled++
		}
	}

	if canceled > 0 {
		logger.Info("Orphaned native event cleanup complete",
			attr.Int("canceled_count", canceled))
	}
}

// parseRoundIDFromDescription extracts the RoundID from a native event description.
// Expected format: "...description...\n---\nRoundID: <uuid>"
func parseRoundIDFromDescription(description string) (sharedtypes.RoundID, bool) {
	if !strings.Contains(description, "RoundID:") {
		return sharedtypes.RoundID{}, false
	}

	matches := roundIDPattern.FindStringSubmatch(description)
	if len(matches) < 2 {
		return sharedtypes.RoundID{}, false
	}

	parsed, err := uuid.Parse(matches[1])
	if err != nil {
		return sharedtypes.RoundID{}, false
	}

	return sharedtypes.RoundID(parsed), true
}

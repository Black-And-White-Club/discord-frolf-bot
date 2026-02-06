package gateway

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	rounddiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/bwmarrin/discordgo"
)

// ScheduledEventRSVPListener translates Discord Guild Scheduled Event
// gateway events (UserAdd/UserRemove) into domain events on the event bus.
type ScheduledEventRSVPListener struct {
	nativeEventMap rounddiscord.NativeEventMap
	eventBus       eventbus.EventBus
	helper         utils.Helpers
	logger         *slog.Logger
}

// NewScheduledEventRSVPListener creates a new RSVP gateway listener.
func NewScheduledEventRSVPListener(
	nativeEventMap rounddiscord.NativeEventMap,
	eventBus eventbus.EventBus,
	helper utils.Helpers,
	logger *slog.Logger,
) *ScheduledEventRSVPListener {
	return &ScheduledEventRSVPListener{
		nativeEventMap: nativeEventMap,
		eventBus:       eventBus,
		helper:         helper,
		logger:         logger,
	}
}

// RegisterGatewayHandlers registers Discord gateway handlers for
// GuildScheduledEventUserAdd and GuildScheduledEventUserRemove.
func (l *ScheduledEventRSVPListener) RegisterGatewayHandlers(session discord.Session) {
	session.AddHandler(func(s *discordgo.Session, event *discordgo.GuildScheduledEventUserAdd) {
		l.handleUserAdd(event)
	})

	session.AddHandler(func(s *discordgo.Session, event *discordgo.GuildScheduledEventUserRemove) {
		l.handleUserRemove(event)
	})
}

// handleUserAdd processes a GuildScheduledEventUserAdd gateway event.
// It resolves the DiscordEventID to a RoundID and publishes a join request.
func (l *ScheduledEventRSVPListener) handleUserAdd(event *discordgo.GuildScheduledEventUserAdd) {
	guildID := sharedtypes.GuildID(event.GuildID)
	userID := sharedtypes.DiscordID(event.UserID)
	discordEventID := event.GuildScheduledEventID

	roundID, ok := l.resolveRoundID(guildID, discordEventID)
	if !ok {
		return
	}

	zeroTag := sharedtypes.TagNumber(0)
	payload := roundevents.ParticipantJoinRequestPayloadV1{
		GuildID:   guildID,
		RoundID:   roundID,
		UserID:    userID,
		Response:  roundtypes.ResponseAccept,
		TagNumber: &zeroTag,
	}

	msg, err := l.helper.CreateNewMessage(payload, roundevents.RoundParticipantJoinRequestedV1)
	if err != nil {
		l.logger.Error("Failed to create join request message from native event RSVP",
			attr.String("guild_id", string(guildID)),
			attr.String("user_id", string(userID)),
			attr.String("discord_event_id", discordEventID),
			attr.Error(err))
		return
	}

	if err := l.eventBus.Publish(roundevents.RoundParticipantJoinRequestedV1, msg); err != nil {
		l.logger.Error("Failed to publish join request from native event RSVP",
			attr.String("guild_id", string(guildID)),
			attr.String("user_id", string(userID)),
			attr.String("round_id", roundID.String()),
			attr.Error(err))
		return
	}

	l.logger.Info("Published join request from native event RSVP",
		attr.String("guild_id", string(guildID)),
		attr.String("user_id", string(userID)),
		attr.String("round_id", roundID.String()),
		attr.String("discord_event_id", discordEventID))
}

// handleUserRemove processes a GuildScheduledEventUserRemove gateway event.
// It resolves the DiscordEventID to a RoundID and publishes a removal request.
func (l *ScheduledEventRSVPListener) handleUserRemove(event *discordgo.GuildScheduledEventUserRemove) {
	guildID := sharedtypes.GuildID(event.GuildID)
	userID := sharedtypes.DiscordID(event.UserID)
	discordEventID := event.GuildScheduledEventID

	roundID, ok := l.resolveRoundID(guildID, discordEventID)
	if !ok {
		return
	}

	payload := roundevents.ParticipantRemovalRequestPayloadV1{
		GuildID: guildID,
		RoundID: roundID,
		UserID:  userID,
	}

	msg, err := l.helper.CreateNewMessage(payload, roundevents.RoundParticipantRemovalRequestedV1)
	if err != nil {
		l.logger.Error("Failed to create removal request message from native event RSVP",
			attr.String("guild_id", string(guildID)),
			attr.String("user_id", string(userID)),
			attr.String("discord_event_id", discordEventID),
			attr.Error(err))
		return
	}

	if err := l.eventBus.Publish(roundevents.RoundParticipantRemovalRequestedV1, msg); err != nil {
		l.logger.Error("Failed to publish removal request from native event RSVP",
			attr.String("guild_id", string(guildID)),
			attr.String("user_id", string(userID)),
			attr.String("round_id", roundID.String()),
			attr.Error(err))
		return
	}

	l.logger.Info("Published removal request from native event RSVP",
		attr.String("guild_id", string(guildID)),
		attr.String("user_id", string(userID)),
		attr.String("round_id", roundID.String()),
		attr.String("discord_event_id", discordEventID))
}

// resolveRoundID resolves a DiscordEventID to a RoundID.
// First checks the in-memory NativeEventMap, then falls back to a NATS
// request-reply via the event bus (for post-restart scenarios).
func (l *ScheduledEventRSVPListener) resolveRoundID(guildID sharedtypes.GuildID, discordEventID string) (sharedtypes.RoundID, bool) {
	// Fast path: check the in-memory map
	roundID, _, ok := l.nativeEventMap.LookupByDiscordEventID(discordEventID)
	if ok {
		return roundID, true
	}

	// Slow path: request-reply via event bus (post-restart fallback)
	l.logger.Info("NativeEventMap miss, falling back to NATS lookup",
		attr.String("guild_id", string(guildID)),
		attr.String("discord_event_id", discordEventID))

	lookupPayload := roundevents.NativeEventLookupRequestPayloadV1{
		GuildID:        guildID,
		DiscordEventID: discordEventID,
	}

	msg, err := l.helper.CreateNewMessage(lookupPayload, roundevents.NativeEventLookupRequestV1)
	if err != nil {
		l.logger.Warn("Failed to create native event lookup request message",
			attr.String("guild_id", string(guildID)),
			attr.String("discord_event_id", discordEventID),
			attr.Error(err))
		return sharedtypes.RoundID{}, false
	}

	if err := l.eventBus.Publish(roundevents.NativeEventLookupRequestV1, msg); err != nil {
		l.logger.Warn("Failed to publish native event lookup request",
			attr.String("guild_id", string(guildID)),
			attr.String("discord_event_id", discordEventID),
			attr.Error(err))
		return sharedtypes.RoundID{}, false
	}

	// Subscribe to the result topic and wait with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resultCh, err := l.eventBus.Subscribe(ctx, roundevents.NativeEventLookupResultV1)
	if err != nil {
		l.logger.Warn("Failed to subscribe to native event lookup result",
			attr.String("guild_id", string(guildID)),
			attr.String("discord_event_id", discordEventID),
			attr.Error(err))
		return sharedtypes.RoundID{}, false
	}

	for {
		select {
		case <-ctx.Done():
			l.logger.Warn("Timeout waiting for native event lookup result",
				attr.String("guild_id", string(guildID)),
				attr.String("discord_event_id", discordEventID))
			return sharedtypes.RoundID{}, false
		case resultMsg, ok := <-resultCh:
			if !ok {
				l.logger.Warn("Native event lookup result channel closed",
					attr.String("guild_id", string(guildID)),
					attr.String("discord_event_id", discordEventID))
				return sharedtypes.RoundID{}, false
			}

			var result roundevents.NativeEventLookupResultPayloadV1
			if err := json.Unmarshal(resultMsg.Payload, &result); err != nil {
				resultMsg.Ack()
				l.logger.Warn("Failed to unmarshal native event lookup result",
					attr.String("guild_id", string(guildID)),
					attr.String("discord_event_id", discordEventID),
					attr.Error(err))
				continue
			}

			resultMsg.Ack()

			// Check if this result is for our request
			if result.DiscordEventID != discordEventID {
				continue
			}

			if !result.Found {
				l.logger.Warn("Native event lookup returned not found",
					attr.String("guild_id", string(guildID)),
					attr.String("discord_event_id", discordEventID))
				return sharedtypes.RoundID{}, false
			}

			// Populate the NativeEventMap for future lookups
			l.nativeEventMap.Store(discordEventID, result.RoundID, guildID)

			l.logger.Info("Resolved native event via NATS lookup",
				attr.String("guild_id", string(guildID)),
				attr.String("discord_event_id", discordEventID),
				attr.String("round_id", result.RoundID.String()))

			return result.RoundID, true
		}
	}
}

package gateway

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
	"time"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig"
	rounddiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
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
// It also handles early event end/cancel to finalize the scorecard embed.
type ScheduledEventRSVPListener struct {
	nativeEventMap        rounddiscord.NativeEventMap
	messageMap            rounddiscord.MessageMap
	pendingNativeEventMap rounddiscord.PendingNativeEventMap
	session               discord.Session
	config                *config.Config
	guildConfig           guildconfig.GuildConfigResolver
	eventBus              eventbus.EventBus
	helper                utils.Helpers
	logger                *slog.Logger
	startedEvents         sync.Map
}

// NewScheduledEventRSVPListener creates a new RSVP gateway listener.
func NewScheduledEventRSVPListener(
	nativeEventMap rounddiscord.NativeEventMap,
	messageMap rounddiscord.MessageMap,
	pendingNativeEventMap rounddiscord.PendingNativeEventMap,
	session discord.Session,
	cfg *config.Config,
	guildConfig guildconfig.GuildConfigResolver,
	eventBus eventbus.EventBus,
	helper utils.Helpers,
	logger *slog.Logger,
) *ScheduledEventRSVPListener {
	return &ScheduledEventRSVPListener{
		nativeEventMap:        nativeEventMap,
		messageMap:            messageMap,
		pendingNativeEventMap: pendingNativeEventMap,
		session:               session,
		config:                cfg,
		guildConfig:           guildConfig,
		eventBus:              eventBus,
		helper:                helper,
		logger:                logger,
	}
}

// resolveEventChannel resolves the event channel ID for a guild, with a logged fallback to global config.
func (l *ScheduledEventRSVPListener) resolveEventChannel(ctx context.Context, guildID string) string {
	if l.guildConfig != nil {
		gc, err := l.guildConfig.GetGuildConfigWithContext(ctx, guildID)
		if err == nil && gc != nil && gc.EventChannelID != "" {
			return gc.EventChannelID
		}
		l.logger.WarnContext(ctx, "Failed to resolve guild-specific event channel, falling back to global config",
			attr.String("guild_id", guildID),
			attr.Error(err))
	}
	return l.config.GetEventChannelID()
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

	session.AddHandler(func(s *discordgo.Session, event *discordgo.GuildScheduledEventUpdate) {
		l.handleEventUpdate(event.GuildScheduledEvent)
	})

	session.AddHandler(func(s *discordgo.Session, event *discordgo.GuildScheduledEventCreate) {
		l.handleEventCreate(event.GuildScheduledEvent)
	})

	session.AddHandler(func(s *discordgo.Session, event *discordgo.GuildScheduledEventDelete) {
		l.handleEventDelete(event.GuildScheduledEvent)
	})
}

// handleEventDelete processes a GuildScheduledEventDelete gateway event.
// It triggers the same cancellation logic as handleEventCanceled.
func (l *ScheduledEventRSVPListener) handleEventDelete(event *discordgo.GuildScheduledEvent) {
	l.logger.Info("Received GuildScheduledEventDelete", attr.String("event_id", event.ID))
	l.handleEventCanceled(event)
}

// handleEventCreate processes a GuildScheduledEventCreate gateway event.
// When a user creates a Discord scheduled event from the UI, this handler
// creates a backend round and posts the scorecard embed without creating
// a duplicate Discord event.
func (l *ScheduledEventRSVPListener) handleEventCreate(event *discordgo.GuildScheduledEvent) {
	// Skip bot-created events (they already have a RoundID in the description)
	if strings.Contains(event.Description, "RoundID:") {
		return
	}

	pendingKey := event.GuildID + "|" + event.Name
	l.pendingNativeEventMap.Store(pendingKey, event.ID)

	// Extract location, default to "TBD"
	location := "TBD"
	if event.EntityMetadata.Location != "" {
		location = event.EntityMetadata.Location
	}

	// Format start time as UTC string
	startTimeStr := event.ScheduledStartTime.UTC().Format("2006-01-02 15:04")

	// Extract creator ID
	creatorID := ""
	if event.CreatorID != "" {
		creatorID = event.CreatorID
	} else if event.Creator != nil {
		creatorID = event.Creator.ID
	}

	// Resolve guild-specific event channel
	channelID := l.resolveEventChannel(context.Background(), event.GuildID)

	desc := roundtypes.Description(event.Description)
	payload := roundevents.CreateRoundRequestedPayloadV1{
		GuildID:     sharedtypes.GuildID(event.GuildID),
		Title:       roundtypes.Title(event.Name),
		Description: &desc,
		StartTime:   startTimeStr,
		Location:    roundtypes.Location(location),
		UserID:      sharedtypes.DiscordID(creatorID),
		ChannelID:   channelID,
		Timezone:    roundtypes.Timezone("UTC"),
	}

	msg, err := l.helper.CreateNewMessage(payload, roundevents.RoundCreationRequestedV1)
	if err != nil {
		l.pendingNativeEventMap.LoadAndDelete(pendingKey)
		l.logger.Error("Failed to create round creation request from Discord event",
			attr.String("guild_id", event.GuildID),
			attr.String("event_name", event.Name),
			attr.String("discord_event_id", event.ID),
			attr.Error(err))
		return
	}

	if err := l.eventBus.Publish(roundevents.RoundCreationRequestedV1, msg); err != nil {
		l.pendingNativeEventMap.LoadAndDelete(pendingKey)
		l.logger.Error("Failed to publish round creation request from Discord event",
			attr.String("guild_id", event.GuildID),
			attr.String("event_name", event.Name),
			attr.String("discord_event_id", event.ID),
			attr.Error(err))
		return
	}

	l.logger.Info("Published round creation request from user-created Discord event",
		attr.String("guild_id", event.GuildID),
		attr.String("event_name", event.Name),
		attr.String("discord_event_id", event.ID))
}

// handleUserAdd processes a GuildScheduledEventUserAdd gateway event.
// It resolves the DiscordEventID to a RoundID and publishes a join request.
func (l *ScheduledEventRSVPListener) handleUserAdd(event *discordgo.GuildScheduledEventUserAdd) {
	guildID := sharedtypes.GuildID(event.GuildID)
	userID := sharedtypes.DiscordID(event.UserID)
	discordEventID := event.GuildScheduledEventID

	roundID, _, ok := l.resolveRoundID(guildID, discordEventID)
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

	// Set metadata needed by HandleRoundParticipantJoined
	if messageID, found := l.messageMap.Load(roundID); found {
		msg.Metadata.Set("discord_message_id", messageID)
	}

	// Resolve guild-specific channel ID
	resolvedChannelID := l.resolveEventChannel(context.Background(), string(guildID))
	if resolvedChannelID != "" {
		msg.Metadata.Set("channel_id", resolvedChannelID)
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

	roundID, _, ok := l.resolveRoundID(guildID, discordEventID)
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

	// Set metadata needed by HandleRoundParticipantRemoved
	if messageID, found := l.messageMap.Load(roundID); found {
		msg.Metadata.Set("discord_message_id", messageID)
	}

	// Resolve guild-specific channel ID
	resolvedChannelID := l.resolveEventChannel(context.Background(), string(guildID))
	if resolvedChannelID != "" {
		msg.Metadata.Set("channel_id", resolvedChannelID)
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

// handleEventUpdate processes a GuildScheduledEventUpdate gateway event.
// Routes to the appropriate handler based on event status.
func (l *ScheduledEventRSVPListener) handleEventUpdate(event *discordgo.GuildScheduledEvent) {
	l.logger.Info("Received GuildScheduledEventUpdate",
		attr.String("event_id", event.ID),
		attr.Int("status", int(event.Status)))

	switch event.Status {
	case discordgo.GuildScheduledEventStatusCanceled:
		l.handleEventCanceled(event)
	case discordgo.GuildScheduledEventStatusCompleted:
		l.handleEventCompleted(event)
	case discordgo.GuildScheduledEventStatusActive:
		l.handleEventStartedEarly(event)
	default:
		// Scheduled or other status — treat as field update
		l.handleEventFieldUpdate(event)
	}
}

// handleEventCanceled publishes a round delete request when a Discord event is canceled.
func (l *ScheduledEventRSVPListener) handleEventCanceled(event *discordgo.GuildScheduledEvent) {
	l.logger.Info("Handling event cancellation", attr.String("event_id", event.ID))
	l.startedEvents.Delete(event.ID)
	guildID := sharedtypes.GuildID(event.GuildID)

	roundID, storedCreatorID, ok := l.resolveRoundID(guildID, event.ID)
	if !ok {
		l.logger.Warn("Failed to resolve round ID for cancellation", attr.String("event_id", event.ID))
		return
	}

	// Extract creator ID for the delete request.
	// Prefer the stored creator ID (from round creation) to bypass auth issues for bot-created events.
	creatorID := string(storedCreatorID)
	if creatorID == "" {
		creatorID = event.CreatorID
		if creatorID == "" && event.Creator != nil {
			creatorID = event.Creator.ID
		}
	}

	payload := roundevents.RoundDeleteRequestPayloadV1{
		GuildID:              guildID,
		RoundID:              roundID,
		RequestingUserUserID: sharedtypes.DiscordID(creatorID),
	}

	msg, err := l.helper.CreateNewMessage(payload, roundevents.RoundDeleteRequestedV1)
	if err != nil {
		l.logger.Error("Failed to create delete request from canceled Discord event",
			attr.String("guild_id", string(guildID)),
			attr.String("discord_event_id", event.ID),
			attr.String("round_id", roundID.String()),
			attr.Error(err))
		return
	}

	// Set metadata needed by HandleRoundDeleted
	if messageID, found := l.messageMap.Load(roundID); found {
		msg.Metadata.Set("discord_message_id", messageID)
	}

	// Resolve guild-specific channel ID
	resolvedChannelID := l.resolveEventChannel(context.Background(), string(guildID))
	if resolvedChannelID != "" {
		msg.Metadata.Set("channel_id", resolvedChannelID)
	}

	if err := l.eventBus.Publish(roundevents.RoundDeleteRequestedV1, msg); err != nil {
		l.logger.Error("Failed to publish delete request from canceled Discord event",
			attr.String("guild_id", string(guildID)),
			attr.String("discord_event_id", event.ID),
			attr.String("round_id", roundID.String()),
			attr.Error(err))
		return
	}

	l.logger.Info("Published round delete request from canceled Discord event",
		attr.String("guild_id", string(guildID)),
		attr.String("discord_event_id", event.ID),
		attr.String("round_id", roundID.String()))
}

// handleEventCompleted finalizes the scorecard embed when an event is completed (ended).
func (l *ScheduledEventRSVPListener) handleEventCompleted(event *discordgo.GuildScheduledEvent) {
	l.startedEvents.Delete(event.ID)
	guildID := sharedtypes.GuildID(event.GuildID)

	roundID, _, ok := l.resolveRoundID(guildID, event.ID)
	if !ok {
		return
	}

	messageID, ok := l.messageMap.Load(roundID)
	if !ok {
		l.logger.Warn("No messageID found for completed event",
			attr.String("discord_event_id", event.ID),
			attr.String("round_id", roundID.String()))
		return
	}

	// Resolve guild-specific channel ID
	resolvedChannelID := l.resolveEventChannel(context.Background(), string(guildID))

	if resolvedChannelID == "" {
		return
	}

	// Fetch the current embed from Discord
	msg, err := l.session.ChannelMessage(resolvedChannelID, messageID)
	if err != nil || len(msg.Embeds) == 0 {
		return
	}

	embed := msg.Embeds[0]
	embed.Color = 0x808080 // Gray for early end
	embed.Footer = &discordgo.MessageEmbedFooter{Text: "Round ended early."}

	// Edit message: update embed, remove action buttons
	_, err = l.session.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    resolvedChannelID,
		ID:         messageID,
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &[]discordgo.MessageComponent{},
	})
	if err != nil {
		l.logger.Error("Failed to finalize embed for completed event",
			attr.String("discord_event_id", event.ID),
			attr.String("round_id", roundID.String()),
			attr.Error(err))
		return
	}

	l.nativeEventMap.Delete(roundID)

	l.logger.Info("Finalized embed for completed event",
		attr.String("discord_event_id", event.ID),
		attr.String("round_id", roundID.String()))
}

// handleEventStartedEarly publishes a round start request when a Discord event
// is set to Active (started early from the Discord UI).
func (l *ScheduledEventRSVPListener) handleEventStartedEarly(event *discordgo.GuildScheduledEvent) {
	// Prevent infinite loops: if we already processed this start, ignore it.
	if _, loaded := l.startedEvents.LoadOrStore(event.ID, true); loaded {
		return
	}

	guildID := sharedtypes.GuildID(event.GuildID)

	roundID, _, ok := l.resolveRoundID(guildID, event.ID)
	if !ok {
		return
	}

	payload := roundevents.RoundStartRequestedPayloadV1{
		GuildID: guildID,
		RoundID: roundID,
	}

	msg, err := l.helper.CreateNewMessage(payload, roundevents.RoundStartRequestedV1)
	if err != nil {
		l.logger.Error("Failed to create start request from early-started Discord event",
			attr.String("guild_id", string(guildID)),
			attr.String("discord_event_id", event.ID),
			attr.String("round_id", roundID.String()),
			attr.Error(err))
		return
	}

	if err := l.eventBus.Publish(roundevents.RoundStartRequestedV1, msg); err != nil {
		l.logger.Error("Failed to publish start request from early-started Discord event",
			attr.String("guild_id", string(guildID)),
			attr.String("discord_event_id", event.ID),
			attr.String("round_id", roundID.String()),
			attr.Error(err))
		return
	}

	l.logger.Info("Published round start request from early-started Discord event",
		attr.String("guild_id", string(guildID)),
		attr.String("discord_event_id", event.ID),
		attr.String("round_id", roundID.String()))
}

// handleEventFieldUpdate publishes a round update request when a Discord event's
// fields (name, description, time, location) are updated from the Discord UI.
func (l *ScheduledEventRSVPListener) handleEventFieldUpdate(event *discordgo.GuildScheduledEvent) {
	guildID := sharedtypes.GuildID(event.GuildID)

	roundID, _, ok := l.resolveRoundID(guildID, event.ID)
	if !ok {
		return
	}

	messageID, ok := l.messageMap.Load(roundID)
	if !ok {
		l.logger.Warn("No messageID found for updated event",
			attr.String("discord_event_id", event.ID),
			attr.String("round_id", roundID.String()))
		return
	}

	// Resolve guild-specific channel ID
	resolvedChannelID := l.resolveEventChannel(context.Background(), string(guildID))

	if resolvedChannelID == "" {
		return
	}

	// Build update payload with all fields from the event
	title := roundtypes.Title(event.Name)
	location := roundtypes.Location(event.EntityMetadata.Location)
	utcTz := roundtypes.Timezone("UTC")

	var startTimeStr *string
	if !event.ScheduledStartTime.IsZero() {
		s := event.ScheduledStartTime.UTC().Format("2006-01-02 15:04")
		startTimeStr = &s
	}

	// Strip the RoundID footer from description before sending update
	desc := event.Description
	if idx := strings.Index(desc, "\n---\nRoundID:"); idx >= 0 {
		desc = desc[:idx]
	}
	description := roundtypes.Description(desc)

	payload := roundevents.UpdateRoundRequestedPayloadV1{
		GuildID:     guildID,
		RoundID:     roundID,
		UserID:      sharedtypes.DiscordID(event.CreatorID),
		ChannelID:   resolvedChannelID,
		MessageID:   messageID,
		Title:       &title,
		Description: &description,
		StartTime:   startTimeStr,
		Timezone:    &utcTz,
		Location:    &location,
	}

	msg, err := l.helper.CreateNewMessage(payload, roundevents.RoundUpdateRequestedV1)
	if err != nil {
		l.logger.Error("Failed to create update request from Discord event update",
			attr.String("guild_id", string(guildID)),
			attr.String("discord_event_id", event.ID),
			attr.String("round_id", roundID.String()),
			attr.Error(err))
		return
	}

	// Set metadata so downstream handlers can update the embed
	msg.Metadata.Set("channel_id", resolvedChannelID)
	msg.Metadata.Set("message_id", messageID)
	msg.Metadata.Set("discord_message_id", messageID)

	if err := l.eventBus.Publish(roundevents.RoundUpdateRequestedV1, msg); err != nil {
		l.logger.Error("Failed to publish update request from Discord event update",
			attr.String("guild_id", string(guildID)),
			attr.String("discord_event_id", event.ID),
			attr.String("round_id", roundID.String()),
			attr.Error(err))
		return
	}

	l.logger.Info("Published round update request from Discord event update",
		attr.String("guild_id", string(guildID)),
		attr.String("discord_event_id", event.ID),
		attr.String("round_id", roundID.String()))
}

// resolveRoundID resolves a DiscordEventID to a RoundID.
// First checks the in-memory NativeEventMap, then falls back to a NATS
// request-reply via the event bus (for post-restart scenarios).
func (l *ScheduledEventRSVPListener) resolveRoundID(guildID sharedtypes.GuildID, discordEventID string) (sharedtypes.RoundID, sharedtypes.DiscordID, bool) {
	// Fast path: check the in-memory map
	roundID, _, creatorID, ok := l.nativeEventMap.LookupByDiscordEventID(discordEventID)
	if ok {
		l.logger.Info("Resolved round ID from memory map", attr.String("event_id", discordEventID), attr.String("round_id", roundID.String()))
		return roundID, creatorID, true
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
		return sharedtypes.RoundID{}, sharedtypes.DiscordID(""), false
	}

	if err := l.eventBus.Publish(roundevents.NativeEventLookupRequestV1, msg); err != nil {
		l.logger.Warn("Failed to publish native event lookup request",
			attr.String("guild_id", string(guildID)),
			attr.String("discord_event_id", discordEventID),
			attr.Error(err))
		return sharedtypes.RoundID{}, sharedtypes.DiscordID(""), false
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
		return sharedtypes.RoundID{}, sharedtypes.DiscordID(""), false
	}

	for {
		select {
		case <-ctx.Done():
			l.logger.Warn("Timeout waiting for native event lookup result",
				attr.String("guild_id", string(guildID)),
				attr.String("discord_event_id", discordEventID))
			return sharedtypes.RoundID{}, sharedtypes.DiscordID(""), false
		case resultMsg, ok := <-resultCh:
			if !ok {
				l.logger.Warn("Native event lookup result channel closed",
					attr.String("guild_id", string(guildID)),
					attr.String("discord_event_id", discordEventID))
				return sharedtypes.RoundID{}, sharedtypes.DiscordID(""), false
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
				return sharedtypes.RoundID{}, sharedtypes.DiscordID(""), false
			}

			// Populate the NativeEventMap for future lookups
			// NOTE: We don't get CreatorID from the lookup result currently, so we pass empty.
			// This means restart-persistence for bot-created event deletion will still fail,
			// but active-session deletion will work.
			l.nativeEventMap.Store(discordEventID, result.RoundID, guildID, sharedtypes.DiscordID(""))

			l.logger.Info("Resolved native event via NATS lookup",
				attr.String("guild_id", string(guildID)),
				attr.String("discord_event_id", discordEventID),
				attr.String("round_id", result.RoundID.String()))

			return result.RoundID, sharedtypes.DiscordID(""), true
		}
	}
}

package roundhandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	roundevents "github.com/Black-And-White-Club/discord-frolf-bot/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round" // Import roundtypes
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleRoundCreated handles the round.created event and creates the Discord event.
func (h *RoundHandlers) HandleRoundCreated(msg *message.Message) error {
	correlationID, payload, err := utils.UnmarshalPayload[roundtypes.Round](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal RoundCreatedPayload: %w", err)
	}

	h.logger.Info("Received RoundCreated event",
		slog.String("correlation_id", correlationID),
	)

	// Validate Required Fields
	if payload.StartTime == nil {
		h.logger.Error("Missing required StartTime in RoundCreated event")
		return fmt.Errorf("missing required StartTime in RoundCreated event")
	}
	if payload.DiscordGuildID == nil || *payload.DiscordGuildID == "" {
		h.logger.Error("Missing required DiscordGuildID in RoundCreated event")
		return fmt.Errorf("missing required DiscordGuildID in RoundCreated event")
	}
	if payload.DiscordChannelID == nil || *payload.DiscordChannelID == "" {
		h.logger.Error("Missing required DiscordChannelID in RoundCreated event")
		return fmt.Errorf("missing required DiscordChannelID in RoundCreated event")
	}

	// Safe Dereferencing
	desc := ""
	if payload.Description != nil {
		desc = *payload.Description
	}

	loc := ""
	if payload.Location != nil {
		loc = *payload.Location
	}

	startTime := *payload.StartTime

	var endTime *time.Time
	if payload.EndTime != nil {
		endTime = payload.EndTime // Already a pointer in Round
	}

	// Use Context with Timeout
	ctx, cancel := context.WithTimeout(msg.Context(), 5*time.Second)
	defer cancel()

	// Create Guild Scheduled Event
	event, err := h.EmbedService.CreateGuildScheduledEvent(
		ctx,
		*payload.DiscordGuildID, *payload.DiscordChannelID, payload.Title, desc, loc, startTime, endTime,
	)
	if err != nil {
		h.logger.Error("Failed to create Guild Scheduled Event", "error", err)
		return err
	}

	startTimeStr := startTime.Format(time.RFC3339) // Format for JSON
	endTimeStr := ""
	if endTime != nil {
		endTimeStr = endTime.Format(time.RFC3339)
	}

	// Use json.Marshal
	eventData := roundevents.GuildScheduledEventCreatedPayload{
		GuildEventID: event.ID,
		ChannelID:    *payload.DiscordChannelID,
		Title:        payload.Title,
		StartTime:    startTimeStr,
		EndTime:      endTimeStr,
		Location:     payload.Location,
	}

	eventBytes, err := json.Marshal(eventData)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}

	eventMsg := message.NewMessage(watermill.NewUUID(), eventBytes)
	eventMsg.Metadata.Set("correlation_id", correlationID)

	// Publish Event
	if err := h.EventBus.Publish("guild.scheduled.event.created", eventMsg); err != nil {
		h.logger.Error("Failed to publish guild.scheduled.event.created event", "error", err)
		return fmt.Errorf("failed to publish guild.scheduled.event.created event: %w", err)
	}
	h.logger.Info("Published guild.scheduled.event.created event", "guild_event_id", event.ID)
	return nil
}

// HandleRoundEmbedEvent handles the guild.scheduled.event.created event.
func (h *RoundHandlers) HandleRoundEmbedEvent(msg *message.Message) error {
	correlationID, payload, err := utils.UnmarshalPayload[roundevents.GuildScheduledEventCreatedPayload](msg, h.logger)
	if err != nil {
		h.logger.Error("Failed to unmarshal GuildScheduledEventCreated payload", "error", err)
		return fmt.Errorf("failed to unmarshal GuildScheduledEventCreatedPayload: %w", err)
	}

	h.logger.Info("Received GuildScheduledEventCreated event",
		slog.String("correlation_id", correlationID),
	)

	loc := ""
	if payload.Location != nil {
		loc = *payload.Location
	}
	// Create the embed message, now using the service, pass context!
	messageID, err := h.EmbedService.CreateRoundEmbed(msg.Context(), payload.ChannelID, payload.Title, payload.StartTime, loc)
	if err != nil {
		h.logger.Error("Failed to create round embed", "error", err)
		return err
	}
	h.logger.Info("Created round embed", "message_id", messageID)

	return nil
}

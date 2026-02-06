package handlers

import (
	"context"
	"fmt"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	"github.com/bwmarrin/discordgo"
)

// HandleRoundCreatedForNativeEvent creates a native Discord Guild Scheduled Event for a new round.
// This is a separate consumer of RoundCreatedV1, parallel to the embed creation handler.
func (h *RoundHandlers) HandleRoundCreatedForNativeEvent(ctx context.Context, payload *roundevents.RoundCreatedPayloadV1) ([]handlerwrapper.Result, error) {
	guildID := string(payload.GuildID)
	roundID := payload.RoundID
	title := string(payload.Title)
	location := string(payload.Location)
	startTime := payload.StartTime
	description := string(payload.Description)

	if startTime == nil {
		return nil, fmt.Errorf("RoundCreatedPayloadV1 missing StartTime for round %s", roundID.String())
	}

	// Convert sharedtypes.StartTime to time.Time
	startTimeValue := time.Time(*startTime)
	endTimeValue := startTimeValue.Add(3 * time.Hour)

	// Create the Discord Guild Scheduled Event parameters
	eventParams := &discordgo.GuildScheduledEventParams{
		Name:                 title,
		Description:          fmt.Sprintf("%s\n---\nRoundID: %s", description, roundID.String()),
		ScheduledStartTime:   &startTimeValue,
		ScheduledEndTime:     &endTimeValue,
		EntityType:           discordgo.GuildScheduledEventEntityTypeExternal, // Type 3
		EntityMetadata: &discordgo.GuildScheduledEventEntityMetadata{
			Location: location,
		},
		PrivacyLevel: discordgo.GuildScheduledEventPrivacyLevelGuildOnly, // Type 2
	}

	// Create the native event via Discord API
	session := h.service.GetSession()
	nativeEvent, err := session.GuildScheduledEventCreate(guildID, eventParams)
	if err != nil {
		// Publish failure event for observability; do not block the embed creation flow
		return []handlerwrapper.Result{
			{
				Topic: roundevents.NativeEventCreateFailedV1,
				Payload: roundevents.NativeEventCreateFailedPayloadV1{
					GuildID: payload.GuildID,
					RoundID: roundID,
					Error:   err.Error(),
				},
			},
		}, nil
	}

	// Store the mapping in the NativeEventMap for RSVP resolution
	nativeEventMap := h.service.GetNativeEventMap()
	if nativeEventMap != nil {
		nativeEventMap.Store(nativeEvent.ID, roundID, payload.GuildID)
	}

	// Publish success event so backend can store the discord_event_id
	return []handlerwrapper.Result{
		{
			Topic: roundevents.NativeEventCreatedV1,
			Payload: roundevents.NativeEventCreatedPayloadV1{
				GuildID:        payload.GuildID,
				RoundID:        roundID,
				DiscordEventID: nativeEvent.ID,
			},
		},
	}, nil
}

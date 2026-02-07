package handlers

import (
	"context"
	"fmt"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
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
	description := string(payload.Description) + fmt.Sprintf("\n---\nRoundID: %s", roundID.String())

	if startTime == nil {
		return nil, fmt.Errorf("RoundCreatedPayloadV1 missing StartTime for round %s", roundID.String())
	}

	// Check if this round originated from a user-created Discord event
	pendingKey := guildID + "|" + title
	if existingEventID, ok := h.service.GetPendingNativeEventMap().LoadAndDelete(pendingKey); ok {
		// Store mapping in NativeEventMap
		nativeEventMap := h.service.GetNativeEventMap()
		if nativeEventMap != nil {
			nativeEventMap.Store(existingEventID, roundID, payload.GuildID, sharedtypes.DiscordID(payload.UserID))
		}

		// Update Discord event description to include RoundID footer (for reconciliation)
		session := h.service.GetSession()
		if session != nil {
			session.GuildScheduledEventEdit(guildID, existingEventID, &discordgo.GuildScheduledEventParams{
				Description: description,
			})
		}

		// Build results: NativeEventCreatedV1 + join request for the creator
		results := []handlerwrapper.Result{
			{
				Topic: roundevents.NativeEventCreatedV1,
				Payload: roundevents.NativeEventCreatedPayloadV1{
					GuildID:        payload.GuildID,
					RoundID:        roundID,
					DiscordEventID: existingEventID,
				},
			},
		}

		// Add creator as a participant (Discord marks them as "interested" but
		// the UserAdd gateway event may fire before the NativeEventMap is populated)
		if payload.UserID != "" {
			zeroTag := sharedtypes.TagNumber(0)
			results = append(results, handlerwrapper.Result{
				Topic: roundevents.RoundParticipantJoinRequestedV1,
				Payload: roundevents.ParticipantJoinRequestPayloadV1{
					GuildID:   payload.GuildID,
					RoundID:   roundID,
					UserID:    sharedtypes.DiscordID(payload.UserID),
					Response:  roundtypes.ResponseAccept,
					TagNumber: &zeroTag,
				},
			})
		}

		return results, nil
	}

	// Convert sharedtypes.StartTime to time.Time
	startTimeValue := time.Time(*startTime)
	endTimeValue := startTimeValue.Add(3 * time.Hour)

	// Create the Discord Guild Scheduled Event parameters
	eventParams := &discordgo.GuildScheduledEventParams{
		Name:               title,
		Description:        description,
		ScheduledStartTime: &startTimeValue,
		ScheduledEndTime:   &endTimeValue,
		EntityType:         discordgo.GuildScheduledEventEntityTypeExternal, // Type 3
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
		nativeEventMap.Store(nativeEvent.ID, roundID, payload.GuildID, sharedtypes.DiscordID(payload.UserID))
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

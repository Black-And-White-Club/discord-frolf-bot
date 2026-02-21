package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	discordroundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/round"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	"github.com/bwmarrin/discordgo"
)

func (h *RoundHandlers) HandleRoundCreateRequested(ctx context.Context, payload *discordroundevents.CreateRoundModalPayloadV1) ([]handlerwrapper.Result, error) {
	// Convert to backend payload and set GuildID
	desc := payload.Description
	backendPayload := roundevents.CreateRoundRequestedPayloadV1{
		GuildID:     sharedtypes.GuildID(payload.GuildID),
		Title:       payload.Title,
		Description: &desc,
		StartTime:   payload.StartTime,
		Location:    payload.Location,
		UserID:      payload.UserID,
		ChannelID:   payload.ChannelID,
		Timezone:    payload.Timezone,
	}

	return []handlerwrapper.Result{
		{
			Topic:   roundevents.RoundCreationRequestedV1,
			Payload: backendPayload,
			Metadata: map[string]string{
				"submitted_at":   time.Now().UTC().Format(time.RFC3339),
				"user_timezone":  string(payload.Timezone),
				"raw_start_time": payload.StartTime,
			},
		},
	}, nil
}

// Handles the RoundCreated Event from the Backend
func (h *RoundHandlers) HandleRoundCreated(ctx context.Context, payload *roundevents.RoundCreatedPayloadV1) ([]handlerwrapper.Result, error) {
	roundID := payload.RoundID
	if payload.GuildID == "" {
		return nil, fmt.Errorf("GuildID missing in payload for round %s", roundID.String())
	}
	if payload.StartTime == nil {
		return []handlerwrapper.Result{
			{
				Topic: roundevents.NativeEventCreateFailedV1,
				Payload: roundevents.NativeEventCreateFailedPayloadV1{
					GuildID: payload.GuildID,
					RoundID: roundID,
					Error:   "RoundCreatedPayloadV1 missing StartTime",
				},
			},
		}, nil
	}

	guildID := string(payload.GuildID)
	channelID := string(payload.ChannelID)
	description := string(payload.Description)
	location := string(payload.Location)

	var results []handlerwrapper.Result

	// 1. Resolve or create native event first. Native event failure does not block
	// embed creation.
	discordEventID, nativeResults := h.resolveNativeEvent(ctx, payload)
	results = append(results, nativeResults...)

	// 2. Send the Embed (Wait for this to complete before sending URL)
	sendResult, err := h.service.GetCreateRoundManager().SendRoundEventEmbed(
		guildID,
		channelID,
		roundtypes.Title(payload.Title),
		roundtypes.Description(description),
		sharedtypes.StartTime(*payload.StartTime),
		roundtypes.Location(location),
		sharedtypes.DiscordID(payload.UserID),
		roundID,
	)
	if err != nil {
		if discordEventID != "" {
			h.logger.ErrorContext(ctx, "Round embed send failed after native event success; returning error for retry",
				attr.String("round_id", roundID.String()),
				attr.String("discord_event_id", discordEventID),
				attr.Error(err))
		}
		return nil, fmt.Errorf("failed to send round event embed: %w", err)
	}

	discordMsg, ok := sendResult.Success.(*discordgo.Message)
	if !ok || discordMsg == nil {
		embedErr := sendResult.Error
		if embedErr == nil {
			embedErr = fmt.Errorf("SendRoundEventEmbed did not return a Discord message on success for round %s", roundID.String())
		}
		if discordEventID != "" {
			h.logger.ErrorContext(ctx, "Round embed send returned invalid success payload after native event success; returning error for retry",
				attr.String("round_id", roundID.String()),
				attr.String("discord_event_id", discordEventID),
				attr.Error(embedErr))
		}
		if sendResult.Error != nil {
			return nil, fmt.Errorf("SendRoundEventEmbed service returned failure: %w", sendResult.Error)
		}
		return nil, fmt.Errorf("SendRoundEventEmbed did not return a Discord message on success for round %s", roundID.String())
	}

	discordMessageID := discordMsg.ID

	// Store the message ID in the map for future lookups
	h.service.GetMessageMap().Store(roundID, discordMessageID)

	results = append(results, handlerwrapper.Result{
		Topic: roundevents.RoundEventMessageIDUpdateV1,
		Payload: roundevents.RoundMessageIDUpdatePayloadV1{
			GuildID: payload.GuildID,
			RoundID: roundID,
		},
		Metadata: map[string]string{
			"discord_message_id": discordMessageID,
			"idempotency_key":    fmt.Sprintf("round-event-message-id-update:%s:%s", roundID.String(), discordMessageID),
		},
	})

	// 3. Send the URL Link directly after Embed if event was created successfully
	if discordEventID != "" {
		urlResult, err := h.service.GetCreateRoundManager().SendRoundEventURL(guildID, channelID, discordEventID)
		if err != nil {
			// Log an error but don't fail the entire handler since the core tasks succeeded
			h.logger.ErrorContext(ctx, "Failed to send event URL message", attr.String("round_id", roundID.String()), attr.Error(err))
		} else if urlResult.Error != nil {
			h.logger.ErrorContext(ctx, "Failed to send event URL message", attr.String("round_id", roundID.String()), attr.Error(urlResult.Error))
		}
	}

	return results, nil
}

func (h *RoundHandlers) resolveNativeEvent(ctx context.Context, payload *roundevents.RoundCreatedPayloadV1) (string, []handlerwrapper.Result) {
	roundID := payload.RoundID
	guildID := string(payload.GuildID)
	title := string(payload.Title)

	nativeEventMap := h.service.GetNativeEventMap()
	if nativeEventMap != nil {
		if existingEventID, ok := nativeEventMap.LookupByRoundID(roundID); ok && existingEventID != "" {
			return existingEventID, []handlerwrapper.Result{
				{
					Topic: roundevents.NativeEventCreatedV1,
					Payload: roundevents.NativeEventCreatedPayloadV1{
						GuildID:        payload.GuildID,
						RoundID:        roundID,
						DiscordEventID: existingEventID,
					},
				},
			}
		}
	}

	pendingKey := guildID + "|" + title
	if pendingNativeEventMap := h.service.GetPendingNativeEventMap(); pendingNativeEventMap != nil {
		if existingEventID, ok := pendingNativeEventMap.LoadAndDelete(pendingKey); ok && existingEventID != "" {
			if nativeEventMap != nil {
				nativeEventMap.Store(existingEventID, roundID, payload.GuildID, sharedtypes.DiscordID(payload.UserID))
			}

			// Best effort: stamp RoundID footer on user-created events for reconciliation.
			if session := h.service.GetSession(); session != nil {
				eventDescription := string(payload.Description) + fmt.Sprintf("\n---\nRoundID: %s", roundID.String())
				if _, err := session.GuildScheduledEventEdit(guildID, existingEventID, &discordgo.GuildScheduledEventParams{
					Description: eventDescription,
				}); err != nil {
					h.logger.WarnContext(ctx, "Failed to stamp RoundID on existing Discord scheduled event",
						attr.String("round_id", roundID.String()),
						attr.String("discord_event_id", existingEventID),
						attr.Error(err))
				}
			}

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

			// Add creator as participant. The gateway user-add event may have fired
			// before the mapping was available.
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

			return existingEventID, results
		}
	}

	nativeResult, err := h.service.GetCreateRoundManager().CreateNativeEvent(
		ctx,
		guildID,
		roundID,
		roundtypes.Title(payload.Title),
		roundtypes.Description(payload.Description),
		sharedtypes.StartTime(*payload.StartTime),
		roundtypes.Location(payload.Location),
		sharedtypes.DiscordID(payload.UserID),
	)
	if err != nil {
		return "", []handlerwrapper.Result{
			{
				Topic: roundevents.NativeEventCreateFailedV1,
				Payload: roundevents.NativeEventCreateFailedPayloadV1{
					GuildID: payload.GuildID,
					RoundID: roundID,
					Error:   err.Error(),
				},
			},
		}
	}
	if nativeResult.Error != nil {
		return "", []handlerwrapper.Result{
			{
				Topic: roundevents.NativeEventCreateFailedV1,
				Payload: roundevents.NativeEventCreateFailedPayloadV1{
					GuildID: payload.GuildID,
					RoundID: roundID,
					Error:   nativeResult.Error.Error(),
				},
			},
		}
	}

	nativeEvent, ok := nativeResult.Success.(*discordgo.GuildScheduledEvent)
	if !ok || nativeEvent == nil || nativeEvent.ID == "" {
		return "", []handlerwrapper.Result{
			{
				Topic: roundevents.NativeEventCreateFailedV1,
				Payload: roundevents.NativeEventCreateFailedPayloadV1{
					GuildID: payload.GuildID,
					RoundID: roundID,
					Error:   fmt.Sprintf("CreateNativeEvent returned unexpected success payload type %T", nativeResult.Success),
				},
			},
		}
	}

	if nativeEventMap != nil {
		nativeEventMap.Store(nativeEvent.ID, roundID, payload.GuildID, sharedtypes.DiscordID(payload.UserID))
	}

	return nativeEvent.ID, []handlerwrapper.Result{
		{
			Topic: roundevents.NativeEventCreatedV1,
			Payload: roundevents.NativeEventCreatedPayloadV1{
				GuildID:        payload.GuildID,
				RoundID:        roundID,
				DiscordEventID: nativeEvent.ID,
			},
		},
	}
}

func (h *RoundHandlers) HandleRoundCreationFailed(ctx context.Context, payload *roundevents.RoundCreationFailedPayloadV1) ([]handlerwrapper.Result, error) {
	// Prepare the error message
	errorMessage := "❌ Round creation failed: " + payload.ErrorMessage

	// Call the gateway handler to update the interaction response with a retry button
	// This is a side-effect only, no outgoing messages
	correlationID, ok := ctx.Value("correlation_id").(string)
	if !ok {
		// correlationID not available in context, skip update
		return nil, nil
	}

	_, err := h.service.GetCreateRoundManager().UpdateInteractionResponseWithRetryButton(ctx, correlationID, errorMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to update interaction response: %w", err)
	}

	return nil, nil
}

func (h *RoundHandlers) HandleRoundValidationFailed(ctx context.Context, payload *roundevents.RoundValidationFailedPayloadV1) ([]handlerwrapper.Result, error) {
	errorMessages := payload.ErrorMessages
	errorMessage := "❌ " + strings.Join(errorMessages, "\n") + " Please try again."

	correlationID, ok := ctx.Value("correlation_id").(string)
	if !ok {
		// correlationID not available in context, skip update
		return nil, nil
	}

	_, err := h.service.GetCreateRoundManager().UpdateInteractionResponseWithRetryButton(ctx, correlationID, errorMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to update interaction response: %w", err)
	}

	return nil, nil
}

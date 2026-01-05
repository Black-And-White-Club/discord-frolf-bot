package leaderboardhandlers

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	sharedleaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/leaderboard"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleTagAssignRequest translates a Discord tag assignment request directly to a batch assignment.
func (h *LeaderboardHandlers) HandleTagAssignRequest(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleTagAssignRequest",
		&sharedleaderboardevents.LeaderboardTagAssignRequestPayloadV1{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			h.Logger.InfoContext(ctx, "Handling TagAssignRequest", attr.CorrelationIDFromMsg(msg))

			discordPayload := payload.(*sharedleaderboardevents.LeaderboardTagAssignRequestPayloadV1)

			// Validation
			if discordPayload.TargetUserID == "" || discordPayload.RequestorID == "" ||
				discordPayload.TagNumber <= 0 || discordPayload.ChannelID == "" || discordPayload.MessageID == "" {
				err := fmt.Errorf("invalid TagAssignRequest payload: missing required fields")
				h.Logger.ErrorContext(ctx, err.Error(), attr.CorrelationIDFromMsg(msg))
				return nil, err
			}

			// Validate MessageID is a valid UUID format
			if _, err := uuid.Parse(discordPayload.MessageID); err != nil {
				err := fmt.Errorf("invalid TagAssignRequest payload: MessageID is not a valid UUID: %w", err)
				h.Logger.ErrorContext(ctx, err.Error(), attr.CorrelationIDFromMsg(msg))
				return nil, err
			}

			// Create single assignment payload
			tagNumber := discordPayload.TagNumber
			// Parse MessageID as UUID for the update tracking ID
			updateUUID, err := uuid.Parse(discordPayload.MessageID)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to parse message ID as UUID", attr.Error(err), attr.CorrelationIDFromMsg(msg))
				return nil, fmt.Errorf("failed to parse message ID as UUID: %w", err)
			}
			updateID := sharedtypes.RoundID(updateUUID)
			backendPayload := leaderboardevents.LeaderboardTagAssignmentRequestedPayloadV1{
				GuildID:    sharedtypes.GuildID(discordPayload.GuildID),
				UserID:     discordPayload.TargetUserID,
				TagNumber:  &tagNumber,
				UpdateID:   updateID,
				Source:     "discord_claim",
				UpdateType: "manual_assignment",
			}

			// Create assignment message
			backendMsg, err := h.Helpers.CreateResultMessage(
				msg,
				backendPayload,
				leaderboardevents.LeaderboardTagAssignmentRequestedV1,
			)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to create assignment message",
					attr.CorrelationIDFromMsg(msg), attr.Error(err))
				return nil, fmt.Errorf("failed to create assignment message: %w", err)
			}

			// Preserve Discord-specific metadata for response handling
			backendMsg.Metadata.Set("user_id", string(discordPayload.TargetUserID))
			backendMsg.Metadata.Set("requestor_id", string(discordPayload.RequestorID))
			backendMsg.Metadata.Set("channel_id", discordPayload.ChannelID)
			backendMsg.Metadata.Set("message_id", discordPayload.MessageID)
			// Propagate guild_id metadata for multi-tenant routing across services
			if discordPayload.GuildID != "" {
				backendMsg.Metadata.Set("guild_id", discordPayload.GuildID)
			}
			backendMsg.Metadata.Set("source", "discord_claim")

			h.Logger.InfoContext(ctx, "Successfully created assignment for Discord claim",
				attr.CorrelationIDFromMsg(msg),
				attr.String("update_id", backendPayload.UpdateID.String()))

			return []*message.Message{backendMsg}, nil
		},
	)(msg)
}

// HandleTagAssignedResponse translates a backend TagAssigned event to a Discord response.
func (h *LeaderboardHandlers) HandleTagAssignedResponse(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleTagAssignedResponse",
		&leaderboardevents.LeaderboardTagAssignedPayloadV1{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			h.Logger.InfoContext(ctx, "Handling TagAssignedResponse", attr.CorrelationIDFromMsg(msg))

			backendPayload := payload.(*leaderboardevents.LeaderboardTagAssignedPayloadV1)
			correlationID := msg.Metadata.Get("correlation_id")

			// If this is from a Discord claim command, update the interaction directly
			if correlationID != "" {
				successMessage := fmt.Sprintf("✅ Successfully claimed tag #%d!", *backendPayload.TagNumber)

				// Get the claim tag manager and update the interaction
				if h.LeaderboardDiscord != nil {
					claimTagManager := h.LeaderboardDiscord.GetClaimTagManager()
					if claimTagManager != nil {
						result, err := claimTagManager.UpdateInteractionResponse(ctx, correlationID, successMessage)
						if err != nil {
							h.Logger.ErrorContext(ctx, "Failed to update Discord interaction for tag success",
								attr.CorrelationIDFromMsg(msg),
								attr.String("correlation_id", correlationID),
								attr.Error(err))
							// Don't fail the whole handler - log and continue
						} else {
							h.Logger.InfoContext(ctx, "Successfully updated Discord interaction for tag claim success",
								attr.CorrelationIDFromMsg(msg),
								attr.String("correlation_id", correlationID),
								attr.String("result", fmt.Sprintf("%v", result.Success)))
						}
					} else {
						h.Logger.WarnContext(ctx, "ClaimTagManager is nil, cannot update Discord interaction",
							attr.CorrelationIDFromMsg(msg),
							attr.String("correlation_id", correlationID))
					}
				} else {
					h.Logger.WarnContext(ctx, "LeaderboardDiscord is nil, cannot update Discord interaction",
						attr.CorrelationIDFromMsg(msg),
						attr.String("correlation_id", correlationID))
				}
			}

			userID := msg.Metadata.Get("user_id")
			requestorID := msg.Metadata.Get("requestor_id")
			channelID := msg.Metadata.Get("channel_id")
			messageID := msg.Metadata.Get("message_id")

			discordPayload := sharedleaderboardevents.LeaderboardTagAssignedPayloadV1{
				TargetUserID: string(backendPayload.UserID),
				TagNumber:    *backendPayload.TagNumber,
				ChannelID:    channelID,
				MessageID:    messageID,
				GuildID:      string(backendPayload.GuildID),
			}

			discordMsg, err := h.Helpers.CreateResultMessage(msg, discordPayload, sharedleaderboardevents.LeaderboardTagAssignedV1)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to create discord message", attr.CorrelationIDFromMsg(msg), attr.Error(err))
				return nil, fmt.Errorf("failed to create discord message: %w", err)
			}

			h.Logger.InfoContext(ctx, "Successfully translated TagAssignedResponse",
				attr.CorrelationIDFromMsg(msg),
				attr.String("user_id", userID),
				attr.String("requestor_id", requestorID),
			)

			return []*message.Message{discordMsg}, nil
		},
	)(msg)
}

// HandleTagAssignFailedResponse translates a backend TagAssignmentFailed event to a Discord response.
func (h *LeaderboardHandlers) HandleTagAssignFailedResponse(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleTagAssignFailedResponse",
		&leaderboardevents.LeaderboardTagAssignmentFailedPayloadV1{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			h.Logger.InfoContext(ctx, "Handling TagAssignFailedResponse", attr.CorrelationIDFromMsg(msg))

			backendPayload := payload.(*leaderboardevents.LeaderboardTagAssignmentFailedPayloadV1)
			correlationID := msg.Metadata.Get("correlation_id")

			// If this is from a Discord claim command, update the interaction directly
			if correlationID != "" {
				errorMessage := fmt.Sprintf("❌ Could not claim tag #%d: %s", *backendPayload.TagNumber, backendPayload.Reason)

				// Get the claim tag manager and update the interaction
				if h.LeaderboardDiscord != nil {
					claimTagManager := h.LeaderboardDiscord.GetClaimTagManager()
					if claimTagManager != nil {
						result, err := claimTagManager.UpdateInteractionResponse(ctx, correlationID, errorMessage)
						if err != nil {
							h.Logger.ErrorContext(ctx, "Failed to update Discord interaction for tag failure",
								attr.CorrelationIDFromMsg(msg),
								attr.String("correlation_id", correlationID),
								attr.Error(err))
							// Don't fail the whole handler - log and continue
						} else {
							h.Logger.InfoContext(ctx, "Successfully updated Discord interaction for tag claim failure",
								attr.CorrelationIDFromMsg(msg),
								attr.String("correlation_id", correlationID),
								attr.String("result", fmt.Sprintf("%v", result.Success)))
						}
					} else {
						h.Logger.WarnContext(ctx, "ClaimTagManager is nil, cannot update Discord interaction",
							attr.CorrelationIDFromMsg(msg),
							attr.String("correlation_id", correlationID))
					}
				} else {
					h.Logger.WarnContext(ctx, "LeaderboardDiscord is nil, cannot update Discord interaction",
						attr.CorrelationIDFromMsg(msg),
						attr.String("correlation_id", correlationID))
				}
			}

			userID := msg.Metadata.Get("user_id")
			requestorID := msg.Metadata.Get("requestor_id")
			channelID := msg.Metadata.Get("channel_id")
			messageID := msg.Metadata.Get("message_id")

			discordPayload := sharedleaderboardevents.LeaderboardTagAssignFailedPayloadV1{
				TargetUserID: string(backendPayload.UserID),
				TagNumber:    *backendPayload.TagNumber,
				Reason:       backendPayload.Reason,
				ChannelID:    channelID,
				MessageID:    messageID,
				GuildID:      string(backendPayload.GuildID),
			}

			discordMsg, err := h.Helpers.CreateResultMessage(msg, discordPayload, sharedleaderboardevents.LeaderboardTagAssignFailedV1)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to create discord message", attr.CorrelationIDFromMsg(msg), attr.Error(err))
				return nil, fmt.Errorf("failed to create discord message: %w", err)
			}

			h.Logger.InfoContext(ctx, "Successfully translated TagAssignFailedResponse",
				attr.CorrelationIDFromMsg(msg),
				attr.String("user_id", userID),
				attr.String("requestor_id", requestorID),
			)

			return []*message.Message{discordMsg}, nil
		},
	)(msg)
}

package handlers

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	discordleaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/leaderboard"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleTagAssignRequest translates a Discord tag assignment request directly to a batch assignment.
func (h *LeaderboardHandlers) HandleTagAssignRequest(ctx context.Context,
	payload *discordleaderboardevents.LeaderboardTagAssignRequestPayloadV1) ([]handlerwrapper.Result, error) {
	h.logger.InfoContext(ctx, "Handling TagAssignRequest")

	discordPayload := payload

	// Validation
	if discordPayload.TargetUserID == "" || discordPayload.RequestorID == "" ||
		discordPayload.TagNumber <= 0 || discordPayload.ChannelID == "" || discordPayload.MessageID == "" {
		err := fmt.Errorf("invalid TagAssignRequest payload: missing required fields")
		h.logger.ErrorContext(ctx, err.Error(),
			attr.String("target_user_id", string(discordPayload.TargetUserID)),
			attr.String("requestor_id", string(discordPayload.RequestorID)),
		)
		return nil, err
	}

	// Validate MessageID is a valid UUID format
	if _, err := uuid.Parse(discordPayload.MessageID); err != nil {
		err := fmt.Errorf("invalid TagAssignRequest payload: MessageID is not a valid UUID: %w", err)
		h.logger.ErrorContext(ctx, err.Error(), attr.Error(err))
		return nil, err
	}

	// Create single assignment payload
	tagNumber := discordPayload.TagNumber
	// Parse MessageID as UUID for the update tracking ID
	updateUUID, err := uuid.Parse(discordPayload.MessageID)
	if err != nil {
		h.logger.ErrorContext(ctx, "Failed to parse message ID as UUID", attr.Error(err))
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

	h.logger.InfoContext(ctx, "Successfully created assignment for Discord claim",
		attr.String("update_id", backendPayload.UpdateID.String()),
		attr.String("target_user_id", string(discordPayload.TargetUserID)),
		attr.Int("tag_number", int(tagNumber)))

	return []handlerwrapper.Result{
		{
			Topic:   leaderboardevents.LeaderboardBatchTagAssignmentRequestedV1,
			Payload: backendPayload,
			Metadata: map[string]string{
				"user_id":      string(discordPayload.TargetUserID),
				"requestor_id": string(discordPayload.RequestorID),
				"channel_id":   discordPayload.ChannelID,
				"message_id":   discordPayload.MessageID,
				"guild_id":     discordPayload.GuildID,
				"source":       "discord_claim",
			},
		},
	}, nil
}

// HandleTagAssignedResponse translates a backend TagAssigned event to a Discord response.
func (h *LeaderboardHandlers) HandleTagAssignedResponse(ctx context.Context,
	payload *leaderboardevents.LeaderboardTagAssignedPayloadV1) ([]handlerwrapper.Result, error) {
	h.logger.InfoContext(ctx, "Handling TagAssignedResponse")

	backendPayload := payload

	// Extract correlation ID from context if available
	correlationID := ""
	if val := ctx.Value("correlation_id"); val != nil {
		if corr, ok := val.(string); ok {
			correlationID = corr
		}
	}

	// If this is from a Discord claim command, update the interaction directly
	if correlationID != "" {
		successMessage := fmt.Sprintf("✅ Successfully claimed tag #%d!", *backendPayload.TagNumber)

		// Get the claim tag manager and update the interaction
		if h.service != nil {
			claimTagManager := h.service.GetClaimTagManager()
			if claimTagManager != nil {
				result, err := claimTagManager.UpdateInteractionResponse(ctx, correlationID, successMessage)
				if err != nil {
					h.logger.ErrorContext(ctx, "Failed to update Discord interaction for tag success",
						attr.String("correlation_id", correlationID),
						attr.Error(err))
					// Don't fail the whole handler - log and continue
				} else {
					h.logger.InfoContext(ctx, "Successfully updated Discord interaction for tag claim success",
						attr.String("correlation_id", correlationID),
						attr.String("result", fmt.Sprintf("%v", result.Success)))
				}
			} else {
				h.logger.WarnContext(ctx, "ClaimTagManager is nil, cannot update Discord interaction",
					attr.String("correlation_id", correlationID))
			}
		} else {
			h.logger.WarnContext(ctx, "LeaderboardDiscord is nil, cannot update Discord interaction",
				attr.String("correlation_id", correlationID))
		}
	}

	discordPayload := discordleaderboardevents.LeaderboardTagAssignedPayloadV1{
		TargetUserID: string(backendPayload.UserID),
		TagNumber:    *backendPayload.TagNumber,
		GuildID:      string(backendPayload.GuildID),
	}

	h.logger.InfoContext(ctx, "Successfully translated TagAssignedResponse",
		attr.String("target_user_id", string(backendPayload.UserID)),
		attr.Int("tag_number", int(*backendPayload.TagNumber)),
	)

	return []handlerwrapper.Result{
		{
			Topic:   discordleaderboardevents.LeaderboardTagAssignedV1,
			Payload: discordPayload,
		},
	}, nil
}

// HandleTagAssignFailedResponse translates a backend TagAssignmentFailed event to a Discord response.
func (h *LeaderboardHandlers) HandleTagAssignFailedResponse(ctx context.Context,
	payload *leaderboardevents.LeaderboardTagAssignmentFailedPayloadV1) ([]handlerwrapper.Result, error) {
	h.logger.InfoContext(ctx, "Handling TagAssignFailedResponse")

	backendPayload := payload

	// Extract correlation ID from context if available
	correlationID := ""
	if val := ctx.Value("correlation_id"); val != nil {
		if corr, ok := val.(string); ok {
			correlationID = corr
		}
	}

	// If this is from a Discord claim command, update the interaction directly
	if correlationID != "" {
		errorMessage := fmt.Sprintf("❌ Could not claim tag #%d: %s", *backendPayload.TagNumber, backendPayload.Reason)

		// Get the claim tag manager and update the interaction
		if h.service != nil {
			claimTagManager := h.service.GetClaimTagManager()
			if claimTagManager != nil {
				result, err := claimTagManager.UpdateInteractionResponse(ctx, correlationID, errorMessage)
				if err != nil {
					h.logger.ErrorContext(ctx, "Failed to update Discord interaction for tag failure",
						attr.String("correlation_id", correlationID),
						attr.Error(err))
					// Don't fail the whole handler - log and continue
				} else {
					h.logger.InfoContext(ctx, "Successfully updated Discord interaction for tag claim failure",
						attr.String("correlation_id", correlationID),
						attr.String("result", fmt.Sprintf("%v", result.Success)))
				}
			} else {
				h.logger.WarnContext(ctx, "ClaimTagManager is nil, cannot update Discord interaction",
					attr.String("correlation_id", correlationID))
			}
		} else {
			h.logger.WarnContext(ctx, "LeaderboardDiscord is nil, cannot update Discord interaction",
				attr.String("correlation_id", correlationID))
		}
	}

	discordPayload := discordleaderboardevents.LeaderboardTagAssignFailedPayloadV1{
		TargetUserID: string(backendPayload.UserID),
		TagNumber:    *backendPayload.TagNumber,
		Reason:       backendPayload.Reason,
		GuildID:      string(backendPayload.GuildID),
	}

	h.logger.InfoContext(ctx, "Successfully translated TagAssignFailedResponse",
		attr.String("target_user_id", string(backendPayload.UserID)),
		attr.Int("tag_number", int(*backendPayload.TagNumber)),
		attr.String("reason", backendPayload.Reason),
	)

	return []handlerwrapper.Result{
		{
			Topic:   discordleaderboardevents.LeaderboardTagAssignFailedV1,
			Payload: discordPayload,
		},
	}, nil
}

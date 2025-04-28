package deleteround

import (
	"context"
	"fmt"
	"strings"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

// HandleDeleteRound handles the delete round button interaction.
func (drm *deleteRoundManager) HandleDeleteRoundButton(ctx context.Context, i *discordgo.InteractionCreate) (DeleteRoundOperationResult, error) {
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "handle_delete_round")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "button")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, i.Member.User.ID)

	drm.logger.InfoContext(ctx, "Handling delete round",
		attr.String("interaction_id", i.ID),
		attr.String("custom_id", i.MessageComponentData().CustomID),
		attr.UserID(sharedtypes.DiscordID(i.Member.User.ID)))

	user := i.Member.User
	customID := i.MessageComponentData().CustomID

	// Extract the round ID safely
	parts := strings.Split(customID, "|")
	if len(parts) < 2 {
		err := fmt.Errorf("invalid custom_id format: %s", customID)
		drm.logger.ErrorContext(ctx, err.Error(), attr.String("custom_id", customID))
		return DeleteRoundOperationResult{Error: err}, nil
	}
	roundIDStr := parts[1]

	// Convert roundID to uuid.UUID
	roundUUID, err := uuid.Parse(roundIDStr)
	if err != nil {
		err = fmt.Errorf("failed to parse round ID as UUID: %w", err)
		drm.logger.ErrorContext(ctx, err.Error(), attr.String("round_id_str", roundIDStr))
		_, err2 := drm.session.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "❌ Invalid Round ID. Please try again.", // User-friendly error
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		if err2 != nil {
			drm.logger.ErrorContext(ctx, "Failed to send ephemeral error message", attr.Error(err2))
			return DeleteRoundOperationResult{Error: fmt.Errorf("failed to parse round ID: %w, and also failed to send error message: %w", err, err2)}, nil
		}
		return DeleteRoundOperationResult{Error: err}, nil
	}
	roundID := sharedtypes.RoundID(roundUUID)
	userID := sharedtypes.DiscordID(user.ID)

	drm.logger.InfoContext(ctx, "Processing delete request",
		attr.RoundID("round_id", roundID),
		attr.UserID(userID))

	err = drm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
	if err != nil {
		err = fmt.Errorf("failed to acknowledge interaction: %w", err)
		drm.logger.ErrorContext(ctx, err.Error())
		return DeleteRoundOperationResult{Error: err}, nil
	}

	drm.logger.InfoContext(ctx, "Calling sendDeleteRequest",
		attr.RoundID("round_id", roundID),
		attr.UserID(userID))

	if err := drm.sendDeleteRequest(ctx, roundID, userID, i.ID); err != nil {
		err = fmt.Errorf("failed to publish delete request: %w", err)
		drm.logger.ErrorContext(ctx, err.Error(), attr.RoundID("round_id", roundID), attr.UserID(userID))
		// Send an ephemeral error response to the user
		_, err2 := drm.session.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "❌ Failed to delete the round.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		if err2 != nil {
			drm.logger.ErrorContext(ctx, "Failed to send ephemeral error message", attr.Error(err2))
			return DeleteRoundOperationResult{Error: fmt.Errorf("failed to publish delete request: %w, and also failed to send error message: %w", err, err2)}, nil
		}
		return DeleteRoundOperationResult{Error: err}, nil
	}

	// Send an ephemeral success message to the user
	_, err = drm.session.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: "🗑️ Round deletion request sent.",
		Flags:   discordgo.MessageFlagsEphemeral,
	})
	if err != nil {
		drm.logger.ErrorContext(ctx, "Failed to send ephemeral follow-up message: %v", attr.Error(err))
		return DeleteRoundOperationResult{Error: fmt.Errorf("failed to send follow-up message: %w", err)}, nil // Return the error
	}
	return DeleteRoundOperationResult{Success: "delete request sent"}, nil
}

// sendDeleteRequest publishes the delete request to the backend.
func (drm *deleteRoundManager) sendDeleteRequest(ctx context.Context, roundID sharedtypes.RoundID, userID sharedtypes.DiscordID, interactionID string) error {
	drm.logger.InfoContext(ctx, "sendDeleteRequest called",
		attr.RoundID("round_id", roundID),
		attr.UserID(userID),
		attr.String("interaction_id", interactionID))

	// Prepare the payload
	payload := roundevents.RoundDeleteRequestPayload{
		RoundID:              roundID,
		RequestingUserUserID: userID,
	}

	// Generate the result message without an original message
	resultMsg, err := drm.helper.CreateResultMessage(nil, payload, roundevents.RoundDeleteRequest)
	if err != nil {
		drm.logger.ErrorContext(ctx, "Failed to create result message", attr.Error(err))
		return fmt.Errorf("failed to create result message: %w", err)
	}

	// Attach the context to the message
	resultMsg.SetContext(ctx)

	drm.logger.InfoContext(ctx, "Publishing delete request",
		attr.String("topic", roundevents.RoundDeleteRequest),
		attr.String("message_id", resultMsg.UUID))

	// Publish the delete request
	if err := drm.publisher.Publish(roundevents.RoundDeleteRequest, resultMsg); err != nil {
		drm.logger.ErrorContext(ctx, "Failed to publish delete request", attr.Error(err))
		return fmt.Errorf("failed to publish delete request: %w", err)
	}

	drm.logger.InfoContext(ctx, "Successfully published delete request",
		attr.RoundID("round_id", roundID),
		attr.String("message_id", resultMsg.UUID))

	return nil
}

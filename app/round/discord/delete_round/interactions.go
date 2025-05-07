package deleteround

import (
	"context"
	"fmt"
	"strings"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
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
		attr.UserID(sharedtypes.DiscordID(i.Member.User.ID)),
		attr.String("discord_message_id", i.Message.ID),
	)

	user := i.Member.User
	customID := i.MessageComponentData().CustomID
	discordMessageID := i.Message.ID

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
		// Send an ephemeral error response to the user
		_, err2 := drm.session.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "❌ Invalid Round ID. Please try again.",
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
		attr.UserID(userID),
		attr.String("discord_message_id", discordMessageID),
	)

	// Acknowledge the interaction immediately
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
		attr.UserID(userID),
		attr.String("interaction_id", i.ID),
		attr.String("discord_message_id", discordMessageID),
	)

	// **Pass the discordMessageID to sendDeleteRequest**
	if err := drm.sendDeleteRequest(ctx, roundID, userID, i.ID, discordMessageID); err != nil {
		err = fmt.Errorf("failed to publish delete request: %w", err)
		drm.logger.ErrorContext(ctx, err.Error(), attr.RoundID("round_id", roundID), attr.UserID(userID), attr.String("discord_message_id", discordMessageID))
		// Send an ephemeral error response to the user
		_, err2 := drm.session.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "❌ Failed to send the round deletion request.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		if err2 != nil {
			drm.logger.ErrorContext(ctx, "Failed to send ephemeral error message", attr.Error(err2))
			return DeleteRoundOperationResult{Error: fmt.Errorf("failed to publish delete request: %w, and also failed to send error message: %w", err, err2)}, nil
		}
		return DeleteRoundOperationResult{Error: err}, nil
	}

	// Send an ephemeral success message to the user indicating the request was sent
	_, err = drm.session.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: "🗑️ Round deletion request sent. The round message will be removed shortly if the deletion is successful.",
		Flags:   discordgo.MessageFlagsEphemeral,
	})
	if err != nil {
		drm.logger.ErrorContext(ctx, "Failed to send ephemeral follow-up message: %v", attr.Error(err))

		return DeleteRoundOperationResult{Success: "delete request sent, but confirmation message failed"}, nil
	}

	return DeleteRoundOperationResult{Success: "delete request sent"}, nil
}

// sendDeleteRequest publishes the delete request to the backend.
func (drm *deleteRoundManager) sendDeleteRequest(ctx context.Context, roundID sharedtypes.RoundID, userID sharedtypes.DiscordID, interactionID string, discordMessageID string) error {
	drm.logger.InfoContext(ctx, "sendDeleteRequest called",
		attr.RoundID("round_id", roundID),
		attr.UserID(userID),
		attr.String("interaction_id", interactionID),
		attr.String("discord_message_id", discordMessageID), // Log the message ID being sent
	)

	// Prepare the payload for the backend
	payload := roundevents.RoundDeleteRequestPayload{
		RoundID:              roundID,
		RequestingUserUserID: userID,
	}

	// **Create the message struct with metadata including the discord_message_id**
	msgToSend := &message.Message{
		Metadata: message.Metadata{
			"discord_message_id": discordMessageID,
		},
	}

	// Generate the result message using the created message struct.
	resultMsg, err := drm.helper.CreateResultMessage(msgToSend, payload, roundevents.RoundDeleteRequest)
	if err != nil {
		drm.logger.ErrorContext(ctx, "Failed to create result message", attr.Error(err))
		return fmt.Errorf("failed to create result message: %w", err)
	}

	resultMsg.SetContext(ctx)

	drm.logger.InfoContext(ctx, "Publishing delete request",
		attr.String("topic", roundevents.RoundDeleteRequest),
		attr.String("message_id", resultMsg.UUID),
		attr.String("discord_message_id", discordMessageID),
	)

	// Publish the delete request message
	if err := drm.publisher.Publish(roundevents.RoundDeleteRequest, resultMsg); err != nil {
		drm.logger.ErrorContext(ctx, "Failed to publish delete request", attr.Error(err))
		return fmt.Errorf("failed to publish delete request: %w", err)
	}

	drm.logger.InfoContext(ctx, "Successfully published delete request",
		attr.RoundID("round_id", roundID),
		attr.String("message_id", resultMsg.UUID),
		attr.String("discord_message_id", discordMessageID),
	)

	return nil
}

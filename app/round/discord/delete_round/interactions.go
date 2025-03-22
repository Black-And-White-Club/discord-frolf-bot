package deleteround

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"strconv"
	"strings"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/bwmarrin/discordgo"
)

// HandleDeleteRound handles the delete round button interaction.
func (drm *deleteRoundManager) HandleDeleteRound(ctx context.Context, i *discordgo.InteractionCreate) {
	slog.Info("HandleDeleteRound called",
		attr.String("interaction_id", i.ID),
		attr.String("custom_id", i.MessageComponentData().CustomID),
		attr.String("user", i.Member.User.Username))
	user := i.Member.User
	customID := i.MessageComponentData().CustomID

	// Extract the round ID safely
	parts := strings.Split(customID, "|")
	if len(parts) < 2 {
		log.Printf("Invalid custom_id format: %s", customID)
		return
	}
	roundIDStr := parts[1]

	// Convert roundID to int64
	roundIDInt, err := strconv.ParseInt(roundIDStr, 10, 64)
	if err != nil {
		log.Printf("Failed to parse round ID: %v", err)
		return
	}
	roundID := roundtypes.ID(roundIDInt)
	userID := roundtypes.UserID(user.ID)

	log.Printf("Processing delete request for round %s by user %s", roundIDStr, user.Username)

	err = drm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
	if err != nil {
		log.Printf("Failed to acknowledge interaction: %v", err)
		return
	}

	slog.Info("Calling sendDeleteRequest",
		attr.String("round_id", roundIDStr),
		attr.String("user_id", user.ID))

	if err := drm.sendDeleteRequest(ctx, roundID, userID, i.ID); err != nil {
		log.Printf("Failed to publish delete request for round %s: %v", roundIDStr, err)

		// Send an ephemeral error response to the user
		_, _ = drm.session.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "❌ Failed to delete the round.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return
	}

	// Send an ephemeral success message to the user
	_, err = drm.session.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: "🗑️ Round deletion request sent.",
		Flags:   discordgo.MessageFlagsEphemeral,
	})
	if err != nil {
		log.Printf("Failed to send ephemeral follow-up message: %v", err)
	}
}

// sendDeleteRequest publishes the delete request to the backend.
func (drm *deleteRoundManager) sendDeleteRequest(ctx context.Context, roundID roundtypes.ID, userID roundtypes.UserID, interactionID string) error {
	slog.Info("sendDeleteRequest called",
		attr.String("round_id", fmt.Sprintf("%d", roundID)),
		attr.String("user_id", string(userID)),
		attr.String("interaction_id", interactionID))

	// Prepare the payload
	payload := roundevents.RoundDeleteRequestPayload{
		RoundID:              roundID,
		RequestingUserUserID: userID,
	}

	// Generate the result message without an original message
	resultMsg, err := drm.helper.CreateResultMessage(nil, payload, roundevents.RoundDeleteRequest)
	if err != nil {
		slog.Error("Failed to create result message", attr.Error(err))
		return fmt.Errorf("failed to create result message: %w", err)
	}

	// Attach the context to the message
	resultMsg.SetContext(ctx)

	slog.Info("Publishing delete request",
		attr.String("topic", roundevents.RoundDeleteRequest),
		attr.String("message_id", resultMsg.UUID))

	// Publish the delete request
	if err := drm.publisher.Publish(roundevents.RoundDeleteRequest, resultMsg); err != nil {
		slog.Error("Failed to publish delete request", attr.Error(err))
		return fmt.Errorf("failed to publish delete request: %w", err)
	}

	slog.Info("Successfully published delete request",
		attr.String("round_id", fmt.Sprintf("%d", roundID)),
		attr.String("message_id", resultMsg.UUID))

	return nil
}

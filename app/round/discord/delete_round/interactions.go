package deleteround

import (
	"context"
	"fmt"
	"strings"

	discordroundevents "github.com/Black-And-White-Club/discord-frolf-bot/app/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

// HandleDeleteRoundButton handles the delete round button interaction.
func (drm *deleteRoundManager) HandleDeleteRoundButton(ctx context.Context, i *discordgo.InteractionCreate) (DeleteRoundOperationResult, error) {
	return drm.operationWrapper(ctx, "HandleDeleteRoundButton", func(ctx context.Context) (DeleteRoundOperationResult, error) {
		ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "handle_delete_round")
		ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "button")

		// Extract the round ID safely - fix the type assertion
		data := i.MessageComponentData()

		parts := strings.Split(data.CustomID, "|")
		if len(parts) != 2 {
			err := fmt.Errorf("invalid custom_id format: expected 'round_delete|<uuid>', got '%s'", data.CustomID)
			drm.logger.ErrorContext(ctx, "Invalid custom_id format for delete round button", attr.Error(err))
			return DeleteRoundOperationResult{Error: err}, nil
		}

		// Convert roundID to uuid.UUID
		roundUUID, err := uuid.Parse(parts[1])
		if err != nil {
			err = fmt.Errorf("failed to parse round ID as UUID: %w", err)
			drm.logger.ErrorContext(ctx, "Failed to parse round ID as UUID",
				attr.String("round_id_string", parts[1]),
				attr.Error(err))

			// Send error followup message
			_, followupErr := drm.session.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Content: "❌ Invalid round ID format.",
				Flags:   discordgo.MessageFlagsEphemeral,
			})
			if followupErr != nil {
				drm.logger.ErrorContext(ctx, "Failed to send error followup message", attr.Error(followupErr))
			}

			return DeleteRoundOperationResult{Error: err}, nil
		}

		// Acknowledge the interaction immediately
		err = drm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			err = fmt.Errorf("failed to acknowledge interaction: %w", err)
			drm.logger.ErrorContext(ctx, "Failed to acknowledge delete round interaction", attr.Error(err))
			return DeleteRoundOperationResult{Error: err}, nil
		}

		// Get user ID safely
		var userID sharedtypes.DiscordID
		if i.Member != nil && i.Member.User != nil {
			userID = sharedtypes.DiscordID(i.Member.User.ID)
		} else if i.User != nil {
			userID = sharedtypes.DiscordID(i.User.ID)
		} else {
			err := fmt.Errorf("unable to determine user ID from interaction")
			drm.logger.ErrorContext(ctx, "Unable to determine user ID", attr.Error(err))
			return DeleteRoundOperationResult{Error: err}, nil
		}

		// Get discord message ID
		discordMessageID := ""
		if i.Message != nil {
			discordMessageID = i.Message.ID
		}

		// Send delete request
		err = drm.sendDeleteRequest(ctx, sharedtypes.RoundID(roundUUID), userID, i.Interaction.ID, discordMessageID, i.GuildID)
		if err != nil {
			drm.logger.ErrorContext(ctx, "Failed to send delete request", attr.Error(err))

			// Send error followup message
			_, followupErr := drm.session.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Content: "❌ Failed to process delete request.",
				Flags:   discordgo.MessageFlagsEphemeral,
			})
			if followupErr != nil {
				drm.logger.ErrorContext(ctx, "Failed to send error followup message", attr.Error(followupErr))
			}

			return DeleteRoundOperationResult{Error: err}, nil
		}

		// Send an ephemeral success message to the user indicating the request was sent
		_, err = drm.session.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "✅ Delete request sent successfully! The round will be deleted shortly.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		if err != nil {
			err = fmt.Errorf("failed to send followup message: %w", err)
			drm.logger.ErrorContext(ctx, "Failed to send success followup message", attr.Error(err))
			return DeleteRoundOperationResult{Error: err}, nil
		}

		drm.logger.InfoContext(ctx, "Successfully processed delete round button interaction",
			attr.String("round_id", roundUUID.String()),
			attr.String("user_id", string(userID)))

		return DeleteRoundOperationResult{Success: "Delete request processed successfully"}, nil
	})
}

// sendDeleteRequest publishes the delete request to the backend.
func (drm *deleteRoundManager) sendDeleteRequest(ctx context.Context, roundID sharedtypes.RoundID, userID sharedtypes.DiscordID, interactionID string, discordMessageID string, guildID string) error {
	// Prepare the payload for the backend
	payload := discordroundevents.DiscordRoundDeleteRequestPayload{
		RoundID:   roundID,
		UserID:    userID,
		ChannelID: "", // Will be populated by the backend if needed
		MessageID: discordMessageID,
		GuildID:   guildID,
	}

	// Create the message struct with metadata including the discord_message_id - fix parameter order
	msg, err := drm.helper.CreateResultMessage(nil, payload, discordroundevents.RoundDeleteRequestTopic)
	if err != nil {
		return fmt.Errorf("failed to create result message: %w", err)
	}

	// Add metadata
	if msg.Metadata == nil {
		msg.Metadata = make(message.Metadata)
	}
	msg.Metadata.Set("interaction_id", interactionID)
	msg.Metadata.Set("discord_message_id", discordMessageID)
	msg.Metadata.Set("requesting_user_id", string(userID))

	// Publish the delete request message
	err = drm.publisher.Publish(discordroundevents.RoundDeleteRequestTopic, msg)
	if err != nil {
		return fmt.Errorf("failed to publish delete request: %w", err)
	}

	drm.logger.InfoContext(ctx, "Successfully published delete request",
		attr.String("round_id", roundID.String()),
		attr.String("user_id", string(userID)),
		attr.String("topic", discordroundevents.RoundDeleteRequestTopic))

	return nil
}

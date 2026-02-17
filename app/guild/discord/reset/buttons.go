package reset

import (
	"context"
	"fmt"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
)

// HandleResetConfirmButton handles the confirmation button click.
func (rm *resetManager) HandleResetConfirmButton(ctx context.Context, i *discordgo.InteractionCreate) error {
	return rm.operationWrapper(ctx, "HandleResetConfirmButton", func(ctx context.Context) error {
		ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "frolf-reset-confirm")
		ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "button")

		rm.logger.InfoContext(ctx, "Guild reset confirmed",
			attr.String("guild_id", i.GuildID),
			attr.String("user_id", getUserID(i)))

		correlationID := resetCorrelationIDFromCustomID(i.MessageComponentData().CustomID)
		if correlationID == "" {
			correlationID = newResetCorrelationID()
		}

		// Acknowledge the interaction with a deferred response
		err := rm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredMessageUpdate,
		})
		if err != nil {
			rm.logger.ErrorContext(ctx, "Failed to acknowledge interaction",
				attr.String("guild_id", i.GuildID),
				attr.Error(err))
			return fmt.Errorf("failed to acknowledge interaction: %w", err)
		}

		// Immediately update the message to show processing state
		processingContent := "⏳ Resetting server configuration...\n\nThis may take a few moments."
		_, err = rm.session.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content:    &processingContent,
			Components: &[]discordgo.MessageComponent{}, // Remove buttons
		})
		if err != nil {
			rm.logger.ErrorContext(ctx, "Failed to update message with processing state",
				attr.String("guild_id", i.GuildID),
				attr.Error(err))
			// Don't return error - continue with the operation even if UI update fails
		}

		// Store the interaction so watermill handlers can send the final response
		if rm.interactionStore != nil {
			err := rm.interactionStore.Set(ctx, correlationID, i.Interaction)
			if err != nil {
				rm.logger.ErrorContext(ctx, "Failed to store interaction",
					attr.String("guild_id", i.GuildID),
					attr.String("correlation_id", correlationID),
					attr.Error(err))
			}
		}

		// Publish the deletion request event
		// The watermill handlers will send the actual success/failure response
		err = rm.publishDeletionRequest(ctx, i.GuildID, correlationID)
		if err != nil {
			// Only handle publish errors here, not backend processing errors
			content := "❌ Failed to send reset request. Please try again or contact support."
			_, editErr := rm.session.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content:    &content,
				Components: &[]discordgo.MessageComponent{}, // Remove buttons to stop spinner
			})
			if editErr != nil {
				rm.logger.ErrorContext(ctx, "Failed to send error response", attr.Error(editErr))
			}
			return err
		}

		rm.logger.InfoContext(ctx, "Reset request published, waiting for backend response",
			attr.String("guild_id", i.GuildID))

		return nil
	})
}

// HandleResetCancelButton handles the cancel button click.
func (rm *resetManager) HandleResetCancelButton(ctx context.Context, i *discordgo.InteractionCreate) error {
	return rm.operationWrapper(ctx, "HandleResetCancelButton", func(ctx context.Context) error {
		ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "frolf-reset-cancel")
		ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "button")

		rm.logger.InfoContext(ctx, "Guild reset cancelled",
			attr.String("guild_id", i.GuildID),
			attr.String("user_id", getUserID(i)))

		// Update the original message to show cancellation
		err := rm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content:    "Reset cancelled. No changes were made.",
				Components: []discordgo.MessageComponent{}, // Remove buttons
				Flags:      discordgo.MessageFlagsEphemeral,
			},
		})

		if err != nil {
			rm.logger.ErrorContext(ctx, "Failed to update message",
				attr.String("guild_id", i.GuildID),
				attr.Error(err))
			return fmt.Errorf("failed to update message: %w", err)
		}

		return nil
	})
}

// publishDeletionRequest publishes the guild config deletion request event.
func (rm *resetManager) publishDeletionRequest(ctx context.Context, guildID, correlationID string) error {
	payload := guildevents.GuildConfigDeletionRequestedPayloadV1{
		GuildID: sharedtypes.GuildID(guildID),
	}

	// Create and publish the message using the helper
	msg, err := rm.helper.CreateNewMessage(payload, guildevents.GuildConfigDeletionRequestedV1)
	if err != nil {
		rm.logger.ErrorContext(ctx, "Failed to create deletion request message",
			attr.String("guild_id", guildID),
			attr.Error(err))
		return fmt.Errorf("failed to create deletion request message: %w", err)
	}

	msg.Metadata.Set("guild_id", guildID)
	if correlationID != "" {
		msg.Metadata.Set("correlation_id", correlationID)
	}

	err = rm.publisher.Publish(guildevents.GuildConfigDeletionRequestedV1, msg)
	if err != nil {
		rm.logger.ErrorContext(ctx, "Failed to publish deletion request",
			attr.String("guild_id", guildID),
			attr.String("topic", guildevents.GuildConfigDeletionRequestedV1),
			attr.Error(err))
		return fmt.Errorf("failed to publish deletion request: %w", err)
	}

	rm.logger.InfoContext(ctx, "Published guild config deletion request",
		attr.String("guild_id", guildID),
		attr.String("topic", guildevents.GuildConfigDeletionRequestedV1))

	return nil
}

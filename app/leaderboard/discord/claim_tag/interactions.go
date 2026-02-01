package claimtag

import (
	"context"
	"fmt"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/discordutils"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/utils"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

func (ctm *claimTagManager) HandleClaimTagCommand(ctx context.Context, i *discordgo.InteractionCreate) (ClaimTagOperationResult, error) {
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "claim_tag")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "command")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, i.Member.User.ID)

	ctm.logger.InfoContext(ctx, "Handling claim tag command",
		attr.UserID(sharedtypes.DiscordID(i.Member.User.ID)))

	// Publish user profile asynchronously
	go utils.PublishUserProfile(context.WithoutCancel(ctx), ctm.eventBus, ctm.logger, i.Member.User, i.Member, i.GuildID)

	// Fetch per-guild config using guildConfigResolver
	_, err := ctm.guildConfigResolver.GetGuildConfigWithContext(ctx, i.GuildID)
	if err != nil {
		ctm.logger.ErrorContext(ctx, "Failed to resolve guild config", attr.Error(err), attr.String("guild_id", i.GuildID))
		return ClaimTagOperationResult{Error: fmt.Errorf("failed to resolve guild config: %w", err)}, err
	}

	return ctm.operationWrapper(ctx, "handle_claim_tag_command", func(ctx context.Context) (ClaimTagOperationResult, error) {
		// Get tag number from command options
		options := i.ApplicationCommandData().Options
		if len(options) == 0 {
			return ClaimTagOperationResult{Error: fmt.Errorf("no tag number provided")}, nil
		}

		tagValue := options[0].IntValue()
		if tagValue < 1 || tagValue > 100 {
			return ClaimTagOperationResult{Error: fmt.Errorf("tag number must be between 1 and 100")}, nil
		}

		// **Send initial "thinking" response to Discord**
		err := ctm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			ctm.logger.ErrorContext(ctx, "Failed to send initial interaction response", attr.Error(err))
			return ClaimTagOperationResult{Error: err}, err
		}

		// Generate unique request ID
		requestID := uuid.New().String()

		// Store interaction for later response
		if ctm.interactionStore != nil {
			if err = ctm.interactionStore.Set(ctx, requestID, i.Interaction); err != nil {
				ctm.logger.ErrorContext(ctx, "Failed to store interaction", attr.Error(err))
				return ClaimTagOperationResult{Error: err}, err
			}
		} else {
			ctm.logger.WarnContext(ctx, "interaction store is nil; cannot persist interaction for callback")
		}

		// Create the batch payload (single-assignment batch) so backend receives the expected schema
		batchPayload := leaderboardevents.LeaderboardBatchTagAssignmentRequestedPayloadV1{
			GuildID:          sharedtypes.GuildID(i.GuildID),
			RequestingUserID: sharedtypes.DiscordID(i.Member.User.ID),
			BatchID:          requestID,
			Assignments: []leaderboardevents.TagAssignmentInfoV1{
				{
					GuildID:   sharedtypes.GuildID(i.GuildID),
					UserID:    sharedtypes.DiscordID(i.Member.User.ID),
					TagNumber: sharedtypes.TagNumber(tagValue),
				},
			},
		}

		// Create and publish the message using the batch topic (keep batch topic as requested)
		msg, err := ctm.helper.CreateNewMessage(batchPayload, leaderboardevents.LeaderboardBatchTagAssignmentRequestedV1)
		if err != nil {
			ctm.logger.ErrorContext(ctx, "Failed to create tag claim message", attr.Error(err))
			return ClaimTagOperationResult{Error: err}, err
		}

		// Add correlation ID to metadata, ensuring map is initialized
		if msg.Metadata == nil {
			msg.Metadata = message.Metadata{}
		}
		msg.Metadata.Set("correlation_id", requestID)

		// Publish the request
		err = ctm.eventBus.Publish(leaderboardevents.LeaderboardBatchTagAssignmentRequestedV1, msg)
		if err != nil {
			ctm.logger.ErrorContext(ctx, "Failed to publish tag claim request", attr.Error(err))
			return ClaimTagOperationResult{Error: err}, err
		}

		ctm.logger.InfoContext(ctx, "Successfully published tag claim request",
			attr.UserID(sharedtypes.DiscordID(i.Member.User.ID)),
			attr.Int("tag_number", int(tagValue)))

		return ClaimTagOperationResult{Success: "claim request sent"}, nil
	})
}

// UpdateInteractionResponse updates the interaction response using the stored interaction
func (ctm *claimTagManager) UpdateInteractionResponse(ctx context.Context, correlationID, message string) (ClaimTagOperationResult, error) {
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "update_interaction_response")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "followup")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CorrelationIDKey, correlationID)

	return ctm.operationWrapper(ctx, "update_interaction_response", func(ctx context.Context) (ClaimTagOperationResult, error) {
		// Retrieve stored interaction using the type-safe helper
		interactionObj, err := discordutils.GetInteraction(ctx, ctm.interactionStore, correlationID)
		if err != nil {
			ctm.logger.ErrorContext(ctx, "no interaction found for correlation ID", attr.String("correlation_id", correlationID), attr.Error(err))
			return ClaimTagOperationResult{Error: err}, nil
		}

		_, err = ctm.session.InteractionResponseEdit(interactionObj, &discordgo.WebhookEdit{
			Content: &message,
		})
		if err != nil {
			ctm.logger.ErrorContext(ctx, "Failed to update interaction response", attr.Error(err))
			return ClaimTagOperationResult{Error: err}, nil
		}

		// Clean up the stored interaction after successful response
		if ctm.interactionStore != nil {
			ctm.interactionStore.Delete(ctx, correlationID)
		}

		return ClaimTagOperationResult{Success: "interaction response updated"}, nil
	})
}

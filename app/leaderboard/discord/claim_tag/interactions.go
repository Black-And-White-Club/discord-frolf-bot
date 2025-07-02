package claimtag

import (
	"context"
	"fmt"
	"time"

	discordleaderboardevents "github.com/Black-And-White-Club/discord-frolf-bot/app/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

func (ctm *claimTagManager) HandleClaimTagCommand(ctx context.Context, i *discordgo.InteractionCreate) (ClaimTagOperationResult, error) {
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "claim_tag")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "command")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, i.Member.User.ID)

	ctm.logger.InfoContext(ctx, "Handling claim tag command",
		attr.UserID(sharedtypes.DiscordID(i.Member.User.ID)))

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
		err = ctm.interactionStore.Set(requestID, i.Interaction, 5*time.Minute)
		if err != nil {
			ctm.logger.ErrorContext(ctx, "Failed to store interaction", attr.Error(err))
			return ClaimTagOperationResult{Error: err}, err
		}

		// Create the payload for tag assignment request
		payload := discordleaderboardevents.LeaderboardTagAssignRequestPayload{
			TargetUserID: sharedtypes.DiscordID(i.Member.User.ID),
			RequestorID:  sharedtypes.DiscordID(i.Member.User.ID),
			TagNumber:    sharedtypes.TagNumber(tagValue),
			ChannelID:    i.ChannelID,
			MessageID:    requestID,
			GuildID:      i.GuildID,
		}

		// Create and publish the message
		msg, err := ctm.helper.CreateNewMessage(payload, discordleaderboardevents.LeaderboardTagAssignRequestTopic)
		if err != nil {
			ctm.logger.ErrorContext(ctx, "Failed to create tag claim message", attr.Error(err))
			return ClaimTagOperationResult{Error: err}, err
		}

		// Add correlation ID to metadata
		msg.Metadata.Set("correlation_id", requestID)

		// Publish the request
		err = ctm.eventBus.Publish(discordleaderboardevents.LeaderboardTagAssignRequestTopic, msg)
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
		interaction, found := ctm.interactionStore.Get(correlationID)
		if !found {
			err := fmt.Errorf("no interaction found for correlation ID: %s", correlationID)
			ctm.logger.ErrorContext(ctx, err.Error(), attr.String("correlation_id", correlationID))
			return ClaimTagOperationResult{Error: err}, nil
		}

		interactionObj, ok := interaction.(*discordgo.Interaction)
		if !ok {
			err := fmt.Errorf("stored interaction is not of type *discordgo.Interaction")
			ctm.logger.ErrorContext(ctx, err.Error(), attr.String("correlation_id", correlationID))
			return ClaimTagOperationResult{Error: err}, nil
		}

		_, err := ctm.session.InteractionResponseEdit(interactionObj, &discordgo.WebhookEdit{
			Content: &message,
		})
		if err != nil {
			ctm.logger.ErrorContext(ctx, "Failed to update interaction response", attr.Error(err))
			return ClaimTagOperationResult{Error: err}, nil
		}

		// Clean up the stored interaction after successful response
		ctm.interactionStore.Delete(correlationID)

		return ClaimTagOperationResult{Success: "interaction response updated"}, nil
	})
}

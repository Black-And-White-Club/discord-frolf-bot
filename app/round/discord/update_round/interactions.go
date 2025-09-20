package updateround

import (
	"context"
	"fmt"
	"strings"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

func (urm *updateRoundManager) HandleEditRoundButton(ctx context.Context, i *discordgo.InteractionCreate) (UpdateRoundOperationResult, error) {
	// Add safety check for Member/User
	var userID string
	if i.Member != nil && i.Member.User != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	} else {
		urm.logger.ErrorContext(ctx, "No user found in interaction")
		return UpdateRoundOperationResult{Error: fmt.Errorf("no user found in interaction")}, nil
	}

	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "handle_edit_round")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "button")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, userID)

	urm.logger.InfoContext(ctx, "Handling edit round button interaction - START",
		attr.String("interaction_id", i.ID),
		attr.String("custom_id", i.MessageComponentData().CustomID),
		attr.String("user_id", userID))

	return urm.operationWrapper(ctx, "handle_edit_round_button", func(ctx context.Context) (UpdateRoundOperationResult, error) {
		customID := i.MessageComponentData().CustomID
		urm.logger.InfoContext(ctx, "Parsing custom ID", attr.String("custom_id", customID))

		parts := strings.Split(customID, "|")
		if len(parts) != 2 { // Expecting: round_edit|<roundID>
			err := fmt.Errorf("invalid custom_id format: expected 'round_edit|<uuid>', got '%s'", customID)
			urm.logger.ErrorContext(ctx, err.Error())

			// Respond with error message to Discord
			if respErr := urm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Invalid button format. Please try again.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			}); respErr != nil {
				urm.logger.ErrorContext(ctx, "Failed to respond to interaction", attr.Error(respErr))
			}

			return UpdateRoundOperationResult{Error: err}, nil
		}

		// Parse UUID and extract guildID
		roundUUID, err := uuid.Parse(parts[1])
		if err != nil {
			err := fmt.Errorf("invalid UUID for round ID: %w", err)
			urm.logger.ErrorContext(ctx, err.Error())

			// Respond with error message to Discord
			if respErr := urm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Invalid round ID. Please try again.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			}); respErr != nil {
				urm.logger.ErrorContext(ctx, "Failed to respond to interaction", attr.Error(respErr))
			}

			return UpdateRoundOperationResult{Error: err}, nil
		}
		roundID := sharedtypes.RoundID(roundUUID)
		guildID := i.GuildID // Get guild ID from interaction context

		urm.logger.InfoContext(ctx, "Opening modal for round edit", attr.RoundID("round_id", roundID), attr.String("guild_id", guildID))

		// Send the update round modal with roundID and guildID
		result, err := urm.SendUpdateRoundModal(ctx, i, roundID)
		urm.logger.InfoContext(ctx, "Modal send result",
			attr.Any("result", result),
			attr.Error(err))

		if err != nil {
			urm.logger.ErrorContext(ctx, "Failed to send modal", attr.Error(err))

			// If modal failed to send, respond with error message
			if respErr := urm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Failed to open edit modal. Please try again.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			}); respErr != nil {
				urm.logger.ErrorContext(ctx, "Failed to respond with error", attr.Error(respErr))
			}

			return UpdateRoundOperationResult{Error: err}, err
		}

		if result.Error != nil {
			urm.logger.ErrorContext(ctx, "Modal send returned error", attr.Error(result.Error))
			return result, nil
		}

		return UpdateRoundOperationResult{Success: "modal sent"}, nil
	})
}

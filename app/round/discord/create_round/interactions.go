package createround

import (
	"context"
	"fmt"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
)

func (crm *createRoundManager) HandleCreateRoundCommand(ctx context.Context, i *discordgo.InteractionCreate) (CreateRoundOperationResult, error) {
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "create_round_command")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "command")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, i.Member.User.ID)

	crm.logger.InfoContext(ctx, "Handling create round command", attr.UserID(sharedtypes.DiscordID(i.Member.User.ID)))

	return crm.operationWrapper(ctx, "handle_create_round_command", func(ctx context.Context) (CreateRoundOperationResult, error) {
		result, err := crm.SendCreateRoundModal(ctx, i)
		if err != nil {
			crm.logger.ErrorContext(ctx, "Failed to send create round modal", attr.Error(err), attr.UserID(sharedtypes.DiscordID(i.Member.User.ID)))
			return CreateRoundOperationResult{Error: err}, err
		}

		if result.Error != nil {
			crm.logger.ErrorContext(ctx, "Error in SendCreateRoundModal operation", attr.Error(result.Error), attr.UserID(sharedtypes.DiscordID(i.Member.User.ID)))
			return result, nil
		}

		return CreateRoundOperationResult{Success: "modal sent"}, nil
	})
}

func (crm *createRoundManager) HandleRetryCreateRound(ctx context.Context, i *discordgo.InteractionCreate) (CreateRoundOperationResult, error) {
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "retry_create_round")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "button")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, i.Member.User.ID)

	crm.logger.InfoContext(ctx, "Handling retry create round", attr.UserID(sharedtypes.DiscordID(i.Member.User.ID)))

	return crm.operationWrapper(ctx, "handle_retry_create_round", func(ctx context.Context) (CreateRoundOperationResult, error) {
		result, err := crm.SendCreateRoundModal(ctx, i)
		if err != nil {
			crm.logger.ErrorContext(ctx, "Critical error in SendCreateRoundModal", attr.Error(err))
			return CreateRoundOperationResult{Error: err}, err
		}

		if result.Error != nil {
			crm.logger.ErrorContext(ctx, "Operation error in SendCreateRoundModal", attr.Error(result.Error))

			msg := "Failed to open the form. Please try using the /createround command again."
			_, updateErr := crm.session.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content:    &msg,
				Components: &[]discordgo.MessageComponent{},
			})

			if updateErr != nil {
				crm.logger.ErrorContext(ctx, "Failed to update error message", attr.Error(updateErr))
			}

			return result, nil
		}

		return CreateRoundOperationResult{Success: "retry modal sent"}, nil
	})
}

func (crm *createRoundManager) UpdateInteractionResponse(ctx context.Context, correlationID, message string, edit ...*discordgo.WebhookEdit) (CreateRoundOperationResult, error) {
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "update_interaction_response")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "followup")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CorrelationIDKey, correlationID)

	return crm.operationWrapper(ctx, "update_interaction_response", func(ctx context.Context) (CreateRoundOperationResult, error) {
		if err := ctx.Err(); err != nil {
			return CreateRoundOperationResult{Error: err}, err
		}
		interaction, found := crm.interactionStore.Get(correlationID)
		if !found {
			err := fmt.Errorf("no interaction found for correlation ID: %s", correlationID)
			crm.logger.ErrorContext(ctx, err.Error(), attr.String("correlation_id", correlationID))
			return CreateRoundOperationResult{Error: err}, nil
		}

		interactionObj, ok := interaction.(*discordgo.Interaction)
		if !ok {
			err := fmt.Errorf("stored interaction is not of type *discordgo.Interaction")
			crm.logger.ErrorContext(ctx, err.Error(), attr.String("correlation_id", correlationID))
			return CreateRoundOperationResult{Error: err}, nil
		}

		var webhookEdit *discordgo.WebhookEdit
		if len(edit) > 0 {
			webhookEdit = edit[0]
			webhookEdit.Content = &message
		} else {
			webhookEdit = &discordgo.WebhookEdit{Content: &message}
		}

		_, err := crm.session.InteractionResponseEdit(interactionObj, webhookEdit)
		if err != nil {
			crm.logger.ErrorContext(ctx, "Failed to update interaction response", attr.Error(err))
			return CreateRoundOperationResult{Error: err}, nil
		}

		return CreateRoundOperationResult{Success: "interaction response updated"}, nil
	})
}

func (crm *createRoundManager) UpdateInteractionResponseWithRetryButton(ctx context.Context, correlationID, message string) (CreateRoundOperationResult, error) {
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "update_interaction_retry_button")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "followup")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CorrelationIDKey, correlationID)

	return crm.operationWrapper(ctx, "update_response_with_retry", func(ctx context.Context) (CreateRoundOperationResult, error) {
		if err := ctx.Err(); err != nil {
			return CreateRoundOperationResult{Error: err}, err
		}
		interaction, found := crm.interactionStore.Get(correlationID)
		if !found {
			err := fmt.Errorf("no interaction found for correlation ID: %s", correlationID)
			crm.logger.ErrorContext(ctx, err.Error(), attr.String("correlation_id", correlationID))
			return CreateRoundOperationResult{Error: err}, nil
		}

		interactionObj, ok := interaction.(*discordgo.Interaction)
		if !ok {
			err := fmt.Errorf("stored interaction is not of type *discordgo.Interaction")
			crm.logger.ErrorContext(ctx, err.Error(), attr.String("correlation_id", correlationID))
			return CreateRoundOperationResult{Error: err}, nil
		}

		_, err := crm.session.InteractionResponseEdit(interactionObj, &discordgo.WebhookEdit{
			Content: &message,
			Components: &[]discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Try Again",
						Style:    discordgo.PrimaryButton,
						CustomID: "retry_create_round",
					},
				}},
			},
		})
		if err != nil {
			crm.logger.ErrorContext(ctx, "Failed to update interaction response with retry button", attr.Error(err))
			return CreateRoundOperationResult{Error: err}, nil
		}

		return CreateRoundOperationResult{Success: "response updated with retry"}, nil
	})
}

package createround

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

func (crm *createRoundManager) HandleCreateRoundCommand(ctx context.Context, i *discordgo.InteractionCreate) {
	slog.Info("Handling create round button press", attr.UserID(i.Member.User.ID))
	err := crm.SendCreateRoundModal(ctx, i)
	if err != nil {
		slog.Error("Failed to send create round modal", attr.UserID(i.Member.User.ID), attr.Error(err))
	}
}

// UpdateInteractionResponse updates the deferred response using the correlation ID.
func (crm *createRoundManager) UpdateInteractionResponse(ctx context.Context, correlationID, message string, edit ...*discordgo.WebhookEdit) error {
	interaction, found := crm.interactionStore.Get(correlationID)
	if !found {
		return fmt.Errorf("no interaction found for correlation ID: %s", correlationID)
	}

	// Make sure interaction is of the correct type
	interactionObj, ok := interaction.(*discordgo.Interaction)
	if !ok {
		return fmt.Errorf("stored interaction is not of type *discordgo.Interaction")
	}

	// Use the full interaction object
	var webhookEdit *discordgo.WebhookEdit
	if len(edit) > 0 {
		webhookEdit = edit[0]
		// Ensure content is set
		webhookEdit.Content = &message
	} else {
		webhookEdit = &discordgo.WebhookEdit{
			Content: &message,
		}
	}

	_, err := crm.session.InteractionResponseEdit(interactionObj, webhookEdit)
	if err != nil {
		slog.Error("Failed to update interaction response", attr.Error(err))
	}
	return err
}

func (crm *createRoundManager) UpdateInteractionResponseWithRetryButton(ctx context.Context, correlationID, message string) error {
	interaction, found := crm.interactionStore.Get(correlationID)
	if !found {
		return fmt.Errorf("no interaction found for correlation ID: %s", correlationID)
	}

	// Make sure interaction is of the correct type
	interactionObj, ok := interaction.(*discordgo.Interaction)
	if !ok {
		return fmt.Errorf("stored interaction is not of type *discordgo.Interaction")
	}

	// Attempt to update the interaction response
	_, err := crm.session.InteractionResponseEdit(interactionObj, &discordgo.WebhookEdit{
		Content:    &message,
		Components: &[]discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.Button{Label: "Try Again", Style: discordgo.PrimaryButton, CustomID: "retry_create_round"}}}},
	})
	if err != nil {
		slog.Error("Failed to update interaction response with retry button", attr.Error(err))
		return err
	}
	return nil
}

func (crm *createRoundManager) HandleRetryCreateRound(ctx context.Context, i *discordgo.InteractionCreate) {
	slog.Info("Handling retry create round button press", attr.UserID(i.Member.User.ID))

	err := crm.SendCreateRoundModal(ctx, i)
	if err != nil {
		slog.Error("Failed to send create round modal", attr.UserID(i.Member.User.ID), attr.Error(err))

		// If modal sending fails, update the message to inform the user
		_, updateErr := crm.session.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content:    stringPtr("Failed to open the form. Please try using the /createround command again."),
			Components: &[]discordgo.MessageComponent{},
		})
		if updateErr != nil {
			slog.Error("Failed to update error message", attr.Error(updateErr))
		}
	}
}

func stringPtr(s string) *string {
	return &s
}

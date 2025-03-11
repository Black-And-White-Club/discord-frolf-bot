package signup

import (
	"context"
	"fmt"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

func (sm *signupManager) SendSignupResult(correlationID string, success bool) error {
	interaction, found := sm.interactionStore.Get(correlationID)
	if !found {
		sm.logger.Error(context.Background(), "Failed to get interaction from store", attr.String("correlation_id", correlationID))
		return fmt.Errorf("interaction not found for correlation ID: %s", correlationID)
	}
	interactionObj, ok := interaction.(*discordgo.Interaction)
	if !ok {
		sm.logger.Error(context.Background(), "Stored interaction is not of type *discordgo.Interaction", attr.String("correlation_id", correlationID))
		return fmt.Errorf("interaction is not of the expected type")
	}
	content := "❌ Signup failed. Please try again."
	if success {
		content = "🎉 Signup successful! Welcome!"
	}
	_, err := sm.session.InteractionResponseEdit(interactionObj, &discordgo.WebhookEdit{
		Content: stringPtr(content),
	})
	if err != nil {
		sm.logger.Error(context.Background(), "Failed to send result", attr.Error(err))
		return fmt.Errorf("failed to send result: %w", err)
	}
	return nil
}

func stringPtr(s string) *string {
	return &s
}

package signup

import (
	"context"
	"fmt"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/bwmarrin/discordgo"
)

// SendSignupResult sends the signup result back to the user.
// failureReason is optional and only used when success is false.
func (sm *signupManager) SendSignupResult(ctx context.Context, correlationID string, success bool, failureReason ...string) (SignupOperationResult, error) {
	// Enrich context for observability
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CorrelationIDKey, correlationID)
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "send_signup_result")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "followup")

	// Wrap the entire logic in the operationWrapper and return its results directly
	return sm.operationWrapper(ctx, "send_signup_result", func(ctx context.Context) (SignupOperationResult, error) {
		sm.logger.InfoContext(ctx, "Processing signup result", attr.String("correlation_id", correlationID))

		interaction, found := sm.interactionStore.Get(correlationID)
		if !found {
			err := fmt.Errorf("interaction not found for correlation ID: %s", correlationID)
			sm.logger.ErrorContext(ctx, err.Error(), attr.String("correlation_id", correlationID))
			return SignupOperationResult{Error: err}, nil
		}

		interactionObj, ok := interaction.(*discordgo.Interaction)
		if !ok {
			err := fmt.Errorf("interaction is not of the expected type")
			sm.logger.ErrorContext(ctx, err.Error(), attr.String("correlation_id", correlationID))
			return SignupOperationResult{Error: err}, nil
		}

		// Add nil check for interactionObj before accessing its fields
		if interactionObj == nil {
			err := fmt.Errorf("retrieved interaction object is nil for correlation ID: %s", correlationID)
			sm.logger.ErrorContext(ctx, err.Error(), attr.String("correlation_id", correlationID))
			return SignupOperationResult{Error: err}, nil
		}

		content := "❌ Signup failed. Please try again."
		if success {
			content = "🎉 Signup successful! Welcome!"
		} else if len(failureReason) > 0 && failureReason[0] != "" {
			// Use the specific failure reason if provided
			content = fmt.Sprintf("❌ Signup failed: %s", failureReason[0])
		}

		_, err := sm.session.InteractionResponseEdit(interactionObj, &discordgo.WebhookEdit{
			Content: stringPtr(content),
		})
		if err != nil {
			sm.logger.ErrorContext(ctx, "Failed to send result", attr.Error(err))
			return SignupOperationResult{Error: fmt.Errorf("failed to send result: %w", err)}, err
		}

		sm.logger.InfoContext(ctx, "Successfully sent signup result response")
		return SignupOperationResult{Success: content}, nil
	})
}

func stringPtr(s string) *string {
	return &s
}

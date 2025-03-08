package discord

import (
	"context"
	"fmt"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

func (d *discordOperations) SendSignupModal(ctx context.Context, i *discordgo.Interaction) error {
	d.logger.Info(ctx, "Preparing to send signup modal", attr.UserID(i.User.ID))
	err := d.session.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "signup_modal",
			Title:    "Frolf Club Signup",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "tag_number",
							Label:       "Tag Number (Optional)",
							Style:       discordgo.TextInputShort,
							Placeholder: "Enter your desired tag number (e.g., 13)",
							Required:    false,
							MaxLength:   3,
							MinLength:   0,
							Value:       "",
						},
					},
				},
			},
		},
	})
	if err != nil {
		d.logger.Error(ctx, "Failed to send signup modal", attr.UserID(i.User.ID), attr.Error(err))
		return fmt.Errorf("failed to send signup modal: %w", err)
	}
	d.logger.Info(ctx, "Signup modal successfully sent!", attr.UserID(i.User.ID))
	return nil
}

func (d *discordOperations) SendSignupAcknowledgment(interactionToken string) {
	d.session.FollowupMessageCreate(&discordgo.Interaction{Token: interactionToken}, true, &discordgo.WebhookParams{
		Content: "✅ Submitted signup request. Processing...",
	})
}

func (d *discordOperations) SendSignupResult(interactionToken string, success bool) {
	content := "❌ Signup failed. Please try again."
	if success {
		content = "🎉 Signup successful! Welcome!"
	}
	d.session.FollowupMessageCreate(&discordgo.Interaction{Token: interactionToken}, true, &discordgo.WebhookParams{
		Content: content,
	})
}

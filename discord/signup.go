package discord

import (
	"context"
	"fmt"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

// SendEphemeralSignupModal sends an ephemeral message with a signup button.
func (d *discordOperations) SendEphemeralSignupModal(ctx context.Context, userID, guildID string, i *discordgo.Interaction) error {
	d.logger.Info(ctx, "Sending ephemeral signup modal", attr.UserID(userID))
	err := d.session.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource, // Show message
		Data: &discordgo.InteractionResponseData{
			CustomID: "signup_modal",
			Title:    "Signup",
			Content:  "Click the button below to signup",
			Flags:    discordgo.MessageFlagsEphemeral, // Only visible to the user
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Signup",
							Style:    discordgo.PrimaryButton,
							CustomID: "signup_button",
						},
					},
				},
			},
		},
	})
	if err != nil {
		d.logger.Error(ctx, "failed to send ephemeral message: %w", attr.UserID(userID), attr.Error(err))
		return fmt.Errorf("failed to send ephemeral message: %w", err)
	}
	return nil
}

func (d *discordOperations) SendSignupModal(ctx context.Context, i *discordgo.Interaction) error {
	d.logger.Info(ctx, "Sending signup modal", attr.UserID(i.User.ID))

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
	return nil
}

package discord

import (
	"context"
	"fmt"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

func (d *discordOperations) SendCreateRoundModal(ctx context.Context, i *discordgo.Interaction) error {
	d.logger.Info(ctx, "Sending create round modal", attr.UserID(i.Member.User.ID))
	// Send the modal as the initial response
	err := d.session.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			Title:    "Create Round",
			CustomID: "create_round_modal",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "title",
							Label:       "Title",
							Style:       discordgo.TextInputShort,
							Placeholder: "Enter the round title",
							Required:    true,
							MaxLength:   100,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "description",
							Label:       "Description",
							Style:       discordgo.TextInputParagraph,
							Placeholder: "Enter a description (optional)",
							Required:    false,
							MaxLength:   500,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "start_time",
							Label:       "Start Time",
							Style:       discordgo.TextInputShort,
							Placeholder: "YYYY-MM-DD HH:MM",
							Required:    true,
							MaxLength:   30,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "timezone",
							Label:       "Timezone (Optional)",
							Style:       discordgo.TextInputShort,
							Placeholder: "America/Chicago (CST)",
							Required:    false,
							MaxLength:   50,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "location",
							Label:       "Location",
							Style:       discordgo.TextInputShort,
							Placeholder: "Enter the location (optional)",
							Required:    false,
							MaxLength:   100,
						},
					},
				},
			},
		},
	})
	if err != nil {
		d.logger.Error(ctx, "Failed to send create round modal", attr.UserID(i.Member.User.ID), attr.Error(err))
		return fmt.Errorf("failed to send create round modal: %w", err)
	}
	return nil
}

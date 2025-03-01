package discord

import (
	"context"
	"fmt"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

// SendCreateRoundModal sends a modal for creating a new round in the user's DM.
func (d *discordOperations) SendCreateRoundModal(ctx context.Context, i *discordgo.Interaction) error {
	d.logger.Info(ctx, "Sending create round modal", attr.UserID(i.Member.User.ID))

	// Step 1: Open a DM channel with the user
	dmChannel, err := d.session.UserChannelCreate(i.Member.User.ID)
	if err != nil {
		d.logger.Error(ctx, "Failed to create DM channel", attr.UserID(i.Member.User.ID), attr.Error(err))
		return fmt.Errorf("failed to create DM channel: %w", err)
	}

	// Step 2: Send a message in the DM to let them know the modal is opening
	_, err = d.session.ChannelMessageSend(dmChannel.ID, "Opening round creation modal...")
	if err != nil {
		d.logger.Error(ctx, "Failed to send DM message", attr.UserID(i.Member.User.ID), attr.Error(err))
		return fmt.Errorf("failed to send DM message: %w", err)
	}

	// Step 3: Respond to the interaction with the modal
	err = d.session.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "create_round_modal",
			Title:    "Create a New Round",
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
							MaxLength:   16,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "end_time",
							Label:       "End Time",
							Style:       discordgo.TextInputShort,
							Placeholder: "YYYY-MM-DD HH:MM",
							Required:    true,
							MaxLength:   16,
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

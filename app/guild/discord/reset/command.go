package reset

import (
	"context"
	"fmt"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/bwmarrin/discordgo"
)

// HandleResetCommand handles the /frolf-reset slash command.
// Shows a confirmation dialog before proceeding with guild config reset.
func (rm *resetManager) HandleResetCommand(ctx context.Context, i *discordgo.InteractionCreate) error {
	return rm.operationWrapper(ctx, "HandleResetCommand", func(ctx context.Context) error {
		ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "frolf-reset")
		ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "command")

		rm.logger.InfoContext(ctx, "Guild reset command received",
			attr.String("guild_id", i.GuildID),
			attr.String("user_id", getUserID(i)))

		// Validate that we have guild context
		if i.GuildID == "" {
			return rm.respondWithError(i, "This command can only be used in a server.")
		}

		// Respond with confirmation dialog
		return rm.showConfirmationDialog(ctx, i, newResetCorrelationID())
	})
}

// showConfirmationDialog presents a confirmation dialog with buttons.
func (rm *resetManager) showConfirmationDialog(ctx context.Context, i *discordgo.InteractionCreate, correlationID string) error {
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "⚠️ Yes, Reset Server Data",
					Style:    discordgo.DangerButton,
					CustomID: resetConfirmCustomID(correlationID),
				},
				discordgo.Button{
					Label:    "Cancel",
					Style:    discordgo.SecondaryButton,
					CustomID: resetCancelCustomID(correlationID),
				},
			},
		},
	}

	err := rm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "## ⚠️ Reset Server Configuration\n\n" +
				"**This will:**\n" +
				"• Deactivate your server's Frolf Bot configuration\n" +
				"• Unregister all bot commands from your server\n" +
				"• Require running `/frolf-setup` again to use the bot\n\n" +
				"**This will NOT delete:**\n" +
				"• Historical round data\n" +
				"• User profiles and scores\n" +
				"• Leaderboard history\n\n" +
				"*You can re-setup the bot at any time by running `/frolf-setup` again.*\n\n" +
				"**Are you sure you want to reset?**",
			Components: components,
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})

	if err != nil {
		rm.logger.ErrorContext(ctx, "Failed to send confirmation dialog",
			attr.String("guild_id", i.GuildID),
			attr.Error(err))
		return fmt.Errorf("failed to send confirmation dialog: %w", err)
	}

	return nil
}

// respondWithError sends an ephemeral error message to the user.
func (rm *resetManager) respondWithError(i *discordgo.InteractionCreate, message string) error {
	return rm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("❌ %s", message),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// getUserID safely extracts the user ID from the interaction.
func getUserID(i *discordgo.InteractionCreate) string {
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User.ID
	}
	if i.User != nil {
		return i.User.ID
	}
	return ""
}

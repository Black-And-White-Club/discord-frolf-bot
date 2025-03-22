package signup

import (
	"context"
	"fmt"
	"log/slog"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

// messageReactionAdd handles MessageReactionAdd events.
func (sm *signupManager) MessageReactionAdd(s discord.Session, r *discordgo.MessageReactionAdd) {
	slog.Info("signupManager.MessageReactionAdd called", attr.UserID(r.UserID))
	signupChannelID := sm.config.Discord.SignupChannelID
	signupMessageID := sm.config.Discord.SignupMessageID
	signupEmoji := sm.config.Discord.SignupEmoji
	if r.ChannelID != signupChannelID || r.MessageID != signupMessageID || r.Emoji.Name != signupEmoji {
		slog.Info("Reaction mismatch",
			attr.UserID(r.UserID),
			attr.String("channel_id", r.ChannelID),
			attr.String("message_id", r.MessageID),
			attr.Any("emoji", r.Emoji.Name))
		return
	}
	slog.Info("Valid reaction detected, processing signup.")
	botUser, err := sm.session.GetBotUser()
	if err != nil {
		slog.Error("Failed to get bot user", attr.Error(err))
		return
	}
	if r.UserID == botUser.ID {
		slog.Info("Ignoring bot's own reaction.")
		return
	}
	slog.Info("Publishing signup reaction event...")
	sm.HandleSignupReactionAdd(context.Background(), r)
}

// handleSignupReactionAdd sends the signup modal.
func (sm *signupManager) HandleSignupReactionAdd(ctx context.Context, r *discordgo.MessageReactionAdd) {
	slog.Info("Handling signup reaction", attr.UserID(r.UserID))

	// Verify the reaction happened in the correct guild
	if r.GuildID != sm.config.Discord.GuildID {
		slog.Warn("Reaction from wrong guild", attr.UserID(r.UserID), attr.String("guildID", r.GuildID))
		return
	}

	dmChannel, err := sm.session.UserChannelCreate(r.UserID)
	if err != nil {
		slog.Error("Failed to create DM channel", attr.UserID(r.UserID), attr.Error(err))
		return
	}
	slog.Info("DM channel created", attr.String("dm_channel_id", dmChannel.ID))

	metadataStr := fmt.Sprintf("signup_button|%s", r.UserID)

	_, err = sm.session.ChannelMessageSendComplex(dmChannel.ID, &discordgo.MessageSend{
		Content: "Click the button below to start your signup!",
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Signup",
						Style:    discordgo.PrimaryButton,
						CustomID: metadataStr,
					},
				},
			},
		},
	})
	if err != nil {
		slog.Error("Failed to send signup button in DM", attr.UserID(r.UserID), attr.Error(err))
	} else {
		slog.Info("✅ Signup button successfully sent in DM!", attr.UserID(r.UserID))
	}
}

// New handler for the button press
func (sm *signupManager) HandleSignupButtonPress(ctx context.Context, i *discordgo.InteractionCreate) {
	// // Explicit type assertion
	// data, ok := i.Interaction.Data.(*discordgo.MessageComponentInteractionData)
	// if !ok {
	// 	slog.Error("❌ Failed to cast Interaction.Data to MessageComponentInteractionData")
	// 	return
	// }

	// slog.Info("Inside HandleSignupButtonPress!",
	// 	attr.String("custom_id", data.CustomID),
	// )

	err := sm.SendSignupModal(ctx, i)
	if err != nil {
		slog.Error("❌ Failed to send signup modal", attr.Error(err))
		return
	}

	slog.Info("✅ Successfully called SendSignupModal")
}

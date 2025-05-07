package signup

import (
	"context"
	"fmt"
	"log/slog"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/bwmarrin/discordgo"
)

// MessageReactionAdd is the top-level Watermill/Discord event hook
// Updated to use the wrapper and return SignupOperationResult, error
func (sm *signupManager) MessageReactionAdd(s discord.Session, r *discordgo.MessageReactionAdd) (SignupOperationResult, error) {
	ctx := context.Background() // Start with a background context for top-level event
	ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, r.UserID)
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "reaction_signup")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "reaction")

	ctx = discordmetrics.WithValue(ctx, discordmetrics.GuildIDKey, r.GuildID) // Add GuildID if available

	// Wrap the entire logic in the operationWrapper
	return sm.operationWrapper(ctx, "message_reaction_add", func(ctx context.Context) (SignupOperationResult, error) {
		sm.logger.InfoContext(ctx, "signupManager.MessageReactionAdd called")

		signupChannelID := sm.config.Discord.SignupChannelID
		signupMessageID := sm.config.Discord.SignupMessageID
		signupEmoji := sm.config.Discord.SignupEmoji

		// Check if the reaction matches the configured signup message and emoji
		if r.ChannelID != signupChannelID || r.MessageID != signupMessageID || r.Emoji.Name != signupEmoji {
			sm.logger.InfoContext(ctx, "Reaction mismatch - ignoring",
				attr.String("channel_id", r.ChannelID),
				attr.String("message_id", r.MessageID),
				attr.Any("emoji", r.Emoji.Name),
				attr.String("expected_channel", signupChannelID),
				attr.String("expected_message", signupMessageID),
				attr.String("expected_emoji", signupEmoji),
			)
			// Return success, indicating the reaction was received but not processed as a signup reaction
			return SignupOperationResult{Success: "reaction mismatch, ignored"}, nil
		}

		sm.logger.InfoContext(ctx, "Valid signup reaction detected, processing...")

		// Get bot user to ignore its own reactions
		botUser, err := sm.session.GetBotUser()
		if err != nil {
			sm.logger.ErrorContext(ctx, "Failed to get bot user", attr.Error(err))
			// Return SignupOperationResult with an error and nil outer error
			return SignupOperationResult{Error: fmt.Errorf("failed to get bot user: %w", err)}, nil
		}
		if r.UserID == botUser.ID {
			sm.logger.InfoContext(ctx, "Ignoring bot's own reaction")
			// Return success, indicating bot's reaction was ignored
			return SignupOperationResult{Success: "ignored bot reaction"}, nil
		}

		sm.logger.InfoContext(ctx, "Publishing signup reaction event...")
		// Delegate to HandleSignupReactionAdd and return its result
		// Note: HandleSignupReactionAdd is also wrapped, so this call will trigger another wrapped operation.
		// This is acceptable, but be mindful of nested spans/logs.
		return sm.HandleSignupReactionAdd(ctx, r)
	})
}

func (sm *signupManager) HandleSignupReactionAdd(ctx context.Context, r *discordgo.MessageReactionAdd) (SignupOperationResult, error) {
	ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, r.UserID)
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "handle_signup_reaction")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "reaction")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.GuildIDKey, r.GuildID)

	// 🔥 Check if context is already cancelled — prevents wasted work
	if ctx.Err() != nil {
		return SignupOperationResult{Error: ctx.Err()}, ctx.Err()
	}

	result, err := sm.operationWrapper(ctx, "handle_signup_reaction", func(ctx context.Context) (SignupOperationResult, error) {
		sm.logger.InfoContext(ctx, "Handling signup reaction")

		if r.GuildID != sm.config.Discord.GuildID {
			sm.logger.WarnContext(ctx, "Reaction from wrong guild", attr.String("guildID", r.GuildID))
			return SignupOperationResult{Error: fmt.Errorf("reaction from unauthorized guild")}, nil
		}

		dmChannel, err := sm.session.UserChannelCreate(r.UserID)
		if err != nil {
			sm.logger.ErrorContext(ctx, "Failed to create DM channel", attr.Error(err))
			return SignupOperationResult{Error: fmt.Errorf("failed to create DM channel: %w", err)}, err
		}
		sm.logger.InfoContext(ctx, "DM channel created", attr.String("dm_channel_id", dmChannel.ID))

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
			sm.logger.ErrorContext(ctx, "Failed to send signup button in DM", attr.Error(err))
			return SignupOperationResult{Error: fmt.Errorf("failed to send signup button in DM: %w", err)}, err
		}

		sm.logger.InfoContext(ctx, "Signup button successfully sent in DM")
		return SignupOperationResult{Success: "signup button sent"}, nil
	})

	return result, err
}

// New handler for the button press
func (sm *signupManager) HandleSignupButtonPress(ctx context.Context, i *discordgo.InteractionCreate) (SignupOperationResult, error) {
	result, err := sm.SendSignupModal(ctx, i)
	if err != nil {
		slog.Error("❌ Failed to send signup modal", attr.Error(err))
	} else {
		slog.Info("✅ Successfully called SendSignupModal")
	}
	return result, err
}

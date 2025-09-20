package signup

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/bwmarrin/discordgo"
)

// normalizeEmoji removes variation selectors and other Unicode modifiers
// to ensure consistent emoji comparison between different representations
func normalizeEmoji(emoji string) string {
	var result strings.Builder
	for _, r := range emoji {
		// Skip Unicode variation selectors (U+FE00-U+FE0F)
		if r >= 0xFE00 && r <= 0xFE0F {
			continue
		}
		// Skip other common Unicode modifiers that can cause mismatches
		if unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Me, r) {
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}

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

		// Determine expected channel and emoji using resolver if guild id exists; otherwise fall back to static config
		var signupChannelID, signupEmoji string
		if r.GuildID != "" && sm.guildConfigResolver != nil {
			cfg, err := sm.guildConfigResolver.GetGuildConfigWithContext(ctx, r.GuildID)
			if err != nil {
				sm.logger.WarnContext(ctx, "Failed to get guild config - guild may not be set up",
					attr.String("guild_id", r.GuildID),
					attr.Error(err),
				)
				return SignupOperationResult{Error: fmt.Errorf("guild not configured - please run /frolf-setup first")}, nil
			}
			if cfg == nil || cfg.SignupChannelID == "" {
				sm.logger.WarnContext(ctx, "Guild config exists but has no signup channel - guild setup incomplete",
					attr.String("guild_id", r.GuildID),
					attr.Bool("cfg_nil", cfg == nil),
				)
				return SignupOperationResult{Error: fmt.Errorf("guild setup incomplete - please run /frolf-setup")}, nil
			}
			signupChannelID = cfg.SignupChannelID
			signupEmoji = cfg.SignupEmoji
		} else {
			// Use static config for checks when guildID not present in the event (tests cover this)
			if sm.config != nil {
				signupChannelID = sm.config.Discord.SignupChannelID
				signupEmoji = sm.config.Discord.SignupEmoji
			}
		}
		if signupEmoji == "" {
			signupEmoji = "🥏"
		}

		sm.logger.InfoContext(ctx, "Signup config resolved from backend",
			attr.String("guild_id", r.GuildID),
			attr.String("signup_channel_id", signupChannelID),
			attr.String("signup_emoji", signupEmoji),
		)

		// Check if the reaction matches the configured signup channel and emoji
		if r.ChannelID != signupChannelID {
			sm.logger.InfoContext(ctx, "Reaction channel mismatch - ignoring",
				attr.String("channel_id", r.ChannelID),
				attr.String("expected_channel", signupChannelID),
			)
			return SignupOperationResult{Success: "reaction mismatch, ignored"}, nil
		}

		// Fast path: compare raw emoji first; normalize only on mismatch
		if r.Emoji.Name != signupEmoji {
			normalizedReactionEmoji := normalizeEmoji(r.Emoji.Name)
			normalizedExpectedEmoji := normalizeEmoji(signupEmoji)
			if normalizedReactionEmoji != normalizedExpectedEmoji {
				sm.logger.InfoContext(ctx, "Reaction emoji mismatch - ignoring",
					attr.Any("emoji", r.Emoji.Name),
					attr.String("normalized_emoji", normalizedReactionEmoji),
					attr.String("expected_emoji", signupEmoji),
					attr.String("normalized_expected_emoji", normalizedExpectedEmoji),
				)
				return SignupOperationResult{Success: "reaction mismatch, ignored"}, nil
			}
		}

		sm.logger.InfoContext(ctx, "Valid signup reaction detected, processing...")

		// Get bot user to ignore its own reactions (only after we've matched channel+emoji)
		botUser, err := sm.session.GetBotUser()
		if err != nil {
			sm.logger.ErrorContext(ctx, "Failed to get bot user", attr.Error(err))
			return SignupOperationResult{Error: fmt.Errorf("failed to get bot user: %w", err)}, nil
		}
		if botUser != nil && r.UserID == botUser.ID {
			sm.logger.InfoContext(ctx, "Ignoring bot's own reaction")
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

		// Validate guild authorization
		if r.GuildID == "" {
			return SignupOperationResult{Error: fmt.Errorf("reaction from unauthorized guild")}, nil
		}
		if sm.guildConfigResolver != nil {
			if cfg, err := sm.guildConfigResolver.GetGuildConfigWithContext(ctx, r.GuildID); err != nil || cfg == nil {
				return SignupOperationResult{Error: fmt.Errorf("reaction from unauthorized guild")}, nil
			}
		} else if sm.config != nil && sm.config.Discord.GuildID != "" {
			if r.GuildID != sm.config.Discord.GuildID {
				return SignupOperationResult{Error: fmt.Errorf("reaction from unauthorized guild")}, nil
			}
		}

		dmChannel, err := sm.session.UserChannelCreate(r.UserID)
		if err != nil {
			sm.logger.ErrorContext(ctx, "Failed to create DM channel", attr.Error(err))
			return SignupOperationResult{Error: fmt.Errorf("failed to create DM channel: %w", err)}, err
		}
		sm.logger.InfoContext(ctx, "DM channel created", attr.String("dm_channel_id", dmChannel.ID))

		// Include guild ID in the button's custom ID so it's available when pressed in DM
		metadataStr := fmt.Sprintf("signup_button|%s|guild_id=%s", r.UserID, r.GuildID)

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
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "handle_signup_button")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "button")

	if i != nil && i.Interaction != nil && i.Interaction.GuildID != "" {
		ctx = discordmetrics.WithValue(ctx, discordmetrics.GuildIDKey, i.Interaction.GuildID)
	}

	// Extract user ID for metrics
	var userID string
	if i != nil && i.Interaction != nil {
		if i.Interaction.Member != nil && i.Interaction.Member.User != nil {
			userID = i.Interaction.Member.User.ID
		} else if i.Interaction.User != nil {
			userID = i.Interaction.User.ID
		}
		if userID != "" {
			ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, userID)
		}
	}

	// Extract GuildID from button CustomID since DM interactions don't have guild context
	var guildID string
	if i != nil && i.Interaction != nil {
		guildID = i.Interaction.GuildID // Try direct first
		if guildID == "" && i.Interaction.Type == discordgo.InteractionMessageComponent {
			// Extract from button CustomID: "signup_button|userID|guild_id=GUILD_ID"
			mcd := i.Interaction.MessageComponentData()
			customID := mcd.CustomID
			if strings.Contains(customID, "guild_id=") {
				parts := strings.Split(customID, "guild_id=")
				if len(parts) == 2 {
					guildID = parts[1]
				}
			}
		}
	}

	return sm.operationWrapper(ctx, "handle_signup_button_press", func(ctx context.Context) (SignupOperationResult, error) {
		// Only store minimal context if we have a token and guildID; avoid extra writes that tests don't expect
		if i != nil && i.Interaction != nil && guildID != "" && i.Interaction.Token != "" {
			_ = sm.interactionStore.Set(i.Interaction.Token+":guild_id", guildID, 10*60) // 10 min expiry
		}
		result, err := sm.SendSignupModal(ctx, i)
		if err != nil {
			sm.logger.ErrorContext(ctx, "❌ Failed to send signup modal", attr.Error(err))
			return SignupOperationResult{Error: err}, nil
		} else {
			sm.logger.InfoContext(ctx, "✅ Successfully called SendSignupModal")
		}
		return result, nil
	})
}

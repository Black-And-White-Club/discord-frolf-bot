package setup

import (
	"context"
	"fmt"
	"strings"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/bwmarrin/discordgo"
)

// SetupResult represents the result of a guild setup operation
type SetupResult struct {
	EventChannelID         string
	EventChannelName       string
	LeaderboardChannelID   string
	LeaderboardChannelName string
	SignupChannelID        string
	SignupChannelName      string
	UserRoleID             string
	EditorRoleID           string
	AdminRoleID            string
	SignupMessageID        string
	SignupEmoji            string
	RoleMappings           map[string]string
}

// SetupConfig represents the configuration for guild setup
type SetupConfig struct {
	GuildName       string
	ChannelPrefix   string
	UserRoleName    string
	EditorRoleName  string
	AdminRoleName   string
	SignupMessage   string
	SignupEmoji     string
	CreateChannels  bool
	CreateRoles     bool
	CreateSignupMsg bool
}

// SendSetupModal sends the guild setup modal to the user
func (s *setupManager) SendSetupModal(ctx context.Context, i *discordgo.InteractionCreate) error {
	return s.sendSetupModalWithCorrelation(ctx, i, newSetupCorrelationID())
}

func (s *setupManager) sendSetupModalWithCorrelation(ctx context.Context, i *discordgo.InteractionCreate, correlationID string) error {
	customID := setupModalCustomID(correlationID)

	return s.operationWrapper(ctx, "send_setup_modal", func(ctx context.Context) error {
		s.logger.InfoContext(ctx, "Sending guild setup modal", "guild_id", i.GuildID)

		// Send the modal as the initial response
		err := s.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				Title:    "🥏 Frolf Bot Setup",
				CustomID: customID,
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.TextInput{
								CustomID:    "channel_prefix",
								Label:       "Channel Name Prefix",
								Style:       discordgo.TextInputShort,
								Placeholder: "frolf (creates frolf-events, frolf-leaderboard, etc.)",
								Required:    false,
								MaxLength:   20,
								Value:       "frolf",
							},
						},
					},
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.TextInput{
								CustomID:    "role_names",
								Label:       "Role Names (User, Editor, Admin)",
								Style:       discordgo.TextInputShort,
								Placeholder: "Frolf Player, Frolf Editor, Frolf Admin",
								Required:    false,
								MaxLength:   150,
								Value:       "Frolf Player, Frolf Editor, Frolf Admin",
							},
						},
					},
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.TextInput{
								CustomID:    "signup_message",
								Label:       "Signup Message",
								Style:       discordgo.TextInputParagraph,
								Placeholder: "React with 🥏 to sign up for frolf events!",
								Required:    false,
								MaxLength:   500,
								Value:       "React with 🥏 to sign up for frolf events!",
							},
						},
					},
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.TextInput{
								CustomID:    "signup_emoji",
								Label:       "Signup Emoji",
								Style:       discordgo.TextInputShort,
								Placeholder: "🥏",
								Required:    false,
								MaxLength:   10,
								Value:       "🥏",
							},
						},
					},
				},
			},
		})
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to send guild setup modal", "guild_id", i.GuildID, "error", err)
			return fmt.Errorf("failed to send guild setup modal: %w", err)
		}

		s.logger.InfoContext(ctx, "Guild setup modal sent successfully", "guild_id", i.GuildID)
		return nil
	})
}

// HandleSetupModalSubmit handles the submission of the guild setup modal
func (s *setupManager) HandleSetupModalSubmit(ctx context.Context, i *discordgo.InteractionCreate) error {
	return s.operationWrapper(ctx, "handle_setup_modal_submit", func(ctx context.Context) error {
		s.logger.InfoContext(ctx, "Handling guild setup modal submission", "guild_id", i.GuildID)

		// Store the interaction so asynchronous backend processing can update the original UI
		correlationID := setupCorrelationIDFromCustomID(i.ModalSubmitData().CustomID)
		if correlationID == "" {
			correlationID = newSetupCorrelationID()
		}

		if s.interactionStore != nil {
			if err := s.interactionStore.Set(ctx, correlationID, i.Interaction); err != nil {
				s.logger.ErrorContext(ctx, "Failed to store interaction for setup modal submission",
					"guild_id", i.GuildID,
					"correlation_id", correlationID,
					"error", err)
				// Non-fatal: continue processing so user still receives acknowledgement
			} else {
				s.logger.DebugContext(ctx, "Stored interaction for setup modal submission",
					"guild_id", i.GuildID,
					"correlation_id", correlationID)
			}
		}

		// Extract form data
		data := i.ModalSubmitData()

		// Safety check for component count
		if len(data.Components) < 4 {
			s.logger.ErrorContext(ctx, "Modal data missing components",
				"guild_id", i.GuildID,
				"expected", 4,
				"received", len(data.Components))
			return s.respondError(i, "Invalid form submission - missing data")
		}

		// Safely extract text input values from components (support value and pointer variants)
		getRow := func(mc discordgo.MessageComponent) (row discordgo.ActionsRow, ok bool) {
			switch v := mc.(type) {
			case discordgo.ActionsRow:
				return v, true
			case *discordgo.ActionsRow:
				if v != nil {
					return *v, true
				}
			}
			return discordgo.ActionsRow{}, false
		}
		getText := func(mc discordgo.MessageComponent) (string, bool) {
			switch v := mc.(type) {
			case discordgo.TextInput:
				return v.Value, true
			case *discordgo.TextInput:
				if v != nil {
					return v.Value, true
				}
			}
			return "", false
		}

		// Extract values with validation
		var channelPrefix, roleNames, signupMessage, signupEmoji string
		rows := data.Components
		if r, ok := getRow(rows[0]); ok && len(r.Components) > 0 {
			if v, ok := getText(r.Components[0]); ok {
				channelPrefix = strings.TrimSpace(v)
			}
		}
		if r, ok := getRow(rows[1]); ok && len(r.Components) > 0 {
			if v, ok := getText(r.Components[0]); ok {
				roleNames = strings.TrimSpace(v)
			}
		}
		if r, ok := getRow(rows[2]); ok && len(r.Components) > 0 {
			if v, ok := getText(r.Components[0]); ok {
				signupMessage = strings.TrimSpace(v)
			}
		}
		if r, ok := getRow(rows[3]); ok && len(r.Components) > 0 {
			if v, ok := getText(r.Components[0]); ok {
				signupEmoji = strings.TrimSpace(v)
			}
		}

		// Parse role names from comma-separated string
		var userRoleName, editorRoleName, adminRoleName string
		if roleNames == "" {
			roleNames = "Frolf Player, Frolf Editor, Frolf Admin"
		}
		roleParts := strings.Split(roleNames, ",")
		if len(roleParts) >= 1 {
			userRoleName = strings.TrimSpace(roleParts[0])
		}
		if len(roleParts) >= 2 {
			editorRoleName = strings.TrimSpace(roleParts[1])
		}
		if len(roleParts) >= 3 {
			adminRoleName = strings.TrimSpace(roleParts[2])
		}

		// Apply defaults
		if channelPrefix == "" {
			channelPrefix = "frolf"
		}
		if userRoleName == "" {
			userRoleName = "Frolf Player"
		}
		if editorRoleName == "" {
			editorRoleName = "Frolf Editor"
		}
		if adminRoleName == "" {
			adminRoleName = "Frolf Admin"
		}
		if signupMessage == "" {
			signupMessage = "React with 🥏 to sign up for frolf events!"
		}
		if signupEmoji == "" {
			signupEmoji = "🥏"
		}

		// Get guild name automatically
		guild, err := s.session.Guild(i.GuildID)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to get guild info", "guild_id", i.GuildID, "error", err)
			return s.respondError(i, "Failed to get guild information")
		}
		guildName := guild.Name

		s.logger.InfoContext(ctx, "Processing guild setup with custom options",
			"guild_id", i.GuildID,
			"guild_name", guildName,
			"channel_prefix", channelPrefix,
			"user_role", userRoleName,
			"editor_role", editorRoleName,
			"admin_role", adminRoleName,
			"signup_message", signupMessage,
			"signup_emoji", signupEmoji)

		// Acknowledge the submission immediately
		err = s.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "🥏 Setting up your guild... This may take a moment.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to acknowledge setup submission", "guild_id", i.GuildID, "error", err)
			return fmt.Errorf("failed to acknowledge setup submission: %w", err)
		}

		// If the guild is already configured, surface that to the user and skip creating resources
		if s.guildConfigResolver != nil {
			existingCfg, cfgErr := s.guildConfigResolver.GetGuildConfigWithContext(ctx, i.GuildID)
			if cfgErr != nil {
				if s.logger != nil {
					s.logger.WarnContext(ctx, "Failed to fetch existing guild config before setup",
						"guild_id", i.GuildID,
						"error", cfgErr,
					)
				}
			} else if existingCfg != nil && existingCfg.IsConfigured() {
				if s.logger != nil {
					s.logger.InfoContext(ctx, "Guild already configured — skipping setup",
						"guild_id", i.GuildID)
				}
				return s.sendFollowupAlreadyConfigured(i, existingCfg)
			}
		}

		// Perform the actual setup - always create channels, roles, and signup message
		result, err := s.performCustomSetup(ctx, i.GuildID, SetupConfig{
			GuildName:       guildName,
			ChannelPrefix:   channelPrefix,
			UserRoleName:    userRoleName,
			EditorRoleName:  editorRoleName,
			AdminRoleName:   adminRoleName,
			SignupMessage:   signupMessage,
			SignupEmoji:     signupEmoji,
			CreateChannels:  true, // Always create channels
			CreateRoles:     true, // Always create roles
			CreateSignupMsg: true, // Always create signup message
		})
		if err != nil {
			s.logger.ErrorContext(ctx, "Custom setup failed", "guild_id", i.GuildID, "error", err)
			return s.sendFollowupError(i, fmt.Sprintf("Setup failed: %v", err))
		}

		s.logger.InfoContext(ctx, "Guild setup completed - config will be available from backend",
			"guild_id", i.GuildID,
			"signup_channel_id", result.SignupChannelID,
			"signup_message_id", result.SignupMessageID,
			"signup_emoji", result.SignupEmoji)

		// Publish setup event to backend
		if err := s.publishSetupEvent(i, result, correlationID); err != nil {
			s.logger.ErrorContext(ctx, "Failed to publish setup event", "guild_id", i.GuildID, "error", err)
			return s.sendFollowupError(i, "Setup completed but failed to save configuration")
		}

		// Send success followup
		return s.sendFollowupSuccess(i, result)
	})
}

// sendFollowupError sends an error message as a followup
func (s *setupManager) sendFollowupError(i *discordgo.InteractionCreate, errMsg string) error {
	_, err := s.session.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: fmt.Sprintf("❌ %s", errMsg),
		Flags:   discordgo.MessageFlagsEphemeral,
	})
	return err
}

// sendFollowupSuccess sends a success message as a followup
func (s *setupManager) sendFollowupSuccess(i *discordgo.InteractionCreate, result *SetupResult) error {
	embed := &discordgo.MessageEmbed{
		Title:       "🥏 Frolf Bot Setup Complete!",
		Description: "Your server is ready for disc golf! Here's what I've set up:",
		Color:       0x00ff00,
		Fields:      []*discordgo.MessageEmbedField{},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Use /create-round to get started!",
		},
	}

	// Add fields based on what was created
	if result.EventChannelID != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "📊 Events Channel",
			Value:  fmt.Sprintf("<#%s>", result.EventChannelID),
			Inline: true,
		})
	}
	if result.LeaderboardChannelID != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "🏆 Leaderboard Channel",
			Value:  fmt.Sprintf("<#%s>", result.LeaderboardChannelID),
			Inline: true,
		})
	}
	if result.SignupChannelID != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "✋ Signup Channel",
			Value:  fmt.Sprintf("<#%s>", result.SignupChannelID),
			Inline: true,
		})
	}
	if result.UserRoleID != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "👥 User Role",
			Value:  fmt.Sprintf("<@&%s>", result.UserRoleID),
			Inline: true,
		})
	}
	if result.EditorRoleID != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "✏️ Editor Role",
			Value:  fmt.Sprintf("<@&%s>", result.EditorRoleID),
			Inline: true,
		})
	}
	if result.AdminRoleID != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "⚡ Admin Role",
			Value:  fmt.Sprintf("<@&%s>", result.AdminRoleID),
			Inline: true,
		})
	}

	_, err := s.session.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{embed},
		Flags:  discordgo.MessageFlagsEphemeral,
	})
	return err
}

// sendFollowupAlreadyConfigured informs the user that setup was skipped because a config already exists.
func (s *setupManager) sendFollowupAlreadyConfigured(i *discordgo.InteractionCreate, cfg *storage.GuildConfig) error {
	embed := &discordgo.MessageEmbed{
		Title:       "🥏 Frolf Bot is already configured",
		Description: "I detected an existing configuration for this server, so I didn't create new channels or roles. Use /frolf-config update to make changes.",
		Color:       0xf9a602,
		Fields:      []*discordgo.MessageEmbedField{},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Use /frolf-config update to adjust settings.",
		},
	}

	if cfg.EventChannelID != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "📊 Events Channel",
			Value:  fmt.Sprintf("<#%s>", cfg.EventChannelID),
			Inline: true,
		})
	}
	if cfg.LeaderboardChannelID != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "🏆 Leaderboard Channel",
			Value:  fmt.Sprintf("<#%s>", cfg.LeaderboardChannelID),
			Inline: true,
		})
	}
	if cfg.SignupChannelID != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "✋ Signup Channel",
			Value:  fmt.Sprintf("<#%s>", cfg.SignupChannelID),
			Inline: true,
		})
	}
	if cfg.RegisteredRoleID != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "👥 User Role",
			Value:  fmt.Sprintf("<@&%s>", cfg.RegisteredRoleID),
			Inline: true,
		})
	}
	if cfg.EditorRoleID != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "✏️ Editor Role",
			Value:  fmt.Sprintf("<@&%s>", cfg.EditorRoleID),
			Inline: true,
		})
	}
	if cfg.AdminRoleID != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "⚡ Admin Role",
			Value:  fmt.Sprintf("<@&%s>", cfg.AdminRoleID),
			Inline: true,
		})
	}
	if cfg.SignupMessageID != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  "📝 Signup Message ID",
			Value: fmt.Sprintf("`%s`", cfg.SignupMessageID),
		})
	}
	if cfg.SignupEmoji != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  "Emoji",
			Value: cfg.SignupEmoji,
		})
	}

	_, err := s.session.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: "ℹ️ Setup skipped — this server is already configured.",
		Embeds:  []*discordgo.MessageEmbed{embed},
		Flags:   discordgo.MessageFlagsEphemeral,
	})
	return err
}

// respondError sends an error response
func (s *setupManager) respondError(i *discordgo.InteractionCreate, errMsg string) error {
	return s.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("❌ %s", errMsg),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

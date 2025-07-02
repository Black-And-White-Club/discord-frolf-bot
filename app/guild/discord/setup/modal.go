package setup

import (
	"context"
	"fmt"
	"strings"

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
	RegisteredRoleID       string
	AdminRoleID            string
	SignupMessageID        string
	RoleMappings           map[string]string
}

// SetupConfig represents the configuration for guild setup
type SetupConfig struct {
	GuildName       string
	ChannelPrefix   string
	PlayerRoleName  string
	AdminRoleName   string
	CreateChannels  bool
	CreateRoles     bool
	CreateSignupMsg bool
}

// SendSetupModal sends the guild setup modal to the user
func (s *setupManager) SendSetupModal(ctx context.Context, i *discordgo.InteractionCreate) error {
	return s.operationWrapper(ctx, "send_setup_modal", func(ctx context.Context) error {
		s.logger.InfoContext(ctx, "Sending guild setup modal", "guild_id", i.GuildID)

		// Send the modal as the initial response
		err := s.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				Title:    "🥏 Frolf Bot Setup",
				CustomID: "guild_setup_modal",
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.TextInput{
								CustomID:    "guild_name",
								Label:       "Guild Display Name",
								Style:       discordgo.TextInputShort,
								Placeholder: "My Frolf Community",
								Required:    false,
								MaxLength:   100,
							},
						},
					},
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
								CustomID:    "player_role_name",
								Label:       "Player Role Name",
								Style:       discordgo.TextInputShort,
								Placeholder: "Frolf Player",
								Required:    false,
								MaxLength:   50,
								Value:       "Frolf Player",
							},
						},
					},
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.TextInput{
								CustomID:    "admin_role_name",
								Label:       "Admin Role Name",
								Style:       discordgo.TextInputShort,
								Placeholder: "Frolf Admin",
								Required:    false,
								MaxLength:   50,
								Value:       "Frolf Admin",
							},
						},
					},
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.TextInput{
								CustomID:    "setup_options",
								Label:       "Setup Options (comma-separated)",
								Style:       discordgo.TextInputParagraph,
								Placeholder: "auto-channels, auto-roles, signup-message (or leave blank for all)",
								Required:    false,
								MaxLength:   200,
								Value:       "auto-channels, auto-roles, signup-message",
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

		// Extract form data
		data := i.ModalSubmitData()
		guildName := strings.TrimSpace(data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value)
		channelPrefix := strings.TrimSpace(data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value)
		playerRoleName := strings.TrimSpace(data.Components[2].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value)
		adminRoleName := strings.TrimSpace(data.Components[3].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value)
		setupOptions := strings.TrimSpace(data.Components[4].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value)

		// Apply defaults
		if channelPrefix == "" {
			channelPrefix = "frolf"
		}
		if playerRoleName == "" {
			playerRoleName = "Frolf Player"
		}
		if adminRoleName == "" {
			adminRoleName = "Frolf Admin"
		}
		if setupOptions == "" {
			setupOptions = "auto-channels, auto-roles, signup-message"
		}

		// Get actual guild name if not provided
		if guildName == "" {
			guild, err := s.session.Guild(i.GuildID)
			if err != nil {
				s.logger.ErrorContext(ctx, "Failed to get guild info", "guild_id", i.GuildID, "error", err)
				return s.respondError(i, "Failed to get guild information")
			}
			guildName = guild.Name
		}

		s.logger.InfoContext(ctx, "Processing guild setup with custom options",
			"guild_id", i.GuildID,
			"guild_name", guildName,
			"channel_prefix", channelPrefix,
			"player_role", playerRoleName,
			"admin_role", adminRoleName,
			"options", setupOptions)

		// Acknowledge the submission immediately
		err := s.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
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

		// Parse setup options
		options := parseSetupOptions(setupOptions)

		// Perform the actual setup
		result, err := s.performCustomSetup(i.GuildID, SetupConfig{
			GuildName:       guildName,
			ChannelPrefix:   channelPrefix,
			PlayerRoleName:  playerRoleName,
			AdminRoleName:   adminRoleName,
			CreateChannels:  options["auto-channels"],
			CreateRoles:     options["auto-roles"],
			CreateSignupMsg: options["signup-message"],
		})
		if err != nil {
			s.logger.ErrorContext(ctx, "Custom setup failed", "guild_id", i.GuildID, "error", err)
			return s.sendFollowupError(i, fmt.Sprintf("Setup failed: %v", err))
		}

		// Publish setup event to backend
		if err := s.publishSetupEvent(i, result); err != nil {
			s.logger.ErrorContext(ctx, "Failed to publish setup event", "guild_id", i.GuildID, "error", err)
			return s.sendFollowupError(i, "Setup completed but failed to save configuration")
		}

		// Send success followup
		return s.sendFollowupSuccess(i, result)
	})
}

// parseSetupOptions parses the comma-separated setup options
func parseSetupOptions(optionsStr string) map[string]bool {
	options := map[string]bool{
		"auto-channels":  false,
		"auto-roles":     false,
		"signup-message": false,
	}

	if optionsStr == "" {
		// Default to all options
		for key := range options {
			options[key] = true
		}
		return options
	}

	parts := strings.Split(optionsStr, ",")
	for _, part := range parts {
		option := strings.TrimSpace(strings.ToLower(part))
		if _, exists := options[option]; exists {
			options[option] = true
		}
	}

	return options
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
	if result.RegisteredRoleID != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "👥 Player Role",
			Value:  fmt.Sprintf("<@&%s>", result.RegisteredRoleID),
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

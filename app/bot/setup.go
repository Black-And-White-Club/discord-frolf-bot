package bot

// import (
// 	"context"
// 	"fmt"
// 	"strings"

// 	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
// 	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
// 	"github.com/bwmarrin/discordgo"
// )

// // ChannelPermissions defines permission settings for a channel
// type ChannelPermissions struct {
// 	RestrictPosting bool     // If true, only allowed roles can post
// 	AllowedRoles    []string // Roles that can post when RestrictPosting is true
// }

// // ServerSetupConfig holds the configuration for setting up a new Discord server
// type ServerSetupConfig struct {
// 	GuildID              string
// 	RequiredChannels     []string // e.g., ["signup", "events", "leaderboard"]
// 	RequiredRoles        []string // e.g., ["User", "Editor", "Admin"]
// 	SignupEmojiName      string   // e.g., "🐍"
// 	CreateSignupMessage  bool
// 	SignupMessageContent string
// 	RegisteredRoleName   string                        // Which role should be the "registered" role
// 	AdminRoleName        string                        // Which role should be the "admin" role
// 	ChannelPermissions   map[string]ChannelPermissions // Channel name -> permissions
// }

// // AutoSetupServer automatically discovers or creates required Discord resources
// func (bot *DiscordBot) AutoSetupServer(ctx context.Context, setupConfig ServerSetupConfig) error {
// 	session := bot.Session.(*discord.DiscordSession).GetUnderlyingSession()

// 	bot.Logger.InfoContext(ctx, "Starting automatic server setup",
// 		attr.String("guild_id", setupConfig.GuildID))

// 	// Verify we can access the guild
// 	guild, err := session.Guild(setupConfig.GuildID)
// 	if err != nil {
// 		return fmt.Errorf("failed to access guild %s: %w", setupConfig.GuildID, err)
// 	}

// 	bot.Logger.InfoContext(ctx, "Found guild",
// 		attr.String("guild_name", guild.Name),
// 		attr.Int("member_count", guild.MemberCount))

// 	// 1. Setup channels
// 	channelIDs, err := bot.setupChannels(ctx, session, setupConfig.GuildID, setupConfig.RequiredChannels)
// 	if err != nil {
// 		return fmt.Errorf("failed to setup channels: %w", err)
// 	}

// 	// 2. Setup roles
// 	roleIDs, err := bot.setupRoles(ctx, session, setupConfig.GuildID, setupConfig.RequiredRoles)
// 	if err != nil {
// 		return fmt.Errorf("failed to setup roles: %w", err)
// 	}

// 	// 3. Setup channel permissions
// 	if len(setupConfig.ChannelPermissions) > 0 {
// 		err = bot.setupChannelPermissions(ctx, session, setupConfig.GuildID, channelIDs, roleIDs, setupConfig.ChannelPermissions)
// 		if err != nil {
// 			bot.Logger.WarnContext(ctx, "Failed to setup some channel permissions", attr.Error(err))
// 		}
// 	}

// 	// 4. Create signup message if requested
// 	var signupMessageID string
// 	if setupConfig.CreateSignupMessage && channelIDs["signup"] != "" {
// 		signupMessageID, err = bot.createSignupMessage(ctx, session, channelIDs["signup"], setupConfig.SignupMessageContent, setupConfig.SignupEmojiName)
// 		if err != nil {
// 			bot.Logger.WarnContext(ctx, "Failed to create signup message", attr.Error(err))
// 		}
// 	}

// 	// 5. Update bot configuration with discovered/created IDs
// 	bot.updateConfiguration(setupConfig.GuildID, channelIDs, roleIDs, signupMessageID, setupConfig)

// 	bot.Logger.InfoContext(ctx, "Server setup completed successfully",
// 		attr.String("guild_id", setupConfig.GuildID),
// 		attr.Any("channels", channelIDs),
// 		attr.Any("roles", roleIDs))

// 	return nil
// }

// func (bot *DiscordBot) setupChannels(ctx context.Context, session *discordgo.Session, guildID string, requiredChannels []string) (map[string]string, error) {
// 	channelIDs := make(map[string]string)

// 	// Get existing channels
// 	channels, err := session.GuildChannels(guildID)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get guild channels: %w", err)
// 	}

// 	// Create a map of existing channels
// 	existingChannels := make(map[string]*discordgo.Channel)
// 	for _, channel := range channels {
// 		if channel.Type == discordgo.ChannelTypeGuildText {
// 			existingChannels[channel.Name] = channel
// 		}
// 	}

// 	// Find or create required channels
// 	for _, channelName := range requiredChannels {
// 		if existingChannel, exists := existingChannels[channelName]; exists {
// 			channelIDs[channelName] = existingChannel.ID
// 			bot.Logger.InfoContext(ctx, "Found existing channel",
// 				attr.String("channel_name", channelName),
// 				attr.String("channel_id", existingChannel.ID))
// 		} else {
// 			// Create new channel
// 			newChannel, err := session.GuildChannelCreate(guildID, channelName, discordgo.ChannelTypeGuildText)
// 			if err != nil {
// 				bot.Logger.ErrorContext(ctx, "Failed to create channel",
// 					attr.String("channel_name", channelName),
// 					attr.Error(err))
// 				continue
// 			}
// 			channelIDs[channelName] = newChannel.ID
// 			bot.Logger.InfoContext(ctx, "Created new channel",
// 				attr.String("channel_name", channelName),
// 				attr.String("channel_id", newChannel.ID))
// 		}
// 	}

// 	return channelIDs, nil
// }

// func (bot *DiscordBot) setupRoles(ctx context.Context, session *discordgo.Session, guildID string, requiredRoles []string) (map[string]string, error) {
// 	roleIDs := make(map[string]string)

// 	// Get existing roles
// 	roles, err := session.GuildRoles(guildID)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get guild roles: %w", err)
// 	}

// 	// Create a map of existing roles
// 	existingRoles := make(map[string]*discordgo.Role)
// 	for _, role := range roles {
// 		existingRoles[role.Name] = role
// 	}

// 	// Find or create required roles
// 	for _, roleName := range requiredRoles {
// 		if existingRole, exists := existingRoles[roleName]; exists {
// 			roleIDs[roleName] = existingRole.ID
// 			bot.Logger.InfoContext(ctx, "Found existing role",
// 				attr.String("role_name", roleName),
// 				attr.String("role_id", existingRole.ID))
// 		} else {
// 			// Create new role with properties
// 			var permissions int64 = 0
// 			if strings.ToLower(roleName) == "admin" {
// 				permissions = discordgo.PermissionAdministrator
// 			}

// 			roleParams := &discordgo.RoleParams{
// 				Name:        roleName,
// 				Permissions: &permissions,
// 				Mentionable: &[]bool{true}[0],
// 			}

// 			newRole, err := session.GuildRoleCreate(guildID, roleParams)
// 			if err != nil {
// 				bot.Logger.ErrorContext(ctx, "Failed to create role",
// 					attr.String("role_name", roleName),
// 					attr.Error(err))
// 				continue
// 			}

// 			roleIDs[roleName] = newRole.ID
// 			bot.Logger.InfoContext(ctx, "Created new role",
// 				attr.String("role_name", roleName),
// 				attr.String("role_id", newRole.ID))
// 		}
// 	}

// 	return roleIDs, nil
// }

// func (bot *DiscordBot) setupChannelPermissions(ctx context.Context, session *discordgo.Session, guildID string, channelIDs map[string]string, roleIDs map[string]string, permissionConfig map[string]ChannelPermissions) error {
// 	for channelName, permissions := range permissionConfig {
// 		channelID, exists := channelIDs[channelName]
// 		if !exists {
// 			continue
// 		}

// 		// Special handling for signup channel
// 		if channelName == "signup" {
// 			// Make signup channel visible to @everyone by default
// 			err := session.ChannelPermissionSet(channelID, guildID, discordgo.PermissionOverwriteTypeRole,
// 				discordgo.PermissionViewChannel, 0)
// 			if err != nil {
// 				bot.Logger.WarnContext(ctx, "Failed to set @everyone view permission for signup",
// 					attr.Error(err))
// 			}

// 			// Hide signup channel from User and Editor roles
// 			for _, roleName := range []string{"User", "Editor"} {
// 				if roleID, roleExists := roleIDs[roleName]; roleExists {
// 					err := session.ChannelPermissionSet(channelID, roleID, discordgo.PermissionOverwriteTypeRole,
// 						0, discordgo.PermissionViewChannel)
// 					if err != nil {
// 						bot.Logger.WarnContext(ctx, "Failed to hide signup channel from role",
// 							attr.String("role_name", roleName),
// 							attr.Error(err))
// 					} else {
// 						bot.Logger.InfoContext(ctx, "Hid signup channel from role",
// 							attr.String("role_name", roleName))
// 					}
// 				}
// 			}

// 			// Ensure Admin can always see signup channel (override)
// 			if adminRoleID, adminExists := roleIDs["Admin"]; adminExists {
// 				err := session.ChannelPermissionSet(channelID, adminRoleID, discordgo.PermissionOverwriteTypeRole,
// 					discordgo.PermissionViewChannel|discordgo.PermissionSendMessages, 0)
// 				if err != nil {
// 					bot.Logger.WarnContext(ctx, "Failed to grant admin access to signup",
// 						attr.Error(err))
// 				} else {
// 					bot.Logger.InfoContext(ctx, "Granted admin override access to signup channel")
// 				}
// 			}

// 			continue // Skip the normal permission handling for signup
// 		}

// 		// **ADD SPECIAL HANDLING FOR EVENTS CHANNEL**
// 		if channelName == "events" {
// 			// Deny @everyone from sending messages but allow viewing and creating threads
// 			err := session.ChannelPermissionSet(channelID, guildID, discordgo.PermissionOverwriteTypeRole,
// 				discordgo.PermissionViewChannel|discordgo.PermissionCreatePublicThreads|discordgo.PermissionSendMessagesInThreads,
// 				discordgo.PermissionSendMessages)
// 			if err != nil {
// 				bot.Logger.WarnContext(ctx, "Failed to set events channel permissions for @everyone",
// 					attr.Error(err))
// 			}

// 			// Allow Admin to post events and manage threads
// 			if adminRoleID, adminExists := roleIDs["Admin"]; adminExists {
// 				err := session.ChannelPermissionSet(channelID, adminRoleID, discordgo.PermissionOverwriteTypeRole,
// 					discordgo.PermissionViewChannel|discordgo.PermissionSendMessages|discordgo.PermissionCreatePublicThreads|discordgo.PermissionSendMessagesInThreads|discordgo.PermissionManageThreads,
// 					0)
// 				if err != nil {
// 					bot.Logger.WarnContext(ctx, "Failed to grant admin access to events channel",
// 						attr.Error(err))
// 				} else {
// 					bot.Logger.InfoContext(ctx, "Granted admin full access to events channel")
// 				}
// 			}

// 			// Allow players to create threads and chat in them, but not post in main channel
// 			for _, roleName := range []string{"User", "Editor"} {
// 				if roleID, roleExists := roleIDs[roleName]; roleExists {
// 					err := session.ChannelPermissionSet(channelID, roleID, discordgo.PermissionOverwriteTypeRole,
// 						discordgo.PermissionViewChannel|discordgo.PermissionCreatePublicThreads|discordgo.PermissionSendMessagesInThreads,
// 						discordgo.PermissionSendMessages)
// 					if err != nil {
// 						bot.Logger.WarnContext(ctx, "Failed to set thread permissions for role in events",
// 							attr.String("role_name", roleName),
// 							attr.Error(err))
// 					} else {
// 						bot.Logger.InfoContext(ctx, "Granted thread permissions to role in events channel",
// 							attr.String("role_name", roleName))
// 					}
// 				}
// 			}

// 			bot.Logger.InfoContext(ctx, "Set events channel as embed-only with thread discussions")
// 			continue // Skip normal permission handling
// 		}

// 		// Normal channel permission handling for other channels
// 		if permissions.RestrictPosting {
// 			// Deny @everyone send messages permission
// 			err := session.ChannelPermissionSet(channelID, guildID, discordgo.PermissionOverwriteTypeRole,
// 				0, discordgo.PermissionSendMessages)
// 			if err != nil {
// 				bot.Logger.WarnContext(ctx, "Failed to restrict channel permissions",
// 					attr.String("channel_name", channelName),
// 					attr.Error(err))
// 				continue
// 			}

// 			// Allow specified roles to post
// 			for _, roleName := range permissions.AllowedRoles {
// 				if roleID, roleExists := roleIDs[roleName]; roleExists {
// 					err := session.ChannelPermissionSet(channelID, roleID, discordgo.PermissionOverwriteTypeRole,
// 						discordgo.PermissionSendMessages|discordgo.PermissionCreateInstantInvite, 0)
// 					if err != nil {
// 						bot.Logger.WarnContext(ctx, "Failed to allow role permission",
// 							attr.String("channel_name", channelName),
// 							attr.String("role_name", roleName),
// 							attr.Error(err))
// 					}
// 				}
// 			}

// 			bot.Logger.InfoContext(ctx, "Set restricted permissions for channel",
// 				attr.String("channel_name", channelName))
// 		}
// 	}
// 	return nil
// }

// func (bot *DiscordBot) createSignupMessage(ctx context.Context, session *discordgo.Session, channelID, content, emojiName string) (string, error) {
// 	if content == "" {
// 		content = "React with 🐍 to sign up for frolf events!"
// 	}

// 	message, err := session.ChannelMessageSend(channelID, content)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to send signup message: %w", err)
// 	}

// 	// Add reaction to the message
// 	err = session.MessageReactionAdd(channelID, message.ID, emojiName)
// 	if err != nil {
// 		bot.Logger.WarnContext(ctx, "Failed to add reaction to signup message", attr.Error(err))
// 	}

// 	bot.Logger.InfoContext(ctx, "Created signup message",
// 		attr.String("channel_id", channelID),
// 		attr.String("discord_message_id", message.ID))

// 	return message.ID, nil
// }

// func (bot *DiscordBot) updateConfiguration(guildID string, channelIDs, roleIDs map[string]string, signupMessageID string, setupConfig ServerSetupConfig) {
// 	// Update the bot's configuration with discovered/created IDs
// 	// NOTE: In single-server mode, we directly update the config structure
// 	// In future multi-tenant mode, this would save to database instead
// 	bot.Config.Discord.GuildID = guildID

// 	// Update channel IDs with specific mappings
// 	if channelID, exists := channelIDs["signup"]; exists {
// 		bot.Config.Discord.SignupChannelID = channelID
// 		if signupMessageID != "" {
// 			bot.Config.Discord.SignupMessageID = signupMessageID
// 		}
// 	}
// 	if channelID, exists := channelIDs["events"]; exists {
// 		bot.Config.Discord.EventChannelID = channelID
// 	}
// 	if channelID, exists := channelIDs["leaderboard"]; exists {
// 		bot.Config.Discord.LeaderboardChannelID = channelID
// 	}

// 	// Update role mappings
// 	if bot.Config.Discord.RoleMappings == nil {
// 		bot.Config.Discord.RoleMappings = make(map[string]string)
// 	}
// 	for roleName, roleID := range roleIDs {
// 		bot.Config.Discord.RoleMappings[roleName] = roleID
// 	}

// 	// Set special roles
// 	if roleID, exists := roleIDs[setupConfig.RegisteredRoleName]; exists {
// 		bot.Config.Discord.RegisteredRoleID = roleID
// 	}
// 	if roleID, exists := roleIDs[setupConfig.AdminRoleName]; exists {
// 		bot.Config.Discord.AdminRoleID = roleID
// 	}

// 	bot.Logger.Info("Configuration updated with new Discord IDs")
// }

// // ValidateSetup checks if the current configuration is valid for the target guild
// func (bot *DiscordBot) ValidateSetup(ctx context.Context, guildID string) error {
// 	session := bot.Session.(*discord.DiscordSession).GetUnderlyingSession()

// 	// Check if guild exists and is accessible
// 	_, err := session.Guild(guildID)
// 	if err != nil {
// 		return fmt.Errorf("cannot access guild %s: %w", guildID, err)
// 	}

// 	// Validate channels exist
// 	channels := []string{bot.Config.GetSignupChannelID(), bot.Config.GetEventChannelID()}
// 	for _, channelID := range channels {
// 		if channelID != "" {
// 			_, err := session.Channel(channelID)
// 			if err != nil {
// 				return fmt.Errorf("cannot access channel %s: %w", channelID, err)
// 			}
// 		}
// 	}

// 	// Validate roles exist
// 	roles, err := session.GuildRoles(guildID)
// 	if err != nil {
// 		return fmt.Errorf("cannot get guild roles: %w", err)
// 	}

// 	roleMap := make(map[string]bool)
// 	for _, role := range roles {
// 		roleMap[role.ID] = true
// 	}

// 	rolesToCheck := []string{bot.Config.GetRegisteredRoleID(), bot.Config.GetAdminRoleID()}
// 	for _, roleID := range bot.Config.GetRoleMappings() {
// 		rolesToCheck = append(rolesToCheck, roleID)
// 	}

// 	for _, roleID := range rolesToCheck {
// 		if roleID != "" && !roleMap[roleID] {
// 			return fmt.Errorf("role %s does not exist in guild", roleID)
// 		}
// 	}

// 	bot.Logger.InfoContext(ctx, "Setup validation passed", attr.String("guild_id", guildID))
// 	return nil
// }

package discord

import (
	"fmt"
	"log/slog"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

// RegisterCommands registers the bot's slash commands with Discord.
// This function supports both single-guild and multi-tenant deployments.
func RegisterCommands(s Session, logger *slog.Logger, guildID string) error {
	// For multi-tenant deployments, register commands globally (empty guildID)
	// For single-guild deployments, register for the specific guild
	appID, err := s.GetBotUser()
	if err != nil {
		return fmt.Errorf("failed to retrieve bot user: %w", err)
	}

	targetGuildID := guildID
	if targetGuildID == "" {
		logger.Info("Registering commands globally for multi-tenant deployment")
	} else {
		logger.Info("Registering commands for specific guild", attr.String("guild_id", targetGuildID))
	}

	_, err = s.ApplicationCommandCreate(appID.ID, targetGuildID, &discordgo.ApplicationCommand{
		Name:        "updaterole",
		Description: "Request a role for a user",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "user",
				Description: "The user to request a role for",
				Required:    true,
			},
		},
	})
	if err != nil {
		logger.Error("Failed to create '/updaterole' command", attr.Error(err))
		return fmt.Errorf("failed to create '/updaterole' command: %w", err)
	}
	logger.Info("registered command: /updaterole")

	_, err = s.ApplicationCommandCreate(appID.ID, targetGuildID, &discordgo.ApplicationCommand{
		Name: "createround",

		Description: "Create A Round",
	})
	if err != nil {
		logger.Error("Failed to create '/createround' command", attr.Error(err))
		return fmt.Errorf("failed to create '/createround' command: %w", err)
	}
	logger.Info("registered command: /createround")

	// Claim tag command
	_, err = s.ApplicationCommandCreate(appID.ID, targetGuildID, &discordgo.ApplicationCommand{
		Name:        "claimtag",
		Description: "Claim a specific tag number on the leaderboard",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "tag",
				Description: "Tag number to claim (1-100)",
				Required:    true,
				MinValue:    func() *float64 { v := 1.0; return &v }(),
				MaxValue:    100,
			},
		},
	})
	if err != nil {
		logger.Error("Failed to create '/claimtag' command", attr.Error(err))
		return fmt.Errorf("failed to create '/claimtag' command: %w", err)
	}
	logger.Info("registered command: /claimtag")

	// Frolf setup command
	_, err = s.ApplicationCommandCreate(appID.ID, targetGuildID, &discordgo.ApplicationCommand{
		Name:        "frolf-setup",
		Description: "Set up Frolf Bot for this server (Admin only)",
	})
	if err != nil {
		logger.Error("Failed to create '/frolf-setup' command", attr.Error(err))
		return fmt.Errorf("failed to create '/frolf-setup' command: %w", err)
	}
	logger.Info("registered command: /frolf-setup")

	return nil
}

// RegisterSetupCommand registers only the setup command for a new guild
func RegisterSetupCommand(s Session, logger *slog.Logger, guildID string) error {
	appID, err := s.GetBotUser()
	if err != nil {
		return fmt.Errorf("failed to retrieve bot user: %w", err)
	}

	_, err = s.ApplicationCommandCreate(appID.ID, guildID, &discordgo.ApplicationCommand{
		Name:        "frolf-setup",
		Description: "Set up Frolf Bot for this server (Admin only)",
	})
	if err != nil {
		logger.Error("Failed to create '/frolf-setup' command",
			attr.Error(err),
			attr.String("guild_id", guildID))
		return fmt.Errorf("failed to create '/frolf-setup' command: %w", err)
	}

	logger.Info("Registered setup command for new guild",
		attr.String("guild_id", guildID))
	return nil
}

// RegisterAllCommandsForGuild registers all bot commands for a configured guild
func RegisterAllCommandsForGuild(s Session, logger *slog.Logger, guildID string) error {
	appID, err := s.GetBotUser()
	if err != nil {
		return fmt.Errorf("failed to retrieve bot user: %w", err)
	}

	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "updaterole",
			Description: "Request a role for a user",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user",
					Description: "The user to request a role for",
					Required:    true,
				},
			},
		},
		{
			Name:        "createround",
			Description: "Create A Round",
		},
		{
			Name:        "claimtag",
			Description: "Claim a specific tag number on the leaderboard",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "tag",
					Description: "Tag number to claim (1-100)",
					Required:    true,
					MinValue:    func() *float64 { v := 1.0; return &v }(),
					MaxValue:    100,
				},
			},
		},
		{
			Name:        "frolf-setup",
			Description: "Set up Frolf Bot for this server (Admin only)",
		},
	}

	for _, cmd := range commands {
		_, err = s.ApplicationCommandCreate(appID.ID, guildID, cmd)
		if err != nil {
			logger.Error("Failed to create command",
				attr.Error(err),
				attr.String("command", cmd.Name),
				attr.String("guild_id", guildID))
			return fmt.Errorf("failed to create '/%s' command: %w", cmd.Name, err)
		}
		logger.Info("Registered command for guild",
			attr.String("command", cmd.Name),
			attr.String("guild_id", guildID))
	}

	return nil
}

// UnregisterAllCommandsForGuild removes all bot commands from a guild
func UnregisterAllCommandsForGuild(s Session, logger *slog.Logger, guildID string) error {
	appID, err := s.GetBotUser()
	if err != nil {
		return fmt.Errorf("failed to retrieve bot user: %w", err)
	}

	// Get existing commands for this guild
	commands, err := s.ApplicationCommands(appID.ID, guildID)
	if err != nil {
		logger.Error("Failed to fetch existing commands",
			attr.Error(err),
			attr.String("guild_id", guildID))
		return fmt.Errorf("failed to fetch existing commands: %w", err)
	}

	// Delete each command
	for _, cmd := range commands {
		err = s.ApplicationCommandDelete(appID.ID, guildID, cmd.ID)
		if err != nil {
			logger.Error("Failed to delete command",
				attr.Error(err),
				attr.String("command", cmd.Name),
				attr.String("guild_id", guildID))
			return fmt.Errorf("failed to delete '/%s' command: %w", cmd.Name, err)
		}
		logger.Info("Unregistered command from guild",
			attr.String("command", cmd.Name),
			attr.String("guild_id", guildID))
	}

	return nil
}

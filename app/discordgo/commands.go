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

	// For multi-tenant mode (empty guildID), only register frolf-setup globally
	if targetGuildID == "" {
		_, err = s.ApplicationCommandCreate(appID.ID, "", &discordgo.ApplicationCommand{
			Name:                     "frolf-setup",
			Description:              "Set up Frolf Bot for this server (Admin only)",
			DefaultMemberPermissions: &[]int64{discordgo.PermissionAdministrator}[0],
		})
		if err != nil {
			logger.Error("Failed to create global '/frolf-setup' command", attr.Error(err))
			return fmt.Errorf("failed to create global '/frolf-setup' command: %w", err)
		}
		logger.Info("registered global command: /frolf-setup")
		return nil
	}

	// For guild-specific registration, register all commands for that guild
	_, err = s.ApplicationCommandCreate(appID.ID, targetGuildID, &discordgo.ApplicationCommand{
		Name:        "updaterole",
		Description: "Request a role for a user (Requires Editor role or higher)",
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
		Name:        "createround",
		Description: "Create a new frolf round (Available to all players)",
	})
	if err != nil {
		logger.Error("Failed to create '/createround' command", attr.Error(err))
		return fmt.Errorf("failed to create '/createround' command: %w", err)
	}
	logger.Info("registered command: /createround")

	_, err = s.ApplicationCommandCreate(appID.ID, targetGuildID, &discordgo.ApplicationCommand{
		Name:        "claimtag",
		Description: "Claim a specific tag number on the leaderboard (Available to all players)",
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

	return nil
}

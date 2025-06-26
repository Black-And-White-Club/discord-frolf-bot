package discord

import (
	"fmt"
	"log/slog"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

// RegisterCommands registers the bot's slash commands with Discord.
// ...existing code...

func RegisterCommands(s Session, logger *slog.Logger, guildID string) error {
	// --- /updaterole Command ---
	appID, err := s.GetBotUser()
	if err != nil {
		return fmt.Errorf("failed to retrieve bot user: %w", err)
	}
	_, err = s.ApplicationCommandCreate(appID.ID, guildID, &discordgo.ApplicationCommand{
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

	_, err = s.ApplicationCommandCreate(appID.ID, guildID, &discordgo.ApplicationCommand{
		Name:        "createround",
		Description: "Create A Round",
	})
	if err != nil {
		logger.Error("Failed to create '/createround' command", attr.Error(err))
		return fmt.Errorf("failed to create '/createround' command: %w", err)
	}
	logger.Info("registered command: /createround")

	// Claim tag command
	_, err = s.ApplicationCommandCreate(appID.ID, guildID, &discordgo.ApplicationCommand{
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
	_, err = s.ApplicationCommandCreate(appID.ID, guildID, &discordgo.ApplicationCommand{
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

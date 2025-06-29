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

	// Register commands globally (empty guildID) if no specific guild is provided
	// This allows the bot to work in any server it's invited to
	targetGuildID := guildID
	if targetGuildID == "" {
		logger.Info("Registering commands globally (will work in all servers)")
		targetGuildID = ""
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
		Name:        "createround",
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

	// Frolf setup command (with options for customization)
	_, err = s.ApplicationCommandCreate(appID.ID, targetGuildID, &discordgo.ApplicationCommand{
		Name:        "frolf-setup",
		Description: "Set up Frolf Bot for this server (Admin only)",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "signup_channel",
				Description: "Name for the signup channel (default: signup)",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "event_channel",
				Description: "Name for the event channel (default: events)",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "leaderboard_channel",
				Description: "Name for the leaderboard channel (default: leaderboard)",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "signup_emoji",
				Description: "Emoji for signup (default: 🐍)",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "signup_message",
				Description: "Signup message content (default provided)",
				Required:    false,
			},
		},
	})
	if err != nil {
		logger.Error("Failed to create '/frolf-setup' command", attr.Error(err))
		return fmt.Errorf("failed to create '/frolf-setup' command: %w", err)
	}
	logger.Info("registered command: /frolf-setup")

	return nil
}

package discord

import (
	"fmt"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

// RegisterCommands registers the bot's slash commands with Discord.
func RegisterCommands(s Session, logger observability.Logger, guildID string) error {
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
	return nil
}

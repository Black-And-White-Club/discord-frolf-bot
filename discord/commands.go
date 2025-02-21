package discord

import (
	"fmt"
	"log/slog"

	"github.com/bwmarrin/discordgo"
)

// RegisterCommands registers the bot's slash commands with Discord.
func RegisterCommands(s Session, logger *slog.Logger) error { // Still takes the interface!

	// Get the *discordgo.Session from our wrapper.  This is the ONLY place
	// we need to do this.
	ds, ok := s.(*DiscordSession)
	if !ok {
		return fmt.Errorf("invalid session type: expected *DiscordSession")
	}
	dgSession := ds.session // Get the *discordgo.Session

	// --- /updaterole Command ---
	user, err := s.GetBotUser()
	if err != nil {
		return fmt.Errorf("failed to retrieve bot user: %w", err)
	}
	_, err = dgSession.ApplicationCommandCreate(user.ID, "", &discordgo.ApplicationCommand{
		Name:         "updaterole",
		Description:  "Updates a user's role",
		DMPermission: func() *bool { b := false; return &b }(),
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "user",
				Description: "The user whose role you want to update",
				Required:    true,
			},
		},
	})
	if err != nil {
		logger.Error("Failed to create '/updaterole' command", slog.Any("error", err))
		return fmt.Errorf("failed to create '/updaterole' command: %w", err)
	}
	logger.Info("registered command: /updaterole")

	return nil
}

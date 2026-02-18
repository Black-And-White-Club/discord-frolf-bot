package invite

import (
	"context"
	"log/slog"

	discordpkg "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/bwmarrin/discordgo"
)

// InviteManager handles the /invite command.
type InviteManager interface {
	HandleInviteCommand(ctx context.Context, i *discordgo.InteractionCreate)
}

type inviteManager struct {
	session discordpkg.Session
	logger  *slog.Logger
	cfg     *config.Config
}

// NewInviteManager creates a new InviteManager.
func NewInviteManager(session discordpkg.Session, logger *slog.Logger, cfg *config.Config) InviteManager {
	return &inviteManager{
		session: session,
		logger:  logger,
		cfg:     cfg,
	}
}

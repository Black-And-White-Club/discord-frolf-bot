package handlers

import (
	"log/slog"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig"
	rounddiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
)

// RoundHandlers handles round-related events.
type RoundHandlers struct {
	service             rounddiscord.RoundDiscordInterface
	helpers             utils.Helpers
	config              *config.Config
	guildConfigResolver guildconfig.GuildConfigResolver
	logger              *slog.Logger
}

// NewRoundHandlers creates a new RoundHandlers.
func NewRoundHandlers(
	logger *slog.Logger,
	config *config.Config,
	helpers utils.Helpers,
	roundDiscord rounddiscord.RoundDiscordInterface,
	guildConfigResolver guildconfig.GuildConfigResolver,
) Handlers {
	return &RoundHandlers{
		service:             roundDiscord,
		helpers:             helpers,
		config:              config,
		guildConfigResolver: guildConfigResolver,
		logger:              logger,
	}
}

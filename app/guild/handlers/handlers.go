package handlers

import (
	"log/slog"

	discordgocommands "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	guilddiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/guild/discord"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/user/discord/signup"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
)

// GuildHandlers handles guild-related events.
type GuildHandlers struct {
	service             guilddiscord.GuildDiscordInterface
	config              *config.Config
	guildConfigResolver guildconfig.GuildConfigResolver
	signupManager       signup.SignupManager
	interactionStore    storage.ISInterface[any]
	session             discordgocommands.Session
	logger              *slog.Logger
}

// NewGuildHandlers creates a new GuildHandlers.
func NewGuildHandlers(
	logger *slog.Logger,
	config *config.Config,
	guildDiscord guilddiscord.GuildDiscordInterface,
	guildConfigResolver guildconfig.GuildConfigResolver,
	signupManager signup.SignupManager,
	interactionStore storage.ISInterface[any],
	session discordgocommands.Session,
) Handlers {
	return &GuildHandlers{
		service:             guildDiscord,
		config:              config,
		guildConfigResolver: guildConfigResolver,
		signupManager:       signupManager,
		interactionStore:    interactionStore,
		session:             session,
		logger:              logger,
	}
}

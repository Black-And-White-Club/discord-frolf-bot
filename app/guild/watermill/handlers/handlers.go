package handlers

import (
	"log/slog"

	discordgocommands "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	guilddiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/guild/discord"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/user/discord/signup"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"go.opentelemetry.io/otel/trace"
)

// GuildHandlers handles guild-related events.
type GuildHandlers struct {
	Logger              *slog.Logger
	Config              *config.Config
	Helpers             utils.Helpers
	GuildDiscord        guilddiscord.GuildDiscordInterface
	GuildConfigResolver guildconfig.GuildConfigResolver
	SignupManager       signup.SignupManager
	InteractionStore    storage.ISInterface[any]
	Session             discordgocommands.Session
	Tracer              trace.Tracer
	Metrics             discordmetrics.DiscordMetrics
}

// NewGuildHandlers creates a new GuildHandlers.
func NewGuildHandlers(
	logger *slog.Logger,
	config *config.Config,
	helpers utils.Helpers,
	guildDiscord guilddiscord.GuildDiscordInterface,
	guildConfigResolver guildconfig.GuildConfigResolver,
	interactionStore storage.ISInterface[any],
	session discordgocommands.Session,
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
	signupManager signup.SignupManager,
) Handlers {
	return &GuildHandlers{
		Logger:              logger,
		Config:              config,
		Helpers:             helpers,
		GuildDiscord:        guildDiscord,
		GuildConfigResolver: guildConfigResolver,
		SignupManager:       signupManager,
		InteractionStore:    interactionStore,
		Session:             session,
		Tracer:              tracer,
		Metrics:             metrics,
	}
}

package roundhandlers

import (
	"log/slog"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig"
	rounddiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"go.opentelemetry.io/otel/trace"
)

// RoundHandlers handles round-related events.
type RoundHandlers struct {
	Logger            *slog.Logger
	Config            *config.Config
	Helpers           utils.Helpers
	RoundDiscord      rounddiscord.RoundDiscordInterface
	Tracer            trace.Tracer
	Metrics           discordmetrics.DiscordMetrics
	GuildConfigResolver guildconfig.GuildConfigResolver
}

// NewRoundHandlers creates a new RoundHandlers.
func NewRoundHandlers(
	logger *slog.Logger,
	config *config.Config,
	helpers utils.Helpers,
	roundDiscord rounddiscord.RoundDiscordInterface,
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
	guildConfigResolver guildconfig.GuildConfigResolver,
) *RoundHandlers {
	return &RoundHandlers{
		Logger:              logger,
		Config:              config,
		Helpers:             helpers,
		RoundDiscord:        roundDiscord,
		Tracer:              tracer,
		Metrics:             metrics,
		GuildConfigResolver: guildConfigResolver,
	}
}

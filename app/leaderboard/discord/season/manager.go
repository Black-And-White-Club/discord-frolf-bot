package season

import (
	"context"
	"log/slog"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/bwmarrin/discordgo"
	"go.opentelemetry.io/otel/trace"
)

// SeasonManager handles season-related Discord interactions.
type SeasonManager interface {
	HandleSeasonCommand(ctx context.Context, i *discordgo.InteractionCreate)
	HandleSeasonStarted(ctx context.Context, payload *leaderboardevents.StartNewSeasonSuccessPayloadV1)
	HandleSeasonStartFailed(ctx context.Context, payload *leaderboardevents.AdminFailedPayloadV1)
	HandleSeasonStandings(ctx context.Context, payload *leaderboardevents.GetSeasonStandingsResponsePayloadV1)
	HandleSeasonStandingsFailed(ctx context.Context, payload *leaderboardevents.AdminFailedPayloadV1)
}

// seasonManager implements SeasonManager.
type seasonManager struct {
	session             discord.Session
	publisher           eventbus.EventBus
	logger              *slog.Logger
	helper              utils.Helpers
	config              *config.Config
	guildConfigResolver guildconfig.GuildConfigResolver
	interactionStore    storage.ISInterface[any]
	guildConfigCache    storage.ISInterface[storage.GuildConfig]
	tracer              trace.Tracer
	metrics             discordmetrics.DiscordMetrics
}

// NewSeasonManager creates a new SeasonManager.
func NewSeasonManager(
	session discord.Session,
	publisher eventbus.EventBus,
	logger *slog.Logger,
	helper utils.Helpers,
	config *config.Config,
	guildConfigResolver guildconfig.GuildConfigResolver,
	interactionStore storage.ISInterface[any],
	guildConfigCache storage.ISInterface[storage.GuildConfig],
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
) SeasonManager {
	return &seasonManager{
		session:             session,
		publisher:           publisher,
		logger:              logger,
		helper:              helper,
		config:              config,
		guildConfigResolver: guildConfigResolver,
		interactionStore:    interactionStore,
		guildConfigCache:    guildConfigCache,
		tracer:              tracer,
		metrics:             metrics,
	}
}

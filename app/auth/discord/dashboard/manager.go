package dashboard

import (
	"context"
	"log/slog"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/auth/permission"
	discordpkg "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/bwmarrin/discordgo"
	"go.opentelemetry.io/otel/trace"
)

// DashboardManager handles the /dashboard command
type DashboardManager interface {
	HandleDashboardCommand(ctx context.Context, i *discordgo.InteractionCreate) error
}

type dashboardManager struct {
	session             discordpkg.Session
	eventBus            eventbus.EventBus
	logger              *slog.Logger
	cfg                 *config.Config
	tracer              trace.Tracer
	metrics             discordmetrics.DiscordMetrics
	guildConfigResolver guildconfig.GuildConfigResolver
	permMapper          permission.Mapper
	interactionStore    storage.ISInterface[any]
	helper              utils.Helpers
}

// NewDashboardManager creates a new dashboard manager.
func NewDashboardManager(
	session discordpkg.Session,
	eventBus eventbus.EventBus,
	logger *slog.Logger,
	cfg *config.Config,
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
	guildConfigResolver guildconfig.GuildConfigResolver,
	permMapper permission.Mapper,
	interactionStore storage.ISInterface[any],
	helper utils.Helpers,
) DashboardManager {
	return &dashboardManager{
		session:             session,
		eventBus:            eventBus,
		logger:              logger,
		cfg:                 cfg,
		tracer:              tracer,
		metrics:             metrics,
		guildConfigResolver: guildConfigResolver,
		permMapper:          permMapper,
		interactionStore:    interactionStore,
		helper:              helper,
	}
}

package discord

import (
	"context"
	"log/slog"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/auth/discord/dashboard"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/auth/discord/invite"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/auth/permission"
	discordpkg "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"go.opentelemetry.io/otel/trace"
)

// AuthDiscord aggregates auth-related Discord services.
type AuthDiscord struct {
	dashboardManager dashboard.DashboardManager
	inviteManager    invite.InviteManager
}

// NewAuthDiscord creates a new AuthDiscord instance.
func NewAuthDiscord(
	ctx context.Context,
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
) (*AuthDiscord, error) {
	dashboardMgr := dashboard.NewDashboardManager(
		session,
		eventBus,
		logger,
		cfg,
		tracer,
		metrics,
		guildConfigResolver,
		permMapper,
		interactionStore,
		helper,
	)

	inviteMgr := invite.NewInviteManager(session, logger, cfg)

	return &AuthDiscord{
		dashboardManager: dashboardMgr,
		inviteManager:    inviteMgr,
	}, nil
}

// GetDashboardManager returns the dashboard manager.
func (d *AuthDiscord) GetDashboardManager() dashboard.DashboardManager {
	return d.dashboardManager
}

// GetInviteManager returns the invite manager.
func (d *AuthDiscord) GetInviteManager() invite.InviteManager {
	return d.inviteManager
}

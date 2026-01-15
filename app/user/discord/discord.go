package userdiscord

import (
	"context"
	"log/slog"

	discordgo "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/user/discord/role"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/user/discord/signup"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/user/discord/udisc"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"go.opentelemetry.io/otel/trace"
)

// UserDiscordInterface defines the interface for UserDiscord.
type UserDiscordInterface interface {
	GetRoleManager() role.RoleManager
	GetSignupManager() signup.SignupManager
	GetUDiscManager() udisc.UDiscManager
}

// UserDiscord encapsulates all user Discord services.
type UserDiscord struct {
	RoleManager   role.RoleManager
	SignupManager signup.SignupManager
	UDiscManager  udisc.UDiscManager
}

func NewUserDiscord(
	ctx context.Context,
	session discordgo.Session,
	publisher eventbus.EventBus,
	logger *slog.Logger,
	helper utils.Helpers,
	config *config.Config,
	guildConfigResolver guildconfig.GuildConfigResolver,
	interactionStore storage.ISInterface[any],
	guildConfigCache storage.ISInterface[storage.GuildConfig],
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
) (UserDiscordInterface, error) {
	roleManager, err := role.NewRoleManager(session, publisher, logger, helper, config, guildConfigResolver, interactionStore, guildConfigCache, tracer, metrics)
	if err != nil {
		return nil, err
	}

	signupManager, err := signup.NewSignupManager(session, publisher, logger, helper, config, guildConfigResolver, interactionStore, guildConfigCache, tracer, metrics)
	if err != nil {
		return nil, err
	}

	udiscManager := udisc.NewUDiscManager(session, publisher, logger, config, interactionStore, guildConfigCache, tracer, metrics)

	return &UserDiscord{
		RoleManager:   roleManager,
		SignupManager: signupManager,
		UDiscManager:  udiscManager,
	}, nil
}

// GetRoleManager returns the RoleManager.
func (ud *UserDiscord) GetRoleManager() role.RoleManager {
	return ud.RoleManager
}

// GetSignupManager returns the SignupManager.
func (ud *UserDiscord) GetSignupManager() signup.SignupManager {
	return ud.SignupManager
}

// GetUDiscManager returns the UDiscManager.
func (ud *UserDiscord) GetUDiscManager() udisc.UDiscManager {
	return ud.UDiscManager
}

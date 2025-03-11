package userdiscord

import (
	"context"

	discordgo "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/user/discord/role"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/user/discord/signup"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
)

// UserDiscordInterface defines the interface for UserDiscord.
type UserDiscordInterface interface {
	GetRoleManager() role.RoleManager
	GetSignupManager() signup.SignupManager
}

// UserDiscord encapsulates all user Discord services.
type UserDiscord struct {
	RoleManager   role.RoleManager
	SignupManager signup.SignupManager
}

// NewUserDiscord creates a new UserDiscord instance.
func NewUserDiscord(
	ctx context.Context,
	session discordgo.Session,
	publisher eventbus.EventBus,
	logger observability.Logger,
	helper utils.Helpers,
	config *config.Config,
	interactionStore storage.ISInterface,
) (UserDiscordInterface, error) {
	roleManager, err := role.NewRoleManager(session, publisher, logger, helper, config, interactionStore)
	if err != nil {
		return nil, err
	}

	signupManager, err := signup.NewSignupManager(session, publisher, logger, helper, config, interactionStore)
	if err != nil {
		return nil, err
	}

	return &UserDiscord{
		RoleManager:   roleManager,
		SignupManager: signupManager,
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

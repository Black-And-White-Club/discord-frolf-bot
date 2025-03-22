package rounddiscord

import (
	"context"

	discordgo "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	createround "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/create_round"
	deleteround "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/delete_round"
	finalizeround "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/finalize_round"
	roundreminder "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/round_reminder"
	roundrsvp "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/round_rsvp"
	scoreround "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/score_round"
	startround "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/start_round"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
)

// RoundDiscordInterface defines the interface for RoundDiscord.
type RoundDiscordInterface interface {
	GetCreateRoundManager() createround.CreateRoundManager
	GetRoundRsvpManager() roundrsvp.RoundRsvpManager
	GetRoundReminderManager() roundreminder.RoundReminderManager
	GetStartRoundManager() startround.StartRoundManager
	GetScoreRoundManager() scoreround.ScoreRoundManager
	GetFinalizeRoundManager() finalizeround.FinalizeRoundManager
	GetDeleteRoundManager() deleteround.DeleteRoundManager
}

// RoundDiscord encapsulates all Round Discord services.
type RoundDiscord struct {
	CreateRoundManager   createround.CreateRoundManager
	RoundRsvpManager     roundrsvp.RoundRsvpManager
	RoundReminderManager roundreminder.RoundReminderManager
	StartRoundManager    startround.StartRoundManager
	ScoreRoundManager    scoreround.ScoreRoundManager
	FinalizeRoundManager finalizeround.FinalizeRoundManager
	DeleteRoundManager   deleteround.DeleteRoundManager
}

// NewRoundDiscord creates a new RoundDiscord instance.
func NewRoundDiscord(
	ctx context.Context,
	session discordgo.Session,
	publisher eventbus.EventBus,
	logger observability.Logger,
	helper utils.Helpers,
	config *config.Config,
	interactionStore storage.ISInterface,
) (RoundDiscordInterface, error) {
	createRoundManager := createround.NewCreateRoundManager(session, publisher, logger, helper, config, interactionStore)
	roundRsvpManager := roundrsvp.NewRoundRsvpManager(session, publisher, logger, helper, config, interactionStore)
	roundReminderManager := roundreminder.NewRoundReminderManager(session, publisher, logger, helper, config)
	startRoundManager := startround.NewStartRoundManager(session, publisher, logger, helper, config)
	scoreRoundManager := scoreround.NewScoreRoundManager(session, publisher, logger, helper, config)
	finalizeRoundManager := finalizeround.NewFinalizeRoundManager(session, publisher, logger, helper, config)
	deleteRoundManager := deleteround.NewDeleteRoundManager(session, publisher, logger, helper, config)

	return &RoundDiscord{
		CreateRoundManager:   createRoundManager,
		RoundRsvpManager:     roundRsvpManager,
		RoundReminderManager: roundReminderManager,
		StartRoundManager:    startRoundManager,
		ScoreRoundManager:    scoreRoundManager,
		FinalizeRoundManager: finalizeRoundManager,
		DeleteRoundManager:   deleteRoundManager,
	}, nil
}

// GetRoleManager returns the RoleManager.
func (rd *RoundDiscord) GetCreateRoundManager() createround.CreateRoundManager {
	return rd.CreateRoundManager
}

func (rd *RoundDiscord) GetRoundRsvpManager() roundrsvp.RoundRsvpManager {
	return rd.RoundRsvpManager
}

func (rd *RoundDiscord) GetRoundReminderManager() roundreminder.RoundReminderManager {
	return rd.RoundReminderManager
}

func (rd *RoundDiscord) GetStartRoundManager() startround.StartRoundManager {
	return rd.StartRoundManager
}

func (rd *RoundDiscord) GetScoreRoundManager() scoreround.ScoreRoundManager {
	return rd.ScoreRoundManager
}

func (rd *RoundDiscord) GetFinalizeRoundManager() finalizeround.FinalizeRoundManager {
	return rd.FinalizeRoundManager
}

func (rd *RoundDiscord) GetDeleteRoundManager() deleteround.DeleteRoundManager {
	return rd.DeleteRoundManager
}

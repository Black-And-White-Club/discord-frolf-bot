package rounddiscord

import (
	"context"
	"log/slog"

	discordgo "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig"
	createround "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/create_round"
	deleteround "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/delete_round"
	finalizeround "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/finalize_round"
	roundreminder "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/round_reminder"
	roundrsvp "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/round_rsvp"
	scoreround "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/score_round"
	scorecardupload "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/scorecard_upload"
	startround "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/start_round"
	tagupdates "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/tag_updates"
	updateround "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/update_round"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"go.opentelemetry.io/otel/trace"
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
	GetUpdateRoundManager() updateround.UpdateRoundManager
	GetTagUpdateManager() tagupdates.TagUpdateManager
	GetScorecardUploadManager() scorecardupload.ScorecardUploadManager
}

// RoundDiscord encapsulates all Round Discord services.
type RoundDiscord struct {
	CreateRoundManager     createround.CreateRoundManager
	RoundRsvpManager       roundrsvp.RoundRsvpManager
	RoundReminderManager   roundreminder.RoundReminderManager
	StartRoundManager      startround.StartRoundManager
	ScoreRoundManager      scoreround.ScoreRoundManager
	FinalizeRoundManager   finalizeround.FinalizeRoundManager
	DeleteRoundManager     deleteround.DeleteRoundManager
	UpdateRoundManager     updateround.UpdateRoundManager
	TagUpdateManager       tagupdates.TagUpdateManager
	ScorecardUploadManager scorecardupload.ScorecardUploadManager
}

// NewRoundDiscord creates a new RoundDiscord instance.
// It now accepts tracer and metrics dependencies.
func NewRoundDiscord(
	ctx context.Context,
	session discordgo.Session,
	publisher eventbus.EventBus,
	logger *slog.Logger,
	helper utils.Helpers,
	config *config.Config,
	interactionStore storage.ISInterface,
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
	guildConfigResolver guildconfig.GuildConfigResolver,
) (RoundDiscordInterface, error) {
	// Pass the new dependencies to the manager constructors
	createRoundManager := createround.NewCreateRoundManager(session, publisher, logger, helper, config, interactionStore, tracer, metrics, guildConfigResolver)
	roundRsvpManager := roundrsvp.NewRoundRsvpManager(session, publisher, logger, helper, config, interactionStore, tracer, metrics, guildConfigResolver)
	roundReminderManager := roundreminder.NewRoundReminderManager(session, publisher, logger, helper, config, tracer, metrics, guildConfigResolver)
	startRoundManager := startround.NewStartRoundManager(session, publisher, logger, helper, config, tracer, metrics, guildConfigResolver)
	scoreRoundManager := scoreround.NewScoreRoundManager(session, publisher, logger, helper, config, tracer, metrics, guildConfigResolver)
	finalizeRoundManager := finalizeround.NewFinalizeRoundManager(session, publisher, logger, helper, config, tracer, metrics, guildConfigResolver)
	deleteRoundManager := deleteround.NewDeleteRoundManager(session, publisher, logger, helper, config, interactionStore, tracer, metrics, guildConfigResolver)
	updateRoundManager := updateround.NewUpdateRoundManager(session, publisher, logger, helper, config, interactionStore, tracer, metrics, guildConfigResolver)
	tagUpdateManager := tagupdates.NewTagUpdateManager(session, publisher, logger, helper, config, tracer, metrics, guildConfigResolver)
	scorecardUploadManager := scorecardupload.NewScorecardUploadManager(session, publisher, logger, config, tracer, metrics)

	return &RoundDiscord{
		CreateRoundManager:     createRoundManager,
		RoundRsvpManager:       roundRsvpManager,
		RoundReminderManager:   roundReminderManager,
		StartRoundManager:      startRoundManager,
		ScoreRoundManager:      scoreRoundManager,
		FinalizeRoundManager:   finalizeRoundManager,
		DeleteRoundManager:     deleteRoundManager,
		UpdateRoundManager:     updateRoundManager,
		TagUpdateManager:       tagUpdateManager,
		ScorecardUploadManager: scorecardUploadManager,
	}, nil
}

// GetCreateRoundManager returns the CreateRoundManager.
func (rd *RoundDiscord) GetCreateRoundManager() createround.CreateRoundManager {
	return rd.CreateRoundManager
}

// GetRoundRsvpManager returns the RoundRsvpManager.
func (rd *RoundDiscord) GetRoundRsvpManager() roundrsvp.RoundRsvpManager {
	return rd.RoundRsvpManager
}

// GetRoundReminderManager returns the RoundReminderManager.
func (rd *RoundDiscord) GetRoundReminderManager() roundreminder.RoundReminderManager {
	return rd.RoundReminderManager
}

// GetStartRoundManager returns the StartRoundManager.
func (rd *RoundDiscord) GetStartRoundManager() startround.StartRoundManager {
	return rd.StartRoundManager
}

// GetScoreRoundManager returns the ScoreRoundManager.
func (rd *RoundDiscord) GetScoreRoundManager() scoreround.ScoreRoundManager {
	return rd.ScoreRoundManager
}

// GetFinalizeRoundManager returns the FinalizeRoundManager.
func (rd *RoundDiscord) GetFinalizeRoundManager() finalizeround.FinalizeRoundManager {
	return rd.FinalizeRoundManager
}

// GetDeleteRoundManager returns the DeleteRoundManager.
func (rd *RoundDiscord) GetDeleteRoundManager() deleteround.DeleteRoundManager {
	return rd.DeleteRoundManager
}

// GetUpdateRoundManager returns the UpdateRoundManager.
func (rd *RoundDiscord) GetUpdateRoundManager() updateround.UpdateRoundManager {
	return rd.UpdateRoundManager
}

func (rd *RoundDiscord) GetTagUpdateManager() tagupdates.TagUpdateManager {
	return rd.TagUpdateManager
}

// GetScorecardUploadManager returns the ScorecardUploadManager.
func (rd *RoundDiscord) GetScorecardUploadManager() scorecardupload.ScorecardUploadManager {
	return rd.ScorecardUploadManager
}

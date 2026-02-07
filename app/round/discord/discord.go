package rounddiscord

import (
	"context"
	"log/slog"
	"sync"

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
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"go.opentelemetry.io/otel/trace"
)

// Import used in NativeEventMap type definitions
var _ = sharedtypes.RoundID{}

// PendingNativeEventMap tracks user-created Discord events awaiting backend round creation.
type PendingNativeEventMap interface {
	Store(key string, discordEventID string)
	LoadAndDelete(key string) (discordEventID string, ok bool)
}

// DefaultPendingNativeEventMap is a thread-safe map backed by sync.Map.
type DefaultPendingNativeEventMap struct {
	m sync.Map
}

// NewPendingNativeEventMap creates a new PendingNativeEventMap.
func NewPendingNativeEventMap() PendingNativeEventMap {
	return &DefaultPendingNativeEventMap{}
}

// Store saves a discordEventID for a pending key (guildID|title).
func (p *DefaultPendingNativeEventMap) Store(key string, discordEventID string) {
	p.m.Store(key, discordEventID)
}

// LoadAndDelete atomically loads and deletes a pending entry.
func (p *DefaultPendingNativeEventMap) LoadAndDelete(key string) (string, bool) {
	val, ok := p.m.LoadAndDelete(key)
	if !ok {
		return "", false
	}
	return val.(string), true
}

// NativeEventMap defines the interface for resolving Discord Event IDs to Round IDs.
type NativeEventMap interface {
	Store(discordEventID string, roundID sharedtypes.RoundID, guildID sharedtypes.GuildID, creatorID sharedtypes.DiscordID)
	LookupByDiscordEventID(discordEventID string) (roundID sharedtypes.RoundID, guildID sharedtypes.GuildID, creatorID sharedtypes.DiscordID, ok bool)
	LookupByRoundID(roundID sharedtypes.RoundID) (discordEventID string, ok bool)
	Delete(roundID sharedtypes.RoundID)
}

// DefaultNativeEventMap is a thread-safe bidirectional map for resolving
// Discord Event IDs to Round IDs and vice versa.
type DefaultNativeEventMap struct {
	// discordEventIDToRound maps DiscordEventID -> eventMapping
	discordEventIDToRound sync.Map

	// roundIDToDiscordEventID maps RoundID -> DiscordEventID
	roundIDToDiscordEventID sync.Map
}

// NewNativeEventMap creates a new thread-safe NativeEventMap.
func NewNativeEventMap() NativeEventMap {
	return &DefaultNativeEventMap{}
}

// Store adds or updates a mapping from DiscordEventID to RoundID.
func (m *DefaultNativeEventMap) Store(discordEventID string, roundID sharedtypes.RoundID, guildID sharedtypes.GuildID, creatorID sharedtypes.DiscordID) {
	m.discordEventIDToRound.Store(discordEventID, &eventMapping{
		roundID:   roundID,
		guildID:   guildID,
		creatorID: creatorID,
	})
	m.roundIDToDiscordEventID.Store(roundID.String(), discordEventID)
}

// LookupByDiscordEventID looks up a RoundID and GuildID by DiscordEventID.
// Returns the RoundID, GuildID, CreatorID and a boolean indicating if the lookup was successful.
func (m *DefaultNativeEventMap) LookupByDiscordEventID(discordEventID string) (sharedtypes.RoundID, sharedtypes.GuildID, sharedtypes.DiscordID, bool) {
	val, ok := m.discordEventIDToRound.Load(discordEventID)
	if !ok {
		return sharedtypes.RoundID{}, sharedtypes.GuildID(""), sharedtypes.DiscordID(""), false
	}
	mapping := val.(*eventMapping)
	return mapping.roundID, mapping.guildID, mapping.creatorID, true
}

// LookupByRoundID looks up a DiscordEventID by RoundID.
// Returns the DiscordEventID and a boolean indicating if the lookup was successful.
func (m *DefaultNativeEventMap) LookupByRoundID(roundID sharedtypes.RoundID) (string, bool) {
	val, ok := m.roundIDToDiscordEventID.Load(roundID.String())
	if !ok {
		return "", false
	}
	return val.(string), true
}

// Delete removes all mappings for a given RoundID.
func (m *DefaultNativeEventMap) Delete(roundID sharedtypes.RoundID) {
	// Look up the DiscordEventID first
	discordEventID, ok := m.LookupByRoundID(roundID)
	if ok {
		m.discordEventIDToRound.Delete(discordEventID)
	}
	// Delete the RoundID mapping
	m.roundIDToDiscordEventID.Delete(roundID.String())
}

// eventMapping holds the round and guild information for a native event.
type eventMapping struct {
	roundID   sharedtypes.RoundID
	guildID   sharedtypes.GuildID
	creatorID sharedtypes.DiscordID
}

// MessageMap defines the interface for storing Round Message IDs.
type MessageMap interface {
	Store(roundID sharedtypes.RoundID, messageID string)
	Load(roundID sharedtypes.RoundID) (string, bool)
	Delete(roundID sharedtypes.RoundID)
}

// DefaultMessageMap is a thread-safe map for storing Round Message IDs.
type DefaultMessageMap struct {
	// roundIDToMessageID maps RoundID -> MessageID
	roundIDToMessageID sync.Map
}

// NewMessageMap creates a new thread-safe MessageMap.
func NewMessageMap() MessageMap {
	return &DefaultMessageMap{}
}

// Store saves a MessageID for a RoundID.
func (m *DefaultMessageMap) Store(roundID sharedtypes.RoundID, messageID string) {
	m.roundIDToMessageID.Store(roundID.String(), messageID)
}

// Load retrieves a MessageID for a RoundID.
func (m *DefaultMessageMap) Load(roundID sharedtypes.RoundID) (string, bool) {
	val, ok := m.roundIDToMessageID.Load(roundID.String())
	if !ok {
		return "", false
	}
	return val.(string), true
}

// Delete removes a mapping for a RoundID.
func (m *DefaultMessageMap) Delete(roundID sharedtypes.RoundID) {
	m.roundIDToMessageID.Delete(roundID.String())
}

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
	GetSession() discordgo.Session
	GetNativeEventMap() NativeEventMap
	GetMessageMap() MessageMap
	GetPendingNativeEventMap() PendingNativeEventMap
}

// RoundDiscord encapsulates all Round Discord services.
type RoundDiscord struct {
	session                discordgo.Session
	nativeEventMap         NativeEventMap
	messageMap             MessageMap
	pendingNativeEventMap  PendingNativeEventMap
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
	interactionStore storage.ISInterface[any],
	guildConfigCache storage.ISInterface[storage.GuildConfig],
	guildConfigResolver guildconfig.GuildConfigResolver,
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
) (RoundDiscordInterface, error) {
	// Pass the new dependencies to the manager constructors
	createRoundManager := createround.NewCreateRoundManager(session, publisher, logger, helper, config, interactionStore, guildConfigCache, tracer, metrics, guildConfigResolver)
	roundRsvpManager := roundrsvp.NewRoundRsvpManager(session, publisher, logger, helper, config, interactionStore, guildConfigCache, tracer, metrics, guildConfigResolver)
	roundReminderManager := roundreminder.NewRoundReminderManager(session, publisher, logger, helper, config, interactionStore, guildConfigCache, tracer, metrics, guildConfigResolver)
	startRoundManager := startround.NewStartRoundManager(session, publisher, logger, helper, config, interactionStore, guildConfigCache, tracer, metrics, guildConfigResolver)
	scoreRoundManager := scoreround.NewScoreRoundManager(session, publisher, logger, helper, config, interactionStore, guildConfigCache, tracer, metrics, guildConfigResolver)
	finalizeRoundManager := finalizeround.NewFinalizeRoundManager(session, publisher, logger, helper, config, interactionStore, guildConfigCache, tracer, metrics, guildConfigResolver)
	deleteRoundManager := deleteround.NewDeleteRoundManager(session, publisher, logger, helper, config, interactionStore, guildConfigCache, tracer, metrics, guildConfigResolver)
	updateRoundManager := updateround.NewUpdateRoundManager(session, publisher, logger, helper, config, interactionStore, guildConfigCache, tracer, metrics, guildConfigResolver)
	tagUpdateManager := tagupdates.NewTagUpdateManager(session, publisher, logger, helper, config, interactionStore, guildConfigCache, tracer, metrics, guildConfigResolver)
	scorecardUploadManager := scorecardupload.NewScorecardUploadManager(ctx, session, publisher, logger, config, interactionStore, guildConfigCache, tracer, metrics)

	return &RoundDiscord{
		session:                session,
		nativeEventMap:         NewNativeEventMap(),
		messageMap:             NewMessageMap(),
		pendingNativeEventMap:  NewPendingNativeEventMap(),
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

// GetSession returns the Discord session.
func (rd *RoundDiscord) GetSession() discordgo.Session {
	return rd.session
}

// GetNativeEventMap returns the NativeEventMap for resolving Discord Event IDs.
func (rd *RoundDiscord) GetNativeEventMap() NativeEventMap {
	return rd.nativeEventMap
}

// GetMessageMap returns the MessageMap for resolving Round Message IDs.
func (rd *RoundDiscord) GetMessageMap() MessageMap {
	return rd.messageMap
}

// GetPendingNativeEventMap returns the PendingNativeEventMap for tracking user-created Discord events.
func (rd *RoundDiscord) GetPendingNativeEventMap() PendingNativeEventMap {
	return rd.pendingNativeEventMap
}

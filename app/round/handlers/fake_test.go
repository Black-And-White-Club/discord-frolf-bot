package handlers

import (
	"context"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	rounddiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord"
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
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
)

// FakeRoundDiscord is a programmable fake for RoundDiscordInterface
type FakeRoundDiscord struct {
	GetCreateRoundManagerFunc     func() createround.CreateRoundManager
	GetRoundRsvpManagerFunc       func() roundrsvp.RoundRsvpManager
	GetRoundReminderManagerFunc   func() roundreminder.RoundReminderManager
	GetStartRoundManagerFunc      func() startround.StartRoundManager
	GetScoreRoundManagerFunc      func() scoreround.ScoreRoundManager
	GetFinalizeRoundManagerFunc   func() finalizeround.FinalizeRoundManager
	GetDeleteRoundManagerFunc     func() deleteround.DeleteRoundManager
	GetUpdateRoundManagerFunc     func() updateround.UpdateRoundManager
	GetTagUpdateManagerFunc       func() tagupdates.TagUpdateManager
	GetScorecardUploadManagerFunc func() scorecardupload.ScorecardUploadManager
	GetMessageMapFunc             func() rounddiscord.MessageMap

	// Holds the sub-fakes
	CreateRoundManager     FakeCreateRoundManager
	RoundRsvpManager       FakeRoundRsvpManager
	RoundReminderManager   FakeRoundReminderManager
	StartRoundManager      FakeStartRoundManager
	ScoreRoundManager      FakeScoreRoundManager
	FinalizeRoundManager   FakeFinalizeRoundManager
	DeleteRoundManager     FakeDeleteRoundManager
	UpdateRoundManager     FakeUpdateRoundManager
	TagUpdateManager       FakeTagUpdateManager
	ScorecardUploadManager FakeScorecardUploadManager
	MessageMap             FakeMessageMap
}

func (f *FakeRoundDiscord) GetCreateRoundManager() createround.CreateRoundManager {
	if f.GetCreateRoundManagerFunc != nil {
		return f.GetCreateRoundManagerFunc()
	}
	return &f.CreateRoundManager
}

func (f *FakeRoundDiscord) GetRoundRsvpManager() roundrsvp.RoundRsvpManager {
	if f.GetRoundRsvpManagerFunc != nil {
		return f.GetRoundRsvpManagerFunc()
	}
	return &f.RoundRsvpManager
}

func (f *FakeRoundDiscord) GetRoundReminderManager() roundreminder.RoundReminderManager {
	if f.GetRoundReminderManagerFunc != nil {
		return f.GetRoundReminderManagerFunc()
	}
	return &f.RoundReminderManager
}

func (f *FakeRoundDiscord) GetStartRoundManager() startround.StartRoundManager {
	if f.GetStartRoundManagerFunc != nil {
		return f.GetStartRoundManagerFunc()
	}
	return &f.StartRoundManager
}

func (f *FakeRoundDiscord) GetScoreRoundManager() scoreround.ScoreRoundManager {
	if f.GetScoreRoundManagerFunc != nil {
		return f.GetScoreRoundManagerFunc()
	}
	return &f.ScoreRoundManager
}

func (f *FakeRoundDiscord) GetFinalizeRoundManager() finalizeround.FinalizeRoundManager {
	if f.GetFinalizeRoundManagerFunc != nil {
		return f.GetFinalizeRoundManagerFunc()
	}
	return &f.FinalizeRoundManager
}

func (f *FakeRoundDiscord) GetDeleteRoundManager() deleteround.DeleteRoundManager {
	if f.GetDeleteRoundManagerFunc != nil {
		return f.GetDeleteRoundManagerFunc()
	}
	return &f.DeleteRoundManager
}

func (f *FakeRoundDiscord) GetUpdateRoundManager() updateround.UpdateRoundManager {
	if f.GetUpdateRoundManagerFunc != nil {
		return f.GetUpdateRoundManagerFunc()
	}
	return &f.UpdateRoundManager
}

func (f *FakeRoundDiscord) GetTagUpdateManager() tagupdates.TagUpdateManager {
	if f.GetTagUpdateManagerFunc != nil {
		return f.GetTagUpdateManagerFunc()
	}
	return &f.TagUpdateManager
}

func (f *FakeRoundDiscord) GetScorecardUploadManager() scorecardupload.ScorecardUploadManager {
	if f.GetScorecardUploadManagerFunc != nil {
		return f.GetScorecardUploadManagerFunc()
	}
	return &f.ScorecardUploadManager
}

func (f *FakeRoundDiscord) GetSession() discord.Session {
	return nil
}

func (f *FakeRoundDiscord) GetNativeEventMap() rounddiscord.NativeEventMap {
	return nil
}

func (f *FakeRoundDiscord) GetPendingNativeEventMap() rounddiscord.PendingNativeEventMap {
	return rounddiscord.NewPendingNativeEventMap()
}

func (f *FakeRoundDiscord) GetMessageMap() rounddiscord.MessageMap {
	if f.GetMessageMapFunc != nil {
		return f.GetMessageMapFunc()
	}
	return &f.MessageMap
}

// FakeMessageMap
type FakeMessageMap struct {
	StoreFunc  func(roundID sharedtypes.RoundID, messageID string)
	LoadFunc   func(roundID sharedtypes.RoundID) (string, bool)
	DeleteFunc func(roundID sharedtypes.RoundID)
}

func (f *FakeMessageMap) Store(roundID sharedtypes.RoundID, messageID string) {
	if f.StoreFunc != nil {
		f.StoreFunc(roundID, messageID)
	}
}

func (f *FakeMessageMap) Load(roundID sharedtypes.RoundID) (string, bool) {
	if f.LoadFunc != nil {
		return f.LoadFunc(roundID)
	}
	return "", false
}

func (f *FakeMessageMap) Delete(roundID sharedtypes.RoundID) {
	if f.DeleteFunc != nil {
		f.DeleteFunc(roundID)
	}
}

// FakeCreateRoundManager
type FakeCreateRoundManager struct {
	HandleCreateRoundCommandFunc                 func(ctx context.Context, i *discordgo.InteractionCreate) (createround.CreateRoundOperationResult, error)
	HandleCreateRoundModalSubmitFunc             func(ctx context.Context, i *discordgo.InteractionCreate) (createround.CreateRoundOperationResult, error)
	UpdateInteractionResponseFunc                func(ctx context.Context, correlationID, message string, edit ...*discordgo.WebhookEdit) (createround.CreateRoundOperationResult, error)
	UpdateInteractionResponseWithRetryButtonFunc func(ctx context.Context, correlationID, message string) (createround.CreateRoundOperationResult, error)
	HandleCreateRoundModalCancelFunc             func(ctx context.Context, i *discordgo.InteractionCreate) (createround.CreateRoundOperationResult, error)
	SendRoundEventEmbedFunc                      func(guildID string, channelID string, title roundtypes.Title, description roundtypes.Description, startTime sharedtypes.StartTime, location roundtypes.Location, creatorID sharedtypes.DiscordID, roundID sharedtypes.RoundID) (createround.CreateRoundOperationResult, error)
	SendCreateRoundModalFunc                     func(ctx context.Context, i *discordgo.InteractionCreate) (createround.CreateRoundOperationResult, error)
	HandleRetryCreateRoundFunc                   func(ctx context.Context, i *discordgo.InteractionCreate) (createround.CreateRoundOperationResult, error)
}

func (f *FakeCreateRoundManager) HandleCreateRoundCommand(ctx context.Context, i *discordgo.InteractionCreate) (createround.CreateRoundOperationResult, error) {
	if f.HandleCreateRoundCommandFunc != nil {
		return f.HandleCreateRoundCommandFunc(ctx, i)
	}
	return createround.CreateRoundOperationResult{}, nil
}

func (f *FakeCreateRoundManager) HandleCreateRoundModalSubmit(ctx context.Context, i *discordgo.InteractionCreate) (createround.CreateRoundOperationResult, error) {
	if f.HandleCreateRoundModalSubmitFunc != nil {
		return f.HandleCreateRoundModalSubmitFunc(ctx, i)
	}
	return createround.CreateRoundOperationResult{}, nil
}

func (f *FakeCreateRoundManager) UpdateInteractionResponse(ctx context.Context, correlationID, message string, edit ...*discordgo.WebhookEdit) (createround.CreateRoundOperationResult, error) {
	if f.UpdateInteractionResponseFunc != nil {
		return f.UpdateInteractionResponseFunc(ctx, correlationID, message, edit...)
	}
	return createround.CreateRoundOperationResult{}, nil
}

func (f *FakeCreateRoundManager) UpdateInteractionResponseWithRetryButton(ctx context.Context, correlationID, message string) (createround.CreateRoundOperationResult, error) {
	if f.UpdateInteractionResponseWithRetryButtonFunc != nil {
		return f.UpdateInteractionResponseWithRetryButtonFunc(ctx, correlationID, message)
	}
	return createround.CreateRoundOperationResult{}, nil
}

func (f *FakeCreateRoundManager) HandleCreateRoundModalCancel(ctx context.Context, i *discordgo.InteractionCreate) (createround.CreateRoundOperationResult, error) {
	if f.HandleCreateRoundModalCancelFunc != nil {
		return f.HandleCreateRoundModalCancelFunc(ctx, i)
	}
	return createround.CreateRoundOperationResult{}, nil
}

func (f *FakeCreateRoundManager) SendRoundEventEmbed(guildID string, channelID string, title roundtypes.Title, description roundtypes.Description, startTime sharedtypes.StartTime, location roundtypes.Location, creatorID sharedtypes.DiscordID, roundID sharedtypes.RoundID) (createround.CreateRoundOperationResult, error) {
	if f.SendRoundEventEmbedFunc != nil {
		return f.SendRoundEventEmbedFunc(guildID, channelID, title, description, startTime, location, creatorID, roundID)
	}
	return createround.CreateRoundOperationResult{}, nil
}

func (f *FakeCreateRoundManager) SendCreateRoundModal(ctx context.Context, i *discordgo.InteractionCreate) (createround.CreateRoundOperationResult, error) {
	if f.SendCreateRoundModalFunc != nil {
		return f.SendCreateRoundModalFunc(ctx, i)
	}
	return createround.CreateRoundOperationResult{}, nil
}

func (f *FakeCreateRoundManager) HandleRetryCreateRound(ctx context.Context, i *discordgo.InteractionCreate) (createround.CreateRoundOperationResult, error) {
	if f.HandleRetryCreateRoundFunc != nil {
		return f.HandleRetryCreateRoundFunc(ctx, i)
	}
	return createround.CreateRoundOperationResult{}, nil
}

// FakeRoundRsvpManager
type FakeRoundRsvpManager struct {
	HandleRoundResponseFunc      func(ctx context.Context, i *discordgo.InteractionCreate) (roundrsvp.RoundRsvpOperationResult, error)
	HandleRsvpJoinButtonFunc     func(ctx context.Context, i *discordgo.InteractionCreate) (roundrsvp.RoundRsvpOperationResult, error)
	HandleRsvpLeaveButtonFunc    func(ctx context.Context, i *discordgo.InteractionCreate) (roundrsvp.RoundRsvpOperationResult, error)
	UpdateRoundEventEmbedFunc    func(ctx context.Context, channelID, messageID string, participants []roundtypes.Participant) (roundrsvp.RoundRsvpOperationResult, error)
	InteractionJoinRoundLateFunc func(ctx context.Context, i *discordgo.InteractionCreate) (roundrsvp.RoundRsvpOperationResult, error)
}

func (f *FakeRoundRsvpManager) HandleRoundResponse(ctx context.Context, i *discordgo.InteractionCreate) (roundrsvp.RoundRsvpOperationResult, error) {
	if f.HandleRoundResponseFunc != nil {
		return f.HandleRoundResponseFunc(ctx, i)
	}
	return roundrsvp.RoundRsvpOperationResult{}, nil
}

func (f *FakeRoundRsvpManager) HandleRsvpJoinButton(ctx context.Context, i *discordgo.InteractionCreate) (roundrsvp.RoundRsvpOperationResult, error) {
	if f.HandleRsvpJoinButtonFunc != nil {
		return f.HandleRsvpJoinButtonFunc(ctx, i)
	}
	return roundrsvp.RoundRsvpOperationResult{}, nil
}

func (f *FakeRoundRsvpManager) HandleRsvpLeaveButton(ctx context.Context, i *discordgo.InteractionCreate) (roundrsvp.RoundRsvpOperationResult, error) {
	if f.HandleRsvpLeaveButtonFunc != nil {
		return f.HandleRsvpLeaveButtonFunc(ctx, i)
	}
	return roundrsvp.RoundRsvpOperationResult{}, nil
}

func (f *FakeRoundRsvpManager) UpdateRoundEventEmbed(ctx context.Context, channelID, messageID string, participants []roundtypes.Participant) (roundrsvp.RoundRsvpOperationResult, error) {
	if f.UpdateRoundEventEmbedFunc != nil {
		return f.UpdateRoundEventEmbedFunc(ctx, channelID, messageID, participants)
	}
	return roundrsvp.RoundRsvpOperationResult{}, nil
}

func (f *FakeRoundRsvpManager) InteractionJoinRoundLate(ctx context.Context, i *discordgo.InteractionCreate) (roundrsvp.RoundRsvpOperationResult, error) {
	if f.InteractionJoinRoundLateFunc != nil {
		return f.InteractionJoinRoundLateFunc(ctx, i)
	}
	return roundrsvp.RoundRsvpOperationResult{}, nil
}

// FakeRoundReminderManager
type FakeRoundReminderManager struct {
	SendRoundReminderFunc func(ctx context.Context, payload *roundevents.DiscordReminderPayloadV1) (roundreminder.RoundReminderOperationResult, error)
}

func (f *FakeRoundReminderManager) SendRoundReminder(ctx context.Context, payload *roundevents.DiscordReminderPayloadV1) (roundreminder.RoundReminderOperationResult, error) {
	if f.SendRoundReminderFunc != nil {
		return f.SendRoundReminderFunc(ctx, payload)
	}
	return roundreminder.RoundReminderOperationResult{}, nil
}

// FakeStartRoundManager
type FakeStartRoundManager struct {
	TransformRoundToScorecardFunc func(ctx context.Context, payload *roundevents.DiscordRoundStartPayloadV1, existingEmbed *discordgo.MessageEmbed) (startround.StartRoundOperationResult, error)
	UpdateRoundToScorecardFunc    func(ctx context.Context, channelID, messageID string, payload *roundevents.DiscordRoundStartPayloadV1) (startround.StartRoundOperationResult, error)
}

func (f *FakeStartRoundManager) TransformRoundToScorecard(ctx context.Context, payload *roundevents.DiscordRoundStartPayloadV1, existingEmbed *discordgo.MessageEmbed) (startround.StartRoundOperationResult, error) {
	if f.TransformRoundToScorecardFunc != nil {
		return f.TransformRoundToScorecardFunc(ctx, payload, existingEmbed)
	}
	return startround.StartRoundOperationResult{}, nil
}

func (f *FakeStartRoundManager) UpdateRoundToScorecard(ctx context.Context, channelID, messageID string, payload *roundevents.DiscordRoundStartPayloadV1) (startround.StartRoundOperationResult, error) {
	if f.UpdateRoundToScorecardFunc != nil {
		return f.UpdateRoundToScorecardFunc(ctx, channelID, messageID, payload)
	}
	return startround.StartRoundOperationResult{}, nil
}

// FakeScoreRoundManager
type FakeScoreRoundManager struct {
	HandleScoreButtonFunc             func(ctx context.Context, i *discordgo.InteractionCreate) (scoreround.ScoreRoundOperationResult, error)
	HandleScoreSubmissionFunc         func(ctx context.Context, i *discordgo.InteractionCreate) (scoreround.ScoreRoundOperationResult, error)
	SendScoreUpdateConfirmationFunc   func(ctx context.Context, channelID string, userID sharedtypes.DiscordID, score *sharedtypes.Score) (scoreround.ScoreRoundOperationResult, error)
	SendScoreUpdateErrorFunc          func(ctx context.Context, userID sharedtypes.DiscordID, errorMsg string) (scoreround.ScoreRoundOperationResult, error)
	UpdateScoreEmbedFunc              func(ctx context.Context, channelID, messageID string, userID sharedtypes.DiscordID, score *sharedtypes.Score) (scoreround.ScoreRoundOperationResult, error)
	UpdateScoreEmbedBulkFunc          func(ctx context.Context, channelID, messageID string, participants []roundtypes.Participant) (scoreround.ScoreRoundOperationResult, error)
	AddLateParticipantToScorecardFunc func(ctx context.Context, channelID, messageID string, participants []roundtypes.Participant) (scoreround.ScoreRoundOperationResult, error)
}

func (f *FakeScoreRoundManager) HandleScoreButton(ctx context.Context, i *discordgo.InteractionCreate) (scoreround.ScoreRoundOperationResult, error) {
	if f.HandleScoreButtonFunc != nil {
		return f.HandleScoreButtonFunc(ctx, i)
	}
	return scoreround.ScoreRoundOperationResult{}, nil
}

func (f *FakeScoreRoundManager) HandleScoreSubmission(ctx context.Context, i *discordgo.InteractionCreate) (scoreround.ScoreRoundOperationResult, error) {
	if f.HandleScoreSubmissionFunc != nil {
		return f.HandleScoreSubmissionFunc(ctx, i)
	}
	return scoreround.ScoreRoundOperationResult{}, nil
}

func (f *FakeScoreRoundManager) SendScoreUpdateConfirmation(ctx context.Context, channelID string, userID sharedtypes.DiscordID, score *sharedtypes.Score) (scoreround.ScoreRoundOperationResult, error) {
	if f.SendScoreUpdateConfirmationFunc != nil {
		return f.SendScoreUpdateConfirmationFunc(ctx, channelID, userID, score)
	}
	return scoreround.ScoreRoundOperationResult{}, nil
}

func (f *FakeScoreRoundManager) SendScoreUpdateError(ctx context.Context, userID sharedtypes.DiscordID, errorMsg string) (scoreround.ScoreRoundOperationResult, error) {
	if f.SendScoreUpdateErrorFunc != nil {
		return f.SendScoreUpdateErrorFunc(ctx, userID, errorMsg)
	}
	return scoreround.ScoreRoundOperationResult{}, nil
}

func (f *FakeScoreRoundManager) UpdateScoreEmbed(ctx context.Context, channelID, messageID string, userID sharedtypes.DiscordID, score *sharedtypes.Score) (scoreround.ScoreRoundOperationResult, error) {
	if f.UpdateScoreEmbedFunc != nil {
		return f.UpdateScoreEmbedFunc(ctx, channelID, messageID, userID, score)
	}
	return scoreround.ScoreRoundOperationResult{}, nil
}

func (f *FakeScoreRoundManager) UpdateScoreEmbedBulk(ctx context.Context, channelID, messageID string, participants []roundtypes.Participant) (scoreround.ScoreRoundOperationResult, error) {
	if f.UpdateScoreEmbedBulkFunc != nil {
		return f.UpdateScoreEmbedBulkFunc(ctx, channelID, messageID, participants)
	}
	return scoreround.ScoreRoundOperationResult{}, nil
}

func (f *FakeScoreRoundManager) AddLateParticipantToScorecard(ctx context.Context, channelID, messageID string, participants []roundtypes.Participant) (scoreround.ScoreRoundOperationResult, error) {
	if f.AddLateParticipantToScorecardFunc != nil {
		return f.AddLateParticipantToScorecardFunc(ctx, channelID, messageID, participants)
	}
	return scoreround.ScoreRoundOperationResult{}, nil
}

// FakeFinalizeRoundManager
type FakeFinalizeRoundManager struct {
	TransformRoundToFinalizedScorecardFunc func(payload roundevents.RoundFinalizedEmbedUpdatePayloadV1) (*discordgo.MessageEmbed, []discordgo.MessageComponent, error)
	FinalizeScorecardEmbedFunc             func(ctx context.Context, eventMessageID string, channelID string, embedPayload roundevents.RoundFinalizedEmbedUpdatePayloadV1) (finalizeround.FinalizeRoundOperationResult, error)
}

func (f *FakeFinalizeRoundManager) TransformRoundToFinalizedScorecard(payload roundevents.RoundFinalizedEmbedUpdatePayloadV1) (*discordgo.MessageEmbed, []discordgo.MessageComponent, error) {
	if f.TransformRoundToFinalizedScorecardFunc != nil {
		return f.TransformRoundToFinalizedScorecardFunc(payload)
	}
	return nil, nil, nil
}

func (f *FakeFinalizeRoundManager) FinalizeScorecardEmbed(ctx context.Context, eventMessageID string, channelID string, embedPayload roundevents.RoundFinalizedEmbedUpdatePayloadV1) (finalizeround.FinalizeRoundOperationResult, error) {
	if f.FinalizeScorecardEmbedFunc != nil {
		return f.FinalizeScorecardEmbedFunc(ctx, eventMessageID, channelID, embedPayload)
	}
	return finalizeround.FinalizeRoundOperationResult{}, nil
}

// FakeDeleteRoundManager
type FakeDeleteRoundManager struct {
	HandleDeleteRoundButtonFunc func(ctx context.Context, i *discordgo.InteractionCreate) (deleteround.DeleteRoundOperationResult, error)
	DeleteRoundEventEmbedFunc   func(ctx context.Context, discordMessageID string, channelID string) (deleteround.DeleteRoundOperationResult, error)
}

func (f *FakeDeleteRoundManager) HandleDeleteRoundButton(ctx context.Context, i *discordgo.InteractionCreate) (deleteround.DeleteRoundOperationResult, error) {
	if f.HandleDeleteRoundButtonFunc != nil {
		return f.HandleDeleteRoundButtonFunc(ctx, i)
	}
	return deleteround.DeleteRoundOperationResult{}, nil
}

func (f *FakeDeleteRoundManager) DeleteRoundEventEmbed(ctx context.Context, discordMessageID string, channelID string) (deleteround.DeleteRoundOperationResult, error) {
	if f.DeleteRoundEventEmbedFunc != nil {
		return f.DeleteRoundEventEmbedFunc(ctx, discordMessageID, channelID)
	}
	return deleteround.DeleteRoundOperationResult{}, nil
}

// FakeUpdateRoundManager
type FakeUpdateRoundManager struct {
	UpdateRoundEventEmbedFunc        func(ctx context.Context, channelID string, messageID string, title *roundtypes.Title, description *roundtypes.Description, startTime *sharedtypes.StartTime, location *roundtypes.Location) (updateround.UpdateRoundOperationResult, error)
	HandleEditRoundButtonFunc        func(ctx context.Context, i *discordgo.InteractionCreate) (updateround.UpdateRoundOperationResult, error)
	SendUpdateRoundModalFunc         func(ctx context.Context, i *discordgo.InteractionCreate, roundID sharedtypes.RoundID) (updateround.UpdateRoundOperationResult, error)
	HandleUpdateRoundModalSubmitFunc func(ctx context.Context, i *discordgo.InteractionCreate) (updateround.UpdateRoundOperationResult, error)
	HandleUpdateRoundModalCancelFunc func(ctx context.Context, i *discordgo.InteractionCreate) (updateround.UpdateRoundOperationResult, error)
}

func (f *FakeUpdateRoundManager) UpdateRoundEventEmbed(ctx context.Context, channelID string, messageID string, title *roundtypes.Title, description *roundtypes.Description, startTime *sharedtypes.StartTime, location *roundtypes.Location) (updateround.UpdateRoundOperationResult, error) {
	if f.UpdateRoundEventEmbedFunc != nil {
		return f.UpdateRoundEventEmbedFunc(ctx, channelID, messageID, title, description, startTime, location)
	}
	return updateround.UpdateRoundOperationResult{}, nil
}

func (f *FakeUpdateRoundManager) HandleEditRoundButton(ctx context.Context, i *discordgo.InteractionCreate) (updateround.UpdateRoundOperationResult, error) {
	if f.HandleEditRoundButtonFunc != nil {
		return f.HandleEditRoundButtonFunc(ctx, i)
	}
	return updateround.UpdateRoundOperationResult{}, nil
}

func (f *FakeUpdateRoundManager) SendUpdateRoundModal(ctx context.Context, i *discordgo.InteractionCreate, roundID sharedtypes.RoundID) (updateround.UpdateRoundOperationResult, error) {
	if f.SendUpdateRoundModalFunc != nil {
		return f.SendUpdateRoundModalFunc(ctx, i, roundID)
	}
	return updateround.UpdateRoundOperationResult{}, nil
}

func (f *FakeUpdateRoundManager) HandleUpdateRoundModalSubmit(ctx context.Context, i *discordgo.InteractionCreate) (updateround.UpdateRoundOperationResult, error) {
	if f.HandleUpdateRoundModalSubmitFunc != nil {
		return f.HandleUpdateRoundModalSubmitFunc(ctx, i)
	}
	return updateround.UpdateRoundOperationResult{}, nil
}

func (f *FakeUpdateRoundManager) HandleUpdateRoundModalCancel(ctx context.Context, i *discordgo.InteractionCreate) (updateround.UpdateRoundOperationResult, error) {
	if f.HandleUpdateRoundModalCancelFunc != nil {
		return f.HandleUpdateRoundModalCancelFunc(ctx, i)
	}
	return updateround.UpdateRoundOperationResult{}, nil
}

// FakeTagUpdateManager
type FakeTagUpdateManager struct {
	UpdateDiscordEmbedsWithTagChangesFunc func(ctx context.Context, payload roundevents.ScheduledRoundsSyncedPayloadV1, tagUpdates map[sharedtypes.DiscordID]*sharedtypes.TagNumber) (tagupdates.TagUpdateOperationResult, error)
	UpdateTagsInEmbedFunc                 func(ctx context.Context, channelID, messageID string, tagUpdates map[sharedtypes.DiscordID]*sharedtypes.TagNumber) (tagupdates.TagUpdateOperationResult, error)
}

func (f *FakeTagUpdateManager) UpdateDiscordEmbedsWithTagChanges(ctx context.Context, payload roundevents.ScheduledRoundsSyncedPayloadV1, tagUpdates map[sharedtypes.DiscordID]*sharedtypes.TagNumber) (tagupdates.TagUpdateOperationResult, error) {
	if f.UpdateDiscordEmbedsWithTagChangesFunc != nil {
		return f.UpdateDiscordEmbedsWithTagChangesFunc(ctx, payload, tagUpdates)
	}
	return tagupdates.TagUpdateOperationResult{}, nil
}

func (f *FakeTagUpdateManager) UpdateTagsInEmbed(ctx context.Context, channelID, messageID string, tagUpdates map[sharedtypes.DiscordID]*sharedtypes.TagNumber) (tagupdates.TagUpdateOperationResult, error) {
	if f.UpdateTagsInEmbedFunc != nil {
		return f.UpdateTagsInEmbedFunc(ctx, channelID, messageID, tagUpdates)
	}
	return tagupdates.TagUpdateOperationResult{}, nil
}

// FakeScorecardUploadManager
type FakeScorecardUploadManager struct {
	HandleScorecardUploadButtonFunc      func(ctx context.Context, i *discordgo.InteractionCreate) (scorecardupload.ScorecardUploadOperationResult, error)
	HandleScorecardUploadModalSubmitFunc func(ctx context.Context, i *discordgo.InteractionCreate) (scorecardupload.ScorecardUploadOperationResult, error)
	HandleFileUploadMessageFunc          func(s discord.Session, m *discordgo.MessageCreate)
	SendUploadErrorFunc                  func(ctx context.Context, channelID, userID, errorMsg string) error
}

func (f *FakeScorecardUploadManager) HandleScorecardUploadButton(ctx context.Context, i *discordgo.InteractionCreate) (scorecardupload.ScorecardUploadOperationResult, error) {
	if f.HandleScorecardUploadButtonFunc != nil {
		return f.HandleScorecardUploadButtonFunc(ctx, i)
	}
	return scorecardupload.ScorecardUploadOperationResult{}, nil
}

func (f *FakeScorecardUploadManager) HandleScorecardUploadModalSubmit(ctx context.Context, i *discordgo.InteractionCreate) (scorecardupload.ScorecardUploadOperationResult, error) {
	if f.HandleScorecardUploadModalSubmitFunc != nil {
		return f.HandleScorecardUploadModalSubmitFunc(ctx, i)
	}
	return scorecardupload.ScorecardUploadOperationResult{}, nil
}

func (f *FakeScorecardUploadManager) HandleFileUploadMessage(s discord.Session, m *discordgo.MessageCreate) {
	if f.HandleFileUploadMessageFunc != nil {
		f.HandleFileUploadMessageFunc(s, m)
	}
}

func (f *FakeScorecardUploadManager) SendUploadError(ctx context.Context, channelID, userID, errorMsg string) error {
	if f.SendUploadErrorFunc != nil {
		return f.SendUploadErrorFunc(ctx, channelID, userID, errorMsg)
	}
	return nil
}

// Ensure interface compliance
var _ rounddiscord.RoundDiscordInterface = (*FakeRoundDiscord)(nil)
var _ createround.CreateRoundManager = (*FakeCreateRoundManager)(nil)
var _ roundrsvp.RoundRsvpManager = (*FakeRoundRsvpManager)(nil)
var _ roundreminder.RoundReminderManager = (*FakeRoundReminderManager)(nil)
var _ startround.StartRoundManager = (*FakeStartRoundManager)(nil)
var _ scoreround.ScoreRoundManager = (*FakeScoreRoundManager)(nil)
var _ finalizeround.FinalizeRoundManager = (*FakeFinalizeRoundManager)(nil)
var _ deleteround.DeleteRoundManager = (*FakeDeleteRoundManager)(nil)
var _ updateround.UpdateRoundManager = (*FakeUpdateRoundManager)(nil)
var _ tagupdates.TagUpdateManager = (*FakeTagUpdateManager)(nil)
var _ scorecardupload.ScorecardUploadManager = (*FakeScorecardUploadManager)(nil)
var _ rounddiscord.MessageMap = (*FakeMessageMap)(nil)

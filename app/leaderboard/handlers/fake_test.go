package handlers

import (
	"context"

	leaderboarddiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/leaderboard/discord"
	claimtag "github.com/Black-And-White-Club/discord-frolf-bot/app/leaderboard/discord/claim_tag"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/leaderboard/discord/history"
	leaderboardupdated "github.com/Black-And-White-Club/discord-frolf-bot/app/leaderboard/discord/leaderboard_updated"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/leaderboard/discord/season"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
)

// FakeLeaderboardDiscord is a programmable fake for LeaderboardDiscordInterface
type FakeLeaderboardDiscord struct {
	GetLeaderboardUpdateManagerFunc func() leaderboardupdated.LeaderboardUpdateManager
	GetClaimTagManagerFunc          func() claimtag.ClaimTagManager
	GetSeasonManagerFunc            func() season.SeasonManager
	GetHistoryManagerFunc           func() history.HistoryManager

	// Holds the sub-fakes
	LeaderboardUpdateManager FakeLeaderboardUpdateManager
	ClaimTagManager          FakeClaimTagManager
	SeasonMgr                FakeSeasonManager
	HistoryMgr               FakeHistoryManager
}

func (f *FakeLeaderboardDiscord) GetLeaderboardUpdateManager() leaderboardupdated.LeaderboardUpdateManager {
	if f.GetLeaderboardUpdateManagerFunc != nil {
		return f.GetLeaderboardUpdateManagerFunc()
	}
	return &f.LeaderboardUpdateManager
}

func (f *FakeLeaderboardDiscord) GetClaimTagManager() claimtag.ClaimTagManager {
	if f.GetClaimTagManagerFunc != nil {
		return f.GetClaimTagManagerFunc()
	}
	return &f.ClaimTagManager
}

func (f *FakeLeaderboardDiscord) GetSeasonManager() season.SeasonManager {
	if f.GetSeasonManagerFunc != nil {
		return f.GetSeasonManagerFunc()
	}
	return &f.SeasonMgr
}

func (f *FakeLeaderboardDiscord) GetHistoryManager() history.HistoryManager {
	if f.GetHistoryManagerFunc != nil {
		return f.GetHistoryManagerFunc()
	}
	return &f.HistoryMgr
}

// FakeLeaderboardUpdateManager implements leaderboardupdated.LeaderboardUpdateManager
type FakeLeaderboardUpdateManager struct {
	HandleLeaderboardPaginationFunc func(ctx context.Context, i *discordgo.InteractionCreate) (leaderboardupdated.LeaderboardUpdateOperationResult, error)
	SendLeaderboardEmbedFunc        func(ctx context.Context, channelID string, leaderboard []leaderboardupdated.LeaderboardEntry, page int32) (leaderboardupdated.LeaderboardUpdateOperationResult, error)
}

func (f *FakeLeaderboardUpdateManager) HandleLeaderboardPagination(ctx context.Context, i *discordgo.InteractionCreate) (leaderboardupdated.LeaderboardUpdateOperationResult, error) {
	if f.HandleLeaderboardPaginationFunc != nil {
		return f.HandleLeaderboardPaginationFunc(ctx, i)
	}
	return leaderboardupdated.LeaderboardUpdateOperationResult{}, nil
}

func (f *FakeLeaderboardUpdateManager) SendLeaderboardEmbed(ctx context.Context, channelID string, leaderboard []leaderboardupdated.LeaderboardEntry, page int32) (leaderboardupdated.LeaderboardUpdateOperationResult, error) {
	if f.SendLeaderboardEmbedFunc != nil {
		return f.SendLeaderboardEmbedFunc(ctx, channelID, leaderboard, page)
	}
	return leaderboardupdated.LeaderboardUpdateOperationResult{}, nil
}

// FakeClaimTagManager implements claimtag.ClaimTagManager
type FakeClaimTagManager struct {
	HandleClaimTagCommandFunc     func(ctx context.Context, i *discordgo.InteractionCreate) (claimtag.ClaimTagOperationResult, error)
	UpdateInteractionResponseFunc func(ctx context.Context, correlationID, message string) (claimtag.ClaimTagOperationResult, error)
}

func (f *FakeClaimTagManager) HandleClaimTagCommand(ctx context.Context, i *discordgo.InteractionCreate) (claimtag.ClaimTagOperationResult, error) {
	if f.HandleClaimTagCommandFunc != nil {
		return f.HandleClaimTagCommandFunc(ctx, i)
	}
	return claimtag.ClaimTagOperationResult{}, nil
}

func (f *FakeClaimTagManager) UpdateInteractionResponse(ctx context.Context, correlationID, message string) (claimtag.ClaimTagOperationResult, error) {
	if f.UpdateInteractionResponseFunc != nil {
		return f.UpdateInteractionResponseFunc(ctx, correlationID, message)
	}
	return claimtag.ClaimTagOperationResult{}, nil
}

// FakeSeasonManager implements season.SeasonManager
type FakeSeasonManager struct {
	HandleSeasonCommandFunc         func(ctx context.Context, i *discordgo.InteractionCreate)
	HandleSeasonStartedFunc         func(ctx context.Context, payload *leaderboardevents.StartNewSeasonSuccessPayloadV1)
	HandleSeasonStartFailedFunc     func(ctx context.Context, payload *leaderboardevents.AdminFailedPayloadV1)
	HandleSeasonStandingsFunc       func(ctx context.Context, payload *leaderboardevents.GetSeasonStandingsResponsePayloadV1)
	HandleSeasonStandingsFailedFunc func(ctx context.Context, payload *leaderboardevents.AdminFailedPayloadV1)
	HandleSeasonEndedFunc           func(ctx context.Context, payload *leaderboardevents.EndSeasonSuccessPayloadV1)
	HandleSeasonEndFailedFunc       func(ctx context.Context, payload *leaderboardevents.AdminFailedPayloadV1)
}

func (f *FakeSeasonManager) HandleSeasonCommand(ctx context.Context, i *discordgo.InteractionCreate) {
	if f.HandleSeasonCommandFunc != nil {
		f.HandleSeasonCommandFunc(ctx, i)
	}
}

func (f *FakeSeasonManager) HandleSeasonStarted(ctx context.Context, payload *leaderboardevents.StartNewSeasonSuccessPayloadV1) {
	if f.HandleSeasonStartedFunc != nil {
		f.HandleSeasonStartedFunc(ctx, payload)
	}
}

func (f *FakeSeasonManager) HandleSeasonStartFailed(ctx context.Context, payload *leaderboardevents.AdminFailedPayloadV1) {
	if f.HandleSeasonStartFailedFunc != nil {
		f.HandleSeasonStartFailedFunc(ctx, payload)
	}
}

func (f *FakeSeasonManager) HandleSeasonStandings(ctx context.Context, payload *leaderboardevents.GetSeasonStandingsResponsePayloadV1) {
	if f.HandleSeasonStandingsFunc != nil {
		f.HandleSeasonStandingsFunc(ctx, payload)
	}
}

func (f *FakeSeasonManager) HandleSeasonStandingsFailed(ctx context.Context, payload *leaderboardevents.AdminFailedPayloadV1) {
	if f.HandleSeasonStandingsFailedFunc != nil {
		f.HandleSeasonStandingsFailedFunc(ctx, payload)
	}
}

func (f *FakeSeasonManager) HandleSeasonEnded(ctx context.Context, payload *leaderboardevents.EndSeasonSuccessPayloadV1) {
	if f.HandleSeasonEndedFunc != nil {
		f.HandleSeasonEndedFunc(ctx, payload)
	}
}

func (f *FakeSeasonManager) HandleSeasonEndFailed(ctx context.Context, payload *leaderboardevents.AdminFailedPayloadV1) {
	if f.HandleSeasonEndFailedFunc != nil {
		f.HandleSeasonEndFailedFunc(ctx, payload)
	}
}

// Ensure interface compliance
var _ leaderboarddiscord.LeaderboardDiscordInterface = (*FakeLeaderboardDiscord)(nil)
var _ leaderboardupdated.LeaderboardUpdateManager = (*FakeLeaderboardUpdateManager)(nil)
var _ claimtag.ClaimTagManager = (*FakeClaimTagManager)(nil)
var _ season.SeasonManager = (*FakeSeasonManager)(nil)
var _ history.HistoryManager = (*FakeHistoryManager)(nil)

// FakeHistoryManager implements history.HistoryManager
type FakeHistoryManager struct {
	HandleHistoryCommandFunc     func(ctx context.Context, i *discordgo.InteractionCreate)
	HandleTagHistoryResponseFunc func(ctx context.Context, payload *leaderboardevents.TagHistoryResponsePayloadV1)
	HandleTagHistoryFailedFunc   func(ctx context.Context, payload *leaderboardevents.TagHistoryFailedPayloadV1)
	HandleTagGraphResponseFunc   func(ctx context.Context, payload *leaderboardevents.TagGraphResponsePayloadV1)
	HandleTagGraphFailedFunc     func(ctx context.Context, payload *leaderboardevents.TagGraphFailedPayloadV1)
}

func (f *FakeHistoryManager) HandleHistoryCommand(ctx context.Context, i *discordgo.InteractionCreate) {
	if f.HandleHistoryCommandFunc != nil {
		f.HandleHistoryCommandFunc(ctx, i)
	}
}

func (f *FakeHistoryManager) HandleTagHistoryResponse(ctx context.Context, payload *leaderboardevents.TagHistoryResponsePayloadV1) {
	if f.HandleTagHistoryResponseFunc != nil {
		f.HandleTagHistoryResponseFunc(ctx, payload)
	}
}

func (f *FakeHistoryManager) HandleTagHistoryFailed(ctx context.Context, payload *leaderboardevents.TagHistoryFailedPayloadV1) {
	if f.HandleTagHistoryFailedFunc != nil {
		f.HandleTagHistoryFailedFunc(ctx, payload)
	}
}

func (f *FakeHistoryManager) HandleTagGraphResponse(ctx context.Context, payload *leaderboardevents.TagGraphResponsePayloadV1) {
	if f.HandleTagGraphResponseFunc != nil {
		f.HandleTagGraphResponseFunc(ctx, payload)
	}
}

func (f *FakeHistoryManager) HandleTagGraphFailed(ctx context.Context, payload *leaderboardevents.TagGraphFailedPayloadV1) {
	if f.HandleTagGraphFailedFunc != nil {
		f.HandleTagGraphFailedFunc(ctx, payload)
	}
}

// FakeHelpers provides a programmable stub for utils.Helpers
type FakeHelpers struct {
	CreateNewMessageFunc    func(payload interface{}, topic string) (*message.Message, error)
	CreateResultMessageFunc func(originalMsg *message.Message, payload interface{}, topic string) (*message.Message, error)
	UnmarshalPayloadFunc    func(msg *message.Message, payload interface{}) error
}

func (f *FakeHelpers) CreateNewMessage(payload interface{}, topic string) (*message.Message, error) {
	if f.CreateNewMessageFunc != nil {
		return f.CreateNewMessageFunc(payload, topic)
	}
	return &message.Message{}, nil
}

func (f *FakeHelpers) CreateResultMessage(originalMsg *message.Message, payload interface{}, topic string) (*message.Message, error) {
	if f.CreateResultMessageFunc != nil {
		return f.CreateResultMessageFunc(originalMsg, payload, topic)
	}
	return &message.Message{}, nil
}

func (f *FakeHelpers) UnmarshalPayload(msg *message.Message, payload interface{}) error {
	if f.UnmarshalPayloadFunc != nil {
		return f.UnmarshalPayloadFunc(msg, payload)
	}
	return nil
}

// FakeGuildConfigResolver provides a programmable stub for GuildConfigResolver
type FakeGuildConfigResolver struct {
	GetGuildConfigWithContextFunc func(ctx context.Context, guildID string) (*storage.GuildConfig, error)
	RequestGuildConfigAsyncFunc   func(ctx context.Context, guildID string)
	IsGuildSetupCompleteFunc      func(guildID string) bool
	HandleGuildConfigReceivedFunc func(ctx context.Context, guildID string, config *storage.GuildConfig)
	HandleBackendErrorFunc        func(ctx context.Context, guildID string, err error)
	ClearInflightRequestFunc      func(ctx context.Context, guildID string)
}

func (f *FakeGuildConfigResolver) GetGuildConfigWithContext(ctx context.Context, guildID string) (*storage.GuildConfig, error) {
	if f.GetGuildConfigWithContextFunc != nil {
		return f.GetGuildConfigWithContextFunc(ctx, guildID)
	}
	return &storage.GuildConfig{}, nil
}

func (f *FakeGuildConfigResolver) RequestGuildConfigAsync(ctx context.Context, guildID string) {
	if f.RequestGuildConfigAsyncFunc != nil {
		f.RequestGuildConfigAsyncFunc(ctx, guildID)
	}
}

func (f *FakeGuildConfigResolver) IsGuildSetupComplete(guildID string) bool {
	if f.IsGuildSetupCompleteFunc != nil {
		return f.IsGuildSetupCompleteFunc(guildID)
	}
	return true
}

func (f *FakeGuildConfigResolver) HandleGuildConfigReceived(ctx context.Context, guildID string, config *storage.GuildConfig) {
	if f.HandleGuildConfigReceivedFunc != nil {
		f.HandleGuildConfigReceivedFunc(ctx, guildID, config)
	}
}

func (f *FakeGuildConfigResolver) HandleBackendError(ctx context.Context, guildID string, err error) {
	if f.HandleBackendErrorFunc != nil {
		f.HandleBackendErrorFunc(ctx, guildID, err)
	}
}

func (f *FakeGuildConfigResolver) ClearInflightRequest(ctx context.Context, guildID string) {
	if f.ClearInflightRequestFunc != nil {
		f.ClearInflightRequestFunc(ctx, guildID)
	}
}

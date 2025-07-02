package leaderboardevents

import (
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

const (
	LeaderboardRetrieveRequestTopic         = "discord.leaderboard.retrieve.request"
	LeaderboardRetrievedTopic               = "discord.leaderboard.retrieved"
	LeaderboardTagAssignRequestTopic        = "discord.leaderboard.tag.assign.request"
	LeaderboardTagAssignedTopic             = "discord.leaderboard.tag.assigned"
	LeaderboardTagAssignFailedTopic         = "discord.leaderboard.tag.assign.failed"
	LeaderboardTagAvailabilityRequestTopic  = "discord.leaderboard.tag.availability.request"
	LeaderboardTagAvailabilityResponseTopic = "discord.leaderboard.tag.availability.response"
	LeaderboardTagSwapRequestTopic          = "discord.leaderboard.tag.swap.request"
	LeaderboardTagSwappedTopic              = "discord.leaderboard.tag.swapped"
	LeaderboardTagSwapFailedTopic           = "discord.leaderboard.tag.swap.failed"
)

type LeaderboardRetrieveRequestPayload struct {
	UserID    sharedtypes.DiscordID `json:"user_id"`
	ChannelID string                `json:"channel_id"`
	MessageID string                `json:"message_id"`
	GuildID   string                `json:"guild_id"`
}

// Corrected LeaderboardRetrievedPayload
type LeaderboardRetrievedPayload struct {
	Leaderboard []leaderboardtypes.LeaderboardEntry `json:"leaderboard"`
	ChannelID   string                              `json:"channel_id"`
	MessageID   string                              `json:"message_id"`
	GuildID     string                              `json:"guild_id"`
}
type LeaderboardTagAssignRequestPayload struct {
	TargetUserID sharedtypes.DiscordID `json:"target_user_id"`
	TagNumber    sharedtypes.TagNumber `json:"tag_number"`
	RequestorID  sharedtypes.DiscordID `json:"requestor_id"`
	ChannelID    string                `json:"channel_id"`
	MessageID    string                `json:"message_id"`
	GuildID      string                `json:"guild_id"`
}
type LeaderboardTagAssignedPayload struct {
	TargetUserID string                `json:"target_user_id"`
	TagNumber    sharedtypes.TagNumber `json:"tag_number"`
	ChannelID    string                `json:"channel_id"`
	MessageID    string                `json:"message_id"`
	GuildID      string                `json:"guild_id"`
}
type LeaderboardTagAssignFailedPayload struct {
	TargetUserID string                `json:"target_user_id"`
	TagNumber    sharedtypes.TagNumber `json:"tag_number"`
	Reason       string                `json:"reason"`
	ChannelID    string                `json:"channel_id"`
	MessageID    string                `json:"message_id"`
	GuildID      string                `json:"guild_id"`
}
type LeaderboardTagAvailabilityRequestPayload struct {
	TagNumber sharedtypes.TagNumber `json:"tag_number"`
	UserID    sharedtypes.DiscordID `json:"user_id"`
	ChannelID string                `json:"channel_id"`
	MessageID string                `json:"message_id"`
	GuildID   string                `json:"guild_id"`
}
type LeaderboardTagAvailabilityResponsePayload struct {
	TagNumber sharedtypes.TagNumber `json:"tag_number"`
	ChannelID string                `json:"channel_id"`
	MessageID string                `json:"message_id"`
	GuildID   string                `json:"guild_id"`
}
type LeaderboardTagSwapRequestPayload struct {
	User1ID     sharedtypes.DiscordID `json:"user1_id"`
	User2ID     sharedtypes.DiscordID `json:"user2_id"`
	RequestorID sharedtypes.DiscordID `json:"requestor_id"`
	ChannelID   string                `json:"channel_id"`
	MessageID   string                `json:"message_id"`
	GuildID     string                `json:"guild_id"`
}
type LeaderboardTagSwappedPayload struct {
	User1ID   sharedtypes.DiscordID `json:"user1_id"`
	User2ID   sharedtypes.DiscordID `json:"user2_id"`
	ChannelID string                `json:"channel_id"`
	MessageID string                `json:"message_id"`
	GuildID   string                `json:"guild_id"`
}
type LeaderboardTagSwapFailedPayload struct {
	User1ID   sharedtypes.DiscordID `json:"user1_id"`
	User2ID   sharedtypes.DiscordID `json:"user2_id"`
	Reason    string                `json:"reason"`
	ChannelID string                `json:"channel_id"`
	MessageID string                `json:"message_id"`
	GuildID   string                `json:"guild_id"`
}

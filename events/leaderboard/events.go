package leaderboardevents

import (
	"github.com/Black-And-White-Club/frolf-bot-shared/events"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
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
	events.CommonMetadata `json:",inline"`
	UserID                string `json:"user_id"`
	ChannelID             string `json:"channel_id"`
	MessageID             string `json:"message_id"`
}

// Corrected LeaderboardRetrievedPayload
type LeaderboardRetrievedPayload struct {
	events.CommonMetadata `json:",inline"`
	Leaderboard           []leaderboardevents.LeaderboardEntry `json:"leaderboard"`
	ChannelID             string                               `json:"channel_id"`
	MessageID             string                               `json:"message_id"`
}

type LeaderboardTagAssignRequestPayload struct {
	events.CommonMetadata `json:",inline"`
	TargetUserID          string `json:"target_user_id"`
	TagNumber             int    `json:"tag_number"`
	RequestorID           string `json:"requestor_id"`
	ChannelID             string `json:"channel_id"`
	MessageID             string `json:"message_id"`
}
type LeaderboardTagAssignedPayload struct {
	events.CommonMetadata `json:",inline"`
	TargetUserID          string `json:"target_user_id"`
	TagNumber             int    `json:"tag_number"`
	ChannelID             string `json:"channel_id"`
	MessageID             string `json:"message_id"`
}

type LeaderboardTagAssignFailedPayload struct {
	events.CommonMetadata `json:",inline"`
	TargetUserID          string `json:"target_user_id"`
	TagNumber             int    `json:"tag_number"`
	Reason                string `json:"reason"`
	ChannelID             string `json:"channel_id"`
	MessageID             string `json:"message_id"`
}

type LeaderboardTagAvailabilityRequestPayload struct {
	events.CommonMetadata `json:",inline"`
	TagNumber             int    `json:"tag_number"`
	UserID                string `json:"user_id"`
	ChannelID             string `json:"channel_id"`
	MessageID             string `json:"message_id"`
}

type LeaderboardTagAvailabilityResponsePayload struct {
	events.CommonMetadata `json:",inline"`
	TagNumber             int    `json:"tag_number"`
	ChannelID             string `json:"channel_id"`
	MessageID             string `json:"message_id"`
}
type LeaderboardTagSwapRequestPayload struct {
	events.CommonMetadata `json:",inline"`
	User1ID               string `json:"user1_id"`
	User2ID               string `json:"user2_id"`
	RequestorID           string `json:"requestor_id"`
	ChannelID             string `json:"channel_id"`
	MessageID             string `json:"message_id"`
}

type LeaderboardTagSwappedPayload struct {
	events.CommonMetadata `json:",inline"`
	User1ID               string `json:"user1_id"`
	User2ID               string `json:"user2_id"`
	ChannelID             string `json:"channel_id"`
	MessageID             string `json:"message_id"`
}

type LeaderboardTagSwapFailedPayload struct {
	events.CommonMetadata `json:",inline"`
	User1ID               string `json:"user1_id"`
	User2ID               string `json:"user2_id"`
	Reason                string `json:"reason"`
	ChannelID             string `json:"channel_id"`
	MessageID             string `json:"message_id"`
}

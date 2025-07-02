package roundevents

import (
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// --- Topics (for Watermill) ---
const (
	// Internal Topics (Discord bot only)
	RoundCreateModalSubmit            = "discord.round.modal.submit"    // From discord handler to round handler
	RoundCreatedTopic                 = "discord.round.created"         // From round handler to discord handler
	RoundCreationFailedTopic          = "discord.round.creation.failed" // From round handler to discord handler
	RoundReminderTopic                = "discord.round.reminder"        // From round handler to discord handler
	RoundStartedTopic                 = "discord.round.started"
	RoundParticipantJoinReqTopic      = "discord.round.participant.join.request" // From discord handler to round handler
	RoundParticipantJoinedTopic       = "discord.round.participantjoined"        // From round handler to discord
	RoundUpdateRequestTopic           = "discord.round.update.request"           // From discord handler to round handler
	RoundUpdatedTopic                 = "discord.round.updated"                  // From round handler to discord handler
	RoundDeleteRequestTopic           = "discord.round.delete.request"           // From discord handler to round handler
	RoundDeletedTopic                 = "discord.round.deleted"
	RoundScoreUpdateRequestTopic      = "discord.round.score.update.request"      // From discord handler to round handler
	RoundParticipantScoreUpdatedTopic = "discord.round.participant.score.updated" // From round handler to discord
	RoundFinalizedTopic               = "discord.round.finalized"
	RoundValidationFailed             = "discord.round.validation.failed"
	RoundUpdateModalSubmit            = "discord.round.update.modal.submit"
	RoundCreatedTraceTopic            = "discord.round.created.trace"
)

// --- Internal Payloads (Discord bot only) ---
// CreateRoundRequestedPayload: From discord handler (modal submit) to round handler.
type CreateRoundRequestedPayload struct {
	Title       roundtypes.Title       `json:"title"`
	Description roundtypes.Description `json:"description"`
	StartTime   string                 `json:"start_time"`
	Location    roundtypes.Location    `json:"location"`
	UserID      sharedtypes.DiscordID  `json:"user_id"`
	ChannelID   string                 `json:"channel_id"`
	Timezone    roundtypes.Timezone    `json:"timezone"`
	GuildID     sharedtypes.GuildID    `json:"guild_id"`
}

// RoundCreatedPayload: From round handler to discord handler (success).
type RoundCreatedPayload struct { // Embed the common metadata
	RoundID     sharedtypes.RoundID   `json:"round_id"`
	Title       roundtypes.Title      `json:"title"`
	StartTime   sharedtypes.StartTime `json:"start_time"`
	Location    roundtypes.Location   `json:"location"`
	RequesterID sharedtypes.DiscordID `json:"requester_id"`
	ChannelID   string                `json:"channel_id"`
	GuildID     sharedtypes.GuildID   `json:"guild_id"`
}

// RoundCreationFailedPayload: From round handler to discord handler (failure).
type RoundCreationFailedPayload struct {
	UserID  sharedtypes.DiscordID `json:"user_id"`
	Reason  string                `json:"reason"`
	GuildID string                `json:"guild_id"`
}

// DiscordRoundReminderPayload: From round handler to discord handler (for sending reminders).
type DiscordRoundReminderPayload struct {
	RoundID      sharedtypes.RoundID `json:"round_id"`
	RoundTitle   roundtypes.Title    `json:"round_title"`
	UserIDs      []string            `json:"user_ids"`
	ReminderType string              `json:"reminder_type"`
	ChannelID    string              `json:"channel_id"`
	GuildID      string              `json:"guild_id"`
}
type DiscordRoundStartPayload struct {
	RoundID   sharedtypes.RoundID    `json:"round_id"`
	Title     roundtypes.Title       `json:"title"`
	Location  *roundtypes.Location   `json:"location"`
	StartTime *sharedtypes.StartTime `json:"start_time"`
	ChannelID string                 `json:"channel_id"`
	GuildID   string                 `json:"guild_id"`
}
type DiscordRoundParticipantJoinRequestPayload struct {
	RoundID    sharedtypes.RoundID   `json:"round_id"`
	UserID     sharedtypes.DiscordID `json:"user_id"`
	ChannelID  string                `json:"channel_id"`
	JoinedLate *bool                 `json:"joined_late,omitempty"`
	GuildID    string                `json:"guild_id"`
}
type DiscordRoundParticipantJoinedPayload struct {
	RoundID   sharedtypes.RoundID   `json:"round_id"`
	UserID    sharedtypes.DiscordID `json:"user_id"`
	TagNumber sharedtypes.TagNumber `json:"tag_number"`
	ChannelID string                `json:"channel_id"`
	GuildID   string                `json:"guild_id"`
}

// --- Update Round ---
type DiscordRoundUpdateRequestPayload struct {
	RoundID     sharedtypes.RoundID     `json:"round_id"`
	UserID      sharedtypes.DiscordID   `json:"user_id"`
	MessageID   string                  `json:"message_id"`
	Title       *roundtypes.Title       `json:"title,omitempty"`
	Description *roundtypes.Description `json:"description,omitempty"`
	StartTime   *sharedtypes.StartTime  `json:"start_time,omitempty"`
	Location    *roundtypes.Location    `json:"location,omitempty"`
	ChannelID   string                  `json:"channel_id"`
	GuildID     sharedtypes.GuildID     `json:"guild_id"`
}
type DiscordRoundUpdatedPayload struct {
	RoundID     sharedtypes.RoundID     `json:"round_id"`
	MessageID   string                  `json:"message_id"`
	ChannelID   string                  `json:"channel_id"`
	Title       *roundtypes.Title       `json:"title,omitempty"`
	Description *roundtypes.Description `json:"description,omitempty"`
	StartTime   *sharedtypes.StartTime  `json:"start_time,omitempty"`
	Location    *roundtypes.Location    `json:"location,omitempty"`
	GuildID     sharedtypes.GuildID     `json:"guild_id"`
}
type DiscordRoundDeleteRequestPayload struct {
	RoundID   sharedtypes.RoundID   `json:"round_id"`
	UserID    sharedtypes.DiscordID `json:"user_id"`
	ChannelID string                `json:"channel_id"`
	MessageID string                `json:"message_id"`
	GuildID   string                `json:"guild_id"`
}
type DiscordRoundDeletedPayload struct {
	RoundID   sharedtypes.RoundID `json:"round_id"`
	ChannelID string              `json:"channel_id"`
	MessageID string              `json:"message_id"`
	GuildID   string              `json:"guild_id"`
}
type DiscordRoundParticipantScoreUpdatedPayload struct {
	RoundID   sharedtypes.RoundID   `json:"round_id"`
	UserID    sharedtypes.DiscordID `json:"user_id"`
	Score     sharedtypes.Score     `json:"score"`
	ChannelID string                `json:"channel_id"`
	MessageID string                `json:"message_id"`
	GuildID   string                `json:"guild_id"`
}
type DiscordRoundScoreUpdateRequestPayload struct {
	RoundID   sharedtypes.RoundID   `json:"round_id"`
	UserID    sharedtypes.DiscordID `json:"user_id"` // The user submitting the score
	Score     sharedtypes.Score     `json:"score"`
	ChannelID string                `json:"channel_id"`
	MessageID string                `json:"message_id"`
	GuildID   string                `json:"guild_id"`
}
type DiscordRoundFinalizedPayload struct {
	RoundID   sharedtypes.RoundID `json:"round_id"`
	ChannelID string              `json:"channel_id"`
	MessageID string              `json:"message_id"`
	GuildID   string              `json:"guild_id"`
}

type DiscordRoundCreatedTracePayload struct {
	RoundID   sharedtypes.RoundID   `json:"round_id"`
	Title     roundtypes.Title      `json:"title"`
	CreatedBy sharedtypes.DiscordID `json:"created_by"`
	GuildID   string                `json:"guild_id"`
}

type DiscordRoundUpdateModalSubmitPayload struct {
	RoundID     sharedtypes.RoundID     `json:"round_id"`
	UserID      sharedtypes.DiscordID   `json:"user_id"`
	MessageID   string                  `json:"message_id"`
	Title       *roundtypes.Title       `json:"title,omitempty"`
	Description *roundtypes.Description `json:"description,omitempty"`
	StartTime   *string                 `json:"start_time,omitempty"`
	Timezone    *roundtypes.Timezone    `json:"timezone,omitempty"`
	Location    *roundtypes.Location    `json:"location,omitempty"`
	ChannelID   string                  `json:"channel_id"`
	GuildID     sharedtypes.GuildID     `json:"guild_id"`
}

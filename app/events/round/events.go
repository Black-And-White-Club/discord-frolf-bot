package roundevents

import (
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/events"
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
	RoundParticipantJoinedTopic       = "discord.round.participant.joined"       // From round handler to discord
	RoundUpdateRequestTopic           = "discord.round.update.request"           // From discord handler to round handler
	RoundUpdatedTopic                 = "discord.round.updated"                  //From round handler to discord handler
	RoundDeleteRequestTopic           = "discord.round.delete.request"           // From discord handler to round handler
	RoundDeletedTopic                 = "discord.round.deleted"
	RoundScoreUpdateRequestTopic      = "discord.round.score.update.request"      //From discord handler to round handler
	RoundParticipantScoreUpdatedTopic = "discord.round.participant.score.updated" //From round handler to discord
	RoundFinalizedTopic               = "discord.round.finalized"
	RoundValidationFailed             = "discord.round.validation.failed"
	RoundCreatedTraceTopic            = "discord.round.created.trace"
)

// --- Internal Payloads (Discord bot only) ---
// CreateRoundRequestedPayload: From discord handler (modal submit) to round handler.
type CreateRoundRequestedPayload struct {
	events.CommonMetadata `json:",inline"`
	Title                 string `json:"title"`
	Description           string `json:"description"`
	StartTime             string `json:"start_time"`
	Location              string `json:"location"`
	UserID                string `json:"user_id"`
	ChannelID             string `json:"channel_id"`
	Timezone              string `json:"timezone"`
}

// RoundCreatedPayload: From round handler to discord handler (success).
type RoundCreatedPayload struct {
	events.CommonMetadata `json:",inline"` // Embed the common metadata
	RoundID               int64            `json:"round_id"`
	Title                 string           `json:"title"`
	StartTime             time.Time        `json:"start_time"`
	Location              string           `json:"location"`
	RequesterID           string           `json:"requester_id"`
	ChannelID             string           `json:"channel_id"`
}

// RoundCreationFailedPayload: From round handler to discord handler (failure).
type RoundCreationFailedPayload struct {
	events.CommonMetadata `json:",inline"`
	UserID                string `json:"user_id"`
	Reason                string `json:"reason"`
}

// DiscordRoundReminderPayload: From round handler to discord handler (for sending reminders).
type DiscordRoundReminderPayload struct {
	events.CommonMetadata `json:",inline"`
	RoundID               int64    `json:"round_id"`
	RoundTitle            string   `json:"round_title"`
	UserIDs               []string `json:"user_ids"`
	ReminderType          string   `json:"reminder_type"`
	ChannelID             string   `json:"channel_id"`
}
type DiscordRoundStartPayload struct {
	events.CommonMetadata `json:",inline"`
	RoundID               int64      `json:"round_id"`
	Title                 string     `json:"title"`
	Location              *string    `json:"location"`
	StartTime             *time.Time `json:"start_time"`
	ChannelID             string     `json:"channel_id"`
}
type DiscordRoundParticipantJoinRequestPayload struct {
	events.CommonMetadata `json:",inline"`
	RoundID               int64  `json:"round_id"`
	UserID                string `json:"user_id"`
	ChannelID             string `json:"channel_id"`
	JoinedLate            *bool  `json:"joined_late,omitempty"`
}
type DiscordRoundParticipantJoinedPayload struct {
	events.CommonMetadata `json:",inline"`
	RoundID               int64  `json:"round_id"`
	UserID                string `json:"user_id"`
	TagNumber             int    `json:"tag_number"`
	ChannelID             string `json:"channel_id"`
}

// --- Update Round ---
type DiscordRoundUpdateRequestPayload struct {
	events.CommonMetadata `json:",inline"`
	RoundID               int64      `json:"round_id"`
	UserID                string     `json:"user_id"`
	MessageID             string     `json:"message_id"`
	Title                 *string    `json:"title,omitempty"`
	Description           *string    `json:"description,omitempty"`
	StartTime             *time.Time `json:"start_time,omitempty"`
	Location              *string    `json:"location,omitempty"`
	ChannelID             string     `json:"channel_id"`
}
type DiscordRoundUpdatedPayload struct {
	events.CommonMetadata `json:",inline"`
	RoundID               int64      `json:"round_id"`
	MessageID             string     `json:"message_id"`
	ChannelID             string     `json:"channel_id"`
	Title                 *string    `json:"title,omitempty"`
	Description           *string    `json:"description,omitempty"`
	StartTime             *time.Time `json:"start_time,omitempty"`
	Location              *string    `json:"location,omitempty"`
}
type DiscordRoundDeleteRequestPayload struct {
	events.CommonMetadata `json:",inline"`
	RoundID               int64  `json:"round_id"`
	UserID                string `json:"user_id"`
	ChannelID             string `json:"channel_id"`
	MessageID             string `json:"message_id"`
}
type DiscordRoundDeletedPayload struct {
	events.CommonMetadata `json:",inline"`
	RoundID               int64  `json:"round_id"`
	ChannelID             string `json:"channel_id"`
	MessageID             string `json:"message_id"`
}
type DiscordRoundParticipantScoreUpdatedPayload struct {
	events.CommonMetadata `json:",inline"`
	RoundID               int64  `json:"round_id"`
	UserID                string `json:"user_id"`
	Score                 int    `json:"score"`
	ChannelID             string `json:"channel_id"`
	MessageID             string `json:"message_id"`
}
type DiscordRoundScoreUpdateRequestPayload struct {
	events.CommonMetadata `json:",inline"`
	RoundID               int64  `json:"round_id"`
	UserID                string `json:"user_id"` // The user submitting the score
	Score                 int    `json:"score"`
	ChannelID             string `json:"channel_id"`
	MessageID             string `json:"message_id"`
}
type DiscordRoundFinalizedPayload struct {
	events.CommonMetadata `json:",inline"`
	RoundID               int64  `json:"round_id"`
	ChannelID             string `json:"channel_id"`
	MessageID             string `json:"message_id"`
}

type DiscordRoundCreatedTracePayload struct {
	RoundID   int64     `json:"round_id"`
	Title     string    `json:"title"`
	CreatedBy string    `json:"created_by"`
	Timestamp time.Time `json:"timestamp"`
}

package scoreevents

import (
	"github.com/Black-And-White-Club/frolf-bot-shared/events"
)

const (
	// We ONLY need these topics now, since we're only handling updates.
	ScoreUpdateRequestTopic  = "discord.score.update.request"
	ScoreUpdateResponseTopic = "discord.score.update.response"
)

// ScoreUpdateRequestPayload is the payload for a manual score update request from Discord.
type ScoreUpdateRequestPayload struct {
	events.CommonMetadata
	RoundID     string `json:"round_id"`
	Participant string `json:"participant"` // Discord ID of the user
	Score       *int   `json:"score"`       // Pointer for "no change" option
	TagNumber   int    `json:"tag_number"`
	UserID      string `json:"user_id"` // Add these for metadata
	ChannelID   string `json:"channel_id"`
	MessageID   string `json:"message_id"`
}

// ScoreUpdateResponsePayload is sent back to Discord after a score update.
type ScoreUpdateResponsePayload struct {
	events.CommonMetadata
	Success     bool   `json:"success"`
	RoundID     string `json:"round_id,omitempty"`
	Participant string `json:"participant,omitempty"`
	Error       string `json:"error,omitempty"` // Include error details
	UserID      string `json:"user_id"`
	ChannelID   string `json:"channel_id"`
	MessageID   string `json:"message_id"`
}

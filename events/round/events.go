package roundevents

import "time"

const (
	RoundStartCreation        = "round.start.creation"
	RoundCreateRequest        = "round.create.request"
	RoundTitleCollected       = "round.title.collected"
	RoundStartTimeCollected   = "round.starttime.collected"
	RoundLocationCollected    = "round.location.collected"
	RoundConfirmationRequest  = "round.confirmation.request"
	RoundConfirmed            = "round.confirmed"
	RoundEditTitle            = "round.edit.title"
	RoundEditStartTime        = "round.edit.starttime"
	RoundEditLocation         = "round.edit.location"
	RoundCreated              = "round.created"
	RoundCreationCanceled     = "round.creation.canceled"
	RoundTrace                = "round.trace"
	RoundTitleResponse        = "round.title.response"
	RoundStartTimeResponse    = "round.starttime.response"
	RoundLocationResponse     = "round.location.response"
	RoundDescriptionCollected = "round.description.collected"
	RoundDescriptionResponse  = "round.description.response"
	RoundEndTimeCollected     = "round.endtime.collected"
	RoundEndTimeResponse      = "round.endtime.response"
)

// CancelRoundCreationPayload defines the payload for canceling round creation.
type CancelRoundCreationPayload struct {
	UserID string `json:"user_id"`
}

// TracePayload defines the payload for trace events.
type TracePayload struct {
	Message string `json:"message"`
}

// RoundEventPayload is a structure to standardize the data sent in events.
type RoundEventPayload struct {
	UserID      string `json:"user_id"`
	Response    string `json:"response,omitempty"`
	Title       string `json:"title,omitempty"`
	StartTime   string `json:"start_time,omitempty"`
	Location    string `json:"location,omitempty"`
	Description string `json:"description,omitempty"`
	EndTime     string `json:"end_time,omitempty"`
}

// Constants for state names
const (
	StateCollectingTitle       = "CollectingTitle"
	StateCollectingStartTime   = "CollectingStartTime"
	StateCollectingLocation    = "CollectingLocation"
	StateConfirmation          = "Confirmation"
	StateCollectingDescription = "CollectingDescription"
	StateCollectingEndTime     = "CollectingEndTime"
)

// RoundCreationContext defines the context for creating a round.
type RoundCreationContext struct {
	UserID        string    `json:"user_id"`
	Title         string    `json:"title"`
	StartTime     time.Time `json:"start_time"`
	EndTime       time.Time `json:"end_time"`
	Description   string    `json:"description"`
	Location      string    `json:"location"`
	State         string    `json:"state"`
	CorrelationID string    `json:"correlation_id"`
}

// GuildScheduledEventCreatedPayload - Payload for the new event.
type GuildScheduledEventCreatedPayload struct {
	GuildEventID string  `json:"guild_event_id"`
	ChannelID    string  `json:"channel_id"`
	Title        string  `json:"title"`
	StartTime    string  `json:"start_time"`
	EndTime      string  `json:"end_time,omitempty"`
	Location     *string `json:"location,omitempty"`
}

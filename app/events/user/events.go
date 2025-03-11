package discorduserevents

import (
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	"github.com/bwmarrin/discordgo"
)

// Event names (constants) - grouped by related functionality for better readability.
const (
	// Signup-related events (Discord-side).
	SignupStarted             = "discord.user.signup.started"
	SignupTagAsk              = "discord.user.signup.tag.ask"
	SignupTagSkip             = "discord.user.signup.tag.skip"
	SignupTagIncludeRequested = "discord.user.signup.tag.include.requested"
	SignupTagPromptSent       = "discord.user.signup.tag.prompt.sent"
	SignupCanceled            = "discord.user.signup.canceled"
	SignupFailed              = "discord.user.signup.failed" // General signup failure (Discord side).
	SignupSuccess             = "discord.user.signup.success"
	InteractionResponded      = "discord.user.interaction.responded"
	SignupFormSubmitted       = "discord.user.signup.formsubmitted"
	SignupSubmission          = "discord.user.signup.submission"
	SignupAddRole             = "discord.user.signup.addrole"
	SignupRoleAdded           = "discord.user.signup.addrolesuccess"
	SignupRoleAdditionFailed  = "discord.user.signup.addrolefailed"
	// Discord DM events
	SendUserDM = "discord.send.dm"
	DMSent     = "discord.user.dmsent"
	DMError    = "discord.user.dmerror"
	// Tag-related events (Discord-side).
	TagNumberRequested = "discord.user.tag.number.requested"
	TagNumberResponse  = "discord.user.tag.number.response" // Payload from the user.
	// Role Update events (Discord-side).
	RoleUpdateCommand     = "discord.user.roleupdatecommand" // From command handler.
	RoleUpdateButtonPress = "discord.user.roleupdatebuttonpress"
	RoleUpdateTimeout     = "discord.user.roleupdateyimeout"
	RoleOptionsRequested  = "discord.user.roleoptionsrequested"
	RoleResponse          = "discord.user.roleresponse"
	RoleResponseFailed    = "discord.user.roleresponsefailed"
	// Trace events, keeping these, I guess.
	SignupResponseTrace     = "discord.user.signup.response.trace"
	RoleUpdateResponseTrace = "discord.user.role.update.response.trace"
	// Generic Trace event
	DiscordEventTrace = "discord.user.event.trace"
)

// --- Payload Structs ---
// RoleUpdateCommandPayload defines the payload for the internal RoleUpdateCommand event.
type RoleUpdateCommandPayload struct {
	TargetUserID string `json:"target_user_id"`
	GuildID      string `json:"guild_id"`
}

// RoleUpdateButtonPressPayload is the payload for the RoleUpdateButtonPress event.
type RoleUpdateButtonPressPayload struct {
	RequesterID         string                 `json:"requester_id"`
	TargetUserID        string                 `json:"target_user_id"`
	SelectedRole        usertypes.UserRoleEnum `json:"selected_role"`
	InteractionID       string                 `json:"interaction_id"`
	InteractionToken    string                 `json:"interaction_token"`
	InteractionCustomID string                 `json:"custom_id"`
	GuildID             string                 `json:"guild_id"`
}

// SendUserDMPayload defines the payload to send a DM to a user.
type SendUserDMPayload struct {
	UserID  string `json:"user_id"`
	Message string `json:"message"`
}

// ---  Payloads below this probably aren't needed anymore ---
// RoleUpdateResponsePayload is the user's response to role options
type RoleUpdateResponsePayload struct {
	Response  string `json:"response"` // The chosen role (or "cancel").
	UserID    string `json:"user_id"`  // The user who responded.
	MessageID string `json:"message_id"`
}

// TagNumberResponsePayload for when the user responsds with their tag number.
type TagNumberResponsePayload struct {
	TagNumber string `json:"tag_number"`
	UserID    string `json:"user_id"`
	MessageID string `json:"message_id"`
}

// SignupStartedPayload defines the payload for the SignupStarted event.
type SignupStartedPayload struct {
	UserID    string `json:"user_id"`
	ChannelID string `json:"channel_id"`
	MessageID string `json:"message_id,omitempty"`
}

// CancelPayload defines the payload for the cancel event.
type CancelPayload struct {
	UserID string `json:"user_id"`
}

// TracePayload defines the payload for the trace event.
type TracePayload struct {
	Message string `json:"message"`
}

// TagNumberRequestedPayload defines the payload for the TagNumberRequested event.
type TagNumberRequestedPayload struct {
	UserID      string                 `json:"user_id"`
	Interaction *discordgo.Interaction `json:"interaction"`
}

// RoleUpdateTimeoutPayload defines the payload for the RoleUpdateTimeout event.
type RoleUpdateTimeoutPayload struct {
	UserID string `json:"user_id"`
}
type TagNumberProvidedPayload struct {
	UserID      string                 `json:"user_id"`
	TagNumber   string                 `json:"tag_number"`
	Interaction *discordgo.Interaction `json:"interaction"`
}

// Success payload
type SignupSuccessPayload struct {
	UserID        string `json:"user_id"`
	CorrelationID string `json:"correlation_id"`
}

// AddRolePayload defines the payload for the AddRole event.
type AddRolePayload struct {
	DiscordID string `json:"user_id"`
	RoleID    string `json:"role_id"`
}

// RoleAddedPayload defines the payload for the RoleAdded event.
type RoleAddedPayload struct {
	DiscordID string `json:"user_id"`
}

// RoleAdditionFailedPayload defines the payload for the RoleAdditionFailed event.
type RoleAdditionFailedPayload struct {
	DiscordID string `json:"user_id"`
	Reason    string `json:"reason"`
}

// Failure payload
// Deprecated
type SignupFailedPayload struct {
	Reason        string `json:"reason"`
	Detail        string `json:"detail"`
	UserID        string `json:"user_id"`
	CorrelationID string `json:"correlation_id"`
}
type SignupFormSubmittedPayload struct {
	UserID           string `json:"user_id"`
	InteractionID    string `json:"interaction_id"`
	InteractionToken string `json:"interaction_token"`
	TagNumber        *int   `json:"tag_number"`
}

// SignupModalData represents the structure of the modal submission data.
type SignupModalData struct {
	CustomID   string                 `json:"custom_id"`
	Components []discordgo.ActionsRow `json:"components"`
}

package userevents

const (
	SignupStarted             = "discord.signup.started"
	SignupTagAsk              = "discord.signup.tag.ask"
	SignupCanceled            = "discord.signup.canceled"
	SignupTagIncludeRequested = "discord.include.tag.number.request"
	SignupTrace               = "discord.signup.response.trace"
	TagNumberRequested        = "discord.tag.number.requested"
	TagNumberResponse         = "discord.tag.number.included.response"
	RoleUpdateTimeout         = "discord.role.update.timeout"
	RoleSelectRequest         = "discord.role.select.request"
	RoleSelectResponse        = "discord.role.select.response"
	RoleUpdateCommand         = "discord.role.update.command"
	RoleUpdateResponseTrace   = "discord.signup.response.trace"
	RoleUpdateTrace           = "discord.signup.response.trace" // I need to do something so that this is not a duplicate
)

// SignupStartedPayload defines the payload for the SignupStarted event.
type SignupStartedPayload struct {
	UserID    string `json:"user_id"`
	ChannelID string `json:"channel_id"`
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
	UserID string `json:"user_id"`
	TagID  string `json:"tag_id"`
}

// RoleUpdateTimeoutPayload defines the payload for the RoleUpdateTimeout event.
type RoleUpdateTimeoutPayload struct {
	UserID string `json:"user_id"`
}

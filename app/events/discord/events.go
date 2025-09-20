package discordevents

const (
	StatusSuccess = "success"
	StatusFail    = "fail"
	SendDM        = "discord.send.dm"
	DMSent        = "discord.user.dmsent"
	DMError       = "discord.user.dmerror"
)

// Interaction response tracking
type InteractionRespondedPayload struct {
	InteractionID string `json:"interaction_id"`
	UserID        string `json:"user_id"`
	Status        string `json:"status"`
	ErrorDetail   string `json:"error_detail"`
	GuildID       string `json:"guild_id"`
}

// DMSentPayload is the payload for the DMSent event.
type DMSentPayload struct {
	UserID  string `json:"user_id"`
	GuildID string `json:"guild_id"`
}

// DMErrorPayload is a common payload for DM-related errors.
type DMErrorPayload struct {
	UserID      string `json:"user_id"`
	ErrorDetail string `json:"error_detail"`
	GuildID     string `json:"guild_id"`
}
type SendDMPayload struct {
	UserID  string `json:"user_id"`
	Message string `json:"message"`
	GuildID string `json:"guild_id"`
}
type InteractionResponse struct {
	InteractionID string
	Token         string
	Message       string
	RetryData     *RetryData
}
type RetryData struct {
	Title       string
	Description string
	StartTime   string
	Location    string
}

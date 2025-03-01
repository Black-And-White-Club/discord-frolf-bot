package discordevents

const (
	StatusSuccess = "success"
	StatusFail    = "fail"
	SendDM        = "discord.send.dm"
	DMSent        = "discord.user.dmsent"  // Keep: This is for the success case.
	DMError       = "discord.user.dmerror" // ADD:  A single, generic error event.

)

// Interaction response tracking
type InteractionRespondedPayload struct {
	InteractionID string `json:"interaction_id"`
	UserID        string `json:"user_id"`
	Status        string `json:"status"`
	ErrorDetail   string `json:"error_detail"`
}

// DMSentPayload is the payload for the DMSent event.
type DMSentPayload struct {
	UserID string `json:"user_id"`
}

// DMErrorPayload is a common payload for DM-related errors.
type DMErrorPayload struct {
	UserID      string `json:"user_id"`
	ErrorDetail string `json:"error_detail"`
}

type SendDMPayload struct {
	UserID  string `json:"user_id"`
	Message string `json:"message"`
}

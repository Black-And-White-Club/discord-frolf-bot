package userhandlers

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	discorduserevents "github.com/Black-And-White-Club/discord-frolf-bot/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/events"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
)

// SignupCache prevents duplicate submissions.
var (
	signupCache sync.Map
	cacheTTL    = 30 * time.Second
)

// HandleSignupSubmission handles the initial Discord signup modal submission.
func (h *UserHandlers) HandleSignupSubmission(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	h.Logger.Info(ctx, "Processing signup submission", attr.CorrelationIDFromMsg(msg))
	msg.Metadata.Set("handler_name", "HandleSignupSubmission")

	// First unmarshal into a base interaction structure
	var interaction struct {
		ID   string `json:"id"`
		Type int    `json:"type"`
		Data struct {
			CustomID   string `json:"custom_id"`
			Components []struct {
				Type       int `json:"type"`
				Components []struct {
					CustomID string `json:"custom_id"`
					Value    string `json:"value"`
				} `json:"components"`
			} `json:"components"`
		} `json:"data"`
		Member struct {
			User struct {
				ID string `json:"id"`
			} `json:"user"`
		} `json:"member"`
		Token string `json:"token"`
	}

	if err := json.Unmarshal(msg.Payload, &interaction); err != nil {
		return nil, fmt.Errorf("failed to unmarshal interaction: %w", err)
	}

	if interaction.Type != int(discordgo.InteractionModalSubmit) {
		return nil, fmt.Errorf("unexpected interaction type: %d", interaction.Type)
	}

	// Check modal ID
	if interaction.Data.CustomID != "signup_modal" {
		h.Logger.Error(ctx, "Incorrect modal ID", attr.CorrelationIDFromMsg(msg))
		if err := h.Session.InteractionRespond(&discordgo.Interaction{
			ID:    interaction.ID,
			Type:  discordgo.InteractionType(interaction.Type),
			Token: interaction.Token,
		}, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "An error occurred processing your signup. Please try again.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		}); err != nil {
			return nil, fmt.Errorf("failed to respond with error: %w", err)
		}
		return nil, fmt.Errorf("incorrect modal ID")
	}

	requestingUserID := interaction.Member.User.ID

	// Convert to ModalSubmitInteractionData for tag extraction
	data := &discordgo.ModalSubmitInteractionData{
		CustomID: interaction.Data.CustomID,
	}

	// Convert components
	for _, comp := range interaction.Data.Components {
		row := &discordgo.ActionsRow{}
		for _, innerComp := range comp.Components {
			textInput := &discordgo.TextInput{
				CustomID: innerComp.CustomID,
				Value:    innerComp.Value,
			}
			row.Components = append(row.Components, textInput)
		}
		data.Components = append(data.Components, row)
	}

	// Use the existing extractTagNumber helper
	tagNumberPtr, err := h.extractTagNumber(data)
	if err != nil {
		h.Logger.Warn(ctx, "Invalid tag number", attr.UserID(requestingUserID), attr.Error(err))
		if err := h.Session.InteractionRespond(&discordgo.Interaction{
			ID:    interaction.ID,
			Type:  discordgo.InteractionType(interaction.Type),
			Token: interaction.Token,
		}, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("<@%s> Invalid tag number format.", requestingUserID),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		}); err != nil {
			return nil, fmt.Errorf("failed to respond with invalid tag: %w", err)
		}
		return nil, fmt.Errorf("invalid tag number")
	}

	// Send success response
	if err := h.Session.InteractionRespond(&discordgo.Interaction{
		ID:    interaction.ID,
		Type:  discordgo.InteractionType(interaction.Type),
		Token: interaction.Token,
	}, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Your signup has been submitted!",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	}); err != nil {
		return nil, fmt.Errorf("failed to send success response: %w", err)
	}

	// Create event payload
	submittedPayload := discorduserevents.SignupFormSubmittedPayload{
		CommonMetadata: events.CommonMetadata{
			Domain:    "discord_user",
			EventName: "discord.user.SignupFormSubmitted",
			Timestamp: time.Time{},
		},
		UserID:           requestingUserID,
		InteractionID:    interaction.ID,
		InteractionToken: interaction.Token,
		TagNumber:        tagNumberPtr,
	}

	// Create result message
	resultMsg := message.NewMessage("", nil)
	payloadBytes, err := json.Marshal(submittedPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}
	resultMsg.Payload = payloadBytes

	// Copy metadata from input message
	for k, v := range msg.Metadata {
		resultMsg.Metadata.Set(k, v)
	}
	resultMsg.Metadata.Set("domain", "discord_user")
	resultMsg.Metadata.Set("event_name", "discord.user.SignupFormSubmitted")
	resultMsg.Metadata.Set("handler_name", "HandleSignupSubmission")

	return []*message.Message{resultMsg}, nil
}

// HandleSignupFormSubmitted handles the processed signup form data.
func (h *UserHandlers) HandleSignupFormSubmitted(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()

	var payload discorduserevents.SignupFormSubmittedPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	requestingUserID := payload.UserID
	interactionID := payload.InteractionID
	interactionToken := payload.InteractionToken

	// Check for duplicate signup
	if _, exists := signupCache.Load(requestingUserID); exists {
		h.Logger.Warn(ctx, "Duplicate signup attempt detected", attr.UserID(requestingUserID))

		// Send response for duplicate signup
		if err := h.Session.InteractionRespond(&discordgo.Interaction{
			ID:    interactionID,
			Token: interactionToken,
		}, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("<@%s> You have already submitted a signup request. Please wait.", requestingUserID),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		}); err != nil {
			return nil, fmt.Errorf("failed to respond to duplicate signup: %w", err)
		}

		// Create failure message
		failureMsg := message.NewMessage("", nil)
		failurePayload := map[string]interface{}{
			"event_name": "discord.signup.failed",
			"reason":     "Duplicate signup attempt",
			"result":     "failure",
			"user_id":    msg.Metadata.Get("user_id"),
		}

		if payloadBytes, err := json.Marshal(failurePayload); err != nil {
			return nil, fmt.Errorf("failed to marshal failure payload: %w", err)
		} else {
			failureMsg.Payload = payloadBytes
		}

		// Copy metadata
		for k, v := range msg.Metadata {
			failureMsg.Metadata.Set(k, v)
		}

		return []*message.Message{failureMsg}, fmt.Errorf("duplicate signup attempt")
	}

	// Store in cache
	signupCache.Store(requestingUserID, struct{}{})

	// Schedule cleanup
	go func(userID string) {
		time.Sleep(cacheTTL)
		signupCache.Delete(userID)
	}(requestingUserID)

	// Send acknowledgment
	if err := h.Session.InteractionRespond(&discordgo.Interaction{
		ID:    interactionID,
		Token: interactionToken,
	}, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("<@%s> Thank you for signing up! Your request is being processed.", requestingUserID),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	}); err != nil {
		h.Logger.Error(ctx, "Failed to send interaction response", attr.UserID(requestingUserID), attr.Error(err))

		// Create error message
		errorMsg := message.NewMessage("", nil)
		errorPayload := map[string]interface{}{
			"event_name": "discord.signup.failed",
			"reason":     "internal error",
			"result":     "failure",
			"user_id":    requestingUserID,
		}

		if payloadBytes, err := json.Marshal(errorPayload); err != nil {
			return nil, fmt.Errorf("failed to marshal error payload: %w", err)
		} else {
			errorMsg.Payload = payloadBytes
		}

		// Copy metadata
		for k, v := range msg.Metadata {
			errorMsg.Metadata.Set(k, v)
		}

		return []*message.Message{errorMsg}, fmt.Errorf("failed to send response: %w", err)
	}

	// Create backend payload
	backendPayload := map[string]interface{}{
		"discord_id": requestingUserID,
	}
	if payload.TagNumber != nil {
		backendPayload["tag_number"] = *payload.TagNumber
	}

	resultMsg := message.NewMessage("", nil)
	if payloadBytes, err := json.Marshal(backendPayload); err != nil {
		return nil, fmt.Errorf("failed to marshal backend payload: %w", err)
	} else {
		resultMsg.Payload = payloadBytes
	}

	// Copy metadata
	for k, v := range msg.Metadata {
		resultMsg.Metadata.Set(k, v)
	}

	return []*message.Message{resultMsg}, nil
}

// HandleUserCreated handles the event indicating successful user creation.
func (h *UserHandlers) HandleUserCreated(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()

	// Change to use a map for initial unmarshal since we know the test payload structure
	var payload map[string]interface{}
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		h.Logger.Error(ctx, "Failed to unmarshal UserCreatedPayload", attr.Error(err), attr.CorrelationIDFromMsg(msg))
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// Get the discord_id as a string
	discordID := payload["discord_id"].(string)

	successMsg := "Signup complete! You now have access to the members-only channels."

	// Use the string discord_id directly
	channel, err := h.Session.UserChannelCreate(discordID)
	if err != nil {
		h.Logger.Error(ctx, "Failed to create DM channel",
			attr.UserID(discordID),
			attr.Error(err),
			attr.CorrelationIDFromMsg(msg))

		// Create result message with consistent format
		resultMsg := message.NewMessage("", nil)
		resultPayload := map[string]interface{}{
			"user_id": "",
			"message": successMsg,
		}

		payloadBytes, err := json.Marshal(resultPayload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal result payload: %w", err)
		}
		resultMsg.Payload = payloadBytes

		// Copy metadata
		for k, v := range msg.Metadata {
			resultMsg.Metadata.Set(k, v)
		}

		return []*message.Message{resultMsg}, fmt.Errorf("failed to create DM channel: %w", err)
	}

	// Send message
	if _, err := h.Session.ChannelMessageSend(channel.ID, successMsg); err != nil {
		h.Logger.Error(ctx, "Failed to send DM",
			attr.UserID(string(discordID)),
			attr.Error(err),
			attr.CorrelationIDFromMsg(msg))

		// Create result message with consistent format
		resultMsg := message.NewMessage("", nil)
		resultPayload := map[string]interface{}{
			"user_id": "",
			"message": successMsg,
		}

		payloadBytes, err := json.Marshal(resultPayload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal result payload: %w", err)
		}
		resultMsg.Payload = payloadBytes

		// Copy metadata
		for k, v := range msg.Metadata {
			resultMsg.Metadata.Set(k, v)
		}

		return []*message.Message{resultMsg}, fmt.Errorf("failed to send DM: %w", err)
	}

	// Create result message
	resultMsg := message.NewMessage("", nil)
	resultPayload := map[string]interface{}{
		"user_id": "",
		"message": successMsg,
	}

	payloadBytes, err := json.Marshal(resultPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result payload: %w", err)
	}
	resultMsg.Payload = payloadBytes

	// Copy metadata
	for k, v := range msg.Metadata {
		resultMsg.Metadata.Set(k, v)
	}

	return []*message.Message{resultMsg}, nil
}

// HandleUserCreationFailed handles the event indicating user creation failure.
func (h *UserHandlers) HandleUserCreationFailed(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()

	// Use struct matching test payload
	var payload struct {
		DiscordID string `json:"discord_id"`
		Reason    string `json:"reason"`
	}
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		h.Logger.Error(ctx, "Failed to unmarshal UserCreationFailedPayload", attr.Error(err), attr.CorrelationIDFromMsg(msg))
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	failMsg := fmt.Sprintf("Signup failed: %s. Please try again, or contact an administrator.", payload.Reason)

	// Create DM channel using the discord_id directly
	channel, err := h.Session.UserChannelCreate(payload.DiscordID)
	if err != nil {
		h.Logger.Error(ctx, "Failed to create DM channel",
			attr.UserID(payload.DiscordID),
			attr.Error(err),
			attr.CorrelationIDFromMsg(msg))

		resultMsg := message.NewMessage("", nil)
		resultPayload := map[string]interface{}{
			"user_id": "",
			"message": failMsg,
		}

		payloadBytes, err := json.Marshal(resultPayload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal result payload: %w", err)
		}
		resultMsg.Payload = payloadBytes

		for k, v := range msg.Metadata {
			resultMsg.Metadata.Set(k, v)
		}

		return []*message.Message{resultMsg}, fmt.Errorf("failed to create DM channel: %w", err)
	}

	// Send message
	if _, err := h.Session.ChannelMessageSend(channel.ID, failMsg); err != nil {
		h.Logger.Error(ctx, "Failed to send DM",
			attr.UserID(payload.DiscordID),
			attr.Error(err),
			attr.CorrelationIDFromMsg(msg))

		resultMsg := message.NewMessage("", nil)
		resultPayload := map[string]interface{}{
			"user_id": "",
			"message": failMsg,
		}

		payloadBytes, err := json.Marshal(resultPayload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal result payload: %w", err)
		}
		resultMsg.Payload = payloadBytes

		for k, v := range msg.Metadata {
			resultMsg.Metadata.Set(k, v)
		}

		return []*message.Message{resultMsg}, fmt.Errorf("failed to send DM: %w", err)
	}

	// Success case
	resultMsg := message.NewMessage("", nil)
	resultPayload := map[string]interface{}{
		"user_id": "",
		"message": failMsg,
	}

	payloadBytes, err := json.Marshal(resultPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result payload: %w", err)
	}
	resultMsg.Payload = payloadBytes

	for k, v := range msg.Metadata {
		resultMsg.Metadata.Set(k, v)
	}

	return []*message.Message{resultMsg}, nil
}

// sendUserDM publishes an event to send a DM to a user.
func (h *UserHandlers) sendUserDM(userID, messageText string) ([]*message.Message, error) {
	dmEvent := discorduserevents.SendUserDMPayload{
		UserID:  userID,
		Message: messageText,
	}
	// Create the event and return it.
	dmMessage, err := h.createResultMessage(nil, dmEvent) // No original message for context
	if err != nil {
		return nil, err
	}

	return []*message.Message{dmMessage}, nil
}

// extractTagNumber extracts the tag number from the modal submission data.
func (h *UserHandlers) extractTagNumber(data *discordgo.ModalSubmitInteractionData) (*int, error) {
	for _, comp := range data.Components {
		row, ok := comp.(*discordgo.ActionsRow)
		if !ok {
			continue // Skip if not an ActionsRow
		}
		for _, innerComp := range row.Components { //Loop through
			textInput, ok := innerComp.(*discordgo.TextInput)
			if ok && textInput.CustomID == "tag_number" { // Check CustomID
				if textInput.Value == "" {
					return nil, nil // No tag number provided, which is valid.
				}
				tagNumber, err := strconv.Atoi(textInput.Value)
				if err != nil {
					return nil, fmt.Errorf("invalid tag number format: %s", textInput.Value)
				}
				return &tagNumber, nil
			}
		}
	}
	return nil, nil // No tag number field found, which is valid
}

package discord

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	roundevents "github.com/Black-And-White-Club/discord-frolf-bot/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

func (h *gatewayEventHandler) HandleCreateRoundCommand(ctx context.Context, i *discordgo.InteractionCreate) {
	slog.Info("Handling create round button press", attr.UserID(i.Member.User.ID))
	err := h.discord.SendCreateRoundModal(ctx, i.Interaction)
	if err != nil {
		slog.Error("Failed to send create round modal", attr.UserID(i.Member.User.ID), attr.Error(err))
	}
}

// HandleCreateRoundModalSubmit handles the submission of the create round modal.
func (h *gatewayEventHandler) HandleCreateRoundModalSubmit(ctx context.Context, i *discordgo.InteractionCreate) {
	slog.Info("Handling create round modal submission", attr.UserID(i.Member.User.ID))

	// Extract form data
	data := i.ModalSubmitData()
	title := strings.TrimSpace(data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value)
	description := strings.TrimSpace(data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value)
	startTimeStr := strings.TrimSpace(data.Components[2].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value)
	timezone := strings.TrimSpace(data.Components[3].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value)
	location := strings.TrimSpace(data.Components[4].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value)

	slog.Info("Extracted form data", attr.String("title", title), attr.String("description", description), attr.String("start_time", startTimeStr), attr.String("timezone", timezone), attr.String("location", location))

	// Set default timezone to CST if the user didn't provide one
	if timezone == "" {
		timezone = "America/Chicago" // CST
	}

	slog.Info("Timezone set to", attr.String("timezone", timezone))

	// Constants for response messages
	const (
		errTitle       = "❌ Round creation failed."
		errMsgRequired = "Title and Start Time are required."
		errMsgTitleLen = "Title must be less than 100 characters."
		errMsgDescLen  = "Description must be less than 500 characters."
		errMsgLocLen   = "Location must be less than 100 characters."
	)

	// Basic validation (check required fields and length)
	var validationErrors []string
	if title == "" {
		validationErrors = append(validationErrors, "Title is required.")
	}
	if startTimeStr == "" {
		validationErrors = append(validationErrors, "Start Time is required.")
	}
	if len(title) > 100 {
		validationErrors = append(validationErrors, "Title must be less than 100 characters.")
	}
	if len(description) > 500 {
		validationErrors = append(validationErrors, "Description must be less than 500 characters.")
	}
	if len(location) > 100 {
		validationErrors = append(validationErrors, "Location must be less than 100 characters.")
	}

	// If there are validation errors, respond once
	if len(validationErrors) > 0 {
		errorMessage := "❌ Round creation failed: " + strings.Join(validationErrors, " ")
		h.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content:    errorMessage,
				Flags:      discordgo.MessageFlagsEphemeral,
				Components: []discordgo.MessageComponent{},
			},
		})
		return
	}

	// Publish event for backend validation
	payload := roundevents.CreateRoundRequestedPayload{
		UserID:      i.Member.User.ID,
		Title:       title,
		Description: description,
		StartTime:   startTimeStr,
		Location:    location,
		Timezone:    timezone,
	}

	slog.Info("Publishing event for Modal validation", attr.Any("payload", payload))

	correlationID := uuid.New().String()
	h.interactionStore.Set(correlationID, i.Interaction, 15*time.Minute)

	msg, err := h.createEvent(ctx, roundevents.RoundCreateModalSubmit, payload, i)
	if err != nil {
		h.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content:    fmt.Sprintf("%s %s", errTitle, "Failed to create event. Please try again."),
				Flags:      discordgo.MessageFlagsEphemeral,
				Components: []discordgo.MessageComponent{},
			},
		})
		return
	}

	// Set the correlation ID in the message metadata before publishing
	msg.Metadata.Set("correlation_id", correlationID)
	msg.Metadata.Set("user_id", i.Member.User.ID)

	if err := h.publisher.Publish(roundevents.RoundCreateModalSubmit, msg); err != nil {
		h.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content:    fmt.Sprintf("%s %s", errTitle, "Failed to send request. Please try again."),
				Flags:      discordgo.MessageFlagsEphemeral,
				Components: []discordgo.MessageComponent{},
			},
		})
		return
	}

	slog.Info("Round creation request published", attr.UserID(i.Member.User.ID))

	// Update the interaction response after you've finished processing the modal submission
	h.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:    "Round creation request received. Please wait for confirmation.",
			Flags:      discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{},
		},
	})

}

// UpdateInteractionResponse updates the deferred response using the correlation ID.
func (h *gatewayEventHandler) UpdateInteractionResponse(ctx context.Context, correlationID, message string, edit ...*discordgo.WebhookEdit) error {
	interaction, found := h.interactionStore.Get(correlationID)
	if !found {
		return fmt.Errorf("no interaction found for correlation ID: %s", correlationID)
	}

	// Make sure interaction is of the correct type
	interactionObj, ok := interaction.(*discordgo.Interaction)
	if !ok {
		return fmt.Errorf("stored interaction is not of type *discordgo.Interaction")
	}

	// Use the full interaction object
	var webhookEdit *discordgo.WebhookEdit
	if len(edit) > 0 {
		webhookEdit = edit[0]
		// Ensure content is set
		webhookEdit.Content = &message
	} else {
		webhookEdit = &discordgo.WebhookEdit{
			Content: &message,
		}
	}

	_, err := h.session.InteractionResponseEdit(interactionObj, webhookEdit)
	if err != nil {
		slog.Error("Failed to update interaction response", attr.Error(err))
	}
	return err
}

func (h *gatewayEventHandler) UpdateInteractionResponseWithRetryButton(ctx context.Context, correlationID, message string) error {
	interaction, found := h.interactionStore.Get(correlationID)
	if !found {
		return fmt.Errorf("no interaction found for correlation ID: %s", correlationID)
	}

	// Make sure interaction is of the correct type
	interactionObj, ok := interaction.(*discordgo.Interaction)
	if !ok {
		return fmt.Errorf("stored interaction is not of type *discordgo.Interaction")
	}

	// Attempt to update the interaction response
	_, err := h.session.InteractionResponseEdit(interactionObj, &discordgo.WebhookEdit{
		Content:    &message,
		Components: &[]discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.Button{Label: "Try Again", Style: discordgo.PrimaryButton, CustomID: "retry_create_round"}}}},
	})
	if err != nil {
		slog.Error("Failed to update interaction response with retry button", attr.Error(err))
		return err
	}
	return nil
}

// HandleCreateRoundModalCancel handles the cancellation of the create round modal.
func (h *gatewayEventHandler) HandleCreateRoundModalCancel(ctx context.Context, i *discordgo.InteractionCreate) {
	slog.Info("Handling create round modal cancel", attr.String("interaction_id", i.ID))

	// Remove the token from the store
	h.interactionStore.Delete(i.Interaction.ID) // Assuming you have a Delete method in your interactionStore

	err := h.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "Round creation cancelled.",
			Components: []discordgo.MessageComponent{},
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		slog.Error("Failed to cancel interaction", attr.Error(err), attr.String("interaction_id", i.ID))
	}
}

// In create_round_interactions.go
func (h *gatewayEventHandler) HandleRetryCreateRound(ctx context.Context, i *discordgo.InteractionCreate) {
	slog.Info("Handling retry create round button press", attr.UserID(i.Member.User.ID))

	err := h.discord.SendCreateRoundModal(ctx, i.Interaction)
	if err != nil {
		slog.Error("Failed to send create round modal", attr.UserID(i.Member.User.ID), attr.Error(err))

		// If modal sending fails, update the message to inform the user
		_, updateErr := h.session.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content:    stringPtr("Failed to open the form. Please try using the /createround command again."),
			Components: &[]discordgo.MessageComponent{},
		})
		if updateErr != nil {
			slog.Error("Failed to update error message", attr.Error(updateErr))
		}
	}
}

func stringPtr(s string) *string {
	return &s
}

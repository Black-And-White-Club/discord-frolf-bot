package createround

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	roundevents "github.com/Black-And-White-Club/discord-frolf-bot/app/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

func (crm *createRoundManager) SendCreateRoundModal(ctx context.Context, i *discordgo.InteractionCreate) error {
	crm.logger.Info(ctx, "Sending create round modal", attr.UserID(i.Member.User.ID))
	// Send the modal as the initial response
	err := crm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			Title:    "Create Round",
			CustomID: "create_round_modal",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "title",
							Label:       "Title",
							Style:       discordgo.TextInputShort,
							Placeholder: "Enter the round title",
							Required:    true,
							MaxLength:   100,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "description",
							Label:       "Description",
							Style:       discordgo.TextInputParagraph,
							Placeholder: "Enter a description (optional)",
							Required:    false,
							MaxLength:   500,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "start_time",
							Label:       "Start Time",
							Style:       discordgo.TextInputShort,
							Placeholder: "YYYY-MM-DD HH:MM",
							Required:    true,
							MaxLength:   30,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "timezone",
							Label:       "Timezone (Optional)",
							Style:       discordgo.TextInputShort,
							Placeholder: "America/Chicago (CST)",
							Required:    false,
							MaxLength:   50,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "location",
							Label:       "Location",
							Style:       discordgo.TextInputShort,
							Placeholder: "Enter the location (optional)",
							Required:    false,
							MaxLength:   100,
						},
					},
				},
			},
		},
	})
	if err != nil {
		crm.logger.Error(ctx, "Failed to send create round modal", attr.UserID(i.Member.User.ID), attr.Error(err))
		return fmt.Errorf("failed to send create round modal: %w", err)
	}
	return nil
}

// HandleCreateRoundModalSubmit handles the submission of the create round modal.
func (crm *createRoundManager) HandleCreateRoundModalSubmit(ctx context.Context, i *discordgo.InteractionCreate) {
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
		errTitle       = "❌ Round creation failecrm."
		errMsgRequired = "Title and Start Time are requirecrm."
		errMsgTitleLen = "Title must be less than 100 characters."
		errMsgDescLen  = "Description must be less than 500 characters."
		errMsgLocLen   = "Location must be less than 100 characters."
	)

	// Basic validation (check required fields and length)
	var validationErrors []string
	if title == "" {
		validationErrors = append(validationErrors, "Title is requirecrm.")
	}
	if startTimeStr == "" {
		validationErrors = append(validationErrors, "Start Time is requirecrm.")
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
		crm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
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
	crm.interactionStore.Set(correlationID, i.Interaction, 15*time.Minute)

	msg, err := crm.createEvent(ctx, roundevents.RoundCreateModalSubmit, payload, i)
	if err != nil {
		crm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
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

	if err := crm.publisher.Publish(roundevents.RoundCreateModalSubmit, msg); err != nil {
		crm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
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
	crm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:    "Round creation request receivecrm. Please wait for confirmation.",
			Flags:      discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{},
		},
	})

}

// HandleCreateRoundModalCancel handles the cancellation of the create round modal.
func (crm *createRoundManager) HandleCreateRoundModalCancel(ctx context.Context, i *discordgo.InteractionCreate) {
	slog.Info("Handling create round modal cancel", attr.String("interaction_id", i.ID))

	// Remove the token from the store
	crm.interactionStore.Delete(i.Interaction.ID)

	err := crm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "Round creation cancellecrm.",
			Components: []discordgo.MessageComponent{},
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		slog.Error("Failed to cancel interaction", attr.Error(err), attr.String("interaction_id", i.ID))
	}
}

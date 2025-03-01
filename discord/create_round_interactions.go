package discord

import (
	"context"
	"fmt"
	"strings"

	roundevents "github.com/Black-And-White-Club/discord-frolf-bot/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

func (h *gatewayEventHandler) HandleCreateRoundCommand(ctx context.Context, i *discordgo.InteractionCreate) {
	h.logger.Info(ctx, "Handling create round button press", attr.UserID(i.Member.User.ID))
	err := h.discord.SendCreateRoundModal(ctx, i.Interaction)
	if err != nil {
		h.logger.Error(ctx, "Failed to send create round modal", attr.UserID(i.Member.User.ID), attr.Error(err))
	}
}

// Add this new function to handle the create round modal submission
func (h *gatewayEventHandler) HandleCreateRoundModalSubmit(ctx context.Context, i *discordgo.InteractionCreate) {
	h.logger.Info(ctx, "Handling create round modal submission", attr.UserID(i.Member.User.ID))

	// Extract form data
	data := i.ModalSubmitData()
	title := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	description := data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	startTimeStr := data.Components[2].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	location := data.Components[4].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

	// Basic validation (check required fields)
	if title == "" || startTimeStr == "" {
		h.respondWithRetry(ctx, i, "❌ Title, Start Time, and End Time are required.", i.Member.User.ID, title, description, startTimeStr, location)
		return
	}

	// Publish event for backend validation
	payload := roundevents.CreateRoundRequestedPayload{
		UserID:      i.Member.User.ID,
		Title:       title,
		Description: description,
		StartTime:   startTimeStr,
		Location:    location,
	}

	msg, err := h.createEvent(ctx, roundevents.CreateRoundRequestedTopic, payload, i)
	if err != nil {
		h.respondWithRetry(ctx, i, "❌ Failed to create event. Please try again.", i.Member.User.ID, title, description, startTimeStr, location)
		return
	}

	if err := h.publisher.Publish(roundevents.CreateRoundRequestedTopic, msg); err != nil {
		h.respondWithRetry(ctx, i, "❌ Failed to send request. Please try again.", i.Member.User.ID, title, description, startTimeStr, location)
		return
	}

	// Send ephemeral success response
	h.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "✅ Your round creation request has been sent for validation. We’ll notify you once it's processed.",
			Flags:   discordgo.MessageFlagsEphemeral, // Private response
		},
	})

	h.logger.Info(ctx, "Round creation request published", attr.UserID(i.Member.User.ID))
}

// Helper function to send error responses
func (h *gatewayEventHandler) respondWithError(ctx context.Context, i *discordgo.InteractionCreate, message string) {
	h.logger.Error(ctx, "Modal Submission Error")
	h.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "❌ " + message,
			Flags:   discordgo.MessageFlagsEphemeral, // Private response
		},
	})
}

func (h *gatewayEventHandler) respondWithRetry(ctx context.Context, i *discordgo.InteractionCreate, errorMessage, userID, title, description, startTime, location string) {
	retryCustomID := fmt.Sprintf("retry_create_round|%s|%s|%s|%s|%s",
		userID, title, description, startTime, location)

	h.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: errorMessage,
			Flags:   discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							CustomID: retryCustomID,
							Label:    "Retry",
							Style:    discordgo.PrimaryButton,
						},
					},
				},
			},
		},
	})
}

type RetryData struct {
	Title       string
	Description string
	StartTime   string
	Location    string
}

// Parses the previous user input from the button's CustomID
func parseRetryData(customID string) RetryData {
	// Example format: "retry_create_round|Title|Description|StartTime|Location"
	parts := strings.Split(customID, "|")
	if len(parts) != 6 {
		return RetryData{}
	}

	return RetryData{
		Title:       parts[1],
		Description: parts[2],
		StartTime:   parts[3],
		Location:    parts[4],
	}
}

func (h *gatewayEventHandler) HandleRetryCreateRound(ctx context.Context, i *discordgo.InteractionCreate) {
	h.logger.Info(ctx, "Handling retry for create round modal", attr.UserID(i.Member.User.ID))

	// Extract previous input from CustomID
	data := i.MessageComponentData().CustomID
	parsedData := parseRetryData(data)

	// Respond by reopening the modal with pre-filled values
	h.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "create_round_modal",
			Title:    "Create a New Round",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID: "title",
							Label:    "Title",
							Style:    discordgo.TextInputShort,
							Required: true,
							Value:    parsedData.Title,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID: "description",
							Label:    "Description",
							Style:    discordgo.TextInputParagraph,
							Required: false,
							Value:    parsedData.Description,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID: "start_time",
							Label:    "Start Time",
							Style:    discordgo.TextInputShort,
							Required: true,
							Value:    parsedData.StartTime,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID: "location",
							Label:    "Location",
							Style:    discordgo.TextInputShort,
							Required: false,
							Value:    parsedData.Location,
						},
					},
				},
			},
		},
	})

	h.logger.Info(ctx, "Reopened create round modal with previous data", attr.UserID(i.Member.User.ID))
}

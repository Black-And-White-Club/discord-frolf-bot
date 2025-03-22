package signup

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

func (sm *signupManager) SendSignupModal(ctx context.Context, i *discordgo.InteractionCreate) error {
	if i == nil || i.Interaction == nil || i.User == nil {
		return fmt.Errorf("interaction or user is nil")
	}

	slog.Info("Preparing to send signup modal", attr.UserID(i.User.ID))

	err := sm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "signup_modal",
			Title:    "Frolf Club Signup",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "tag_number",
							Label:       "Tag Number (Optional)",
							Style:       discordgo.TextInputShort,
							Placeholder: "Enter your desired tag number (e.g., 13)",
							Required:    false,
							MaxLength:   3,
							MinLength:   0,
							Value:       "",
						},
					},
				},
			},
		},
	})
	if err != nil {
		slog.Error("❌ Failed to send signup modal", attr.UserID(i.User.ID), attr.Error(err))
		return fmt.Errorf("failed to send signup modal: %w", err)
	}

	slog.Info("✅ Signup modal successfully sent!", attr.UserID(i.User.ID))
	return nil
}

// HandleSignupModalSubmit handles the submission of the signup modal.
func (sm *signupManager) HandleSignupModalSubmit(ctx context.Context, i *discordgo.InteractionCreate) {
	if i == nil || i.Interaction == nil {
		slog.Error("❌ Interaction is nil")
		return
	}
	if i.Interaction.ID == "" {
		slog.Error("❌ Interaction ID is missing")
		return
	}
	if i.Interaction.Token == "" {
		slog.Error("❌ Interaction Token is missing")
		return
	}
	if i.Interaction.Data == nil {
		slog.Error("❌ Interaction Data is missing")
		return
	}

	// Check if the interaction is a modal submission
	if i.Interaction.Type != discordgo.InteractionModalSubmit {
		slog.Error("❌ Interaction is not a modal submission")
		return
	}

	// Extract the user ID based on whether the interaction is in a guild or DM
	var userID string
	if i.Member != nil {
		// Interaction is in a guild
		userID = i.Member.User.ID
	} else if i.User != nil {
		// Interaction is in a DM
		userID = i.User.ID
	} else {
		slog.Error("❌ Unable to determine user ID: both Member and User are nil")
		return
	}

	slog.Info("HandlingModalSubmit", attr.String("custom_id", i.ModalSubmitData().CustomID))

	// Acknowledge the modal submission
	err := sm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "✅ Signup request submitted successfully! Processing...",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		slog.Error("❌ Failed to acknowledge modal submission", attr.Error(err))
		return
	}

	data := i.ModalSubmitData()
	tagNumberPtr, err := sm.extractTagNumber(data)
	if err != nil {
		slog.Warn("⚠️ Invalid tag number", attr.Error(err))
		return
	}

	payload := userevents.UserSignupRequestPayload{
		DiscordID: usertypes.DiscordID(userID),
		TagNumber: tagNumberPtr,
	}

	correlationID := uuid.New().String()
	slog.Info("Storing interaction reference in cache", attr.String("correlation_id", correlationID))

	sm.interactionStore.Set(correlationID, i.Interaction, 10*time.Minute)

	slog.Info("Creating event message...")

	msg, err := sm.createEvent(ctx, userevents.UserSignupRequest, payload, i)
	if err != nil {
		slog.Error("❌ Failed to create event", attr.Error(err))
		return
	}

	msg.Metadata.Set("correlation_id", correlationID)
	msg.Metadata.Set("user_id", userID)

	slog.Info("Publishing signup form submitted event...")
	if err := sm.publisher.Publish(userevents.UserSignupRequest, msg); err != nil {
		slog.Error("❌ Failed to publish event", attr.Error(err))
		return
	}
	slog.Info("Signup form event published successfully")
}

// extractTagNumber extracts the tag number from the modal submission data.
func (sm *signupManager) extractTagNumber(data discordgo.ModalSubmitInteractionData) (*int, error) {
	for _, comp := range data.Components {
		row, ok := comp.(*discordgo.ActionsRow)
		if !ok {
			continue
		}
		for _, innerComp := range row.Components {
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

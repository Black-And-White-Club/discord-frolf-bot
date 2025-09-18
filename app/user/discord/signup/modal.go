package signup

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

func (sm *signupManager) SendSignupModal(ctx context.Context, i *discordgo.InteractionCreate) (SignupOperationResult, error) {
	if ctx.Err() != nil {
		return SignupOperationResult{Error: ctx.Err()}, ctx.Err()
	}
	return sm.operationWrapper(ctx, "send_signup_modal", func(ctx context.Context) (SignupOperationResult, error) {
		// Early validation - test expects these to return SignupOperationResult{Error: err}, err
		if i == nil || i.Interaction == nil {
			return SignupOperationResult{Error: errors.New("interaction is nil or incomplete")}, errors.New("interaction is nil or incomplete")
		}

		// Check for user in either Member or direct User field
		userID := ""
		if i.Interaction.Member != nil && i.Interaction.Member.User != nil {
			userID = i.Interaction.Member.User.ID
		} else if i.Interaction.User != nil {
			userID = i.Interaction.User.ID
		} else {
			return SignupOperationResult{Error: errors.New("user is nil in interaction")}, errors.New("user is nil in interaction")
		}

		// Store the interaction AFTER validation checks
		err := sm.interactionStore.Set(i.Interaction.ID, i.Interaction, 10*time.Minute)
		if err != nil {
			return SignupOperationResult{}, fmt.Errorf("failed to store interaction: %w", err)
		}

		sm.logger.InfoContext(ctx, "Preparing to send signup modal",
			attr.String("user_id", userID))

		// Get guild ID from button custom ID if not available directly; only read component data for component type
		guildID := i.Interaction.GuildID
		if guildID == "" && i.Interaction.Type == discordgo.InteractionMessageComponent {
			// Extract from button CustomID: "signup_button|userID|guild_id=GUILD_ID"
			customID := ""
			mcd := i.Interaction.MessageComponentData()
			if mcd.CustomID != "" {
				customID = mcd.CustomID
			}
			if strings.Contains(customID, "guild_id=") {
				parts := strings.Split(customID, "guild_id=")
				if len(parts) == 2 {
					guildID = parts[1]
				}
			}
		}

		// Send the modal with your existing components
		// Include guild ID in the custom ID so it's available on modal submit
		customID := fmt.Sprintf("signup_modal|guild_id=%s", guildID)
		err = sm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				CustomID: customID,
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
			sm.logger.ErrorContext(ctx, "Failed to send signup modal",
				attr.String("user_id", userID),
				attr.Error(err))
			// Tests expect nil in result.Error for operation errors
			return SignupOperationResult{}, err
		}

		sm.logger.InfoContext(ctx, "Signup modal successfully sent!",
			attr.String("user_id", userID))
		return SignupOperationResult{Success: "modal sent"}, nil
	})
}

// HandleSignupModalSubmit handles the submission of the signup modal.
func (sm *signupManager) HandleSignupModalSubmit(ctx context.Context, i *discordgo.InteractionCreate) (SignupOperationResult, error) {
	if err := ctx.Err(); err != nil {
		sm.logger.ErrorContext(ctx, "Context cancelled before handling signup modal submit", attr.Error(err))
		return SignupOperationResult{Error: err}, err
	}
	if i == nil {
		sm.logger.ErrorContext(context.Background(), "InteractionCreate is nil in HandleSignupModalSubmit")
		return SignupOperationResult{Error: fmt.Errorf("interaction is nil or incomplete")}, fmt.Errorf("interaction is nil or incomplete")
	}
	if i.Interaction == nil {
		sm.logger.ErrorContext(ctx, "Interaction is nil in HandleSignupModalSubmit")
		return SignupOperationResult{Error: fmt.Errorf("interaction is nil or incomplete")}, fmt.Errorf("interaction is nil or incomplete")
	}

	userID := ""
	if i.Member != nil && i.Member.User != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}

	if userID == "" {
		return SignupOperationResult{Error: fmt.Errorf("user ID is missing")}, fmt.Errorf("user ID is missing")
	}

	// Try to get guildID from interaction, or from the modal custom_id if missing
	guildID := i.Interaction.GuildID
	if guildID == "" {
		customID := i.ModalSubmitData().CustomID
		if strings.HasPrefix(customID, "signup_modal|guild_id=") {
			parts := strings.SplitN(customID, "guild_id=", 2)
			if len(parts) == 2 {
				guildID = parts[1]
			}
		}
		// Fallback to configured default guild ID if still empty (tests rely on config.Discord.GuildID)
		if guildID == "" && sm.config != nil && sm.config.Discord.GuildID != "" {
			guildID = sm.config.Discord.GuildID
		}
	}

	ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, userID)
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "handle_signup_modal_submit")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "modal_submit")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.GuildIDKey, guildID)

	result, err := sm.operationWrapper(ctx, "handle_signup_modal_submit", func(ctx context.Context) (SignupOperationResult, error) {
		sm.logger.InfoContext(ctx, "HandlingModalSubmit", attr.String("custom_id", i.ModalSubmitData().CustomID))

		err := sm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Signup request submitted successfully! Processing...",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			return SignupOperationResult{Error: fmt.Errorf("failed to acknowledge modal submission: %w", err)}, err
		}

		data := i.ModalSubmitData()
		tagNumberPtr, err := sm.extractTagNumber(data)
		if err != nil {
			_ = sm.sendFollowupMessage(i.Interaction, fmt.Sprintf("Invalid tag number: %v", err))
			return SignupOperationResult{Error: fmt.Errorf("invalid tag number: %w", err)}, err
		}

		payload := userevents.UserSignupRequestPayload{
			GuildID:   sharedtypes.GuildID(guildID),
			UserID:    sharedtypes.DiscordID(userID),
			TagNumber: tagNumberPtr,
		}

		correlationID := uuid.New().String()
		sm.interactionStore.Set(i.Interaction.Token, i.Interaction, 10*time.Minute)

		msg, err := BuildUserSignupRequestMessage(ctx, payload, i)
		if err != nil {
			_ = sm.sendFollowupMessage(i.Interaction, "Error processing signup. Try again later.")
			return SignupOperationResult{Error: fmt.Errorf("failed to build signup message: %w", err)}, err
		}
		// Overwrite correlation id with deterministic one for trace continuity if needed
		msg.Metadata.Set("correlation_id", correlationID)
		if err := sm.publisher.Publish(userevents.UserSignupRequest, msg); err != nil {
			_ = sm.sendFollowupMessage(i.Interaction, "Failed to publish signup event.")
			return SignupOperationResult{Error: fmt.Errorf("failed to publish signup event: %w", err)}, err
		}

		return SignupOperationResult{Success: "signup event published"}, nil
	})

	return result, err
}

// extractTagNumber extracts the tag number from the modal submission data.
func (sm *signupManager) extractTagNumber(data discordgo.ModalSubmitInteractionData) (*sharedtypes.TagNumber, error) {
	for _, comp := range data.Components {
		row, ok := comp.(*discordgo.ActionsRow)
		if !ok {
			continue
		}
		for _, innerComp := range row.Components {
			textInput, ok := innerComp.(*discordgo.TextInput)
			if ok && textInput.CustomID == "tag_number" {
				if textInput.Value == "" {
					return nil, nil // Tag number is optional, return nil if empty
				}
				tagNumber, err := strconv.Atoi(textInput.Value)
				if err != nil {
					return nil, fmt.Errorf("tag number must be a valid number, received '%s'", textInput.Value)
				}
				typed := sharedtypes.TagNumber(tagNumber)
				return &typed, nil
			}
		}
	}
	// If the tag_number component was not found, treat it as optional and return nil
	return nil, nil
}

// sendFollowupMessage is a helper to send a followup message to an interaction.
func (sm *signupManager) sendFollowupMessage(interaction *discordgo.Interaction, content string) error {
	// Use FollowupMessageCreate to send a new message after the initial response
	_, err := sm.session.FollowupMessageCreate(interaction, true, &discordgo.WebhookParams{
		Content: content,
		Flags:   discordgo.MessageFlagsEphemeral, // Make the followup message ephemeral (only visible to the user)
	})
	if err != nil {
		sm.logger.Error("Failed to send ephemeral followup message", attr.Error(err))
		return fmt.Errorf("failed to send ephemeral followup message: %w", err)
	}
	sm.logger.Info("Successfully sent ephemeral followup message")
	return nil
}

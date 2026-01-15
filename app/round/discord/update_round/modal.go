package updateround

import (
	"context"
	"fmt"
	"strings"
	"time"

	discordroundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

func (urm *updateRoundManager) SendUpdateRoundModal(ctx context.Context, i *discordgo.InteractionCreate, roundID sharedtypes.RoundID) (UpdateRoundOperationResult, error) {
	var opErr error

	if err := ctx.Err(); err != nil {
		opErr = err
		return UpdateRoundOperationResult{Error: opErr}, opErr
	}

	if i == nil || i.Interaction == nil {
		opErr = fmt.Errorf("interaction is nil or incomplete")
		return UpdateRoundOperationResult{Error: opErr}, opErr
	}

	userID := ""
	if i.Member != nil && i.Member.User != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}

	if userID == "" {
		opErr = fmt.Errorf("user ID is missing")
		return UpdateRoundOperationResult{Error: opErr}, opErr
	}

	// Multi-tenant: Extract guildID from interaction
	guildID := i.Interaction.GuildID
	if guildID == "" {
		// Try to extract from custom ID if present (for edge cases)
		if i.Message != nil && i.Message.ID != "" {
			// Optionally parse from customID if you encode it in the button
		}
	}

	ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, userID)
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "send_update_round_modal")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "command")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.GuildIDKey, guildID)

	result, _ := urm.operationWrapper(ctx, "send_update_round_modal", func(ctx context.Context) (UpdateRoundOperationResult, error) {
		if err := ctx.Err(); err != nil {
			return UpdateRoundOperationResult{Error: err}, err
		}
		urm.logger.InfoContext(ctx, "Sending update round modal",
			attr.UserID(sharedtypes.DiscordID(userID)),
			attr.RoundID("round_id", roundID))

		// ✅ Get the original message ID from the button interaction
		var messageID string
		if i.Message != nil {
			messageID = i.Message.ID
		}

		urm.logger.InfoContext(ctx, "Including message ID in modal CustomID",
			attr.String("message_id", messageID),
			attr.RoundID("round_id", roundID))

		// ✅ Include only roundID and messageID in CustomID (keep under 100 chars)
		// Format: update_round_modal|<roundID>|<messageID>
		customID := fmt.Sprintf("update_round_modal|%s|%s", roundID, messageID)

		// Send the modal as the initial response
		err := urm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				Title:    "Update Round",
				CustomID: customID, // Now includes message ID and guild ID
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
			urm.logger.ErrorContext(ctx, "Failed to send update round modal",
				attr.UserID(sharedtypes.DiscordID(userID)),
				attr.Error(err))
			opErr = fmt.Errorf("failed to send update round modal: %w", err)
			return UpdateRoundOperationResult{Error: opErr}, opErr
		}
		return UpdateRoundOperationResult{Success: "modal sent"}, nil
	})
	return result, opErr
}

// HandleUpdateRoundModalSubmit handles the submission of the update round modal.
func (urm *updateRoundManager) HandleUpdateRoundModalSubmit(ctx context.Context, i *discordgo.InteractionCreate) (UpdateRoundOperationResult, error) {
	if i == nil || i.Interaction == nil {
		return UpdateRoundOperationResult{Error: fmt.Errorf("interaction is nil or incomplete")}, fmt.Errorf("interaction is nil or incomplete")
	}

	userID := ""
	if i.Member != nil && i.Member.User != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}

	if userID == "" {
		return UpdateRoundOperationResult{Error: fmt.Errorf("user ID is missing")}, fmt.Errorf("user ID is missing")
	}

	ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, userID)
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "handle_update_round_modal_submit")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "modal_submit")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.GuildIDKey, i.GuildID)

	return urm.operationWrapper(ctx, "handle_update_round_modal_submit", func(ctx context.Context) (UpdateRoundOperationResult, error) {
		// Check for context cancellation first
		if err := ctx.Err(); err != nil {
			return UpdateRoundOperationResult{Error: err}, err
		}

		urm.logger.InfoContext(ctx, "Handling update round modal submission", attr.UserID(sharedtypes.DiscordID(userID)))

		// Extract roundID and messageID from modal CustomID
		data := i.ModalSubmitData()
		customID := data.CustomID
		urm.logger.InfoContext(ctx, "Processing modal submission", attr.String("custom_id", customID))

		parts := strings.Split(customID, "|")
		if len(parts) < 3 { // Expecting 3 parts: modal_name|roundID|messageID
			err := fmt.Errorf("invalid modal custom_id format: %s (expected 3 parts)", customID)
			urm.logger.ErrorContext(ctx, err.Error())

			// Respond with error message
			if respErr := urm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Invalid modal format. Please try again.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			}); respErr != nil {
				urm.logger.ErrorContext(ctx, "Failed to respond with error", attr.Error(respErr))
			}

			return UpdateRoundOperationResult{Error: err}, err
		}

		// Extract roundID and messageID
		urm.logger.InfoContext(ctx, "DEBUG: Extracted parts from CustomID",
			attr.String("full_custom_id", customID),
			attr.String("round_id_part", parts[1]),
			attr.String("message_id_part", parts[2]))

		roundUUID, err := uuid.Parse(parts[1])
		if err != nil {
			err := fmt.Errorf("invalid UUID in modal custom_id: %w", err)
			urm.logger.ErrorContext(ctx, err.Error())

			// Respond with error message
			if respErr := urm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Invalid round ID. Please try again.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			}); respErr != nil {
				urm.logger.ErrorContext(ctx, "Failed to respond with error", attr.Error(respErr))
			}

			return UpdateRoundOperationResult{Error: err}, err
		}
		roundID := sharedtypes.RoundID(roundUUID)
		messageID := parts[2]

		// Add debugging here too
		urm.logger.InfoContext(ctx, "DEBUG: Parsed roundID and messageID",
			attr.RoundID("round_id", roundID),
			attr.String("round_id_string", roundID.String()),
			attr.String("message_id", messageID))

		// Extract form data
		title := roundtypes.Title(strings.TrimSpace(data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value))
		description := roundtypes.Description(strings.TrimSpace(data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value))
		startTimeStr := strings.TrimSpace(data.Components[2].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value)
		timezone := roundtypes.Timezone(strings.TrimSpace(data.Components[3].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value))
		location := roundtypes.Location(strings.TrimSpace(data.Components[4].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value))

		urm.logger.InfoContext(ctx, "Extracted form data",
			attr.RoundID("round_id", roundID),
			attr.String("title", string(title)),
			attr.String("description", string(description)),
			attr.String("start_time", startTimeStr),
			attr.String("timezone", string(timezone)),
			attr.String("location", string(location)))

		// Set default timezone to CST if the user didn't provide one
		if timezone == "" {
			timezone = "America/Chicago" // CST
		}

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
			errorMessage := "❌ Round update failed: " + strings.Join(validationErrors, " ")
			err := urm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content:    errorMessage,
					Flags:      discordgo.MessageFlagsEphemeral,
					Components: []discordgo.MessageComponent{},
				},
			})
			if err != nil {
				return UpdateRoundOperationResult{Error: fmt.Errorf("failed to send validation error: %w", err)}, fmt.Errorf("failed to send validation error: %w", err)
			}
			validationErr := fmt.Errorf("validation failed: %s", strings.Join(validationErrors, " "))
			return UpdateRoundOperationResult{Error: validationErr}, validationErr
		}

		// Defer all parsing of relative/absolute time expressions to backend. Only lightweight guard here.
		if len(startTimeStr) > 120 {
			errMsg := "❌ Start Time input too long. Please shorten it."
			_ = urm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Content: errMsg, Flags: discordgo.MessageFlagsEphemeral},
			})
			return UpdateRoundOperationResult{Error: fmt.Errorf("start time too long")}, fmt.Errorf("start time too long")
		}
		// No local parsing; backend will interpret using submitted_at + user_timezone + raw_start_time.

		// Acknowledge receipt of the modal submission after successful validation
		err = urm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content:    "Round update request received.",
				Flags:      discordgo.MessageFlagsEphemeral,
				Components: []discordgo.MessageComponent{},
			},
		})
		if err != nil {
			acknowledgeErr := fmt.Errorf("failed to acknowledge submission: %w", err)
			return UpdateRoundOperationResult{Error: acknowledgeErr}, acknowledgeErr
		}

		// Build Discord payload for internal bus; backend handler will convert to backend payload
		tz := roundtypes.Timezone(timezone)
		payload := discordroundevents.RoundUpdateModalSubmittedPayloadV1{
			GuildID:     sharedtypes.GuildID(i.GuildID),
			RoundID:     roundID,
			UserID:      sharedtypes.DiscordID(userID),
			ChannelID:   i.ChannelID,
			MessageID:   messageID,
			Title:       &title,
			Description: &description,
			Location:    &location,
			StartTime:   &startTimeStr,
			Timezone:    &tz,
		}

		// ✅ Add debugging for payload
		urm.logger.InfoContext(ctx, "DEBUG: Final payload being published",
			attr.Any("payload", payload),
			attr.RoundID("payload_round_id", payload.RoundID),
			attr.String("payload_message_id", messageID))

		correlationID := uuid.New().String()
		if err := urm.interactionStore.Set(ctx, correlationID, i.Interaction); err != nil {
			storeErr := fmt.Errorf("failed to store interaction: %w", err)
			return UpdateRoundOperationResult{Error: storeErr}, storeErr
		}

		msg, err := urm.createEvent(ctx, discordroundevents.RoundUpdateModalSubmittedV1, payload, i)
		if err != nil {
			updateErr := fmt.Errorf("failed to update event: %w", err)
			return UpdateRoundOperationResult{Error: updateErr}, updateErr
		}

		// Set the correlation ID in the message metadata before publishing
		msg.Metadata.Set("correlation_id", correlationID)
		msg.Metadata.Set("user_id", userID)
		msg.Metadata.Set("submitted_at", time.Now().UTC().Format(time.RFC3339))
		msg.Metadata.Set("user_timezone", string(timezone))
		msg.Metadata.Set("raw_start_time", startTimeStr)

		if err := urm.publisher.Publish(discordroundevents.RoundUpdateModalSubmittedV1, msg); err != nil {
			publishErr := fmt.Errorf("failed to publish event: %w", err)
			return UpdateRoundOperationResult{Error: publishErr}, publishErr
		}

		urm.logger.InfoContext(ctx, "Round update request published", attr.UserID(sharedtypes.DiscordID(userID)))
		return UpdateRoundOperationResult{Success: "round update request published"}, nil
	})
}

// HandleUpdateRoundModalCancel handles the cancellation of the update round modal.
func (urm *updateRoundManager) HandleUpdateRoundModalCancel(ctx context.Context, i *discordgo.InteractionCreate) (UpdateRoundOperationResult, error) {
	if i == nil || i.Interaction == nil {
		return UpdateRoundOperationResult{Error: fmt.Errorf("interaction is nil or incomplete")}, fmt.Errorf("interaction is nil or incomplete")
	}

	userID := ""
	if i.Member != nil && i.Member.User != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}

	ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, userID)
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "handle_update_round_modal_cancel")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "button")

	return urm.operationWrapper(ctx, "handle_update_round_modal_cancel", func(ctx context.Context) (UpdateRoundOperationResult, error) {
		// Check for context cancellation first
		if err := ctx.Err(); err != nil {
			return UpdateRoundOperationResult{Error: err}, err
		}
		urm.logger.InfoContext(ctx, "Handling update round modal cancel", attr.String("interaction_id", i.ID))

		// Remove the token from the store
		urm.interactionStore.Delete(ctx, i.Interaction.ID)

		err := urm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content:    "Round update cancelled.",
				Components: []discordgo.MessageComponent{},
				Flags:      discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			urm.logger.ErrorContext(ctx, "Failed to cancel interaction", attr.Error(err), attr.String("interaction_id", i.ID))
			return UpdateRoundOperationResult{Error: err}, fmt.Errorf("failed to send cancellation response: %w", err)
		}

		return UpdateRoundOperationResult{Success: "round update cancelled"}, nil
	})
}

package createround

import (
	"context"
	"fmt"
	"strings"

	discordroundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
)

func (crm *createRoundManager) SendCreateRoundModal(ctx context.Context, i *discordgo.InteractionCreate) (CreateRoundOperationResult, error) {
	var opErr error // Declare an error variable

	if err := ctx.Err(); err != nil { // Check for context cancellation here
		opErr = err
		return CreateRoundOperationResult{Error: opErr}, opErr
	}

	if i == nil || i.Interaction == nil {
		opErr = fmt.Errorf("interaction is nil or incomplete")
		return CreateRoundOperationResult{Error: opErr}, opErr
	}

	userID := ""
	if i.Member != nil && i.Member.User != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}

	if userID == "" {
		opErr = fmt.Errorf("user ID is missing")
		return CreateRoundOperationResult{Error: opErr}, opErr
	}

	ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, userID)
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "send_create_round_modal")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "command")

	result, _ := crm.operationWrapper(ctx, "send_create_round_modal", func(ctx context.Context) (CreateRoundOperationResult, error) {
		if err := ctx.Err(); err != nil {
			return CreateRoundOperationResult{Error: err}, err
		}
		crm.logger.InfoContext(ctx, "Sending create round modal", attr.UserID(sharedtypes.DiscordID(userID)))

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
								Label:       "Start Time (YYYY-MM-DD HH:MM)",
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
								Label:       "Timezone (Optional Default: CST)",
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
								Placeholder: "Enter the location",
								Required:    true,
								MaxLength:   100,
							},
						},
					},
				},
			},
		})
		if err != nil {
			crm.logger.ErrorContext(ctx, "Failed to send create round modal",
				attr.UserID(sharedtypes.DiscordID(userID)),
				attr.Error(err))
			opErr = fmt.Errorf("failed to send create round modal: %w", err)
			return CreateRoundOperationResult{Error: opErr}, opErr
		}
		return CreateRoundOperationResult{Success: "modal sent"}, nil
	})
	return result, opErr
}

// HandleCreateRoundModalSubmit handles the submission of the create round modal.
func (crm *createRoundManager) HandleCreateRoundModalSubmit(ctx context.Context, i *discordgo.InteractionCreate) (CreateRoundOperationResult, error) {
	if i == nil || i.Interaction == nil {
		return CreateRoundOperationResult{Error: fmt.Errorf("interaction is nil or incomplete")}, fmt.Errorf("interaction is nil or incomplete")
	}

	userID := ""
	if i.Member != nil && i.Member.User != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}

	if userID == "" {
		return CreateRoundOperationResult{Error: fmt.Errorf("user ID is missing")}, fmt.Errorf("user ID is missing")
	}

	ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, userID)
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "handle_create_round_modal_submit")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "modal_submit")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.GuildIDKey, i.GuildID)

	if i.Message != nil {
		ctx = discordmetrics.WithValue(ctx, discordmetrics.MessageIDKey, i.Message.ID)
	} else {
		ctx = discordmetrics.WithValue(ctx, discordmetrics.MessageIDKey, "modal_submission")
	}

	return crm.operationWrapper(ctx, "handle_create_round_modal_submit", func(ctx context.Context) (CreateRoundOperationResult, error) {
		// Check for context cancellation first
		if err := ctx.Err(); err != nil {
			return CreateRoundOperationResult{Error: err}, err
		}

		crm.logger.InfoContext(ctx, "Handling create round modal submission", attr.UserID(sharedtypes.DiscordID(userID)))

		// Extract form data
		data := i.ModalSubmitData()
		title := strings.TrimSpace(data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value)
		description := roundtypes.Description(strings.TrimSpace(data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value))
		startTimeStr := strings.TrimSpace(data.Components[2].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value)
		timezone := strings.TrimSpace(data.Components[3].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value)
		location := roundtypes.Location(strings.TrimSpace(data.Components[4].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value))

		crm.logger.InfoContext(ctx, "Extracted form data",
			attr.String("title", title),
			attr.String("description", string(description)),
			attr.String("start_time", startTimeStr),
			attr.String("timezone", timezone),
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
			errorMessage := "❌ Round creation failed: " + strings.Join(validationErrors, " ")
			err := crm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content:    errorMessage,
					Flags:      discordgo.MessageFlagsEphemeral,
					Components: []discordgo.MessageComponent{},
				},
			})
			if err != nil {
				return CreateRoundOperationResult{Error: fmt.Errorf("failed to send validation error: %w", err)}, fmt.Errorf("failed to send validation error: %w", err)
			}
			validationErr := fmt.Errorf("validation failed: %s", strings.Join(validationErrors, " "))
			return CreateRoundOperationResult{Error: validationErr}, validationErr
		}

		// Acknowledge receipt of the modal submission
		err := crm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content:    "Round creation request received",
				Flags:      discordgo.MessageFlagsEphemeral,
				Components: []discordgo.MessageComponent{},
			},
		})
		if err != nil {
			acknowledgeErr := fmt.Errorf("failed to acknowledge submission: %w", err)
			return CreateRoundOperationResult{Error: acknowledgeErr}, acknowledgeErr
		}

		// Lookup the event channel ID from guildconfig
		var eventChannelID string
		if crm.guildConfigResolver != nil {
			guildConfig, err := crm.guildConfigResolver.GetGuildConfigWithContext(ctx, i.GuildID)
			if err == nil && guildConfig != nil && guildConfig.EventChannelID != "" {
				eventChannelID = guildConfig.EventChannelID
			} else {
				crm.logger.WarnContext(ctx, "Failed to resolve event channel ID, falling back to interaction channel", attr.Error(err))
				eventChannelID = i.ChannelID
			}
		} else {
			eventChannelID = i.ChannelID
		}

		// Publish event for backend validation
		payload := discordroundevents.CreateRoundModalPayloadV1{
			UserID:      sharedtypes.DiscordID(userID),
			Title:       roundtypes.Title(title),
			Description: description,
			StartTime:   startTimeStr,
			Location:    location,
			Timezone:    roundtypes.Timezone(timezone),
			ChannelID:   eventChannelID,
			GuildID:     sharedtypes.GuildID(i.GuildID),
		}

		crm.logger.InfoContext(ctx, "Publishing event for Modal validation", attr.Any("payload", payload))

		msg, correlationID, err := crm.createEvent(ctx, discordroundevents.RoundCreateModalSubmittedV1, payload, i)
		if err != nil {
			createErr := fmt.Errorf("failed to create event: %w", err)
			return CreateRoundOperationResult{Error: createErr}, createErr
		}

		if err := crm.interactionStore.Set(ctx, correlationID, i.Interaction); err != nil {
			storeErr := fmt.Errorf("failed to store interaction: %w", err)
			return CreateRoundOperationResult{Error: storeErr}, storeErr
		}

		// Set the correlation ID in the message metadata before publishing
		msg.Metadata.Set("correlation_id", correlationID)
		msg.Metadata.Set("user_id", userID)

		if err := crm.publisher.Publish(discordroundevents.RoundCreateModalSubmittedV1, msg); err != nil {
			publishErr := fmt.Errorf("failed to publish event: %w", err)
			return CreateRoundOperationResult{Error: publishErr}, publishErr
		}

		crm.logger.InfoContext(ctx, "Round creation request published", attr.UserID(sharedtypes.DiscordID(userID)))
		return CreateRoundOperationResult{Success: "round creation request published"}, nil
	})
}

// HandleCreateRoundModalCancel handles the cancellation of the create round modal.
func (crm *createRoundManager) HandleCreateRoundModalCancel(ctx context.Context, i *discordgo.InteractionCreate) (CreateRoundOperationResult, error) {
	if i == nil || i.Interaction == nil {
		return CreateRoundOperationResult{Error: fmt.Errorf("interaction is nil or incomplete")}, fmt.Errorf("interaction is nil or incomplete")
	}

	userID := ""
	if i.Member != nil && i.Member.User != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}

	ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, userID)
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "handle_create_round_modal_cancel")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "button")

	return crm.operationWrapper(ctx, "handle_create_round_modal_cancel", func(ctx context.Context) (CreateRoundOperationResult, error) {
		// Check for context cancellation first
		if err := ctx.Err(); err != nil {
			return CreateRoundOperationResult{Error: err}, err
		}
		crm.logger.InfoContext(ctx, "Handling create round modal cancel", attr.String("interaction_id", i.ID))

		// Remove the token from the store
		crm.interactionStore.Delete(ctx, i.Interaction.ID)

		err := crm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content:    "Round creation cancelled.",
				Components: []discordgo.MessageComponent{},
				Flags:      discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			crm.logger.ErrorContext(ctx, "Failed to cancel interaction", attr.Error(err), attr.String("interaction_id", i.ID))
			return CreateRoundOperationResult{Error: err}, fmt.Errorf("failed to send cancellation response: %w", err)
		}

		return CreateRoundOperationResult{Success: "round creation cancelled"}, nil
	})
}

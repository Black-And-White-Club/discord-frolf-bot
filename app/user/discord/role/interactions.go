package role

import (
	"context"
	"fmt"
	"strings"
	"time"

	discorduserevents "github.com/Black-And-White-Club/discord-frolf-bot/app/events/user"
	messagecreator "github.com/Black-And-White-Club/discord-frolf-bot/app/shared/utils"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

// RespondToRoleRequest presents the role selection buttons.
func (rm *roleManager) RespondToRoleRequest(ctx context.Context, interactionID, interactionToken string, targetUserID sharedtypes.DiscordID) (RoleOperationResult, error) {
	// Enrich context for observability
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "respond_to_role_request")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "response")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, string(targetUserID))

	rm.logger.InfoContext(ctx, "Responding to role request",
		attr.String("interaction_id", interactionID),
		attr.UserID(targetUserID))

	return rm.operationWrapper(ctx, "respond_to_role_request", func(ctx context.Context) (RoleOperationResult, error) {
		if ctx.Err() != nil {
			rm.logger.WarnContext(ctx, "Context is cancelled, aborting responding to role request")
			return RoleOperationResult{
				Success: "", // Explicitly set to empty string for error case
				Error:   ctx.Err(),
			}, nil
		}

		var buttons []discordgo.MessageComponent
		// Iterate over the role mappings from the config
		for role := range rm.config.GetRoleMappings() {
			buttons = append(buttons, discordgo.Button{
				Label:    role,
				Style:    discordgo.PrimaryButton,
				CustomID: fmt.Sprintf("role_button_%s", role),
			})
		}
		buttons = append(buttons, discordgo.Button{
			Label:    "Cancel",
			Style:    discordgo.DangerButton,
			CustomID: "role_button_cancel",
		})

		err := rm.session.InteractionRespond(&discordgo.Interaction{ID: interactionID, Token: interactionToken}, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Please choose a role for <@%s>:", targetUserID),
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{Components: buttons},
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			rm.logger.ErrorContext(ctx, "Failed to respond to role request",
				attr.String("interaction_id", interactionID),
				attr.UserID(targetUserID),
				attr.Error(err))
			return RoleOperationResult{
				Success: "", // Explicitly set to empty string for error case
				Error:   fmt.Errorf("failed to respond to role request: %w", err),
			}, nil
		}

		rm.logger.DebugContext(ctx, "Successfully responded to role request",
			attr.String("interaction_id", interactionID),
			attr.UserID(targetUserID))
		return RoleOperationResult{Success: "role request response sent"}, nil
	})
}

// RespondToRoleButtonPress acknowledges the button press with an update message.
func (rm *roleManager) RespondToRoleButtonPress(ctx context.Context, interactionID, interactionToken string, requesterID sharedtypes.DiscordID, selectedRole string, targetUserID sharedtypes.DiscordID) (RoleOperationResult, error) {
	// Enrich context for observability
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "respond_to_button_press")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "response")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, requesterID)

	rm.logger.InfoContext(ctx, "Responding to role button press",
		attr.String("interaction_id", interactionID),
		attr.UserID(requesterID),
		attr.String("selected_role", selectedRole),
		attr.UserID(targetUserID))

	return rm.operationWrapper(ctx, "respond_to_role_button_press", func(ctx context.Context) (RoleOperationResult, error) {
		if ctx.Err() != nil {
			rm.logger.WarnContext(ctx, "Context is cancelled, aborting responding to role button press")
			return RoleOperationResult{
				Success: "", // Explicitly set to empty string for error case
				Error:   ctx.Err(),
			}, nil
		}

		updateMsg := fmt.Sprintf("<@%s> has requested role '%s' for <@%s>. Request is being processed.",
			requesterID, selectedRole, targetUserID)

		err := rm.session.InteractionRespond(&discordgo.Interaction{ID: interactionID, Token: interactionToken}, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content:    updateMsg,
				Components: []discordgo.MessageComponent{},
				Flags:      discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			rm.logger.ErrorContext(ctx, "Failed to acknowledge role button press",
				attr.String("interaction_id", interactionID),
				attr.UserID(requesterID),
				attr.String("selected_role", selectedRole),
				attr.UserID(targetUserID),
				attr.Error(err))
			return RoleOperationResult{
				Success: "", // Explicitly set to empty string for error case
				Error:   fmt.Errorf("failed to acknowledge role button press: %w", err),
			}, nil
		}

		rm.logger.DebugContext(ctx, "Successfully acknowledged role button press",
			attr.String("interaction_id", interactionID),
			attr.UserID(requesterID),
			attr.String("selected_role", selectedRole),
			attr.UserID(targetUserID))
		return RoleOperationResult{Success: "button press acknowledged"}, nil
	})
}

// HandleRoleRequestCommand handles the /updaterole command.
func (rm *roleManager) HandleRoleRequestCommand(ctx context.Context, i *discordgo.InteractionCreate) (RoleOperationResult, error) {
	// Enrich context for observability
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "role_request")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "command")

	if i == nil || i.Interaction == nil {
		rm.logger.ErrorContext(ctx, "Interaction is nil")
		return RoleOperationResult{}, fmt.Errorf("interaction is nil")
	}

	if i.GuildID != "" {
		ctx = discordmetrics.WithValue(ctx, discordmetrics.GuildIDKey, i.GuildID)
	}

	rm.logger.InfoContext(ctx, "Handling /updaterole command")

	// Get the user from the interaction
	var userID string
	if i.Member != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	} else {
		rm.logger.ErrorContext(ctx, "User  is nil")
		return RoleOperationResult{}, fmt.Errorf("user is nil")
	}

	ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, userID)

	rm.logger.InfoContext(ctx, "Processing role request",
		attr.String("user_id", userID))

	// Use operation wrapper for handling the role request response
	result, err := rm.operationWrapper(ctx, "handle_role_request_command", func(ctx context.Context) (RoleOperationResult, error) {
		if ctx.Err() != nil {
			rm.logger.WarnContext(ctx, "Context is cancelled, aborting role request handling")
			return RoleOperationResult{Error: ctx.Err()}, nil
		}

		result, err := rm.RespondToRoleRequest(ctx, i.Interaction.ID, i.Interaction.Token, sharedtypes.DiscordID(userID))
		if err != nil {
			rm.logger.ErrorContext(ctx, "Failed to respond to role request",
				attr.UserID(sharedtypes.DiscordID(userID)),
				attr.Error(err))
			return RoleOperationResult{Error: err}, nil
		}

		return result, nil
	})
	if err != nil {
		rm.logger.ErrorContext(ctx, "Operation wrapper failed", attr.Error(err))
		return RoleOperationResult{}, err
	}

	if result.Error != nil {
		rm.logger.ErrorContext(ctx, "Failed to handle role request", attr.Error(result.Error))
		return RoleOperationResult{Error: result.Error}, nil
	}

	return result, nil
}

// HandleRoleButtonPress handles role button presses.
func (rm *roleManager) HandleRoleButtonPress(ctx context.Context, i *discordgo.InteractionCreate) (RoleOperationResult, error) {
	// Enrich context for observability
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "role_button")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "component")

	// Check for nil dependencies
	if rm.publisher == nil {
		rm.logger.ErrorContext(ctx, "Publisher is nil - cannot proceed")
		return RoleOperationResult{}, fmt.Errorf("publisher is nil")
	}
	if rm.interactionStore == nil {
		rm.logger.ErrorContext(ctx, "InteractionStore is nil - cannot proceed")
		return RoleOperationResult{}, fmt.Errorf("interaction store is nil")
	}

	// Check for nil interaction
	if i == nil || i.Interaction == nil {
		rm.logger.ErrorContext(ctx, "Received nil interaction - exiting function")
		return RoleOperationResult{}, fmt.Errorf("received nil interaction")
	}

	if i.GuildID != "" {
		ctx = discordmetrics.WithValue(ctx, discordmetrics.GuildIDKey, i.GuildID)
	}

	rm.logger.InfoContext(ctx, "Handling role button press")

	result, err := rm.operationWrapper(ctx, "handle_role_button", func(ctx context.Context) (RoleOperationResult, error) {
		if ctx.Err() != nil {
			rm.logger.WarnContext(ctx, "Context is cancelled, aborting handling of role button press")
			return RoleOperationResult{Error: ctx.Err()}, nil
		}

		// Type assertion for MessageComponentInteractionData
		var data *discordgo.MessageComponentInteractionData
		switch d := i.Interaction.Data.(type) {
		case *discordgo.MessageComponentInteractionData:
			data = d
		default:
			err := fmt.Errorf("unexpected interaction data type: %T", i.Interaction.Data)
			rm.logger.ErrorContext(ctx, err.Error())
			return RoleOperationResult{Error: err}, nil
		}

		// Extract role from CustomID
		roleStr := strings.TrimPrefix(data.CustomID, "role_button_")
		rm.logger.DebugContext(ctx, "Extracted role string", attr.String("role_str", roleStr))

		// Validate role string
		roleEnum := sharedtypes.UserRoleEnum(roleStr)
		if roleEnum == "" {
			err := fmt.Errorf("invalid role string: %s", roleStr)
			rm.logger.ErrorContext(ctx, err.Error())
			return RoleOperationResult{Error: err}, nil
		}

		// Ensure mentions exist before accessing index 0
		if i.Message == nil || len(i.Message.Mentions) == 0 || i.Message.Mentions[0] == nil {
			err := fmt.Errorf("no valid mentions found in the interaction message")
			rm.logger.ErrorContext(ctx, err.Error())
			return RoleOperationResult{Error: err}, nil
		}

		// Ensure `i.Member` is not nil before accessing `User .ID`
		if i.Member == nil || i.Member.User == nil {
			err := fmt.Errorf("interaction Member or User is nil")
			rm.logger.ErrorContext(ctx, err.Error())
			return RoleOperationResult{Error: err}, nil
		}

		requesterID := sharedtypes.DiscordID(i.Member.User.ID)
		targetUserID := sharedtypes.DiscordID(i.Message.Mentions[0].ID)

		ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, requesterID)

		// Construct event payload
		payload := discorduserevents.RoleUpdateButtonPressPayload{
			RequesterID:         requesterID,
			TargetUserID:        targetUserID,
			SelectedRole:        roleEnum,
			InteractionID:       i.Interaction.ID,
			InteractionToken:    i.Interaction.Token,
			InteractionCustomID: data.CustomID,
			GuildID:             i.GuildID,
		}

		rm.logger.DebugContext(ctx, "Constructed event payload", attr.Any("payload", payload))

		// Generate and store correlation ID for tracking
		correlationID := uuid.New().String()
		rm.logger.InfoContext(ctx, "Storing interaction reference in cache", attr.String("correlation_id", correlationID))

		if err := rm.interactionStore.Set(correlationID, i.Interaction, 10*time.Minute); err != nil {
			rm.logger.ErrorContext(ctx, "Failed to store interaction reference in cache", attr.Error(err))
			return RoleOperationResult{Error: err}, nil
		}

		// Create Watermill event
		msg, err := messagecreator.BuildWatermillMessageFromInteraction(
			discorduserevents.RoleUpdateButtonPress,
			payload,
			i,
			rm.helper,
			rm.config,
		)
		if err != nil {
			rm.logger.ErrorContext(ctx, "Failed to create event", attr.Error(err))
			return RoleOperationResult{Error: err}, nil
		}

		msg.Metadata.Set("correlation_id", correlationID)

		// Publish the event to JetStream
		rm.logger.DebugContext(ctx, "Publishing event to JetStream", attr.String("event", discorduserevents.RoleUpdateButtonPress))
		if err := rm.publisher.Publish(discorduserevents.RoleUpdateButtonPress, msg); err != nil {
			rm.logger.ErrorContext(ctx, "Failed to publish event", attr.Error(err))
			return RoleOperationResult{Error: err}, nil
		}

		rm.logger.InfoContext(ctx, "Event published successfully", attr.String("event", discorduserevents.RoleUpdateButtonPress))
		return RoleOperationResult{Success: "role button processed"}, nil
	})
	if err != nil {
		rm.logger.ErrorContext(ctx, "Operation wrapper failed", attr.Error(err))
		return RoleOperationResult{}, err
	}

	if result.Error != nil {
		rm.logger.ErrorContext(ctx, "Failed to handle role button press", attr.Error(result.Error))

		// Notify the user of the failure
		_, err := rm.session.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: ptr("Something went wrong while processing your request. Please try again later."),
		})
		if err != nil {
			rm.logger.ErrorContext(ctx, "Failed to send error response to user", attr.Error(err))
		}
		return RoleOperationResult{Error: result.Error}, nil
	}

	return result, nil
}

// HandleRoleCancelButton handles the role cancel button.
func (rm *roleManager) HandleRoleCancelButton(ctx context.Context, i *discordgo.InteractionCreate) (RoleOperationResult, error) {
	// Enrich context for observability
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "role_cancel")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "component")

	if ctx.Err() != nil {
		rm.logger.WarnContext(ctx, "Context is cancelled, aborting handling of role cancel button")
		return RoleOperationResult{Error: ctx.Err()}, nil
	}

	if i == nil || i.Interaction == nil {
		rm.logger.WarnContext(ctx, "Received nil interaction")
		return RoleOperationResult{Error: fmt.Errorf("received nil interaction")}, nil
	}

	if i.GuildID != "" {
		ctx = discordmetrics.WithValue(ctx, discordmetrics.GuildIDKey, i.GuildID)
	}

	rm.logger.InfoContext(ctx, "Handling role cancel button")

	result, err := rm.operationWrapper(ctx, "handle_role_cancel", func(ctx context.Context) (RoleOperationResult, error) {
		if ctx.Err() != nil {
			rm.logger.WarnContext(ctx, "Context is cancelled, aborting handling of role cancel button")
			return RoleOperationResult{Error: ctx.Err()}, nil
		}

		// Delete the interaction from the store
		rm.interactionStore.Delete(i.Interaction.ID)

		err := rm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content:    "Role request cancelled.",
				Components: []discordgo.MessageComponent{}, // Remove buttons
				Flags:      discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			rm.logger.ErrorContext(ctx, "Failed to cancel interaction", attr.Error(err))
			return RoleOperationResult{Error: err}, nil
		}

		return RoleOperationResult{Success: "role request cancelled"}, nil
	})
	if err != nil {
		rm.logger.ErrorContext(ctx, "Operation wrapper failed", attr.Error(err))
		return RoleOperationResult{}, err
	}

	if result.Error != nil {
		rm.logger.ErrorContext(ctx, "Failed to handle role cancel button", attr.Error(result.Error))
		return RoleOperationResult{Error: result.Error}, nil
	}

	return result, nil
}

// Helper function for string pointers
func ptr(s string) *string {
	return &s
}

package role

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"time"

	discorduserevents "github.com/Black-And-White-Club/discord-frolf-bot/app/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

// RespondToRoleRequest presents the role selection buttons.
func (rm *roleManager) RespondToRoleRequest(ctx context.Context, interactionID, interactionToken, targetUserID string) error {
	rm.logger.Info(ctx, "Responding to role request",
		attr.String("interaction_id", interactionID),
		attr.String("target_user_id", targetUserID),
	)
	var buttons []discordgo.MessageComponent
	// Iterate over the role mappings from the config
	for role := range rm.config.Discord.RoleMappings {
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
		rm.logger.Error(ctx, "Failed to respond to role request",
			attr.String("interaction_id", interactionID),
			attr.String("target_user_id", targetUserID),
			attr.Error(err),
		)
		return fmt.Errorf("failed to respond to role request: %w", err)
	}
	rm.logger.Debug(ctx, "Successfully responded to role request",
		attr.String("interaction_id", interactionID),
		attr.String("target_user_id", targetUserID),
	)
	return nil
}

// RespondToRoleButtonPress acknowledges the button press with an update message.
func (rm *roleManager) RespondToRoleButtonPress(ctx context.Context, interactionID, interactionToken, requesterID, selectedRole, targetUserID string) error {
	rm.logger.Info(ctx, "Responding to role button press",
		attr.String("interaction_id", interactionID),
		attr.String("requester_id", requesterID),
		attr.String("selected_role", selectedRole),
		attr.String("target_user_id", targetUserID),
	)
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
		rm.logger.Error(ctx, "Failed to acknowledge role button press",
			attr.String("interaction_id", interactionID),
			attr.String("requester_id", requesterID),
			attr.String("selected_role", selectedRole),
			attr.String("target_user_id", targetUserID),
			attr.Error(err),
		)
		return fmt.Errorf("failed to acknowledge role button press: %w", err)
	}
	rm.logger.Debug(ctx, "Successfully acknowledged role button press",
		attr.String("interaction_id", interactionID),
		attr.String("requester_id", requesterID),
		attr.String("selected_role", selectedRole),
		attr.String("target_user_id", targetUserID),
	)
	return nil
}

// HandleRoleRequestCommand handles the /updaterole command.
func (rm *roleManager) HandleRoleRequestCommand(ctx context.Context, i *discordgo.InteractionCreate) {
	if i == nil || i.Interaction == nil {
		slog.Error("❌ Interaction is nil")
		return
	}

	slog.Info("🔄 Handling /updaterole command", attr.String("interaction_id", i.Interaction.ID))

	// Get the user from the interaction
	user := i.Interaction.User
	if user == nil {
		slog.Error("❌ User is nil")
		return
	}

	slog.Info("✅ User: %s", attr.String("user", fmt.Sprintf("%+v", user)))

	// Respond to the role request with buttons for role selection
	err := rm.RespondToRoleRequest(ctx, i.Interaction.ID, i.Interaction.Token, user.ID)
	if err != nil {
		slog.Error("Failed to respond to role request", attr.UserID(user.ID), attr.Error(err))
	}
}

// HandleRoleButtonPress handles role button presses.
// HandleRoleButtonPress handles role button presses.
func (rm *roleManager) HandleRoleButtonPress(ctx context.Context, i *discordgo.InteractionCreate) {
	// Check for nil dependencies
	if rm.publisher == nil {
		slog.Error("Publisher is nil - cannot proceed")
		return
	}
	if rm.interactionStore == nil {
		slog.Error("InteractionStore is nil - cannot proceed")
		return
	}

	// Check for nil interaction
	if i == nil || i.Interaction == nil {
		slog.Error("Received nil interaction - exiting function")
		return
	}

	slog.Info("Handling role button press", attr.String("interaction_id", i.Interaction.ID))

	// Type assertion for MessageComponentInteractionData
	var data *discordgo.MessageComponentInteractionData
	switch d := i.Interaction.Data.(type) {
	case *discordgo.MessageComponentInteractionData:
		data = d
	default:
		slog.Error("Unexpected interaction data type", attr.Any("data_type", reflect.TypeOf(d)))
		return
	}

	// Extract role from CustomID
	roleStr := strings.TrimPrefix(data.CustomID, "role_button_")
	slog.Debug("Extracted role string", attr.String("role_str", roleStr))

	// Validate role string
	roleEnum := usertypes.UserRoleEnum(roleStr)
	if roleEnum == "" {
		slog.Error("Invalid role string - exiting function", attr.String("role_str", roleStr))
		return
	}

	// Ensure mentions exist before accessing index 0
	if i.Message == nil || len(i.Message.Mentions) == 0 || i.Message.Mentions[0] == nil {
		slog.Error("No valid mentions found in the interaction message - exiting function")
		return
	}

	// Ensure `i.Member` is not nil before accessing `User .ID`
	if i.Member == nil || i.Member.User == nil {
		slog.Error("Interaction Member or User is nil - exiting function")
		return
	}

	// Construct event payload
	payload := discorduserevents.RoleUpdateButtonPressPayload{
		RequesterID:         i.Member.User.ID,
		TargetUserID:        i.Message.Mentions[0].ID,
		SelectedRole:        roleEnum,
		InteractionID:       i.Interaction.ID,
		InteractionToken:    i.Interaction.Token,
		InteractionCustomID: data.CustomID,
		GuildID:             i.GuildID,
	}
	slog.Debug("Constructed event payload", attr.Any("payload", payload))

	// Generate and store correlation ID for tracking
	correlationID := uuid.New().String()
	slog.Info("Storing interaction reference in cache", attr.String("correlation_id", correlationID))
	if err := rm.interactionStore.Set(correlationID, i.Interaction, 10*time.Minute); err != nil {
		slog.Error("Failed to store interaction reference in cache", attr.Error(err))
		return // Early return if storing fails
	}

	// Create Watermill event
	msg, err := rm.createEvent(ctx, discorduserevents.RoleUpdateButtonPress, payload, i)
	if err != nil {
		slog.Error("Failed to create event", attr.Error(err))
		return
	}

	msg.Metadata.Set("correlation_id", correlationID)

	// Publish the event to JetStream
	slog.Debug("Publishing event to JetStream", attr.String("event", discorduserevents.RoleUpdateButtonPress))
	if err := rm.publisher.Publish(discorduserevents.RoleUpdateButtonPress, msg); err != nil {
		slog.Error("Failed to publish event", attr.Error(err), attr.String("interaction_id", i.Interaction.ID))

		// Notify the user of the failure
		_, err := rm.session.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: ptr("Something went wrong while processing your request. Please try again later."),
		})
		if err != nil {
			slog.Error("Failed to send error response to user", attr.Error(err))
		}
		return
	}

	slog.Info("Event published successfully", attr.String("event", discorduserevents.RoleUpdateButtonPress))
}

// Helper function for string pointers
func ptr(s string) *string {
	return &s
}

// HandleRoleCancelButton handles the role cancel button.
func (rm *roleManager) HandleRoleCancelButton(ctx context.Context, i *discordgo.InteractionCreate) {
	if ctx.Err() != nil {
		slog.Warn("Context is cancelled, aborting handling of role cancel button")
		return
	}

	if i == nil || i.Interaction == nil {
		slog.Warn("Received nil interaction")
		return
	}

	slog.Info("Handling role cancel button", attr.String("interaction_id", i.Interaction.ID))

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
		slog.Error("Failed to cancel interaction", attr.Error(err), attr.String("interaction_id", i.Interaction.ID))
	}
}

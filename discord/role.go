package discord

import (
	"context"
	"fmt"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr" // Correct path
	"github.com/bwmarrin/discordgo"
)

// RespondToRoleRequest presents the role selection buttons.
func (d *discordOperations) RespondToRoleRequest(ctx context.Context, interactionID, interactionToken, targetUserID string) error {
	d.logger.Info(ctx, "Responding to role request",
		attr.String("interaction_id", interactionID),
		attr.String("target_user_id", targetUserID),
	)

	var buttons []discordgo.MessageComponent
	// Iterate over the role mappings from the config
	for role, _ := range d.config.Discord.RoleMappings {
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

	err := d.session.InteractionRespond(&discordgo.Interaction{ID: interactionID, Token: interactionToken}, &discordgo.InteractionResponse{
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
		d.logger.Error(ctx, "Failed to respond to role request",
			attr.String("interaction_id", interactionID),
			attr.String("target_user_id", targetUserID),
			attr.Error(err),
		)
		return fmt.Errorf("failed to respond to role request: %w", err)
	}

	d.logger.Debug(ctx, "Successfully responded to role request",
		attr.String("interaction_id", interactionID),
		attr.String("target_user_id", targetUserID),
	)
	return nil
}

// RespondToRoleButtonPress acknowledges the button press with an update message.
func (d *discordOperations) RespondToRoleButtonPress(ctx context.Context, interactionID, interactionToken, requesterID, selectedRole, targetUserID string) error {
	d.logger.Info(ctx, "Responding to role button press",
		attr.String("interaction_id", interactionID),
		attr.String("requester_id", requesterID),
		attr.String("selected_role", selectedRole),
		attr.String("target_user_id", targetUserID),
	)

	updateMsg := fmt.Sprintf("<@%s> has requested role '%s' for <@%s>. Request is being processed.",
		requesterID, selectedRole, targetUserID)

	err := d.session.InteractionRespond(&discordgo.Interaction{ID: interactionID, Token: interactionToken}, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    updateMsg,
			Components: []discordgo.MessageComponent{},
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		d.logger.Error(ctx, "Failed to acknowledge role button press",
			attr.String("interaction_id", interactionID),
			attr.String("requester_id", requesterID),
			attr.String("selected_role", selectedRole),
			attr.String("target_user_id", targetUserID),
			attr.Error(err),
		)
		return fmt.Errorf("failed to acknowledge role button press: %w", err)
	}

	d.logger.Debug(ctx, "Successfully acknowledged role button press",
		attr.String("interaction_id", interactionID),
		attr.String("requester_id", requesterID),
		attr.String("selected_role", selectedRole),
		attr.String("target_user_id", targetUserID),
	)
	return nil
}

// EditRoleUpdateResponse edits the original interaction response with the result of the role update.
func (d *discordOperations) EditRoleUpdateResponse(ctx context.Context, interactionToken, content string) error {
	d.logger.Info(ctx, "Editing role update response",
		attr.String("interaction_token", interactionToken),
	)

	_, err := d.session.InteractionResponseEdit(
		&discordgo.Interaction{Token: interactionToken}, // Wrap token
		&discordgo.WebhookEdit{
			Content: &content,
		},
	)
	if err != nil {
		d.logger.Error(ctx, "Failed to edit role update response",
			attr.String("interaction_token", interactionToken),
			attr.Error(err),
		)
		return fmt.Errorf("failed to edit role update response: %w", err)
	}

	d.logger.Debug(ctx, "Successfully edited role update response",
		attr.String("interaction_token", interactionToken),
	)
	return nil
}

func (d *discordOperations) AddRoleToUser(ctx context.Context, guildID, userID, roleID string) error {
	d.logger.Info(ctx, "Adding role to user",
		attr.String("guild_id", guildID),
		attr.String("user_id", userID),
		attr.String("role_id", roleID),
	)
	err := d.session.GuildMemberRoleAdd(guildID, userID, roleID)
	if err != nil {
		d.logger.Error(ctx, "Failed to add role to user",
			attr.String("guild_id", guildID),
			attr.String("user_id", userID),
			attr.String("role_id", roleID),
			attr.Error(err),
		)
		return fmt.Errorf("failed to add role %s to user %s in guild %s: %w", roleID, userID, guildID, err)
	}
	d.logger.Debug(ctx, "Successfully added role to user",
		attr.String("guild_id", guildID),
		attr.String("user_id", userID),
		attr.String("role_id", roleID),
	)
	return nil
}

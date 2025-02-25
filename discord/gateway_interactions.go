// discord/gateway_interactions.go
package discord

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	discorduserevents "github.com/Black-And-White-Club/discord-frolf-bot/events/user"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	"github.com/bwmarrin/discordgo"
)

// interactionCreate handles the InteractionCreate event.
func (h *gatewayEventHandler) interactionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()

	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		if i.ApplicationCommandData().Name == "rolerequest" {
			h.handleRoleRequestCommand(ctx, i)
		}
	// ... other command handlers ...

	case discordgo.InteractionMessageComponent:
		if strings.HasPrefix(i.MessageComponentData().CustomID, "role_button_") {
			if i.MessageComponentData().CustomID == "role_button_cancel" {
				h.handleRoleCancelButton(ctx, i)
			} else {
				h.handleRoleButtonPress(ctx, i)
			}
		} else if i.MessageComponentData().CustomID == "signup_button" {
			h.handleSignupButtonPress(ctx, i)
		}
		// ... other component handlers ...
	case discordgo.InteractionModalSubmit:
		if i.ModalSubmitData().CustomID == "signup_modal" {
			h.handleSignupModalSubmit(ctx, i)
		}
	}
}

// handleRoleRequestCommand handles the /rolerequest command.
func (h *gatewayEventHandler) handleRoleRequestCommand(ctx context.Context, i *discordgo.InteractionCreate) {
	h.logger.Info(ctx, "Handling /rolerequest command", attr.String("interaction_id", i.ID))

	user, err := h.session.User(i.ApplicationCommandData().Options[0].UserValue(nil).ID)
	if err != nil {
		h.logger.Error(ctx, "Failed to get user", attr.Error(err))
		return
	}

	payload := discorduserevents.RoleUpdateCommandPayload{
		TargetUserID: user.ID,
		GuildID:      i.GuildID,
	}
	msg, err := h.createEvent(ctx, discorduserevents.RoleUpdateCommand, payload)
	if err != nil {
		h.logger.Error(ctx, "Failed to create event", attr.Error(err))
		return
	}

	msg.Metadata.Set("interaction_id", i.Interaction.ID)
	msg.Metadata.Set("interaction_token", i.Interaction.Token)
	msg.Metadata.Set("guild_id", i.GuildID)

	if err := h.publisher.Publish(discorduserevents.RoleUpdateCommand, msg); err != nil {
		h.logger.Error(ctx, "Failed to publish event", attr.Error(err), attr.String("interaction_id", i.ID))
		return
	}
}

// handleRoleButtonPress handles role button presses.
func (h *gatewayEventHandler) handleRoleButtonPress(ctx context.Context, i *discordgo.InteractionCreate) {
	h.logger.Info(ctx, "Handling role button press", attr.String("interaction_id", i.ID))

	roleStr := strings.TrimPrefix(i.MessageComponentData().CustomID, "role_button_")
	payload := discorduserevents.RoleUpdateButtonPressPayload{
		RequesterID:         i.Member.User.ID,
		TargetUserID:        i.Message.Mentions[0].ID,
		SelectedRole:        usertypes.UserRoleEnum(roleStr),
		InteractionID:       i.Interaction.ID,
		InteractionToken:    i.Interaction.Token,
		InteractionCustomID: i.MessageComponentData().CustomID,
		GuildID:             i.GuildID,
	}

	msg, err := h.createEvent(ctx, discorduserevents.RoleUpdateButtonPress, payload)
	if err != nil {
		h.logger.Error(ctx, "Failed to create event", attr.Error(err))
		return
	}
	msg.Metadata.Set("interaction_id", i.Interaction.ID)
	msg.Metadata.Set("interaction_token", i.Interaction.Token)
	msg.Metadata.Set("guild_id", i.GuildID)

	if err := h.publisher.Publish(discorduserevents.RoleUpdateButtonPress, msg); err != nil {
		h.logger.Error(ctx, "Failed to publish event", attr.Error(err), attr.String("interaction_id", i.ID))
		return
	}
}

func (h *gatewayEventHandler) handleRoleCancelButton(ctx context.Context, i *discordgo.InteractionCreate) {
	h.logger.Info(ctx, "Handling role cancel button", attr.String("interaction_id", i.ID))
	err := h.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "Role request cancelled.",
			Components: []discordgo.MessageComponent{}, // Remove buttons
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		h.logger.Error(ctx, "Failed to cancel interaction", attr.Error(err), attr.String("interaction_id", i.ID))
	}
}

// messageReactionAdd handles MessageReactionAdd events.
func (h *gatewayEventHandler) messageReactionAdd(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	// Configuration for the signup channel and message ID.
	signupChannelID := h.config.Discord.SignupChannelID
	signupMessageID := h.config.Discord.SignupMessageID

	if r.ChannelID != signupChannelID || r.MessageID != signupMessageID {
		return // Ignore reactions on other messages.
	}
	botUser, err := h.session.GetBotUser()
	if err != nil {
		h.logger.Error(context.Background(), "Failed to get bot user", attr.Error(err))
		return // Critical error: Can't proceed without bot user ID.
	}
	//Ignore reactions from the bot
	if r.UserID == botUser.ID {
		return
	}

	h.handleSignupReactionAdd(context.Background(), r)
}

// handleSignupReactionAdd sends the signup modal.
func (h *gatewayEventHandler) handleSignupReactionAdd(ctx context.Context, r *discordgo.MessageReactionAdd) {
	h.logger.Info(ctx, "Handling signup reaction", attr.UserID(r.UserID))

	// Attempt to create a DM channel.
	_, err := h.session.UserChannelCreate(r.UserID)
	if err != nil {
		h.logger.Error(ctx, "Failed to create DM channel for signup", attr.UserID(r.UserID), attr.Error(err))
		return // Exit; we can't send the modal.  Error is logged.
	}

	// Check if r.Member is nil
	if r.Member == nil {
		h.logger.Error(ctx, "r.Member is nil in handleSignupReactionAdd", attr.UserID(r.UserID))
		return // Exit; we can't get the user information.
	}

	//We know we have the channel
	//Create the interaction
	i := &discordgo.Interaction{
		Type:      discordgo.InteractionMessageComponent,
		Token:     "signup_token_" + r.UserID, // Generate a unique token
		Member:    r.Member,
		User:      r.Member.User,
		GuildID:   r.GuildID,
		ID:        "signup_id_" + r.UserID, // Generate a unique interaction id
		ChannelID: r.ChannelID,
	}
	err = h.discord.SendSignupModal(ctx, i)
	if err != nil {
		h.logger.Error(ctx, "Failed to send signup modal", attr.UserID(r.UserID), attr.Error(err))
		// Log the error.  Consider additional error handling (retry, etc.)
	}
}

// New handler for the button press
func (h *gatewayEventHandler) handleSignupButtonPress(ctx context.Context, i *discordgo.InteractionCreate) {
	h.logger.Info(ctx, "Sending signup modal", attr.UserID(i.Member.User.ID))
	err := h.discord.SendSignupModal(ctx, i.Interaction)
	if err != nil {
		h.logger.Error(ctx, "Failed to send signup modal", attr.UserID(i.Member.User.ID), attr.Error(err))
		// Handle the error - e.g., send an ephemeral message in the signup channel.
	}
}

func (h *gatewayEventHandler) handleSignupModalSubmit(ctx context.Context, i *discordgo.InteractionCreate) {
	h.logger.Info(ctx, "Handling signup modal submission", attr.UserID(i.Member.User.ID))

	data := i.ModalSubmitData()

	// Use the existing extractTagNumber helper
	tagNumberPtr, err := h.extractTagNumber(&data)
	if err != nil {
		h.logger.Warn(ctx, "Invalid tag number", attr.UserID(i.Member.User.ID), attr.Error(err))
		// Respond with an error message
		h.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("<@%s> Invalid tag number format.", i.Member.User.ID),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	// Create the event payload.
	payload := userevents.UserSignupRequestPayload{
		DiscordID: usertypes.DiscordID(i.Member.User.ID),
		TagNumber: tagNumberPtr,
	}

	// Create the Watermill message.
	msg, err := h.createEvent(ctx, userevents.UserSignupRequest, payload)
	if err != nil {
		h.logger.Error(ctx, "Failed to create event", attr.Error(err))
		// Respond with an error message
		h.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("<@%s> An internal error has occurred.", i.Member.User.ID),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	msg.Metadata.Set("interaction_id", i.Interaction.ID)
	msg.Metadata.Set("interaction_token", i.Interaction.Token)
	msg.Metadata.Set("guild_id", i.GuildID)

	// Publish the event.
	if err := h.publisher.Publish(userevents.UserSignupRequest, msg); err != nil {
		h.logger.Error(ctx, "Failed to publish event", attr.Error(err))
		// Respond with an error message
		h.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("<@%s> An internal error has occurred.", i.Member.User.ID),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	// Acknowledge the interaction (IMMEDIATELY - within 3 seconds).
	err = h.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		h.logger.Error(ctx, "failed to acknowledge signup modal interaction", attr.Error(err), attr.UserID(i.Member.User.ID))
	}
}

// extractTagNumber extracts the tag number from the modal submission data.
func (h *gatewayEventHandler) extractTagNumber(data *discordgo.ModalSubmitInteractionData) (*int, error) {
	for _, comp := range data.Components {
		row, ok := comp.(*discordgo.ActionsRow)
		if !ok {
			continue // Skip if not an ActionsRow
		}
		for _, innerComp := range row.Components { //Loop through
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

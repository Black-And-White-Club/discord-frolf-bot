// discord/gateway_interactions.go
package discord

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	discorduserevents "github.com/Black-And-White-Club/discord-frolf-bot/events/user"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	"github.com/bwmarrin/discordgo"
)

// interactionCreate handles the InteractionCreate event.
// Update the interactionCreate function
func (h *gatewayEventHandler) InteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()

	// Log the interaction type for debugging
	slog.Info("InteractionCreate handler triggered!", attr.Int("interaction_type", int(i.Type)))

	switch i.Type {

	// Slash Command Handling
	case discordgo.InteractionApplicationCommand:
		commandName := i.ApplicationCommandData().Name
		slog.Info("Handling ApplicationCommand", attr.String("command_name", commandName))

		switch commandName {
		case "updaterole":
			h.HandleRoleRequestCommand(ctx, i)
		case "createround":
			h.HandleCreateRoundCommand(ctx, i)
		}

	// Button / Component Handling
	case discordgo.InteractionMessageComponent:
		customID := i.MessageComponentData().CustomID
		slog.Info("Handling MessageComponent interaction", attr.String("custom_id", customID))

		switch {
		case strings.HasPrefix(customID, "role_button_"):
			if customID == "role_button_cancel" {
				h.HandleRoleCancelButton(ctx, i)
			} else {
				h.HandleRoleButtonPress(ctx, i)
			}
		case strings.HasPrefix(customID, "signup_button|"):
			slog.Info("✅ Calling HandleSignupButtonPress...")
			h.HandleSignupButtonPress(ctx, i)
		case strings.HasPrefix(customID, "retry_create_round"):
			slog.Info("🔄 Retrying Create Round...")
			h.HandleRetryCreateRound(ctx, i)
		}

	// Modal Submission Handling
	case discordgo.InteractionModalSubmit:
		customID := i.ModalSubmitData().CustomID
		slog.Info("Handling ModalSubmit", attr.String("custom_id", customID))

		switch customID {
		case "signup_modal":
			h.HandleSignupModalSubmit(ctx, i)
		case "create_round_modal":
			h.HandleCreateRoundModalSubmit(ctx, i)
		}
	}
}

// handleRoleRequestCommand handles the /rolerequest command.
func (h *gatewayEventHandler) HandleRoleRequestCommand(ctx context.Context, i *discordgo.InteractionCreate) {
	slog.Info("Handling /updaterole command", attr.String("interaction_id", i.ID))

	user, err := h.session.User(i.ApplicationCommandData().Options[0].UserValue(nil).ID)
	if err != nil {
		slog.Error("Failed to get user", attr.Error(err))
		return
	}

	payload := discorduserevents.RoleUpdateCommandPayload{
		TargetUserID: user.ID,
		GuildID:      i.GuildID,
	}
	msg, err := h.createEvent(ctx, discorduserevents.RoleUpdateCommand, payload, i)
	if err != nil {
		slog.Error("Failed to create event", attr.Error(err))
		return
	}

	msg.Metadata.Set("interaction_id", i.Interaction.ID)
	msg.Metadata.Set("interaction_token", i.Interaction.Token)
	msg.Metadata.Set("guild_id", i.GuildID)

	if err := h.publisher.Publish(discorduserevents.RoleUpdateCommand, msg); err != nil {
		slog.Error("Failed to publish event", attr.Error(err), attr.String("interaction_id", i.ID))
		return
	}
}

// handleRoleButtonPress handles role button presses.
func (h *gatewayEventHandler) HandleRoleButtonPress(ctx context.Context, i *discordgo.InteractionCreate) {
	slog.Info("Handling role button press", attr.String("interaction_id", i.ID))

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

	msg, err := h.createEvent(ctx, discorduserevents.RoleUpdateButtonPress, payload, i)
	if err != nil {
		slog.Error("Failed to create event", attr.Error(err))
		return
	}
	msg.Metadata.Set("interaction_id", i.Interaction.ID)
	msg.Metadata.Set("interaction_token", i.Interaction.Token)
	msg.Metadata.Set("guild_id", i.GuildID)

	if err := h.publisher.Publish(discorduserevents.RoleUpdateButtonPress, msg); err != nil {
		slog.Error("Failed to publish event", attr.Error(err), attr.String("interaction_id", i.ID))
		return
	}
}

func (h *gatewayEventHandler) HandleRoleCancelButton(ctx context.Context, i *discordgo.InteractionCreate) {
	slog.Info("Handling role cancel button", attr.String("interaction_id", i.ID))
	err := h.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "Role request cancelled.",
			Components: []discordgo.MessageComponent{}, // Remove buttons
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		slog.Error("Failed to cancel interaction", attr.Error(err), attr.String("interaction_id", i.ID))
	}
}

// messageReactionAdd handles MessageReactionAdd events.
func (h *gatewayEventHandler) MessageReactionAdd(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	slog.Info("GatewayHandler.MessageReactionAdd called", attr.UserID(r.UserID))

	signupChannelID := h.config.Discord.SignupChannelID
	signupMessageID := h.config.Discord.SignupMessageID
	signupEmoji := h.config.Discord.SignupEmoji

	if r.ChannelID != signupChannelID || r.MessageID != signupMessageID || r.Emoji.Name != signupEmoji {
		slog.Info("Reaction mismatch",
			attr.UserID(r.UserID),
			attr.String("channel_id", r.ChannelID),
			attr.String("message_id", r.MessageID),
			attr.Any("emoji", r.Emoji.Name))
		return
	}

	slog.Info("Valid reaction detected, processing signup.")

	botUser, err := h.session.GetBotUser()
	if err != nil {
		slog.Error("Failed to get bot user", attr.Error(err))
		return
	}

	if r.UserID == botUser.ID {
		slog.Info("Ignoring bot's own reaction.")
		return
	}

	slog.Info("Publishing signup reaction event...")
	h.HandleSignupReactionAdd(context.Background(), r)
}

// handleSignupReactionAdd sends the signup modal.
func (h *gatewayEventHandler) HandleSignupReactionAdd(ctx context.Context, r *discordgo.MessageReactionAdd) {
	slog.Info("Handling signup reaction", attr.UserID(r.UserID))

	// ✅ Verify guild ID
	if r.GuildID != h.config.Discord.GuildID {
		slog.Warn("Reaction from wrong guild", attr.UserID(r.UserID), attr.String("guildID", r.GuildID))
		return
	}

	slog.Info("Attempting to create DM channel...")

	// ✅ Attempt to create a DM channel
	dmChannel, err := h.session.UserChannelCreate(r.UserID)
	if err != nil {
		slog.Error("Failed to create DM channel for signup", attr.UserID(r.UserID), attr.Error(err))
		return
	}

	slog.Info("DM channel created", attr.String("dm_channel_id", dmChannel.ID))

	// ✅ Store a placeholder identifier + user ID in `CustomID`
	metadataStr := fmt.Sprintf("signup_button|%s", r.UserID)

	// ✅ Send the ephemeral message with the "Signup" button
	_, err = h.session.ChannelMessageSendComplex(dmChannel.ID, &discordgo.MessageSend{
		Content: "Click the button below to start signup:",
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Signup",
						Style:    discordgo.PrimaryButton,
						CustomID: metadataStr, // ✅ Store placeholder + user ID
					},
				},
			},
		},
	})

	if err != nil {
		slog.Error("Failed to send ephemeral signup message", attr.UserID(r.UserID), attr.Error(err))
	} else {
		slog.Info("Ephemeral signup message successfully sent!", attr.UserID(r.UserID))
	}
}

// New handler for the button press
func (h *gatewayEventHandler) HandleSignupButtonPress(ctx context.Context, i *discordgo.InteractionCreate) {
	slog.Info("✅ Signup button clicked!", attr.String("custom_id", i.MessageComponentData().CustomID))

	var userID string
	if i.Member != nil {
		userID = i.Member.User.ID
	} else {
		userID = i.User.ID
	}

	// ✅ Send the signup modal and log if it fails
	err := h.discord.SendSignupModal(ctx, i.Interaction)
	if err != nil {
		slog.Error("❌ Failed to send signup modal", attr.Error(err))
		h.tokenStore.Get(userID) //remove on error.
	} else {
		slog.Info("✅ Signup modal successfully sent!")
	}
}

func (h *gatewayEventHandler) HandleSignupModalSubmit(ctx context.Context, i *discordgo.InteractionCreate) {
	slog.Info("Handling ModalSubmit", attr.String("custom_id", i.ModalSubmitData().CustomID))

	// ✅ 1️⃣ Acknowledge interaction immediately with a confirmation message
	err := h.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "✅ Signup request submitted successfully! Processing...",
			Flags:   discordgo.MessageFlagsEphemeral, // Make it ephemeral
		},
	})
	if err != nil {
		slog.Error("❌ Failed to acknowledge modal submission", attr.Error(err))
		return
	}

	// ✅ 2️⃣ Get user ID
	var userID string
	if i.Member != nil {
		userID = i.Member.User.ID
	} else {
		userID = i.User.ID
	}

	// ✅ 3️⃣ Process form submission
	data := i.ModalSubmitData()
	tagNumberPtr, err := h.extractTagNumber(&data)
	if err != nil {
		slog.Warn("⚠️ Invalid tag number", attr.Error(err))
		// Backend will handle error DM
		return
	}

	// ✅ 4️⃣ Create event payload
	payload := userevents.UserSignupRequestPayload{
		DiscordID: usertypes.DiscordID(userID),
		TagNumber: tagNumberPtr,
	}

	slog.Info("📝 Creating event", attr.String("user_id", userID))

	// ✅ 5️⃣ Create event message
	msg, err := h.createEvent(ctx, userevents.UserSignupRequest, payload, i)
	if err != nil {
		slog.Error("❌ Failed to create event", attr.Error(err))
		// Backend will handle error DM
		return
	}

	// ✅ 6️⃣ Publish event to backend
	slog.Info("🚀 Publishing signup form submitted event")
	if err := h.publisher.Publish(userevents.UserSignupRequest, msg); err != nil {
		slog.Error("❌ Failed to publish event", attr.Error(err))
		return
	}

	slog.Info("✅ Signup form event published successfully")
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

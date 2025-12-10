package udisc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

// HandleSetUDiscNameCommand handles the /set-udisc-name slash command.
func (m *udiscManager) HandleSetUDiscNameCommand(ctx context.Context, i *discordgo.InteractionCreate) (UDiscOperationResult, error) {
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "set_udisc_name")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "application_command")

	return m.operationWrapper(ctx, "HandleSetUDiscNameCommand", func(ctx context.Context) (UDiscOperationResult, error) {
		// Extract user ID
		userID := ""
		if i.Member != nil && i.Member.User != nil {
			userID = i.Member.User.ID
		} else if i.User != nil {
			userID = i.User.ID
		}

		if userID == "" {
			return UDiscOperationResult{Error: fmt.Errorf("user ID is missing")}, fmt.Errorf("user ID is missing")
		}

		guildID := i.GuildID
		if guildID == "" && m.config != nil {
			guildID = m.config.Discord.GuildID
		}

		ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, userID)
		ctx = discordmetrics.WithValue(ctx, discordmetrics.GuildIDKey, guildID)

		// Extract username and name from command options
		var username, name string
		if len(i.ApplicationCommandData().Options) > 0 {
			username = strings.TrimSpace(i.ApplicationCommandData().Options[0].StringValue())
		}
		if len(i.ApplicationCommandData().Options) > 1 {
			name = strings.TrimSpace(i.ApplicationCommandData().Options[1].StringValue())
		}

		if username == "" && name == "" {
			err := m.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Please provide at least a UDisc username or name.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			if err != nil {
				return UDiscOperationResult{Error: err}, err
			}
			return UDiscOperationResult{Error: fmt.Errorf("both fields are empty")}, fmt.Errorf("both fields are empty")
		}

		m.logger.InfoContext(ctx, "Setting UDisc identity",
			attr.String("user_id", userID),
			attr.String("guild_id", guildID),
			attr.String("username", username),
			attr.String("name", name),
		)

		// Publish event to backend
		var usernamePtr, namePtr *string
		if username != "" {
			usernamePtr = &username
		}
		if name != "" {
			namePtr = &name
		}

		payload := userevents.UpdateUDiscIdentityRequestPayload{
			GuildID:  sharedtypes.GuildID(guildID),
			UserID:   sharedtypes.DiscordID(userID),
			Username: usernamePtr,
			Name:     namePtr,
		}

		// Create message
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return UDiscOperationResult{Error: err}, err
		}

		correlationID := uuid.New().String()
		msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
		msg.Metadata.Set("correlation_id", correlationID)
		msg.Metadata.Set("user_id", userID)
		msg.Metadata.Set("guild_id", guildID)

		if err := m.publisher.Publish(userevents.UpdateUDiscIdentityRequest, msg); err != nil {
			m.logger.ErrorContext(ctx, "Failed to publish UDisc update event",
				attr.Error(err),
			)

			_ = m.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Failed to update UDisc name. Please try again later.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return UDiscOperationResult{Error: err}, err
		}

		// Send confirmation
		confirmMsg := "✅ UDisc identity updated:\n"
		if username != "" {
			confirmMsg += fmt.Sprintf("Username: **%s**\n", username)
		}
		if name != "" {
			confirmMsg += fmt.Sprintf("Name: **%s**\n", name)
		}
		confirmMsg += "\nThese will be used to match your scores when importing UDisc scorecards."

		err = m.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: confirmMsg,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			return UDiscOperationResult{Error: err}, err
		}

		// Store interaction for potential follow-up messages
		m.storeInteraction(i.Interaction.ID, i.Interaction)

		return UDiscOperationResult{Success: "udisc_name_set"}, nil
	})
}

// storeInteraction is a helper to store interaction data (implementation can be added if needed)
func (m *udiscManager) storeInteraction(id string, interaction *discordgo.Interaction) {
	// Store for 10 minutes in case we need to send follow-up messages
	// This would need an interaction store injected into the manager if we want persistence
	_ = id
	_ = interaction
}

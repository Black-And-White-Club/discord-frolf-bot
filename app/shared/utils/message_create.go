package messagecreator

import (
	"fmt"

	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
)

func BuildWatermillMessageFromInteraction(topic string, payload interface{}, interaction *discordgo.InteractionCreate, helper utils.Helpers, config *config.Config) (*message.Message, error) {
	msg, err := helper.CreateNewMessage(payload, topic)
	if err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	msg.Metadata.Set("handler_name", "discord")
	msg.Metadata.Set("domain", "discord")

	if interaction != nil && interaction.Interaction != nil {
		msg.Metadata.Set("interaction_id", interaction.Interaction.ID)
		msg.Metadata.Set("interaction_token", interaction.Interaction.Token)
	}

	if config != nil {
		msg.Metadata.Set("guild_id", config.GetGuildID())
	}

	return msg, nil
}

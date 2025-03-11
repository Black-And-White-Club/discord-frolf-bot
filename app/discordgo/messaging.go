// discord/messaging.go
package discord

import (
	"context"
	"fmt"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

// SendDM sends a direct message to a user.
func (d *discordOperations) SendDM(ctx context.Context, userID, message string) (*discordgo.Message, error) {
	channel, err := d.session.UserChannelCreate(userID)
	if err != nil {
		d.logger.Error(ctx, "Failed to create DM channel", attr.Error(err))
		return nil, fmt.Errorf("failed to create DM channel: %w", err)
	}
	msg, err := d.session.ChannelMessageSend(channel.ID, message)
	if err != nil {
		d.logger.Error(ctx, "Failed to send DM", attr.Error(err))
		return nil, fmt.Errorf("failed to send DM: %w", err)
	}
	d.logger.Info(ctx, "DM sent successfully", attr.String("discord_message_id", msg.ID), attr.String("discord_channel_id", msg.ChannelID))
	return msg, nil
}

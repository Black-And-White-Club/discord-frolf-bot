package updateround

import (
	"context"
	"fmt"
	"time"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
)

func (urm *updateRoundManager) UpdateRoundEventEmbed(ctx context.Context, channelID string, messageID string, title *roundtypes.Title, description *roundtypes.Description, startTime *sharedtypes.StartTime, location *roundtypes.Location) (UpdateRoundOperationResult, error) {
	return urm.operationWrapper(ctx, "UpdateRoundEventEmbed", func(ctx context.Context) (UpdateRoundOperationResult, error) {
		// Fetch the original message first to get existing data
		msg, err := urm.session.ChannelMessage(channelID, messageID)
		if err != nil {
			return UpdateRoundOperationResult{Error: fmt.Errorf("failed to fetch message: %w", err)}, err
		}

		if len(msg.Embeds) == 0 {
			return UpdateRoundOperationResult{Error: fmt.Errorf("no embeds found in message")}, nil
		}

		originalEmbed := msg.Embeds[0]

		// Get fields that should remain unchanged
		creatorInfo := ""
		if originalEmbed.Footer != nil {
			creatorInfo = originalEmbed.Footer.Text
		}

		// Handle the fields we want to update
		updatedTitle := originalEmbed.Title
		if title != nil {
			updatedTitle = fmt.Sprintf("**%s**", string(*title))
		}

		updatedDesc := originalEmbed.Description
		if description != nil {
			updatedDesc = string(*description)
		}

		// Handle time update
		var unixTimestamp int64
		var updatedTimestamp string

		if startTime != nil {
			timeValue := time.Time(*startTime)
			unixTimestamp = timeValue.Unix()
			updatedTimestamp = timeValue.Format(time.RFC3339)

		} else {
			// If no startTime is provided, keep the original timestamp
			updatedTimestamp = originalEmbed.Timestamp
			if t, err := time.Parse(time.RFC3339, originalEmbed.Timestamp); err == nil {
				unixTimestamp = t.Unix()
			}
		}

		// Check if any updates are needed
		if updatedTitle == originalEmbed.Title && updatedDesc == originalEmbed.Description && updatedTimestamp == originalEmbed.Timestamp && location == nil {
			// No updates provided, return early
			return UpdateRoundOperationResult{Success: msg}, nil
		}

		// Keep existing values for fields that weren't provided for update
		fields := []*discordgo.MessageEmbedField{}

		// Add time field
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:  "📅 Time",
			Value: fmt.Sprintf("<t:%d:f>  (**Starts**: <t:%d:R>)", unixTimestamp, unixTimestamp),
		})

		// Add location field
		locationValue := ""
		if len(originalEmbed.Fields) > 1 {
			locationValue = originalEmbed.Fields[1].Value
		}
		if location != nil {
			locationValue = string(*location)
		}
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:  "📍 Location",
			Value: locationValue,
		})

		// Add participant fields (preserve existing values)
		for i := 2; i < len(originalEmbed.Fields); i++ {
			fields = append(fields, originalEmbed.Fields[i])
		}

		// Create the updated embed
		updatedEmbed := &discordgo.MessageEmbed{
			Title:       updatedTitle,
			Description: updatedDesc,
			Color:       originalEmbed.Color,
			Fields:      fields,
			Footer: &discordgo.MessageEmbedFooter{
				Text: creatorInfo,
			},
			Timestamp: updatedTimestamp,
		}

		// Update the message only if there are changes
		updatedMsg, err := urm.session.ChannelMessageEditEmbed(channelID, messageID, updatedEmbed)
		if err != nil {
			return UpdateRoundOperationResult{Error: fmt.Errorf("failed to update embed: %w", err)}, err
		}

		return UpdateRoundOperationResult{Success: updatedMsg}, nil
	})
}

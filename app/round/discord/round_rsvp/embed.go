package roundrsvp

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// UpdateRoundEventEmbed updates the round event embed with new participant information.
func (rrm *roundRsvpManager) UpdateRoundEventEmbed(ctx context.Context, channelID string, messageID sharedtypes.RoundID, acceptedParticipants, declinedParticipants, tentativeParticipants []roundtypes.Participant) (RoundRsvpOperationResult, error) {
	return rrm.operationWrapper(ctx, "UpdateRoundEventEmbed", func(ctx context.Context) (RoundRsvpOperationResult, error) {
		// Fetch the message from the channel
		msg, err := rrm.session.ChannelMessage(channelID, messageID.String())
		if err != nil {
			wrappedErr := fmt.Errorf("failed to fetch message: %w", err)
			rrm.logger.ErrorContext(ctx, "Failed to fetch message for embed update",
				attr.Error(wrappedErr),
				attr.String("channel_id", channelID),
				attr.String("message_id", messageID.String()))
			return RoundRsvpOperationResult{Error: wrappedErr}, wrappedErr
		}

		if len(msg.Embeds) == 0 {
			err := fmt.Errorf("no embeds found in message")
			rrm.logger.ErrorContext(ctx, "No embeds found in message",
				attr.String("channel_id", channelID),
				attr.String("message_id", messageID.String()))
			return RoundRsvpOperationResult{Error: err}, err
		}

		// Update the embed message
		embed := msg.Embeds[0]

		// Ensure we have enough fields in the embed
		if len(embed.Fields) < 5 {
			err := fmt.Errorf("embed does not have expected fields")
			rrm.logger.ErrorContext(ctx, "Embed doesn't have expected fields",
				attr.String("channel_id", channelID),
				attr.String("message_id", messageID.String()),
				attr.Int("field_count", len(embed.Fields)))
			return RoundRsvpOperationResult{Error: err}, err
		}

		embed.Fields[2].Value = rrm.formatParticipants(ctx, acceptedParticipants)
		embed.Fields[3].Value = rrm.formatParticipants(ctx, declinedParticipants)
		embed.Fields[4].Value = rrm.formatParticipants(ctx, tentativeParticipants)

		// Update the message in the channel
		updatedMsg, err := rrm.session.ChannelMessageEditEmbed(channelID, messageID.String(), embed)
		if err != nil {
			wrappedErr := fmt.Errorf("failed to update embed: %w", err)
			rrm.logger.ErrorContext(ctx, "Failed to update round event embed",
				attr.Error(wrappedErr),
				attr.String("channel_id", channelID),
				attr.String("message_id", messageID.String()))
			return RoundRsvpOperationResult{Error: wrappedErr}, wrappedErr
		}

		rrm.logger.InfoContext(ctx, "Successfully updated round event embed",
			attr.String("channel_id", channelID),
			attr.String("message_id", messageID.String()),
			attr.Int("accepted_count", len(acceptedParticipants)),
			attr.Int("declined_count", len(declinedParticipants)),
			attr.Int("tentative_count", len(tentativeParticipants)))

		return RoundRsvpOperationResult{Success: updatedMsg}, nil
	})
}

// formatParticipants formats the participant list for the embed.
func (rrm *roundRsvpManager) formatParticipants(ctx context.Context, participants []roundtypes.Participant) string {
	if len(participants) == 0 {
		return "-"
	}

	// Separate participants into those with and without a tag number
	var withTag []roundtypes.Participant
	var withoutTag []roundtypes.Participant

	for _, participant := range participants {
		if participant.TagNumber != nil && *participant.TagNumber > 0 {
			withTag = append(withTag, participant)
		} else {
			withoutTag = append(withoutTag, participant)
		}
	}

	// Sort by TagNumber in ascending order
	sort.Slice(withTag, func(i, j int) bool {
		return *withTag[i].TagNumber < *withTag[j].TagNumber
	})

	// Merge sorted participants with those without a tag
	sortedParticipants := append(withTag, withoutTag...)

	// Format participant list
	var names []string
	for _, participant := range sortedParticipants {
		user, err := rrm.session.User(string(participant.UserID))
		if err != nil {
			rrm.logger.ErrorContext(ctx, "Failed to get user",
				attr.Error(err),
				attr.String("user_id", string(participant.UserID)))
			names = append(names, "Unknown User")
			continue
		}

		// Fetch the user's nickname from the guild if available
		member, err := rrm.session.GuildMember(rrm.config.Discord.GuildID, string(participant.UserID))
		displayName := user.Username
		if err == nil && member.Nick != "" {
			displayName = member.Nick
		}

		// Append with or without TagNumber
		if participant.TagNumber != nil && *participant.TagNumber > 0 {
			names = append(names, fmt.Sprintf("%s Tag: %d", displayName, *participant.TagNumber))
		} else {
			names = append(names, displayName)
		}
	}

	return strings.Join(names, "\n")
}

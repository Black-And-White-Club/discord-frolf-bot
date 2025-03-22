package roundrsvp

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
)

// UpdateRoundEventEmbed updates the round event embed with new participant information.
func (rrm *roundRsvpManager) UpdateRoundEventEmbed(channelID string, messageID roundtypes.EventMessageID, acceptedParticipants, declinedParticipants, tentativeParticipants []roundtypes.Participant) error {
	// Fetch the message from the channel
	msg, err := rrm.session.ChannelMessage(channelID, string(messageID))
	if err != nil {
		return err
	}

	// Update the embed message
	embed := msg.Embeds[0]
	embed.Fields[2].Value = rrm.formatParticipants(acceptedParticipants)
	embed.Fields[3].Value = rrm.formatParticipants(declinedParticipants)
	embed.Fields[4].Value = rrm.formatParticipants(tentativeParticipants)

	// Update the message in the channel
	_, err = rrm.session.ChannelMessageEditEmbed(channelID, string(messageID), embed)
	if err != nil {
		slog.Error("Failed to update round event embed", attr.Error(err))
		return err
	}

	return nil
}

// formatParticipants formats the participant list for the embed.
func (rrm *roundRsvpManager) formatParticipants(participants []roundtypes.Participant) string {
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
			slog.Error("Failed to get user", attr.Error(err), attr.String("user_id", string(participant.UserID)))
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

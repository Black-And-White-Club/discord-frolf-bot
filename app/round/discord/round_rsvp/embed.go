package roundrsvp

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
)

// UpdateRoundEventEmbed updates the round event embed with new participant information.
func (rrm *roundRsvpManager) UpdateRoundEventEmbed(channelID, messageID string, acceptedParticipants, declinedParticipants, tentativeParticipants []roundtypes.Participant) error {
	// Fetch the message from the channel
	msg, err := rrm.session.ChannelMessage(channelID, messageID)
	if err != nil {
		return err
	}

	// Update the embed message
	embed := msg.Embeds[0]
	embed.Fields[2].Value = rrm.formatParticipants(acceptedParticipants)
	embed.Fields[3].Value = rrm.formatParticipants(declinedParticipants)
	embed.Fields[4].Value = rrm.formatParticipants(tentativeParticipants)

	// Update the message in the channel
	_, err = rrm.session.ChannelMessageEditEmbed(channelID, messageID, embed)
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

	var names []string
	for _, participant := range participants {
		user, err := rrm.session.User(string(participant.UserID))
		if err != nil {
			slog.Error("Failed to get user", attr.Error(err), attr.String("user_id", string(participant.UserID)))
			// Return a placeholder instead of the error directly
			names = append(names, fmt.Sprintf("Unknown User (Tag: %d) - %s", participant.TagNumber, participant.Response))
			continue
		}
		names = append(names, fmt.Sprintf("%s (Tag: %d) - %s", user.Username, participant.TagNumber, participant.Response))
	}
	return strings.Join(names, "\n")
}

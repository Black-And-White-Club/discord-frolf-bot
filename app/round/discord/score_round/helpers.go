package scoreround

import (
	"fmt"
	"strings"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

func isTeamHeader(fieldName string) bool {
	return strings.Contains(fieldName, "Total")
}

func formatParticipantLine(
	userID sharedtypes.DiscordID,
	score *sharedtypes.Score,
	tag *sharedtypes.TagNumber,
) string {
	tagDisplay := ""
	if tag != nil && *tag > 0 {
		tagDisplay = fmt.Sprintf(" %s %d", tagPrefix, *tag)
	}

	scoreDisplay := fmt.Sprintf(" — %s %s", scorePrefix, scoreNoData)
	if score != nil {
		scoreDisplay = fmt.Sprintf(" — %s %+d", scorePrefix, *score)
	}

	return fmt.Sprintf("<@%s>%s%s", userID, tagDisplay, scoreDisplay)
}

func isParticipantField(name, value string) bool {
	ln := strings.ToLower(name)
	return strings.Contains(ln, "accepted") ||
		strings.Contains(ln, "tentative") ||
		strings.Contains(ln, "declined") ||
		strings.Contains(ln, "participants") ||
		strings.Contains(ln, "✅") ||
		strings.Contains(ln, "❓") ||
		strings.Contains(ln, "❌") ||
		strings.Contains(value, "<@")
}

// participantsToEmbedLines converts a slice of participants to embed-ready lines
func participantsToEmbedLines(participants []roundtypes.Participant) []string {
	lines := make([]string, len(participants))
	for i, p := range participants {
		lines[i] = formatParticipantLine(p.UserID, p.Score, p.TagNumber)
	}
	return lines
}

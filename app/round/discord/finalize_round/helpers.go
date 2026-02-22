package finalizeround

import (
	"fmt"
	"sort"
	"strings"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
)

// renderTeamsFinalizedFields formats finalized teams into Discord embed fields.
// Teams are sorted by score (lowest first, since lower is better in disc golf).
// It also returns a map of DiscordIDs that are present in the teams.
func renderTeamsFinalizedFields(teams []roundtypes.NormalizedTeam) ([]*discordgo.MessageEmbedField, map[sharedtypes.DiscordID]struct{}) {

	fields := make([]*discordgo.MessageEmbedField, 0, len(teams))
	usedParticipants := make(map[sharedtypes.DiscordID]struct{})

	// Sort teams by Total score (lowest first for disc golf)
	sortedTeams := make([]roundtypes.NormalizedTeam, len(teams))
	copy(sortedTeams, teams)
	sort.Slice(sortedTeams, func(i, j int) bool {
		return sortedTeams[i].Total < sortedTeams[j].Total
	})

	for i, team := range sortedTeams {
		lines := make([]string, 0, len(team.Members))

		for _, m := range team.Members {
			// Track used participants if they have a Discord ID
			if m.UserID != nil {
				usedParticipants[*m.UserID] = struct{}{}
			}
			// Use the unified helper with Displayable interface methods
			lines = append(lines, roundtypes.DisplayName(m.UserIDPointer(), m.RawNameString()))
		}

		// Use rank emoji and ordinal position instead of UUID
		emoji := teamRankEmoji(i, len(sortedTeams))
		header := fmt.Sprintf("%s Team %d — %+d", emoji, i+1, team.Total)
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   header,
			Value:  strings.Join(lines, "\n"),
			Inline: false,
		})
	}

	return fields, usedParticipants
}

// teamRankEmoji returns an emoji based on the team's rank position
func teamRankEmoji(index, total int) string {
	switch total {
	case 1:
		return "😢"
	case 2:
		if index == 0 {
			return "🥇"
		}
		return "🗑️"
	default:
		switch index {
		case 0:
			return "🥇"
		case 1:
			return "🥈"
		case 2:
			return "🥉"
		case total - 1:
			return "🗑️"
		default:
			return "🥏"
		}
	}
}

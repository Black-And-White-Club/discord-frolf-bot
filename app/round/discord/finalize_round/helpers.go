package finalizeround

import (
	"fmt"
	"strings"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
)

// renderTeamsFinalizedFields formats finalized teams into Discord embed fields.
// It also returns a map of DiscordIDs that are present in the teams.
func renderTeamsFinalizedFields(teams []roundtypes.NormalizedTeam) ([]*discordgo.MessageEmbedField, map[sharedtypes.DiscordID]struct{}) {

	fields := make([]*discordgo.MessageEmbedField, 0, len(teams))
	usedParticipants := make(map[sharedtypes.DiscordID]struct{})

	for _, team := range teams {
		lines := make([]string, 0, len(team.Members))

		for _, m := range team.Members {
			// Track used participants if they have a Discord ID
			if m.UserID != nil {
				usedParticipants[*m.UserID] = struct{}{}
			}
			// Use the unified helper with Displayable interface methods
			lines = append(lines, roundtypes.DisplayName(m.UserIDPointer(), m.RawNameString()))
		}

		header := fmt.Sprintf("🏆 Team %s — %+d", team.TeamID, team.Total)
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   header,
			Value:  strings.Join(lines, "\n"),
			Inline: false,
		})
	}

	return fields, usedParticipants
}

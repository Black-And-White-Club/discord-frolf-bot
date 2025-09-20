package scoreround

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// parseFinalizedEmbedParticipants extracts user IDs and scores from a finalized round embed.
// Field Value format: "Score: <value|Even|--> (<@id>)".
func parseFinalizedEmbedParticipants(embed *discordgo.MessageEmbed) map[string]*int {
	result := make(map[string]*int)
	if embed == nil {
		return result
	}
	for _, f := range embed.Fields {
		if f == nil || f.Value == "" || !strings.Contains(f.Value, "(<@") || !strings.HasPrefix(f.Value, scorePrefix) {
			continue
		}
		openIdx := strings.LastIndex(f.Value, "(<@")
		closeIdx := strings.LastIndex(f.Value, ")")
		if openIdx == -1 || closeIdx == -1 || closeIdx <= openIdx+3 {
			continue
		}
		mention := f.Value[openIdx+1 : closeIdx] // <@id>
		id := strings.TrimSuffix(strings.TrimPrefix(mention, "<@"), ">")
		vParts := strings.Split(f.Value, ":")
		if len(vParts) < 2 {
			continue
		}
		raw := strings.TrimSpace(vParts[1])
		if idx := strings.Index(raw, "("); idx != -1 { // trim trailing mention copy
			raw = strings.TrimSpace(raw[:idx])
		}
		if raw == scoreNoData {
			result[id] = nil
			continue
		}
		if strings.EqualFold(raw, "Even") {
			z := 0
			result[id] = &z
			continue
		}
		raw = strings.TrimPrefix(raw, "+")
		if sv, err := strconv.Atoi(raw); err == nil {
			val := sv
			result[id] = &val
		}
	}
	return result
}

// buildPrefillLines converts parsed participants into modal prefill lines preserving existing scores.
func buildPrefillLines(participants map[string]*int) []string {
	lines := make([]string, 0, len(participants))
	ids := make([]string, 0, len(participants))
	for id := range participants {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, userID := range ids {
		scorePtr := participants[userID]
		var scoreStr string
		if scorePtr == nil {
			scoreStr = scoreNoData
		} else if *scorePtr == 0 {
			scoreStr = "0"
		} else {
			scoreStr = fmt.Sprintf("%+d", *scorePtr)
		}
		lines = append(lines, fmt.Sprintf("<@%s>=%s", userID, scoreStr))
	}
	return lines
}

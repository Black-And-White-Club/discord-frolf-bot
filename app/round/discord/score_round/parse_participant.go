package scoreround

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// Regex to extract participant data from legacy (non-finalized) embed lines.
var participantLineRegex = regexp.MustCompile(`<@!?([a-zA-Z0-9]+)>` +
	`(?:\s+` + tagPrefix + `\s*(\d+))?` +
	`(?:\s*[—–-]\s*` + scorePrefix + `\s*([+\-]?\d+|` + scoreNoData + `))?`)

func (srm *scoreRoundManager) parseParticipantLine(ctx context.Context, line string) (sharedtypes.DiscordID, *sharedtypes.Score, *sharedtypes.TagNumber, bool) {
	srm.logger.DebugContext(ctx, "Parsing participant line", attr.String("line", line))
	match := participantLineRegex.FindStringSubmatch(line)
	if len(match) < 2 || match[1] == "" {
		return "", nil, nil, false
	}

	userID := sharedtypes.DiscordID(match[1])

	var tagNumber *sharedtypes.TagNumber
	if len(match) > 2 && match[2] != "" {
		if parsedTag, err := parseInt(match[2]); err == nil {
			tn := sharedtypes.TagNumber(parsedTag)
			tagNumber = &tn
		}
	}

	var score *sharedtypes.Score
	if len(match) > 3 && match[3] != "" {
		scoreStr := strings.TrimSpace(match[3])
		if scoreStr != scoreNoData {
			if parsedScore, err := parseInt(scoreStr); err == nil {
				sc := sharedtypes.Score(parsedScore)
				score = &sc
			}
		}
	}
	return userID, score, tagNumber, true
}

func parseInt(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%+d", &result)
	if err != nil {
		_, err = fmt.Sscanf(s, "%d", &result)
	}
	return result, err
}

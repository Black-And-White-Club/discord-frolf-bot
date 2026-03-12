package challenge

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	clubtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/club"
	"github.com/bwmarrin/discordgo"
)

func buildChallengeEmbed(challenge clubtypes.ChallengeDetail) *discordgo.MessageEmbed {
	description := fmt.Sprintf("%s challenged %s",
		participantMention(challenge.ChallengerExternalID, challenge.ChallengerUserUUID),
		participantMention(challenge.DefenderExternalID, challenge.DefenderUserUUID),
	)

	fields := []*discordgo.MessageEmbedField{
		{
			Name:   "Current Tags",
			Value:  fmt.Sprintf("%s vs %s", formatTag(challenge.CurrentTags.Challenger), formatTag(challenge.CurrentTags.Defender)),
			Inline: true,
		},
		{
			Name:   "Original Tags",
			Value:  fmt.Sprintf("%s vs %s", formatTag(challenge.OriginalTags.Challenger), formatTag(challenge.OriginalTags.Defender)),
			Inline: true,
		},
		{
			Name:   "Challenge ID",
			Value:  fmt.Sprintf("`%s`", challenge.ID),
			Inline: false,
		},
	}

	if challenge.LinkedRound != nil && challenge.LinkedRound.IsActive {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Linked Round",
			Value:  fmt.Sprintf("`%s`", challenge.LinkedRound.RoundID),
			Inline: false,
		})
	}

	if deadline := challengeDeadline(challenge); deadline != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Deadline",
			Value:  deadline,
			Inline: false,
		})
	}

	embed := &discordgo.MessageEmbed{
		Title:       challengeTitle(challenge.Status),
		Description: description,
		Fields:      fields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Challenges schedule a round. Normal tag rules still apply to everyone who joins.",
		},
		Timestamp: challenge.OpenedAt.Format(time.RFC3339),
		Color:     challengeColor(challenge.Status),
	}

	return embed
}

func buildChallengeComponents(cfg *config.Config, challenge clubtypes.ChallengeDetail) []discordgo.MessageComponent {
	rows := []discordgo.MessageComponent{}
	buttons := []discordgo.MessageComponent{}

	if challenge.Status == clubtypes.ChallengeStatusOpen {
		buttons = append(buttons,
			discordgo.Button{
				Label:    "Accept",
				Style:    discordgo.SuccessButton,
				CustomID: challengeAcceptPrefix + challenge.ID,
			},
			discordgo.Button{
				Label:    "Decline",
				Style:    discordgo.DangerButton,
				CustomID: challengeDeclinePrefix + challenge.ID,
			},
		)
	} else if challenge.Status == clubtypes.ChallengeStatusAccepted && (challenge.LinkedRound == nil || !challenge.LinkedRound.IsActive) {
		buttons = append(buttons, discordgo.Button{
			Label:    "Schedule Round",
			Style:    discordgo.PrimaryButton,
			CustomID: challengeSchedulePrefix + challenge.ID,
		})
	}

	if appURL := challengeAppURL(cfg, challenge.ID); appURL != "" {
		buttons = append(buttons, discordgo.Button{
			Label: "Open in App",
			Style: discordgo.LinkButton,
			URL:   appURL,
		})
	}

	if len(buttons) > 0 {
		rows = append(rows, discordgo.ActionsRow{Components: buttons})
	}

	return rows
}

func challengeTitle(status clubtypes.ChallengeStatus) string {
	switch status {
	case clubtypes.ChallengeStatusAccepted:
		return "Accepted Challenge"
	case clubtypes.ChallengeStatusDeclined:
		return "Declined Challenge"
	case clubtypes.ChallengeStatusWithdrawn:
		return "Withdrawn Challenge"
	case clubtypes.ChallengeStatusExpired:
		return "Expired Challenge"
	case clubtypes.ChallengeStatusCompleted:
		return "Completed Challenge"
	case clubtypes.ChallengeStatusHidden:
		return "Hidden Challenge"
	default:
		return "Open Challenge"
	}
}

func challengeColor(status clubtypes.ChallengeStatus) int {
	switch status {
	case clubtypes.ChallengeStatusAccepted:
		return 0x0ea5e9
	case clubtypes.ChallengeStatusDeclined, clubtypes.ChallengeStatusWithdrawn, clubtypes.ChallengeStatusHidden:
		return 0xef4444
	case clubtypes.ChallengeStatusCompleted:
		return 0x14b8a6
	case clubtypes.ChallengeStatusExpired:
		return 0xf59e0b
	default:
		return 0x22c55e
	}
}

func challengeDeadline(challenge clubtypes.ChallengeDetail) string {
	switch challenge.Status {
	case clubtypes.ChallengeStatusOpen:
		if challenge.OpenExpiresAt != nil {
			return challenge.OpenExpiresAt.Format(time.RFC822)
		}
	case clubtypes.ChallengeStatusAccepted:
		if challenge.AcceptedExpiresAt != nil {
			return challenge.AcceptedExpiresAt.Format(time.RFC822)
		}
	}
	return ""
}

func participantMention(externalID *string, userUUID string) string {
	if externalID != nil && *externalID != "" {
		return fmt.Sprintf("<@%s>", *externalID)
	}
	if len(userUUID) > 8 {
		return fmt.Sprintf("`%s`", userUUID[:8])
	}
	return fmt.Sprintf("`%s`", userUUID)
}

func formatTag(tag *int) string {
	if tag == nil {
		return "unranked"
	}
	return fmt.Sprintf("#%d", *tag)
}

func openAppURL(cfg *config.Config) string {
	if cfg == nil {
		return ""
	}
	baseURL := strings.TrimRight(cfg.PWA.BaseURL, "/")
	if baseURL == "" {
		return ""
	}
	return baseURL
}

func challengeAppURL(cfg *config.Config, challengeID string) string {
	baseURL := openAppURL(cfg)
	if baseURL == "" || challengeID == "" {
		return ""
	}
	return fmt.Sprintf("%s/challenges/%s", baseURL, url.PathEscape(challengeID))
}

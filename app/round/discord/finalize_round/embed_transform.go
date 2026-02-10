package finalizeround

import (
	"context"
	"fmt"
	"sort"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

const (
	finalizedColor = 0x0000FF

	fieldStarted  = "📅 Started"
	fieldLocation = "📍 Location"

	overrideButtonID = "round_bulk_score_override"
)

type participantWithUser struct {
	UserID    sharedtypes.DiscordID
	Username  string
	Score     *sharedtypes.Score
	TagNumber *sharedtypes.TagNumber
	Points    *int
}

func (frm *finalizeRoundManager) TransformRoundToFinalizedScorecard(payload roundevents.RoundFinalizedEmbedUpdatePayloadV1) (*discordgo.MessageEmbed, []discordgo.MessageComponent, error) {

	if payload.RoundID == sharedtypes.RoundID(uuid.Nil) {
		return nil, nil, fmt.Errorf("invalid payload: round ID is empty")
	}

	var (
		embed      *discordgo.MessageEmbed
		components []discordgo.MessageComponent
	)

	ctx := context.Background()

	_, err := frm.operationWrapper(ctx, "TransformRoundToFinalizedScorecard", func(ctx context.Context) (FinalizeRoundOperationResult, error) {
		frm.logger.InfoContext(ctx, "Processing round finalization",
			attr.RoundID("round_id", payload.RoundID),
			attr.String("title", string(payload.Title)),
		)

		guildID := string(payload.GuildID)
		if guildID == "" {
			guildID = frm.config.GetGuildID()
		}

		participants := make(map[sharedtypes.DiscordID]*participantWithUser)

		for i, p := range payload.Participants {
			if p.UserID == "" {
				frm.logger.WarnContext(ctx, "Skipping participant with empty UserID", attr.Int("index", i))
				continue
			}

			user, err := frm.session.User(string(p.UserID))
			if err != nil {
				frm.logger.WarnContext(ctx, "Failed to fetch user",
					attr.Error(err),
					attr.String("user_id", string(p.UserID)),
				)
				continue
			}

			username := user.Username
			if member, err := frm.session.GuildMember(guildID, string(p.UserID)); err == nil && member != nil && member.Nick != "" {
				username = member.Nick
			}

			participants[p.UserID] = &participantWithUser{
				UserID:    p.UserID,
				Username:  username,
				Score:     p.Score,
				TagNumber: p.TagNumber,
				Points:    p.Points,
			}
		}

		ordered := make([]*participantWithUser, 0, len(participants))
		for _, p := range participants {
			ordered = append(ordered, p)
		}

		sort.Slice(ordered, func(i, j int) bool {
			a, b := ordered[i], ordered[j]

			if a.Score == nil && b.Score == nil {
				return compareByTagThenUser(a, b)
			}
			if a.Score == nil {
				return false
			}
			if b.Score == nil {
				return true
			}
			if *a.Score != *b.Score {
				return *a.Score < *b.Score
			}
			return compareByTagThenUser(a, b)
		})

		participantFields := buildParticipantFields(ordered)

		location := ""
		if payload.Location != "" {
			location = string(payload.Location)
		}

		fields := []*discordgo.MessageEmbedField{}

		if payload.StartTime != nil {
			fields = append(fields, &discordgo.MessageEmbedField{
				Name:  fieldStarted,
				Value: fmt.Sprintf("<t:%d:f>", time.Time(*payload.StartTime).Unix()),
			})
		}

		fields = append(fields, &discordgo.MessageEmbedField{
			Name:  fieldLocation,
			Value: location,
		})

		if len(payload.Teams) > 0 {
			teamFields, _ := renderTeamsFinalizedFields(payload.Teams)
			fields = append(fields, teamFields...)
		} else {
			fields = append(fields, participantFields...)
		}

		title := "Round Finalized"
		if payload.Title != "" {
			title = fmt.Sprintf("**%s** - Round Finalized", payload.Title)
		}

		embed = &discordgo.MessageEmbed{
			Title:       title,
			Description: fmt.Sprintf("Round at %s has been finalized. Admin/Editor access required for score updates.", location),
			Color:       finalizedColor,
			Fields:      fields,
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Round has been finalized. Only admins/editors can update scores.",
			},
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}

		if len(payload.Teams) > 0 {
			// For teams rounds, clear all buttons (no score override for doubles)
			components = []discordgo.MessageComponent{}
		} else {
			components = []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Score Override",
							Style:    discordgo.DangerButton,
							CustomID: fmt.Sprintf("%s|%s", overrideButtonID, payload.RoundID),
							Emoji:    &discordgo.ComponentEmoji{Name: "🛠️"},
						},
					},
				},
			}
		}

		frm.logger.InfoContext(ctx, "Successfully transformed round to finalized scorecard",
			attr.RoundID("round_id", payload.RoundID),
		)

		return FinalizeRoundOperationResult{Success: true}, nil
	})

	if err != nil {
		frm.logger.ErrorContext(ctx, "Failed to transform round to finalized scorecard",
			attr.Error(err),
			attr.RoundID("round_id", payload.RoundID),
		)
		return nil, nil, fmt.Errorf("failed to transform round to finalized scorecard: %w", err)
	}

	if embed == nil {
		return nil, nil, fmt.Errorf("transformed embed is nil for round %s", payload.RoundID)
	}

	return embed, components, nil
}

func compareByTagThenUser(a, b *participantWithUser) bool {
	if a.TagNumber != nil && b.TagNumber != nil {
		return *a.TagNumber < *b.TagNumber
	}
	if a.TagNumber != nil {
		return true
	}
	if b.TagNumber != nil {
		return false
	}
	return string(a.UserID) < string(b.UserID)
}

func buildParticipantFields(participants []*participantWithUser) []*discordgo.MessageEmbedField {
	total := len(participants)
	fields := make([]*discordgo.MessageEmbedField, 0, total)

	for i, p := range participants {
		score := "Score: --"
		if p.Score != nil {
			if *p.Score == 0 {
				score = "Score: Even"
			} else {
				score = fmt.Sprintf("Score: %+d", *p.Score)
			}
		}

		if p.Points != nil {
			score = fmt.Sprintf("%s • %d pts", score, *p.Points)
		}

		emoji := rankEmoji(i, total)

		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("%s %s", emoji, p.Username),
			Value:  fmt.Sprintf("%s (<@%s>)", score, p.UserID),
			Inline: false,
		})
	}

	return fields
}

func rankEmoji(index, total int) string {
	switch total {
	case 1:
		return "😢"
	case 2:
		if index == 0 {
			return "🥇"
		}
		return "🗑️"
	case 3:
		switch index {
		case 0:
			return "🥇"
		case 1:
			return "🥈"
		default:
			return "🗑️"
		}
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
			return "🏌️"
		}
	}
}

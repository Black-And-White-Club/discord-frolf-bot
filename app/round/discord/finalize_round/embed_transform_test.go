package finalizeround

import (
	"context"
	"fmt"
	"testing"
	"time"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestTransformRoundToFinalizedScorecard(t *testing.T) {
	fixedTime := time.Date(2025, 3, 15, 10, 0, 0, 0, time.UTC)
	roundID := sharedtypes.RoundID(uuid.New())

	score := func(v sharedtypes.Score) *sharedtypes.Score { return &v }
	start := (*sharedtypes.StartTime)(&fixedTime)

	tests := []struct {
		name       string
		payload    roundevents.RoundFinalizedEmbedUpdatePayloadV1
		mockUsers  map[string]string // userID -> username
		mockNicks  map[string]string // userID -> nick
		wantFields []string
		wantButton bool
		wantErr    bool
	}{
		{
			name: "participants sorted by score asc",
			payload: roundevents.RoundFinalizedEmbedUpdatePayloadV1{
				RoundID:   roundID,
				Location:  "Test Course",
				StartTime: start,
				Participants: []roundtypes.Participant{
					{UserID: "u1", Score: score(3)},
					{UserID: "u2", Score: score(-1)},
				},
			},
			mockUsers: map[string]string{
				"u1": "Alice",
				"u2": "Bob",
			},
			wantFields: []string{
				"🥇 Bob | Score: -1 (<@u2>)",
				"🗑️ Alice | Score: +3 (<@u1>)",
			},
			wantButton: true,
		},
		{
			name: "nickname preferred over username",
			payload: roundevents.RoundFinalizedEmbedUpdatePayloadV1{
				RoundID:   roundID,
				Location:  "Test Course",
				StartTime: start,
				Participants: []roundtypes.Participant{
					{UserID: "u1", Score: score(0)},
				},
			},
			mockUsers: map[string]string{"u1": "Alice"},
			mockNicks: map[string]string{"u1": "Ace"},
			wantFields: []string{
				"😢 Ace | Score: Even (<@u1>)",
			},
			wantButton: true,
		},
		{
			name: "nil score rendered as -- and sorted last",
			payload: roundevents.RoundFinalizedEmbedUpdatePayloadV1{
				RoundID:   roundID,
				Location:  "Test Course",
				StartTime: start,
				Participants: []roundtypes.Participant{
					{UserID: "u1", Score: nil},
					{UserID: "u2", Score: score(-2)},
				},
			},
			mockUsers: map[string]string{
				"u1": "Alice",
				"u2": "Bob",
			},
			wantFields: []string{
				"🥇 Bob | Score: -2 (<@u2>)",
				"🗑️ Alice | Score: -- (<@u1>)",
			},
			wantButton: true,
		},
		{
			name: "participant skipped when user fetch fails",
			payload: roundevents.RoundFinalizedEmbedUpdatePayloadV1{
				RoundID:   roundID,
				Location:  "Test Course",
				StartTime: start,
				Participants: []roundtypes.Participant{
					{UserID: "missing", Score: score(1)},
				},
			},
			mockUsers:  map[string]string{},
			wantFields: nil,
			wantButton: true,
		},
		{
			name: "participant with points",
			payload: roundevents.RoundFinalizedEmbedUpdatePayloadV1{
				RoundID:   roundID,
				Location:  "Test Course",
				StartTime: start,
				Participants: []roundtypes.Participant{
					{UserID: "u1", Score: score(0), Points: intPtr(10)},
					{UserID: "u2", Score: score(-2), Points: intPtr(20)},
				},
			},
			mockUsers: map[string]string{
				"u1": "Alice",
				"u2": "Bob",
			},
			wantFields: []string{
				"🥇 Bob | Score: -2 • 20 pts (<@u2>)",
				"🗑️ Alice | Score: Even • 10 pts (<@u1>)",
			},
			wantButton: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeSession := discord.NewFakeSession()

			fakeSession.UserFunc = func(id string, options ...discordgo.RequestOption) (*discordgo.User, error) {
				if name, ok := tt.mockUsers[id]; ok {
					return &discordgo.User{Username: name}, nil
				}
				return nil, fmt.Errorf("user not found")
			}

			fakeSession.GuildMemberFunc = func(guildID, id string, options ...discordgo.RequestOption) (*discordgo.Member, error) {
				if nick, ok := tt.mockNicks[id]; ok {
					return &discordgo.Member{Nick: nick}, nil
				}
				return nil, fmt.Errorf("member not found")
			}

			frm := &finalizeRoundManager{
				session: fakeSession,
				logger:  loggerfrolfbot.NoOpLogger,
				config: &config.Config{
					Discord: config.DiscordConfig{GuildID: "guild-id"},
				},
				operationWrapper: func(_ context.Context, _ string, fn func(context.Context) (FinalizeRoundOperationResult, error)) (FinalizeRoundOperationResult, error) {
					return fn(context.Background())
				},
			}

			embed, components, err := frm.TransformRoundToFinalizedScorecard(tt.payload)

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, embed)

			got := fieldValues(embed)
			require.ElementsMatch(t, tt.wantFields, got[2:]) // skip started + location

			if tt.wantButton {
				require.NotEmpty(t, components)
			} else {
				require.Empty(t, components)
			}
		})
	}
}

func fieldValues(embed *discordgo.MessageEmbed) []string {
	out := make([]string, 0, len(embed.Fields))
	for _, f := range embed.Fields {
		out = append(out, fmt.Sprintf("%s | %s", f.Name, f.Value))
	}
	return out
}

func findField(embed *discordgo.MessageEmbed, name string) *discordgo.MessageEmbedField {
	for _, f := range embed.Fields {
		if f.Name == name {
			return f
		}
	}
	return nil
}

func intPtr(v int) *int {
	return &v
}

package finalizeround

import (
	"context"
	"fmt"
	"testing"
	"time"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSession := discordmocks.NewMockSession(ctrl)

			for id, name := range tt.mockUsers {
				mockSession.EXPECT().
					User(id).
					Return(&discordgo.User{Username: name}, nil)
			}

			for id, nick := range tt.mockNicks {
				mockSession.EXPECT().
					GuildMember("guild-id", id).
					Return(&discordgo.Member{Nick: nick}, nil)
			}

			// Allow other User/GuildMember calls (e.g. missing users or members) to return errors
			mockSession.EXPECT().
				User(gomock.Any()).
				AnyTimes().
				Return(nil, fmt.Errorf("user not found"))

			mockSession.EXPECT().
				GuildMember(gomock.Any(), gomock.Any()).
				AnyTimes().
				Return(nil, fmt.Errorf("member not found"))

			frm := &finalizeRoundManager{
				session: mockSession,
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

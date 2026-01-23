package finalizeround

import (
	"context"
	"fmt"
	"strings"
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
	"go.uber.org/mock/gomock"
)

func Test_finalizeRoundManager_FinalizeScorecardEmbed(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	fixedTime := time.Date(2025, 3, 15, 10, 0, 0, 0, time.UTC)

	timePtr := func(t time.Time) *time.Time { return &t }

	baseMessage := func() *discordgo.Message {
		return &discordgo.Message{
			ID: testRoundID.String(),
			Embeds: []*discordgo.MessageEmbed{
				{
					Title: "Test Round",
					Fields: []*discordgo.MessageEmbedField{
						{Name: "📍 Location", Value: "Preserved Course"},
					},
				},
			},
		}
	}

	tests := []struct {
		name      string
		setup     func(*finalizeRoundManager, *discordmocks.MockSession)
		payload   roundevents.RoundFinalizedEmbedUpdatePayloadV1
		channelID string
		messageID string
		expectErr bool
	}{
		{
			name: "successfully preserves location when payload missing",
			setup: func(frm *finalizeRoundManager, m *discordmocks.MockSession) {
				m.EXPECT().
					ChannelMessage("test-channel", testRoundID.String()).
					Return(baseMessage(), nil)

				m.EXPECT().
					ChannelMessageEditComplex(gomock.Any()).
					DoAndReturn(func(edit *discordgo.MessageEdit, _ ...discordgo.RequestOption) (*discordgo.Message, error) {
						embed := (*edit.Embeds)[0]
						found := false
						for _, f := range embed.Fields {
							if strings.Contains(strings.ToLower(f.Name), "location") {
								found = true
								if f.Value != "Preserved Course" {
									t.Fatalf("location not preserved, got %q", f.Value)
								}
							}
						}
						if !found {
							t.Fatal("location field missing in edited embed")
						}
						return &discordgo.Message{ID: edit.ID}, nil
					})
			},
			payload: roundevents.RoundFinalizedEmbedUpdatePayloadV1{
				RoundID:      testRoundID,
				Title:        "Test Round",
				StartTime:    (*sharedtypes.StartTime)(timePtr(fixedTime)),
				Participants: []roundtypes.Participant{},
			},
			channelID: "test-channel",
			messageID: testRoundID.String(),
			expectErr: false,
		},
		{
			name: "fails when ChannelMessage fetch fails",
			setup: func(frm *finalizeRoundManager, m *discordmocks.MockSession) {
				m.EXPECT().
					ChannelMessage("test-channel", testRoundID.String()).
					Return(nil, fmt.Errorf("discord error"))
			},
			payload: roundevents.RoundFinalizedEmbedUpdatePayloadV1{
				RoundID: testRoundID,
				Title:   "Test Round",
			},
			channelID: "test-channel",
			messageID: testRoundID.String(),
			expectErr: true,
		},
		{
			name:  "fails with empty message ID",
			setup: func(frm *finalizeRoundManager, m *discordmocks.MockSession) {},
			payload: roundevents.RoundFinalizedEmbedUpdatePayloadV1{
				RoundID: testRoundID,
			},
			channelID: "test-channel",
			messageID: "",
			expectErr: true,
		},
		{
			name: "fails with nil session",
			setup: func(frm *finalizeRoundManager, m *discordmocks.MockSession) {
				frm.session = nil
			},
			payload: roundevents.RoundFinalizedEmbedUpdatePayloadV1{
				RoundID: testRoundID,
			},
			channelID: "test-channel",
			messageID: testRoundID.String(),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSession := discordmocks.NewMockSession(ctrl)

			frm := &finalizeRoundManager{
				session: mockSession,
				logger:  loggerfrolfbot.NoOpLogger,
				config: &config.Config{
					Discord: config.DiscordConfig{GuildID: "guild-id"},
				},
				operationWrapper: func(
					ctx context.Context,
					_ string,
					fn func(context.Context) (FinalizeRoundOperationResult, error),
				) (FinalizeRoundOperationResult, error) {
					return fn(ctx)
				},
			}

			if tt.setup != nil {
				tt.setup(frm, mockSession)
			}

			res, err := frm.FinalizeScorecardEmbed(
				context.Background(),
				tt.messageID,
				tt.channelID,
				tt.payload,
			)

			if tt.expectErr {
				if err == nil && res.Error == nil {
					t.Fatalf("expected error, got none")
				}
				return
			}

			if err != nil || res.Error != nil {
				t.Fatalf("unexpected error: %v %v", err, res.Error)
			}

			if res.Success == nil {
				t.Fatalf("expected Success message, got nil")
			}
		})
	}
}

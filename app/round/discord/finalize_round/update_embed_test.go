package finalizeround

import (
	"context"
	"fmt"
	"strings"
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
		setup     func(*finalizeRoundManager, *discord.FakeSession)
		payload   roundevents.RoundFinalizedEmbedUpdatePayloadV1
		channelID string
		messageID string
		expectErr bool
	}{
		{
			name: "successfully preserves location when payload missing",
			setup: func(frm *finalizeRoundManager, m *discord.FakeSession) {
				m.ChannelMessageFunc = func(channelID, messageID string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return baseMessage(), nil
				}

				m.ChannelMessageEditComplexFunc = func(edit *discordgo.MessageEdit, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					embed := (*edit.Embeds)[0]
					found := false
					for _, f := range embed.Fields {
						if strings.Contains(strings.ToLower(f.Name), "location") {
							found = true
							if f.Value != "Preserved Course" {
								t.Errorf("location not preserved, got %q", f.Value)
							}
						}
					}
					if !found {
						t.Error("location field missing in edited embed")
					}
					return &discordgo.Message{ID: edit.ID}, nil
				}
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
			setup: func(frm *finalizeRoundManager, m *discord.FakeSession) {
				m.ChannelMessageFunc = func(channelID, messageID string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return nil, fmt.Errorf("discord error")
				}
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
			setup: func(frm *finalizeRoundManager, m *discord.FakeSession) {},
			payload: roundevents.RoundFinalizedEmbedUpdatePayloadV1{
				RoundID: testRoundID,
			},
			channelID: "test-channel",
			messageID: "",
			expectErr: true,
		},
		{
			name: "fails with nil session",
			setup: func(frm *finalizeRoundManager, m *discord.FakeSession) {
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
			fakeSession := discord.NewFakeSession()

			frm := &finalizeRoundManager{
				session: fakeSession,
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
				tt.setup(frm, fakeSession)
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

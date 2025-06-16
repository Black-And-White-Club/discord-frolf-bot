package leaderboardhandlers

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"testing"

	leaderboardupdated "github.com/Black-And-White-Club/discord-frolf-bot/app/leaderboard/discord/leaderboard_updated"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/leaderboard/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	util_mocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestLeaderboardHandlers_HandleBatchTagAssigned(t *testing.T) {
	tests := []struct {
		name    string
		msg     *message.Message
		want    []*message.Message
		wantErr bool
		setup   func(*gomock.Controller, *mocks.MockLeaderboardDiscordInterface, *util_mocks.MockHelpers, *config.Config)
	}{
		{
			name: "successful_batch_tag_assigned",
			msg: &message.Message{
				UUID:    "1",
				Payload: []byte(`{"requesting_user_id": "user123", "batch_id": "batch456", "assignment_count": 3, "assignments": [{"user_id": "user1", "tag_number": 1}, {"user_id": "user2", "tag_number": 2}, {"user_id": "user3", "tag_number": 3}]}`),
				Metadata: message.Metadata{
					"correlation_id": "correlation_id",
				},
			},
			want:    []*message.Message{{}},
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockLeaderboardDiscord *mocks.MockLeaderboardDiscordInterface, mockHelper *util_mocks.MockHelpers, cfg *config.Config) {
				expectedPayload := leaderboardevents.BatchTagAssignedPayload{
					RequestingUserID: "user123",
					BatchID:          "batch456",
					AssignmentCount:  3,
					Assignments: []leaderboardevents.TagAssignmentInfo{
						{UserID: "user1", TagNumber: 1},
						{UserID: "user2", TagNumber: 2},
						{UserID: "user3", TagNumber: 3},
					},
				}

				// Configure Discord channel ID
				cfg.Discord.LeaderboardChannelID = "test-channel-id"

				// Make sure this is called by the wrapper
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&leaderboardevents.BatchTagAssignedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*leaderboardevents.BatchTagAssignedPayload) = expectedPayload
						return nil
					}).
					Times(1)

				mockLeaderboardUpdateManager := mocks.NewMockLeaderboardUpdateManager(ctrl)
				mockLeaderboardDiscord.EXPECT().GetLeaderboardUpdateManager().Return(mockLeaderboardUpdateManager).Times(1)

				// Check that the entries are correctly formatted and sorted
				expectedEntries := []leaderboardupdated.LeaderboardEntry{
					{Rank: 1, UserID: "user1"},
					{Rank: 2, UserID: "user2"},
					{Rank: 3, UserID: "user3"},
				}

				mockLeaderboardUpdateManager.EXPECT().
					SendLeaderboardEmbed(
						gomock.Any(),
						"test-channel-id",
						matchLeaderboardEntries(expectedEntries),
						int32(1),
					).
					Return(leaderboardupdated.LeaderboardUpdateOperationResult{}, nil).
					Times(1)

				expectedTracePayload := map[string]interface{}{
					"event_type":  "batch_assignment_completed",
					"status":      "embed_sent",
					"channel_id":  "test-channel-id",
					"entry_count": 3,
					"batch_id":    "batch456",
				}

				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), expectedTracePayload, leaderboardevents.LeaderboardTraceEvent).
					Return(&message.Message{}, nil).
					Times(1)
			},
		},
		{
			name: "empty_assignments",
			msg: &message.Message{
				UUID:    "2",
				Payload: []byte(`{"requesting_user_id": "user123", "batch_id": "batch456", "assignment_count": 0, "assignments": []}`),
				Metadata: message.Metadata{
					"correlation_id": "correlation_id",
				},
			},
			want:    nil,
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockLeaderboardDiscord *mocks.MockLeaderboardDiscordInterface, mockHelper *util_mocks.MockHelpers, cfg *config.Config) {
				expectedPayload := leaderboardevents.BatchTagAssignedPayload{
					RequestingUserID: "user123",
					BatchID:          "batch456",
					AssignmentCount:  0,
					Assignments:      []leaderboardevents.TagAssignmentInfo{},
				}

				// Make sure this is called by the wrapper
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&leaderboardevents.BatchTagAssignedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*leaderboardevents.BatchTagAssignedPayload) = expectedPayload
						return nil
					}).
					Times(1)

				// We should not call any other methods since the assignments are empty
				mockLeaderboardDiscord.EXPECT().GetLeaderboardUpdateManager().Times(0)
				mockHelper.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
		},
		// ... update other test cases similarly
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockHelper := util_mocks.NewMockHelpers(ctrl)
			mockLeaderboardDiscord := mocks.NewMockLeaderboardDiscordInterface(ctrl)
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
			mockMetrics := &discordmetrics.NoOpMetrics{}
			mockTracer := noop.NewTracerProvider().Tracer("test")
			cfg := &config.Config{}

			tt.setup(ctrl, mockLeaderboardDiscord, mockHelper, cfg)

			h := &LeaderboardHandlers{
				Logger:             mockLogger,
				Config:             cfg,
				Helpers:            mockHelper,
				LeaderboardDiscord: mockLeaderboardDiscord,
				Tracer:             mockTracer,
				Metrics:            mockMetrics,
				handlerWrapper: func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
					return wrapHandler(handlerName, unmarshalTo, handlerFunc, mockLogger, mockMetrics, mockTracer, mockHelper)
				},
			}

			got, err := h.HandleBatchTagAssigned(tt.msg) // Changed method name
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleBatchTagAssigned() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleBatchTagAssigned() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function to match leaderboard entries ignoring order
func matchLeaderboardEntries(expected []leaderboardupdated.LeaderboardEntry) gomock.Matcher {
	return leaderboardEntriesMatcher{expected: expected}
}

type leaderboardEntriesMatcher struct {
	expected []leaderboardupdated.LeaderboardEntry
}

func (m leaderboardEntriesMatcher) Matches(x interface{}) bool {
	entries, ok := x.([]leaderboardupdated.LeaderboardEntry)
	if !ok {
		return false
	}

	if len(entries) != len(m.expected) {
		return false
	}

	// Check that all entries match
	for i, entry := range entries {
		if entry.Rank != m.expected[i].Rank || entry.UserID != m.expected[i].UserID {
			return false
		}
	}

	return true
}

func (m leaderboardEntriesMatcher) String() string {
	return fmt.Sprintf("is equal to %v", m.expected)
}

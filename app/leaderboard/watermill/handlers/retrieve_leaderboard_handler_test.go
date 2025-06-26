package leaderboardhandlers

import (
	"context"
	"errors"
	"log/slog"
	"reflect"
	"testing"

	discordleaderboardevents "github.com/Black-And-White-Club/discord-frolf-bot/app/events/leaderboard"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	util_mocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func TestLeaderboardHandlers_HandleLeaderboardRetrieveRequest(t *testing.T) {
	tests := []struct {
		name      string
		msg       *message.Message
		want      []*message.Message
		wantErr   bool
		mockSetup func(*util_mocks.MockHelpers)
	}{
		{
			name: "successful request handling",
			msg: &message.Message{
				UUID:    "test-request",
				Payload: []byte(`{}`),
				Metadata: message.Metadata{
					"correlation_id": "test-correlation",
				},
			},
			want: []*message.Message{{}},
			mockSetup: func(mockHelper *util_mocks.MockHelpers) {
				mockHelper.EXPECT().CreateResultMessage(
					gomock.Any(),
					leaderboardevents.GetLeaderboardRequestPayload{},
					leaderboardevents.GetLeaderboardRequest,
				).Return(&message.Message{}, nil).Times(1)
			},
		},
		{
			name: "create message error",
			msg: &message.Message{
				UUID:    "test-error",
				Payload: []byte(`{}`),
			},
			wantErr: true,
			mockSetup: func(mockHelper *util_mocks.MockHelpers) {
				mockHelper.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(nil, errors.New("creation failed")).Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockHelper := util_mocks.NewMockHelpers(ctrl)
			if tt.mockSetup != nil {
				tt.mockSetup(mockHelper)
			}

			h := &LeaderboardHandlers{
				Logger:  slog.Default(),
				Helpers: mockHelper,
				handlerWrapper: func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
					return func(msg *message.Message) ([]*message.Message, error) {
						return handlerFunc(context.Background(), msg, &discordleaderboardevents.LeaderboardRetrieveRequestPayload{})
					}
				},
			}

			got, err := h.HandleLeaderboardRetrieveRequest(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleLeaderboardRetrieveRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleLeaderboardRetrieveRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLeaderboardHandlers_HandleLeaderboardData(t *testing.T) {
	tests := []struct {
		name      string
		msg       *message.Message
		want      []*message.Message
		wantErr   bool
		mockSetup func(*util_mocks.MockHelpers)
	}{
		{
			name: "handle leaderboard update notification",
			msg: &message.Message{
				Metadata: message.Metadata{"topic": leaderboardevents.LeaderboardUpdated},
			},
			want: []*message.Message{{}},
			mockSetup: func(mockHelper *util_mocks.MockHelpers) {
				mockHelper.EXPECT().CreateResultMessage(
					gomock.Any(),
					leaderboardevents.GetLeaderboardRequestPayload{},
					leaderboardevents.GetLeaderboardRequest,
				).Return(&message.Message{}, nil).Times(1)
			},
		},
		{
			name: "handle valid leaderboard response",
			msg: &message.Message{
				Metadata: message.Metadata{"topic": "backend.leaderboard.get.response"},
				Payload:  []byte(`{"leaderboard":[{"tag_number":1,"user_id":"user1"}]}`),
			},
			want: []*message.Message{{}},
			mockSetup: func(mockHelper *util_mocks.MockHelpers) {
				mockHelper.EXPECT().UnmarshalPayload(
					gomock.Any(),
					gomock.Any(),
				).Return(nil).Times(1)

				mockHelper.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(),
					discordleaderboardevents.LeaderboardRetrievedTopic,
				).Return(&message.Message{}, nil).Times(1)
			},
		},
		{
			name: "unmarshal error",
			msg: &message.Message{
				Metadata: message.Metadata{"topic": "backend.leaderboard.get.response"},
			},
			wantErr: true,
			mockSetup: func(mockHelper *util_mocks.MockHelpers) {
				mockHelper.EXPECT().UnmarshalPayload(
					gomock.Any(),
					gomock.Any(),
				).Return(errors.New("unmarshal failed")).Times(1)
			},
		},
		{
			name: "create discord message error",
			msg: &message.Message{
				Metadata: message.Metadata{"topic": "backend.leaderboard.get.response"},
				Payload:  []byte(`{"leaderboard":[{"tag_number":1,"user_id":"user1"}]}`),
			},
			wantErr: true,
			mockSetup: func(mockHelper *util_mocks.MockHelpers) {
				mockHelper.EXPECT().UnmarshalPayload(
					gomock.Any(),
					gomock.Any(),
				).Return(nil).Times(1)

				mockHelper.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(),
					discordleaderboardevents.LeaderboardRetrievedTopic,
				).Return(nil, errors.New("creation failed")).Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockHelper := util_mocks.NewMockHelpers(ctrl)
			if tt.mockSetup != nil {
				tt.mockSetup(mockHelper)
			}

			h := &LeaderboardHandlers{
				Logger:  slog.Default(),
				Helpers: mockHelper,
				handlerWrapper: func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
					return func(msg *message.Message) ([]*message.Message, error) {
						return handlerFunc(context.Background(), msg, nil)
					}
				},
			}

			got, err := h.HandleLeaderboardData(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleLeaderboardData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleLeaderboardData() = %v, want %v", got, tt.want)
			}
		})
	}
}

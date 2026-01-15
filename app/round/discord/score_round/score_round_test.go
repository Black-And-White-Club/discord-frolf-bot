package scoreround

import (
	"context"
	"errors"
	"log/slog"
	"reflect"
	"strings"
	"testing"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	guildconfigmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	storagemocks "github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot-shared/eventbus/mocks"
	utilsmocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	discordmetricsmocks "github.com/Black-And-White-Club/frolf-bot-shared/observability/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestNewScoreRoundManager(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	mockHelper := utilsmocks.NewMockHelpers(ctrl)
	mockConfig := &config.Config{}
	mockTracer := noop.NewTracerProvider().Tracer("test")
	mockMetrics := discordmetricsmocks.NewMockDiscordMetrics(ctrl)
	mockGuildConfigResolver := guildconfigmocks.NewMockGuildConfigResolver(ctrl)
	mockInteractionStore := storagemocks.NewMockISInterface[any](ctrl)
	mockGuildConfigCache := storagemocks.NewMockISInterface[storage.GuildConfig](ctrl)

	manager := NewScoreRoundManager(mockSession, mockEventBus, logger, mockHelper, mockConfig, mockInteractionStore, mockGuildConfigCache, mockTracer, mockMetrics, mockGuildConfigResolver)
	impl, ok := manager.(*scoreRoundManager)
	if !ok {
		t.Fatalf("Expected *scoreRoundManager, got %T", manager)
	}

	if impl.session != mockSession {
		t.Error("Expected session to be assigned")
	}
	if impl.publisher != mockEventBus {
		t.Error("Expected publisher to be assigned")
	}
	if impl.logger != logger {
		t.Error("Expected logger to be assigned")
	}
	if impl.helper != mockHelper {
		t.Error("Expected helper to be assigned")
	}
	if impl.config != mockConfig {
		t.Error("Expected config to be assigned")
	}
	if impl.tracer != mockTracer {
		t.Error("Expected tracer to be assigned")
	}
	if impl.metrics != mockMetrics {
		t.Error("Expected metrics to be assigned")
	}
	if impl.operationWrapper == nil {
		t.Error("Expected operationWrapper to be set")
	}
	if impl.interactionStore != mockInteractionStore {
		t.Error("Expected interactionStore to be assigned")
	}
	if impl.guildConfigCache != mockGuildConfigCache {
		t.Error("Expected guildConfigCache to be assigned")
	}
}

func Test_wrapScoreRoundOperation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	mockMetrics := discordmetricsmocks.NewMockDiscordMetrics(ctrl)
	tracer := noop.NewTracerProvider().Tracer("test")

	tests := []struct {
		name        string
		operation   string
		fn          func(context.Context) (ScoreRoundOperationResult, error)
		expectErr   string
		expectRes   ScoreRoundOperationResult
		mockMetrics func()
	}{
		{
			name:      "success path",
			operation: "handle_success",
			fn: func(ctx context.Context) (ScoreRoundOperationResult, error) {
				return ScoreRoundOperationResult{Success: "success"}, nil
			},
			expectRes: ScoreRoundOperationResult{Success: "success"},
			mockMetrics: func() {
				mockMetrics.EXPECT().RecordAPIRequestDuration(gomock.Any(), "handle_success", gomock.Any()).Times(1)
				mockMetrics.EXPECT().RecordAPIRequest(gomock.Any(), "handle_success").Times(1)
			},
		},
		{
			name:      "fn returns error",
			operation: "handle_error",
			fn: func(ctx context.Context) (ScoreRoundOperationResult, error) {
				return ScoreRoundOperationResult{}, errors.New("operation failed")
			},
			expectErr: "handle_error operation error: operation failed",
			expectRes: ScoreRoundOperationResult{Error: errors.New("handle_error operation error: operation failed")},
			mockMetrics: func() {
				mockMetrics.EXPECT().RecordAPIRequestDuration(gomock.Any(), "handle_error", gomock.Any()).Times(1)
				mockMetrics.EXPECT().RecordAPIError(gomock.Any(), "handle_error", "operation_error").Times(1)
			},
		},
		{
			name:      "result has error",
			operation: "handle_result_error",
			fn: func(ctx context.Context) (ScoreRoundOperationResult, error) {
				return ScoreRoundOperationResult{Error: errors.New("result error")}, nil
			},
			expectRes: ScoreRoundOperationResult{Error: errors.New("result error")},
			mockMetrics: func() {
				mockMetrics.EXPECT().RecordAPIRequestDuration(gomock.Any(), "handle_result_error", gomock.Any()).Times(1)
				mockMetrics.EXPECT().RecordAPIError(gomock.Any(), "handle_result_error", "result_error").Times(1)
			},
		},
		{
			name:      "panic recovery",
			operation: "handle_panic",
			fn: func(ctx context.Context) (ScoreRoundOperationResult, error) {
				panic("unexpected panic")
			},
			expectRes: ScoreRoundOperationResult{Error: nil},
			mockMetrics: func() {
				mockMetrics.EXPECT().RecordAPIRequestDuration(gomock.Any(), "handle_panic", gomock.Any()).Times(1)
				mockMetrics.EXPECT().RecordAPIError(gomock.Any(), "handle_panic", "panic").Times(1)
			},
		},
		{
			name:      "nil fn",
			operation: "handle_nil",
			fn:        nil,
			expectRes: ScoreRoundOperationResult{Error: errors.New("operation function is nil")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockMetrics != nil {
				tt.mockMetrics()
			}
			got, err := wrapScoreRoundOperation(context.Background(), tt.operation, tt.fn, logger, tracer, mockMetrics)

			if tt.expectErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
					t.Fatalf("Expected error to contain %q, got %v", tt.expectErr, err)
				}
			} else if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Compare results, handling potential nil errors gracefully
			if tt.expectRes.Success != nil && !reflect.DeepEqual(got.Success, tt.expectRes.Success) {
				t.Errorf("Success = %v, want %v", got.Success, tt.expectRes.Success)
			}
			if tt.expectRes.Error != nil {
				if got.Error == nil || !strings.Contains(got.Error.Error(), tt.expectRes.Error.Error()) {
					t.Errorf("Error = %v, want %v", got.Error, tt.expectRes.Error)
				}
			} else if got.Error != nil {
				t.Errorf("Expected no error in result, got %v", got.Error)
			}
		})
	}
}

func Test_HandleScoreButton(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	mockHelper := utilsmocks.NewMockHelpers(ctrl)
	mockConfig := &config.Config{}
	tracer := noop.NewTracerProvider().Tracer("test")
	mockInteractionStore := storagemocks.NewMockISInterface[any](ctrl)
	mockGuildConfigCache := storagemocks.NewMockISInterface[storage.GuildConfig](ctrl)
	mockGuildCfg := guildconfigmocks.NewMockGuildConfigResolver(ctrl)

	// Pass nil metrics so wrapper doesn't record metrics during tests
	manager := NewScoreRoundManager(mockSession, mockEventBus, logger, mockHelper, mockConfig, mockInteractionStore, mockGuildConfigCache, tracer, nil, mockGuildCfg)
	srm := manager.(*scoreRoundManager)
	// Allow optional guild config lookup without strict expectations
	mockGuildCfg.EXPECT().GetGuildConfigWithContext(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()

	t.Run("single modal path", func(t *testing.T) {
		i := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
			ID:        "id1",
			Type:      discordgo.InteractionMessageComponent,
			GuildID:   "g1",
			ChannelID: "c1",
			Member:    &discordgo.Member{User: &discordgo.User{ID: "u1"}},
			Message:   &discordgo.Message{ID: "m1"},
			Data:      discordgo.MessageComponentInteractionData{CustomID: scoreButtonPrefix + "round-123|extra"},
		}}

		mockSession.EXPECT().InteractionRespond(i.Interaction, gomock.Any()).Return(nil)

		res, err := srm.HandleScoreButton(context.Background(), i)
		if err != nil || res.Error != nil {
			t.Fatalf("unexpected error: %v %v", err, res.Error)
		}
	})

	t.Run("bulk modal with prefill", func(t *testing.T) {
		i := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
			ID:        "id2",
			Type:      discordgo.InteractionMessageComponent,
			GuildID:   "g1",
			ChannelID: "c1",
			Member:    &discordgo.Member{User: &discordgo.User{ID: "u2"}},
			Message: &discordgo.Message{ID: "m2", Embeds: []*discordgo.MessageEmbed{{
				Fields: []*discordgo.MessageEmbedField{{Name: "Final", Value: "Score: +1 (<@123>)"}},
			}}},
			Data: discordgo.MessageComponentInteractionData{CustomID: bulkOverrideButtonPrefix + "round-xyz|extra"},
		}}

		// grant admin override permissions via config role match
		srm.config = &config.Config{Discord: config.DiscordConfig{AdminRoleID: "admin"}}
		i.Member.Roles = []string{"admin"}

		mockSession.EXPECT().InteractionRespond(i.Interaction, gomock.Any()).Return(nil)

		res, err := srm.HandleScoreButton(context.Background(), i)
		if err != nil || res.Error != nil {
			t.Fatalf("unexpected error: %v %v", err, res.Error)
		}
	})

	t.Run("invalid custom id", func(t *testing.T) {
		i := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
			ID:     "id3",
			Type:   discordgo.InteractionMessageComponent,
			Member: &discordgo.Member{User: &discordgo.User{ID: "u3"}},
			// Use a CustomID without the required delimiter to trigger error branch
			Data: discordgo.MessageComponentInteractionData{CustomID: "round_enter_score"},
		}}

		res, err := srm.HandleScoreButton(context.Background(), i)
		if err != nil {
			t.Fatalf("unexpected wrapper error: %v", err)
		}
		if res.Error == nil {
			t.Fatalf("expected error result for invalid custom id")
		}
	})

	t.Run("permission denied for override", func(t *testing.T) {
		i := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
			ID:        "id4",
			Type:      discordgo.InteractionMessageComponent,
			GuildID:   "g1",
			ChannelID: "c1",
			Member:    &discordgo.Member{User: &discordgo.User{ID: "u4"}},
			Data:      discordgo.MessageComponentInteractionData{CustomID: bulkOverrideButtonPrefix + "round-xyz|extra"},
		}}

		mockSession.EXPECT().InteractionRespond(i.Interaction, gomock.Any()).Return(nil)

		res, err := srm.HandleScoreButton(context.Background(), i)
		if err != nil || res.Error != nil {
			t.Fatalf("unexpected error: %v %v", err, res.Error)
		}
		if res.Success == nil {
			t.Fatalf("expected success result indicating permission denied branch handled")
		}
	})
}

func Test_HandleScoreSubmission_Single(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	mockHelper := utilsmocks.NewMockHelpers(ctrl)
	mockConfig := &config.Config{}
	tracer := noop.NewTracerProvider().Tracer("test")
	mockInteractionStore := storagemocks.NewMockISInterface[any](ctrl)
	mockGuildConfigCache := storagemocks.NewMockISInterface[storage.GuildConfig](ctrl)
	mockGuildCfg := guildconfigmocks.NewMockGuildConfigResolver(ctrl)

	// Pass nil metrics so wrapper doesn't record metrics during tests
	manager := NewScoreRoundManager(mockSession, mockEventBus, logger, mockHelper, mockConfig, mockInteractionStore, mockGuildConfigCache, tracer, nil, mockGuildCfg)

	srm := manager.(*scoreRoundManager)
	mockGuildCfg.EXPECT().GetGuildConfigWithContext(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()

	build := func(customID string, components []discordgo.MessageComponent) *discordgo.InteractionCreate {
		return &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
			ID:      "id-sub",
			Type:    discordgo.InteractionModalSubmit,
			GuildID: "g1",
			Member:  &discordgo.Member{User: &discordgo.User{ID: "u1"}},
			Message: &discordgo.Message{ID: "m1"},
			Data: discordgo.ModalSubmitInteractionData{
				CustomID:   customID,
				Components: components,
			},
		}}
	}

	t.Run("invalid custom id format", func(t *testing.T) {
		i := build(submitSingleModalPrefix+"bad", nil)
		// Ephemeral error response expected
		mockSession.EXPECT().InteractionRespond(i.Interaction, gomock.Any()).Return(nil)
		res, err := srm.HandleScoreSubmission(context.Background(), i)
		if err != nil || res.Error == nil {
			t.Fatalf("expected error result, got err=%v res=%v", err, res)
		}
	})

	t.Run("invalid round id", func(t *testing.T) {
		i := build(submitSingleModalPrefix+"not-a-uuid|u1", nil)
		mockSession.EXPECT().InteractionRespond(i.Interaction, gomock.Any()).Return(nil)
		res, err := srm.HandleScoreSubmission(context.Background(), i)
		if err != nil || res.Error == nil {
			t.Fatalf("expected error result for invalid round id")
		}
	})

	t.Run("empty score", func(t *testing.T) {
		i := build(submitSingleModalPrefix+"550e8400-e29b-41d4-a716-446655440000|u1", []discordgo.MessageComponent{
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "score_input", Value: ""}}},
		})
		mockSession.EXPECT().InteractionRespond(i.Interaction, gomock.Any()).Return(nil)
		mockSession.EXPECT().FollowupMessageCreate(i.Interaction, true, gomock.Any()).Return(&discordgo.Message{}, nil)
		res, err := srm.HandleScoreSubmission(context.Background(), i)
		if err != nil || res.Error == nil {
			t.Fatalf("expected error result for empty score")
		}
	})

	t.Run("invalid number", func(t *testing.T) {
		i := build(submitSingleModalPrefix+"550e8400-e29b-41d4-a716-446655440000|u1", []discordgo.MessageComponent{
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "score_input", Value: "abc"}}},
		})
		mockSession.EXPECT().InteractionRespond(i.Interaction, gomock.Any()).Return(nil)
		mockSession.EXPECT().FollowupMessageCreate(i.Interaction, true, gomock.Any()).Return(&discordgo.Message{}, nil)
		res, err := srm.HandleScoreSubmission(context.Background(), i)
		if err != nil || res.Error == nil {
			t.Fatalf("expected error result for invalid number")
		}
	})

	t.Run("out of range", func(t *testing.T) {
		i := build(submitSingleModalPrefix+"550e8400-e29b-41d4-a716-446655440000|u1", []discordgo.MessageComponent{
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "score_input", Value: "1000"}}},
		})
		mockSession.EXPECT().InteractionRespond(i.Interaction, gomock.Any()).Return(nil)
		mockSession.EXPECT().FollowupMessageCreate(i.Interaction, true, gomock.Any()).Return(&discordgo.Message{}, nil)
		res, err := srm.HandleScoreSubmission(context.Background(), i)
		if err != nil || res.Error == nil {
			t.Fatalf("expected error result for out-of-range")
		}
	})

	t.Run("success publish", func(t *testing.T) {
		i := build(submitSingleModalPrefix+"550e8400-e29b-41d4-a716-446655440000|u1", []discordgo.MessageComponent{
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "score_input", Value: "5"}}},
		})
		mockSession.EXPECT().InteractionRespond(i.Interaction, gomock.Any()).Return(nil)
		// After validation it sends a followup on success at end
		mockHelper.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), gomock.Any()).Return(&message.Message{UUID: "x"}, nil)
		mockEventBus.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(nil)
		mockHelper.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), gomock.Any()).Return(&message.Message{UUID: "y"}, nil).AnyTimes()
		mockEventBus.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		mockSession.EXPECT().FollowupMessageCreate(i.Interaction, true, gomock.Any()).Return(&discordgo.Message{}, nil).AnyTimes()
		res, err := srm.HandleScoreSubmission(context.Background(), i)
		if err != nil || res.Error != nil {
			t.Fatalf("unexpected error: %v %v", err, res.Error)
		}
	})
}

func Test_HandleScoreSubmission_Bulk(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	mockHelper := utilsmocks.NewMockHelpers(ctrl)
	mockConfig := &config.Config{}
	tracer := noop.NewTracerProvider().Tracer("test")
	mockInteractionStore := storagemocks.NewMockISInterface[any](ctrl)
	mockGuildConfigCache := storagemocks.NewMockISInterface[storage.GuildConfig](ctrl)
	mockGuildCfg := guildconfigmocks.NewMockGuildConfigResolver(ctrl)

	// Pass nil metrics so wrapper doesn't record metrics during tests
	manager := NewScoreRoundManager(mockSession, mockEventBus, logger, mockHelper, mockConfig, mockInteractionStore, mockGuildConfigCache, tracer, nil, mockGuildCfg)

	srm := manager.(*scoreRoundManager)
	// Allow guild config lookups
	mockGuildCfg.EXPECT().GetGuildConfigWithContext(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()

	build := func(customID string, bulk string, embed *discordgo.MessageEmbed) *discordgo.InteractionCreate {
		comps := []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "bulk_scores_input", Value: bulk}}}}
		return &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
			ID:        "id-bulk",
			Type:      discordgo.InteractionModalSubmit,
			GuildID:   "g1",
			ChannelID: "c1",
			Member:    &discordgo.Member{User: &discordgo.User{ID: "u1"}},
			Message:   &discordgo.Message{ID: "m1", Embeds: []*discordgo.MessageEmbed{embed}},
			Data:      discordgo.ModalSubmitInteractionData{CustomID: customID, Components: comps},
		}}
	}

	t.Run("no updates found summarizes", func(t *testing.T) {
		// Empty bulk input should result in no updates
		i := build(submitBulkOverridePrefix+"550e8400-e29b-41d4-a716-446655440000|u1", "", &discordgo.MessageEmbed{})
		mockSession.EXPECT().InteractionRespond(i.Interaction, gomock.Any()).Return(nil)
		res, err := srm.HandleScoreSubmission(context.Background(), i)
		if err != nil || res.Error != nil {
			t.Fatalf("unexpected error: %v %v", err, res.Error)
		}
	})

	t.Run("publishes updates when present", func(t *testing.T) {
		embed := &discordgo.MessageEmbed{Fields: []*discordgo.MessageEmbedField{{Name: "Final", Value: "Score: +1 (<@123>)"}}}
		// Change 123 from +1 to +2
		i := build(submitBulkOverridePrefix+"550e8400-e29b-41d4-a716-446655440000|u1", "<@123>=+2", embed)

		mockSession.EXPECT().GuildMember(gomock.Any(), gomock.Any()).Return(&discordgo.Member{User: &discordgo.User{ID: "123", Username: "x"}}, nil).AnyTimes()
		mockHelper.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), gomock.Any()).Return(&message.Message{UUID: "bulk"}, nil)
		mockEventBus.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(nil)
		mockSession.EXPECT().InteractionRespond(i.Interaction, gomock.Any()).Return(nil)

		res, err := srm.HandleScoreSubmission(context.Background(), i)
		if err != nil || res.Error != nil {
			t.Fatalf("unexpected error: %v %v", err, res.Error)
		}
	})
}

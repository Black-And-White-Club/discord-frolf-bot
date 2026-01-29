package scoreround

import (
	"context"
	"errors"
	"log/slog"
	"reflect"
	"strings"
	"testing"
	"time"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/testutils"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestNewScoreRoundManager(t *testing.T) {
	fakeSession := discord.NewFakeSession()
	fakeEventBus := &testutils.FakeEventBus{}
	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	fakeHelper := &testutils.FakeHelpers{}
	fakeConfig := &config.Config{}
	mockTracer := noop.NewTracerProvider().Tracer("test")
	fakeMetrics := &testutils.FakeDiscordMetrics{}
	fakeGuildConfigResolver := &testutils.FakeGuildConfigResolver{}
	fakeInteractionStore := &testutils.FakeStorage[any]{}
	fakeGuildConfigCache := &testutils.FakeStorage[storage.GuildConfig]{}

	manager := NewScoreRoundManager(fakeSession, fakeEventBus, logger, fakeHelper, fakeConfig, fakeInteractionStore, fakeGuildConfigCache, mockTracer, fakeMetrics, fakeGuildConfigResolver)
	impl, ok := manager.(*scoreRoundManager)
	if !ok {
		t.Fatalf("Expected *scoreRoundManager, got %T", manager)
	}

	if impl.session != fakeSession {
		t.Error("Expected session to be assigned")
	}
	if impl.publisher != fakeEventBus {
		t.Error("Expected publisher to be assigned")
	}
	if impl.logger != logger {
		t.Error("Expected logger to be assigned")
	}
	if impl.helper != fakeHelper {
		t.Error("Expected helper to be assigned")
	}
	if impl.config != fakeConfig {
		t.Error("Expected config to be assigned")
	}
	if impl.tracer != mockTracer {
		t.Error("Expected tracer to be assigned")
	}
	if impl.metrics != fakeMetrics {
		t.Error("Expected metrics to be assigned")
	}
	if impl.operationWrapper == nil {
		t.Error("Expected operationWrapper to be set")
	}
	if impl.interactionStore != fakeInteractionStore {
		t.Error("Expected interactionStore to be assigned")
	}
	if impl.guildConfigCache != fakeGuildConfigCache {
		t.Error("Expected guildConfigCache to be assigned")
	}
}

func Test_wrapScoreRoundOperation(t *testing.T) {
	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	fakeMetrics := &testutils.FakeDiscordMetrics{}
	tracer := noop.NewTracerProvider().Tracer("test")

	tests := []struct {
		name      string
		operation string
		fn        func(context.Context) (ScoreRoundOperationResult, error)
		expectErr string
		expectRes ScoreRoundOperationResult
		setup     func()
	}{
		{
			name:      "success path",
			operation: "handle_success",
			fn: func(ctx context.Context) (ScoreRoundOperationResult, error) {
				return ScoreRoundOperationResult{Success: "success"}, nil
			},
			expectRes: ScoreRoundOperationResult{Success: "success"},
			setup: func() {
				fakeMetrics.RecordAPIRequestDurationFunc = func(ctx context.Context, operation string, duration time.Duration) {
					if operation != "handle_success" {
						t.Errorf("expected operation handle_success, got %s", operation)
					}
				}
				fakeMetrics.RecordAPIRequestFunc = func(ctx context.Context, operation string) {
					if operation != "handle_success" {
						t.Errorf("expected operation handle_success, got %s", operation)
					}
				}
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
			setup: func() {
				fakeMetrics.RecordAPIRequestDurationFunc = func(ctx context.Context, operation string, duration time.Duration) {
					if operation != "handle_error" {
						t.Errorf("expected operation handle_error, got %s", operation)
					}
				}
				fakeMetrics.RecordAPIErrorFunc = func(ctx context.Context, operation, errorType string) {
					if operation != "handle_error" || errorType != "operation_error" {
						t.Errorf("expected operation handle_error and errorType operation_error, got %s, %s", operation, errorType)
					}
				}
			},
		},
		{
			name:      "result has error",
			operation: "handle_result_error",
			fn: func(ctx context.Context) (ScoreRoundOperationResult, error) {
				return ScoreRoundOperationResult{Error: errors.New("result error")}, nil
			},
			expectRes: ScoreRoundOperationResult{Error: errors.New("result error")},
			setup: func() {
				fakeMetrics.RecordAPIRequestDurationFunc = func(ctx context.Context, operation string, duration time.Duration) {
					if operation != "handle_result_error" {
						t.Errorf("expected operation handle_result_error, got %s", operation)
					}
				}
				fakeMetrics.RecordAPIErrorFunc = func(ctx context.Context, operation, errorType string) {
					if operation != "handle_result_error" || errorType != "result_error" {
						t.Errorf("expected operation handle_result_error and errorType result_error, got %s, %s", operation, errorType)
					}
				}
			},
		},
		{
			name:      "panic recovery",
			operation: "handle_panic",
			fn: func(ctx context.Context) (ScoreRoundOperationResult, error) {
				panic("unexpected panic")
			},
			expectRes: ScoreRoundOperationResult{Error: nil},
			setup: func() {
				fakeMetrics.RecordAPIRequestDurationFunc = func(ctx context.Context, operation string, duration time.Duration) {
					if operation != "handle_panic" {
						t.Errorf("expected operation handle_panic, got %s", operation)
					}
				}
				fakeMetrics.RecordAPIErrorFunc = func(ctx context.Context, operation, errorType string) {
					if operation != "handle_panic" || errorType != "panic" {
						t.Errorf("expected operation handle_panic and errorType panic, got %s, %s", operation, errorType)
					}
				}
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
			if tt.setup != nil {
				tt.setup()
			}
			got, err := wrapScoreRoundOperation(context.Background(), tt.operation, tt.fn, logger, tracer, fakeMetrics)

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
	fakeSession := discord.NewFakeSession()
	fakeEventBus := &testutils.FakeEventBus{}
	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	fakeHelper := &testutils.FakeHelpers{}
	fakeConfig := &config.Config{}
	tracer := noop.NewTracerProvider().Tracer("test")
	fakeInteractionStore := &testutils.FakeStorage[any]{}
	fakeGuildConfigCache := &testutils.FakeStorage[storage.GuildConfig]{}
	fakeGuildCfg := &testutils.FakeGuildConfigResolver{}

	// Pass nil metrics so wrapper doesn't record metrics during tests
	manager := NewScoreRoundManager(fakeSession, fakeEventBus, logger, fakeHelper, fakeConfig, fakeInteractionStore, fakeGuildConfigCache, tracer, nil, fakeGuildCfg)
	srm := manager.(*scoreRoundManager)

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

		fakeSession.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
			return nil
		}

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

		fakeSession.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
			return nil
		}

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

		fakeSession.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
			return nil
		}

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
	fakeSession := discord.NewFakeSession()
	fakeEventBus := &testutils.FakeEventBus{}
	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	fakeHelper := &testutils.FakeHelpers{}
	fakeConfig := &config.Config{}
	tracer := noop.NewTracerProvider().Tracer("test")
	fakeInteractionStore := &testutils.FakeStorage[any]{}
	fakeGuildConfigCache := &testutils.FakeStorage[storage.GuildConfig]{}
	fakeGuildCfg := &testutils.FakeGuildConfigResolver{}

	// Pass nil metrics so wrapper doesn't record metrics during tests
	manager := NewScoreRoundManager(fakeSession, fakeEventBus, logger, fakeHelper, fakeConfig, fakeInteractionStore, fakeGuildConfigCache, tracer, nil, fakeGuildCfg)

	srm := manager.(*scoreRoundManager)

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
		fakeSession.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
			return nil
		}
		res, err := srm.HandleScoreSubmission(context.Background(), i)
		if err != nil || res.Error == nil {
			t.Fatalf("expected error result, got err=%v res=%v", err, res)
		}
	})

	t.Run("invalid round id", func(t *testing.T) {
		i := build(submitSingleModalPrefix+"not-a-uuid|u1", nil)
		fakeSession.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
			return nil
		}
		res, err := srm.HandleScoreSubmission(context.Background(), i)
		if err != nil || res.Error == nil {
			t.Fatalf("expected error result for invalid round id")
		}
	})

	t.Run("empty score", func(t *testing.T) {
		i := build(submitSingleModalPrefix+"550e8400-e29b-41d4-a716-446655440000|u1", []discordgo.MessageComponent{
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "score_input", Value: ""}}},
		})
		fakeSession.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
			return nil
		}
		fakeSession.FollowupMessageCreateFunc = func(i *discordgo.Interaction, wait bool, data *discordgo.WebhookParams, options ...discordgo.RequestOption) (*discordgo.Message, error) {
			return &discordgo.Message{}, nil
		}
		res, err := srm.HandleScoreSubmission(context.Background(), i)
		if err != nil || res.Error == nil {
			t.Fatalf("expected error result for empty score")
		}
	})

	t.Run("invalid number", func(t *testing.T) {
		i := build(submitSingleModalPrefix+"550e8400-e29b-41d4-a716-446655440000|u1", []discordgo.MessageComponent{
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "score_input", Value: "abc"}}},
		})
		fakeSession.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
			return nil
		}
		fakeSession.FollowupMessageCreateFunc = func(i *discordgo.Interaction, wait bool, data *discordgo.WebhookParams, options ...discordgo.RequestOption) (*discordgo.Message, error) {
			return &discordgo.Message{}, nil
		}
		res, err := srm.HandleScoreSubmission(context.Background(), i)
		if err != nil || res.Error == nil {
			t.Fatalf("expected error result for invalid number")
		}
	})

	t.Run("out of range", func(t *testing.T) {
		i := build(submitSingleModalPrefix+"550e8400-e29b-41d4-a716-446655440000|u1", []discordgo.MessageComponent{
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "score_input", Value: "1000"}}},
		})
		fakeSession.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
			return nil
		}
		fakeSession.FollowupMessageCreateFunc = func(i *discordgo.Interaction, wait bool, data *discordgo.WebhookParams, options ...discordgo.RequestOption) (*discordgo.Message, error) {
			return &discordgo.Message{}, nil
		}
		res, err := srm.HandleScoreSubmission(context.Background(), i)
		if err != nil || res.Error == nil {
			t.Fatalf("expected error result for out-of-range")
		}
	})

	t.Run("success publish", func(t *testing.T) {
		i := build(submitSingleModalPrefix+"550e8400-e29b-41d4-a716-446655440000|u1", []discordgo.MessageComponent{
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "score_input", Value: "5"}}},
		})
		fakeSession.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
			return nil
		}
		// After validation it sends a followup on success at end
		fakeHelper.CreateResultMessageFunc = func(originalMsg *message.Message, payload interface{}, topic string) (*message.Message, error) {
			return &message.Message{UUID: "x"}, nil
		}
		fakeEventBus.PublishFunc = func(topic string, messages ...*message.Message) error {
			return nil
		}
		fakeSession.FollowupMessageCreateFunc = func(i *discordgo.Interaction, wait bool, data *discordgo.WebhookParams, options ...discordgo.RequestOption) (*discordgo.Message, error) {
			return &discordgo.Message{}, nil
		}
		res, err := srm.HandleScoreSubmission(context.Background(), i)
		if err != nil || res.Error != nil {
			t.Fatalf("unexpected error: %v %v", err, res.Error)
		}
	})
}

func Test_HandleScoreSubmission_Bulk(t *testing.T) {
	fakeSession := discord.NewFakeSession()
	fakeEventBus := &testutils.FakeEventBus{}
	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	fakeHelper := &testutils.FakeHelpers{}
	fakeConfig := &config.Config{}
	tracer := noop.NewTracerProvider().Tracer("test")
	fakeInteractionStore := &testutils.FakeStorage[any]{}
	fakeGuildConfigCache := &testutils.FakeStorage[storage.GuildConfig]{}
	fakeGuildCfg := &testutils.FakeGuildConfigResolver{}

	// Pass nil metrics so wrapper doesn't record metrics during tests
	manager := NewScoreRoundManager(fakeSession, fakeEventBus, logger, fakeHelper, fakeConfig, fakeInteractionStore, fakeGuildConfigCache, tracer, nil, fakeGuildCfg)

	srm := manager.(*scoreRoundManager)

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
		fakeSession.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
			return nil
		}
		res, err := srm.HandleScoreSubmission(context.Background(), i)
		if err != nil || res.Error != nil {
			t.Fatalf("unexpected error: %v %v", err, res.Error)
		}
	})

	t.Run("publishes updates when present", func(t *testing.T) {
		embed := &discordgo.MessageEmbed{Fields: []*discordgo.MessageEmbedField{{Name: "Final", Value: "Score: +1 (<@123>)"}}}
		// Change 123 from +1 to +2
		i := build(submitBulkOverridePrefix+"550e8400-e29b-41d4-a716-446655440000|u1", "<@123>=+2", embed)

		fakeSession.GuildMemberFunc = func(guildID, userID string, options ...discordgo.RequestOption) (*discordgo.Member, error) {
			return &discordgo.Member{User: &discordgo.User{ID: "123", Username: "x"}}, nil
		}
		fakeHelper.CreateResultMessageFunc = func(originalMsg *message.Message, payload interface{}, topic string) (*message.Message, error) {
			return &message.Message{UUID: "bulk"}, nil
		}
		fakeEventBus.PublishFunc = func(topic string, messages ...*message.Message) error {
			return nil
		}
		fakeSession.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
			return nil
		}

		res, err := srm.HandleScoreSubmission(context.Background(), i)
		if err != nil || res.Error != nil {
			t.Fatalf("unexpected error: %v %v", err, res.Error)
		}
	})
}

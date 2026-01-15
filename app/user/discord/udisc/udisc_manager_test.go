package udisc

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot-shared/eventbus/mocks"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestNewUDiscManager_defaultsTracerWhenNil(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))

	mgr := NewUDiscManager(
		discordmocks.NewMockSession(ctrl),
		eventbusmocks.NewMockEventBus(ctrl),
		logger,
		nil, // cfg
		nil, // interactionStore
		nil, // guildConfigCache
		nil, // tracer
		nil, // metrics
	)

	impl, ok := mgr.(*udiscManager)
	if !ok {
		t.Fatalf("expected concrete type *udiscManager")
	}
	if impl.tracer == nil {
		t.Fatalf("expected tracer to be non-nil")
	}
	if impl.operationWrapper == nil {
		t.Fatalf("expected operationWrapper to be configured")
	}
}

func TestOperationWrapper_successAndErrorBranches(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))
	tracer := noop.NewTracerProvider().Tracer("test")

	// Success path
	if _, err := operationWrapper(context.Background(), "op", func(ctx context.Context) (UDiscOperationResult, error) {
		return UDiscOperationResult{Success: "ok"}, nil
	}, logger, tracer); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	// Error path
	wantErr := errors.New("boom")
	if _, err := operationWrapper(context.Background(), "op", func(ctx context.Context) (UDiscOperationResult, error) {
		return UDiscOperationResult{Error: wantErr}, wantErr
	}, logger, tracer); err == nil {
		t.Fatalf("expected error")
	}
}

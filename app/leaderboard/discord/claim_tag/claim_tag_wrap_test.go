package claimtag

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
)

func Test_wrapClaimTagOperation_Variants(t *testing.T) {
	logger := slogDiscard()
	tracer := otel.Tracer("test")
	var metrics noMetrics
	ctx := context.Background()

	// nil function
	if res, err := wrapClaimTagOperation(ctx, "nil_fn", nil, logger, tracer, metrics); err != nil || res.Error == nil {
		t.Fatalf("expected result.Error for nil fn, got err=%v res=%v", err, res)
	}

	// fn returns error
	_, err := wrapClaimTagOperation(ctx, "fn_error", func(ctx context.Context) (ClaimTagOperationResult, error) {
		return ClaimTagOperationResult{}, errors.New("boom")
	}, logger, tracer, metrics)
	if err == nil {
		t.Fatalf("expected wrapped error from fn")
	}

	// result has error
	res, err := wrapClaimTagOperation(ctx, "result_error", func(ctx context.Context) (ClaimTagOperationResult, error) {
		return ClaimTagOperationResult{Error: errors.New("bad")}, nil
	}, logger, tracer, metrics)
	if err != nil || res.Error == nil {
		t.Fatalf("expected result.Error without outer error")
	}

	// panic recovery
	_, err = wrapClaimTagOperation(ctx, "panic", func(ctx context.Context) (ClaimTagOperationResult, error) {
		panic("kaboom")
	}, logger, tracer, metrics)
	if err == nil {
		t.Fatalf("expected panic to be recovered with error")
	}

	// success
	res, err = wrapClaimTagOperation(ctx, "success", func(ctx context.Context) (ClaimTagOperationResult, error) {
		return ClaimTagOperationResult{Success: true}, nil
	}, logger, tracer, metrics)
	if err != nil || res.Error != nil || res.Success != true {
		t.Fatalf("expected success path, got res=%v err=%v", res, err)
	}
}

// helpers for tests
type noMetrics struct{}

func (noMetrics) RecordAPIRequestDuration(ctx context.Context, endpoint string, duration time.Duration) {
}
func (noMetrics) RecordAPIRequest(ctx context.Context, endpoint string)                         {}
func (noMetrics) RecordAPIError(ctx context.Context, endpoint string, errorType string)         {}
func (noMetrics) RecordRateLimit(ctx context.Context, endpoint string, resetTime time.Duration) {}
func (noMetrics) RecordWebsocketEvent(ctx context.Context, eventType string)                    {}
func (noMetrics) RecordWebsocketReconnect(ctx context.Context)                                  {}
func (noMetrics) RecordWebsocketDisconnect(ctx context.Context, reason string)                  {}
func (noMetrics) RecordHandlerAttempt(ctx context.Context, handlerName string)                  {}
func (noMetrics) RecordHandlerSuccess(ctx context.Context, handlerName string)                  {}
func (noMetrics) RecordHandlerFailure(ctx context.Context, handlerName string)                  {}
func (noMetrics) RecordHandlerDuration(ctx context.Context, handlerName string, duration time.Duration) {
}

func slogDiscard() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

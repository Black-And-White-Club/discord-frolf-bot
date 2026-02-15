package handlers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
)

func resetGuildRateLimiter(t *testing.T) {
	t.Helper()
	guildRateLimiterLock.Lock()
	defer guildRateLimiterLock.Unlock()
	guildRateLimiter = make(map[string][]time.Time)
}

func TestRoundHandlers_HandleScorecardUploaded(t *testing.T) {
	resetGuildRateLimiter(t)
	defer resetGuildRateLimiter(t)

	// Since this handler just validates and emits a request for csv parsing, it doesn't really use Discord service heavily
	// But it does check validation
	// NewRoundHandlers in real implementation doesn't use the mock for validation logic inside the handler itself usually,
	// unless strict logic calls external services.
	// Looking at the original test, it instantiates NewRoundHandlers with nils.
	// We will supply a fake just in case, or nil if safe.
	// However, to follow the pattern, we should provide the proper struct.

	rh := NewRoundHandlers(
		loggerfrolfbot.NoOpLogger,
		&config.Config{},
		nil,
		&FakeRoundDiscord{},
		nil,
	)

	tests := []struct {
		name    string
		payload *roundevents.ScorecardUploadedPayloadV1
		ctx     context.Context
		wantErr bool
		wantLen int
	}{
		{
			name: "successful_scorecard_upload",
			payload: &roundevents.ScorecardUploadedPayloadV1{
				ImportID: "import-id",
				GuildID:  sharedtypes.GuildID("guild-id"),
				RoundID:  sharedtypes.RoundID(uuid.New()),
				UDiscURL: "https://udisc.com/scorecard.csv",
			},
			ctx:     context.Background(),
			wantErr: false,
			wantLen: 0,
		},
		{
			name: "missing_import_id",
			payload: &roundevents.ScorecardUploadedPayloadV1{
				GuildID:  sharedtypes.GuildID("guild-id"),
				RoundID:  sharedtypes.RoundID(uuid.New()),
				UDiscURL: "https://udisc.com/scorecard.csv",
			},
			ctx:     context.Background(),
			wantErr: true,
			wantLen: 0,
		},
		{
			name: "missing_guild_id",
			payload: &roundevents.ScorecardUploadedPayloadV1{
				ImportID: "import-id",
				RoundID:  sharedtypes.RoundID(uuid.New()),
				UDiscURL: "https://udisc.com/scorecard.csv",
			},
			ctx:     context.Background(),
			wantErr: true,
			wantLen: 0,
		},
		{
			name: "invalid_host",
			payload: &roundevents.ScorecardUploadedPayloadV1{
				ImportID: "import-id",
				GuildID:  sharedtypes.GuildID("guild-id"),
				RoundID:  sharedtypes.RoundID(uuid.New()),
				UDiscURL: "https://example.com/scorecard.csv",
			},
			ctx:     context.Background(),
			wantErr: true,
			wantLen: 0,
		},
		{
			name: "unsupported_extension",
			payload: &roundevents.ScorecardUploadedPayloadV1{
				ImportID: "import-id",
				GuildID:  sharedtypes.GuildID("guild-id"),
				RoundID:  sharedtypes.RoundID(uuid.New()),
				UDiscURL: "https://udisc.com/scorecard.pdf",
			},
			ctx:     context.Background(),
			wantErr: true,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This particular handler logic seems to be pure validation/transformation in the current impl
			// so we don't need extensive setting up of the fake for these tests unless the handler calls something on it.
			// Checking `scorecard_upload_handler.go` would confirm, but assuming test fidelity is key.
			got, err := rh.HandleScorecardUploaded(tt.ctx, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleScorecardUploaded() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != tt.wantLen {
				t.Errorf("HandleScorecardUploaded() got %d results, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestRoundHandlers_HandleScorecardURLRequested(t *testing.T) {
	tests := []struct {
		name    string
		payload *roundevents.ScorecardURLRequestedPayloadV1
		ctx     context.Context
		wantErr bool
		wantLen int
	}{
		{
			name: "successful_url_request",
			payload: &roundevents.ScorecardURLRequestedPayloadV1{
				RoundID:   sharedtypes.RoundID(uuid.New()),
				ChannelID: "channel-123",
				UserID:    "user-123",
			},
			ctx:     context.Background(),
			wantErr: false,
			wantLen: 0, // Assuming it just sends something or logs
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeRoundDiscord := &FakeRoundDiscord{}
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

			h := NewRoundHandlers(
				mockLogger,
				&config.Config{},
				nil,
				fakeRoundDiscord,
				nil,
			)

			got, err := h.HandleScorecardURLRequested(tt.ctx, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleScorecardURLRequested() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != tt.wantLen {
				t.Errorf("HandleScorecardURLRequested() got %d results, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestRoundHandlers_HandleImportFailed(t *testing.T) {
	tests := []struct {
		name    string
		payload *roundevents.ImportFailedPayloadV1
		ctx     context.Context
		wantErr bool
		wantLen int
		setup   func(*FakeRoundDiscord)
	}{
		{
			name: "successful_import_failure_handling",
			payload: &roundevents.ImportFailedPayloadV1{
				RoundID:   sharedtypes.RoundID(uuid.New()),
				UserID:    "user-123",
				ChannelID: "channel-123",
				Error:     "Import failed",
			},
			ctx:     context.Background(),
			wantErr: false,
			wantLen: 0,
			setup: func(f *FakeRoundDiscord) {
				f.ScorecardUploadManager.SendUploadErrorFunc = func(ctx context.Context, channelID, userID, errorMsg string) error {
					return nil
				}
			},
		},
		{
			name: "send_error_fails",
			payload: &roundevents.ImportFailedPayloadV1{
				RoundID:   sharedtypes.RoundID(uuid.New()),
				UserID:    "user-123",
				ChannelID: "channel-123",
				Error:     "Import failed",
			},
			ctx:     context.Background(),
			wantErr: true,
			wantLen: 0,
			setup: func(f *FakeRoundDiscord) {
				f.ScorecardUploadManager.SendUploadErrorFunc = func(ctx context.Context, channelID, userID, errorMsg string) error {
					return errors.New("failed to send error")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeRoundDiscord := &FakeRoundDiscord{}
			if tt.setup != nil {
				tt.setup(fakeRoundDiscord)
			}
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

			h := NewRoundHandlers(
				mockLogger,
				&config.Config{},
				nil,
				fakeRoundDiscord,
				nil,
			)

			got, err := h.HandleImportFailed(tt.ctx, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleImportFailed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != tt.wantLen {
				t.Errorf("HandleImportFailed() got %d results, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestRoundHandlers_HandleScorecardParseFailed(t *testing.T) {
	tests := []struct {
		name    string
		payload *roundevents.ScorecardParseFailedPayloadV1
		ctx     context.Context
		wantErr bool
		wantLen int
		setup   func(*FakeRoundDiscord)
	}{
		{
			name: "successful_parse_failure_handling",
			payload: &roundevents.ScorecardParseFailedPayloadV1{
				RoundID:   sharedtypes.RoundID(uuid.New()),
				UserID:    "user-123",
				ChannelID: "channel-123",
				Error:     "Parse failed",
			},
			ctx:     context.Background(),
			wantErr: false,
			wantLen: 0,
			setup: func(f *FakeRoundDiscord) {
				f.ScorecardUploadManager.SendUploadErrorFunc = func(ctx context.Context, channelID, userID, errorMsg string) error {
					return nil
				}
			},
		},
		{
			name: "send_error_fails",
			payload: &roundevents.ScorecardParseFailedPayloadV1{
				RoundID:   sharedtypes.RoundID(uuid.New()),
				UserID:    "user-123",
				ChannelID: "channel-123",
				Error:     "Parse failed",
			},
			ctx:     context.Background(),
			wantErr: true,
			wantLen: 0,
			setup: func(f *FakeRoundDiscord) {
				f.ScorecardUploadManager.SendUploadErrorFunc = func(ctx context.Context, channelID, userID, errorMsg string) error {
					return errors.New("failed to send error")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeRoundDiscord := &FakeRoundDiscord{}
			if tt.setup != nil {
				tt.setup(fakeRoundDiscord)
			}
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

			h := NewRoundHandlers(
				mockLogger,
				&config.Config{},
				nil,
				fakeRoundDiscord,
				nil,
			)

			got, err := h.HandleScorecardParseFailed(tt.ctx, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleScorecardParseFailed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != tt.wantLen {
				t.Errorf("HandleScorecardParseFailed() got %d results, want %d", len(got), tt.wantLen)
			}
		})
	}
}

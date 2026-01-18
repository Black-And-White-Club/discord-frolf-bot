package roundhandlers

import (
	"context"
	"testing"
	"time"

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

	rh := NewRoundHandlers(
		loggerfrolfbot.NoOpLogger,
		nil,
		nil,
		nil,
		nil,
	)

	tests := []struct {
		name       string
		payload    *roundevents.ScorecardUploadedPayloadV1
		ctx        context.Context
		wantErr    bool
		wantLen    int
	}{
		{
			name: "successful_scorecard_upload",
			payload: &roundevents.ScorecardUploadedPayloadV1{
				ImportID: "import-id",
				GuildID:  sharedtypes.GuildID("guild-id"),
				RoundID:  sharedtypes.RoundID(uuid.New()),
				UDiscURL: "https://example.com/scorecard.csv",
			},
			ctx:     context.Background(),
			wantErr: false,
			wantLen: 0,
		},
		{
			name: "missing_import_id",
			payload: &roundevents.ScorecardUploadedPayloadV1{
				GuildID: sharedtypes.GuildID("guild-id"),
				RoundID: sharedtypes.RoundID(uuid.New()),
				UDiscURL: "https://example.com/scorecard.csv",
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
				UDiscURL: "https://example.com/scorecard.pdf",
			},
			ctx:     context.Background(),
			wantErr: true,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := rh.HandleScorecardUploaded(tt.ctx, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleScorecardUploaded() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != tt.wantLen {
				t.Errorf("HandleScorecardUploaded() got %d results, want %d", len(got), tt.wantLen)
				return
			}
		})
	}
}

func TestRoundHandlers_HandleScorecardParseFailed(t *testing.T) {
	rh := NewRoundHandlers(
		loggerfrolfbot.NoOpLogger,
		nil,
		nil,
		nil,
		nil,
	)

	tests := []struct {
		name       string
		payload    *roundevents.ScorecardParseFailedPayloadV1
		ctx        context.Context
		wantErr    bool
		wantLen    int
	}{
		{
			name: "successful_parse_failure_handling",
			payload: &roundevents.ScorecardParseFailedPayloadV1{
				ImportID: "import-id",
				GuildID:  sharedtypes.GuildID("guild-id"),
				Error:    "parse error",
			},
			ctx:     context.Background(),
			wantErr: false,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := rh.HandleScorecardParseFailed(tt.ctx, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleScorecardParseFailed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != tt.wantLen {
				t.Errorf("HandleScorecardParseFailed() got %d results, want %d", len(got), tt.wantLen)
				return
			}
		})
	}
}

func TestRoundHandlers_HandleImportFailed(t *testing.T) {
	rh := NewRoundHandlers(
		loggerfrolfbot.NoOpLogger,
		nil,
		nil,
		nil,
		nil,
	)

	tests := []struct {
		name       string
		payload    *roundevents.ImportFailedPayloadV1
		ctx        context.Context
		wantErr    bool
		wantLen    int
	}{
		{
			name: "successful_import_failure_handling",
			payload: &roundevents.ImportFailedPayloadV1{
				ImportID: "import-id",
				GuildID:  sharedtypes.GuildID("guild-id"),
				Error:    "import error",
			},
			ctx:     context.Background(),
			wantErr: false,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := rh.HandleImportFailed(tt.ctx, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleImportFailed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != tt.wantLen {
				t.Errorf("HandleImportFailed() got %d results, want %d", len(got), tt.wantLen)
				return
			}
		})
	}
}

func TestRoundHandlers_HandleScorecardURLRequested(t *testing.T) {
	rh := NewRoundHandlers(
		loggerfrolfbot.NoOpLogger,
		nil,
		nil,
		nil,
		nil,
	)

	tests := []struct {
		name       string
		payload    *roundevents.ScorecardURLRequestedPayloadV1
		ctx        context.Context
		wantErr    bool
		wantLen    int
	}{
		{
			name: "successful_scorecard_url_request",
			payload: &roundevents.ScorecardURLRequestedPayloadV1{
				ImportID: "import-id",
				GuildID:  sharedtypes.GuildID("guild-id"),
				UserID:   sharedtypes.DiscordID("user-id"),
			},
			ctx:     context.Background(),
			wantErr: false,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := rh.HandleScorecardURLRequested(tt.ctx, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleScorecardURLRequested() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != tt.wantLen {
				t.Errorf("HandleScorecardURLRequested() got %d results, want %d", len(got), tt.wantLen)
				return
			}
		})
	}
}

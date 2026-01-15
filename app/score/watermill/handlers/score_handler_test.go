package scorehandlers

import (
	"context"
	"testing"

	discordscoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/score"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
)

func TestHandleScoreUpdateRequestTyped(t *testing.T) {
	tagNum := sharedtypes.TagNumber(1)

	tests := []struct {
		name    string
		payload *discordscoreevents.ScoreUpdateRequestDiscordPayloadV1
		wantErr bool
		wantLen int
	}{
		{
			name: "valid_payload_produces_backend_request",
			payload: &discordscoreevents.ScoreUpdateRequestDiscordPayloadV1{
				GuildID:   "guild-1",
				RoundID:   sharedtypes.RoundID(uuid.New()),
				UserID:    sharedtypes.DiscordID("user-1"),
				Score:     sharedtypes.Score(3),
				TagNumber: &tagNum,
				ChannelID: "chan-1",
				MessageID: "msg-1",
			},
			wantErr: false,
			wantLen: 1,
		},
		{
			name:    "nil_payload_returns_error",
			payload: nil,
			wantErr: true,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sh := &ScoreHandlers{Logger: loggerfrolfbot.NoOpLogger}
			got, err := sh.HandleScoreUpdateRequestTyped(context.Background(), tt.payload)
			if (err != nil) != tt.wantErr {
				t.Fatalf("unexpected error state: %v", err)
			}
			if err != nil {
				return
			}
			if len(got) != tt.wantLen {
				t.Fatalf("expected %d results, got %d", tt.wantLen, len(got))
			}

			// Additional checks for the valid case
			if tt.wantLen > 0 {
				res := got[0]
				if res.Topic != sharedevents.ScoreUpdateRequestedV1 {
					t.Fatalf("unexpected topic: %s", res.Topic)
				}
				p, ok := res.Payload.(sharedevents.ScoreUpdateRequestedPayloadV1)
				if !ok {
					t.Fatalf("unexpected payload type: %T", res.Payload)
				}
				if p.RoundID != tt.payload.RoundID || p.UserID != tt.payload.UserID || p.Score != tt.payload.Score {
					t.Fatalf("payload fields mismatch: got %+v", p)
				}
				if res.Metadata["user_id"] != string(tt.payload.UserID) || res.Metadata["channel_id"] != tt.payload.ChannelID || res.Metadata["message_id"] != tt.payload.MessageID {
					t.Fatalf("metadata mismatch: %v", res.Metadata)
				}
			}
		})
	}
}

func TestHandleScoreUpdateSuccessTyped(t *testing.T) {
	tests := []struct {
		name    string
		payload *sharedevents.ScoreUpdatedPayloadV1
		wantErr bool
		wantLen int
	}{
		{
			name: "success_produces_discord_response",
			payload: &sharedevents.ScoreUpdatedPayloadV1{
				RoundID: sharedtypes.RoundID(uuid.New()),
				Score:   sharedtypes.Score(2),
			},
			wantErr: false,
			wantLen: 1,
		},
		{
			name:    "nil_payload_returns_error",
			payload: nil,
			wantErr: true,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sh := &ScoreHandlers{Logger: loggerfrolfbot.NoOpLogger}
			got, err := sh.HandleScoreUpdateSuccessTyped(context.Background(), tt.payload)
			if (err != nil) != tt.wantErr {
				t.Fatalf("unexpected error state: %v", err)
			}
			if err != nil {
				return
			}
			if len(got) != tt.wantLen {
				t.Fatalf("expected %d results, got %d", tt.wantLen, len(got))
			}

			if tt.wantLen > 0 {
				res := got[0]
				if res.Topic != discordscoreevents.ScoreUpdateResponseDiscordV1 {
					t.Fatalf("unexpected topic: %s", res.Topic)
				}
				m, ok := res.Payload.(map[string]interface{})
				if !ok {
					t.Fatalf("unexpected payload type: %T", res.Payload)
				}
				if m["type"] != "score_update_success" || m["round_id"] != tt.payload.RoundID || m["score"] != tt.payload.Score {
					t.Fatalf("response payload mismatch: %v", m)
				}
			}
		})
	}
}

func TestHandleScoreUpdateFailureTyped(t *testing.T) {
	tests := []struct {
		name    string
		payload *sharedevents.ScoreUpdateFailedPayloadV1
		wantErr bool
		wantLen int
	}{
		{
			name: "suppressed_known_business_failure",
			payload: &sharedevents.ScoreUpdateFailedPayloadV1{
				Reason:  "score record not found for aggregate",
				RoundID: sharedtypes.RoundID(uuid.New()),
				GuildID: "guild-1",
				UserID:  sharedtypes.DiscordID("user-1"),
			},
			wantErr: false,
			wantLen: 0,
		},
		{
			name: "other_failure_produces_discord_error",
			payload: &sharedevents.ScoreUpdateFailedPayloadV1{
				Reason:  "database timeout",
				RoundID: sharedtypes.RoundID(uuid.New()),
			},
			wantErr: false,
			wantLen: 1,
		},
		{
			name:    "nil_payload_returns_error",
			payload: nil,
			wantErr: true,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sh := &ScoreHandlers{Logger: loggerfrolfbot.NoOpLogger}
			got, err := sh.HandleScoreUpdateFailureTyped(context.Background(), tt.payload)
			if (err != nil) != tt.wantErr {
				t.Fatalf("unexpected error state: %v", err)
			}
			if err != nil {
				return
			}

			if tt.wantLen == 0 {
				if len(got) != 0 {
					t.Fatalf("expected no results, got: %+v", got)
				}
				return
			}

			if len(got) != tt.wantLen {
				t.Fatalf("expected %d results, got %d", tt.wantLen, len(got))
			}

			res := got[0]
			if res.Topic != discordscoreevents.ScoreUpdateFailedDiscordV1 {
				t.Fatalf("unexpected topic: %s", res.Topic)
			}
			m, ok := res.Payload.(map[string]interface{})
			if !ok {
				t.Fatalf("unexpected payload type: %T", res.Payload)
			}
			if m["error"] != tt.payload.Reason {
				t.Fatalf("expected error %q in payload, got %v", tt.payload.Reason, m)
			}
		})
	}
}

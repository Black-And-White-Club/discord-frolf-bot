package handlers

import (
	"context"
	"testing"

	embedpagination "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/embed_pagination"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

func TestRoundHandlers_HandleRoundCompleted(t *testing.T) {
	__codexTDCases := []struct {
		name string
	}{
		{name: "default"},
	}

	for _, __codexTDCase := range __codexTDCases {
		t.Run(__codexTDCase.name, func(t *testing.T) {
			t.Parallel()

			h := &RoundHandlers{logger: loggerfrolfbot.NoOpLogger}

			t.Run("clears by payload round data message id", func(t *testing.T) {
				t.Parallel()
				messageID := "msg-round-complete-1"

				embedpagination.Set(&embedpagination.Snapshot{
					MessageID: messageID,
					Kind:      embedpagination.SnapshotKindLines,
				})

				payload := &roundevents.RoundCompletedPayloadV1{
					RoundID: sharedtypes.RoundID{},
					RoundData: roundtypes.Round{
						EventMessageID: messageID,
					},
				}

				_, err := h.HandleRoundCompleted(context.Background(), payload)
				if err != nil {
					t.Fatalf("HandleRoundCompleted() error = %v, want nil", err)
				}

				if _, found := embedpagination.Get(messageID); found {
					t.Fatalf("expected snapshot for %q to be deleted", messageID)
				}
			})

			t.Run("falls back to metadata message id", func(t *testing.T) {
				t.Parallel()
				messageID := "msg-round-complete-2"

				embedpagination.Set(&embedpagination.Snapshot{
					MessageID: messageID,
					Kind:      embedpagination.SnapshotKindLines,
				})

				payload := &roundevents.RoundCompletedPayloadV1{}
				ctx := context.WithValue(context.Background(), "discord_message_id", messageID)

				_, err := h.HandleRoundCompleted(ctx, payload)
				if err != nil {
					t.Fatalf("HandleRoundCompleted() error = %v, want nil", err)
				}

				if _, found := embedpagination.Get(messageID); found {
					t.Fatalf("expected snapshot for %q to be deleted", messageID)
				}
			})
		})
	}
}

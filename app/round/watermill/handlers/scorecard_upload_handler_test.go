package roundhandlers

import (
	"encoding/json"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
)

func resetGuildRateLimiter(t *testing.T) {
	t.Helper()
	guildRateLimiterLock.Lock()
	defer guildRateLimiterLock.Unlock()
	guildRateLimiter = make(map[string][]time.Time)
}

func TestRoundHandlers_ScorecardUploadedEvent(t *testing.T) {
	resetGuildRateLimiter(t)
	defer resetGuildRateLimiter(t)

	rh := &RoundHandlers{
		Logger: loggerfrolfbot.NoOpLogger,
		Tracer: noop.NewTracerProvider().Tracer("test"),
	}

	t.Run("payload too large", func(t *testing.T) {
		big := make([]byte, maxScorecardPayloadBytes+1)
		msg := message.NewMessage("id", big)

		err := rh.ScorecardUploadedEvent(msg)
		if err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		msg := message.NewMessage("id", []byte("not-json"))
		err := rh.ScorecardUploadedEvent(msg)
		if err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("missing identifiers", func(t *testing.T) {
		payload := roundevents.ScorecardUploadedPayloadV1{}
		b, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		msg := message.NewMessage("id", b)
		err = rh.ScorecardUploadedEvent(msg)
		if err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("unsupported extension", func(t *testing.T) {
		roundID := uuid.New()
		payload := roundevents.ScorecardUploadedPayloadV1{
			ImportID: "import-id",
			GuildID:  sharedtypes.GuildID("guild-id"),
			RoundID:  sharedtypes.RoundID(roundID),
			UDiscURL: "https://example.com/scorecard.pdf",
		}
		b, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		msg := message.NewMessage("id", b)
		err = rh.ScorecardUploadedEvent(msg)
		if err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("rate limit exceeded", func(t *testing.T) {
		guildID := sharedtypes.GuildID("guild-id")
		now := time.Now()

		guildRateLimiterLock.Lock()
		guildRateLimiter[string(guildID)] = make([]time.Time, maxEventsPerGuildPerMinute)
		for i := 0; i < maxEventsPerGuildPerMinute; i++ {
			guildRateLimiter[string(guildID)][i] = now
		}
		guildRateLimiterLock.Unlock()

		roundID := uuid.New()
		payload := roundevents.ScorecardUploadedPayloadV1{
			ImportID: "import-id",
			GuildID:  guildID,
			RoundID:  sharedtypes.RoundID(roundID),
			UDiscURL: "",
		}
		b, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		msg := message.NewMessage("id", b)
		err = rh.ScorecardUploadedEvent(msg)
		if err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("success", func(t *testing.T) {
		resetGuildRateLimiter(t)

		roundID := uuid.New()
		payload := roundevents.ScorecardUploadedPayloadV1{
			ImportID: "import-id",
			GuildID:  sharedtypes.GuildID("guild-id"),
			RoundID:  sharedtypes.RoundID(roundID),
		}
		b, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		msg := message.NewMessage("id", b)
		err = rh.ScorecardUploadedEvent(msg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestRoundHandlers_ScorecardAuxEvents_InvalidJSON(t *testing.T) {
	rh := &RoundHandlers{
		Logger: loggerfrolfbot.NoOpLogger,
		Tracer: noop.NewTracerProvider().Tracer("test"),
	}

	t.Run("ScorecardParseFailedEvent", func(t *testing.T) {
		msg := message.NewMessage("id", []byte("not-json"))
		err := rh.ScorecardParseFailedEvent(msg)
		if err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("ImportFailedEvent", func(t *testing.T) {
		msg := message.NewMessage("id", []byte("not-json"))
		err := rh.ImportFailedEvent(msg)
		if err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("ScorecardURLRequestedEvent", func(t *testing.T) {
		msg := message.NewMessage("id", []byte("not-json"))
		err := rh.ScorecardURLRequestedEvent(msg)
		if err == nil {
			t.Fatalf("expected error")
		}
	})
}

package rounddiscord

import (
	"context"
	"testing"

	discordgo "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"go.opentelemetry.io/otel"
	"go.uber.org/mock/gomock"
)

// Minimal nil-safe construction test; should not invoke external calls.
func TestNewRoundDiscord_ConstructsAndExposesManagers(t *testing.T) {
	ctx := context.Background()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var session discordgo.Session = discordmocks.NewMockSession(ctrl)
	var publisher eventbus.EventBus = nil
	var helper utils.Helpers = nil
	cfg := &config.Config{}
	var store storage.ISInterface = nil
	tracer := otel.Tracer("test")
	var metrics discordmetrics.DiscordMetrics = nil
	var resolver guildconfig.GuildConfigResolver = nil

	rd, err := NewRoundDiscord(ctx, session, publisher, nil, helper, cfg, store, tracer, metrics, resolver)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rd == nil {
		t.Fatalf("expected non-nil RoundDiscordInterface")
	}

	if rd.GetCreateRoundManager() == nil {
		t.Fatalf("expected non-nil CreateRoundManager")
	}
	if rd.GetRoundRsvpManager() == nil {
		t.Fatalf("expected non-nil RoundRsvpManager")
	}
	if rd.GetRoundReminderManager() == nil {
		t.Fatalf("expected non-nil RoundReminderManager")
	}
	if rd.GetStartRoundManager() == nil {
		t.Fatalf("expected non-nil StartRoundManager")
	}
	if rd.GetScoreRoundManager() == nil {
		t.Fatalf("expected non-nil ScoreRoundManager")
	}
	if rd.GetFinalizeRoundManager() == nil {
		t.Fatalf("expected non-nil FinalizeRoundManager")
	}
	if rd.GetDeleteRoundManager() == nil {
		t.Fatalf("expected non-nil DeleteRoundManager")
	}
	if rd.GetUpdateRoundManager() == nil {
		t.Fatalf("expected non-nil UpdateRoundManager")
	}
	if rd.GetTagUpdateManager() == nil {
		t.Fatalf("expected non-nil TagUpdateManager")
	}
}

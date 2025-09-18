package leaderboarddiscord

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

// Minimal nil-safe stubs; NewLeaderboardDiscord shouldn't invoke any methods during construction
func TestNewLeaderboardDiscord_ConstructsAndExposesManagers(t *testing.T) {
	ctx := context.Background()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	var session discordgo.Session = discordmocks.NewMockSession(ctrl)
	var publisher eventbus.EventBus = nil
	var helper utils.Helpers = nil
	cfg := &config.Config{}
	var resolver guildconfig.GuildConfigResolver = nil
	var store storage.ISInterface = nil
	tracer := otel.Tracer("test")
	var metrics discordmetrics.DiscordMetrics = nil
	ld, err := NewLeaderboardDiscord(ctx, session, publisher, nil, helper, cfg, resolver, store, tracer, metrics)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ld == nil {
		t.Fatalf("expected non-nil LeaderboardDiscordInterface")
	}

	// Verify getters return non-nil managers
	if ld.GetLeaderboardUpdateManager() == nil {
		t.Fatalf("expected non-nil LeaderboardUpdateManager")
	}
	if ld.GetClaimTagManager() == nil {
		t.Fatalf("expected non-nil ClaimTagManager")
	}
}

// testingLogger is a minimal placeholder to satisfy *slog.Logger type via nil; not used.
type testingLogger struct{}

// nilTracer is a placeholder type alias; we pass nil so constructor doesn't use it.
type nilTracer = interface{}

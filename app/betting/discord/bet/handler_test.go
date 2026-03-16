package bet

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	discordpkg "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/bwmarrin/discordgo"
	"go.opentelemetry.io/otel/trace/noop"
)

func newTestManager(session discordpkg.Session, baseURL string) *betManager {
	cfg := &config.Config{}
	cfg.PWA.BaseURL = baseURL
	return &betManager{
		session: session,
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:     cfg,
		tracer:  noop.NewTracerProvider().Tracer("test"),
		metrics: discordmetrics.NewNoop(),
	}
}

func newInteraction(guildID, userID string) *discordgo.InteractionCreate {
	ic := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			GuildID: guildID,
		},
	}
	if userID != "" {
		ic.Member = &discordgo.Member{
			User: &discordgo.User{ID: userID},
		}
	}
	return ic
}

func TestHandleBetCommand_SendsPWALink(t *testing.T) {
	fs := discordpkg.NewFakeSession()

	var respondedContent string
	fs.InteractionRespondFunc = func(i *discordgo.Interaction, resp *discordgo.InteractionResponse, _ ...discordgo.RequestOption) error {
		respondedContent = resp.Data.Content
		return nil
	}

	mgr := newTestManager(fs, "https://custom.example.com")
	mgr.HandleBetCommand(context.Background(), newInteraction("guild1", "user1"))

	if !strings.Contains(respondedContent, "https://custom.example.com/betting") {
		t.Errorf("expected PWA betting URL in response, got: %q", respondedContent)
	}
	if !contains(fs.Trace(), "InteractionRespond") {
		t.Error("expected InteractionRespond to be called")
	}
}

func TestHandleBetCommand_FallbackURL_WhenBaseURLEmpty(t *testing.T) {
	fs := discordpkg.NewFakeSession()

	var respondedContent string
	fs.InteractionRespondFunc = func(i *discordgo.Interaction, resp *discordgo.InteractionResponse, _ ...discordgo.RequestOption) error {
		respondedContent = resp.Data.Content
		return nil
	}

	mgr := newTestManager(fs, "")
	mgr.HandleBetCommand(context.Background(), newInteraction("guild1", "user1"))

	if !strings.Contains(respondedContent, "https://frolf-bot.duckdns.org/betting") {
		t.Errorf("expected hardcoded fallback URL, got: %q", respondedContent)
	}
}

func TestHandleBetCommand_TrailingSlashTrimmed(t *testing.T) {
	fs := discordpkg.NewFakeSession()

	var respondedContent string
	fs.InteractionRespondFunc = func(i *discordgo.Interaction, resp *discordgo.InteractionResponse, _ ...discordgo.RequestOption) error {
		respondedContent = resp.Data.Content
		return nil
	}

	mgr := newTestManager(fs, "https://custom.example.com/")
	mgr.HandleBetCommand(context.Background(), newInteraction("guild1", "user1"))

	if strings.Contains(respondedContent, "//betting") {
		t.Errorf("expected trailing slash to be trimmed, got: %q", respondedContent)
	}
	if !strings.Contains(respondedContent, "https://custom.example.com/betting") {
		t.Errorf("expected trimmed URL, got: %q", respondedContent)
	}
}

func TestHandleBetCommand_NilMember_NoRespond(t *testing.T) {
	fs := discordpkg.NewFakeSession()

	mgr := newTestManager(fs, "https://custom.example.com")
	// interaction with no Member set
	mgr.HandleBetCommand(context.Background(), newInteraction("guild1", ""))

	if contains(fs.Trace(), "InteractionRespond") {
		t.Error("expected InteractionRespond NOT to be called when member is nil")
	}
}

func TestHandleBetCommand_SessionError_LogsAndContinues(t *testing.T) {
	fs := discordpkg.NewFakeSession()
	fs.InteractionRespondFunc = func(*discordgo.Interaction, *discordgo.InteractionResponse, ...discordgo.RequestOption) error {
		return errors.New("discord api error")
	}

	mgr := newTestManager(fs, "https://custom.example.com")
	// Should not panic even when session returns error
	mgr.HandleBetCommand(context.Background(), newInteraction("guild1", "user1"))

	if !contains(fs.Trace(), "InteractionRespond") {
		t.Error("expected InteractionRespond to be called")
	}
}

func TestHandleBetCommand_ResponseIsEphemeral(t *testing.T) {
	fs := discordpkg.NewFakeSession()

	var flags discordgo.MessageFlags
	fs.InteractionRespondFunc = func(i *discordgo.Interaction, resp *discordgo.InteractionResponse, _ ...discordgo.RequestOption) error {
		flags = resp.Data.Flags
		return nil
	}

	mgr := newTestManager(fs, "https://custom.example.com")
	mgr.HandleBetCommand(context.Background(), newInteraction("guild1", "user1"))

	if flags != discordgo.MessageFlagsEphemeral {
		t.Errorf("expected ephemeral flag, got flags=%v", flags)
	}
}

func contains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}

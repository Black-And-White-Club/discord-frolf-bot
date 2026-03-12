package challenge

import (
	"testing"
	"time"

	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	clubtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/club"
	"github.com/bwmarrin/discordgo"
)

func TestBuildChallengeComponents_OpenChallengeShowsRespondButtonsAndDeepLink(t *testing.T) {
	components := buildChallengeComponents(
		&config.Config{PWA: config.PWAConfig{BaseURL: "https://app.example"}},
		clubtypes.ChallengeDetail{
			ChallengeSummary: clubtypes.ChallengeSummary{
				ID:       "challenge-1",
				Status:   clubtypes.ChallengeStatusOpen,
				OpenedAt: time.Now().UTC(),
			},
		},
	)

	row := singleActionsRow(t, components)
	if len(row.Components) != 3 {
		t.Fatalf("expected 3 buttons, got %d", len(row.Components))
	}

	assertButtonLabel(t, row.Components[0], "Accept")
	assertButtonLabel(t, row.Components[1], "Decline")
	assertLinkButton(t, row.Components[2], "Open in App", "https://app.example/challenges/challenge-1")
}

func TestBuildChallengeComponents_AcceptedUnlinkedChallengeShowsScheduleAndDeepLink(t *testing.T) {
	components := buildChallengeComponents(
		&config.Config{PWA: config.PWAConfig{BaseURL: "https://app.example"}},
		clubtypes.ChallengeDetail{
			ChallengeSummary: clubtypes.ChallengeSummary{
				ID:       "challenge-2",
				Status:   clubtypes.ChallengeStatusAccepted,
				OpenedAt: time.Now().UTC(),
			},
		},
	)

	row := singleActionsRow(t, components)
	if len(row.Components) != 2 {
		t.Fatalf("expected 2 buttons, got %d", len(row.Components))
	}

	button, ok := row.Components[0].(discordgo.Button)
	if !ok {
		t.Fatalf("expected button component, got %T", row.Components[0])
	}
	if button.Label != "Schedule Round" {
		t.Fatalf("expected schedule button label, got %q", button.Label)
	}
	if button.CustomID != challengeSchedulePrefix+"challenge-2" {
		t.Fatalf("expected schedule button custom ID, got %q", button.CustomID)
	}

	assertLinkButton(t, row.Components[1], "Open in App", "https://app.example/challenges/challenge-2")
}

func singleActionsRow(t *testing.T, components []discordgo.MessageComponent) discordgo.ActionsRow {
	t.Helper()
	if len(components) != 1 {
		t.Fatalf("expected exactly one action row, got %d", len(components))
	}

	row, ok := components[0].(discordgo.ActionsRow)
	if !ok {
		t.Fatalf("expected actions row, got %T", components[0])
	}
	return row
}

func assertButtonLabel(t *testing.T, component discordgo.MessageComponent, want string) {
	t.Helper()

	button, ok := component.(discordgo.Button)
	if !ok {
		t.Fatalf("expected button component, got %T", component)
	}
	if button.Label != want {
		t.Fatalf("expected button label %q, got %q", want, button.Label)
	}
}

func assertLinkButton(t *testing.T, component discordgo.MessageComponent, wantLabel, wantURL string) {
	t.Helper()

	button, ok := component.(discordgo.Button)
	if !ok {
		t.Fatalf("expected button component, got %T", component)
	}
	if button.Label != wantLabel {
		t.Fatalf("expected link label %q, got %q", wantLabel, button.Label)
	}
	if button.URL != wantURL {
		t.Fatalf("expected link URL %q, got %q", wantURL, button.URL)
	}
}

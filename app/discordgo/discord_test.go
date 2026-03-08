package discord

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestDiscordSession_WebhookExecute_ForwardsWebhookID(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	originalEndpointWebhooks := discordgo.EndpointWebhooks
	discordgo.EndpointWebhooks = server.URL + "/webhooks/"
	t.Cleanup(func() {
		discordgo.EndpointWebhooks = originalEndpointWebhooks
	})

	underlying, err := discordgo.New("Bot unit-test-token")
	if err != nil {
		t.Fatalf("failed to create discordgo session: %v", err)
	}
	underlying.Client = server.Client()

	session := NewDiscordSession(underlying, testLogger())
	if _, err := session.WebhookExecute("webhook-123", "token-abc", false, &discordgo.WebhookParams{Content: "hello"}); err != nil {
		t.Fatalf("WebhookExecute returned error: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("expected POST request, got %q", gotMethod)
	}
	if gotPath != "/webhooks/webhook-123/token-abc" {
		t.Fatalf("expected webhook path to include webhook id, got %q", gotPath)
	}
}

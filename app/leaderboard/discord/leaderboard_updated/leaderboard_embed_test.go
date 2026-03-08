package leaderboardupdated

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"testing"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
)

// ---------------------------------------------------------------------------
// buildLeaderboardDescription unit tests
// ---------------------------------------------------------------------------

func Test_buildLeaderboardDescription(t *testing.T) {
	t.Run("empty leaderboard returns placeholder", func(t *testing.T) {
		got := buildLeaderboardDescription(nil)
		if got != "*No entries yet.*" {
			t.Errorf("unexpected description: %q", got)
		}
	})

	t.Run("single entry shows gold medal", func(t *testing.T) {
		entries := []LeaderboardEntry{{Rank: 1, UserID: "user1"}}
		got := buildLeaderboardDescription(entries)
		if !strings.Contains(got, "🥇") {
			t.Errorf("expected gold medal, got: %q", got)
		}
		if !strings.Contains(got, "@user1") {
			t.Errorf("expected user label, got: %q", got)
		}
	})

	t.Run("valid discord id prefers mention over display name", func(t *testing.T) {
		entries := []LeaderboardEntry{{Rank: 1, UserID: "839877196898238526", DisplayName: "Alice"}}
		got := buildLeaderboardDescription(entries)
		if !strings.Contains(got, "<@839877196898238526>") {
			t.Errorf("expected discord mention for valid ID, got: %q", got)
		}
	})

	t.Run("mention-formatted IDs are normalized", func(t *testing.T) {
		entries := []LeaderboardEntry{{Rank: 1, UserID: "<@!839877196898238526>"}}
		got := buildLeaderboardDescription(entries)
		if !strings.Contains(got, "<@839877196898238526>") {
			t.Errorf("expected normalized mention id, got: %q", got)
		}
		if strings.Contains(got, "<@<@!839877196898238526>>") {
			t.Errorf("expected no nested mention format, got: %q", got)
		}
	})

	t.Run("short numeric pseudo-id falls back to plain label", func(t *testing.T) {
		entries := []LeaderboardEntry{{Rank: 1, UserID: "23"}}
		got := buildLeaderboardDescription(entries)
		if !strings.Contains(got, "@23") {
			t.Errorf("expected plain pseudo-id label, got: %q", got)
		}
		if strings.Contains(got, "<@23>") {
			t.Errorf("expected no mention for short pseudo-id, got: %q", got)
		}
	})

	t.Run("short numeric pseudo-id prefers display name when available", func(t *testing.T) {
		entries := []LeaderboardEntry{{Rank: 1, UserID: "23", DisplayName: "muffinmaster123"}}
		got := buildLeaderboardDescription(entries)
		if !strings.Contains(got, "@muffinmaster123") {
			t.Errorf("expected display-name fallback label, got: %q", got)
		}
		if strings.Contains(got, "@23") {
			t.Errorf("expected numeric pseudo-id to be replaced by display name, got: %q", got)
		}
	})

	t.Run("placeholder user labels prefer display name when available", func(t *testing.T) {
		entries := []LeaderboardEntry{{Rank: 1, UserID: "Tag 23 Placeholder", DisplayName: "muffinmaster123"}}
		got := buildLeaderboardDescription(entries)
		if !strings.Contains(got, "@muffinmaster123") {
			t.Errorf("expected display-name fallback label, got: %q", got)
		}
		if strings.Contains(got, "@Tag 23 Placeholder") {
			t.Errorf("expected placeholder label to be replaced, got: %q", got)
		}
	})

	t.Run("raw @handle is preserved without double @", func(t *testing.T) {
		entries := []LeaderboardEntry{{Rank: 1, UserID: "@farrmich"}}
		got := buildLeaderboardDescription(entries)
		if !strings.Contains(got, "@farrmich") {
			t.Errorf("expected raw handle, got: %q", got)
		}
		if strings.Contains(got, "@@farrmich") {
			t.Errorf("expected no duplicated @ prefix, got: %q", got)
		}
	})

	t.Run("last place gets trash emoji", func(t *testing.T) {
		entries := createTestLeaderboard(5)
		got := buildLeaderboardDescription(entries)
		if !strings.Contains(got, "🗑️") {
			t.Errorf("expected trash emoji for last place, got: %q", got)
		}
	})

	t.Run("entry with points shows pts and rds", func(t *testing.T) {
		entries := createTestLeaderboardWithPoints(1)
		got := buildLeaderboardDescription(entries)
		if !strings.Contains(got, "pts") {
			t.Errorf("expected points display, got: %q", got)
		}
	})

	t.Run("description is capped at maxDescriptionLength", func(t *testing.T) {
		// Create a huge leaderboard to force truncation
		entries := createTestLeaderboard(500)
		got := buildLeaderboardDescription(entries)
		if len(got) > maxDescriptionLength {
			t.Errorf("description exceeds maxDescriptionLength: got %d chars", len(got))
		}
		if !strings.Contains(got, "truncated") {
			t.Errorf("expected truncation notice in description")
		}
	})
}

// ---------------------------------------------------------------------------
// SendLeaderboardEmbed integration tests
// ---------------------------------------------------------------------------

func Test_leaderboardUpdateManager_SendLeaderboardEmbed(t *testing.T) {
	channelID := "test-channel"

	tests := []struct {
		name        string
		setupFake   func(t *testing.T, fakeSession *discord.FakeSession)
		leaderboard []LeaderboardEntry
		page        int32
		expectErr   bool
	}{
		{
			name: "Empty leaderboard",
			setupFake: func(t *testing.T, fakeSession *discord.FakeSession) {
				fakeSession.ChannelMessageSendComplexFunc = func(chID string, send *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					if len(send.Embeds) != 1 {
						t.Errorf("Expected 1 embed, got %d", len(send.Embeds))
					}
					embed := send.Embeds[0]
					if embed.Title != "🏆 Leaderboard" {
						t.Errorf("Unexpected title: %s", embed.Title)
					}
					if embed.Description != "*No entries yet.*" {
						t.Errorf("Unexpected description: %q", embed.Description)
					}
					if len(embed.Fields) != 0 {
						t.Errorf("Expected no fields, got %d", len(embed.Fields))
					}
					// No pagination buttons
					if len(send.Components) != 0 {
						t.Errorf("Expected no components, got %d", len(send.Components))
					}
					return &discordgo.Message{ID: "test-message-id", Embeds: send.Embeds}, nil
				}
			},
			leaderboard: []LeaderboardEntry{},
			page:        1,
			expectErr:   false,
		},
		{
			name: "Single page leaderboard (less than 10 entries)",
			setupFake: func(t *testing.T, fakeSession *discord.FakeSession) {
				fakeSession.ChannelMessageSendComplexFunc = func(chID string, send *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					embed := send.Embeds[0]
					if !strings.Contains(embed.Description, "🥇") {
						t.Errorf("Expected gold medal in description, got: %q", embed.Description)
					}
					if !strings.Contains(embed.Description, "@user1") {
						t.Errorf("Expected user1 in description, got: %q", embed.Description)
					}
					if len(embed.Fields) != 0 {
						t.Errorf("Expected no fields, got %d", len(embed.Fields))
					}
					if len(send.Components) != 0 {
						t.Errorf("Expected no pagination components, got %d", len(send.Components))
					}
					return &discordgo.Message{ID: "test-message-id", Embeds: send.Embeds}, nil
				}
			},
			leaderboard: []LeaderboardEntry{
				{Rank: 1, UserID: "user1"},
				{Rank: 2, UserID: "user2"},
				{Rank: 3, UserID: "user3"},
				{Rank: 4, UserID: "user4"},
				{Rank: 5, UserID: "user5"},
			},
			page:      1,
			expectErr: false,
		},
		{
			name: "Large leaderboard (all entries in description, no pagination)",
			setupFake: func(t *testing.T, fakeSession *discord.FakeSession) {
				fakeSession.ChannelMessageSendComplexFunc = func(chID string, send *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					embed := send.Embeds[0]
					// Description should be non-empty but within limit
					if len(embed.Description) > maxDescriptionLength {
						t.Errorf("Description exceeds limit: %d chars", len(embed.Description))
					}
					// No pagination
					if len(send.Components) != 0 {
						t.Errorf("Expected no pagination components, got %d", len(send.Components))
					}
					return &discordgo.Message{ID: "test-message-id", Embeds: send.Embeds}, nil
				}
			},
			leaderboard: createTestLeaderboard(15),
			page:        1,
			expectErr:   false,
		},
		{
			name: "Leaderboard with points display",
			setupFake: func(t *testing.T, fakeSession *discord.FakeSession) {
				fakeSession.ChannelMessageSendComplexFunc = func(chID string, send *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					embed := send.Embeds[0]
					if !strings.Contains(embed.Description, "pts") {
						t.Errorf("Expected points in description, got: %q", embed.Description)
					}
					return &discordgo.Message{ID: "test-message-id", Embeds: send.Embeds}, nil
				}
			},
			leaderboard: createTestLeaderboardWithPoints(3),
			page:        1,
			expectErr:   false,
		},
		{
			name: "Discord API error",
			setupFake: func(t *testing.T, fakeSession *discord.FakeSession) {
				fakeSession.ChannelMessageSendComplexFunc = func(chID string, send *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return nil, fmt.Errorf("discord API error")
				}
			},
			leaderboard: createTestLeaderboard(5),
			page:        1,
			expectErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeSession := discord.NewFakeSession()
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

			if tt.setupFake != nil {
				tt.setupFake(t, fakeSession)
			}

			lum := &leaderboardUpdateManager{
				logger:             mockLogger,
				session:            fakeSession,
				messageByChannelID: make(map[string]string),
				operationWrapper: func(ctx context.Context, name string, fn func(ctx context.Context) (LeaderboardUpdateOperationResult, error)) (LeaderboardUpdateOperationResult, error) {
					return fn(ctx)
				},
			}

			got, err := lum.SendLeaderboardEmbed(context.Background(), channelID, tt.leaderboard, tt.page)

			if (err != nil) != tt.expectErr {
				t.Errorf("SendLeaderboardEmbed() error = %v, wantErr %v", err, tt.expectErr)
				return
			}

			if !tt.expectErr {
				if got.Success == nil {
					t.Errorf("SendLeaderboardEmbed() Success is nil")
				}
			}
		})
	}
}

func TestSendLeaderboardEmbed_EditsTrackedMessage(t *testing.T) {
	channelID := "test-channel"
	trackedMessageID := "leaderboard-message-123"

	fakeSession := discord.NewFakeSession()
	fakeSession.ChannelMessageSendComplexFunc = func(channelID string, data *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error) {
		t.Fatalf("unexpected send call when a tracked message exists")
		return nil, nil
	}
	fakeSession.ChannelMessageEditComplexFunc = func(m *discordgo.MessageEdit, options ...discordgo.RequestOption) (*discordgo.Message, error) {
		if m.Channel != channelID {
			t.Fatalf("unexpected channel: got %s want %s", m.Channel, channelID)
		}
		if m.ID != trackedMessageID {
			t.Fatalf("unexpected message id: got %s want %s", m.ID, trackedMessageID)
		}
		return &discordgo.Message{ID: m.ID, ChannelID: m.Channel}, nil
	}

	lum := &leaderboardUpdateManager{
		logger:             slog.New(slog.NewTextHandler(io.Discard, nil)),
		session:            fakeSession,
		messageByChannelID: map[string]string{channelID: trackedMessageID},
		operationWrapper: func(ctx context.Context, name string, fn func(ctx context.Context) (LeaderboardUpdateOperationResult, error)) (LeaderboardUpdateOperationResult, error) {
			return fn(ctx)
		},
	}

	got, err := lum.SendLeaderboardEmbed(context.Background(), channelID, createTestLeaderboard(3), 1)
	if err != nil {
		t.Fatalf("SendLeaderboardEmbed() error = %v", err)
	}

	msg, ok := got.Success.(*discordgo.Message)
	if !ok {
		t.Fatalf("expected *discordgo.Message success payload")
	}
	if msg.ID != trackedMessageID {
		t.Fatalf("unexpected edited message id: got %s want %s", msg.ID, trackedMessageID)
	}
}

func TestSendLeaderboardEmbed_UnknownTrackedMessageFallsBackToSend(t *testing.T) {
	channelID := "test-channel"
	oldMessageID := "deleted-message-123"
	newMessageID := "new-message-456"

	fakeSession := discord.NewFakeSession()
	fakeSession.ChannelMessageEditComplexFunc = func(m *discordgo.MessageEdit, options ...discordgo.RequestOption) (*discordgo.Message, error) {
		return nil, &discordgo.RESTError{
			Message: &discordgo.APIErrorMessage{
				Code:    discordgo.ErrCodeUnknownMessage,
				Message: "Unknown Message",
			},
		}
	}
	fakeSession.ChannelMessageSendComplexFunc = func(channelID string, data *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error) {
		return &discordgo.Message{ID: newMessageID, ChannelID: channelID}, nil
	}

	lum := &leaderboardUpdateManager{
		logger:             slog.New(slog.NewTextHandler(io.Discard, nil)),
		session:            fakeSession,
		messageByChannelID: map[string]string{channelID: oldMessageID},
		operationWrapper: func(ctx context.Context, name string, fn func(ctx context.Context) (LeaderboardUpdateOperationResult, error)) (LeaderboardUpdateOperationResult, error) {
			return fn(ctx)
		},
	}

	got, err := lum.SendLeaderboardEmbed(context.Background(), channelID, createTestLeaderboard(2), 1)
	if err != nil {
		t.Fatalf("SendLeaderboardEmbed() error = %v", err)
	}

	msg, ok := got.Success.(*discordgo.Message)
	if !ok {
		t.Fatalf("expected *discordgo.Message success payload")
	}
	if msg.ID != newMessageID {
		t.Fatalf("unexpected sent message id: got %s want %s", msg.ID, newMessageID)
	}
	if tracked := lum.getTrackedMessageID(channelID); tracked != newMessageID {
		t.Fatalf("tracked message id not updated: got %s want %s", tracked, newMessageID)
	}
}

func TestSendLeaderboardEmbed_DiscoversExistingMessageAfterRestart(t *testing.T) {
	channelID := "test-channel"
	botID := "bot-123"
	existingMessageID := "existing-message-789"

	fakeSession := discord.NewFakeSession()
	fakeSession.GetBotUserFunc = func() (*discordgo.User, error) {
		return &discordgo.User{ID: botID}, nil
	}
	fakeSession.ChannelMessagesFunc = func(channelID string, limit int, beforeID, afterID, aroundID string, options ...discordgo.RequestOption) ([]*discordgo.Message, error) {
		return []*discordgo.Message{
			{
				ID:        existingMessageID,
				ChannelID: channelID,
				Author:    &discordgo.User{ID: botID},
				Embeds: []*discordgo.MessageEmbed{
					{Title: leaderboardEmbedTitle},
				},
			},
		}, nil
	}
	fakeSession.ChannelMessageEditComplexFunc = func(m *discordgo.MessageEdit, options ...discordgo.RequestOption) (*discordgo.Message, error) {
		return &discordgo.Message{ID: m.ID, ChannelID: m.Channel}, nil
	}
	fakeSession.ChannelMessageSendComplexFunc = func(channelID string, data *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error) {
		t.Fatalf("unexpected send call; expected edit of discovered message")
		return nil, nil
	}

	lum := &leaderboardUpdateManager{
		logger:             slog.New(slog.NewTextHandler(io.Discard, nil)),
		session:            fakeSession,
		messageByChannelID: make(map[string]string),
		operationWrapper: func(ctx context.Context, name string, fn func(ctx context.Context) (LeaderboardUpdateOperationResult, error)) (LeaderboardUpdateOperationResult, error) {
			return fn(ctx)
		},
	}

	got, err := lum.SendLeaderboardEmbed(context.Background(), channelID, createTestLeaderboard(4), 1)
	if err != nil {
		t.Fatalf("SendLeaderboardEmbed() error = %v", err)
	}

	msg, ok := got.Success.(*discordgo.Message)
	if !ok {
		t.Fatalf("expected *discordgo.Message success payload")
	}
	if msg.ID != existingMessageID {
		t.Fatalf("unexpected edited message id: got %s want %s", msg.ID, existingMessageID)
	}
	if tracked := lum.getTrackedMessageID(channelID); tracked != existingMessageID {
		t.Fatalf("tracked message id not persisted after discovery: got %s want %s", tracked, existingMessageID)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func createTestLeaderboard(count int) []LeaderboardEntry {
	entries := make([]LeaderboardEntry, count)
	for i := 0; i < count; i++ {
		entries[i] = LeaderboardEntry{
			Rank:   sharedtypes.TagNumber(i + 1),
			UserID: sharedtypes.DiscordID(fmt.Sprintf("user%d", i+1)),
		}
	}
	return entries
}

func createTestLeaderboardWithPoints(count int) []LeaderboardEntry {
	entries := make([]LeaderboardEntry, count)
	for i := 0; i < count; i++ {
		entries[i] = LeaderboardEntry{
			Rank:         sharedtypes.TagNumber(i + 1),
			UserID:       sharedtypes.DiscordID(fmt.Sprintf("user%d", i+1)),
			TotalPoints:  (count - i) * 10,
			RoundsPlayed: i + 1,
		}
	}
	return entries
}

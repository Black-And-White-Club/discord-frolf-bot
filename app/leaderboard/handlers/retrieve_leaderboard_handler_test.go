package handlers

import (
	"context"
	"io"
	"log/slog"
	"slices"
	"testing"

	leaderboardupdated "github.com/Black-And-White-Club/discord-frolf-bot/app/leaderboard/discord/leaderboard_updated"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	discordleaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/leaderboard"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
)

func TestHandleLeaderboardRetrieveRequest(t *testing.T) {
	tests := []struct {
		name    string
		payload *discordleaderboardevents.LeaderboardRetrieveRequestPayloadV1
		wantErr bool
	}{
		{
			name: "successful_retrieve_request",
			payload: &discordleaderboardevents.LeaderboardRetrieveRequestPayloadV1{
				GuildID: "guild123",
			},
			wantErr: false,
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	h := NewLeaderboardHandlers(
		logger,
		nil,
		nil,
		nil,
		nil,
	)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			results, err := h.HandleLeaderboardRetrieveRequest(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleLeaderboardRetrieveRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(results) == 0 {
				t.Errorf("expected results, got empty slice")
			}
		})
	}
}

func TestHandleLeaderboardUpdatedNotification(t *testing.T) {
	tests := []struct {
		name    string
		payload *leaderboardevents.LeaderboardUpdatedPayloadV1
		wantErr bool
	}{
		{
			name: "successful_updated_notification",
			payload: &leaderboardevents.LeaderboardUpdatedPayloadV1{
				GuildID: sharedtypes.GuildID("guild123"),
			},
			wantErr: false,
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	h := NewLeaderboardHandlers(
		logger,
		nil,
		nil,
		nil,
		nil,
	)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			results, err := h.HandleLeaderboardUpdatedNotification(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleLeaderboardUpdatedNotification() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(results) == 0 {
				t.Errorf("expected results, got empty slice")
			}
		})
	}
}

func TestHandleLeaderboardResponse(t *testing.T) {
	tests := []struct {
		name    string
		payload *leaderboardevents.GetLeaderboardResponsePayloadV1
		wantErr bool
	}{
		{
			name: "successful_leaderboard_response",
			payload: &leaderboardevents.GetLeaderboardResponsePayloadV1{
				GuildID: sharedtypes.GuildID("guild123"),
				Leaderboard: []leaderboardtypes.LeaderboardEntry{
					{
						TagNumber: sharedtypes.TagNumber(1),
						UserID:    sharedtypes.DiscordID("user1"),
					},
					{
						TagNumber: sharedtypes.TagNumber(2),
						UserID:    sharedtypes.DiscordID("user2"),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "empty_leaderboard_response",
			payload: &leaderboardevents.GetLeaderboardResponsePayloadV1{
				GuildID:     sharedtypes.GuildID("guild123"),
				Leaderboard: []leaderboardtypes.LeaderboardEntry{},
			},
			wantErr: false,
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	h := NewLeaderboardHandlers(
		logger,
		nil,
		nil,
		nil,
		nil,
	)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			results, err := h.HandleLeaderboardResponse(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleLeaderboardResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(results) == 0 {
				t.Errorf("expected results, got empty slice")
			}
		})
	}
}

func TestHandleLeaderboardResponse_SendsFullSnapshotEmbed(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	cfg := &config.Config{}
	cfg.Discord.LeaderboardChannelID = "leaderboard-channel"

	fakeDiscord := &FakeLeaderboardDiscord{}
	var gotChannelID string
	var gotRanks []sharedtypes.TagNumber
	var gotUserIDs []sharedtypes.DiscordID
	var gotDisplayNames []string
	fakeDiscord.LeaderboardUpdateManager.SendLeaderboardEmbedFunc = func(
		ctx context.Context,
		channelID string,
		leaderboard []leaderboardupdated.LeaderboardEntry,
		page int32,
	) (leaderboardupdated.LeaderboardUpdateOperationResult, error) {
		gotChannelID = channelID
		gotRanks = make([]sharedtypes.TagNumber, 0, len(leaderboard))
		gotUserIDs = make([]sharedtypes.DiscordID, 0, len(leaderboard))
		gotDisplayNames = make([]string, 0, len(leaderboard))
		for _, entry := range leaderboard {
			gotRanks = append(gotRanks, entry.Rank)
			gotUserIDs = append(gotUserIDs, entry.UserID)
			gotDisplayNames = append(gotDisplayNames, entry.DisplayName)
		}
		return leaderboardupdated.LeaderboardUpdateOperationResult{Success: "ok"}, nil
	}

	h := NewLeaderboardHandlers(
		logger,
		cfg,
		nil,
		fakeDiscord,
		nil,
	)

	payload := &leaderboardevents.GetLeaderboardResponsePayloadV1{
		GuildID: sharedtypes.GuildID("guild123"),
		Leaderboard: []leaderboardtypes.LeaderboardEntry{
			{TagNumber: 13, UserID: "user13"},
			{TagNumber: 1, UserID: "user1"},
		},
		Profiles: map[sharedtypes.DiscordID]*usertypes.UserProfile{
			"user1":  {UserID: "user1", DisplayName: "Alice"},
			"user13": {UserID: "user13", DisplayName: "Bob"},
		},
	}

	results, err := h.HandleLeaderboardResponse(context.Background(), payload)
	if err != nil {
		t.Fatalf("HandleLeaderboardResponse() error = %v", err)
	}
	if len(results) == 0 {
		t.Fatalf("expected handler results, got empty")
	}
	if gotChannelID != "leaderboard-channel" {
		t.Fatalf("unexpected channel: got %s want %s", gotChannelID, "leaderboard-channel")
	}
	if !slices.Equal(gotRanks, []sharedtypes.TagNumber{1, 13}) {
		t.Fatalf("unexpected ranks: got %v want [1 13]", gotRanks)
	}
	if !slices.Equal(gotUserIDs, []sharedtypes.DiscordID{"user1", "user13"}) {
		t.Fatalf("unexpected users: got %v want [user1 user13]", gotUserIDs)
	}
	if !slices.Equal(gotDisplayNames, []string{"Alice", "Bob"}) {
		t.Fatalf("unexpected display names: got %v want [Alice Bob]", gotDisplayNames)
	}
}

func TestHandleLeaderboardResponse_ProfileFallsBackToUDiscUsername(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	cfg := &config.Config{}
	cfg.Discord.LeaderboardChannelID = "leaderboard-channel"

	fakeDiscord := &FakeLeaderboardDiscord{}
	var gotDisplayNames []string
	fakeDiscord.LeaderboardUpdateManager.SendLeaderboardEmbedFunc = func(
		ctx context.Context,
		channelID string,
		leaderboard []leaderboardupdated.LeaderboardEntry,
		page int32,
	) (leaderboardupdated.LeaderboardUpdateOperationResult, error) {
		gotDisplayNames = make([]string, 0, len(leaderboard))
		for _, entry := range leaderboard {
			gotDisplayNames = append(gotDisplayNames, entry.DisplayName)
		}
		return leaderboardupdated.LeaderboardUpdateOperationResult{Success: "ok"}, nil
	}

	h := NewLeaderboardHandlers(
		logger,
		cfg,
		nil,
		fakeDiscord,
		nil,
	)

	udiscUsername := "muffinmaster123"
	payload := &leaderboardevents.GetLeaderboardResponsePayloadV1{
		GuildID: sharedtypes.GuildID("guild123"),
		Leaderboard: []leaderboardtypes.LeaderboardEntry{
			{TagNumber: 23, UserID: "23"},
		},
		Profiles: map[sharedtypes.DiscordID]*usertypes.UserProfile{
			"23": {
				UserID:        "23",
				DisplayName:   "",
				UDiscUsername: &udiscUsername,
			},
		},
	}

	_, err := h.HandleLeaderboardResponse(context.Background(), payload)
	if err != nil {
		t.Fatalf("HandleLeaderboardResponse() error = %v", err)
	}
	if !slices.Equal(gotDisplayNames, []string{"muffinmaster123"}) {
		t.Fatalf("unexpected display names: got %v want [muffinmaster123]", gotDisplayNames)
	}
}

func TestHandleLeaderboardResponse_ProfilePlaceholderDisplayNameFallsBackToUDiscUsername(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	cfg := &config.Config{}
	cfg.Discord.LeaderboardChannelID = "leaderboard-channel"

	fakeDiscord := &FakeLeaderboardDiscord{}
	var gotDisplayNames []string
	fakeDiscord.LeaderboardUpdateManager.SendLeaderboardEmbedFunc = func(
		ctx context.Context,
		channelID string,
		leaderboard []leaderboardupdated.LeaderboardEntry,
		page int32,
	) (leaderboardupdated.LeaderboardUpdateOperationResult, error) {
		gotDisplayNames = make([]string, 0, len(leaderboard))
		for _, entry := range leaderboard {
			gotDisplayNames = append(gotDisplayNames, entry.DisplayName)
		}
		return leaderboardupdated.LeaderboardUpdateOperationResult{Success: "ok"}, nil
	}

	h := NewLeaderboardHandlers(
		logger,
		cfg,
		nil,
		fakeDiscord,
		nil,
	)

	udiscUsername := "muffinmaster123"
	payload := &leaderboardevents.GetLeaderboardResponsePayloadV1{
		GuildID: sharedtypes.GuildID("guild123"),
		Leaderboard: []leaderboardtypes.LeaderboardEntry{
			{TagNumber: 23, UserID: "23"},
		},
		Profiles: map[sharedtypes.DiscordID]*usertypes.UserProfile{
			"23": {
				UserID:        "23",
				DisplayName:   "Tag 23 Placeholder",
				UDiscUsername: &udiscUsername,
			},
		},
	}

	_, err := h.HandleLeaderboardResponse(context.Background(), payload)
	if err != nil {
		t.Fatalf("HandleLeaderboardResponse() error = %v", err)
	}
	if !slices.Equal(gotDisplayNames, []string{"muffinmaster123"}) {
		t.Fatalf("unexpected display names: got %v want [muffinmaster123]", gotDisplayNames)
	}
}

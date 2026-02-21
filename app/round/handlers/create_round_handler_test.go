package handlers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	rounddiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord"
	createround "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/create_round"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	discordroundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/round"
	sharedroundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/round"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

func TestRoundHandlers_HandleRoundCreateRequested(t *testing.T) {
	tests := []struct {
		name    string
		payload *discordroundevents.CreateRoundModalPayloadV1
		ctx     context.Context
		want    []handlerwrapper.Result
		wantErr bool
		wantLen int // Expected number of results
	}{
		{
			name: "successful_round_create_request",
			payload: &sharedroundevents.CreateRoundModalPayloadV1{
				GuildID:     "123456789",
				Title:       "Test Round",
				Description: "Test Description",
				Location:    "Test Location",
				StartTime:   "2024-01-01T12:00:00Z",
				UserID:      "user123",
				ChannelID:   "channel123",
				Timezone:    "America/New_York",
			},
			ctx:     context.Background(),
			want:    nil, // We'll check the length instead of deep equality
			wantErr: false,
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeRoundDiscord := &FakeRoundDiscord{}
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

			h := NewRoundHandlers(
				mockLogger,
				&config.Config{},
				nil,
				fakeRoundDiscord,
				nil,
			)

			got, err := h.HandleRoundCreateRequested(tt.ctx, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundCreateRequested() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != tt.wantLen {
				t.Errorf("HandleRoundCreateRequested() got %d results, want %d", len(got), tt.wantLen)
				return
			}

			if tt.wantLen > 0 {
				result := got[0]
				if result.Topic != roundevents.RoundCreationRequestedV1 {
					t.Errorf("HandleRoundCreateRequested() topic = %s, want %s", result.Topic, roundevents.RoundCreationRequestedV1)
				}
				if result.Payload == nil {
					t.Errorf("HandleRoundCreateRequested() payload is nil")
				}
			}
		})
	}
}

func TestRoundHandlers_HandleRoundCreated(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	parsedTime, _ := time.Parse(time.RFC3339, "2024-01-01T12:00:00Z")
	startTime := sharedtypes.StartTime(parsedTime)
	guildID := sharedtypes.GuildID("123456789")
	channelID := "1344376922888474625"

	tests := []struct {
		name              string
		payload           *roundevents.RoundCreatedPayloadV1
		ctx               context.Context
		wantErr           bool
		wantLen           int
		wantNativeCreated bool
		wantNativeFailed  bool
		wantEmbedUpdate   bool
		setup             func(*FakeRoundDiscord)
	}{
		{
			name: "successful_round_creation",
			payload: &roundevents.RoundCreatedPayloadV1{
				GuildID: guildID,
				BaseRoundPayload: roundtypes.BaseRoundPayload{
					RoundID:     testRoundID,
					Title:       roundtypes.Title("Test Round"),
					Description: roundtypes.Description("Test Description"),
					Location:    roundtypes.Location("Test Location"),
					StartTime:   &startTime,
					UserID:      sharedtypes.DiscordID("user_id"),
				},
				ChannelID: channelID,
			},
			ctx:               context.Background(),
			wantErr:           false,
			wantLen:           2,
			wantNativeCreated: true,
			wantEmbedUpdate:   true,
			setup: func(f *FakeRoundDiscord) {
				f.CreateRoundManager.CreateNativeEventFunc = func(ctx context.Context, guildID string, roundID sharedtypes.RoundID, title roundtypes.Title, description roundtypes.Description, startTime sharedtypes.StartTime, location roundtypes.Location, userID sharedtypes.DiscordID) (createround.CreateRoundOperationResult, error) {
					return createround.CreateRoundOperationResult{
						Success: &discordgo.GuildScheduledEvent{
							ID: "native-event-123",
						},
					}, nil
				}
				f.CreateRoundManager.SendRoundEventEmbedFunc = func(guildID string, channelID string, title roundtypes.Title, description roundtypes.Description, startTime sharedtypes.StartTime, location roundtypes.Location, creatorID sharedtypes.DiscordID, roundID sharedtypes.RoundID) (createround.CreateRoundOperationResult, error) {
					return createround.CreateRoundOperationResult{
						Success: &discordgo.Message{
							ID:        "discord-message-123",
							ChannelID: channelID,
						},
					}, nil
				}
				f.CreateRoundManager.SendRoundEventURLFunc = func(guildID string, channelID string, eventID string) (createround.CreateRoundOperationResult, error) {
					return createround.CreateRoundOperationResult{}, nil
				}
			},
		},
		{
			name: "send_round_event_embed_fails",
			payload: &roundevents.RoundCreatedPayloadV1{
				GuildID: guildID,
				BaseRoundPayload: roundtypes.BaseRoundPayload{
					RoundID:     testRoundID,
					Title:       roundtypes.Title("Test Round"),
					Description: roundtypes.Description("Test Description"),
					Location:    roundtypes.Location("Test Location"),
					StartTime:   &startTime,
					UserID:      sharedtypes.DiscordID("user_id"),
				},
				ChannelID: channelID,
			},
			ctx:               context.Background(),
			wantErr:           true,
			wantLen:           0,
			wantNativeCreated: false,
			setup: func(f *FakeRoundDiscord) {
				f.CreateRoundManager.CreateNativeEventFunc = func(ctx context.Context, guildID string, roundID sharedtypes.RoundID, title roundtypes.Title, description roundtypes.Description, startTime sharedtypes.StartTime, location roundtypes.Location, userID sharedtypes.DiscordID) (createround.CreateRoundOperationResult, error) {
					return createround.CreateRoundOperationResult{
						Success: &discordgo.GuildScheduledEvent{
							ID: "native-event-123",
						},
					}, nil
				}
				f.CreateRoundManager.SendRoundEventEmbedFunc = func(guildID string, channelID string, title roundtypes.Title, description roundtypes.Description, startTime sharedtypes.StartTime, location roundtypes.Location, creatorID sharedtypes.DiscordID, roundID sharedtypes.RoundID) (createround.CreateRoundOperationResult, error) {
					return createround.CreateRoundOperationResult{}, errors.New("failed to send round event embed")
				}
			},
		},
		{
			name: "create_native_event_result_error_is_emitted",
			payload: &roundevents.RoundCreatedPayloadV1{
				GuildID: guildID,
				BaseRoundPayload: roundtypes.BaseRoundPayload{
					RoundID:     testRoundID,
					Title:       roundtypes.Title("Test Round"),
					Description: roundtypes.Description("Test Description"),
					Location:    roundtypes.Location("Test Location"),
					StartTime:   &startTime,
					UserID:      sharedtypes.DiscordID("user_id"),
				},
				ChannelID: channelID,
			},
			ctx:              context.Background(),
			wantErr:          false,
			wantLen:          2,
			wantNativeFailed: true,
			wantEmbedUpdate:  true,
			setup: func(f *FakeRoundDiscord) {
				f.CreateRoundManager.CreateNativeEventFunc = func(ctx context.Context, guildID string, roundID sharedtypes.RoundID, title roundtypes.Title, description roundtypes.Description, startTime sharedtypes.StartTime, location roundtypes.Location, userID sharedtypes.DiscordID) (createround.CreateRoundOperationResult, error) {
					return createround.CreateRoundOperationResult{Error: errors.New("discord api failed")}, nil
				}
				f.CreateRoundManager.SendRoundEventEmbedFunc = func(guildID string, channelID string, title roundtypes.Title, description roundtypes.Description, startTime sharedtypes.StartTime, location roundtypes.Location, creatorID sharedtypes.DiscordID, roundID sharedtypes.RoundID) (createround.CreateRoundOperationResult, error) {
					return createround.CreateRoundOperationResult{
						Success: &discordgo.Message{
							ID:        "discord-message-123",
							ChannelID: channelID,
						},
					}, nil
				}
			},
		},
		{
			name: "missing_guild_id",
			payload: &roundevents.RoundCreatedPayloadV1{
				GuildID: "", // Missing GuildID
				BaseRoundPayload: roundtypes.BaseRoundPayload{
					RoundID:     testRoundID,
					Title:       roundtypes.Title("Test Round"),
					Description: roundtypes.Description("Test Description"),
					Location:    roundtypes.Location("Test Location"),
					StartTime:   &startTime,
					UserID:      sharedtypes.DiscordID("user_id"),
				},
				ChannelID: channelID,
			},
			ctx:              context.Background(),
			wantErr:          true,
			wantLen:          0,
			wantNativeFailed: false,
			setup: func(f *FakeRoundDiscord) {
				f.CreateRoundManager.CreateNativeEventFunc = func(ctx context.Context, guildID string, roundID sharedtypes.RoundID, title roundtypes.Title, description roundtypes.Description, startTime sharedtypes.StartTime, location roundtypes.Location, userID sharedtypes.DiscordID) (createround.CreateRoundOperationResult, error) {
					return createround.CreateRoundOperationResult{
						Success: &discordgo.GuildScheduledEvent{
							ID: "native-event-123",
						},
					}, nil
				}
				f.CreateRoundManager.SendRoundEventEmbedFunc = func(guildID string, channelID string, title roundtypes.Title, description roundtypes.Description, startTime sharedtypes.StartTime, location roundtypes.Location, creatorID sharedtypes.DiscordID, roundID sharedtypes.RoundID) (createround.CreateRoundOperationResult, error) {
					return createround.CreateRoundOperationResult{
						Success: &discordgo.Message{
							ID:        "discord-message-123",
							ChannelID: channelID,
						},
					}, nil
				}
				f.CreateRoundManager.SendRoundEventURLFunc = func(guildID string, channelID string, eventID string) (createround.CreateRoundOperationResult, error) {
					return createround.CreateRoundOperationResult{}, nil
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeRoundDiscord := &FakeRoundDiscord{}
			if tt.setup != nil {
				tt.setup(fakeRoundDiscord)
			}
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

			h := NewRoundHandlers(
				mockLogger,
				&config.Config{},
				nil,
				fakeRoundDiscord,
				nil,
			)

			got, err := h.HandleRoundCreated(tt.ctx, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundCreated() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != tt.wantLen {
				t.Errorf("HandleRoundCreated() got %d results, want %d", len(got), tt.wantLen)
				return
			}

			foundNativeCreated := false
			foundNativeFailed := false
			foundEmbedUpdate := false
			for _, result := range got {
				switch result.Topic {
				case roundevents.NativeEventCreatedV1:
					foundNativeCreated = true
				case roundevents.NativeEventCreateFailedV1:
					foundNativeFailed = true
				case roundevents.RoundEventMessageIDUpdateV1:
					foundEmbedUpdate = true
				}
			}

			if foundNativeCreated != tt.wantNativeCreated {
				t.Errorf("HandleRoundCreated() NativeEventCreatedV1 present = %v, want %v", foundNativeCreated, tt.wantNativeCreated)
			}
			if foundNativeFailed != tt.wantNativeFailed {
				t.Errorf("HandleRoundCreated() NativeEventCreateFailedV1 present = %v, want %v", foundNativeFailed, tt.wantNativeFailed)
			}
			if foundEmbedUpdate != tt.wantEmbedUpdate {
				t.Errorf("HandleRoundCreated() RoundEventMessageIDUpdateV1 present = %v, want %v", foundEmbedUpdate, tt.wantEmbedUpdate)
			}
		})
	}
}

func TestRoundHandlers_HandleRoundCreated_RetriesEmbedAfterNativeEventSuccess(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	parsedTime, _ := time.Parse(time.RFC3339, "2024-01-01T12:00:00Z")
	startTime := sharedtypes.StartTime(parsedTime)
	guildID := sharedtypes.GuildID("123456789")
	channelID := "1344376922888474625"

	nativeMap := rounddiscord.NewNativeEventMap()
	createNativeCalls := 0
	sendEmbedCalls := 0

	fakeRoundDiscord := &FakeRoundDiscord{
		NativeEventMap: nativeMap,
	}
	fakeRoundDiscord.CreateRoundManager.CreateNativeEventFunc = func(ctx context.Context, guildID string, roundID sharedtypes.RoundID, title roundtypes.Title, description roundtypes.Description, startTime sharedtypes.StartTime, location roundtypes.Location, userID sharedtypes.DiscordID) (createround.CreateRoundOperationResult, error) {
		createNativeCalls++
		return createround.CreateRoundOperationResult{
			Success: &discordgo.GuildScheduledEvent{
				ID: "native-event-123",
			},
		}, nil
	}
	fakeRoundDiscord.CreateRoundManager.SendRoundEventEmbedFunc = func(guildID string, channelID string, title roundtypes.Title, description roundtypes.Description, startTime sharedtypes.StartTime, location roundtypes.Location, creatorID sharedtypes.DiscordID, roundID sharedtypes.RoundID) (createround.CreateRoundOperationResult, error) {
		sendEmbedCalls++
		if sendEmbedCalls == 1 {
			return createround.CreateRoundOperationResult{}, errors.New("discord transient failure")
		}
		return createround.CreateRoundOperationResult{
			Success: &discordgo.Message{
				ID:        "discord-message-123",
				ChannelID: channelID,
			},
		}, nil
	}

	h := NewRoundHandlers(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		&config.Config{},
		nil,
		fakeRoundDiscord,
		nil,
	)

	payload := &roundevents.RoundCreatedPayloadV1{
		GuildID: guildID,
		BaseRoundPayload: roundtypes.BaseRoundPayload{
			RoundID:     testRoundID,
			Title:       roundtypes.Title("Retry Round"),
			Description: roundtypes.Description("Retry Description"),
			Location:    roundtypes.Location("Retry Location"),
			StartTime:   &startTime,
			UserID:      sharedtypes.DiscordID("user_id"),
		},
		ChannelID: channelID,
	}

	firstResults, firstErr := h.HandleRoundCreated(context.Background(), payload)
	if firstErr == nil {
		t.Fatalf("first HandleRoundCreated() error = nil, want non-nil")
	}
	if len(firstResults) != 0 {
		t.Fatalf("first HandleRoundCreated() results len = %d, want 0", len(firstResults))
	}
	if createNativeCalls != 1 {
		t.Fatalf("first HandleRoundCreated() CreateNativeEvent calls = %d, want 1", createNativeCalls)
	}

	secondResults, secondErr := h.HandleRoundCreated(context.Background(), payload)
	if secondErr != nil {
		t.Fatalf("second HandleRoundCreated() error = %v, want nil", secondErr)
	}
	if createNativeCalls != 1 {
		t.Fatalf("second HandleRoundCreated() CreateNativeEvent calls = %d, want 1 (reuse native map)", createNativeCalls)
	}
	if sendEmbedCalls != 2 {
		t.Fatalf("HandleRoundCreated() SendRoundEventEmbed calls = %d, want 2", sendEmbedCalls)
	}

	foundNativeCreated := false
	foundEmbedUpdate := false
	for _, result := range secondResults {
		switch result.Topic {
		case roundevents.NativeEventCreatedV1:
			foundNativeCreated = true
		case roundevents.RoundEventMessageIDUpdateV1:
			foundEmbedUpdate = true
		}
	}
	if !foundNativeCreated {
		t.Fatalf("second HandleRoundCreated() expected NativeEventCreatedV1 result")
	}
	if !foundEmbedUpdate {
		t.Fatalf("second HandleRoundCreated() expected RoundEventMessageIDUpdateV1 result")
	}

	if mappedEventID, ok := nativeMap.LookupByRoundID(testRoundID); !ok || mappedEventID != "native-event-123" {
		t.Fatalf("NativeEventMap.LookupByRoundID() = (%s, %v), want (native-event-123, true)", mappedEventID, ok)
	}
}

func TestRoundHandlers_HandleRoundCreated_ReusesPendingNativeEvent(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	parsedTime, _ := time.Parse(time.RFC3339, "2024-01-01T12:00:00Z")
	startTime := sharedtypes.StartTime(parsedTime)
	guildID := sharedtypes.GuildID("123456789")

	pendingMap := rounddiscord.NewPendingNativeEventMap()
	pendingMap.Store(string(guildID)+"|Pending Round", "existing-native-event-123")
	nativeMap := rounddiscord.NewNativeEventMap()

	session := discord.NewFakeSession()
	editCalled := false
	session.GuildScheduledEventEditFunc = func(guildID, eventID string, params *discordgo.GuildScheduledEventParams, options ...discordgo.RequestOption) (*discordgo.GuildScheduledEvent, error) {
		editCalled = true
		if eventID != "existing-native-event-123" {
			t.Fatalf("GuildScheduledEventEdit() eventID = %s, want existing-native-event-123", eventID)
		}
		if params == nil || params.Description == "" {
			t.Fatalf("GuildScheduledEventEdit() missing description")
		}
		return &discordgo.GuildScheduledEvent{ID: eventID}, nil
	}

	createNativeCalled := false
	fakeRoundDiscord := &FakeRoundDiscord{
		PendingNativeEventMap: pendingMap,
		NativeEventMap:        nativeMap,
		Session:               session,
	}
	fakeRoundDiscord.CreateRoundManager.CreateNativeEventFunc = func(ctx context.Context, guildID string, roundID sharedtypes.RoundID, title roundtypes.Title, description roundtypes.Description, startTime sharedtypes.StartTime, location roundtypes.Location, userID sharedtypes.DiscordID) (createround.CreateRoundOperationResult, error) {
		createNativeCalled = true
		return createround.CreateRoundOperationResult{}, nil
	}
	fakeRoundDiscord.CreateRoundManager.SendRoundEventEmbedFunc = func(guildID string, channelID string, title roundtypes.Title, description roundtypes.Description, startTime sharedtypes.StartTime, location roundtypes.Location, creatorID sharedtypes.DiscordID, roundID sharedtypes.RoundID) (createround.CreateRoundOperationResult, error) {
		return createround.CreateRoundOperationResult{
			Success: &discordgo.Message{
				ID:        "discord-message-123",
				ChannelID: channelID,
			},
		}, nil
	}

	h := NewRoundHandlers(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		&config.Config{},
		nil,
		fakeRoundDiscord,
		nil,
	)

	results, err := h.HandleRoundCreated(context.Background(), &roundevents.RoundCreatedPayloadV1{
		GuildID: guildID,
		BaseRoundPayload: roundtypes.BaseRoundPayload{
			RoundID:     testRoundID,
			Title:       roundtypes.Title("Pending Round"),
			Description: roundtypes.Description("Pending Description"),
			Location:    roundtypes.Location("Pending Location"),
			StartTime:   &startTime,
			UserID:      sharedtypes.DiscordID("user_id"),
		},
		ChannelID: "channel-id",
	})
	if err != nil {
		t.Fatalf("HandleRoundCreated() error = %v", err)
	}
	if createNativeCalled {
		t.Fatalf("HandleRoundCreated() unexpectedly called CreateNativeEvent for pending native event")
	}
	if !editCalled {
		t.Fatalf("HandleRoundCreated() expected GuildScheduledEventEdit to be called for pending native event")
	}

	foundNativeCreated := false
	foundJoinRequested := false
	for _, result := range results {
		switch result.Topic {
		case roundevents.NativeEventCreatedV1:
			foundNativeCreated = true
		case roundevents.RoundParticipantJoinRequestedV1:
			foundJoinRequested = true
		}
	}
	if !foundNativeCreated {
		t.Fatalf("HandleRoundCreated() expected NativeEventCreatedV1 result for pending native event")
	}
	if !foundJoinRequested {
		t.Fatalf("HandleRoundCreated() expected RoundParticipantJoinRequestedV1 for event creator")
	}

	if mappedEventID, ok := nativeMap.LookupByRoundID(testRoundID); !ok || mappedEventID != "existing-native-event-123" {
		t.Fatalf("NativeEventMap.LookupByRoundID() = (%s, %v), want (existing-native-event-123, true)", mappedEventID, ok)
	}
}

func TestRoundHandlers_HandleRoundCreationFailed(t *testing.T) {
	tests := []struct {
		name    string
		payload *roundevents.RoundCreationFailedPayloadV1
		ctx     context.Context
		wantErr bool
		setup   func(*FakeRoundDiscord)
	}{
		{
			name: "successful_round_creation_failed",
			payload: &roundevents.RoundCreationFailedPayloadV1{
				ErrorMessage: "Test Reason",
			},
			ctx:     context.WithValue(context.Background(), "correlation_id", "correlation_id"),
			wantErr: false,
			setup: func(f *FakeRoundDiscord) {
				f.CreateRoundManager.UpdateInteractionResponseWithRetryButtonFunc = func(ctx context.Context, correlationID, message string) (createround.CreateRoundOperationResult, error) {
					if correlationID != "correlation_id" {
						return createround.CreateRoundOperationResult{}, errors.New("unexpected correlation_id")
					}
					return createround.CreateRoundOperationResult{}, nil
				}
			},
		},
		{
			name: "update_interaction_response_fails",
			payload: &roundevents.RoundCreationFailedPayloadV1{
				ErrorMessage: "Test Reason",
			},
			ctx:     context.WithValue(context.Background(), "correlation_id", "correlation_id"),
			wantErr: true,
			setup: func(f *FakeRoundDiscord) {
				f.CreateRoundManager.UpdateInteractionResponseWithRetryButtonFunc = func(ctx context.Context, correlationID, message string) (createround.CreateRoundOperationResult, error) {
					return createround.CreateRoundOperationResult{}, errors.New("failed to update interaction response")
				}
			},
		},
		{
			name: "missing_correlation_id",
			payload: &roundevents.RoundCreationFailedPayloadV1{
				ErrorMessage: "Test Reason",
			},
			ctx:     context.Background(), // No correlation_id in context
			wantErr: false,
			setup: func(f *FakeRoundDiscord) {
				// No setup needed
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeRoundDiscord := &FakeRoundDiscord{}
			if tt.setup != nil {
				tt.setup(fakeRoundDiscord)
			}
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

			h := NewRoundHandlers(
				mockLogger,
				&config.Config{},
				nil,
				fakeRoundDiscord,
				nil,
			)

			got, err := h.HandleRoundCreationFailed(tt.ctx, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundCreationFailed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Handler should always return nil results (side-effect only)
			if len(got) > 0 {
				t.Errorf("HandleRoundCreationFailed() expected nil or empty results, got %v", got)
			}
		})
	}
}

func TestRoundHandlers_HandleRoundValidationFailed(t *testing.T) {
	tests := []struct {
		name    string
		payload *roundevents.RoundValidationFailedPayloadV1
		ctx     context.Context
		wantErr bool
		setup   func(*FakeRoundDiscord)
	}{
		{
			name: "successful_round_validation_failed",
			payload: &roundevents.RoundValidationFailedPayloadV1{
				ErrorMessages: []string{"Error 1", "Error 2"},
			},
			ctx:     context.WithValue(context.Background(), "correlation_id", "correlation_id"),
			wantErr: false,
			setup: func(f *FakeRoundDiscord) {
				f.CreateRoundManager.UpdateInteractionResponseWithRetryButtonFunc = func(ctx context.Context, correlationID, message string) (createround.CreateRoundOperationResult, error) {
					return createround.CreateRoundOperationResult{}, nil
				}
			},
		},
		{
			name: "update_interaction_response_fails",
			payload: &roundevents.RoundValidationFailedPayloadV1{
				ErrorMessages: []string{"Error 1", "Error 2"},
			},
			ctx:     context.WithValue(context.Background(), "correlation_id", "correlation_id"),
			wantErr: true,
			setup: func(f *FakeRoundDiscord) {
				f.CreateRoundManager.UpdateInteractionResponseWithRetryButtonFunc = func(ctx context.Context, correlationID, message string) (createround.CreateRoundOperationResult, error) {
					return createround.CreateRoundOperationResult{}, errors.New("failed to update interaction response")
				}
			},
		},
		{
			name: "missing_correlation_id",
			payload: &roundevents.RoundValidationFailedPayloadV1{
				ErrorMessages: []string{"Error A", "Error B"},
			},
			ctx:     context.Background(), // No correlation_id in context
			wantErr: false,
			setup: func(f *FakeRoundDiscord) {
				// No setup needed
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeRoundDiscord := &FakeRoundDiscord{}
			if tt.setup != nil {
				tt.setup(fakeRoundDiscord)
			}
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

			h := NewRoundHandlers(
				mockLogger,
				&config.Config{},
				nil,
				fakeRoundDiscord,
				nil,
			)

			got, err := h.HandleRoundValidationFailed(tt.ctx, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundValidationFailed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Handler should always return nil results (side-effect only)
			if len(got) > 0 {
				t.Errorf("HandleRoundValidationFailed() expected nil or empty results, got %v", got)
			}
		})
	}
}

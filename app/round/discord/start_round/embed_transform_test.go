package startround

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

func Test_startRoundManager_TransformRoundToScorecard(t *testing.T) {
	timePtr := func(t time.Time) *time.Time { return &t }

	fixedTime := time.Date(2025, 3, 15, 10, 0, 0, 0, time.UTC)
	testRoundID := sharedtypes.RoundID(uuid.New())

	buildExpectedComponents := func(roundID sharedtypes.RoundID) []discordgo.MessageComponent {
		return []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Enter Score",
						Style:    discordgo.PrimaryButton,
						CustomID: fmt.Sprintf("round_enter_score|%s", roundID),
						Emoji:    &discordgo.ComponentEmoji{Name: "💰"},
					},
					discordgo.Button{
						Label:    "Join Round LATE",
						Style:    discordgo.SecondaryButton,
						CustomID: fmt.Sprintf("round_join_late|%s", roundID),
						Emoji:    &discordgo.ComponentEmoji{Name: "🏃"},
					},
					discordgo.Button{
						Label:    "Upload Scorecard",
						Style:    discordgo.SuccessButton,
						CustomID: fmt.Sprintf("round_upload_scorecard|%s", roundID),
						Emoji:    &discordgo.ComponentEmoji{Name: "📋"},
					},
				},
			},
		}
	}

	tests := []struct {
		name               string
		payload            *roundevents.DiscordRoundStartPayloadV1
		expectError        bool
		expectedEmbed      *discordgo.MessageEmbed
		expectedComponents []discordgo.MessageComponent
	}{
		{
			name: "No participants",
			payload: &roundevents.DiscordRoundStartPayloadV1{
				RoundID:   testRoundID,
				Title:     "Test Round",
				Location:  (roundtypes.Location)("Test Course"),
				StartTime: (*sharedtypes.StartTime)(timePtr(fixedTime)),
			},
			expectError: false,
			expectedEmbed: &discordgo.MessageEmbed{
				Title:       "**Test Round** - Round Started",
				Description: "Round at Test Course has started!",
				Color:       0x00AA00,
				Fields: []*discordgo.MessageEmbedField{
					{Name: "📅 Started", Value: fmt.Sprintf("<t:%d:f>", fixedTime.Unix())},
					{Name: "📍 Location", Value: "Test Course"},
					{Name: "👥 Participants", Value: "*No participants*"},
				},
				Footer: &discordgo.MessageEmbedFooter{
					Text: "Round in progress. Use the buttons below to join or record your score.",
				},
			},
			expectedComponents: buildExpectedComponents(testRoundID),
		},
		{
			name: "Accepted and tentative participants",
			payload: &roundevents.DiscordRoundStartPayloadV1{
				RoundID:   testRoundID,
				Title:     "Mixed Round",
				Location:  (roundtypes.Location)("Test Course"),
				StartTime: (*sharedtypes.StartTime)(timePtr(fixedTime)),
				Participants: []roundevents.RoundParticipantV1{
					{UserID: "user-1", Response: roundtypes.ResponseAccept},
					{UserID: "user-2", Response: roundtypes.ResponseTentative},
					{UserID: "user-3", Response: roundtypes.ResponseAccept},
				},
			},
			expectError: false,
			expectedEmbed: &discordgo.MessageEmbed{
				Title:       "**Mixed Round** - Round Started",
				Description: "Round at Test Course has started!",
				Color:       0x00AA00,
				Fields: []*discordgo.MessageEmbedField{
					{Name: "📅 Started", Value: fmt.Sprintf("<t:%d:f>", fixedTime.Unix())},
					{Name: "📍 Location", Value: "Test Course"},
					{Name: "👥 Participants", Value: "<@user-1> — Score: --\n<@user-2> — Score: --\n<@user-3> — Score: --"},
				},
				Footer: &discordgo.MessageEmbedFooter{
					Text: "Round in progress. Use the buttons below to join or record your score.",
				},
			},
			expectedComponents: buildExpectedComponents(testRoundID),
		},
		{
			name: "Participants with tag numbers",
			payload: &roundevents.DiscordRoundStartPayloadV1{
				RoundID:   testRoundID,
				Title:     "Tagged Round",
				Location:  (roundtypes.Location)("Test Course"),
				StartTime: (*sharedtypes.StartTime)(timePtr(fixedTime)),
				Participants: []roundevents.RoundParticipantV1{
					{UserID: "user-1", TagNumber: func() *sharedtypes.TagNumber { t := sharedtypes.TagNumber(1); return &t }(), Response: roundtypes.ResponseAccept},
					{UserID: "user-2", TagNumber: func() *sharedtypes.TagNumber { t := sharedtypes.TagNumber(2); return &t }(), Response: roundtypes.ResponseAccept},
				},
			},
			expectError: false,
			expectedEmbed: &discordgo.MessageEmbed{
				Title:       "**Tagged Round** - Round Started",
				Description: "Round at Test Course has started!",
				Color:       0x00AA00,
				Fields: []*discordgo.MessageEmbedField{
					{Name: "📅 Started", Value: fmt.Sprintf("<t:%d:f>", fixedTime.Unix())},
					{Name: "📍 Location", Value: "Test Course"},
					{Name: "👥 Participants", Value: "<@user-1> Tag: 1 — Score: --\n<@user-2> Tag: 2 — Score: --"},
				},
				Footer: &discordgo.MessageEmbedFooter{
					Text: "Round in progress. Use the buttons below to join or record your score.",
				},
			},
			expectedComponents: buildExpectedComponents(testRoundID),
		},
		{
			name: "Nil StartTime triggers error",
			payload: &roundevents.DiscordRoundStartPayloadV1{
				RoundID:   testRoundID,
				Title:     "Error Round",
				Location:  (roundtypes.Location)("Test Course"),
				StartTime: nil,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srm := &startRoundManager{
				logger: loggerfrolfbot.NoOpLogger,
				operationWrapper: func(ctx context.Context, _ string, fn func(ctx context.Context) (StartRoundOperationResult, error)) (StartRoundOperationResult, error) {
					return fn(ctx)
				},
			}

			result, err := srm.TransformRoundToScorecard(context.Background(), tt.payload, nil)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			successData, ok := result.Success.(struct {
				Embed      *discordgo.MessageEmbed
				Components []discordgo.MessageComponent
			})
			if !ok {
				t.Fatalf("Failed to cast result.Success to expected type")
			}

			gotEmbed := successData.Embed
			gotComponents := successData.Components

			// Ignore timestamp for comparison
			gotEmbed.Timestamp = ""

			if !reflect.DeepEqual(gotEmbed, tt.expectedEmbed) {
				t.Errorf("Embed mismatch:\nGot: %+v\nWant: %+v", gotEmbed, tt.expectedEmbed)
			}
			if !reflect.DeepEqual(gotComponents, tt.expectedComponents) {
				t.Errorf("Components mismatch:\nGot: %+v\nWant: %+v", gotComponents, tt.expectedComponents)
			}
		})
	}
}

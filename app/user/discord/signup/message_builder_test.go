package signup

import (
	"context"
	"testing"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

func TestBuildUserSignupRequestMessage_Success(t *testing.T) {
	payload := userevents.UserSignupRequestPayload{
		GuildID: sharedtypes.GuildID("123"),
		UserID:  sharedtypes.DiscordID("456"),
	}
	msg, err := BuildUserSignupRequestMessage(context.TODO(), payload, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Metadata.Get("guild_id") != string(payload.GuildID) {
		t.Errorf("guild_id metadata mismatch")
	}
	if msg.Metadata.Get("user_id") != string(payload.UserID) {
		t.Errorf("user_id metadata mismatch")
	}
	if msg.Metadata.Get("message_type") == "" {
		t.Errorf("message_type missing")
	}
	if msg.Metadata.Get("correlation_id") == "" {
		t.Errorf("correlation_id missing")
	}
}

func TestBuildUserSignupRequestMessage_Validation(t *testing.T) {
	// Missing both
	if _, err := BuildUserSignupRequestMessage(context.TODO(), userevents.UserSignupRequestPayload{}, nil); err == nil {
		t.Fatalf("expected error for missing guild/user ids")
	}
	// Only user missing
	if _, err := BuildUserSignupRequestMessage(context.TODO(), userevents.UserSignupRequestPayload{GuildID: sharedtypes.GuildID("123")}, nil); err == nil {
		t.Fatalf("expected error for missing user id")
	}
}

package gateway

import (
	"testing"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
)

func TestParseRoundIDFromDescription(t *testing.T) {
	validUUID := uuid.New().String()

	tests := []struct {
		name        string
		description string
		wantOK      bool
		wantID      string
	}{
		{
			name:        "valid description with RoundID footer",
			description: "Some round description\n---\nRoundID: " + validUUID,
			wantOK:      true,
			wantID:      validUUID,
		},
		{
			name:        "description without RoundID",
			description: "Some round description without footer",
			wantOK:      false,
		},
		{
			name:        "empty description",
			description: "",
			wantOK:      false,
		},
		{
			name:        "RoundID with invalid UUID",
			description: "Description\n---\nRoundID: not-a-uuid",
			wantOK:      false,
		},
		{
			name:        "RoundID in middle of description",
			description: "Start\nRoundID: " + validUUID + "\nEnd",
			wantOK:      true,
			wantID:      validUUID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roundID, ok := parseRoundIDFromDescription(tt.description)
			if ok != tt.wantOK {
				t.Errorf("parseRoundIDFromDescription() ok = %v, want %v", ok, tt.wantOK)
				return
			}
			if tt.wantOK {
				expectedID := sharedtypes.RoundID(uuid.MustParse(tt.wantID))
				if roundID != expectedID {
					t.Errorf("parseRoundIDFromDescription() roundID = %v, want %v", roundID, expectedID)
				}
			}
		})
	}
}

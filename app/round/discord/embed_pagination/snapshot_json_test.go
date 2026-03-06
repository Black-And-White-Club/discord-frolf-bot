package embedpagination

import (
	"encoding/json"
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestSnapshotJSONRoundTrip(t *testing.T) {
	__codexTDCases := []struct {
		name string
	}{
		{name: "default"},
	}

	for _, __codexTDCase := range __codexTDCases {
		t.Run(__codexTDCase.name, func(t *testing.T) {
			t.Parallel()

			original := &Snapshot{
				MessageID:   "123",
				Kind:        SnapshotKindLines,
				Title:       "Scorecard",
				Description: "Round details",
				Color:       0x123456,
				Timestamp:   "2026-02-22T12:00:00Z",
				Footer: &discordgo.MessageEmbedFooter{
					Text: "Footer",
				},
				StaticFields: []*discordgo.MessageEmbedField{
					{Name: "Date", Value: "Soon"},
				},
				ParticipantFieldName: "Participants",
				LineItems:            []string{"1. <@111>", "2. <@222>"},
				BaseComponents: []discordgo.MessageComponent{
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.Button{
								Label:    "Join",
								Style:    discordgo.PrimaryButton,
								CustomID: "join_btn",
							},
						},
					},
				},
				CurrentPage: 2,
			}

			raw, err := json.Marshal(original)
			if err != nil {
				t.Fatalf("marshal snapshot: %v", err)
			}

			var decoded Snapshot
			if err := json.Unmarshal(raw, &decoded); err != nil {
				t.Fatalf("unmarshal snapshot: %v", err)
			}

			if decoded.MessageID != original.MessageID {
				t.Fatalf("MessageID = %q, want %q", decoded.MessageID, original.MessageID)
			}
			if decoded.CurrentPage != original.CurrentPage {
				t.Fatalf("CurrentPage = %d, want %d", decoded.CurrentPage, original.CurrentPage)
			}
			if len(decoded.BaseComponents) != 1 {
				t.Fatalf("BaseComponents len = %d, want 1", len(decoded.BaseComponents))
			}

			row, ok := decoded.BaseComponents[0].(discordgo.ActionsRow)
			if !ok {
				t.Fatalf("decoded component type = %T, want discordgo.ActionsRow", decoded.BaseComponents[0])
			}
			if len(row.Components) != 1 {
				t.Fatalf("row components len = %d, want 1", len(row.Components))
			}

			button, ok := row.Components[0].(discordgo.Button)
			if !ok {
				t.Fatalf("decoded row component type = %T, want discordgo.Button", row.Components[0])
			}
			if button.CustomID != "join_btn" {
				t.Fatalf("button custom ID = %q, want %q", button.CustomID, "join_btn")
			}
		})
	}
}

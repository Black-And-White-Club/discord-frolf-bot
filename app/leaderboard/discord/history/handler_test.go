package history

import "testing"

func TestFormatHistoryMemberLabel(t *testing.T) {
	t.Run("mention-formatted IDs are converted to plain text", func(t *testing.T) {
		got := formatHistoryMemberLabel("<@!839877196898238526>")
		if got != "839877196898238526" {
			t.Fatalf("formatHistoryMemberLabel() = %q, want %q", got, "839877196898238526")
		}
	})

	t.Run("raw handles are rendered without the @ prefix", func(t *testing.T) {
		got := formatHistoryMemberLabel("@farrmich")
		if got != "farrmich" {
			t.Fatalf("formatHistoryMemberLabel() = %q, want %q", got, "farrmich")
		}
	})

	t.Run("plain names are rendered without the @ prefix", func(t *testing.T) {
		got := formatHistoryMemberLabel("Bobby Waldron")
		if got != "Bobby Waldron" {
			t.Fatalf("formatHistoryMemberLabel() = %q, want %q", got, "Bobby Waldron")
		}
	})
}

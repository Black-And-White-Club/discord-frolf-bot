package udisc

import (
	"context"
	"testing"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	"github.com/bwmarrin/discordgo"
)

type fakeUDiscManager struct {
	calls int
}

func (m *fakeUDiscManager) HandleSetUDiscNameCommand(ctx context.Context, i *discordgo.InteractionCreate) (UDiscOperationResult, error) {
	m.calls++
	return UDiscOperationResult{Success: "ok"}, nil
}

func TestRegisterUDiscInteractions_registersSlashCommandHandler(t *testing.T) {
	reg := interactions.NewRegistry()
	mgr := &fakeUDiscManager{}

	RegisterUDiscInteractions(reg, mgr)

	i := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type: discordgo.InteractionApplicationCommand,
		Data: discordgo.ApplicationCommandInteractionData{Name: "set-udisc-name"},
	}}
	reg.HandleInteraction(&discordgo.Session{}, i)

	if mgr.calls != 1 {
		t.Fatalf("expected manager called once, got %d", mgr.calls)
	}
}

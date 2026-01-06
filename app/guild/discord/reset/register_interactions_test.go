package reset

import (
	"context"
	"testing"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

// localResetManager is a minimal stub implementing ResetManager to avoid import cycles in tests.
type localResetManager struct {
	commandCalled       int
	confirmButtonCalled int
	cancelButtonCalled  int
}

func (l *localResetManager) HandleResetCommand(ctx context.Context, i *discordgo.InteractionCreate) error {
	l.commandCalled++
	return nil
}

func (l *localResetManager) HandleResetConfirmButton(ctx context.Context, i *discordgo.InteractionCreate) error {
	l.confirmButtonCalled++
	return nil
}

func (l *localResetManager) HandleResetCancelButton(ctx context.Context, i *discordgo.InteractionCreate) error {
	l.cancelButtonCalled++
	return nil
}

func TestRegisterHandlers_WiresManager(t *testing.T) {
	reg := interactions.NewRegistry()
	lm := &localResetManager{}

	RegisterHandlers(reg, lm)

	// 1) Slash command: frolf-reset
	slash := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		ID:   uuid.New().String(),
		Type: discordgo.InteractionApplicationCommand,
		Data: discordgo.ApplicationCommandInteractionData{Name: "frolf-reset"},
	}}
	reg.HandleInteraction(&discordgo.Session{}, slash)

	// 2) Confirm button: frolf_reset_confirm
	confirmButton := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		ID:   uuid.New().String(),
		Type: discordgo.InteractionMessageComponent,
		Data: discordgo.MessageComponentInteractionData{CustomID: "frolf_reset_confirm"},
	}}
	reg.HandleInteraction(&discordgo.Session{}, confirmButton)

	// 3) Cancel button: frolf_reset_cancel
	cancelButton := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		ID:   uuid.New().String(),
		Type: discordgo.InteractionMessageComponent,
		Data: discordgo.MessageComponentInteractionData{CustomID: "frolf_reset_cancel"},
	}}
	reg.HandleInteraction(&discordgo.Session{}, cancelButton)

	if lm.commandCalled != 1 || lm.confirmButtonCalled != 1 || lm.cancelButtonCalled != 1 {
		t.Fatalf("expected handlers called once each, got command=%d confirm=%d cancel=%d",
			lm.commandCalled, lm.confirmButtonCalled, lm.cancelButtonCalled)
	}
}

func TestRegisterHandlers_CommandOnly(t *testing.T) {
	reg := interactions.NewRegistry()
	lm := &localResetManager{}

	RegisterHandlers(reg, lm)

	// Only call the command
	slash := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		ID:   uuid.New().String(),
		Type: discordgo.InteractionApplicationCommand,
		Data: discordgo.ApplicationCommandInteractionData{Name: "frolf-reset"},
	}}
	reg.HandleInteraction(&discordgo.Session{}, slash)

	if lm.commandCalled != 1 || lm.confirmButtonCalled != 0 || lm.cancelButtonCalled != 0 {
		t.Fatalf("expected only command handler called, got command=%d confirm=%d cancel=%d",
			lm.commandCalled, lm.confirmButtonCalled, lm.cancelButtonCalled)
	}
}

func TestRegisterHandlers_ButtonsOnly(t *testing.T) {
	reg := interactions.NewRegistry()
	lm := &localResetManager{}

	RegisterHandlers(reg, lm)

	// Call both buttons without command
	confirmButton := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		ID:   uuid.New().String(),
		Type: discordgo.InteractionMessageComponent,
		Data: discordgo.MessageComponentInteractionData{CustomID: "frolf_reset_confirm"},
	}}
	reg.HandleInteraction(&discordgo.Session{}, confirmButton)

	cancelButton := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		ID:   uuid.New().String(),
		Type: discordgo.InteractionMessageComponent,
		Data: discordgo.MessageComponentInteractionData{CustomID: "frolf_reset_cancel"},
	}}
	reg.HandleInteraction(&discordgo.Session{}, cancelButton)

	if lm.commandCalled != 0 || lm.confirmButtonCalled != 1 || lm.cancelButtonCalled != 1 {
		t.Fatalf("expected only button handlers called, got command=%d confirm=%d cancel=%d",
			lm.commandCalled, lm.confirmButtonCalled, lm.cancelButtonCalled)
	}
}

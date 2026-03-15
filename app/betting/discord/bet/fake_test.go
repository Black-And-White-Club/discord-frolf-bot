package bet

import (
	"context"

	"github.com/bwmarrin/discordgo"
)

// FakeBetManager is a test double for BetManager.
type FakeBetManager struct {
	HandleBetCommandFunc func(ctx context.Context, i *discordgo.InteractionCreate)
	calls                []string
}

func (f *FakeBetManager) HandleBetCommand(ctx context.Context, i *discordgo.InteractionCreate) {
	f.calls = append(f.calls, "HandleBetCommand")
	if f.HandleBetCommandFunc != nil {
		f.HandleBetCommandFunc(ctx, i)
	}
}

// Called returns the list of method names invoked on the fake.
func (f *FakeBetManager) Called() []string {
	out := make([]string, len(f.calls))
	copy(out, f.calls)
	return out
}

var _ BetManager = (*FakeBetManager)(nil)

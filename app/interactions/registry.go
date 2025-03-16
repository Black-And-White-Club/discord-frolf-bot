// interactions/registry.go
package interactions

import (
	"context"
	"log/slog"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type Registry struct {
	handlers map[string]func(ctx context.Context, i *discordgo.InteractionCreate)
}

func NewRegistry() *Registry {
	return &Registry{
		handlers: make(map[string]func(ctx context.Context, i *discordgo.InteractionCreate)),
	}
}

func (r *Registry) RegisterHandler(id string, handler func(ctx context.Context, i *discordgo.InteractionCreate)) {
	r.handlers[id] = handler
}

func (r *Registry) HandleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()
	var id string

	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		id = i.ApplicationCommandData().Name
	case discordgo.InteractionMessageComponent:
		id = i.MessageComponentData().CustomID
	case discordgo.InteractionModalSubmit:
		// Extract modal submission data
		modalData := i.ModalSubmitData()
		if modalData.CustomID == "" {
			slog.Error("❌ Modal submission data is invalid: CustomID is empty")
			return
		}
		id = modalData.CustomID
	}

	if handler, ok := r.handlers[id]; ok {
		handler(ctx, i)
		return
	}

	for key, handler := range r.handlers {
		if strings.HasPrefix(id, key) {
			handler(ctx, i)
			return
		}
	}
}

// app/user/watermill/handlers/user/handlers.go
package userhandlers

import (
	"context"
	"log/slog"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	userdiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/user/discord"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	discorduserevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/user"
	shareduserevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/user"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	"go.opentelemetry.io/otel/trace"
)

// Handler defines the interface for user-related event handlers.
type Handler interface {
	HandleUserCreated(ctx context.Context, payload *userevents.UserCreatedPayloadV1) ([]handlerwrapper.Result, error)
	HandleUserCreationFailed(ctx context.Context, payload *userevents.UserCreationFailedPayloadV1) ([]handlerwrapper.Result, error)
	HandleAddRole(ctx context.Context, payload *discorduserevents.AddRolePayloadV1) ([]handlerwrapper.Result, error)
	HandleRoleAdded(ctx context.Context, payload *discorduserevents.RoleAddedPayloadV1) ([]handlerwrapper.Result, error)
	HandleRoleAdditionFailed(ctx context.Context, payload *discorduserevents.RoleAdditionFailedPayloadV1) ([]handlerwrapper.Result, error)
	HandleRoleUpdateCommand(ctx context.Context, payload *discorduserevents.RoleUpdateCommandPayloadV1) ([]handlerwrapper.Result, error)
	HandleRoleUpdateButtonPress(ctx context.Context, payload *shareduserevents.RoleUpdateButtonPressPayloadV1) ([]handlerwrapper.Result, error)
	HandleRoleUpdated(ctx context.Context, payload *userevents.UserRoleUpdatedPayloadV1) ([]handlerwrapper.Result, error)
	HandleRoleUpdateFailed(ctx context.Context, payload *userevents.UserRoleUpdateFailedPayloadV1) ([]handlerwrapper.Result, error)
}

// UserHandlers handles user-related events.
type UserHandlers struct {
	logger      *slog.Logger
	config      *config.Config
	helper      utils.Helpers
	userDiscord userdiscord.UserDiscordInterface
	interactionStore storage.ISInterface[any]
	guildConfigCache storage.ISInterface[storage.GuildConfig]
	tracer      trace.Tracer
	metrics     discordmetrics.DiscordMetrics
}

// NewUserHandlers creates a new UserHandlers struct.
func NewUserHandlers(
	logger *slog.Logger,
	config *config.Config,
	helpers utils.Helpers,
	userDiscord userdiscord.UserDiscordInterface,
	interactionStore storage.ISInterface[any],
	guildConfigCache storage.ISInterface[storage.GuildConfig],
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
) Handler {
	return &UserHandlers{
		logger:      logger,
		config:      config,
		helper:      helpers,
		userDiscord: userDiscord,
		interactionStore: interactionStore,
		guildConfigCache: guildConfigCache,
		tracer:      tracer,
		metrics:     metrics,
	}
}

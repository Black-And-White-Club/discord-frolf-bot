package role

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/bwmarrin/discordgo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// RoleManager defines the interface for role operations.
type RoleManager interface {
	AddRoleToUser(ctx context.Context, guildID string, userID sharedtypes.DiscordID, roleID string) (RoleOperationResult, error)
	EditRoleUpdateResponse(ctx context.Context, correlationID string, content string) (RoleOperationResult, error)
	HandleRoleRequestCommand(ctx context.Context, i *discordgo.InteractionCreate) (RoleOperationResult, error)
	HandleRoleButtonPress(ctx context.Context, i *discordgo.InteractionCreate) (RoleOperationResult, error)
	HandleRoleCancelButton(ctx context.Context, i *discordgo.InteractionCreate) (RoleOperationResult, error)
	RespondToRoleRequest(ctx context.Context, interactionID, interactionToken string, targetUserID sharedtypes.DiscordID) (RoleOperationResult, error)
	RespondToRoleButtonPress(ctx context.Context, interactionID, interactionToken string, requesterID sharedtypes.DiscordID, selectedRole string, targetUserID sharedtypes.DiscordID) (RoleOperationResult, error)
}

type roleManager struct {
	session             discord.Session
	publisher           eventbus.EventBus
	logger              *slog.Logger
	helper              utils.Helpers
	config              *config.Config // Deprecated: use guildConfigResolver for per-guild config
	guildConfigResolver guildconfig.GuildConfigResolver
	interactionStore    storage.ISInterface
	tracer              trace.Tracer
	metrics             discordmetrics.DiscordMetrics
	operationWrapper    func(ctx context.Context, opName string, fn func(ctx context.Context) (RoleOperationResult, error)) (RoleOperationResult, error)
}

func NewRoleManager(
	session discord.Session,
	publisher eventbus.EventBus,
	logger *slog.Logger,
	helper utils.Helpers,
	config *config.Config, // Deprecated: use guildConfigResolver for per-guild config
	guildConfigResolver guildconfig.GuildConfigResolver,
	interactionStore storage.ISInterface,
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
) (RoleManager, error) {
	if logger != nil {
		logger.InfoContext(context.Background(), "Creating RoleManager")
	}

	rm := &roleManager{
		session:             session,
		publisher:           publisher,
		logger:              logger,
		helper:              helper,
		config:              config, // Deprecated
		guildConfigResolver: guildConfigResolver,
		interactionStore:    interactionStore,
		tracer:              tracer,
		metrics:             metrics,
		operationWrapper: func(ctx context.Context, opName string, fn func(ctx context.Context) (RoleOperationResult, error)) (RoleOperationResult, error) {
			return wrapRoleOperation(ctx, opName, fn, logger, tracer, metrics)
		},
	}

	return rm, nil
}

// operation wrapper
func wrapRoleOperation(
	ctx context.Context,
	operationName string,
	fn func(ctx context.Context) (RoleOperationResult, error),
	logger *slog.Logger,
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
) (result RoleOperationResult, err error) {
	if fn == nil {
		return RoleOperationResult{}, errors.New("operation function is nil")
	}

	// Handle nil tracer
	if tracer == nil {
		tracer = noop.NewTracerProvider().Tracer("noop")
	}

	ctx, span := tracer.Start(ctx, operationName, trace.WithAttributes(
		attribute.String("operation", operationName),
	))
	defer span.End()

	start := time.Now()

	defer func() {
		duration := time.Since(start)
		if logger != nil {
			logger.InfoContext(ctx, fmt.Sprintf("Completed %s", operationName),
				attr.String("duration_sec", fmt.Sprintf("%.2f", duration.Seconds())),
			)
		}
		if metrics != nil {
			metrics.RecordAPIRequestDuration(ctx, operationName, duration)
		}
	}()

	// Use named return parameters (result, err) so we can modify them in the defer
	defer func() {
		if r := recover(); r != nil {
			// Set the err return value when a panic occurs
			err = fmt.Errorf("panic in %s: %v", operationName, r)
			if logger != nil {
				logger.ErrorContext(ctx, "Recovered from panic", attr.Error(err))
			}
			span.RecordError(err)
			if metrics != nil {
				metrics.RecordAPIError(ctx, operationName, "panic")
			}
		}
	}()

	result, err = fn(ctx)
	if err != nil {
		wrapped := fmt.Errorf("%s operation error: %w", operationName, err)
		if logger != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("Error in %s", operationName), attr.Error(wrapped))
		}
		span.RecordError(wrapped)
		if metrics != nil {
			metrics.RecordAPIError(ctx, operationName, "operation_error")
		}
		return RoleOperationResult{}, wrapped
	}

	if result.Error != nil {
		span.RecordError(result.Error)
		if metrics != nil {
			metrics.RecordAPIError(ctx, operationName, "result_error")
		}
	} else if metrics != nil {
		metrics.RecordAPIRequest(ctx, operationName)
	}

	return result, nil
}

type RoleOperationResult struct {
	Success interface{}
	Failure interface{}
	Error   error
}

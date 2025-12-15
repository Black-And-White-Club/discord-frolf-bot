package scorecardupload

import (
	"context"
	"log/slog"
	"sync"
	"time"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// pendingUpload tracks a file upload that's expected from a user.
type pendingUpload struct {
	RoundID        sharedtypes.RoundID
	GuildID        sharedtypes.GuildID
	Notes          string
	EventMessageID string // ID of the original round embed message
	CreatedAt      time.Time
}

// ScorecardUploadManager defines the interface for scorecard upload operations.
type ScorecardUploadManager interface {
	HandleScorecardUploadButton(ctx context.Context, i *discordgo.InteractionCreate) (ScorecardUploadOperationResult, error)
	HandleScorecardUploadModalSubmit(ctx context.Context, i *discordgo.InteractionCreate) (ScorecardUploadOperationResult, error)
	HandleFileUploadMessage(s discord.Session, m *discordgo.MessageCreate)
	SendUploadError(ctx context.Context, channelID, errorMsg string) error
}

// scorecardUploadManager implements the ScorecardUploadManager interface.
type scorecardUploadManager struct {
	session          discord.Session
	publisher        eventbus.EventBus
	logger           *slog.Logger
	config           *config.Config
	tracer           trace.Tracer
	metrics          discordmetrics.DiscordMetrics
	operationWrapper func(ctx context.Context, opName string, fn func(ctx context.Context) (ScorecardUploadOperationResult, error)) (ScorecardUploadOperationResult, error)
	pendingUploads   map[string]*pendingUpload // key: "userID:channelID"
	pendingMutex     sync.RWMutex
}

// NewScorecardUploadManager creates a new ScorecardUploadManager instance.
func NewScorecardUploadManager(
	ctx context.Context,
	session discord.Session,
	publisher eventbus.EventBus,
	logger *slog.Logger,
	cfg *config.Config,
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
) ScorecardUploadManager {
	if tracer == nil {
		tracer = noop.NewTracerProvider().Tracer("")
	}

	m := &scorecardUploadManager{
		session:        session,
		publisher:      publisher,
		logger:         logger,
		config:         cfg,
		tracer:         tracer,
		metrics:        metrics,
		pendingUploads: make(map[string]*pendingUpload),
		operationWrapper: func(ctx context.Context, opName string, fn func(ctx context.Context) (ScorecardUploadOperationResult, error)) (ScorecardUploadOperationResult, error) {
			return operationWrapper(ctx, opName, fn, logger, tracer)
		},
	}

	// Start background cleanup of old pending uploads
	go m.cleanupPendingUploads(ctx)

	return m
}

// ScorecardUploadOperationResult represents the result of a scorecard upload operation.
type ScorecardUploadOperationResult struct {
	Success interface{}
	Failure interface{}
	Error   error
}

// operationWrapper handles common logging and tracing for scorecard upload operations.
func operationWrapper(
	ctx context.Context,
	opName string,
	fn func(ctx context.Context) (ScorecardUploadOperationResult, error),
	logger *slog.Logger,
	tracer trace.Tracer,
) (result ScorecardUploadOperationResult, err error) {
	ctx, span := tracer.Start(ctx, opName)
	defer span.End()

	logger.InfoContext(ctx, opName+" triggered",
		attr.String("operation", opName),
		attr.ExtractCorrelationID(ctx),
	)

	result, err = fn(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "Error in "+opName,
			attr.ExtractCorrelationID(ctx),
			attr.Error(err),
		)
		span.RecordError(err)
		return result, err
	}

	logger.InfoContext(ctx, opName+" completed successfully",
		attr.String("operation", opName),
		attr.ExtractCorrelationID(ctx),
	)
	return result, nil
}

// cleanupPendingUploads runs in background and removes stale pending uploads.
func (m *scorecardUploadManager) cleanupPendingUploads(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			m.logger.Info("Stopping pending upload cleanup", attr.Error(ctx.Err()))
			return
		case <-ticker.C:
			m.pendingMutex.Lock()
			now := time.Now()
			for key, pending := range m.pendingUploads {
				if now.Sub(pending.CreatedAt) > 5*time.Minute {
					delete(m.pendingUploads, key)
					m.logger.Info("Cleaned up stale pending upload",
						attr.String("key", key),
						attr.String("round_id", pending.RoundID.String()),
					)
				}
			}
			m.pendingMutex.Unlock()
		}
	}
}

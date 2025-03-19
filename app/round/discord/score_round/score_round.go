package scoreround

import (
	"context"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/bwmarrin/discordgo"
)

// ScoreRoundManager defines the interface for create round operations.
type ScoreRoundManager interface {
	HandleScoreButton(ctx context.Context, i *discordgo.InteractionCreate)
	HandleScoreSubmission(ctx context.Context, i *discordgo.InteractionCreate)
	SendScoreUpdateConfirmation(channelID string, userID roundtypes.UserID, score *int) error
	SendScoreUpdateError(userID roundtypes.UserID, errorMsg string) error
}

// scoreRoundManager implements the ScoreRoundManager interface.
type scoreRoundManager struct {
	session   discord.Session
	publisher eventbus.EventBus
	logger    observability.Logger
	helper    utils.Helpers
	config    *config.Config
}

// NewScoreRoundManager creates a new ScoreRoundManager instance.
func NewScoreRoundManager(session discord.Session, publisher eventbus.EventBus, logger observability.Logger, helper utils.Helpers, config *config.Config) ScoreRoundManager {
	logger.Info(context.Background(), "Creating ScoreRoundManager",
		attr.Any("session", session),
		attr.Any("publisher", publisher),
		attr.Any("config", config),
	)
	return &scoreRoundManager{
		session:   session,
		publisher: publisher,
		logger:    logger,
		helper:    helper,
		config:    config,
	}
}

// userHasRole checks if a user has a specific Discord role
// func userHasRole(userRoles []string, requiredRoleID string) bool {
// 	for _, roleID := range userRoles {
// 		if roleID == requiredRoleID {
// 			return true
// 		}
// 	}
// 	return false
// }

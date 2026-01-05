package userhandlers

import (
	"context"
	"fmt"

	shareduserevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/user"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleUserCreated handles the UserCreated event from the backend.
func (h *UserHandlers) HandleUserCreated(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleUserCreated",
		&userevents.UserCreatedPayloadV1{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			createdPayload := payload.(*userevents.UserCreatedPayloadV1)

			rolePayload := shareduserevents.AddRolePayloadV1{
				UserID:  createdPayload.UserID,
				RoleID:  h.Config.GetRegisteredRoleID(),
				GuildID: string(createdPayload.GuildID),
			}
			roleMsg, err := h.Helper.CreateResultMessage(msg, rolePayload, shareduserevents.SignupAddRoleV1)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to create add role event", attr.Error(err))
				return nil, fmt.Errorf("failed to create add role event: %w", err)
			}
			return []*message.Message{roleMsg}, nil
		},
	)(msg)
}

// HandleUserCreationFailed handles the UserCreationFailed event from the backend.
func (h *UserHandlers) HandleUserCreationFailed(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleUserCreationFailed",
		&userevents.UserCreationFailedPayloadV1{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			failPayload := payload.(*userevents.UserCreationFailedPayloadV1)
			correlationID := msg.Metadata.Get("correlation_id")

			// Log the failure reason explicitly
			h.Logger.ErrorContext(ctx, "User creation failed",
				attr.CorrelationIDFromMsg(msg),
				attr.String("reason", failPayload.Reason))

			// Respond with the default failure message to the user
			_, err := h.UserDiscord.GetSignupManager().SendSignupResult(ctx, correlationID, false)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to send signup failure response",
					attr.Error(err),
					attr.CorrelationIDFromMsg(msg))
				return nil, fmt.Errorf("failed to send signup failure: %w", err)
			}

			h.Logger.InfoContext(ctx, "Sent signup failure result",
				attr.Any("result", nil), // The result is nil in case of success
				attr.CorrelationIDFromMsg(msg))
			return nil, nil
		},
	)(msg)
}

// HandleRoleAdded handles the RoleAdded event.
func (h *UserHandlers) HandleRoleAdded(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleRoleAdded",
		&shareduserevents.RoleAddedPayloadV1{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			correlationID := msg.Metadata.Get("correlation_id")

			result, err := h.UserDiscord.GetSignupManager().SendSignupResult(ctx, correlationID, true)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to send signup success response", attr.Error(err), attr.CorrelationIDFromMsg(msg))
				return nil, fmt.Errorf("failed to send signup success: %w", err)
			}

			h.Logger.InfoContext(ctx, "Sent signup success result", attr.Any("result", result), attr.CorrelationIDFromMsg(msg))
			return nil, nil
		},
	)(msg)
}

// HandleRoleAdditionFailed handles the RoleAdditionFailed event.
func (h *UserHandlers) HandleRoleAdditionFailed(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleRoleAdditionFailed",
		&shareduserevents.RoleAdditionFailedPayloadV1{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			correlationID := msg.Metadata.Get("correlation_id")

			result, err := h.UserDiscord.GetSignupManager().SendSignupResult(ctx, correlationID, false)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to send signup failure response", attr.Error(err), attr.CorrelationIDFromMsg(msg))
				return nil, fmt.Errorf("failed to send signup failure: %w", err)
			}

			h.Logger.InfoContext(ctx, "Sent signup failure result", attr.Any("result", result), attr.CorrelationIDFromMsg(msg))
			return nil, nil
		},
	)(msg)
}

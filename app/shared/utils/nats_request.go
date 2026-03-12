package messagecreator

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
)

// NATSRequest sends a JSON-encoded request to a NATS subject and waits for a JSON-encoded response.
func NATSRequest[Req any, Resp any](ctx context.Context, eb eventbus.EventBus, subject string, req Req, timeout time.Duration) (*Resp, error) {
	conn := eb.GetNATSConnection()
	if conn == nil {
		return nil, fmt.Errorf("NATS connection not available")
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	msg, err := conn.RequestWithContext(ctx, subject, reqBytes)
	if err != nil {
		return nil, fmt.Errorf("NATS request failed: %w", err)
	}

	var resp Resp
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &resp, nil
}

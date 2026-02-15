package reset

import (
	"strings"

	"github.com/google/uuid"
)

const (
	resetConfirmLegacyCustomID      = "frolf_reset_confirm"
	resetCancelLegacyCustomID       = "frolf_reset_cancel"
	resetConfirmCorrelationIDPrefix = "frolf_reset_confirm|cid="
	resetCancelCorrelationIDPrefix  = "frolf_reset_cancel|cid="
)

func newResetCorrelationID() string {
	return uuid.NewString()
}

func resetConfirmCustomID(correlationID string) string {
	if correlationID == "" {
		return resetConfirmLegacyCustomID
	}
	return resetConfirmCorrelationIDPrefix + correlationID
}

func resetCancelCustomID(correlationID string) string {
	if correlationID == "" {
		return resetCancelLegacyCustomID
	}
	return resetCancelCorrelationIDPrefix + correlationID
}

func resetCorrelationIDFromCustomID(customID string) string {
	if strings.HasPrefix(customID, resetConfirmCorrelationIDPrefix) {
		return strings.TrimPrefix(customID, resetConfirmCorrelationIDPrefix)
	}
	if strings.HasPrefix(customID, resetCancelCorrelationIDPrefix) {
		return strings.TrimPrefix(customID, resetCancelCorrelationIDPrefix)
	}
	return ""
}

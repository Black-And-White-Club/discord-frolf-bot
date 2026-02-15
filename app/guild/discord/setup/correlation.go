package setup

import (
	"strings"

	"github.com/google/uuid"
)

const (
	setupModalLegacyCustomID      = "guild_setup_modal"
	setupModalCorrelationIDPrefix = "guild_setup_modal|cid="
)

func newSetupCorrelationID() string {
	return uuid.NewString()
}

func setupModalCustomID(correlationID string) string {
	if correlationID == "" {
		return setupModalLegacyCustomID
	}
	return setupModalCorrelationIDPrefix + correlationID
}

func setupCorrelationIDFromCustomID(customID string) string {
	if strings.HasPrefix(customID, setupModalCorrelationIDPrefix) {
		return strings.TrimPrefix(customID, setupModalCorrelationIDPrefix)
	}
	return ""
}

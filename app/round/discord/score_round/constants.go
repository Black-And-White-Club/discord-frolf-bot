package scoreround

// Centralized constants for score round interaction & embed handling.

const (
	// Custom ID prefixes
	scoreButtonPrefix        = "round_enter_score|"
	bulkOverrideButtonPrefix = "round_bulk_score_override|"
	submitSingleModalPrefix  = "submit_score_modal|"
	submitBulkOverridePrefix = "submit_score_bulk_override|"

	// Score bounds (disc golf reasonable range)
	scoreMin = -36
	scoreMax = 72

	// Generic placeholders
	placeholderNoParticipants = "*No participants*"
	scorePrefix               = "Score:"
	scoreNoData               = "--"
	tagPrefix                 = "Tag:"
)

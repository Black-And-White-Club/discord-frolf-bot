package redaction

const redactedValue = "[redacted]"

// RedactSecret returns a fixed placeholder for non-empty secrets.
func RedactSecret(secret string) string {
	if secret == "" {
		return ""
	}
	return redactedValue
}

package sentryx

import "strings"

// Errors the stack recovers from on its own: websocket reconnects, app-state sync/decode and retry
// receipts, and request-scoped transport failures. Expected background behavior, so callers
// downgrade them to Debug and skip Sentry.
//
// The list lives here rather than next to the whatsmeow logger because the same texts reach Sentry
// through two doors, the zap logger and the Gin middleware, and one source of truth is what stops a
// filter from covering only half the paths.
var expectedNoise = []string{
	"failed to read frame header",
	"error reading from websocket",
	"websocket not connected",
	"autoreconnect",
	"keepalive timeout",
	"failed to resync app state",
	"failed to sync app state",
	"failed to handle retry receipt",
	"failed to decode app state",
	"context canceled",
	"info query timed out",
	"failed to send usync query",
}

// IsExpectedNoise reports whether the message is a known self-recovering failure.
func IsExpectedNoise(msg string) bool {
	m := strings.ToLower(msg)
	for _, p := range expectedNoise {
		if strings.Contains(m, p) {
			return true
		}
	}
	return false
}

// Fingerprint builds a stable Sentry grouping key for messages carrying identifiers. These errors
// interpolate message IDs, JIDs and phone numbers into the text, so using it as the issue title
// splits one recurring failure into hundreds of one-off issues. Groups by scope plus the message
// skeleton, with identifier-looking tokens replaced by a placeholder.
func Fingerprint(scope, msg string) []string {
	skeleton := msg
	if idx := strings.IndexAny(skeleton, ":("); idx > 0 {
		skeleton = skeleton[:idx]
	}

	fields := strings.Fields(skeleton)
	for i, f := range fields {
		if looksLikeIdentifier(f) {
			fields[i] = "<id>"
		}
	}
	skeleton = strings.Join(fields, " ")

	if len(skeleton) > 60 {
		skeleton = skeleton[:60]
	}
	return []string{scope, skeleton}
}

// looksLikeIdentifier reports whether a token is a JID, message ID, phone number or similar
// per-occurrence value rather than a stable word of the message.
func looksLikeIdentifier(tok string) bool {
	if strings.ContainsAny(tok, "@/") {
		return true
	}
	digits := 0
	for _, r := range tok {
		if r >= '0' && r <= '9' {
			digits++
		}
	}
	// Words never carry this many digits; IDs and phone numbers always do.
	return digits >= 4
}

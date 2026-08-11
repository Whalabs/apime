package sentryx

import "strings"

// Errors that the stack recovers from on its own. They are expected background behavior, not
// incidents, so callers downgrade them to Debug and skip Sentry reporting.
//
// Three families live here:
//   - websocket reconnects (auto-recovered by whatsmeow);
//   - app-state sync/decode and retry receipts, which the library retries by itself;
//   - request-scoped transport failures (socket down, context canceled) that surface on the HTTP
//     layer when a send lands mid-reconnect. The request already answers 503, and the caller
//     retries: reporting it again only creates noise.
//
// The list lives here, and not next to the whatsmeow logger, because the same texts reach Sentry
// through two different doors: the zap logger and the Gin middleware. Keeping one source of truth
// is what stops a filter from covering only half the paths.
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

// Fingerprint builds a stable Sentry grouping key for messages that carry identifiers.
//
// Errors here interpolate message IDs, JIDs and phone numbers straight into the text, so using it
// as the issue title splits one recurring failure into hundreds of one-off issues. We group by
// scope plus the message skeleton: the leading sentence with every identifier-looking token
// replaced by a placeholder. That keeps distinct failures apart while collapsing the same one
// into a single issue with a counter.
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

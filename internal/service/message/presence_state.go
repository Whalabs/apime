package message

import (
	"strings"
	"sync"
	"time"
)

// Tracks what has already been signaled to WhatsApp, so we don't repeat a signal that still holds.
// Besides the wasted round-trip on every send, an always-online presence is one of the patterns
// automation detection looks at.
var (
	availableAt sync.Map // instanceID -> time.Time of the last `available`
	composingAt sync.Map // instanceID + "|" + chatJID -> time.Time of the last `composing`
)

const (
	// `available` is client state and holds until it's replaced or the connection drops. Drops are
	// handled by event (see ForgetInstancePresence), so this TTL isn't the correctness guarantee —
	// it's a net for a reconnect that slips through. whatsmeow forces a reconnect within 3 minutes
	// of a failed keepalive, and that reconnect clears this state.
	availableTTL = 15 * time.Minute

	// On the WhatsApp Web protocol presence expires in ~10s (not the 25s of the official Cloud API,
	// which is a different path). 8s stays under it with margin for clock skew: above that the
	// cache would claim "still open" for an indicator the server already dissolved, and the message
	// would go out with no signal at all.
	//
	// Rarely used in practice: every send ends in `paused`, which closes the indicator and clears
	// the mark. It covers concurrent sends to the same chat and a failed `paused`.
	composingTTL = 8 * time.Second
)

func chatKey(instanceID, chatJID string) string {
	return instanceID + "|" + chatJID
}

func expired(m *sync.Map, key string, ttl time.Duration) bool {
	value, ok := m.Load(key)
	if !ok {
		return true
	}
	at, ok := value.(time.Time)
	return !ok || time.Since(at) >= ttl
}

// needsAvailable reports whether `available` is worth resending for this instance, marking the send.
func needsAvailable(instanceID string) bool {
	if !expired(&availableAt, instanceID, availableTTL) {
		return false
	}
	availableAt.Store(instanceID, time.Now())
	return true
}

// needsComposing reports whether the typing indicator is worth reopening, marking the opening.
func needsComposing(instanceID, chatJID string) bool {
	key := chatKey(instanceID, chatJID)
	if !expired(&composingAt, key, composingTTL) {
		return false
	}
	composingAt.Store(key, time.Now())
	return true
}

// forgetComposing is called when sending `paused`: the indicator is closed, so the next send has to
// actually reopen it.
func forgetComposing(instanceID, chatJID string) {
	composingAt.Delete(chatKey(instanceID, chatJID))
}

// ForgetInstancePresence clears the state on connect/disconnect: after reconnecting, nothing
// signaled on the previous session still holds on the other side.
func ForgetInstancePresence(instanceID string) {
	availableAt.Delete(instanceID)
	prefix := instanceID + "|"
	composingAt.Range(func(k, _ any) bool {
		if key, ok := k.(string); ok && strings.HasPrefix(key, prefix) {
			composingAt.Delete(key)
		}
		return true
	})
}

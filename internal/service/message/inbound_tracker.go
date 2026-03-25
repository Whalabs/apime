package message

import (
	"sync"
	"time"
)

const inboundTTL = 14 * 24 * time.Hour // 14 days

// inboundEntry tracks the last inbound message per chat for auto MarkRead
type inboundEntry struct {
	messageID string
	senderJID string
	trackedAt time.Time
}

var (
	inboundTracker sync.Map // key: "instanceID:chatJID" → value: inboundEntry
	cleanupOnce    sync.Once
)

// TrackInbound stores the last inbound message ID for a given chat.
// Called by the event handler when a message is received.
// The Send function uses this to auto-mark messages as read before sending.
func TrackInbound(instanceID, chatJID, messageID, senderJID string) {
	cleanupOnce.Do(startCleanupLoop)
	key := instanceID + ":" + chatJID
	inboundTracker.Store(key, inboundEntry{
		messageID: messageID,
		senderJID: senderJID,
		trackedAt: time.Now(),
	})
}

// popLastInbound returns and removes the last inbound message for a given chat.
// Returns false if no inbound message is tracked or if the entry has expired.
func popLastInbound(instanceID, chatJID string) (inboundEntry, bool) {
	key := instanceID + ":" + chatJID
	val, ok := inboundTracker.LoadAndDelete(key)
	if !ok {
		return inboundEntry{}, false
	}
	entry := val.(inboundEntry)
	if time.Since(entry.trackedAt) > inboundTTL {
		return inboundEntry{}, false
	}
	return entry, true
}

func startCleanupLoop() {
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()
			inboundTracker.Range(func(key, val any) bool {
				if now.Sub(val.(inboundEntry).trackedAt) > inboundTTL {
					inboundTracker.Delete(key)
				}
				return true
			})
		}
	}()
}

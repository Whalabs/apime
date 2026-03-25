package message

import (
	"context"
	"math/rand"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

// simulatePresenceDelay handles the typing/recording delay with optional micro-pauses.
// For text: breaks "typing..." into segments with brief pauses (type → pause → type → send).
// For audio: continuous recording without pauses (humans hold the record button).
// The initial ChatPresenceComposing must already be sent before calling this.
func simulatePresenceDelay(ctx context.Context, client *whatsmeow.Client, toJID types.JID, media types.ChatPresenceMedia, totalDelay time.Duration) {
	// Audio: continuous recording, no micro-pauses
	if media == types.ChatPresenceMediaAudio || totalDelay < 3*time.Second {
		time.Sleep(totalDelay)
		return
	}

	// Text: micro-pauses for delays > 3s
	numPauses := 1
	if totalDelay > 5*time.Second {
		numPauses = 1 + rand.Intn(2) // 1-2
	}
	if totalDelay > 7*time.Second {
		numPauses = 2 + rand.Intn(2) // 2-3
	}

	// Each pause 300-700ms
	pauseDurations := make([]time.Duration, numPauses)
	var totalPauseTime time.Duration
	for i := range pauseDurations {
		p := time.Duration(300+rand.Intn(400)) * time.Millisecond
		pauseDurations[i] = p
		totalPauseTime += p
	}

	// Distribute remaining time across typing segments
	typingTime := totalDelay - totalPauseTime
	numSegments := numPauses + 1
	avgSegment := typingTime / time.Duration(numSegments)

	for i := 0; i < numSegments; i++ {
		// Jitter ±20% per segment
		var jitter time.Duration
		if jitterRange := int64(avgSegment) / 5; jitterRange > 0 {
			jitter = time.Duration(rand.Int63n(jitterRange*2+1) - jitterRange)
		}
		segDuration := avgSegment + jitter
		if segDuration < 400*time.Millisecond {
			segDuration = 400 * time.Millisecond
		}

		time.Sleep(segDuration)

		if i < numPauses {
			_ = client.SendChatPresence(ctx, toJID, types.ChatPresencePaused, media)
			time.Sleep(pauseDurations[i])
			_ = client.SendChatPresence(ctx, toJID, types.ChatPresenceComposing, media)
		}
	}
}

func presenceMediaType(msgType string) types.ChatPresenceMedia {
	if msgType == "audio" {
		return types.ChatPresenceMediaAudio
	}
	return types.ChatPresenceMediaText
}

func calculatePresenceDelay(input SendInput) time.Duration {
	var base int
	switch input.Type {
	case "text":
		// ~40ms per char, min 1.5s, max 8s
		base = len(input.Text) * 40
		if base < 1500 {
			base = 1500
		}
		if base > 8000 {
			base = 8000
		}
	case "audio":
		// Based on audio duration (seconds)
		if input.Seconds > 0 {
			base = input.Seconds * 1000
			if base > 30000 {
				base = 30000
			}
		} else {
			base = 3000
		}
	case "image", "video":
		// Base delay + size factor + caption
		sizeMB := len(input.MediaData) / (1024 * 1024)
		base = 2000 + sizeMB*300
		if base > 8000 {
			base = 8000
		}
		if input.Caption != "" {
			extra := len(input.Caption) * 40
			if extra > 5000 {
				extra = 5000
			}
			base += extra
		}
	case "document":
		base = 1500
		if input.Caption != "" {
			extra := len(input.Caption) * 40
			if extra > 5000 {
				extra = 5000
			}
			base += extra
		}
	default:
		base = 1500
	}
	// Jitter ±10%
	jitter := rand.Intn(base/5+1) - base/10
	return time.Duration(base+jitter) * time.Millisecond
}

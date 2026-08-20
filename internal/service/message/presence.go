package message

import (
	"context"
	"math/rand"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

// gaussianJitter returns a factor around 1.0 to vary a duration. Uniform jitter spreads values
// evenly across the range, which is recognizable — people don't distribute their time that way.
// Tails are clamped at two standard deviations, since the normal curve reaches absurd values.
func gaussianJitter(stddev float64) float64 {
	factor := 1 + rand.NormFloat64()*stddev
	if lower := 1 - 2*stddev; factor < lower {
		factor = lower
	}
	if upper := 1 + 2*stddev; factor > upper {
		factor = upper
	}
	return factor
}

// sleepCtx waits but gives up if the context is canceled. The simulation runs in parallel with
// message preparation, so it can still be pending after the send was aborted; with a plain
// time.Sleep the typing indicator would linger for the contact. Returns false when canceled.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return true
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}

// simulatePresenceDelay handles the typing/recording delay with optional micro-pauses.
// For text: breaks "typing..." into segments with brief pauses (type → pause → type → send).
// For audio: continuous recording without pauses (humans hold the record button).
// The initial ChatPresenceComposing must already be sent before calling this.
func simulatePresenceDelay(ctx context.Context, client *whatsmeow.Client, toJID types.JID, media types.ChatPresenceMedia, totalDelay time.Duration) {
	// Audio: continuous recording, no micro-pauses
	if media == types.ChatPresenceMediaAudio || totalDelay < 3*time.Second {
		sleepCtx(ctx, totalDelay)
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

	// Pauses around 500ms, clustered near the mean instead of spread evenly over 300-700.
	pauseDurations := make([]time.Duration, numPauses)
	var totalPauseTime time.Duration
	for i := range pauseDurations {
		p := time.Duration(500 * gaussianJitter(0.20) * float64(time.Millisecond))
		pauseDurations[i] = p
		totalPauseTime += p
	}

	// Distribute remaining time across typing segments
	typingTime := totalDelay - totalPauseTime
	numSegments := numPauses + 1
	avgSegment := typingTime / time.Duration(numSegments)

	for i := 0; i < numSegments; i++ {
		// Each segment varies around its mean rather than uniformly within it.
		segDuration := time.Duration(float64(avgSegment) * gaussianJitter(0.20))
		if segDuration < 400*time.Millisecond {
			segDuration = 400 * time.Millisecond
		}

		if !sleepCtx(ctx, segDuration) {
			return
		}

		if i < numPauses {
			_ = client.SendChatPresence(ctx, toJID, types.ChatPresencePaused, media)
			if !sleepCtx(ctx, pauseDurations[i]) {
				return
			}
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

// Floor for the typing indicator: even when the contact already waited, typing is signaled briefly.
// Sending with no indicator at all is the pattern automation detection looks for.
const typingFloor = 1200 * time.Millisecond

// Cap for "recording". Basing it on the full audio duration produced 30s waits (the source of the
// near-minute send peaks). What's noticed is the delay until the message arrives, not whether a
// 50s audio took 50s or 15s to record.
const recordingCap = 15 * time.Second

// adjustForFamiliarity only reduces the delay on an open conversation, where the contact wrote
// first and a fast reply is what a human would do.
//
// Otherwise the behavior is EXACTLY the previous one (fixed 1.5-2.5s "new conversation" bump).
// That's deliberate: first contact without inbound is the campaign profile, and this delay adds to
// the interval the campaign already waits between recipients. Shortening it would speed up every
// campaign already configured without anyone asking — that's raising send rate silently.
func adjustForFamiliarity(base time.Duration, hasRecentInbound bool) time.Duration {
	if hasRecentInbound {
		return base
	}
	return base + time.Duration(1500+rand.Intn(1000))*time.Millisecond
}

// subtractElapsed discounts the time the contact ALREADY waited since their last message. Adding
// the full delay on top of a wait that already happened doesn't humanize, it just delays. The floor
// keeps the indicator visible.
func subtractElapsed(delay, elapsed time.Duration) time.Duration {
	remaining := delay - elapsed
	if remaining < typingFloor {
		return typingFloor
	}
	return remaining
}

// reconnectFactor brakes the first seconds after connecting: coming back from a reconnect dumping
// messages at full speed is an automation pattern. Decays from 2x to 1x across the window.
func reconnectFactor(sinceConnect, window time.Duration) float64 {
	if sinceConnect >= window || window <= 0 {
		return 1.0
	}
	return 1.0 + (1.0 - float64(sinceConnect)/float64(window))
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
		// Based on audio duration (seconds), capped — see recordingCap.
		if input.Seconds > 0 {
			base = input.Seconds * 1000
			if base > int(recordingCap/time.Millisecond) {
				base = int(recordingCap / time.Millisecond)
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
	// Jitter around ±10%, clustered near the mean (see gaussianJitter).
	return time.Duration(float64(base)*gaussianJitter(0.10)) * time.Millisecond
}

package message

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/socket"
)

const (
	// A 503 stream error or a dropped socket takes whatsmeow 15-35s to recover. The old 2+4+8s
	// backoff gave up before that, failing sends that would have gone out a few seconds later.
	reconnectWaitBudget = 45 * time.Second
	// Back-to-back failures belong to the same outage; a quiet gap longer than this starts a new one.
	outageForgetAfter  = 60 * time.Second
	reconnectPollEvery = 500 * time.Millisecond
)

// isSocketDown reports a send that failed because there was no socket to write to. In that case no
// frame left the process, so waiting and resending can never duplicate the message.
func isSocketDown(err error) bool {
	return errors.Is(err, whatsmeow.ErrNotConnected) || errors.Is(err, socket.ErrSocketClosed)
}

type outage struct {
	since    time.Time
	lastSeen time.Time
}

// Sends of one instance are serialized by instancelock, so without a shared deadline every queued
// message would wait the full budget again during a long outage, pushing callers past their timeout.
type outageTracker struct {
	mu sync.Mutex
	m  map[string]outage
}

var outages = &outageTracker{m: make(map[string]outage)}

// deadline records a socket failure for the instance and returns until when it is worth waiting.
func (t *outageTracker) deadline(instanceID string, now time.Time) time.Time {
	t.mu.Lock()
	defer t.mu.Unlock()
	o, ok := t.m[instanceID]
	if !ok || now.Sub(o.lastSeen) > outageForgetAfter {
		o.since = now
	}
	o.lastSeen = now
	t.m[instanceID] = o
	return o.since.Add(reconnectWaitBudget)
}

func (t *outageTracker) clear(instanceID string) {
	t.mu.Lock()
	delete(t.m, instanceID)
	t.mu.Unlock()
}

// waitUntilConnected polls instead of using client.WaitForConnection so the wait honors ctx: when
// the caller gives up, nothing must be sent behind its back.
func waitUntilConnected(ctx context.Context, connected func() bool, deadline time.Time) bool {
	for {
		if connected() {
			return true
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return false
		}
		select {
		case <-ctx.Done():
			return false
		case <-time.After(min(remaining, reconnectPollEvery)):
		}
	}
}

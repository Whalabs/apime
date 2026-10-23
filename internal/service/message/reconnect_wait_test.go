package message

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/socket"
)

func TestIsSocketDownMatchesWrappedErrors(t *testing.T) {
	cases := []error{
		fmt.Errorf("failed to send message node: %w", whatsmeow.ErrNotConnected),
		fmt.Errorf("failed to get device list: %w", fmt.Errorf("failed to send usync query: %w", whatsmeow.ErrNotConnected)),
		fmt.Errorf("failed to send message node: %w", socket.ErrSocketClosed),
	}
	for _, err := range cases {
		if !isSocketDown(err) {
			t.Fatalf("expected socket-down for %q", err)
		}
	}
}

func TestIsSocketDownRejectsAmbiguousErrors(t *testing.T) {
	// A timeout may mean the frame went out and only the ack was lost: resending could duplicate.
	cases := []error{
		whatsmeow.ErrIQTimedOut,
		context.DeadlineExceeded,
		errors.New("server returned error 463"),
		errors.New("websocket not connected"),
	}
	for _, err := range cases {
		if isSocketDown(err) {
			t.Fatalf("did not expect socket-down for %q", err)
		}
	}
}

func TestOutageDeadlineIsSharedWithinOneOutage(t *testing.T) {
	tr := &outageTracker{m: make(map[string]outage)}
	t0 := time.Now()
	first := tr.deadline("i1", t0)
	second := tr.deadline("i1", t0.Add(30*time.Second))
	if !first.Equal(second) {
		t.Fatalf("queued sends must share the outage deadline: %v != %v", first, second)
	}
	if !first.Equal(t0.Add(reconnectWaitBudget)) {
		t.Fatalf("unexpected deadline %v", first)
	}
}

func TestOutageDeadlineResetsAfterQuietGap(t *testing.T) {
	tr := &outageTracker{m: make(map[string]outage)}
	t0 := time.Now()
	tr.deadline("i1", t0)
	later := t0.Add(outageForgetAfter + time.Second)
	if got := tr.deadline("i1", later); !got.Equal(later.Add(reconnectWaitBudget)) {
		t.Fatalf("a new outage must get a fresh budget, got %v", got)
	}
}

func TestOutageDeadlineIsPerInstance(t *testing.T) {
	tr := &outageTracker{m: make(map[string]outage)}
	t0 := time.Now()
	tr.deadline("i1", t0)
	if got := tr.deadline("i2", t0.Add(40*time.Second)); !got.Equal(t0.Add(40*time.Second + reconnectWaitBudget)) {
		t.Fatalf("instances must not share outages, got %v", got)
	}
}

func TestOutageClearGivesFreshBudget(t *testing.T) {
	tr := &outageTracker{m: make(map[string]outage)}
	t0 := time.Now()
	tr.deadline("i1", t0)
	tr.clear("i1")
	at := t0.Add(10 * time.Second)
	if got := tr.deadline("i1", at); !got.Equal(at.Add(reconnectWaitBudget)) {
		t.Fatalf("expected fresh budget after clear, got %v", got)
	}
}

func TestWaitUntilConnectedReturnsOnReconnect(t *testing.T) {
	calls := 0
	connected := func() bool { calls++; return calls >= 3 }
	if !waitUntilConnected(context.Background(), connected, time.Now().Add(5*time.Second)) {
		t.Fatal("expected reconnect to be detected")
	}
}

func TestWaitUntilConnectedGivesUpAtDeadline(t *testing.T) {
	start := time.Now()
	if waitUntilConnected(context.Background(), func() bool { return false }, start.Add(300*time.Millisecond)) {
		t.Fatal("expected timeout")
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("overshot the deadline: %v", elapsed)
	}
}

func TestWaitUntilConnectedHonorsCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if waitUntilConnected(ctx, func() bool { return false }, time.Now().Add(10*time.Second)) {
		t.Fatal("a cancelled caller must not be waited on")
	}
}

func TestWaitUntilConnectedPastDeadlineStillChecksOnce(t *testing.T) {
	if !waitUntilConnected(context.Background(), func() bool { return true }, time.Now().Add(-time.Second)) {
		t.Fatal("an already connected client must not be reported as down")
	}
}

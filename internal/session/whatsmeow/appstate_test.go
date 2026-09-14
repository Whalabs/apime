package whatsmeow

import (
	"testing"
	"time"

	"go.mau.fi/whatsmeow/appstate"
	"go.uber.org/zap"
)

func TestParseCollectionsDefaultsToTheOnesThatBreakFeatures(t *testing.T) {
	got, err := parseCollections(nil)
	if err != nil {
		t.Fatalf("parseCollections(nil) failed: %v", err)
	}
	want := []appstate.WAPatchName{appstate.WAPatchCriticalUnblockLow, appstate.WAPatchRegular}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("index %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestParseCollectionsRejectsUnknownName(t *testing.T) {
	// A typo must fail loudly: silently syncing nothing would look like a working repair.
	for _, bad := range []string{"regular_lo", "contacts", "", "REGULAR"} {
		if _, err := parseCollections([]string{bad}); err == nil {
			t.Errorf("parseCollections(%q) should have failed", bad)
		}
	}
}

func TestParseCollectionsAcceptsEveryRealCollection(t *testing.T) {
	for _, name := range appstate.AllPatchNames {
		got, err := parseCollections([]string{string(name)})
		if err != nil {
			t.Fatalf("parseCollections(%q) failed: %v", name, err)
		}
		if len(got) != 1 || got[0] != name {
			t.Errorf("parseCollections(%q) = %v", name, got)
		}
	}
}

func TestResyncCooldownBlocksTheSecondAttempt(t *testing.T) {
	// The cooldown is the guard against hammering WhatsApp: a collection that keeps failing would
	// otherwise resync in a loop, since the failure re-fires the event that triggers the repair.
	m := &Manager{log: zap.NewNop()}
	if !m.markAppStateResync("inst", appstate.WAPatchRegular) {
		t.Fatal("first attempt should be allowed")
	}
	if m.markAppStateResync("inst", appstate.WAPatchRegular) {
		t.Error("second attempt should be blocked by the cooldown")
	}
}

func TestResyncCooldownIsPerInstanceAndCollection(t *testing.T) {
	m := &Manager{log: zap.NewNop()}
	m.markAppStateResync("inst-a", appstate.WAPatchRegular)

	if !m.markAppStateResync("inst-b", appstate.WAPatchRegular) {
		t.Error("another instance must not inherit the cooldown")
	}
	if !m.markAppStateResync("inst-a", appstate.WAPatchCriticalUnblockLow) {
		t.Error("another collection must not inherit the cooldown")
	}
}

func TestResyncCooldownExpires(t *testing.T) {
	m := &Manager{log: zap.NewNop()}
	m.markAppStateResync("inst", appstate.WAPatchRegular)

	m.mu.Lock()
	m.appStateResync["inst:"+string(appstate.WAPatchRegular)] = time.Now().Add(-appStateResyncCooldown - time.Second)
	m.mu.Unlock()

	if !m.markAppStateResync("inst", appstate.WAPatchRegular) {
		t.Error("attempt should be allowed once the cooldown has elapsed")
	}
}

func TestClearResyncAllowsAnImmediateRetry(t *testing.T) {
	// A failed attempt must not consume the cooldown window: nothing was synced, so barring the
	// next try would leave the collection broken for the whole interval.
	m := &Manager{log: zap.NewNop()}
	m.markAppStateResync("inst", appstate.WAPatchRegular)
	m.clearAppStateResync("inst", appstate.WAPatchRegular)

	if !m.markAppStateResync("inst", appstate.WAPatchRegular) {
		t.Error("attempt after a cleared mark should be allowed")
	}
}

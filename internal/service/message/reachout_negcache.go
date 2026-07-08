package message

import (
	"sync"
	"time"
)

// reachoutNegTTL is how long a contact that returned error 463 (reach-out timelock) stays
// blocked for a given instance before a send is attempted again. Short by design: any inbound
// from the contact releases the block immediately (see reachoutRelease), so this TTL only bounds
// the "cold and refusing" window.
const reachoutNegTTL = 6 * time.Hour

// reachoutNegCache implements the "smart" 463 block: GLOBAL on write, PER-CONNECTION on release.
//
//   - A contact that returns 463 is stored blocked for the instance that hit it. Because the
//     scheduler/operator may switch connections to retry the same cold contact (the exact pattern
//     that got a device 403-logged-out), each instance that hits 463 accumulates in the contact's
//     set — so switching connection does not bypass the guard.
//   - Release is per-connection: when the contact messages a given instance (inbound), only THAT
//     instance is unblocked. This is correct, not just conservative — the tctoken is per-connection
//     (each device has its own), so instance A still lacks the token even if the contact replied to B.
//
// Key = "contactKey" (device-less PN via normalizeChatKey). Value = map[instanceID]expiry.
var (
	reachoutNegCache sync.Map // contactKey -> map[string]time.Time
	reachoutNegMu    sync.Mutex
	reachoutCleanup  sync.Once
)

// reachoutBlocked reports whether sending from instanceID to contactKey is currently blocked by a
// recent 463. Expired entries are cleaned lazily.
func reachoutBlocked(instanceID, contactKey string) bool {
	v, ok := reachoutNegCache.Load(contactKey)
	if !ok {
		return false
	}
	reachoutNegMu.Lock()
	defer reachoutNegMu.Unlock()
	m, _ := v.(map[string]time.Time)
	if m == nil {
		return false
	}
	exp, ok := m[instanceID]
	if !ok {
		return false
	}
	if time.Now().After(exp) {
		delete(m, instanceID)
		if len(m) == 0 {
			reachoutNegCache.Delete(contactKey)
		}
		return false
	}
	return true
}

// reachoutStore records a 463 block for (contactKey, instanceID). Global per contact, so any later
// send from ANY instance that also hit 463 is barred; switching connections does not evade it.
func reachoutStore(instanceID, contactKey string) {
	reachoutCleanup.Do(startReachoutCleanup)
	reachoutNegMu.Lock()
	defer reachoutNegMu.Unlock()
	v, _ := reachoutNegCache.LoadOrStore(contactKey, map[string]time.Time{})
	m, _ := v.(map[string]time.Time)
	if m == nil {
		m = map[string]time.Time{}
		reachoutNegCache.Store(contactKey, m)
	}
	m[instanceID] = time.Now().Add(reachoutNegTTL)
}

// reachoutRelease unblocks ONLY the given instance for the contact (called on inbound). The contact
// talked to this connection → this connection now has/will get the tctoken. Other connections stay
// blocked until the contact talks to them too.
func reachoutRelease(instanceID, contactKey string) {
	v, ok := reachoutNegCache.Load(contactKey)
	if !ok {
		return
	}
	reachoutNegMu.Lock()
	defer reachoutNegMu.Unlock()
	m, _ := v.(map[string]time.Time)
	if m == nil {
		return
	}
	delete(m, instanceID)
	if len(m) == 0 {
		reachoutNegCache.Delete(contactKey)
	}
}

// ReleaseReachoutOnInbound is the public entry point called by the webhook layer when an inbound
// message arrives: it unblocks this instance for the contact (the chatJID is normalized to the same
// device-less key used on send). No-op if nothing was blocked.
func ReleaseReachoutOnInbound(instanceID, chatJID string) {
	reachoutRelease(instanceID, normalizeChatKey(chatJID))
}

func startReachoutCleanup() {
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()
			reachoutNegCache.Range(func(key, val any) bool {
				reachoutNegMu.Lock()
				m, _ := val.(map[string]time.Time)
				if m != nil {
					for inst, exp := range m {
						if now.After(exp) {
							delete(m, inst)
						}
					}
					if len(m) == 0 {
						reachoutNegCache.Delete(key)
					}
				}
				reachoutNegMu.Unlock()
				return true
			})
		}
	}()
}

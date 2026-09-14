package whatsmeow

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/appstate"
	"go.uber.org/zap"
)

// appStateResyncCooldown is the minimum interval between full syncs of the same collection on the
// same instance. A full sync is what a freshly linked client does, so doing it now and then is
// ordinary traffic; doing it in a loop is not, and a patch that keeps failing would loop forever
// without this. It also bounds a subtler cost: whatsmeow holds appStateSyncLock across the whole
// network round trip, so a full sync stalls the processing of incoming server_sync notifications.
const appStateResyncCooldown = 15 * time.Minute

// resyncableCollections are the app state collections a caller may ask to resync, mapped from the
// names WhatsApp itself uses. Restricted to a known set so a typo becomes an error instead of a
// silent no-op against a collection that does not exist.
var resyncableCollections = map[string]appstate.WAPatchName{
	string(appstate.WAPatchCriticalBlock):      appstate.WAPatchCriticalBlock,
	string(appstate.WAPatchCriticalUnblockLow): appstate.WAPatchCriticalUnblockLow,
	string(appstate.WAPatchRegular):            appstate.WAPatchRegular,
	string(appstate.WAPatchRegularHigh):        appstate.WAPatchRegularHigh,
	string(appstate.WAPatchRegularLow):         appstate.WAPatchRegularLow,
}

// ResyncAppState discards the local copy of the given app state collections and downloads a fresh
// snapshot from WhatsApp. It is the repair for a collection that diverged (mismatching LTHash),
// which whatsmeow reports and then gives up on: incremental patches can no longer be applied over
// a base the server disagrees with, and nothing retries on its own.
//
// Passing no collection resyncs the two that break user-visible features when stale:
// critical_unblock_low (the contact list, which is the audience of a status) and regular (which
// carries the broadcast lists).
//
// The result separates what was resynced from what was skipped by the cooldown, so a caller that
// gets nothing back can tell "already done recently" from "nothing happened".
func (m *Manager) ResyncAppState(ctx context.Context, instanceID string, collections []string) (resynced, skipped []string, err error) {
	names, err := parseCollections(collections)
	if err != nil {
		return nil, nil, err
	}

	m.mu.RLock()
	client, exists := m.clients[instanceID]
	m.mu.RUnlock()
	if !exists || client == nil {
		return nil, nil, fmt.Errorf("instância não conectada")
	}
	// The connection check is not a nicety. FetchAppState deletes the stored version BEFORE the
	// network request, and does not roll back, so calling it offline leaves the collection at
	// version 0 with no snapshot to replace it.
	if !client.IsConnected() || !client.IsLoggedIn() {
		return nil, nil, fmt.Errorf("instância não conectada")
	}

	resynced = make([]string, 0, len(names))
	skipped = make([]string, 0, len(names))
	for _, name := range names {
		if !m.markAppStateResync(instanceID, name) {
			m.log.Info("resync de app state ignorado, ainda em cooldown",
				zap.String("instance_id", instanceID),
				zap.String("collection", string(name)))
			skipped = append(skipped, string(name))
			continue
		}
		if err := client.FetchAppState(ctx, name, true, false); err != nil {
			// Drop the cooldown mark: the attempt did not complete, so the next one should not be
			// barred by a sync that never happened.
			m.clearAppStateResync(instanceID, name)
			return resynced, skipped, fmt.Errorf("falha ao ressincronizar %s: %w", name, err)
		}
		m.log.Info("app state ressincronizado",
			zap.String("instance_id", instanceID),
			zap.String("collection", string(name)))
		resynced = append(resynced, string(name))
	}
	return resynced, skipped, nil
}

// parseCollections maps the requested names, defaulting to the two that matter to the API surface.
func parseCollections(collections []string) ([]appstate.WAPatchName, error) {
	if len(collections) == 0 {
		return []appstate.WAPatchName{
			appstate.WAPatchCriticalUnblockLow,
			appstate.WAPatchRegular,
		}, nil
	}
	names := make([]appstate.WAPatchName, 0, len(collections))
	for _, raw := range collections {
		name, ok := resyncableCollections[strings.TrimSpace(raw)]
		if !ok {
			return nil, fmt.Errorf("coleção inválida: %q", raw)
		}
		names = append(names, name)
	}
	return names, nil
}

// markAppStateResync reports whether a resync may run now, recording the attempt when it may.
func (m *Manager) markAppStateResync(instanceID string, name appstate.WAPatchName) bool {
	key := instanceID + ":" + string(name)
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.appStateResync == nil {
		m.appStateResync = make(map[string]time.Time)
	}
	if last, ok := m.appStateResync[key]; ok && time.Since(last) < appStateResyncCooldown {
		return false
	}
	m.appStateResync[key] = time.Now()
	return true
}

func (m *Manager) clearAppStateResync(instanceID string, name appstate.WAPatchName) {
	key := instanceID + ":" + string(name)
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.appStateResync, key)
}

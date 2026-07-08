package instancelock

import "sync"

var locks sync.Map // instanceID -> *sync.Mutex

// Acquire serializes all WhatsApp-facing actions of a single instance
// (send, react, edit, delete, presence). Different instances stay parallel.
// Returns the release func; call it with defer.
func Acquire(instanceID string) func() {
	m, _ := locks.LoadOrStore(instanceID, &sync.Mutex{})
	mu := m.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

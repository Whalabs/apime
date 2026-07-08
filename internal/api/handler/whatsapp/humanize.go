package whatsapp

import (
	"math/rand"
	"time"
)

// humanPause adds a short randomized delay before low-level actions (react,
// edit, delete) so they don't fire back-to-back at machine speed.
func humanPause() {
	time.Sleep(time.Duration(400+rand.Intn(700)) * time.Millisecond)
}

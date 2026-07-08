package instance

import (
	"sync"
	"time"
)

// pictureNegTTL is how long a JID rejected by the server for picture queries
// stays cached before being retried.
const pictureNegTTL = 6 * time.Hour

var pictureNegCache sync.Map // jid string -> expiry time.Time

func pictureNegHit(jid string) bool {
	v, ok := pictureNegCache.Load(jid)
	if !ok {
		return false
	}
	exp, _ := v.(time.Time)
	if time.Now().After(exp) {
		pictureNegCache.Delete(jid)
		return false
	}
	return true
}

func pictureNegStore(jid string) {
	pictureNegCache.Store(jid, time.Now().Add(pictureNegTTL))
}

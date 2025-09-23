package session

import (
	"sync"
	"time"
)

// NonceCache stores seen (keyid, nonce) pairs with a TTL to prevent replays.
type NonceCache struct {
	ttl  time.Duration
	data sync.Map        // keyid -> *sync.Map (nonce -> expiryUnix)
	tick *time.Ticker
	stop chan struct{}
}

// NewNonceCache creates a TTL-based replay cache (typical TTL: 5–10 minutes).
func NewNonceCache(ttl time.Duration) *NonceCache {
	nc := &NonceCache{
		ttl:  ttl,
		stop: make(chan struct{}),
		tick: time.NewTicker(time.Minute),
	}
	go nc.gcLoop()
	return nc
}

// Seen returns true if (keyid, nonce) was seen before; otherwise records it and returns false.
func (n *NonceCache) Seen(keyid, nonce string) bool {
	if keyid == "" || nonce == "" {
		return false
	}
	exp := time.Now().Add(n.ttl).Unix()

	v, _ := n.data.LoadOrStore(keyid, &sync.Map{}) // inner: nonce -> expiryUnix
	m := v.(*sync.Map)

	if old, ok := m.Load(nonce); ok {
		if prevExp, _ := old.(int64); prevExp >= time.Now().Unix() {
			return true // replay
		}
	}
	m.Store(nonce, exp)
	return false
}

// DeleteKey removes all nonces for a keyid (call on keyid unbind/session close).
func (n *NonceCache) DeleteKey(keyid string) {
	n.data.Delete(keyid)
}

// Close stops the background GC.
func (n *NonceCache) Close() {
	close(n.stop)
	if n.tick != nil {
		n.tick.Stop()
	}
}

func (n *NonceCache) gcLoop() {
	for {
		select {
		case <-n.tick.C:
			now := time.Now().Unix()
			n.data.Range(func(k, v any) bool {
				m := v.(*sync.Map)
				empty := true
				m.Range(func(nk, nv any) bool {
					if exp, _ := nv.(int64); exp < now {
						m.Delete(nk)
					} else {
						empty = false
					}
					return true
				})
				if empty {
					n.data.Delete(k)
				}
				return true
			})
		case <-n.stop:
			return
		}
	}
}

package hpke

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/registry010"
)

// A shared marker makes value copies of a transferred session unusable too.
// All record operations still serialize on the original endpoint mutex.
type recordLifetime010 struct {
	disabled                 atomic.Bool
	used, http               atomic.Bool
	sampledMono, sampledWall atomic.Int64
	checked                  atomic.Int64
	active, wall             atomic.Int64
	confirmed                atomic.Bool
}

func newRecordLifetime010(t registry010.Stamp) *recordLifetime010 {
	l := &recordLifetime010{}
	l.active.Store(t.MonoMS)
	l.wall.Store(t.Unix)
	l.checked.Store(t.MonoMS)
	l.sampledMono.Store(t.MonoMS)
	l.sampledWall.Store(t.Unix)
	return l
}
func (s *AuthenticatedCompletion010) unavailable010() bool {
	return s.lifetime == nil || s.lifetime.disabled.Load()
}

// NonHTTPOwner010 exclusively owns an unused non-HTTP record session. Value
// copies of this handle share the same session and cannot reset its history.
// Keep it private inside the trusted protocol adapter; it is not a wire permit.
type NonHTTPOwner010 struct{ session *AuthenticatedCompletion010 }

// TakeNonHTTP transfers an unused session exactly once, invalidating the source
// and all its value copies. HTTP-bound, used and unavailable sessions reject the
// transfer. Close on an old handle cannot erase the new owner's keys.
func (s *AuthenticatedCompletion010) TakeNonHTTP() (*NonHTTPOwner010, error) {
	if s == nil || s.endpoint == nil {
		return nil, errCompletion010
	}
	s.endpoint.mu.Lock()
	defer s.endpoint.mu.Unlock()
	if s.unavailable010() || s.closed || s.endpoint.retired.Load() || s.httpTarget != "" || s.lifetime.used.Load() || s.lifetime.http.Load() || len(s.sent)+len(s.received) != 0 {
		return nil, errCompletion010
	}
	moved := *s
	moved.lifetime = newRecordLifetime010(s.created)
	moved.lifetime.confirmed.Store(s.confirmed)
	moved.lifetime.active.Store(s.lifetime.active.Load())
	moved.lifetime.wall.Store(s.lifetime.wall.Load())
	moved.lifetime.sampledMono.Store(s.lifetime.sampledMono.Load())
	moved.lifetime.sampledWall.Store(s.lifetime.sampledWall.Load())
	moved.lifetime.checked.Store(s.lifetime.checked.Load())
	s.lifetime.disabled.Store(true)
	return &NonHTTPOwner010{session: &moved}, nil
}
func (o *NonHTTPOwner010) Initiator() bool {
	return o != nil && o.session != nil && o.session.initiator
}
func (o *NonHTTPOwner010) CreatedMonoMS() int64 {
	if o == nil || o.session == nil {
		return -1
	}
	return o.session.created.MonoMS
}

// LocalNow is a bounded local sample, not a registry lookup. The provisioned
// Clock must be thread-safe, non-reentrant and bounded. Its original time origin
// is preserved across ownership transfer. Current registry checks remain required.
func (o *NonHTTPOwner010) LocalNow() (now time.Duration, err error) {
	defer func() {
		if recover() != nil {
			now = 0
			err = errCompletion010
		}
	}()
	if o == nil || o.session == nil {
		return 0, errCompletion010
	}
	s := o.session
	t, err := s.endpoint.clock.Now()
	mono, wall := s.lifetime.sampledMono.Load(), s.lifetime.sampledWall.Load()
	if err != nil || t.MonoMS < mono || t.MonoMS < s.lifetime.checked.Load() || t.Unix < wall || t.Unix > 9007199254740691 || s.unavailable010() || s.endpoint.retired.Load() || t.MonoMS < s.created.MonoMS || t.MonoMS < s.lifetime.active.Load() || t.Unix < s.created.Unix || t.Unix < s.lifetime.wall.Load() || t.MonoMS-s.created.MonoMS >= 3600000 || t.MonoMS-s.lifetime.active.Load() >= 600000 || !pinnedLive010(t.Unix, s.a, s.b) || (!s.initiator && !s.lifetime.confirmed.Load() && !pendingLive010(t, s.created, s.expires)) || t.MonoMS > int64((1<<63-1)/time.Millisecond) {
		return 0, errCompletion010
	}
	if !s.lifetime.sampledMono.CompareAndSwap(mono, t.MonoMS) || !s.lifetime.sampledWall.CompareAndSwap(wall, t.Unix) {
		return 0, errCompletion010
	}
	return time.Duration(t.MonoMS) * time.Millisecond, nil
}
func (o *NonHTTPOwner010) Check(ctx context.Context) error {
	if o == nil || o.session == nil || ctx == nil || ctx.Err() != nil {
		return errCompletion010
	}
	return o.session.Check(ctx)
}
func (o *NonHTTPOwner010) SealRequest(ctx context.Context, b []byte, ttl int64) ([]byte, error) {
	if o == nil || o.session == nil || ctx == nil || ctx.Err() != nil {
		return nil, errCompletion010
	}
	return o.session.SealRequest(ctx, b, ttl)
}
func (o *NonHTTPOwner010) OpenRequest(ctx context.Context, b []byte) ([]byte, error) {
	if o == nil || o.session == nil || ctx == nil || ctx.Err() != nil {
		return nil, errCompletion010
	}
	return o.session.OpenRequest(ctx, b)
}
func (o *NonHTTPOwner010) SealResponse(ctx context.Context, id string, b []byte, success bool, code string, ttl int64) ([]byte, error) {
	if o == nil || o.session == nil || ctx == nil || ctx.Err() != nil {
		return nil, errCompletion010
	}
	return o.session.SealResponse(ctx, id, b, success, code, ttl)
}
func (o *NonHTTPOwner010) OpenResponse(ctx context.Context, b []byte) (*SessionResponse010, error) {
	if o == nil || o.session == nil || ctx == nil || ctx.Err() != nil {
		return nil, errCompletion010
	}
	return o.session.OpenResponse(ctx, b)
}
func (o *NonHTTPOwner010) Close() {
	if o != nil && o.session != nil {
		o.session.Close()
	}
}

// Observe returns the local start of a fresh registry validation operation. It is
// diagnostic timing, never a portable authorization token or a dispatch permit.
func (o *NonHTTPOwner010) Observe(ctx context.Context) (time.Duration, error) {
	if o == nil || o.session == nil || ctx == nil || ctx.Err() != nil {
		return 0, errCompletion010
	}
	s := o.session
	s.endpoint.mu.Lock()
	defer s.endpoint.mu.Unlock()
	start, err := s.checkCurrent010(ctx)
	if err != nil || start.MonoMS > int64((1<<63-1)/time.Millisecond) {
		return 0, errCompletion010
	}
	return time.Duration(start.MonoMS) * time.Millisecond, nil
}

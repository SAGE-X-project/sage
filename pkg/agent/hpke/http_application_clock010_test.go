package hpke

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/registry010"
)

type applicationLocalClock010 func() (registry010.Stamp, error)

func (f applicationLocalClock010) Now() (registry010.Stamp, error) { return f() }

func TestHTTPApplicationLocalStampSample010(t *testing.T) {
	f := applicationPair010(t, true)
	calls := 0
	want := registry010.Stamp{MonoMS: 1234, Unix: 101}
	f.left.session.endpoint.clock = applicationLocalClock010(func() (registry010.Stamp, error) {
		calls++
		return want, nil
	})
	beforeReads := f.reads.calls.Load()
	active, checked, wall := f.left.session.lifetime.active.Load(), f.left.session.lifetime.checked.Load(), f.left.session.lifetime.wall.Load()
	stamp, err := f.left.LocalStamp()
	if err != nil || stamp != want || calls != 1 {
		t.Fatalf("one full sample: %+v %v calls=%d", stamp, err, calls)
	}
	// Mutating the returned value cannot alter ordered clock history.
	stamp.MonoMS, stamp.Unix = -1, -1
	now, err := f.left.LocalNow()
	if err != nil || now != 1234*time.Millisecond || calls != 2 {
		t.Fatalf("LocalNow parity: %v %v calls=%d", now, err, calls)
	}
	copyOwner := *f.left
	stamp, err = copyOwner.LocalStamp()
	if err != nil || stamp != want || calls != 3 {
		t.Fatal("copied owner lost the same sampled origin", stamp, err, calls)
	}
	if f.reads.calls.Load() != beforeReads || f.left.session.lifetime.active.Load() != active || f.left.session.lifetime.checked.Load() != checked || f.left.session.lifetime.wall.Load() != wall {
		t.Fatal("local sample observed Registry or extended published lifetime")
	}
	copyOwner.Close()
	if stamp, err = f.left.LocalStamp(); err == nil || stamp != (registry010.Stamp{}) {
		t.Fatal("copied close did not retire the shared owner", stamp, err)
	}
}

func TestHTTPApplicationLocalStampRefusals010(t *testing.T) {
	for _, mode := range []string{"nil", "zero", "wrong-mode", "missing-endpoint", "closed", "retired", "clock-error", "clock-panic", "negative-mono", "wall-rollback", "mono-rollback", "idle", "lifetime", "pending", "key-expired", "overflow"} {
		t.Run(mode, func(t *testing.T) {
			f := applicationPair010(t, true)
			owner := f.left
			switch mode {
			case "nil":
				owner = nil
			case "zero":
				owner = &HTTPApplicationOwner010{}
			case "wrong-mode":
				copySession := *f.left.session
				copySession.application = false
				owner = &HTTPApplicationOwner010{session: &copySession}
			case "missing-endpoint":
				owner = &HTTPApplicationOwner010{session: &AuthenticatedCompletion010{application: true}}
			case "closed":
				owner.Close()
			case "retired":
				owner.session.endpoint.Close()
			case "clock-error":
				owner.session.endpoint.clock = applicationLocalClock010(func() (registry010.Stamp, error) { return registry010.Stamp{}, errors.New("fixture clock unavailable") })
			case "clock-panic":
				owner.session.endpoint.clock = applicationLocalClock010(func() (registry010.Stamp, error) { panic("fixture clock panic") })
			case "negative-mono":
				f.control.mono = -1
			case "wall-rollback":
				f.control.utc = 99
			case "mono-rollback":
				f.control.mono = 10
				if _, err := owner.LocalStamp(); err != nil {
					t.Fatal(err)
				}
				f.control.mono = 9
			case "idle":
				f.control.mono = 600000
			case "lifetime":
				// Keep idle fresh to isolate the hard lifetime boundary.
				f.control.mono = 3600000
				owner.session.lifetime.active.Store(3600000)
			case "pending":
				owner = f.right
				f.control.mono = 300000
				f.control.utc = 400
			case "key-expired":
				// Establish actual pinned expiry through a new valid handshake.
				a, b, c := completionPair(t)
				c.expiry = 101
				pending, hello, err := a.Start(context.Background(), completionBob, completionBob+"#signing-1", 300)
				if err != nil {
					t.Fatal(err)
				}
				responder, reply, err := b.Respond(context.Background(), hello, 300)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(responder.Close)
				session, err := pending.Complete(context.Background(), reply)
				if err != nil {
					t.Fatal(err)
				}
				owner, err = session.TakeHTTPApplication("https://agent.example/messages")
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(owner.Close)
				c.utc = 101
			case "overflow":
				n := int64((1<<63-1)/time.Millisecond) + 1
				// Isolate Duration representability while retaining local liveness.
				owner.session.created.MonoMS = n
				owner.session.lifetime.active.Store(n)
				owner.session.lifetime.checked.Store(n)
				f.control.mono = n
			}
			if stamp, err := owner.LocalStamp(); err == nil || stamp != (registry010.Stamp{}) {
				t.Fatal("invalid sample released", stamp, err)
			}
			if now, err := owner.LocalNow(); err == nil || now != 0 {
				t.Fatal("LocalNow differs from local sample refusal", now, err)
			}
		})
	}
}

func TestHTTPApplicationLocalStampIsNotAuthority010(t *testing.T) {
	f := applicationPair010(t, true)
	f.control.mode = "source-error"
	before := f.reads.calls.Load()
	if stamp, err := f.left.LocalStamp(); err != nil || stamp != (registry010.Stamp{MonoMS: 0, Unix: 100}) {
		t.Fatal("local clock unnecessarily depended on Source", stamp, err)
	}
	if f.reads.calls.Load() != before {
		t.Fatal("local sample performed a Source lookup")
	}
	if f.left.Check(context.Background()) == nil || f.reads.calls.Load() <= before {
		t.Fatal("local sample granted authority despite unavailable Source")
	}
	if _, err := f.left.LocalStamp(); err == nil {
		t.Fatal("local sample revived a failed owner")
	}
}

func TestHTTPApplicationLocalStampConcurrentProgress010(t *testing.T) {
	f := applicationPair010(t, true)
	// Deterministic interleaving: validation publishes a newer bound after this
	// timer captured its old bounds but before its original clock sample returns.
	calls := 0
	f.left.session.endpoint.clock = applicationLocalClock010(func() (registry010.Stamp, error) {
		calls++
		if calls == 1 {
			f.left.session.lifetime.checked.Store(10)
			f.left.session.lifetime.active.Store(10)
			f.left.session.lifetime.wall.Store(101)
			return registry010.Stamp{MonoMS: 5, Unix: 100}, nil
		}
		return registry010.Stamp{MonoMS: 10, Unix: 101}, nil
	})
	if stamp, err := f.left.LocalStamp(); err != nil || stamp != (registry010.Stamp{MonoMS: 5, Unix: 100}) {
		t.Fatal("concurrent publication mistaken for rollback", stamp, err)
	}
	if stamp, err := f.left.LocalStamp(); err != nil || stamp != (registry010.Stamp{MonoMS: 10, Unix: 101}) || calls != 2 {
		t.Fatal("new publication not retained", stamp, err, calls)
	}
}

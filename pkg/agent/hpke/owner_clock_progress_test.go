package hpke

import (
	"context"
	"testing"

	"github.com/sage-x-project/sage/pkg/agent/registry010"
)

// This is a schedule of ordinary local validation and timer observations,
// using the established inert session fixture. No peer input is modified.
func TestNonHTTPOwnerTimerProgressPreservesValidationStart(t *testing.T) {
	a, _, clock := recordPair010(t, 0)
	owner, err := a.TakeNonHTTP()
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	clock.mono = 1000
	start, err := owner.session.endpoint.sample()
	if err != nil {
		t.Fatal(err)
	}
	clock.mono = 1001
	if _, err = owner.LocalNow(); err != nil {
		t.Fatal(err)
	}
	if err = owner.session.recordLive(start); err != nil {
		t.Fatal("normal timer progress invalidated validation start", err)
	}
	clock.mono = 1002
	if _, err = owner.Observe(context.Background()); err != nil {
		t.Fatal("fresh final observation refused", err)
	}
}

func TestNonHTTPOwnerTimerProgressStillRejectsProtocolRollback(t *testing.T) {
	for _, kind := range []string{"monotonic", "wall"} {
		t.Run(kind, func(t *testing.T) {
			a, _, clock := recordPair010(t, 0)
			owner, err := a.TakeNonHTTP()
			if err != nil {
				t.Fatal(err)
			}
			defer owner.Close()
			clock.mono, clock.utc = 1000, 101
			if _, err = owner.LocalNow(); err != nil {
				t.Fatal(err)
			}
			clock.mono, clock.utc = 1001, 101
			if kind == "monotonic" {
				clock.mono = 999
			} else {
				clock.utc = 100
			}
			if owner.Check(context.Background()) == nil {
				t.Fatal("protocol validation accepted rollback after timer")
			}
		})
	}
}

type ownerClockPublication struct {
	clock *completionControl
	owner *NonHTTPOwner010
}

func (c ownerClockPublication) Now() (registry010.Stamp, error) {
	stamp, err := c.clock.Now()
	// Deterministic publication just after this timer sampled its clock.
	// The next sample advances normally; this is neither a clock rollback
	// nor a reentrant call into the clock, endpoint or native coordinator.
	c.owner.session.lifetime.checked.Store(stamp.MonoMS + 1)
	c.owner.session.lifetime.active.Store(stamp.MonoMS + 1)
	c.clock.mono = stamp.MonoMS + 1
	return stamp, err
}

func TestNonHTTPOwnerValidationPublicationPreservesTimerSample(t *testing.T) {
	a, _, clock := recordPair010(t, 0)
	owner, err := a.TakeNonHTTP()
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	clock.mono = 1000
	owner.session.endpoint.clock = ownerClockPublication{clock: clock, owner: owner}
	if _, err = owner.LocalNow(); err != nil {
		t.Fatal("new validation publication invalidated earlier timer sample", err)
	}
	owner.session.endpoint.clock = clock
	if _, err = owner.LocalNow(); err != nil {
		t.Fatal("following fresh timer sample refused", err)
	}
}

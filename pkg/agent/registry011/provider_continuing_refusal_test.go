package registry011

import (
	"context"
	"testing"
)

// Faults are fixed finite unit states. There is no runtime hostile peer,
// generated exploit, swapped filesystem or means to bypass a host control.
func TestProviderContinuingAdmissionRefusesUnavailablePrefixesWithoutReset(t *testing.T) {
	for _, mode := range []string{"small-original-claim", "prefix-count", "prefix-main", "clock-capacity", "budget", "already-started"} {
		t.Run(mode, func(t *testing.T) {
			var s *providerCustodyStore
			var g *providerGuardianOwner
			if mode == "small-original-claim" {
				s, g = providerGuardianSettledFixture(t)
			} else {
				s, g = providerContinuingSettledFixture(t, true)
			}
			stored, origin, sequence := s.stored, s.setupOrigin, s.sequence
			switch mode {
			case "prefix-count":
				g.prefix.counts[0]++
			case "prefix-main":
				g.prefix.main.ordinal++
			case "clock-capacity":
				g.clock.mu.Lock()
				g.clock.maximum = g.clock.read + g.bounds.claim.clocks + 10
				g.clock.mu.Unlock()
			case "budget":
				g.integration.ledger.current = nil
			case "already-started":
				g.integration.bootstrap = admissionTestDigest("a")
			}
			o, err := newProviderContinuingBootstrapCandidate(s)
			if err == nil || o != nil || s.sequence != sequence || s.setupOrigin != origin || s.stored != stored || g.ordinal > 1 || s.last != nil {
				t.Fatal("unavailable prefix reset an origin, admitted T or changed a hold", mode, err)
			}
			if mode != "budget" && mode != "already-started" && (!g.failed || !s.failed || g.ordinal != 1 || s.activeMeter.Load() != nil) {
				t.Fatal("consumed G was recovered after a refused continuation", mode)
			}
		})
	}
	if o, err := newProviderContinuingBootstrapCandidate(nil); err == nil || o != nil {
		t.Fatal("missing store created a continuing owner")
	}
}

func TestProviderContinuingExecutionRefusesFiniteUnavailableState(t *testing.T) {
	for _, mode := range []string{"context", "cancelled", "ticket", "purpose", "guardian", "ticket-original", "binding-original", "metadata", "prefix", "clock", "execution-quota"} {
		t.Run(mode, func(t *testing.T) {
			s, o := providerContinuingPreparedFixture(t)
			stored, origin := s.stored, s.setupOrigin
			ctx := context.Background()
			switch mode {
			case "context":
				ctx = nil
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			case "ticket":
				o.ticket = nil
			case "purpose":
				o.binding.purpose = "unsupported-unit-purpose"
			case "guardian":
				o.guardian.pin.ordinal++
			case "ticket-original":
				delete(s.objects, "ticket:"+o.ticket.digest)
			case "binding-original":
				for key, ref := range s.objects {
					if ref.kind == "guardian-admission-binding" {
						delete(s.objects, key)
					}
				}
			case "metadata":
				o.claim.metadata++
			case "prefix":
				o.guardian.prefix.mapSizes[0]++
			case "clock":
				o.clock.failed = true
			case "execution-quota":
				o.execution.actual[2] = o.execution.upper[2]
			}
			result, err := o.executeBootstrapCandidate(ctx)
			if err == nil || result != nil || !s.failed || !o.guardian.failed || o.state != providerContinuingFailed ||
				s.stored != stored || s.setupOrigin != origin || s.sequence != 5 || s.last != nil || s.activeMeter.Load() != nil ||
				o.execution.actual[0] != 0 || o.execution.actual[1] != 0 || o.integration.bootstrap != "" {
				t.Fatal("unavailable finite state dispatched, refunded or published", mode, err, s.sequence)
			}
		})
	}
}

func TestProviderContinuingPrivateConsumptionBurnsAtMostOnce(t *testing.T) {
	s, o := providerContinuingPreparedFixture(t)
	s.mu.Lock()
	defer s.mu.Unlock()
	permit, err := o.consumeLocked()
	if err != nil || permit == nil || permit.used || permit.ack != nil || s.sequence != 6 || !o.compositeLocked() {
		t.Fatal("C did not retain its separate private receipt", err)
	}
	if again, err := o.consumeLocked(); err == nil || again != nil || s.sequence != 6 || o.beginRemainingLocked(permit) {
		t.Fatal("C was consumed twice or R opened before burn")
	}
	if !o.burnLocked(permit) || !permit.used || o.burnLocked(permit) || o.execution.actual[8] != o.ticket.segment-o.previousTicket.segment || o.execution.actual[0] != 0 || o.execution.actual[1] != 0 {
		t.Fatal("local permit repeated or invented transport")
	}
	if !o.beginRemainingLocked(permit) || o.beginRemainingLocked(permit) || !o.execution.closed || o.remaining == nil || !o.fits(o.remaining) || o.releaseLocked() {
		t.Fatal("remaining reused E or released before retained S")
	}
	o.failLocked()
}

func TestProviderContinuingBurnRefusalNeverRestoresThePermit(t *testing.T) {
	for _, mode := range []string{"prefix", "clock", "request"} {
		t.Run(mode, func(t *testing.T) {
			s, o := providerContinuingPreparedFixture(t)
			s.mu.Lock()
			defer s.mu.Unlock()
			permit, err := o.consumeLocked()
			if err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "prefix":
				o.guardian.prefix.counts[1]++
			case "clock":
				o.clock.failed = true
			case "request":
				o.request = &providerIntegrationRequest{}
				permit.request = o.request
			}
			expected := providerContinuingUsed
			if mode == "clock" {
				expected = providerContinuingFailed // The owned sampler quarantines its caller.
			}
			if o.burnLocked(permit) || !permit.used || o.state != expected || o.burnLocked(permit) || s.sequence != 6 {
				t.Fatal("post-burn refusal recovered a permission", mode)
			}
			o.failLocked()
		})
	}
}

func TestProviderContinuingMissingAndCopiedOwnersCannotSupplyAuthority(t *testing.T) {
	var missing *providerContinuingOwner
	if missing.selectedLocked() || missing.fits(nil) || missing.writingLocked(nil) || missing.consumeStorageLocked(1) || missing.compositeLocked() ||
		missing.burnLocked(nil) || missing.beginRemainingLocked(nil) || missing.releaseLocked() || missing.consumeGuardianStorageLocked(1) {
		t.Fatal("missing private owner accepted candidate DATA")
	}
	if _, err := missing.sampleLocked(); err == nil {
		t.Fatal("missing owner supplied a clock")
	}
	if result, err := missing.executeBootstrapCandidate(context.Background()); err == nil || result != nil {
		t.Fatal("missing owner executed local work")
	}
	missing.failLocked()
	s, o := providerContinuingPreparedFixture(t)
	copy := *o // This struct has no mutex; the private owner self-pointer stays original.
	if copy.selectedLocked() || copy.fits(o.execution) || copy.writingLocked(s) || copy.consumeStorageLocked(1) || copy.compositeLocked() {
		t.Fatal("copied DATA restored a private continuing role")
	}
}

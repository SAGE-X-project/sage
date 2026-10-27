package registry011

import (
	"context"
	"testing"
)

// Fixed local quota/metadata states and ordinary owned unavailable paths only.
// No hostile transport, changed executable, host bypass or retry is used.
func TestProviderContinuingBootstrapKeepsHoldsOnLateLocalRefusal(t *testing.T) {
	for _, mode := range []string{"policy", "blocks", "measurement", "settlement-path", "last-clock"} {
		t.Run(mode, func(t *testing.T) {
			s, o := providerContinuingPreparedFixture(t)
			stored, origin := s.stored, s.setupOrigin
			switch mode {
			case "policy":
				policy := *o.integration.policy
				policy.raw = "null"
				o.integration.policy = &policy
			case "blocks":
				o.execution.actual[3] = o.execution.upper[3]
			case "measurement":
				o.execution.reservation[10]++
			case "settlement-path":
				if s.custody.root.Mkdir(s.name("intent", 7), 0700) != nil {
					t.Fatal("owned unavailable S path")
				}
			case "last-clock":
				o.clock.mu.Lock()
				o.clock.maximum = o.clock.read + 9
				o.clock.mu.Unlock()
			}
			data, err := o.executeBootstrapCandidate(context.Background())
			if err == nil || data != nil || !s.failed || !o.guardian.failed || o.state != providerContinuingFailed ||
				s.stored != stored || s.setupOrigin != origin || o.permit == nil || !o.permit.used || !o.consumption ||
				s.sequence < 6 || s.sequence > 7 || s.activeMeter.Load() != nil || o.integration.bootstrap != "" {
				t.Fatal("late refusal refunded, retried or published local work", mode, err, s.sequence)
			}
		})
	}
}

func TestProviderContinuingAdmissionCannotSkipUnavailableGuardianOrMainOriginals(t *testing.T) {
	for _, mode := range []string{"claim-path", "main-path", "guardian-cache", "prior-guardian"} {
		t.Run(mode, func(t *testing.T) {
			s, g := providerContinuingSettledFixture(t, true)
			stored, origin := s.stored, s.setupOrigin
			switch mode {
			case "claim-path":
				if s.custody.root.Mkdir(s.name("guardian-intent", 1), 0700) != nil {
					t.Fatal("owned unavailable G path")
				}
			case "main-path":
				if s.custody.root.Mkdir(s.name("intent", 5), 0700) != nil {
					t.Fatal("owned unavailable T path")
				}
			case "guardian-cache":
				s.frontierReplay.body.guardian = nil
			case "prior-guardian":
				s.frontierReplay.body.guardian.head.ordinal = g.bounds.attempts
			}
			data, err := newProviderContinuingBootstrapCandidate(s)
			if err == nil || data != nil || !s.failed || !g.failed || g.ordinal != 1 || g.ready ||
				s.stored <= stored || s.setupOrigin != origin || s.sequence != 4 || s.last != nil || s.activeMeter.Load() != nil {
				t.Fatal("consumed G/T uncertainty recovered a hold or admitted a ticket", mode, err, s.sequence)
			}
		})
	}
}

func TestProviderContinuingHelpersRejectMissingOrUnretainedChanges(t *testing.T) {
	s, o := providerContinuingPreparedFixture(t)
	s.mu.Lock()
	defer s.mu.Unlock()
	defer o.failLocked()
	if _, err := o.failedBootstrapLocked(nil); err == nil {
		t.Fatal("missing burn receipt obtained failed settlement")
	}
	if _, err := providerContinuingBootstrapObject(nil, FinalizedHeader{}, nil); err == nil {
		t.Fatal("missing typed header became a bootstrap")
	}
	if o.commitLocked("unknown", admissionTestDigest("a"), nil, nil, CandidateClockDiagnostic{}) == nil {
		t.Fatal("unknown transition obtained retention")
	}
	root, state := o.integration.root.raw, o.state
	o.integration.root.raw, o.state = "null", providerContinuingConsumption
	err := o.commitLocked("consume", admissionTestDigest("a"), o.integration.ledger.current, []providerIntegrationObjectInput{{"consumption", []byte(`{"format":"sage-provider-budget-consumption-v1"}`)}}, s.frontierReplay.current.clock)
	o.integration.root.raw, o.state = root, state
	if err == nil || s.sequence != 5 || o.change != nil || o.permit != nil || o.execution.actual[9] != 0 {
		t.Fatal("unretained malformed proposal changed custody")
	}
	limits := o.integration.ledger.limits[1]
	o.integration.ledger.limits[1] = candidateWorkVector{}
	_, ok := providerContinuingSegment(o.integration.ledger, o.plan.reservation)
	o.integration.ledger.limits[1] = limits
	if ok {
		t.Fatal("exhausted segment invented a new allowance")
	}
}

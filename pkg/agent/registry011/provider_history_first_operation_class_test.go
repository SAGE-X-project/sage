package registry011

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func TestProviderHistoryFirstOperationClassPreservesLargeObjectsAndMaximumMembers(t *testing.T) {
	for _, name := range []string{"maximum-members", "multiple-stages", "recovery-before-admission", "first-finalized"} {
		t.Run(name, func(t *testing.T) {
			var s *providerCustodyStore
			switch name {
			case "maximum-members":
				o, d := historyMemberDispatchFixture(t, providerHistoryGroupMembers)
				s = o.store
				if d == nil || d.order.count != 32 || d.coordinator.ticket.table.count != 32 || !d.wholeResultPackFitsLocked() ||
					o.phase != "dispatch-prepared" || d.meter != o.meter || d.coordinator == d.custodian {
					t.Fatal("original32-member T/C/dispatch and complete pack envelope required")
				}
			case "multiple-stages":
				o := historyMultistageOwner(t, 3)
				s = o.store
				if o.parent.bounds.stages != 3 || o.phase != "prepared" {
					t.Fatal("original three-stage input and initial owner required")
				}
			case "recovery-before-admission":
				o, _ := historyRecoveryBeforeAdmission(t)
				s = o.store
				if o.phase != "prepared" || o.parent.recovery.initial == nil {
					t.Fatal("original pre-admission recovery checkpoint required")
				}
			case "first-finalized":
				s = historyFirstFinalizedStore(t, historyFirstFinalizedResponse(t), nil)
			}
			p, b := s.history, s.config.bounds
			first := p.firstOwner
			// Independent literal from the fixed header class and unchanged
			// global1MiB headroom. Startup remains its old larger ceiling.
			want := candidateWorkVector{0: 1, 1: 32768, 2: 103465298, 8: 1, 9: 817372, 10: 11727834507083}
			if b.maximumObjectBytes != 1<<20 || first == nil || first != s.firstOperation || first.state != providerFirstOperationDone ||
				first.plan.reservation.work != want || first.plan.headroom != 1099512676354 || p.bounds.startup.work[2] != 1177583354 ||
				first.limits.operation[2] != 256<<20 || first.limits.segment[2] != 256<<20 || p.bounds.startup.retained != 948444 ||
				!p.startupClaimed || s.stored != p.held.retained || first.origin != p.original || s.setupOrigin != p.original ||
				first.clock != p.clock || first.clock != s.clock || s.activeMeter.Load() != nil || s.failed || p.failed {
				t.Fatal("first class borrowed a smaller global object, limit, original hold or owner", name)
			}
			for coordinate, amount := range first.plan.reservation.work {
				if amount > first.limits.operation[coordinate] || amount > first.limits.segment[coordinate] || amount > p.bounds.startup.work[coordinate] {
					t.Fatal("complete original class plan exceeds an unchanged enclosing coordinate", coordinate)
				}
			}
		})
	}
}

func historyClassFirstPInspection(t *testing.T, decoded uint64) *providerCustodyStore {
	t.Helper()
	_, capture := providerFrontierBoundsFixture()
	capture.Timeout = 10 * time.Second
	return providerHistoryProvisionedInspectionFixture(t, providerScanFixtureSelection{
		capture: &capture, pin: true, pinLargeObject: true, historyProgram: true,
		historyGroupWrite: true, historyGroupConsume: true, historyGroupDispatch: true,
		historyRecoveryNamespace: true, historyRecoveryLegacy: true, historyFirstPDecoded: decoded,
	})
}

func TestProviderHistoryFirstOperationClassChecksExactAndShortOriginalPolicy(t *testing.T) {
	for _, name := range []string{"exact", "one-short"} {
		t.Run(name, func(t *testing.T) {
			limit := uint64(103465298)
			if name == "one-short" {
				limit--
			}
			// Both policy levels are selected BEFORE profile/anchor/H0.
			// Equal levels retain the policy's operation<=segment contract.
			s := historyClassFirstPInspection(t, limit)
			p, f := s.history, s.firstAdmissionOwner
			stored, reads, pin, actual, origin := s.stored, s.clock.read, s.frontierFiles.pin, f.meter.actual, p.original
			before := providerHistoryProvisionedOriginalBytes(t, s.config)
			if s.firstLimits.operation[2] != limit || s.firstLimits.segment[2] != limit || s.config.bounds.maximumObjectBytes != 1<<20 {
				t.Fatal("finite original policy was not selected before inspection")
			}
			first, err := newProviderFirstOperationCandidate(s)
			if name == "one-short" {
				if err == nil || first != nil || !s.failed || !p.failed || p.startupClaimed || s.firstOperation != nil ||
					f.state != providerFirstAdmissionFailed || f.replayStarted || f.custodianStarted || f.plan != nil ||
					f.meter.actual != actual || s.clock.read != reads || s.stored != stored || s.frontierFiles.pin != pin ||
					s.activeMeter.Load() != nil || !reflect.DeepEqual(before, providerHistoryProvisionedOriginalBytes(t, s.config)) {
					t.Fatal("one missing decoded byte reached I/T, replaced the original observer or refunded the hold", err)
				}
				if again, err := newProviderFirstOperationCandidate(s); err == nil || again != nil ||
					f.meter.actual != actual || s.clock.read != reads || s.stored != stored {
					t.Fatal("failed original policy was revived by another constructor", err)
				}
				return
			}
			if err != nil || first == nil || first.plan.reservation.work[2] != limit {
				t.Fatal("exact original decoded policy should reach its closed first operation", err)
			}
			conn := providerIntegrationOwnedTLS(t, providerFirstOperationRequestBody, providerFirstOperationResponse())
			if result, err := first.executeCandidateWithConnection(context.Background(), conn); err != nil || result == nil ||
				first.state != providerFirstOperationDone || p.firstOwner != first || !p.startupClaimed || p.start == nil ||
				first.origin != origin || p.original != origin || first.clock != s.clock || s.stored != stored ||
				s.clock.read <= reads || s.activeMeter.Load() != nil || s.failed || p.failed {
				t.Fatal("exact class policy failed original C/X/S or renewed its lifetime", err)
			}
		})
	}
}

func TestProviderHistoryFirstOperationClassKeepsGenericQuoteAndWholeZero(t *testing.T) {
	s := historyClassFirstPInspection(t, 0)
	b, capture, f := s.config.bounds, s.config.capture, s.firstAdmissionOwner
	config, stored, reads, actual := s.config, s.stored, s.clock.read, f.meter.actual
	before := providerHistoryProvisionedOriginalBytes(t, s.config)
	class, ok := providerFirstOperationPlanUpper(b, capture, f.scanPin.ordinal, f.records, s.retainedRaw, f.readerOutput)
	// This copied tuple is inert DATA. Removing history from the copy cannot
	// change the live original profile/clock or authorize a generic operation.
	genericBounds := b
	genericBounds.history = nil
	generic, genericOK := providerFirstOperationPlanUpper(genericBounds, capture, f.scanPin.ordinal, f.records, s.retainedRaw, f.readerOutput)
	wantClass := candidateWorkVector{0: 1, 1: 32768, 2: 103465298, 8: 1, 9: 817372, 10: 11727834507083}
	wantGeneric := candidateWorkVector{0: 1, 1: 32768, 2: 1177583354, 8: 1, 9: 817372, 10: 1236965042337323}
	if !ok || !genericOK || class.reservation.work != wantClass || generic.reservation.work != wantGeneric ||
		class.storage != generic.storage || class.capture != generic.capture || class.headroom != generic.headroom ||
		class.firstExtra != generic.firstExtra || class.headroom != 1099512676354 ||
		generic.reservation.work[2] <= s.firstLimits.operation[2] || class.reservation.work[2] > s.firstLimits.operation[2] {
		t.Fatal("typed first-P normalization changed generic output, headroom or disjoint extra work", class, generic)
	}
	for _, name := range []string{"ordinal", "output-overflow", "capture"} {
		ordinal, output, selected := f.scanPin.ordinal, f.readerOutput, capture
		switch name {
		case "ordinal":
			ordinal = 0
		case "output-overflow":
			output = ^uint64(0)
		case "capture":
			selected.Timeout = 0
		}
		if got, ok := providerFirstOperationPlanUpper(b, selected, ordinal, f.records, s.retainedRaw, output); ok || got != (providerFirstOperationPlan{}) {
			t.Fatal("invalid or overflowing history child leaked a partial plan", name, got)
		}
	}
	if s.config != config || s.stored != stored || s.clock.read != reads || f.meter.actual != actual ||
		s.firstOperation != nil || s.history.startupClaimed || s.activeMeter.Load() != nil ||
		!reflect.DeepEqual(before, providerHistoryProvisionedOriginalBytes(t, s.config)) {
		t.Fatal("numeric class/generic planning changed an original or consumed permission")
	}
}

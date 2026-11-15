package registry011

import (
	"context"
	"reflect"
	"strconv"
	"testing"
)

// Fixed, in-memory UNIT faults only. Files, checkpoint descriptors and the
// original meter/limits are unchanged. Restoration uses independent copies,
// never values read back from the implementation's baseline.
func historyCurrentClaimUnitFault(t *testing.T, o *providerHistoryGroupOwner, mode string) func() {
	t.Helper()
	a, c := o.admission.claim, o.last.claim
	oldA, oldC, oldFrame := *a, *c, *c.frame
	oldAModel, oldCModel := *a.coordinator, *c.coordinator
	oldATicket := *a.coordinator.ticket
	oldFrontier, oldCustodian := *o.store.frontierFiles, *o.custodian
	switch mode {
	case "T-purpose":
		a.purpose = "identity"
	case "T-highwater":
		a.highwater += " "
	case "C-highwater":
		c.highwater += " "
	case "C-clock":
		c.frame.clock.WallMS++
	case "T-model":
		a.coordinator.head = admissionTestDigest("f")
	case "HEAD-raw":
		o.store.frontierFiles.pinRaw += " "
	case "group-coordinator-total":
		o.coordinator.charged.work[10]++
	case "group-custodian-total":
		o.custodian.charged.work[10]++
	case "C-model":
		c.coordinator.head = admissionTestDigest("f")
	case "T-table":
		a.coordinator.ticket.table.digest = admissionTestDigest("f")
	default:
		t.Fatal("unknown fixed UNIT fault", mode)
	}
	return func() {
		*a, *c, *c.frame = oldA, oldC, oldFrame
		*a.coordinator, *c.coordinator, *a.coordinator.ticket = oldAModel, oldCModel, oldATicket
		*o.store.frontierFiles, *o.custodian = oldFrontier, oldCustodian
	}
}

func TestProviderHistoryMemberReplayBindsCurrentFullReaderClaimInputs(t *testing.T) {
	for _, mode := range []string{"T-purpose", "T-highwater", "C-highwater", "C-clock", "T-model", "HEAD-raw", "group-coordinator-total", "group-custodian-total"} {
		for role := range 2 {
			t.Run(mode+"-"+strconv.Itoa(role), func(t *testing.T) {
				o, d := historyMemberDispatchFixture(t, 1)
				historyMemberReplayEnter(t, o, d)
				oracle, err := o.replayCommittedConsumptionLocked(context.Background())
				if err != nil || oracle == nil {
					t.Fatal("genuine unchanged full-reader positive prerequisite", err)
				}
				if got, err := d.replayMemberEquivalentLocked(context.Background(), role); err != nil || got == nil || got.head != oracle.head {
					t.Fatal("genuine selected equivalent positive prerequisite", role, err)
				}
				restore := historyCurrentClaimUnitFault(t, o, mode)
				if !o.committedConsumptionShapeLocked() || !d.memberReplaySelectedLocked() || !o.activeLocked(o.meter) {
					t.Fatal("this fault must reach the observed boundary, not an unrelated scalar rejection", mode)
				}
				if got, err := o.replayCommittedConsumptionLocked(context.Background()); err == nil || got != nil || o.meter.failed {
					t.Fatal("original full replay did not independently refuse the changed expectation", mode, err)
				}
				before, clock, sent, next := o.meter.actual, o.store.clock.read, o.order.sent, o.order.next
				if got, err := d.replayMemberEquivalentLocked(context.Background(), role); err == nil || got != nil || o.meter.failed || o.meter.closed ||
					o.meter.actual[0] != before[0] || o.meter.actual[1] != before[1] || o.meter.actual[10] <= before[10] ||
					o.store.clock.read != clock || o.order.sent != sent || o.order.next != next || d.retained != 0 {
					t.Fatal("changed current expectation escaped, was unobserved, dispatched, or reset", mode, role, err)
				}
				for i, n := range before {
					if o.meter.actual[i] < n {
						t.Fatal("refunded original work", i)
					}
				}
				restore()
				if got, err := d.replayMemberEquivalentLocked(context.Background(), role); err != nil || got == nil || got.head != oracle.head {
					t.Fatal("independently restored UNIT values lost the original positive", mode, role, err)
				}
			})
		}
	}
}

func TestProviderHistoryMemberClaimFaultQuarantinesAtRealPrepareEntry(t *testing.T) {
	for _, mode := range []string{"T-purpose", "T-highwater", "C-highwater", "C-clock", "T-model", "HEAD-raw", "group-coordinator-total", "group-custodian-total"} {
		t.Run(mode, func(t *testing.T) {
			o, d := historyMemberDispatchFixture(t, 1)
			pin, ack, held, before, clocks := o.pin, o.last, o.store.stored, o.meter.actual, o.store.clock.read
			q := d.memberReplaySeal
			historyCurrentClaimUnitFault(t, o, mode) // Entered failure is never restored/reused.
			member := d.coordinator.ticket.table.members[0]
			conn := providerIntegrationOwnedTLS(t, member.request.body, `{"jsonrpc":"2.0","id":1,"result":"0xaa36a7"}`)
			defer conn.Close()
			out, err := o.retainMemberCandidateWithConnection(context.Background(), d, 0, conn)
			if err == nil || out != nil || !o.failed || !o.store.failed || !o.meter.failed || !o.meter.closed || !q.revoked ||
				d.memberReplaySeal != nil || o.parent.recovery.memberReplaySeal != nil || q.claims != ([2]providerHistoryMemberClaimInput{}) ||
				o.pin != pin || o.last != ack || o.store.stored != held || o.order.sent != 0 || o.order.next != 0 || d.retained != 0 ||
				o.meter.actual[0] != before[0] || o.meter.actual[1] != before[1] || o.meter.actual[10] <= before[10] ||
				o.store.clock.read != clocks+1 || o.store.activeMeter.Load() != nil {
				t.Fatal("real prepare did not quarantine before request consumption or transport", mode, err)
			}
			for i, n := range before {
				if o.meter.actual[i] < n {
					t.Fatal("entered failure refunded work", i)
				}
			}
		})
	}
}

func TestProviderHistoryCurrentClaimModelAndTableAreAlsoScalarBound(t *testing.T) {
	for _, mode := range []string{"C-model", "T-table"} {
		t.Run(mode, func(t *testing.T) {
			o, d := historyMemberDispatchFixture(t, 1)
			historyMemberReplayEnter(t, o, d)
			if got, err := o.replayCommittedConsumptionLocked(context.Background()); err != nil || got == nil {
				t.Fatal("original full positive prerequisite", err)
			}
			restore := historyCurrentClaimUnitFault(t, o, mode)
			if o.committedConsumptionShapeLocked() {
				t.Fatal("the existing scalar model/table provenance guard must refuse", mode)
			}
			before, clock := o.meter.actual, o.store.clock.read
			if got, err := o.replayCommittedConsumptionLocked(context.Background()); err == nil || got != nil {
				t.Fatal("old full reader scalar guard", mode, err)
			}
			if got, err := d.replayMemberEquivalentLocked(context.Background(), 0); err == nil || got != nil ||
				o.meter.actual != before || o.store.clock.read != clock || o.meter.failed {
				t.Fatal("scalar rejection performed observation or was mistaken for the new boundary", mode, err)
			}
			restore()
			if !d.memberReplaySeal.claimsMatchObservedLocked(o.meter) {
				t.Fatal("restored original claim expectations", mode)
			}
		})
	}
}

func TestProviderHistoryMemberReplayKeepsOriginalFrontierAndGroupModelSelection(t *testing.T) {
	for _, mode := range []string{"nil-frontier", "frontier-copy", "custodian-copy"} {
		t.Run(mode, func(t *testing.T) {
			o, d := historyMemberDispatchFixture(t, 1)
			historyMemberReplayEnter(t, o, d)
			if got, err := o.replayCommittedConsumptionLocked(context.Background()); err != nil || got == nil {
				t.Fatal("genuine full original prerequisite", err)
			}
			if got, err := d.replayMemberEquivalentLocked(context.Background(), 0); err != nil || got == nil {
				t.Fatal("genuine equivalent prerequisite", err)
			}
			frontier, custodian := o.store.frontierFiles, o.custodian
			frontierCopy, custodianCopy := *frontier, *custodian
			frontierCopy.owner, custodianCopy.owner = &frontierCopy, &custodianCopy
			switch mode {
			case "nil-frontier":
				o.store.frontierFiles = nil
			case "frontier-copy":
				o.store.frontierFiles = &frontierCopy
			case "custodian-copy":
				o.custodian = &custodianCopy
			}
			before, clocks := o.meter.actual, o.store.clock.read
			if d.memberReplaySelectedLocked() {
				t.Fatal("a copy with repaired self-owner replaced an original selector", mode)
			}
			if got, err := d.replayMemberEquivalentLocked(context.Background(), 0); err == nil || got != nil ||
				o.meter.actual != before || o.store.clock.read != clocks || o.meter.failed {
				t.Fatal("detached original selector reached observation", mode, err)
			}
			o.store.frontierFiles, o.custodian = frontier, custodian
			if got, err := d.replayMemberEquivalentLocked(context.Background(), 0); err != nil || got == nil {
				t.Fatal("restored original selector was not positive", mode, err)
			}
		})
	}
}

func TestProviderHistoryMemberClaimProjectionRefusesIncompleteAndUnfundedInputs(t *testing.T) {
	o, d := historyMemberDispatchFixture(t, 1)
	historyMemberReplayEnter(t, o, d)
	m, q, a := o.meter, d.memberReplaySeal, o.admission
	if !q.claimsMatchObservedLocked(m) {
		t.Fatal("original projected input prerequisite")
	}
	var absent *providerHistoryMemberReplaySeal
	for _, incomplete := range []*providerHistoryMemberReplaySeal{nil, {}, {dispatch: &providerHistoryGroupDispatch{}}} {
		if incomplete.captureClaimsObservedLocked(m) || incomplete.claimsMatchObservedLocked(m) {
			t.Fatal("incomplete original owner was accepted")
		}
	}
	if absent.captureClaimsObservedLocked(nil) || q.captureClaimsObservedLocked(nil) || q.claimsMatchObservedLocked(nil) {
		t.Fatal("missing observer was accepted")
	}
	for _, ack := range []*providerHistoryGroupAcknowledgment{nil, {}} {
		if _, ok := providerHistoryMemberClaimInputObservedLocked(ack, m); ok {
			t.Fatal("missing claim was accepted")
		}
	}
	if _, ok := providerHistoryMemberClaimInputObservedLocked(a, nil); ok {
		t.Fatal("missing original observer")
	}
	// These are direct selector faults; no active claim is reconstructed.
	for _, mode := range []string{"owner", "group", "ack-owner", "meter", "frame", "inventory", "coordinator", "coordinator-owner", "ticket"} {
		oldA, oldC, oldModel := *a, *a.claim, *a.claim.coordinator
		c := a.claim
		switch mode {
		case "owner":
			c.owner = nil
		case "group":
			c.group = nil
		case "ack-owner":
			a.owner = nil
		case "meter":
			c.meter = nil
		case "frame":
			c.frame = nil
		case "inventory":
			c.inventory = nil
		case "coordinator":
			c.coordinator = nil
		case "coordinator-owner":
			c.coordinator.owner = nil
		case "ticket":
			c.coordinator.ticket = nil
		}
		before := m.actual
		copy := *q
		if _, ok := providerHistoryMemberClaimInputObservedLocked(a, m); ok || copy.captureClaimsObservedLocked(m) || copy.claimsMatchObservedLocked(m) || m.actual != before || m.failed {
			t.Fatal("incomplete claim reached observation", mode)
		}
		*a, *c = oldA, oldC
		*c.coordinator = oldModel
	}
	// A copied unregistered seal is only a UNIT DATA receiver, not authority.
	copy := *q
	copy.claims, copy.claimComparisons = [2]providerHistoryMemberClaimInput{}, [2]uint64{}
	if !copy.captureClaimsObservedLocked(m) || copy.claims != q.claims || copy.claimComparisons != q.claimComparisons {
		t.Fatal("fixed typed capture differs from the original successful baseline")
	}
	if !q.claimsMatchObservedLocked(m) {
		t.Fatal("selector restoration lost the genuine baseline")
	}
	for _, phase := range []string{"copy", "weight", "compare"} {
		t.Run(phase, func(t *testing.T) {
			o, d := historyMemberDispatchFixture(t, 1)
			historyMemberReplayEnter(t, o, d)
			m, q := o.meter, d.memberReplaySeal
			end, ok := providerUpperSum(o.windowStart[10], o.windowUpper[10])
			remaining := uint64(1)
			if phase != "copy" {
				remaining = uint64(reflect.TypeFor[providerHistoryMemberClaimInput]().Size()) + 1
			}
			if !ok || end < remaining || end-remaining < m.actual[10] || end > m.upper[10] {
				t.Fatal("checked monotone inside-original-window UNIT prefix")
			}
			m.actual[10] = end - remaining
			if !m.preflightWork(candidateWorkVector{}) {
				t.Fatal("positive entered prefix prerequisite")
			}
			before := m.actual
			accepted := false
			if phase == "compare" {
				accepted = q.claimsMatchObservedLocked(m)
			} else {
				copy := *q
				accepted = copy.captureClaimsObservedLocked(m)
			}
			if accepted || !m.failed || m.actual[10] < before[10] || (phase != "copy" && m.actual[10] != end) {
				t.Fatal("unfunded copy, weight or equality did not retain its exact entered prefix", phase)
			}
		})
	}
}

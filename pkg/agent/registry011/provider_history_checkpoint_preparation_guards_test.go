package registry011

import (
	"context"
	"reflect"
	"testing"
)

// Enter the next original C preparation once under this UNIT's genuine T.
// All source operations use the original observer and window. The fresh claim
// is still unselected DATA: no order is consumed and no C file is written.
// Cleanup permanently fails this unused continuation instead of restarting it.
func historyCheckpointConsumptionPreparationUnit(t *testing.T) (*providerHistoryGroupOwner, *providerHistoryWriteClaim) {
	t.Helper()
	o, upper := historyAdmissionWriterFixture(t)
	admission, err := o.retainAdmissionCandidate(context.Background(), "selection", []*providerIntegrationRequest{historyGroupSimple(t)}, upper)
	if err != nil || admission == nil {
		t.Fatal("genuine retained T required", err)
	}
	s, m := o.store, o.meter
	s.mu.Lock()
	t.Cleanup(func() { o.failLocked(); s.mu.Unlock() })
	if o.phase != "admitted" || !o.admissionShapeLocked() || s.activeMeter.Load() != nil || o.parent.recovery.pending != nil {
		t.Fatal("unused original consumption preparation required")
	}
	o.phase = "preparing-consume"
	s.activeMeter.Store(m)
	if !o.fits(m) || !o.identityLocked() {
		t.Fatal("original consumption observer required")
	}
	begin, err := s.samplePrimitive()
	if err != nil || !o.clockFits(begin) {
		t.Fatal("original consumption bracket required", err)
	}
	base, err := o.replayCommittedAdmissionLocked(context.Background(), s.frontierFiles.pinRaw)
	if err != nil {
		t.Fatal("independent retained T reconstruction required", err)
	}
	consumption, err := newProviderHistoryGroupConsumptionObserved(base.ticket, base.head, o.inventory, m)
	if err != nil {
		t.Fatal(err)
	}
	object, err := newProviderHistoryStoredObjectObserved("consumption", consumption.raw, s.config.bounds.maximumObjectBytes, m)
	if err != nil {
		t.Fatal(err)
	}
	inventory, err := newProviderHistoryInventoryObserved(o.inventory, []providerHistoryStoredObject{object}, min(s.config.bounds.maximumMetadataPerCommit, candidateLedgerMetadataLimit), m)
	if err != nil {
		t.Fatal(err)
	}
	coordinator := providerHistoryLedgerCopyObserved(base, m)
	if coordinator == nil || coordinator.consumeObserved(consumption, m) != nil {
		t.Fatal("fresh consumption DATA model required")
	}
	frame, err := newProviderHistoryCustodyFrameObserved(s.config, o.scope, o.pin, base.ticket.table.stage, "consume", o.inventory, inventory.digest, base.head, coordinator.head, consumption.digest, providerClockDiagnostic(begin), m)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := frame.after.bytesObserved(s.config, m)
	charge, ok := providerHistoryStorageChargeFor(inventory, []providerHistoryStoredObject{object}, frame, uint64(len(raw)), s.config.bounds.fileOverheadBytes)
	if err != nil || !ok {
		t.Fatal("original unselected physical charge DATA required", err)
	}
	c := &providerHistoryWriteClaim{group: o, meter: m, frame: frame, inventory: inventory, count: 1,
		admission: admission, consumption: inventory.entries[0], coordinator: coordinator, charge: charge, highwater: providerReplayString(raw, m)}
	c.owner, c.fresh[0] = c, object
	return o, c
}

func TestProviderHistoryCheckpointPreparationRejectsIncompletePhaseAndCapacityData(t *testing.T) {
	for _, mode := range []string{"intact", "admission-phase", "unknown-phase", "missing-admission", "settlement-before-consumption", "physical-overflow", "invalid-completion-count", "missing-highwater"} {
		t.Run(mode, func(t *testing.T) {
			o, claim := historyCheckpointConsumptionPreparationUnit(t)
			s, m, r := o.store, o.meter, o.parent.recovery
			ack, pin, confirmed, held, clocks, upper := o.last, o.pin, r.confirmed, s.stored, s.clock.read, m.upper
			before := providerHistoryProvisionedOriginalBytes(t, s.config)
			switch mode {
			case "admission-phase":
				claim.frame.phase = "admit"
			case "unknown-phase":
				claim.frame.phase = "unknown private phase DATA"
			case "missing-admission":
				claim.admission = nil
			case "settlement-before-consumption":
				claim.frame.phase = "settle"
			case "physical-overflow":
				claim.charge.physical = ^uint64(0)
			case "invalid-completion-count":
				// Invalid bookkeeping remains invalid; it is neither reset nor
				// used to restore a consumed member or a later execution role.
				r.completed = -1
			case "missing-highwater":
				claim.highwater = ""
			}
			work := m.actual
			accepted := r.prepareWriteCheckpointLocked(claim)
			if mode == "intact" {
				if !accepted || claim.checkpoint == nil || r.pending != claim.checkpoint || r.pendingAck != claim.checkpoint.ack ||
					claim.checkpoint.previous != confirmed || claim.checkpoint.frame.phase != "consume" {
					t.Fatal("complete original consumption preparation required")
				}
			} else if accepted || claim.checkpoint != nil || r.pending != nil || r.pendingAck != nil {
				t.Fatal("invalid private preparation published a pending checkpoint", mode)
			}
			if claim.used || claim.renamed || o.last != ack || o.pin != pin || r.confirmed != confirmed || s.stored != held ||
				s.clock.read != clocks || m.upper != upper || m.failed || m.closed || s.activeMeter.Load() != m ||
				o.order != nil || r.order != nil || s.frontierFiles.historyPending != nil ||
				!reflect.DeepEqual(before, providerHistoryProvisionedOriginalBytes(t, s.config)) {
				t.Fatal("checkpoint DATA preparation wrote C, consumed a member or changed original selection", mode)
			}
			for coordinate, value := range work {
				if m.actual[coordinate] < value {
					t.Fatal("checkpoint refusal refunded preparation work", coordinate)
				}
			}
		})
	}
}

func TestProviderHistoryInitialCheckpointRejectsMissingPrivateHeadWithoutPublishingData(t *testing.T) {
	input, _ := historyFirstOriginalClosureUnit(t)
	o, s, r := input.group, input.group.store, input.group.parent.recovery
	scope := providerHistoryCustodyScope{initial: o.start.main, exchange: o.start.exchange, origin: providerClockDiagnostic(o.parent.original)}
	before := providerHistoryProvisionedOriginalBytes(t, s.config)
	head := s.frontierFiles.pinRaw
	if head == "" || r.initial != nil || r.confirmed != nil || !r.unspentLocked() || !o.activeLocked(o.meter) {
		t.Fatal("unused original initial checkpoint role required")
	}
	// The physical original remains intact. Only missing producer-side head
	// DATA is tested, and this consumed bridge is never resumed or published.
	s.frontierFiles.pinRaw = ""
	if out := r.prepareInitialCheckpointLocked(o, scope, admissionTestDigest("a")); out != nil ||
		r.initial != nil || r.confirmed != nil || r.pending != nil || o.last != nil || o.coordinator != nil ||
		!reflect.DeepEqual(before, providerHistoryProvisionedOriginalBytes(t, s.config)) {
		t.Fatal("missing private original head supplied an initial checkpoint")
	}
}

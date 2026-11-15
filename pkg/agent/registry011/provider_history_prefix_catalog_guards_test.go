package registry011

import (
	"context"
	"reflect"
	"testing"
)

// An inert initial-stage selector only, with no store, native clock, retained
// original, valid program derivation or execution observer. A numeric answer
// from this private predicate must never stand in for production eligibility.
func historyPrefixScalarUnit() *providerHistoryGroupOwner {
	r := &providerHistoryRecoveryOwner{}
	o := &providerHistoryGroupOwner{parent: &providerHistoryProgramOwner{recovery: r}, start: &providerHistoryProgramStart{}}
	o.owner = o
	o.parent.bounds.stages = 2
	c := &providerHistoryRecoveryCheckpoint{recovery: r, group: o}
	c.owner = c
	r.initial, o.stageBefore = c, c
	return o
}

func TestProviderHistoryPrefixSelectorsRejectIncompletePrivateLinks(t *testing.T) {
	for _, mode := range []string{"same", "missing-owner", "initial-ack", "invalid-prior-stage", "missing-base", "zero-stage-cap"} {
		t.Run(mode, func(t *testing.T) {
			o := historyPrefixScalarUnit()
			if o.stageBaseLocked() != o.stageBefore || o.nextStageLocked() != 1 {
				t.Fatal("ordinary inert initial-stage DATA precursor")
			}
			switch mode {
			case "missing-owner":
				o.owner = nil
			case "initial-ack":
				o.stageBefore.ack = &providerHistoryGroupAcknowledgment{}
			case "invalid-prior-stage":
				c := &providerHistoryRecoveryCheckpoint{recovery: o.parent.recovery, group: o}
				c.owner, o.stageBefore = c, c
			case "missing-base":
				o.stageBefore = nil
			case "zero-stage-cap":
				o.parent.bounds.stages = 0 // Invalid scalar DATA, never a selected profile.
			}
			stage := o.nextStageLocked()
			var wantStage uint64
			if mode == "same" {
				wantStage = 1
			}
			if stage != wantStage || o.store != nil || o.parent.group != nil || o.meter != nil {
				t.Fatal("incomplete private stage links produced a next stage or an execution role", mode, stage)
			}
		})
	}
	var absent *providerHistoryGroupOwner
	if absent.stageBaseLocked() != nil || absent.healthyCheckpointShapeLocked(nil) || absent.selectedHistoryHeadLocked("") || absent.archivedMemberNameLocked(nil, 0, "response") != "" {
		t.Fatal("missing original owner selected a checkpoint, head or archived input")
	}
}

func TestProviderHistoryPrefixReadersRejectCancellationBeforeAnyOwnerUse(t *testing.T) {
	var absent *providerHistoryGroupOwner
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, c := range []context.Context{nil, ctx} {
		if out, err := absent.replayHealthyPrefixLocked(c, nil, ""); err == nil || out != nil {
			t.Fatal("cancelled original prefix reader produced DATA")
		}
		if absent.applyHealthyCheckpointLocked(c, nil, nil) == nil || absent.applyHealthyRecordLocked(c, nil, nil, nil, [8]providerHistoryStoredObject{}) == nil {
			t.Fatal("cancelled private checkpoint reader reached an owner or produced DATA")
		}
	}
}

// Closed original-shaped byte DATA only. These independent private ledgers
// have no real checkpoint selection, store roots, clock or dispatch role.
func historyPrefixRecordDataUnit(t *testing.T, phase string) (*providerHistoryGroupOwner, *providerHistoryHealthyPrefix, *providerHistoryRecoveryCheckpoint, *providerHistoryCustodyFrame, [8]providerHistoryStoredObject) {
	t.Helper()
	l := historyLedgerFixture(t, historyLedgerMaximum())
	ticket := historyRecordTicket(t, 1, 1, l.head)
	methods, capture := historyGroupBounds()
	pack, err := newProviderHistoryOriginalPackObserved(1, ticket.table.digest, "requests", 0, 1, []providerHistoryOriginalInput{{"request", ticket.table.members[0].request.wire}}, providerHistoryOriginalPackBytes, nil)
	if err != nil {
		t.Fatal("ordinary request-pack DATA precursor", err)
	}
	o := &providerHistoryGroupOwner{meter: historyGroupCodecMeter(), parent: &providerHistoryProgramOwner{}, store: &providerCustodyStore{}}
	o.parent.bounds.program, o.parent.bounds.costs = ticket.program, ticket.costs
	o.store.config.profile, o.store.config.capture = ticket.profile, capture
	o.store.config.bounds.methods, o.store.config.bounds.maximumObjectBytes = &methods, providerHistoryOriginalPackBytes
	d := &providerHistoryRecoveryAdmissionDescriptor{purpose: ticket.table.purpose, table: ticket.table.digest, members: 1, upper: ticket.upper,
		requests: providerHistoryStoredReference{kind: "requests", digest: pack.digest, bytes: uint64(len(pack.raw))},
		ticket:   providerHistoryStoredReference{kind: "ticket", digest: ticket.digest, bytes: uint64(len(ticket.raw))},
		clock:    CandidateClockDiagnostic{ClockID: "unit-diagnostic", Epoch: admissionTestDigest("a"), WallMS: 1000, MonotonicMS: 10}}
	c := &providerHistoryRecoveryCheckpoint{admission: d, inventory: providerHistoryInventory{count: 2}}
	f := &providerHistoryCustodyFrame{phase: "admit", stage: 1, operation: ticket.digest, clock: d.clock}
	p := &providerHistoryHealthyPrefix{ledger: l, inventory: ticket.inventory}
	objects := [8]providerHistoryStoredObject{{kind: "requests", raw: pack.raw, digest: pack.digest}, {kind: "ticket", raw: ticket.raw, digest: ticket.digest}}
	if phase == "consume" {
		if l.admit(ticket) != nil {
			t.Fatal("ordinary private admitted ledger precursor")
		}
		consumption, err := newProviderHistoryGroupConsumption(l.ticket, l.head, p.inventory)
		if err != nil {
			t.Fatal(err)
		}
		c.inventory.count, f.phase, f.operation = 1, "consume", consumption.digest
		objects = [8]providerHistoryStoredObject{{kind: "consumption", raw: consumption.raw, digest: consumption.digest}}
	}
	return o, p, c, f, objects
}

func TestProviderHistoryHealthyRecordRejectsWrongObjectAndOperationData(t *testing.T) {
	for _, mode := range []string{"admit", "consume", "admit-count", "admit-operation", "consume-count", "consume-operation", "unknown-phase"} {
		t.Run(mode, func(t *testing.T) {
			phase := "admit"
			if mode == "consume" || mode == "consume-count" || mode == "consume-operation" {
				phase = "consume"
			}
			o, p, c, f, objects := historyPrefixRecordDataUnit(t, phase)
			switch mode {
			case "admit-count":
				c.inventory.count = 1
			case "admit-operation", "consume-operation":
				f.operation = admissionTestDigest("f")
			case "consume-count":
				c.inventory.count = 2
			case "unknown-phase":
				f.phase = "unsupported-unit-phase"
			}
			err := o.applyHealthyRecordLocked(context.Background(), p, c, f, objects)
			positive := mode == "admit" || mode == "consume"
			if (err == nil) != positive || o.meter.failed || o.meter.actual[0] != 0 || o.meter.actual[1] != 0 || o.meter.actual[9] != 0 || o.meter.clocks != 0 ||
				o.owner != nil || o.store.owner != nil || o.parent.group != nil || o.last != nil {
				t.Fatal("inconsistent private original-shaped record accepted or gained a live role", mode, err)
			}
		})
	}
}

func TestProviderHistoryCatalogRejectsPrivateNameCollisionsAndWrongCopiedHead(t *testing.T) {
	o, original := historyAdmissionCatalogFixture(t)
	before, ok := o.prefixHistoryCatalogLocked(o.stageBefore)
	if !ok || !o.prepareAdmissionCatalogLocked(original) || !o.exactAdmissionCatalogLocked(original, false) {
		t.Fatal("genuine initial original catalog precursor")
	}
	pin, ack, held, clocks := o.pin, o.last, o.store.stored, o.store.clock.read
	for _, mode := range []string{"missing-root", "object-collision", "inventory-collision", "invalid-object"} {
		t.Run(mode, func(t *testing.T) {
			c, inventory := *original, *original.inventory
			c.owner, c.inventory = &c, &inventory
			names := [2]map[string]bool{{}, {}}
			for root := range names {
				for name, value := range before[root] {
					names[root][name] = value
				}
			}
			expected := replayTestClone(t, names)
			switch mode {
			case "missing-root":
				names[0] = nil
			case "object-collision":
				names[1][providerHistoryStoredName(c.inventory.entries[0], o.meter)] = true
			case "inventory-collision":
				names[1][providerHistoryInventoryName(c.inventory.digest, o.meter)] = true
			case "invalid-object":
				c.inventory.entries[0].kind = "unsupported-unit-kind"
			}
			supplied := replayTestClone(t, names)
			if mode == "invalid-object" {
				if o.prepareAdmissionCatalogLocked(&c) {
					t.Fatal("invalid private object acquired prepared names")
				}
			} else if after, ok := o.afterHistoryCatalogLocked(names, &c); ok || after[0] != nil || after[1] != nil {
				t.Fatal("incomplete or colliding private catalog published a name set", mode)
			}
			if !reflect.DeepEqual(names, supplied) || !reflect.DeepEqual(before, expected) || !o.activeLocked(o.meter) || o.meter.failed || o.pin != pin ||
				o.last != ack || o.store.stored != held || o.store.clock.read != clocks || o.store.frontierFiles.historyPending != nil || o.claimActiveLocked(&c) {
				t.Fatal("private catalog refusal changed original selection, inputs or acquired a write claim", mode)
			}
		})
	}
	// The actual original highwater is read and parsed first. Only a private
	// copy's expected profile differs; no retained file or limit is changed.
	c := *o.parent.recovery.initial
	raw, err := o.store.read(o.store.custody, o.store.name("highwater", c.pin.ordinal), int64(o.store.config.bounds.maximumMetadataPerCommit))
	h, parseErr := parseProviderFrontierHighwaterObserved(raw, o.store.config, o.meter)
	if err != nil || parseErr != nil || h != c.pin || !providerReplayEqual(raw, providerReplayBytes(c.head, o.meter), o.meter) {
		t.Fatal("genuine original highwater precursor", err, parseErr)
	}
	c.pin.profile = admissionTestDigest("e")
	if c.pin.profile == h.profile {
		t.Fatal("distinct private expected profile")
	}
	if o.applyHealthyCheckpointLocked(context.Background(), &providerHistoryHealthyPrefix{}, &c) == nil || !o.activeLocked(o.meter) || o.meter.failed ||
		o.pin != pin || o.last != ack || o.store.stored != held || o.store.clock.read != clocks || o.store.frontierFiles.historyPending != nil {
		t.Fatal("mismatched copied highwater was accepted or changed original execution")
	}
}

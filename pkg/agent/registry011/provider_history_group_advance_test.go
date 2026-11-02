package registry011

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
	"time"
)

func historyMultistageTestUpper(capture ProviderCaptureLimits) providerHistoryGroupUpper {
	u := providerHistoryGroupUpper{physical: 8 << 20, retainedRaw: 4 << 20, payload: 2 << 20, metadata: 2 << 20, objects: 128, files: 256, clocks: 256}
	u.execution[0], u.execution[1], u.execution[10] = 1, uint64(capture.WireBytes+1), 1<<48
	u.custody[2], u.custody[9], u.custody[10] = 256<<20, 32<<20, 1<<58
	return u
}

// These separate synthetic UNIT subjects are fixed before profile/anchor/H0.
// Every stage has one member, so even three complete global wire uppers plus
// startup fit the unchanged enclosing wire rule. Existing fixtures stay as-is.
// This exercises real original owners; it is NOT a production derivation.
func providerHistoryMultistageTestBounds(t *testing.T, b *providerIntegrationBounds, capture ProviderCaptureLimits, stages uint64) {
	t.Helper()
	if b.history == nil || stages < 2 || stages > 3 {
		t.Fatal("separate two/three-stage UNIT input required before H0")
	}
	h, u := b.history, historyMultistageTestUpper(capture)
	work, ok := u.work()
	initial, initOK := providerHistoryGroupInitializationUpper(*b)
	if !ok || !initOK {
		t.Fatal("checked original initialization and whole-stage uppers")
	}
	for i, value := range work {
		product, a := providerUpperProduct(value, stages)
		total, c := providerUpperSum(initial[i], product)
		if !a || !c {
			t.Fatal("whole-stage work sum overflow")
		}
		h.future.work[i] = total
	}
	for _, pair := range []struct {
		input  uint64
		output *uint64
	}{{u.physical, &h.future.physical}, {u.retainedRaw, &h.future.retainedRaw}, {u.payload, &h.future.payload},
		{u.metadata, &h.future.metadata}, {u.objects, &h.future.objects}, {u.files, &h.future.files}} {
		value, ok := providerUpperProduct(pair.input, stages)
		if !ok {
			t.Fatal("whole-stage retained domain overflow")
		}
		*pair.output = value
	}
	clocks, a := providerUpperProduct(u.clocks, stages)
	clocks, c := providerUpperSum(2, clocks)
	if !a || !c {
		t.Fatal("initialization native samples omitted or overflowed")
	}
	h.future.clocks, h.stages, h.records, h.commits = clocks, stages, 3*stages, 3*stages
	h.derivation.dispatch.advance = candidateWorkVector{2: 128 << 20, 10: 1 << 56}
	h.derivation.dispatch.advanceClocks, h.derivation.dispatch.advanceTailMS = 2, 299999
	h.derivation.snapshot = *h
	if !h.derivation.dispatch.advanceValid() || !b.valid() {
		t.Fatal("separate complete multistage UNIT envelope exceeds enclosing original limits")
	}
}

func historyMultistageOwner(t *testing.T, stages uint64) *providerHistoryGroupOwner {
	return historyMultistageOwnerWithRows(t, stages, nil)
}

func historyMultistageOwnerWithRows(t *testing.T, stages uint64, rows *providerHistoryDispatchBounds) *providerHistoryGroupOwner {
	return historyMultistageOwnerWithMemberEnvelope(t, stages, rows, 0)
}

func historyMultistageOwnerWithMemberEnvelope(t *testing.T, stages uint64, rows *providerHistoryDispatchBounds, members int) *providerHistoryGroupOwner {
	t.Helper()
	_, capture := providerFrontierBoundsFixture()
	// Race-instrumented original-prefix replays precede the owned transport.
	// Select this finite UNIT allowance before H0, within the existing lifetime.
	capture.Timeout = 2 * time.Minute
	s := providerHistoryProvisionedInspectionFixture(t, providerScanFixtureSelection{capture: &capture,
		pin: true, pinLargeObject: true, historyProgram: true, historyGroupWrite: true, historyGroupConsume: true,
		historyGroupDispatch: true, historyDispatchRows: rows, historyStages: stages, historyStageMembers: members, historyRecoveryNamespace: true, historyRecoveryLegacy: true, historyRecoveryGroup: true})
	first, err := newProviderFirstOperationCandidate(s)
	if err != nil {
		t.Fatal("original pre-held first operation", err)
	}
	conn := providerIntegrationOwnedTLS(t, providerFirstOperationRequestBody, providerFirstOperationResponse())
	if out, err := first.executeCandidateWithConnection(context.Background(), conn); err != nil || out == nil {
		t.Fatal("genuine first exchange", err)
	}
	o, err := newProviderHistoryGroupOwnerCandidate(s)
	if err != nil {
		t.Fatal("same original lifetime group", err)
	}
	return o
}

func historyMultistageDispatch(t *testing.T, o *providerHistoryGroupOwner, stage uint64) *providerHistoryGroupDispatch {
	return historyMultistageDispatchMembers(t, o, stage, 1)
}

func historyMultistageDispatchMembers(t *testing.T, o *providerHistoryGroupOwner, stage uint64, members int) *providerHistoryGroupDispatch {
	t.Helper()
	if members < 1 || members > 3 {
		t.Fatal("one to three members in the separate multistage UNIT envelope")
	}
	requests := make([]*providerIntegrationRequest, members)
	for i := range requests {
		method := "eth_chainId"
		if (stage+uint64(i))%2 == 0 {
			method = "web3_clientVersion"
		}
		requests[i] = historyGroupRequest(t, method, []any{})
	}
	upper := historyMultistageTestUpper(o.store.config.capture)
	wire, ok := providerUpperProduct(uint64(members), uint64(o.store.config.capture.WireBytes+1))
	if !ok {
		t.Fatal("checked exact member wire UNIT upper")
	}
	upper.execution[0], upper.execution[1] = uint64(members), wire
	ticket, err := o.retainAdmissionCandidate(context.Background(), "selection", requests, upper)
	if err != nil || ticket == nil || ticket.claim.frame.stage != stage || ticket.claim.coordinator.ticket.table.stage != stage {
		t.Fatal("actual consecutive stage T", stage, err)
	}
	ack, err := o.retainConsumptionCandidate(context.Background(), ticket)
	if err != nil || ack == nil || ack.claim.frame.stage != stage {
		t.Fatal("actual consecutive stage C", stage, err)
	}
	d, err := o.prepareDispatchCandidate(context.Background(), ack)
	if err != nil || d == nil || d.coordinator.ticket.table.stage != stage {
		t.Fatal("actual same-stage dispatch", stage, err)
	}
	return d
}

func historyMultistageSettle(t *testing.T, o *providerHistoryGroupOwner, stage uint64) *providerHistorySettledResults {
	t.Helper()
	d := historyMultistageDispatch(t, o, stage)
	historyRetainNormalPrefix(t, o, d, 1)
	out, err := o.retainSettlementCandidate(context.Background(), d)
	if err != nil || out == nil || out.ack.claim.frame.stage != stage {
		t.Fatal("actual consecutive successful S", stage, err)
	}
	return out
}

func TestProviderHistoryStageAdvancePreservesTheOriginalLifetimeAcrossTwoAndThreeGroups(t *testing.T) {
	for _, stages := range []uint64{2, 3} {
		t.Run(strconv.FormatUint(stages, 10), func(t *testing.T) {
			o := historyMultistageOwner(t, stages)
			s, r, m := o.store, o.parent.recovery, o.meter
			start, original, held, ceiling := o.start, o.parent.original, s.stored, o.parent.clockCeiling
			var old []*providerHistoryRecoveryMemberArchive
			for stage := uint64(1); stage <= stages; stage++ {
				out := historyMultistageSettle(t, o, stage)
				checkpoint, charged, retained := r.confirmed, o.coordinator.charged, o.retention
				old = append(old, checkpoint.memberArchive)
				if o.coordinator.stage != stage || o.coordinator.sequence != 3*stage || o.pin.sequence != start.main.sequence+3*stage {
					t.Fatal("stage-relative ledger and physical bridge offsets differ")
				}
				work, reads, pin, files := m.actual, s.clock.read, o.pin, providerHistoryProvisionedOriginalBytes(t, s.config)
				err := o.advanceStageCandidate(context.Background(), out.ack)
				if stage == stages {
					if err == nil || m.actual != work || s.clock.read != reads || o.phase != "settled" || o.pin != pin {
						t.Fatal("exhausted original stage maximum observed or renewed work")
					}
				} else if err != nil || o.phase != "prepared" || o.stageBefore != checkpoint || o.last != out.ack || r.confirmed != checkpoint ||
					o.order != nil || o.admission != nil || o.dispatch != nil || r.order != nil || r.consumption != nil || r.originalDispatch != nil ||
					r.completed != 0 || r.consumed != 0 || r.sent != 0 || !r.memberSelectionLocked(o) || !o.admissionBaseShapeLocked() ||
					s.clock.read != reads+2 || m.actual[0] != work[0] || m.actual[1] != work[1] || m.actual[2] <= work[2] || m.actual[10] <= work[10] ||
					o.coordinator.charged != charged || o.custodian.charged != charged || o.retention != retained || o.window || o.pin != pin {
					t.Fatal("transition renewed a lifetime or lost a confirmed original", stage, err)
				}
				if o.start != start || o.parent.original != original || o.parent.clockCeiling != ceiling || s.stored != held ||
					o.meter != m || m.failed || m.closed || s.activeMeter.Load() != nil || o.failed || s.failed {
					t.Fatal("whole original clock/hold/observer changed")
				}
				for _, archive := range old {
					if !archive.shapeLocked(r, archive.checkpoint) || archive.order.state != providerHistoryOrderClosed ||
						archive.order.consumed != 1 || archive.order.sent != 1 || archive.order.next != 1 || archive.count != 1 {
						t.Fatal("an old consumed/sent/completed original was reset")
					}
				}
				after := providerHistoryProvisionedOriginalBytes(t, s.config)
				if len(files) != len(after) {
					t.Fatal("read-only transition created files")
				}
				for name, raw := range files {
					if after[name] != raw {
						t.Fatal("read-only transition changed an original", name)
					}
				}
			}
		})
	}
}

func TestProviderHistoryStageAdvanceDetachedEntriesAreInertAndTheValidOriginalStillAdvances(t *testing.T) {
	o := historyMultistageOwner(t, 2)
	out := historyMultistageSettle(t, o, 1)
	for _, mode := range []string{"nil-context", "cancelled", "nil-ack", "copied-ack", "copied-owner", "authority", "changed-cap", "wrong-phase", "current-attempt", "copied-stage-base"} {
		t.Run(mode, func(t *testing.T) {
			ctx, owner, ack := context.Background(), o, out.ack
			phase, authority, upper, attempt, base := o.phase, o.meter.authority, o.meter.upper, o.dispatch.attempt, o.stageBefore
			switch mode {
			case "nil-context":
				ctx = nil
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			case "nil-ack":
				ack = nil
			case "copied-ack":
				copy := *ack
				ack = &copy
			case "copied-owner":
				copy := *o
				owner = &copy
			case "authority":
				o.meter.authority = false
			case "changed-cap":
				o.meter.upper[10]--
			case "wrong-phase":
				o.phase = "prepared"
			case "current-attempt":
				o.dispatch.attempt = &providerHistoryDispatchAttempt{}
			case "copied-stage-base":
				copy := *base
				o.stageBefore = &copy
			}
			work, reads, last, pin, retained := o.meter.actual, o.store.clock.read, o.last, o.pin, o.retention
			err := owner.advanceStageCandidate(ctx, ack)
			o.phase, o.meter.authority, o.meter.upper, o.dispatch.attempt, o.stageBefore = phase, authority, upper, attempt, base
			if err == nil || o.meter.actual != work || o.store.clock.read != reads || o.last != last || o.pin != pin || o.retention != retained ||
				o.failed || o.meter.closed || o.store.activeMeter.Load() != nil || !o.successfulSettlementShapeLocked() {
				t.Fatal("detached entry observed, poisoned or replaced the valid original", mode, err)
			}
		})
	}
	if err := o.advanceStageCandidate(context.Background(), out.ack); err != nil {
		t.Fatal("valid original positive control after inert entries", err)
	}
	work, reads := o.meter.actual, o.store.clock.read
	if o.advanceStageCandidate(context.Background(), out.ack) == nil || o.meter.actual != work || o.store.clock.read != reads {
		t.Fatal("the same completed transition ran twice")
	}
	if json.Unmarshal([]byte(`{}`), &providerHistoryHealthyPrefix{}) == nil {
		t.Fatal("serialized DATA manufactured a healthy reconstruction")
	}
}

func TestProviderHistoryStageAdvanceKeepsArchivedMembersThroughANextTBeforeCFailure(t *testing.T) {
	for _, retainNextT := range []bool{false, true} {
		t.Run(strconv.FormatBool(retainNextT), func(t *testing.T) {
			o := historyMultistageOwner(t, 2)
			out := historyMultistageSettle(t, o, 1)
			archive := o.parent.recovery.confirmed.memberArchive
			if o.advanceStageCandidate(context.Background(), out.ack) != nil {
				t.Fatal("genuine next-stage transition")
			}
			historyCheckStageScratch(t, archive.checkpoint, true)
			if retainNextT {
				ack, err := o.retainAdmissionCandidate(context.Background(), "identity", []*providerIntegrationRequest{historyGroupRequest(t, "eth_chainId", []any{})}, historyMultistageTestUpper(o.store.config.capture))
				if err != nil || ack == nil || ack.claim.frame.stage != 2 {
					t.Fatal("genuine next T without a new C", err)
				}
			}
			r, want := historyRecoveryGroupStart(t, o)
			records := 3
			if retainNextT {
				records++
			}
			for pass := range 2 {
				historyRecoveryGroupInitial(t, r, pass, want)
				g, err := r.replayGroupModelCandidate(context.Background(), pass)
				if err != nil || g == nil || !g.selectedLocked(r) || g.confirmedModel != r.confirmed.model || g.retention != o.retention ||
					len(g.records) != records || !archive.shapeLocked(r, archive.checkpoint) || archive.order.consumed != 1 || archive.order.sent != 1 ||
					r.consumed != 0 || r.order != nil || r.completed != 0 || o.meter == r.meter || o.store.activeMeter.Load() != nil {
					t.Fatal("next-stage terminal DATA lost earlier genuine S or recreated a current debit", pass, err)
				}
			}
		})
	}
}

func TestProviderHistoryStageAdvanceEarlierOriginalChangesQuarantineWithoutRenewal(t *testing.T) {
	for _, kind := range []string{"requests", "response", "tls", "history-pending", "HEAD"} {
		t.Run(kind, func(t *testing.T) {
			o := historyMultistageOwner(t, 2)
			out := historyMultistageSettle(t, o, 1)
			s, r := o.store, o.parent.recovery
			checkpoint, ack, pin, held, retained := r.confirmed, o.last, o.pin, s.stored, o.retention
			name, root := "HEAD", s.config.custodyDirectory
			switch kind {
			case "requests":
				name, root = providerHistoryStoredName(checkpoint.admission.requests, nil), s.config.evidenceDirectory
			case "response":
				name, root = checkpoint.memberArchive.members[0].capture.intent.responseName, s.config.evidenceDirectory
			case "tls":
				name, root = checkpoint.memberArchive.members[0].capture.intent.tlsName, s.config.evidenceDirectory
			case "history-pending":
				name, root = s.name(kind, checkpoint.pin.sequence), s.config.evidenceDirectory
			}
			if os.WriteFile(filepath.Join(root, name), []byte("owned unit original mismatch"), 0600) != nil {
				t.Fatal("change only this test's owned original")
			}
			if o.advanceStageCandidate(context.Background(), out.ack) == nil || !o.failed || !s.failed || !o.meter.closed || !o.meter.failed ||
				o.last != ack || r.confirmed != checkpoint || o.pin != pin || s.stored != held || o.retention != retained ||
				checkpoint.memberArchive.order.consumed != 1 || checkpoint.memberArchive.order.sent != 1 || s.activeMeter.Load() != nil {
				t.Fatal("invalid original advanced, refunded or acknowledged a replacement")
			}
		})
	}
}

func TestProviderHistoryMultistageAdmissionRejectsAnExcessiveWholeUpperBeforeEffects(t *testing.T) {
	o := historyMultistageOwner(t, 2)
	for _, coordinate := range []int{1, 2, 9, 10, 11} {
		upper := historyMultistageTestUpper(o.store.config.capture)
		if coordinate == 11 {
			upper.clocks = o.parent.bounds.future.clocks + 1
		} else {
			upper.custody[coordinate] = o.parent.bounds.future.work[coordinate] + 1
			if coordinate == 1 {
				upper.custody[1], upper.execution[1] = 0, o.parent.bounds.future.work[1]+1
			}
		}
		actual, reads := o.meter.actual, o.store.clock.read
		if out, err := o.retainAdmissionCandidate(context.Background(), "selection", []*providerIntegrationRequest{historyGroupRequest(t, "eth_chainId", []any{})}, upper); err == nil || out != nil ||
			o.meter.actual != actual || o.store.clock.read != reads || o.phase != "prepared" || o.failed || o.window {
			t.Fatal("insufficient original whole upper was admitted or observed", coordinate, err)
		}
	}
	if historyMultistageSettle(t, o, 1) == nil {
		t.Fatal("valid complete upper positive control")
	}
}

// The earlier T's unspent 256-file upper must not fund this next T's own
// eight-file upper. T requires nine allocations. The original whole hold and
// cumulative charged upper still have ample space in this genuine precursor.
func TestProviderHistoryStageStorageCannotBorrowAnEarlierUnusedUpper(t *testing.T) {
	o := historyMultistageOwner(t, 2)
	out := historyMultistageSettle(t, o, 1)
	checkpoint, archive := o.parent.recovery.confirmed, o.parent.recovery.confirmed.memberArchive
	if o.advanceStageCandidate(context.Background(), out.ack) != nil {
		t.Fatal("genuine next-stage transition")
	}
	upper := historyMultistageTestUpper(o.store.config.capture)
	upper.objects, upper.files = 2, 8
	if !upper.valid(1) {
		t.Fatal("valid smaller next-stage upper")
	}
	if _, ok := o.coordinator.charged.add(upper, o.parent.bounds.future); !ok {
		t.Fatal("whole cumulative capacity must not explain this refusal")
	}
	old, held, retained := providerHistoryProvisionedOriginalBytes(t, o.store.config), o.store.stored, o.retention
	next, err := o.retainAdmissionCandidate(context.Background(), "identity", []*providerIntegrationRequest{historyGroupRequest(t, "eth_chainId", []any{})}, upper)
	if err == nil || next != nil || !o.failed || o.last != out.ack || o.parent.recovery.confirmed != checkpoint ||
		o.parent.recovery.pending != nil || o.retention != retained || o.store.stored != held ||
		o.stageRetentionStart != checkpoint.retention || archive.order.consumed != 1 || archive.order.sent != 1 {
		t.Fatal("later T borrowed unused physical/file capacity or changed an old ACK", err)
	}
	after := providerHistoryProvisionedOriginalBytes(t, o.store.config)
	if len(old) != len(after) {
		t.Fatal("over-budget T wrote an original before its own stage admission")
	}
	for name, raw := range old {
		if after[name] != raw {
			t.Fatal("earlier original changed", name)
		}
	}
}

func TestProviderHistoryStageStorageChecksEveryDomainWithoutRefundOrUnderflow(t *testing.T) {
	start := providerHistoryStorageCharge{101, 103, 107, 109, 113, 127}
	upper := providerHistoryGroupUpper{physical: 29, retainedRaw: 23, payload: 19, metadata: 17, objects: 13, files: 31}
	limits := [6]uint64{upper.physical, upper.retainedRaw, upper.payload, upper.metadata, upper.objects, upper.files}
	before := [6]uint64{start.physical, start.raw, start.payload, start.metadata, start.objects, start.files}
	names := [6]string{"physical", "raw", "payload", "metadata", "objects", "files"}
	for index, name := range names {
		t.Run(name, func(t *testing.T) {
			for _, delta := range []int{-1, 0, 1} {
				values := before
				values[index] += limits[index]
				if delta < 0 {
					values[index]--
				} else if delta > 0 {
					values[index]++
				}
				total := providerHistoryStorageCharge{values[0], values[1], values[2], values[3], values[4], values[5]}
				if total.withinStage(start, upper) != (delta <= 0) {
					t.Fatal("exact or one-over original stage limit", name, delta)
				}
			}
			values := before
			values[index]--
			total := providerHistoryStorageCharge{values[0], values[1], values[2], values[3], values[4], values[5]}
			if total.withinStage(start, upper) {
				t.Fatal("underflow refunded an earlier stage", name)
			}
		})
	}
}

// These are finite DATA faults of the pure scalar predicate, not native
// callback replacement or an end-sample attack reproduction. A genuine final
// advance below separately exercises the actual bracket and publication.
func TestProviderHistoryStageFinalScalarClosureRequiresOriginalRoleAndBounds(t *testing.T) {
	o := historyMultistageOwner(t, 2)
	out := historyMultistageSettle(t, o, 1)
	s, m := o.store, o.meter
	s.mu.Lock()
	phase, active, start, clockStart := o.phase, s.activeMeter.Load(), o.advanceStart, o.advanceClockStart
	o.phase, o.advanceStart, o.advanceClockStart = "advancing", m.actual, m.clocks
	s.activeMeter.Store(m)
	if !o.fitsWork(m, candidateWorkVector{}) {
		s.mu.Unlock()
		t.Fatal("valid original scalar control")
	}
	for _, mode := range []string{"authority", "active-meter", "config", "row", "clock-selection", "window", "window-upper", "advance-start", "stage-baseline"} {
		authority, config, rows, window, windowUpper, advanceStart, baseline := m.authority, s.config, o.parent.dispatchBounds, o.window, o.windowUpper, o.advanceStart, o.stageRetentionStart
		s.clock.mu.Lock()
		maximum := s.clock.maximum
		if mode == "clock-selection" {
			s.clock.maximum--
		}
		s.clock.mu.Unlock()
		switch mode {
		case "authority":
			m.authority = false
		case "active-meter":
			s.activeMeter.Store(nil)
		case "config":
			s.config.profile = "owned unit selection mismatch"
		case "row":
			o.parent.dispatchBounds.advance[10]--
		case "window":
			o.window = false
		case "window-upper":
			o.windowUpper[2] = 0
		case "advance-start":
			o.advanceStart[2] = m.actual[2] + 1
		case "stage-baseline":
			o.stageRetentionStart.files++
		}
		work, reads := m.actual, s.clock.read
		accepted := o.fitsWork(m, candidateWorkVector{})
		m.authority, s.config, o.parent.dispatchBounds, o.window, o.windowUpper, o.advanceStart, o.stageRetentionStart = authority, config, rows, window, windowUpper, advanceStart, baseline
		s.clock.mu.Lock()
		s.clock.maximum = maximum
		s.clock.mu.Unlock()
		s.activeMeter.Store(m)
		if accepted || m.actual != work || s.clock.read != reads || !o.fitsWork(m, candidateWorkVector{}) {
			s.mu.Unlock()
			t.Fatal("scalar selection fault was accepted or observed", mode)
		}
	}
	o.phase, o.advanceStart, o.advanceClockStart = phase, start, clockStart
	s.activeMeter.Store(active)
	s.mu.Unlock()
	work, reads := m.actual, s.clock.read
	if err := o.advanceStageCandidate(context.Background(), out.ack); err != nil || s.clock.read != reads+2 || o.phase != "prepared" ||
		m.actual[0] != work[0] || m.actual[1] != work[1] || o.meter != m || o.parent.clock != s.clock {
		t.Fatal("actual native bracket and final scalar publication positive control", err)
	}
}

func historyStageScratchClaims(c *providerHistoryRecoveryCheckpoint) [3]*providerHistoryWriteClaim {
	return [3]*providerHistoryWriteClaim{c.admission.checkpoint.claim, c.previous.claim, c.claim}
}

func historyCheckStageScratch(t *testing.T, c *providerHistoryRecoveryCheckpoint, retired bool) {
	t.Helper()
	for _, claim := range historyStageScratchClaims(c) {
		if retired {
			if claim.coordinator != nil || claim.fresh != ([8]providerHistoryStoredObject{}) ||
				claim.beforeCatalog[0] != nil || claim.beforeCatalog[1] != nil ||
				claim.afterCatalog[0] != nil || claim.afterCatalog[1] != nil {
				t.Fatal("completed old stage still retains writer scratch")
			}
		} else if claim.coordinator == nil || claim.fresh[0].raw == "" ||
			claim.beforeCatalog[0] == nil || claim.beforeCatalog[1] == nil ||
			claim.afterCatalog[0] == nil || claim.afterCatalog[1] == nil {
			t.Fatal("genuine current-stage writer scratch precursor missing")
		}
	}
	d := c.memberArchive.dispatch
	if retired {
		if d.coordinator != nil || d.custodian != nil {
			t.Fatal("completed old dispatch still retains full models")
		}
	} else if d.coordinator == nil || d.custodian == nil {
		t.Fatal("genuine current dispatch model precursor missing")
	}
}

func TestProviderHistoryStageAdvanceRetiresScratchWithoutMutatingOldModelsOrOriginals(t *testing.T) {
	o := historyMultistageOwner(t, 3)
	var retired []*providerHistoryRecoveryCheckpoint
	for stage := uint64(1); stage <= 3; stage++ {
		out := historyMultistageSettle(t, o, stage)
		checkpoint := o.parent.recovery.confirmed
		historyCheckStageScratch(t, checkpoint, false)
		d := checkpoint.memberArchive.dispatch
		claims := historyStageScratchClaims(checkpoint)
		oldModels := [5]*providerHistoryGroupLedger{claims[0].coordinator, claims[1].coordinator, claims[2].coordinator, d.coordinator, d.custodian}
		var values [5]providerHistoryGroupLedger
		for i, model := range oldModels {
			values[i] = *model
		}
		archive, order := *checkpoint.memberArchive, *d.order
		var checkpoints [3]providerHistoryRecoveryCheckpoint
		for i, claim := range claims {
			checkpoints[i] = *claim.checkpoint
		}
		originals := providerHistoryProvisionedOriginalBytes(t, o.store.config)
		oldWork, oldReads, oldHeld, oldRetention := o.meter.actual, o.store.clock.read, o.store.stored, o.retention
		err := o.advanceStageCandidate(context.Background(), out.ack)
		if stage == 3 {
			if err == nil || o.meter.actual != oldWork || o.store.clock.read != oldReads {
				t.Fatal("final stage advanced or observed after its maximum")
			}
			historyCheckStageScratch(t, checkpoint, false)
		} else {
			if err != nil || o.stageBefore != checkpoint || o.last != out.ack || !o.admissionBaseShapeLocked() {
				t.Fatal("genuine advance did not select original next stage", err)
			}
			historyCheckStageScratch(t, checkpoint, true)
			retired = append(retired, checkpoint)
			for _, old := range oldModels {
				if o.coordinator == old || o.custodian == old {
					t.Fatal("retirement selected an old cached ledger as a fresh replay")
				}
			}
		}
		// Deliberately retain these aliases in this UNIT to detect in-place
		// mutation. Pointer retirement is not measured garbage collection/RAM.
		for i, model := range oldModels {
			if *model != values[i] {
				t.Fatal("old ledger was changed instead of dropping scratch pointer", i)
			}
		}
		for i, claim := range claims {
			if *claim.checkpoint != checkpoints[i] || claim.frame.model != claim.checkpoint.model ||
				claim.checkpoint.claim != claim || claim.checkpoint.ack.claim != claim {
				t.Fatal("retirement changed original checkpoint or ACK provenance", i)
			}
		}
		if *checkpoint.memberArchive != archive || *d.order != order || !checkpoint.memberArchive.shapeLocked(o.parent.recovery, checkpoint) ||
			o.store.stored != oldHeld || o.retention != oldRetention {
			t.Fatal("retirement changed archive/consumed bits or refunded reservation")
		}
		for _, old := range retired {
			historyCheckStageScratch(t, old, true)
			if !old.memberArchive.shapeLocked(o.parent.recovery, old) {
				t.Fatal("later stage lost an earlier archive")
			}
		}
		after := providerHistoryProvisionedOriginalBytes(t, o.store.config)
		if len(after) != len(originals) {
			t.Fatal("retirement changed original file set")
		}
		for name, raw := range originals {
			if after[name] != raw {
				t.Fatal("retirement changed original file", name)
			}
		}
	}
}

func TestProviderHistoryStageScratchRemainsOnCancelledOrFailedAdvance(t *testing.T) {
	for _, mismatch := range []bool{false, true} {
		t.Run(strconv.FormatBool(mismatch), func(t *testing.T) {
			o := historyMultistageOwner(t, 2)
			out := historyMultistageSettle(t, o, 1)
			checkpoint := o.parent.recovery.confirmed
			historyCheckStageScratch(t, checkpoint, false)
			claims := historyStageScratchClaims(checkpoint)
			models := [3]*providerHistoryGroupLedger{claims[0].coordinator, claims[1].coordinator, claims[2].coordinator}
			ctx := context.Background()
			if mismatch {
				// Ordinary owned UNIT data mismatch only; no traffic or exploit.
				name := providerHistoryStoredName(checkpoint.admission.requests, nil)
				if os.WriteFile(filepath.Join(o.store.config.evidenceDirectory, name), []byte("unit original mismatch"), 0600) != nil {
					t.Fatal("owned UNIT input")
				}
			} else {
				cancelled, cancel := context.WithCancel(ctx)
				cancel()
				ctx = cancelled
			}
			if o.advanceStageCandidate(ctx, out.ack) == nil {
				t.Fatal("invalid advance accepted")
			}
			historyCheckStageScratch(t, checkpoint, false)
			for i, claim := range claims {
				if claim.coordinator != models[i] {
					t.Fatal("failed transition retired original scratch", i)
				}
			}
			if checkpoint != o.parent.recovery.confirmed || out.ack != o.last || checkpoint.memberArchive.order.sent != 1 ||
				checkpoint.memberArchive.order.consumed != 1 || checkpoint.memberArchive.order.next != 1 {
				t.Fatal("failed transition replaced checkpoint or rewound original bits")
			}
		})
	}
}

// This prices only fixed structural traffic, not native CPU, write barriers,
// allocator pages, GC timing or complete retained/backing memory.
func TestProviderHistoryStageRetirementFitsTheSingleExistingStructuralCharge(t *testing.T) {
	word := uint64(reflect.TypeFor[*providerHistoryWriteClaim]().Size())
	stringHeader := uint64(reflect.TypeFor[string]().Size())
	if word > 8 || stringHeader > 16 {
		t.Fatal("unsupported structural layout")
	}
	fresh := uint64(reflect.TypeFor[[8]providerHistoryStoredObject]().Size())
	maps := uint64(reflect.TypeFor[[2]map[string]bool]().Size())
	retired := 3*(fresh+2*maps+word) + 2*word
	old := 13*word + uint64(reflect.TypeFor[providerHistoryStorageCharge]().Size()) +
		uint64(reflect.TypeFor[[providerHistoryGroupMembers]*providerHistoryRecoveryMemberComplete]().Size()) +
		uint64(reflect.TypeFor[int]().Size()) + 2*uint64(reflect.TypeFor[uint32]().Size()) +
		2*uint64(reflect.TypeFor[bool]().Size()) + stringHeader
	// Capture claims/dispatch once. The fixed32-word local budget includes
	// the extra three checkpoint pointers and constant array/range indices.
	combined := 2*(retired+old+4*word) + 32*word
	if combined > providerHistoryRecoveryArchiveSlotBytes {
		t.Fatal("fixed tail outgrew its ONE original pre-end charge", combined)
	}
	if word == 8 && (old != 442 || retired != 1288 || combined != 3780) {
		t.Fatal("independent conservative64-bit structural literals", old, retired, combined)
	}
	if word == 4 && (old != 250 || retired != 644 || combined != 1948) {
		t.Fatal("independent conservative32-bit structural literals", old, retired, combined)
	}
}

func TestProviderHistoryCollectionAdvanceRetiresPointersWithoutChangingTheOldCursor(t *testing.T) {
	s := historyCollectionAdmissionStoreWithPack(t, true, nil, true)
	o, err := newProviderHistoryGroupOwnerCandidate(s)
	if err != nil || o == nil {
		t.Fatal("genuine complete-pack collection owner", err)
	}
	admission, err := o.retainCollectionAdmissionCandidate(context.Background())
	if err != nil || admission == nil {
		t.Fatal("genuine collection T", err)
	}
	consumed, err := o.retainConsumptionCandidate(context.Background(), admission)
	if err != nil || consumed == nil {
		t.Fatal("genuine collection C", err)
	}
	d, err := o.prepareDispatchCandidate(context.Background(), consumed)
	if err != nil || d == nil {
		t.Fatal("genuine collection dispatch", err)
	}
	if !d.wholeResultPackFitsLocked() {
		t.Fatal("complete prospective collection result pack must fit BEFORE the first member")
	}
	table := d.coordinator.ticket.table
	before := o.coordinator.collection
	results := historyCollectionBoundaryResults(t, before, []FinalizedHeader{before.selection.input.journal.Deployment}, &table)
	for ordinal, result := range results {
		response := string(replayTestJSON(t, map[string]any{"jsonrpc": "2.0", "id": 1, "result": json.RawMessage(result.result)}))
		conn := providerIntegrationOwnedTLS(t, table.members[ordinal].request.body, response)
		if out, err := o.retainMemberCandidateWithConnection(context.Background(), d, ordinal, conn); err != nil || out == nil {
			t.Fatal("ordinary recorded collection response through actual owned TLS", ordinal, err)
		}
	}
	out, err := o.retainSettlementCandidate(context.Background(), d)
	if err != nil || out == nil || o.coordinator.collection == nil || o.coordinator.collection.completed != 1 || o.coordinator.collection.phase != "content" {
		t.Fatal("actual first collection S", err)
	}
	checkpoint := o.parent.recovery.confirmed
	historyCheckStageScratch(t, checkpoint, false)
	oldModel, oldCursor := o.coordinator, o.coordinator.collection
	if out.ack.claim.coordinator != oldModel || len(oldCursor.spine) != 1 || oldCursor == before || oldCursor.semantic == before.semantic {
		t.Fatal("genuine live S model/cursor alias precursor")
	}
	oldValue, spine, semantic := *oldModel, &oldCursor.spine[0], oldCursor.semantic
	oldRoot := oldCursor.commitmentObserved(providerReplayUnitMeter())
	originals := providerHistoryProvisionedOriginalBytes(t, o.store.config)
	if err := o.advanceStageCandidate(context.Background(), out.ack); err != nil {
		t.Fatal("actual collection advance", err)
	}
	historyCheckStageScratch(t, checkpoint, true)
	if *oldModel != oldValue || oldModel.collection != oldCursor || oldCursor.semantic != semantic || &oldCursor.spine[0] != spine || oldRoot == "" ||
		oldCursor.commitmentObserved(providerReplayUnitMeter()) != oldRoot || o.coordinator.collection == oldCursor || o.custodian.collection == oldCursor ||
		o.coordinator.collection == o.custodian.collection || &o.coordinator.collection.spine[0] == spine || &o.custodian.collection.spine[0] == spine ||
		o.coordinator.collection.semantic == semantic || o.custodian.collection.semantic == semantic || o.coordinator.head != checkpoint.model {
		t.Fatal("retirement mutated an aliased cursor or trusted an old model")
	}
	after := providerHistoryProvisionedOriginalBytes(t, o.store.config)
	if len(after) != len(originals) {
		t.Fatal("collection advance changed original file set")
	}
	for name, raw := range originals {
		if after[name] != raw {
			t.Fatal("collection advance changed original", name)
		}
	}
	// The next actual T/C must reconstruct the earlier files AFTER retirement.
	next, err := o.retainCollectionAdmissionCandidate(context.Background())
	if err != nil || next == nil || next.claim.frame.stage != 2 || next.claim.purpose != "content" {
		t.Fatal("next original collection T after retirement", err)
	}
	ack, err := o.retainConsumptionCandidate(context.Background(), next)
	if err != nil || ack == nil || ack.claim.frame.stage != 2 || !checkpoint.memberArchive.shapeLocked(o.parent.recovery, checkpoint) {
		t.Fatal("next original collection C lost retired historical evidence", err)
	}
}

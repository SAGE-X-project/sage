package registry011

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
)

// Genuine original P and collection owners run against existing benign
// recorded responses over owned TLS. No external RPC or operating Source is
// selected. The separate numeric UNIT row precedes profile, anchor and H0.
func TestProviderHistoryFinalReaderConnectsCollectionAndTwoFreshOriginalModels(t *testing.T) {
	s, in, recorded := historyCollectionRecordedRunStoreWithFinal(t, true)
	p, f := s.history, s.history.final
	if f == nil || f.reader == p.recovery || f.reader.meter == p.recovery.meter ||
		f.reader.meter.historyFinal != f || f.reader.meter.historyRecovery != nil || f.used || f.closed ||
		f.reader.custodyRoot != p.recovery.custodyRoot || f.reader.evidenceRoot != p.recovery.evidenceRoot ||
		f.reader.unspentLocked() || f.reader.beginInspectionCandidate(context.Background(), nil) == nil {
		t.Fatal("final observer was not separate and preselected before H0")
	}
	held, origin, ceiling, clocks := s.stored, p.original, p.clockCeiling, s.clock.read
	recoveryWork := p.recovery.meter.actual
	out, err := p.runVerifiedCollectionWithConnectionsCandidate(context.Background(), historyCollectionRecordedRunConnections(t, in, recorded))
	if err != nil || out == nil {
		historyFinalFailureDiagnostics(t, f)
		t.Fatal("normal collection and independent successful readback", err)
	}
	r, g := f.reader, p.group
	if r.meter.actual[3] == 0 || r.meter.actual[4] == 0 || r.meter.actual[6] == 0 ||
		r.meter.actual[3] > f.bounds.read.work[3] || r.meter.actual[4] > f.bounds.read.work[4] || r.meter.actual[6] > f.bounds.read.work[6] ||
		f.bounds.read.work[0] != 0 || f.bounds.read.work[1] != 0 || f.bounds.read.work[7] != 0 || f.bounds.read.work[8] != 0 || f.bounds.read.work[9] != 0 {
		t.Fatal("final original block/transaction/log observations were suppressed, unfunded or given effects")
	}
	if out.owner != out || out.final != f || out.checkpoint != p.recovery.confirmed || out.catalog != r.catalogResult ||
		out.cursor == nil || out.cursor.phase != "done" || !out.cursor.semantic.atEnd() || out.cursor.completed != 4 ||
		out.cursor == g.coordinator.collection || out.cursor == g.custodian.collection ||
		out.cursor == r.groupModels[0].ledger.collection || out.cursor == r.groupModels[1].ledger.collection ||
		out.cursor.semantic == r.groupModels[0].ledger.collection.semantic || out.cursor.semantic == r.groupModels[1].ledger.collection.semantic ||
		r.groupModels[0] == r.groupModels[1] || r.groupModels[0].ledger == r.groupModels[1].ledger ||
		r.initialModels[0] == r.initialModels[1] || r.catalogBefore == r.catalogAfter || r.inputFiles == 0 || r.inputBytes == 0 {
		t.Fatal("result reused producer state or shared independently read models")
	}
	if !f.used || !f.closed || f.failed || !r.used || r.active || !r.closed || r.failed ||
		!r.meter.closed || r.meter.failed || r.meter.authority || r.meter.clocks != 2 || r.meter.historyClockClaims != 2 ||
		s.clock.read != clocks+g.meter.clocks+2 || s.stored != held || p.original != origin || p.clockCeiling != ceiling ||
		g.phase != "verified" || g.failed || !g.meter.closed || g.meter.failed || p.failed || s.failed ||
		s.activeMeter.Load() != nil || p.recovery.meter.actual != recoveryWork || !p.recovery.unspentLocked() ||
		r.catalogResult.selectedLocked(r) || out.work != r.meter.actual || out.begin != f.begin || out.end == (ClockSample{}) {
		t.Fatal("successful final verification changed an old role, hold or clock budget")
	}
	for i, n := range r.meter.actual {
		if i == 10 {
			n -= 2
		}
		if n > f.bounds.read.work[i] {
			t.Fatal("read work borrowed a reserved clock unit", i)
		}
	}
	work, execution, reads, pin := r.meter.actual, g.meter.actual, s.clock.read, g.pin
	originals := providerHistoryProvisionedOriginalBytes(t, s.config)
	if again, err := p.runVerifiedCollectionCandidate(context.Background()); err == nil || again != nil {
		t.Fatal("completed collection was entered twice")
	}
	if again, err := f.verifyCandidate(context.Background(), nil); err == nil || again != nil {
		t.Fatal("closed final reader was entered twice")
	}
	if p.recovery.beginInspectionCandidate(context.Background(), g) == nil || r.beginInspectionCandidate(context.Background(), g) == nil ||
		r.meter.actual != work || g.meter.actual != execution || s.clock.read != reads || g.pin != pin || g.failed ||
		!reflect.DeepEqual(originals, providerHistoryProvisionedOriginalBytes(t, s.config)) {
		t.Fatal("reentry revived a failure lane, sampled time or changed originals")
	}
}

// A distinct, insufficient UNIT selection precedes H0. The benign complete
// collection runs unchanged; its selected semantic read must be refused without
// borrowing execution work. This end-to-end assertion does not identify the
// exact failing primitive. The genuine-owner primitive test checks first-false
// block/transaction/frame/log debit separately, without another full collection.
func TestProviderHistoryFinalReaderRefusesUnfundedReadWithoutExecutionBorrow(t *testing.T) {
	s, in, recorded := historyCollectionRecordedRunStoreWithFinalSelection(t, true, 0, true)
	p, f := s.history, s.history.final
	held, origin, ceiling, clocks := s.stored, p.original, p.clockCeiling, s.clock.read
	recoveryWork := p.recovery.meter.actual
	if f.bounds.read.work[3] != 0 || f.bounds.read.work[4] == 0 || f.bounds.read.work[6] == 0 {
		t.Fatal("distinct pre-H0 insufficient structural selection")
	}
	out, err := p.runVerifiedCollectionWithConnectionsCandidate(context.Background(), historyCollectionRecordedRunConnections(t, in, recorded))
	r, g, m := f.reader, p.group, f.reader.meter
	if err == nil || out != nil || g == nil || !f.used || !f.closed || !f.failed ||
		!r.used || r.active || !r.closed || !r.failed || !m.closed || !m.failed || m.authority ||
		m.actual[2] == 0 || m.actual[10] <= 1 || m.actual[3] != 0 || m.actual[4] != 0 || m.actual[5] != 0 || m.actual[6] != 0 ||
		m.actual[0] != 0 || m.actual[1] != 0 || m.actual[7] != 0 || m.actual[8] != 0 || m.actual[9] != 0 ||
		m.clocks != 1 || m.historyClockClaims != 1 || r.initialModels[0] == nil || !r.initialModels[0].finalizedSelected ||
		r.initialModels[0].firstFinalized != in.journal.Deployment || r.groupModels[0] != nil || r.groupModels[1] != nil ||
		!g.failed || !g.meter.closed || !g.meter.failed || g.meter.actual != f.executionWork || g.meter.clocks != f.executionClocks ||
		!p.failed || !s.failed || !s.clock.historyRevoked || s.activeMeter.Load() != nil ||
		s.clock.read != clocks+f.executionClocks+1 || s.stored != held || p.original != origin || p.clockCeiling != ceiling ||
		p.recovery.meter.actual != recoveryWork || p.recovery.used {
		t.Fatal("selected unfunded semantic read was accepted, lost its original predecessor or borrowed another role", err)
	}
	work, execution, reads, pin := m.actual, g.meter.actual, s.clock.read, g.pin
	if again, err := f.verifyCandidate(context.Background(), nil); err == nil || again != nil ||
		m.actual != work || g.meter.actual != execution || s.clock.read != reads || s.stored != held || g.pin != pin ||
		s.activeMeter.Load() != nil || !f.failed || !g.failed {
		t.Fatal("failed structural read renewed a counter, original hold, clock or execution role")
	}
}

func TestProviderHistoryFinalReaderRequiresSelectionAndCompleteOriginalCollection(t *testing.T) {
	var p *providerHistoryProgramOwner
	var f *providerHistoryFinalOwner
	var r *providerHistoryRecoveryOwner
	f.stopLocked(true)
	if f.pristineLocked() || f.executionShapeLocked() || f.provenanceMatchesLocked() || r.recordedOwnerLocked() != nil {
		t.Fatal("nil private selection gained original provenance")
	}
	if out, err := f.finishCandidate(context.Background(), nil); err == nil || out != nil {
		t.Fatal("nil final finisher")
	}
	if out, err := p.runVerifiedCollectionCandidate(context.Background()); err == nil || out != nil {
		t.Fatal("nil parent")
	}
	if out, err := f.verifyCandidate(context.Background(), nil); err == nil || out != nil ||
		newProviderHistoryFinalBeforeOpen(nil) != nil || f.bindOriginalRootsBeforeReadLocked() || f.activeLocked(nil) {
		t.Fatal("nil final owner")
	}
	if json.Unmarshal([]byte(`{}`), &providerHistoryFinalOwner{}) == nil || json.Unmarshal([]byte(`{}`), &providerHistoryFinalResult{}) == nil {
		t.Fatal("serialized DATA constructed a final owner or result")
	}
	// This normal first-P store has no final row. Refusal is before the first
	// collection group, file write, native sample or network request.
	s, _, _ := historyCollectionRecordedRunStore(t)
	p = s.history
	reads, held := s.clock.read, s.stored
	if out, err := p.runVerifiedCollectionCandidate(context.Background()); err == nil || out != nil ||
		p.final != nil || p.group != nil || s.clock.read != reads || s.stored != held || p.failed {
		t.Fatal("unselected final route started the collector")
	}
	selected, _, _ := historyCollectionRecordedRunStoreWithFinal(t, true)
	f = selected.history.final
	reads, held = selected.clock.read, selected.stored
	if f.bindOriginalRootsBeforeReadLocked() || newProviderHistoryFinalBeforeOpen(selected.history) != nil {
		t.Fatal("an opened store acquired another final root or observer")
	}
	for _, mode := range []string{"not-collected", "cancelled", "nil-context", "copied-owner"} {
		t.Run(mode, func(t *testing.T) {
			ctx := context.Background()
			subject := f
			switch mode {
			case "cancelled":
				cancelled, cancel := context.WithCancel(ctx)
				cancel()
				ctx = cancelled
			case "nil-context":
				ctx = nil
			case "copied-owner":
				copy := *f
				subject = &copy
			}
			if out, err := subject.verifyCandidate(ctx, nil); err == nil || out != nil ||
				f.used || f.closed || f.failed || f.reader.meter.actual != (candidateWorkVector{}) ||
				selected.history.group != nil || selected.clock.read != reads || selected.stored != held || selected.failed {
				t.Fatal("premature final entry touched the original collection")
			}
		})
	}
}

// One authentic completed collection is enough for all scalar domain checks.
// No malformed traffic, filesystem substitution or executable bypass is used.
func TestProviderHistoryFinalReaderCancellationQuarantinesOriginalWithoutRefund(t *testing.T) {
	s, in, recorded := historyCollectionRecordedRunStoreWithFinal(t, true)
	p, f := s.history, s.history.final
	settled, err := p.runCollectionWithConnectionsCandidate(context.Background(), historyCollectionRecordedRunConnections(t, in, recorded))
	if err != nil || settled == nil {
		t.Fatal("genuine completed collection", err)
	}
	g := p.group
	work, clocks, held, pin, consumed, sent := g.meter.actual, s.clock.read, s.stored, g.pin, g.order.consumed, g.order.sent
	if f.beginCandidate(context.Background(), settled) != nil {
		t.Fatal("original successful-final entry")
	}
	m := f.reader.meter
	if !f.activeLocked(m) || !f.provenanceMatchesLocked() || f.reader.recordedOwnerLocked() != p.recovery ||
		m.clocks != 1 || g.failed || !g.meter.closed || g.meter.failed || s.clock.read != clocks+1 {
		t.Fatal("final C0 reused failed execution or a replacement clock")
	}
	historyFinalObservationRouting(t, f)
	for _, kind := range []providerWorkEvent{providerWriteEvent, providerSyncEvent, providerRenameEvent, providerTLSEvent} {
		if providerHistoryRecoveryEventAllowed(kind) {
			t.Fatal("read-only event domain")
		}
	}
	// Directly inspect the finite read-window predicate with inert DATA. The
	// real meter/owner and its actual prefix remain unchanged by these checks.
	probe := &providerWorkMeter{actual: f.bounds.read.work, clocks: 1}
	probe.actual[10]++
	if !f.fitsReadWindowLocked(probe, candidateWorkVector{}) || f.fitsReadWindowLocked(probe, candidateWorkVector{10: 1}) ||
		f.fitsReadWindowLocked(probe, candidateWorkVector{0: 1}) || f.fitsReadWindowLocked(&providerWorkMeter{actual: candidateWorkVector{10: ^uint64(0)}}, candidateWorkVector{10: 1}) ||
		f.fitsReadWindowLocked(&providerWorkMeter{clocks: 1}, candidateWorkVector{}) {
		t.Fatal("read envelope borrowed clocks or accepted overflow")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if out, err := f.reader.reconstructOriginalsCandidate(ctx); err == nil || out != nil {
		t.Fatal("cancelled final reconstruction produced DATA")
	}
	// This is the same final caller cleanup, under the original store lock.
	s.mu.Lock()
	f.stopLocked(true)
	s.mu.Unlock()
	if !f.used || !f.closed || !f.failed || f.reader.active || !f.reader.closed || !m.closed || !m.failed ||
		!g.failed || !g.meter.failed || !g.meter.closed || !p.failed || !s.failed || !s.clock.historyRevoked ||
		s.clock.read != clocks+1 || g.meter.actual != work || s.stored != held || g.pin != pin ||
		g.order.consumed != consumed || g.order.sent != sent || s.activeMeter.Load() != nil {
		t.Fatal("final failure refunded the original prefix or renewed a role")
	}
	before := m.actual
	if out, err := f.verifyCandidate(context.Background(), settled); err == nil || out != nil || m.actual != before || s.clock.read != clocks+1 {
		t.Fatal("failed final owner was retried")
	}
	if f.reader.beginInspectionCandidate(context.Background(), g) == nil || f.reader.unspentLocked() {
		t.Fatal("successful-final reader entered the failed-inspection lane")
	}
	// A closed observer must refuse before any legacy unobserved fallback.
	if raw := providerReplayPhaseBytes("ordinary", m); raw != nil ||
		providerReplayPhaseRawTextEqual([]byte("ordinary"), "ordinary", m) ||
		providerCustodySetupDiagnosticObserved(s.config, providerClockDiagnostic(f.begin), m) != nil ||
		m.actual != before {
		t.Fatal("closed final observer silently used an unobserved legacy codec")
	}
}

func historyFinalObservationRouting(t *testing.T, f *providerHistoryFinalOwner) {
	t.Helper()
	m, s := f.reader.meter, f.parent.store
	before := m.actual
	if !providerFirstOperationObserved(m) || string(providerReplayPhaseBytes("plain", m)) != "plain" ||
		m.actual[10]-before[10] != 6 || m.actual[2] != before[2] {
		t.Fatal("final phase byte copy bypassed its concrete observer")
	}
	before = m.actual
	if !providerReplayPhaseRawTextEqual([]byte("plain"), "plain", m) || m.actual[10]-before[10] != 17 {
		t.Fatal("final raw/text comparison omitted its copy or comparison")
	}
	// The setup wire bytes stay identical to the legacy DATA encoding. The
	// final role additionally pays for the exact joined-original byte copy.
	legacy := providerReplayUnitMeter()
	want := providerCustodySetupDiagnosticObserved(s.config, providerClockDiagnostic(f.begin), legacy)
	before = m.actual
	got := providerCustodySetupDiagnosticObserved(s.config, providerClockDiagnostic(f.begin), m)
	id := s.config.identity
	copyBytes := uint64(len(s.config.anchor) + len(id.Instance) + len(id.Policy) + len(id.SourceID))
	if len(got) == 0 || string(got) != string(want) || m.actual[10]-before[10] != legacy.actual[10]+copyBytes+1 {
		t.Fatal("setup selected legacy unobserved copy behavior for the final role")
	}
	entries := []providerIntegrationObjectRef{{kind: "ticket", digest: admissionTestDigest("b")}, {kind: "exchange", digest: admissionTestDigest("a")}}
	before = m.actual
	if !providerInventoryOrderObserved(entries, m) || entries[0].kind != "exchange" || m.actual[10] <= before[10] || m.failed {
		t.Fatal("final inventory ordering lost bounded observed comparisons")
	}
}

// Report only scalar progress and admitted counters after the caller returns.
// This diagnostic reads no original payloads, invokes no owner predicates and
// changes no meter, failure flag, cancellation decision or test expectation.
func historyFinalFailureDiagnostics(t *testing.T, f *providerHistoryFinalOwner) {
	t.Helper()
	if f == nil || f.reader == nil || f.reader.meter == nil {
		t.Log("final reader diagnostic: no selected reader")
		return
	}
	r, m := f.reader, f.reader.meter
	t.Logf("final reader progress: inputFiles=%d/%d inputBytes=%d/%d clocks=%d claims=%d actual=%v upper=%v catalogBefore=%t catalogAfter=%t catalogResult=%t", r.inputFiles, r.bounds.maximumFiles, r.inputBytes, r.bounds.maximumBytes, m.clocks, m.historyClockClaims, m.actual, m.upper, r.catalogBefore != nil, r.catalogAfter != nil, r.catalogResult != nil)
	for pass := range 2 {
		t.Logf("final reader pass %d: inventory=%t scan=%t prefix=%t initial=%t group=%t completed=%t tail=%t", pass, r.inventoryWalks[pass] != nil, r.scanWalks[pass] != nil, r.legacyPrefixes[pass] != nil, r.initialModels[pass] != nil, r.groupModels[pass] != nil, r.completedMemberModels[pass] != nil, r.tailModels[pass] != nil)
	}
}

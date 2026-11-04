package registry011

import (
	"context"
	"encoding/json"
	"testing"
)

func TestProviderHistoryRecoveryBoundsSeparateReadOnlyResources(t *testing.T) {
	s := providerHistoryParentScalarFixture(t)
	b, catalog := s.config.bounds.history.recovery, s.config.bounds.history.catalogEntries
	if !b.valid(s.config.bounds, catalog) || len(b.object()) != 3 {
		t.Fatal("finite separate recovery row precursor")
	}
	for _, mode := range []string{"decode", "units", "files", "file-limit", "bytes", "byte-limit", "file-overflow", "byte-overflow"} {
		t.Run(mode, func(t *testing.T) {
			row, store, entries := b, s.config.bounds, catalog
			switch mode {
			case "decode":
				row.work[2] = 0
			case "units":
				row.work[10] = 0
			case "files":
				row.maximumFiles = 0
			case "file-limit":
				row.maximumFiles = entries*2 + 1
			case "bytes":
				row.maximumBytes = 0
			case "byte-limit":
				row.maximumBytes = store.maximumStoredBytes*2 + 1
			case "file-overflow":
				entries = ^uint64(0)
			case "byte-overflow":
				store.maximumStoredBytes = ^uint64(0)
			}
			if row.valid(store, entries) {
				t.Fatal("incomplete or overflowing recovery row accepted", mode)
			}
		})
	}
	for _, i := range []int{0, 1, 7, 8, 9} {
		row := b
		row.work[i] = 1
		if row.valid(s.config.bounds, catalog) {
			t.Fatal("read-only lane accepted an effect coordinate", i)
		}
	}
	for _, coordinate := range []int{3, 4, 5, 6} {
		row := b
		row.work[coordinate] = 1
		if !row.valid(s.config.bounds, catalog) {
			t.Fatal("original historical semantic observation was confused with an effect", coordinate)
		}
	}
}

// Independent structural literals for two monotone original readbacks. This
// constructs manifest DATA only: no store, native clock, transport or owner of
// an operating program is opened. JSON additionally prices both its successful
// projection and one first-aborting projection; RLP has no projection counters.
func TestProviderHistoryReadSemanticReservationUsesOriginalSourceCounts(t *testing.T) {
	for _, tc := range []struct {
		name, encoding string
		effects        uint64
		want           candidateWorkVector
	}{
		{"rlp", "debug-raw-rlp-v1", 0, candidateWorkVector{3: 4, 4: 6144, 6: 24576}},
		{"json", EthereumJSONBlocks, 0, candidateWorkVector{3: 8, 4: 10240, 6: 40960}},
		{"rlp-with-effect", "debug-raw-rlp-v1", 1, candidateWorkVector{3: 4, 4: 6144, 5: 1024, 6: 40960}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			manifest := historyReadSemanticManifest(t, tc.encoding, tc.effects)
			before := *manifest
			got, ok := providerHistoryCollectionReadSemanticUpper(manifest)
			if !ok || got != tc.want || !providerHistoryReadWorkDomain(got) || *manifest != before {
				t.Fatal("two original readbacks lost their independent source-count literal or changed the selected input", got, ok)
			}
		})
	}
}

func TestProviderHistoryReadSemanticReservationRefusesUnboundManifest(t *testing.T) {
	if out, ok := providerHistoryCollectionReadSemanticUpper(nil); ok || out != (candidateWorkVector{}) {
		t.Fatal("nil manifest returned a structural reservation")
	}
	for _, mode := range []string{"copied", "input", "counts", "raw"} {
		t.Run(mode, func(t *testing.T) {
			manifest := historyReadSemanticManifest(t, "debug-raw-rlp-v1", 0)
			switch mode {
			case "copied":
				copy := *manifest
				manifest = &copy
			case "input":
				manifest.input.contentBlocks++
			case "counts":
				manifest.counts.contentWindows++
			case "raw":
				manifest.raw += " "
			}
			if out, ok := providerHistoryCollectionReadSemanticUpper(manifest); ok || out != (candidateWorkVector{}) {
				t.Fatal("unbound manifest returned complete or partial structural work", mode)
			}
		})
	}
}

func historyReadSemanticManifest(t *testing.T, encoding string, effects uint64) *providerHistoryCollectionManifest {
	t.Helper()
	in := historyCollectionInput(t)
	in.encoding, in.gas = encoding, SepoliaAmsterdamGasProfile
	in.maximumSelected, in.blocks, in.native = sepoliaDeploymentBlock, 1, 1
	in.effects, in.versions, in.presentStops = effects, 0, 0
	in.windowBlocks, in.contentBlocks = 1, 1
	in.journal.Replay.MaxTransactions, in.journal.Replay.MaxFrames = 1024, 256
	in.journal.Replay.MaxLogs, in.journal.Replay.MaxEvidenceBytes = 4096, 8<<20
	in.capture.BodyBytes = 240 << 10
	return historyCollectionManifest(t, in)
}

// These are direct shared-decoder primitive checks, not a full block replay or
// proof of an end-to-end final reader's failure location. Each case uses a fresh
// genuine failed-inspection owner whose deficient semantic row was selected
// before H0. No live allowance, role, counter or observer is substituted.
func TestProviderHistoryReadSemanticPrimitiveRefusalPreservesOriginalPrefix(t *testing.T) {
	for _, tc := range []struct {
		name       string
		coordinate int
		observe    func(*replayBudget) bool
	}{
		{"block", 3, (*replayBudget).blockObserved},
		{"transaction", 4, (*replayBudget).transactionObserved},
		{"frame", 5, (*replayBudget).frameObserved},
		{"log", 6, func(b *replayBudget) bool { return b.logsObserved(1) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, _ := historyMemberDispatchFixture(t, 1)
			s, r := g.store, g.parent.recovery
			s.mu.Lock()
			g.failLocked()
			s.mu.Unlock()
			if r.beginInspectionCandidate(context.Background(), g) != nil {
				t.Fatal("original failed-inspection entry")
			}
			s.mu.Lock()
			m := r.meter
			m.mu.Lock()
			eligible := m.owner == m && m.historyRecovery == r && m.hasOwner() && !m.failed && !m.closed &&
				r.bounds.work[tc.coordinate] == 0 && m.upper[tc.coordinate] == 0
			m.mu.Unlock()
			execution, reads, held, pin := g.meter.actual, s.clock.read, s.stored, g.pin
			if !eligible || !m.event(providerCopyEvent, 0) {
				s.mu.Unlock()
				t.Fatal("genuine active original owner before the targeted primitive")
			}
			before := m.actual
			budget := replayBudget{observer: m}
			if tc.observe(&budget) || !m.failed || m.actual != before || m.clocks != 0 ||
				tc.observe(&budget) || m.actual != before {
				s.mu.Unlock()
				t.Fatal("unfunded shared primitive suppressed its observation, partly debited or retried", tc.coordinate)
			}
			r.stopLocked(true)
			s.mu.Unlock()
			if !r.used || r.active || !r.closed || !r.failed || !m.closed || !m.failed ||
				g.meter.actual != execution || s.clock.read != reads || s.stored != held || g.pin != pin ||
				s.activeMeter.Load() != nil || r.beginInspectionCandidate(context.Background(), g) == nil || m.actual != before {
				t.Fatal("primitive refusal refunded the original prefix or restored an execution role")
			}
		})
	}
}

func TestProviderHistoryRecoveryIsHeldBeforeH0WithoutReducingGroupFunds(t *testing.T) {
	s := providerHistoryParentScalarFixture(t)
	b := s.config.bounds
	execution, ok := providerScanUpperSum(providerHistoryPreIUpper(b), b.history.startup, providerScanUpper{work: b.history.future.work, retained: b.history.future.physical, clocks: b.history.future.clocks})
	held, other := providerHistoryWholeHold(b)
	if !ok || !other || held.retained != execution.retained || held.clocks != execution.clocks {
		t.Fatal("separate read-only row changed storage or native clocks")
	}
	for i, n := range held.work {
		if n != execution.work[i]+b.history.recovery.work[i] {
			t.Fatal("recovery row was omitted, overlapped or spent from group execution", i)
		}
	}
	s.firstLimits.work[2] = execution.work[2]
	if p, err := newProviderHistoryProgramBeforeOpenCandidate(s); err == nil || p != nil || s.history != nil || s.clock.read != 0 || s.clock.history != nil {
		t.Fatal("unfunded recovery reached H0 or changed the original selection")
	}
	fresh := providerHistoryParentScalarFixture(t)
	p, err := newProviderHistoryProgramBeforeOpenCandidate(fresh)
	if err != nil || p.recovery == nil || !p.recovery.unspentLocked() || p.recovery.meter == fresh.activeMeter.Load() ||
		p.recovery.meter.upper != fresh.config.bounds.history.recovery.work || p.bounds.future != fresh.config.bounds.history.future ||
		fresh.clock.read != 0 || fresh.custody != nil || fresh.evidence != nil {
		t.Fatal("read-only reservation was installed after native observation or reused execution funds", err)
	}
}

func TestProviderHistoryRecoveryPrimitiveKindsExcludeEffects(t *testing.T) {
	for _, item := range []struct {
		kind    providerWorkEvent
		allowed bool
	}{
		{providerReadEvent, true}, {providerWriteEvent, false}, {providerCodecEvent, true}, {providerHashEvent, true},
		{providerCopyEvent, true}, {providerClockEvent, false}, {providerSyncEvent, false}, {providerTLSEvent, false},
		{providerEncodeEvent, true}, {providerRenameEvent, false}, {providerCompareEvent, true}, {providerWorkEvent(255), false},
	} {
		if providerHistoryRecoveryEventAllowed(item.kind) != item.allowed {
			t.Fatal("read-only primitive classification", item.kind)
		}
	}
}

func TestProviderHistoryRecoveryUsesItsOneOriginalLaneWithoutRevivingExecution(t *testing.T) {
	o, _ := historyMemberDispatchFixture(t, 1)
	s, p, old := o.store, o.parent, o.meter
	r := p.recovery
	if r == nil || !r.unspentLocked() || r.meter == old || r.custody != s.custody || r.evidence != s.evidence || r.custodyRoot != s.custody.root || r.evidenceLock != s.evidence.lock || r.originalMeter != old || r.originalGroup != o {
		t.Fatal("genuine pre-H0 original recovery precursor")
	}
	s.mu.Lock()
	o.failLocked() // Finite terminal role test, not an uncertain disk recovery.
	s.mu.Unlock()
	actual, reads, held, pin, ack, next, sent := old.actual, s.clock.read, s.stored, o.pin, o.last, o.order.next, o.order.sent
	if err := r.beginInspectionCandidate(context.Background(), o); err != nil || !r.used || !r.active || r.closed || s.activeMeter.Load() != nil ||
		old.actual != actual || s.clock.read != reads || s.stored != held || !old.failed || !old.closed || !p.failed || !o.failed || !s.failed || !s.clock.historyRevoked {
		t.Fatal("inspection restored execution, renewed its time or changed its reservation", err)
	}
	// These are observer DATA operations, not actual retained-file replay.
	s.mu.Lock()
	for _, kind := range []providerWorkEvent{providerReadEvent, providerCodecEvent, providerHashEvent, providerCopyEvent, providerEncodeEvent, providerCompareEvent} {
		if !r.meter.event(kind, 4) {
			s.mu.Unlock()
			t.Fatal("funded original read-only observer refused DATA work")
		}
	}
	if r.meter.actual[2] != 4 || r.meter.actual[10] != 54 || r.meter.actual[0] != 0 || r.meter.actual[1] != 0 || r.meter.actual[9] != 0 || r.meter.clocks != 0 {
		s.mu.Unlock()
		t.Fatal("read-only actuals were confused with execution or native samples")
	}
	// Existing phase DATA adapters must not take their unobserved legacy
	// fallback merely because this read-only role never installs activeMeter.
	if !providerFirstOperationObserved(r.meter) || string(providerReplayPhaseBytes("abc", r.meter)) != "abc" || !providerReplayPhaseRawTextEqual([]byte("abc"), "abc", r.meter) ||
		r.meter.actual[10] != 69 || r.meter.actual[2] != 4 {
		s.mu.Unlock()
		t.Fatal("read-only phase copy or comparison fell through to legacy accounting")
	}
	before := r.meter.actual
	if r.meter.event(providerClockEvent, 0) || !r.meter.failed || r.meter.actual != before || r.meter.clocks != 0 || r.meter.event(providerReadEvent, 1) || r.meter.actual != before {
		s.mu.Unlock()
		t.Fatal("forbidden clock event or poisoned observer continued")
	}
	r.stopLocked(true)
	s.mu.Unlock()
	if r.active || !r.used || !r.closed || !r.failed || !r.meter.closed || r.beginInspectionCandidate(context.Background(), o) == nil ||
		old.actual != actual || s.clock.read != reads || s.stored != held || o.pin != pin || o.last != ack || o.order.next != next || o.order.sent != sent || s.activeMeter.Load() != nil {
		t.Fatal("spent lane was recreated, refunded or published a recovered ACK")
	}
	if result, err := r.meter.finish(nil, &CandidateAdmissionTicket{}, ClockSample{}); err == nil || result != nil {
		t.Fatal("read-only observer became an execution measurement")
	}
	if _, ok := (&providerMeasuredWork{meter: r.meter}).claim(nil, &CandidateAdmissionTicket{}, ClockSample{}); ok {
		t.Fatal("read-only observer supplied an execution claim")
	}
}

func TestProviderHistoryRecoveryEntryRejectsDetachedInputsBeforeObservation(t *testing.T) {
	o, _ := historyMemberDispatchFixture(t, 1)
	s, r := o.store, o.parent.recovery
	if r.beginInspectionCandidate(context.Background(), o) == nil || r.used {
		t.Fatal("active execution lent its unused recovery row")
	}
	s.mu.Lock()
	o.failLocked()
	s.mu.Unlock()
	for _, mode := range []string{"nil-context", "cancelled", "copied-owner", "nil-group", "copied-group", "upper", "reservation", "hybrid", "authority", "clocks", "closed-meter", "failed-meter", "selection", "policy", "methods", "tls", "scan", "continuation", "first", "limits", "open-execution", "clock-not-revoked", "clock-selection", "execution-meter", "custody-root", "evidence-lock", "closed-store", "clock-claim", "remaining", "ticket"} {
		t.Run(mode, func(t *testing.T) {
			input, group, ctx := r, o, context.Background()
			restore := func() {}
			switch mode {
			case "nil-context":
				ctx = nil
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			case "copied-owner":
				copy := *r
				input = &copy
			case "nil-group":
				group = nil
			case "copied-group":
				copy := *o
				group = &copy
			case "upper":
				r.meter.upper[2]--
				restore = func() { r.meter.upper[2]++ }
			case "reservation":
				r.meter.reservation[2]--
				restore = func() { r.meter.reservation[2]++ }
			case "hybrid":
				r.meter.integration = &providerIntegration{}
				restore = func() { r.meter.integration = nil }
			case "authority":
				r.meter.authority = true
				restore = func() { r.meter.authority = false }
			case "clocks":
				r.meter.maximumClocks = 1
				restore = func() { r.meter.maximumClocks = 0 }
			case "closed-meter":
				r.meter.closed = true
				restore = func() { r.meter.closed = false }
			case "failed-meter":
				r.meter.failed = true
				restore = func() { r.meter.failed = false }
			case "selection":
				s.config.profile = admissionTestDigest("f")
				restore = func() { s.config.profile = r.config.profile }
			case "policy":
				s.config.policy.raw += " "
				restore = func() { s.config.policy.raw = r.parent.policy.raw }
			case "methods":
				s.config.bounds.methods.scalar = providerMethodLimit{}
				restore = func() { *s.config.bounds.methods = r.parent.methods }
			case "tls":
				s.config.bounds.tls.chains++
				restore = func() { *s.config.bounds.tls = r.parent.tls }
			case "scan":
				s.config.bounds.scan.maximumChunks++
				restore = func() { *s.config.bounds.scan = r.parent.scan }
			case "continuation":
				s.config.bounds.continuation.maximumRecords++
				restore = func() { *s.config.bounds.continuation = r.parent.continuation }
			case "first":
				s.config.bounds.firstAdmission.durationMS++
				restore = func() { *s.config.bounds.firstAdmission = r.parent.first }
			case "limits":
				s.firstLimits.work[10]--
				restore = func() { *s.firstLimits = r.parent.limits }
			case "open-execution":
				o.meter.closed = false
				restore = func() { o.meter.closed = true }
			case "clock-not-revoked":
				s.clock.historyRevoked = false
				restore = func() { s.clock.historyRevoked = true }
			case "clock-claim":
				r.meter.historyClockClaims = 1
				restore = func() { r.meter.historyClockClaims = 0 }
			case "remaining":
				r.meter.remaining[2] = 1
				restore = func() { r.meter.remaining[2] = 0 }
			case "ticket":
				r.meter.ticket = admissionTestDigest("e")
				restore = func() { r.meter.ticket = "" }
			case "custody-root":
				original := s.custody.root
				s.custody.root = nil
				restore = func() { s.custody.root = original }
			case "evidence-lock":
				original := s.evidence.lock
				s.evidence.lock = nil
				restore = func() { s.evidence.lock = original }
			case "closed-store":
				s.closed = true
				restore = func() { s.closed = false }
			case "execution-meter":
				original := o.meter
				o.meter = &providerWorkMeter{}
				restore = func() { o.meter = original }
			case "clock-selection":
				s.clock.maximum--
				restore = func() { s.clock.maximum++ }
			}
			defer restore()
			actual, old, reads, held := r.meter.actual, o.meter.actual, s.clock.read, s.stored
			if input.beginInspectionCandidate(ctx, group) == nil || r.used || r.active || r.meter.actual != actual || o.meter.actual != old || s.clock.read != reads || s.stored != held || s.activeMeter.Load() != nil {
				t.Fatal("invalid entry observed, spent or revived an original role", mode)
			}
		})
	}
	for _, value := range []any{&providerHistoryRecoveryOwner{}, &providerHistoryRecoveryBounds{}} {
		if json.Unmarshal([]byte(`{}`), value) == nil {
			t.Fatal("JSON manufactured the pre-H0 recovery lane")
		}
	}
}

func TestProviderHistoryRecoveryStoreCloseRevokesTheUnusedOriginalLane(t *testing.T) {
	o, _ := historyMemberDispatchFixture(t, 1)
	s, r := o.store, o.parent.recovery
	if !r.unspentLocked() || s.close() != nil || !s.closed || !r.closed || !r.failed || !r.meter.closed || !r.meter.failed || r.used ||
		r.beginInspectionCandidate(context.Background(), o) == nil {
		t.Fatal("closed original roots left a reusable inspection lane")
	}
}

// Finite observer DATA only: no actual store, native clock, file write or
// claimed execution result. The healthy ordinary precursor isolates the two
// generic guards from the other reasons a poisoned/closed original refuses.
func historyRecoveryMeasurementDataFixture(t *testing.T) (*providerWorkMeter, *providerIntegration, *CandidateAdmissionTicket, ClockSample) {
	t.Helper()
	c := &providerIntegration{}
	c.owner = c
	work := candidateWorkVector{9: 1, 10: 2}
	m := newProviderWorkMeter(c, providerOperationReservation{work: work, execution: work})
	clock := ClockSample{ClockID: "finite-observer-DATA", Epoch: admissionTestDigest("a"), WallMS: 1000, MonotonicMS: 1000}
	ticket := &CandidateAdmissionTicket{digest: admissionTestDigest("b"), work: work, deadline: 2000, intent: &CandidateAdmissionIntent{clock: providerClockDiagnostic(clock)}}
	m.ticket = ticket.digest
	if !m.event(providerWriteEvent, 1) || m.failed || m.closed || m.actual != work {
		t.Fatal("ordinary finite observer DATA precursor")
	}
	return m, c, ticket, clock
}

func TestProviderHistoryRecoveryRejectsGenericMeasurementRoleInIsolation(t *testing.T) {
	ordinary, c, ticket, clock := historyRecoveryMeasurementDataFixture(t)
	positive, err := ordinary.finish(c, ticket, clock)
	if err != nil || positive == nil || !ordinary.closed || ordinary.failed {
		t.Fatal("otherwise-valid ordinary DATA finish precursor", err)
	}
	if _, ok := positive.claim(c, ticket, clock); !ok {
		t.Fatal("otherwise-valid ordinary DATA claim precursor")
	}
	for _, mode := range []string{"finish", "claim"} {
		t.Run(mode, func(t *testing.T) {
			m, owner, selected, end := historyRecoveryMeasurementDataFixture(t)
			actual := m.actual
			if mode == "finish" {
				m.historyRecovery = &providerHistoryRecoveryOwner{} // Inert hybrid DATA, never an original lane.
				if result, err := m.finish(owner, selected, end); err == nil || result != nil || m.closed || !m.failed || m.actual != actual || m.claimed {
					t.Fatal("recovery hybrid passed ordinary finish or partially finalized DATA")
				}
			} else {
				result, err := m.finish(owner, selected, end)
				if err != nil || result == nil || m.failed || !m.closed || m.claimed {
					t.Fatal("healthy closed ordinary DATA precursor", err)
				}
				m.historyRecovery = &providerHistoryRecoveryOwner{} // The sole changed generic-claim condition.
				if _, ok := result.claim(owner, selected, end); ok || m.claimed || m.failed || m.actual != actual {
					t.Fatal("recovery hybrid became an ordinary execution claim")
				}
			}
		})
	}
}

// Unused-lane shape checks are scalar-only and must observe every call. These
// pre-open UNIT selections never open roots or sample the native backend.
func TestProviderHistoryRecoveryUnusedSealsRejectUnexpectedState(t *testing.T) {
	for slot := range 2 {
		s := providerHistoryParentScalarFixture(t)
		p, err := newProviderHistoryProgramBeforeOpenCandidate(s)
		if err != nil || p == nil || p.recovery == nil || !p.matchesLocked() || !p.recovery.unspentLocked() {
			t.Fatal("original pre-open unused recovery selection", err)
		}
		r := p.recovery
		held, stored, original, execution, observation := p.held, s.stored, p.original, s.activeMeter.Load().actual, r.meter.actual
		// Even a nonnil all-zero seal is observed state, not an unused lane.
		r.tailSeals[slot] = &providerHistoryRecoveryTailData{}
		if r.unspentLocked() || p.matchesLocked() || p.failed || r.used || r.active || r.closed || r.failed || r.meter.closed || r.meter.failed ||
			p.held != held || s.stored != stored || p.original != original || s.clock.read != 0 || s.activeMeter.Load().actual != execution || r.meter.actual != observation {
			t.Fatal("unused seal identity was ignored or shape checking consumed original resources", slot)
		}
	}
}

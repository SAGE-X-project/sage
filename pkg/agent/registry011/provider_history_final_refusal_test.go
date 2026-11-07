package registry011

import (
	"context"
	"testing"
)

// An owned UNIT context cancels only after the real second sample. There is
// no replacement clock, injected native callback or production fault hook.
type historyFinalEndCancellation struct {
	context.Context
	cancel  context.CancelFunc
	final   *providerHistoryFinalOwner
	reached bool
}

func (c *historyFinalEndCancellation) Err() error {
	if c.final.reader.meter.clocks == 2 && c.final.reader.meter.historyClockClaims == 2 {
		c.reached = true
		c.cancel()
	}
	return c.Context.Err()
}

func TestProviderHistoryFinalVerificationAutomaticallyClosesEnteredReadAndLateCancellation(t *testing.T) {
	historyFinalCancellationCase(t, "scan")
}

func TestProviderHistoryFinalVerificationClosesAtCatalogEnd(t *testing.T) {
	historyFinalCancellationCase(t, "catalog-end")
}

func TestProviderHistoryFinalVerificationClosesAfterNativeEnd(t *testing.T) {
	historyFinalCancellationCase(t, "after-native-end")
}

// Each case still owns a fresh collection and caller. Separate roots allow CI
// to assign whole named scenarios without splitting or sharing their state.
func historyFinalCancellationCase(t *testing.T, phase string) {
	t.Helper()
	t.Run(phase, func(t *testing.T) {
		s, in, recorded := historyCollectionRecordedRunStoreWithFinal(t, true)
		p, f := s.history, s.history.final
		clocks, held, original := s.clock.read, s.stored, p.original
		base, cancel := context.WithCancel(context.Background())
		defer cancel()
		readCancel := &historyOriginalCancellationContext{Context: base, cancel: cancel, reader: f.reader, phase: phase}
		endCancel := &historyFinalEndCancellation{Context: base, cancel: cancel, final: f}
		var ctx context.Context = readCancel
		finalClocks := uint64(1)
		if phase == "after-native-end" {
			ctx, finalClocks = endCancel, 2
		}
		out, err := p.runVerifiedCollectionWithConnectionsCandidate(ctx, historyCollectionRecordedRunConnections(t, in, recorded))
		if err == nil || out != nil || phase == "after-native-end" && !endCancel.reached || phase != "after-native-end" && !readCancel.reached {
			historyFinalFailureDiagnostics(t, f)
			t.Fatal("real entered cancellation did not reach its selected prefix", phase, err)
		}
		r, g := f.reader, p.group
		if g == nil || r.inputFiles == 0 || r.inputBytes == 0 || r.catalogBefore == nil ||
			!f.used || !f.closed || !f.failed || r.active || !r.closed || !r.failed || !r.meter.closed || !r.meter.failed ||
			!g.failed || !g.meter.closed || !g.meter.failed || !p.failed || !s.failed || s.activeMeter.Load() != nil ||
			r.meter.clocks != finalClocks || s.clock.read != clocks+g.meter.clocks+finalClocks ||
			g.meter.actual != f.executionWork || g.meter.clocks != f.executionClocks || s.stored != held || p.original != original ||
			p.recovery.confirmed == nil || p.recovery.confirmed.frame.phase != "settle" || p.recovery.confirmed.failedS ||
			g.pin != p.recovery.confirmed.pin || g.last != p.recovery.confirmed.ack || g.order.consumed != (1<<7)-1 || g.order.sent != (1<<7)-1 {
			t.Fatal("automatic final cleanup changed the original S, debit or hold", phase)
		}
		work, files, bytes, reads := r.meter.actual, r.inputFiles, r.inputBytes, s.clock.read
		if again, err := f.verifyCandidate(context.Background(), nil); err == nil || again != nil ||
			r.meter.actual != work || r.inputFiles != files || r.inputBytes != bytes || s.clock.read != reads {
			t.Fatal("cancelled verification restarted an observer or native clock")
		}
	})
}

func TestProviderHistoryFinalReadCannotBorrowTheUnspentEndClockUnit(t *testing.T) {
	// Exactly two read units pay the existing entry-copy event. This finite
	// selection precedes profile/anchor/H0 and is never changed after entry.
	s, in, recorded := historyCollectionRecordedRunStoreWithFinalUnits(t, true, 2)
	p, f := s.history, s.history.final
	settled, err := p.runCollectionWithConnectionsCandidate(context.Background(), historyCollectionRecordedRunConnections(t, in, recorded))
	if err != nil || settled == nil || f.beginCandidate(context.Background(), settled) != nil {
		t.Fatal("genuine collection and separately funded original C0", err)
	}
	m, g := f.reader.meter, p.group
	before, execution, clocks, held := m.actual, g.meter.actual, s.clock.read, s.stored
	if before[10] != 3 || m.clocks != 1 || m.historyClockClaims != 1 || m.upper[10]-before[10] != 1 ||
		m.event(providerReadEvent, 0) || !m.failed || m.actual != before || m.clocks != 1 || s.clock.read != clocks {
		t.Fatal("read consumed the separately reserved final clock unit")
	}
	s.mu.Lock()
	f.stopLocked(true)
	s.mu.Unlock()
	if !f.failed || !f.closed || !g.failed || !p.failed || g.meter.actual != execution || s.clock.read != clocks || s.stored != held ||
		s.activeMeter.Load() != nil || f.bounds.read.work[10] != 2 || m.actual != before {
		t.Fatal("exhausted final read refunded its prefix or changed its admitted row")
	}
}

// Every negative gets a fresh real collection and two complete independent
// original reconstructions. Only private UNIT result DATA is varied; no
// hostile response, original-file edit or failed-owner reset is involved.
func TestProviderHistoryFinalFinishRejectsForeignStaleAndNonIndependentData(t *testing.T) {
	historyFinalFinishRefusalCase(t, "foreign-catalog")
}

func TestProviderHistoryFinalFinishRejectsStaleCatalog(t *testing.T) {
	historyFinalFinishRefusalCase(t, "stale-catalog")
}

func TestProviderHistoryFinalFinishRejectsMissingModel(t *testing.T) {
	historyFinalFinishRefusalCase(t, "missing-model")
}

func TestProviderHistoryFinalFinishRejectsSharedLedger(t *testing.T) {
	historyFinalFinishRefusalCase(t, "shared-ledger")
}

func TestProviderHistoryFinalFinishRejectsSharedCursor(t *testing.T) {
	historyFinalFinishRefusalCase(t, "shared-cursor")
}

func TestProviderHistoryFinalFinishRejectsSharedSemantic(t *testing.T) {
	historyFinalFinishRefusalCase(t, "shared-semantic")
}

func TestProviderHistoryFinalFinishRejectsUnfinishedCursor(t *testing.T) {
	historyFinalFinishRefusalCase(t, "unfinished")
}

func TestProviderHistoryFinalFinishRejectsWrongSelection(t *testing.T) {
	historyFinalFinishRefusalCase(t, "wrong-selection")
}

func TestProviderHistoryFinalFinishRejectsMismatchedCommitment(t *testing.T) {
	historyFinalFinishRefusalCase(t, "commitment-mismatch")
}

func historyFinalFinishRefusalCase(t *testing.T, mode string) {
	t.Helper()
	t.Run(mode, func(t *testing.T) {
		s, in, recorded := historyCollectionRecordedRunStoreWithFinal(t, true)
		p, f := s.history, s.history.final
		settled, err := p.runCollectionWithConnectionsCandidate(context.Background(), historyCollectionRecordedRunConnections(t, in, recorded))
		if err != nil || settled == nil || f.beginCandidate(context.Background(), settled) != nil {
			historyFinalFailureDiagnostics(t, f)
			t.Fatal("genuine final entry", err)
		}
		r, g := f.reader, p.group
		catalog, err := r.reconstructOriginalsCandidate(context.Background())
		if err != nil || catalog == nil || !catalog.selectedLocked(r) || !r.groupModels[0].selectedLocked(r) || !r.groupModels[1].selectedLocked(r) ||
			r.groupModels[0] == r.groupModels[1] || r.groupModels[0].ledger == r.groupModels[1].ledger {
			historyFinalFailureDiagnostics(t, f)
			t.Fatal("complete distinct original models before negative", err)
		}
		left, right := r.groupModels[0].ledger.collection, r.groupModels[1].ledger.collection
		if left == nil || right == nil || left == right || !left.valid() || !right.valid() ||
			left.phase != "done" || right.phase != "done" || !left.semantic.atEnd() || !right.semantic.atEnd() {
			t.Fatal("healthy final selected endpoints before negative")
		}
		before, clocks, execution, held, pin, ack := r.meter.actual, s.clock.read, g.meter.actual, s.stored, g.pin, g.last
		switch mode {
		case "foreign-catalog":
			copy := *catalog
			copy.owner, copy.recovery = &copy, p.recovery
			catalog = &copy
		case "stale-catalog":
			copy := *catalog
			copy.owner, copy.work[10] = &copy, copy.work[10]+1
			r.catalogResult, catalog = &copy, &copy
		case "missing-model":
			r.groupModels[1] = nil
		case "shared-ledger":
			r.groupModels[1].ledger = r.groupModels[0].ledger
		case "shared-cursor":
			r.groupModels[1].ledger.collection = left
		case "shared-semantic":
			right.semantic = left.semantic
		case "unfinished":
			left.phase = "content"
		case "wrong-selection":
			right.selection.manifest = admissionTestDigest("f")
		case "commitment-mismatch":
			right.client = "different bounded ordinary client"
			if !right.valid() || !right.semantic.atEnd() || right.selection != left.selection {
				t.Fatal("commitment negative must reach observed commitment comparison")
			}
		}
		out, err := f.finishCandidate(context.Background(), catalog)
		if err == nil || out != nil {
			t.Fatal("unselected final DATA escaped", mode)
		}
		// Honor the staged private caller contract exactly as verify does.
		// No role is restored or tried again after this entered failure.
		s.mu.Lock()
		healthy := !r.meter.failed && !r.meter.closed && f.activeLocked(r.meter) &&
			r.meter.clocks == 1 && r.meter.historyClockClaims == 1 && s.clock.read == clocks
		if !healthy {
			f.stopLocked(true)
			s.mu.Unlock()
			t.Fatal("DATA mismatch was masked by an observer or native-clock failure")
		}
		f.stopLocked(true)
		s.mu.Unlock()
		if !f.failed || !f.closed || r.active || !g.failed || !p.failed || !s.failed || s.activeMeter.Load() != nil ||
			s.clock.read != clocks || g.meter.actual != execution || s.stored != held || g.pin != pin || g.last != ack {
			t.Fatal("selected final rejection changed original execution or sampled C1")
		}
		if mode != "commitment-mismatch" && r.meter.actual != before ||
			mode == "commitment-mismatch" && r.meter.actual[10] <= before[10] {
			t.Fatal("negative reached an earlier guard or lost its measured comparison")
		}
	})
}

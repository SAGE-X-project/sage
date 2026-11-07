package registry011

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
)

func historyRecoveryModelPrefix(t *testing.T, r *providerHistoryRecoveryOwner, pass int, want map[string]providerIntegrationObjectRef) {
	t.Helper()
	historyRecoveryLegacySource(t, r, pass, want)
	if p, err := r.replayLegacyPrefixCandidate(context.Background(), pass); err != nil || p == nil || !p.selectedLocked(r) {
		t.Fatal("genuine actual inventory/scan/legacy prefix", err)
	}
}

func TestProviderHistoryRecoveryInitialModelsReconstructBudgetMessagesAndIndexesIndependently(t *testing.T) {
	o, r, want := historyRecoveryInventoryFixtureWithLegacy(t, true)
	s := o.store
	old, reads, held, pin, ack := o.meter.actual, s.clock.read, s.stored, o.pin, o.last
	before := providerHistoryProvisionedOriginalBytes(t, s.config)
	// Private producer caches are deliberately unavailable. Neither immutable
	// source pass may borrow a prior semantic model or its wire-binding lookup.
	s.frontierReplay, s.bindings = nil, nil
	for pass := range 2 {
		historyRecoveryModelPrefix(t, r, pass, want)
		files, bytes, work := r.inputFiles, r.inputBytes, r.meter.actual
		model, err := r.replayInitialModelCandidate(context.Background(), pass)
		if err != nil || model == nil || !model.selectedLocked(r) || r.inputFiles != files+2 || r.inputBytes != bytes || r.meter.actual[10] <= work[10] || !model.scanReadsVerified {
			t.Fatal("fresh full initial semantic model from actual originals", pass, err)
		}
		f, n := model.frontier, r.initialDescriptor
		ledger := f.body.ledger
		state, exists := ledger.tickets[n.ticket]
		if model.root.digest != n.enrollment || f.current.enrollment != n.enrollment || f.current.budget != n.budget || f.current.objects != n.objects ||
			f.body.intent.digest != n.admission || f.body.firstExchange != n.exchange || f.indexes.counts != [3]uint64{3, 4, 2} ||
			ledger.current.counts != [3]uint64{1, 1, 1} || ledger.current.totals.bootstrap.outstanding != (candidateWorkVector{}) ||
			ledger.current.privateMetadata == 0 || f.body.scanMetadata == 0 || !exists || state.consumption == nil || state.terminal != model.prefix.records[3].operation {
			t.Fatal("initial model omitted an actual budget, consumption, settlement, index or exchange relationship")
		}
		for _, used := range model.used {
			for _, consumed := range used {
				if !consumed {
					t.Fatal("a merely hashed object bypassed semantic reconstruction")
				}
			}
		}
		if pass == 1 {
			previous := r.initialModels[0]
			if previous == model || previous.frontier == f || previous.frontier.body == f.body || previous.frontier.body.ledger == ledger ||
				previous.frontier.indexes == f.indexes || previous.root == model.root || previous.frontier.body.policy == f.body.policy ||
				previous.frontier.current != f.current || previous.frontier.indexes.heads != f.indexes.heads ||
				previous.frontier.body.ledger.current.digest != ledger.current.digest || previous.frontier.body.ledger.current.privateMetadata != ledger.current.privateMetadata {
				t.Fatal("two independent semantic models share mutable state or disagree on the actual prefix")
			}
		}
	}
	if o.meter.actual != old || s.clock.read != reads || s.stored != held || o.pin != pin || o.last != ack || s.activeMeter.Load() != nil ||
		!o.failed || !o.meter.closed || !r.active || r.failed || r.closed || s.objects != nil || s.frontierReplay != nil || s.bindings != nil ||
		!reflect.DeepEqual(before, providerHistoryProvisionedOriginalBytes(t, s.config)) {
		t.Fatal("DATA semantic reconstruction changed originals, used producer caches or revived execution")
	}
	copy := *r.initialModels[0]
	copy.owner = &copy
	if copy.selectedLocked(r) || json.Unmarshal([]byte(`{}`), &providerHistoryRecoveryInitialModel{}) == nil {
		t.Fatal("copied or serialized DATA manufactured the selected semantic model")
	}
}

func TestProviderHistoryRecoveryInitialModelRefusesUnselectedOrRepeatedSourcesBeforeEffects(t *testing.T) {
	_, r, want := historyRecoveryInventoryFixtureWithLegacy(t, true)
	work := r.meter.actual
	if out, err := r.replayInitialModelCandidate(context.Background(), 0); err == nil || out != nil || r.meter.actual != work || !r.active {
		t.Fatal("semantic model selected without its actual legacy prefix", err)
	}
	historyRecoveryModelPrefix(t, r, 0, want)
	historyRecoveryModelPrefix(t, r, 1, want)
	for _, mode := range []string{"negative-pass", "excess-pass", "second-before-first", "nil-prefix", "copied-prefix", "incomplete-scan"} {
		t.Run(mode, func(t *testing.T) {
			original := r.legacyPrefixes[0]
			work, files, bytes := r.meter.actual, r.inputFiles, r.inputBytes
			pass := 0
			switch mode {
			case "negative-pass":
				pass = -1
			case "excess-pass":
				pass = 2
			case "second-before-first":
				pass = 1
			case "nil-prefix":
				r.legacyPrefixes[0] = nil
			case "copied-prefix":
				copy := *original
				copy.owner = original
				r.legacyPrefixes[0] = &copy
			case "incomplete-scan":
				original.scan.complete = false
			}
			out, err := r.replayInitialModelCandidate(context.Background(), pass)
			r.legacyPrefixes[0], original.scan.complete = original, true
			if err == nil || out != nil || r.meter.actual != work || r.inputFiles != files || r.inputBytes != bytes || !r.active || r.closed {
				t.Fatal("unselected semantic source spent the row or published a model", mode, err)
			}
		})
	}
	if out, err := r.replayInitialModelCandidate(context.Background(), 0); err != nil || out == nil || !out.selectedLocked(r) {
		t.Fatal("positive control after isolated source refusals", err)
	}
	work = r.meter.actual
	if out, err := r.replayInitialModelCandidate(context.Background(), 0); err == nil || out != nil || r.meter.actual != work || !r.active {
		t.Fatal("repeated semantic model reset or renewed its observer", err)
	}
}

// Pure intermediate DATA faults exercise later semantic guards after genuine
// prior transitions. They are neither modified network traffic nor a host or
// wire attack helper, and they never enter a live custodian's model.
func TestProviderHistoryRecoveryInitialSemanticGuardsRejectInconsistentFreshState(t *testing.T) {
	for _, mode := range []string{"retired-enrollment", "changed-deadline", "changed-prior-frontier", "wrong-index-head", "early-terminal", "missing-consumption", "preset-exchange", "setup-receipt-state", "main-receipt-index", "rewound-setup-ordinal"} {
		t.Run(mode, func(t *testing.T) {
			o, r, want := historyRecoveryInventoryFixtureWithLegacy(t, true)
			historyRecoveryModelPrefix(t, r, 0, want)
			old, reads, held, pin, ack := o.meter.actual, o.store.clock.read, o.store.stored, o.pin, o.last
			r.store.mu.Lock()
			defer r.store.mu.Unlock()
			f, err := newProviderFrontierReplayObserved(r.config, r.meter)
			if err != nil {
				t.Fatal("genuine fresh DATA replay", err)
			}
			f.meter, f.body.meter, f.indexes.meter = r.meter, r.meter, r.meter
			m := &providerHistoryRecoveryInitialModel{recovery: r, prefix: r.legacyPrefixes[0], frontier: f}
			m.owner = m
			for i, frame := range m.prefix.source.frames {
				m.used[i] = make([]bool, len(frame.objects))
			}
			through := 2
			if mode == "missing-consumption" || mode == "preset-exchange" {
				through = 3
			}
			if mode == "setup-receipt-state" || mode == "main-receipt-index" {
				through = 0
			}
			for i := 0; i < through; i++ {
				m.through = uint64(i + 1)
				if m.apply(m.prefix.records[i]) != nil {
					t.Fatal("genuine semantic precursor must succeed before the isolated fault", mode, i)
				}
			}
			switch mode {
			case "retired-enrollment":
				m.root.status = "retired"
			case "changed-deadline":
				f.current.deadline++
			case "changed-prior-frontier":
				f.current.previous = admissionTestDigest("e")
			case "wrong-index-head":
				f.indexes.heads[1] = admissionTestDigest("e")
			case "early-terminal":
				key := r.initialDescriptor.ticket
				state := f.body.ledger.tickets[key]
				state.terminal = admissionTestDigest("e")
				f.body.ledger.tickets[key] = state
			case "missing-consumption":
				key := r.initialDescriptor.ticket
				state := f.body.ledger.tickets[key]
				state.consumption = nil
				f.body.ledger.tickets[key] = state
			case "preset-exchange":
				f.body.firstExchange = admissionTestDigest("e")
			case "setup-receipt-state":
				m.prefix.scan.records[1].state = admissionTestDigest("e")
			case "main-receipt-index":
				m.prefix.scan.records[0].operations = admissionTestDigest("e")
			case "rewound-setup-ordinal":
				// Original setup0 belongs before E0. Rewinding an arithmetic
				// cache cannot select it anew at a later main boundary.
				f.setupOrdinal = -1
			}
			m.through = uint64(through + 1)
			if m.apply(m.prefix.records[through]) == nil || r.initialModels[0] != nil || o.meter.actual != old || o.store.clock.read != reads ||
				o.store.stored != held || o.pin != pin || o.last != ack || o.store.activeMeter.Load() != nil {
				t.Fatal("inconsistent intermediate DATA was accepted or renewed original execution", mode)
			}
		})
	}
}

func TestProviderHistoryRecoveryInitialModelCancellationStopsBeforeObservation(t *testing.T) {
	for _, mode := range []string{"nil", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			o, r, want := historyRecoveryInventoryFixtureWithLegacy(t, true)
			historyRecoveryModelPrefix(t, r, 0, want)
			var ctx context.Context
			if mode == "cancel" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(context.Background())
				cancel()
			}
			work, files, bytes, old, reads := r.meter.actual, r.inputFiles, r.inputBytes, o.meter.actual, o.store.clock.read
			out, err := r.replayInitialModelCandidate(ctx, 0)
			if err == nil || out != nil || r.initialModels[0] != nil || r.active || !r.closed || !r.failed || r.meter.actual != work ||
				r.inputFiles != files || r.inputBytes != bytes || o.meter.actual != old || o.store.clock.read != reads {
				t.Fatal("cancelled otherwise-valid semantic replay reached an effect or revived execution", err)
			}
		})
	}
}

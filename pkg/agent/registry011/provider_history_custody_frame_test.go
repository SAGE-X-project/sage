package registry011

import (
	"bytes"
	"strconv"
	"strings"
	"testing"
)

// The owned setup fixture completes its original first TLS exchange. The
// frame operations then use retained diagnostic DATA without another clock
// sample, an ACK, a history HEAD write or a group dispatch.
func providerHistoryFrameFixture(t *testing.T) (providerCustodyConfig, providerHistoryCustodyScope) {
	t.Helper()
	s := providerHistoryGroupStartFixture(t, true)
	return s.config, providerHistoryCustodyScope{initial: s.history.start.main, exchange: s.history.start.exchange, origin: providerClockDiagnostic(s.history.original)}
}

func providerHistoryFrameData(t *testing.T, config providerCustodyConfig, scope providerHistoryCustodyScope, before providerFrontierHighwater, phase string, offset int64) *providerHistoryCustodyFrame {
	t.Helper()
	clock := scope.origin
	clock.WallMS, clock.MonotonicMS = clock.WallMS+offset, clock.MonotonicMS+offset
	f, err := newProviderHistoryCustodyFrameObserved(config, scope, before, 1, phase,
		admissionTestDigest("a"), admissionTestDigest("b"), admissionTestDigest("c"), admissionTestDigest("d"), admissionTestDigest("e"), clock, nil)
	if err != nil {
		t.Fatal("closed frame data", err)
	}
	return f
}

func TestProviderHistoryCustodyFramesBindAcyclicOrderedPhysicalTransitions(t *testing.T) {
	config, scope := providerHistoryFrameFixture(t)
	before := scope.initial
	for index, phase := range []string{"admit", "consume", "settle"} {
		f := providerHistoryFrameData(t, config, scope, before, phase, int64(index+1))
		intent := contentJSON([]byte(f.intent)).(map[string]any)
		commit := contentJSON([]byte(f.commit)).(map[string]any)
		frontier := contentJSON([]byte(f.frontier)).(map[string]any)
		previous, err := before.bytes(config)
		if err != nil || f.after.previous != providerCandidateHash("sage-provider-custody-highwater-v1", previous) ||
			f.after.sequence != uint64(index+5) || f.after.ordinal != scope.initial.ordinal+uint64(index+1) ||
			f.after.identity != before.identity || f.after.setup != before.setup || f.after.fence != before.fence ||
			intent["sequence"] != strconv.Itoa(index+5) || intent["phase"] != phase || intent["previous_commit"] != before.commit ||
			intent["previous_inventory"] != f.previousInventory || intent["inventory"] != f.inventory ||
			len(commit) != 2 || commit["intent"] != providerCandidateHash(providerHistoryCustodyIntentDomain, []byte(f.intent)) ||
			f.after.commit != providerCandidateHash(providerHistoryCustodyCommitDomain, []byte(f.commit)) ||
			frontier["commit"] != f.after.commit || frontier["inventory"] != f.inventory || frontier["model"] != f.model ||
			f.after.frontier != providerCandidateHash(providerHistoryCustodyFrontierDomain, []byte(f.frontier)) {
			t.Fatal("frame lost its ordered acyclic physical relation", phase, err)
		}
		parsed, err := parseProviderHistoryCustodyFrameObserved(config, scope, before, f.after, []byte(f.intent), []byte(f.commit), []byte(f.frontier), nil)
		if err != nil || *parsed != *f {
			t.Fatal("original canonical frame data did not reconstruct", phase, err)
		}
		if checkProviderHighwaterMain(before, f.after, config, []byte(f.intent), []byte(f.commit), []byte(f.frontier)) {
			t.Fatal("new history domains entered the old single-request main parser")
		}
		before = f.after
	}
}

func TestProviderHistoryCustodyFrameRejectsChangedContextAndMalformedOriginals(t *testing.T) {
	config, scope := providerHistoryFrameFixture(t)
	f := providerHistoryFrameData(t, config, scope, scope.initial, "admit", 1)
	for _, mode := range []string{"empty", "oversize", "trailing", "unknown", "duplicate", "stage", "phase", "clock", "commit", "frontier", "after", "before", "scope", "origin", "exchange"} {
		t.Run(mode, func(t *testing.T) {
			selected, before, after := scope, f.before, f.after
			intent, commit, frontier := []byte(f.intent), []byte(f.commit), []byte(f.frontier)
			switch mode {
			case "empty":
				intent = nil
			case "oversize":
				frontier = make([]byte, providerHistoryCustodyFrameBytes+1)
			case "trailing":
				intent = append(intent, '\n')
			case "unknown":
				intent = append([]byte(`{"unknown":true,`), intent[1:]...)
			case "duplicate":
				intent = append([]byte(`{"phase":"admit",`), intent[1:]...)
			case "stage":
				intent = bytes.Replace(intent, []byte(`"stage":"1"`), []byte(`"stage":"0"`), 1)
			case "phase":
				intent = bytes.Replace(intent, []byte(`"phase":"admit"`), []byte(`"phase":"settle"`), 1)
			case "clock":
				object := contentJSON(intent).(map[string]any)
				object["clock"] = nil
				intent = providerIntegrationJSON(object)
			case "commit":
				commit = []byte(`{}`)
			case "frontier":
				frontier = []byte(`{}`)
			case "after":
				after.commit = admissionTestDigest("f")
			case "before":
				before.ordinal++
			case "scope":
				selected.initial.frontier = admissionTestDigest("f")
			case "origin":
				selected.origin.Epoch = admissionTestDigest("f")
			case "exchange":
				selected.exchange = admissionTestDigest("f")
			}
			if got, err := parseProviderHistoryCustodyFrameObserved(config, selected, before, after, intent, commit, frontier, nil); err == nil || got != nil {
				t.Fatal("changed originals or caller-selected context became a valid frame", mode)
			}
		})
	}
}

func TestProviderHistoryCustodyFrameRefusesUnboundedOrDetachedInputs(t *testing.T) {
	config, scope := providerHistoryFrameFixture(t)
	f := providerHistoryFrameData(t, config, scope, scope.initial, "admit", 1)
	for _, mode := range []string{"no-history", "invalid-history", "initial", "setup", "exchange", "empty-clock", "unicode-clock", "space-clock", "control-clock", "large-clock", "stage-zero", "stage-limit", "unknown-phase", "wrong-sequence", "first-commit", "first-frontier", "inventory", "model", "operation", "regression", "expired"} {
		t.Run(mode, func(t *testing.T) {
			c, selected, before := config, scope, scope.initial
			stage, phase, inventory, model, operation, clock := uint64(1), "admit", f.inventory, f.model, f.operation, f.clock
			switch mode {
			case "no-history":
				c.bounds.history = nil
			case "invalid-history":
				h := *c.bounds.history
				h.derivation = nil
				c.bounds.history = &h
			case "initial":
				selected.initial.sequence--
			case "setup":
				before.setup = admissionTestDigest("f")
			case "exchange":
				selected.exchange = ""
			case "empty-clock":
				selected.origin.ClockID, clock.ClockID = "", ""
			case "unicode-clock":
				selected.origin.ClockID, clock.ClockID = "단위", "단위"
			case "space-clock":
				selected.origin.ClockID, clock.ClockID = "unit clock", "unit clock"
			case "control-clock":
				selected.origin.ClockID, clock.ClockID = "unit\nclock", "unit\nclock"
			case "large-clock":
				selected.origin.ClockID = strings.Repeat("x", 257)
				clock.ClockID = selected.origin.ClockID
			case "stage-zero":
				stage = 0
			case "stage-limit":
				stage = c.bounds.history.stages + 1
			case "unknown-phase":
				phase = "unknown"
			case "wrong-sequence":
				before.sequence++
			case "first-commit":
				before.commit = admissionTestDigest("f")
			case "first-frontier":
				before.frontier = admissionTestDigest("f")
			case "inventory":
				inventory = f.previousInventory
			case "model":
				model = f.previousModel
			case "operation":
				operation = ""
			case "regression":
				clock.MonotonicMS = selected.origin.MonotonicMS - 1
			case "expired":
				clock.WallMS = selected.origin.WallMS + int64(c.bounds.history.durationMS) + 1
				clock.MonotonicMS = selected.origin.MonotonicMS + int64(c.bounds.history.durationMS) + 1
			}
			if got, err := newProviderHistoryCustodyFrameObserved(c, selected, before, stage, phase, f.previousInventory, inventory, f.previousModel, model, operation, clock, nil); err == nil || got != nil {
				t.Fatal("detached or unbounded frame inputs were accepted", mode)
			}
		})
	}
}

func TestProviderHistoryCustodyFrameObservationPreservesBytesAndRefusesDepletion(t *testing.T) {
	config, scope := providerHistoryFrameFixture(t)
	want := providerHistoryFrameData(t, config, scope, scope.initial, "admit", 1)
	// Existing arithmetic-only observers cannot enter the separately closed
	// live group owner or create a retained acknowledgement.
	m := providerReplayUnitMeter()
	got, err := newProviderHistoryCustodyFrameObserved(config, scope, scope.initial, 1, "admit", want.previousInventory, want.inventory, want.previousModel, want.model, want.operation, want.clock, m)
	if err != nil || *got != *want || m.failed || m.actual[10] == 0 || m.actual[2] != 0 || m.actual[0] != 0 || m.actual[9] != 0 {
		t.Fatal("constructor observation changed canonical data or fabricated effects", err)
	}
	reader := providerReplayUnitMeter()
	parsed, err := parseProviderHistoryCustodyFrameObserved(config, scope, scope.initial, want.after, []byte(want.intent), []byte(want.commit), []byte(want.frontier), reader)
	if err != nil || *parsed != *want || reader.failed || reader.actual[2] != uint64(len(want.intent)) || reader.actual[10] <= m.actual[10] || reader.actual[0] != 0 || reader.actual[9] != 0 {
		t.Fatal("parser omitted its original decode or changed the frame", err)
	}
	for _, mode := range []string{"constructor-work", "parser-work", "parser-decode"} {
		t.Run(mode, func(t *testing.T) {
			short := providerReplayUnitMeter()
			if mode == "parser-decode" {
				short.upper[2] = uint64(len(want.intent)) - 1
			} else {
				short.upper[10] = 0
			}
			var f *providerHistoryCustodyFrame
			var err error
			if mode == "constructor-work" {
				f, err = newProviderHistoryCustodyFrameObserved(config, scope, scope.initial, 1, "admit", want.previousInventory, want.inventory, want.previousModel, want.model, want.operation, want.clock, short)
			} else {
				f, err = parseProviderHistoryCustodyFrameObserved(config, scope, scope.initial, want.after, []byte(want.intent), []byte(want.commit), []byte(want.frontier), short)
			}
			if err == nil || f != nil || !short.failed || short.actual != (candidateWorkVector{}) {
				t.Fatal("depleted observation returned a frame or performed an unfunded primitive", mode)
			}
		})
	}
}

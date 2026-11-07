package registry011

import (
	"reflect"
	"strconv"
	"testing"
)

// Only inert cursor DATA is varied here. The quote reserves no live authority;
// the separately owned full original replays must still derive the whole T.
func TestProviderHistoryCollectionAdmissionCountMatchesIndependentGrammar(t *testing.T) {
	for _, tc := range []struct {
		name string
		want int
	}{
		{"open", 18}, {"ancestry-short", 4}, {"ancestry-maximum", 32}, {"ancestry-tail", 1},
		{"content-before-fork", 20}, {"content-after-fork", 30}, {"content-mixed", 25}, {"content-tail", 3},
		{"effects-maximum", 32}, {"effects-tail", 1}, {"state", 1}, {"grants", 1},
		{"history-maximum", 32}, {"history-tail", 1}, {"close", 7},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, b, capture := historyCollectionPlanFixture(t, 71, 64, 10)
			switch tc.name {
			case "ancestry-short", "ancestry-maximum", "ancestry-tail":
				c.phase = "ancestry"
				switch tc.name {
				case "ancestry-short":
					c.next = c.low + 3
				case "ancestry-tail":
					c.next = c.low
				}
			case "content-before-fork", "content-after-fork", "content-mixed", "content-tail":
				c.mode, c.phase = "content", "content"
				c.low, c.high, c.next = sepoliaDeploymentBlock, sepoliaDeploymentBlock+9, sepoliaDeploymentBlock
				c.spine = make([]FinalizedHeader, 71)
				for i := range c.spine {
					c.spine[i] = FinalizedHeader{c.selection.selected.Hash, c.selection.selected.ParentHash, c.selection.selected.Number - uint64(i), sepoliaAmsterdamTime - 1}
				}
				c.spine[0] = c.selection.selected
				for i := uint64(0); i < 10; i++ {
					if tc.name != "content-before-fork" && (tc.name != "content-mixed" || i < 5) {
						c.spine[c.selection.selected.Number-c.low-i].Timestamp = sepoliaAmsterdamTime
					}
				}
				if tc.name == "content-tail" {
					c.next = c.high
				}
			case "effects-maximum", "effects-tail":
				c.mode, c.phase = "content", "effects"
				c.high, c.low = c.selection.selected.Number, c.selection.selected.Number-3
				for i := range 33 {
					c.effects = append(c.effects, providerHistoryCollectionEffect{block: c.selection.selected.Hash, transaction: admissionTestDigest("a"), number: c.selection.selected.Number, index: uint64(i)})
				}
				if tc.name == "effects-tail" {
					c.effectNext, c.traces = 32, 32
				}
				if !c.effectFactsObserved(providerReplayUnitMeter()) {
					t.Fatal("positive effect DATA must match its known window and spine")
				}
			case "state", "grants":
				c.mode, c.phase = "content", tc.name
				c.high, c.low = c.selection.selected.Number, c.selection.selected.Number-3
				if stop, ok := c.spineHeader(c.high, providerReplayUnitMeter()); !ok || stop != c.selection.selected {
					t.Fatal("positive state/grants DATA requires the actual known stop header")
				}
			case "history-maximum", "history-tail":
				c.mode, c.phase, c.historyNext, c.stopVersion = "content", "history", 1, 33
				c.high, c.low = c.selection.selected.Number, c.selection.selected.Number-3
				if stop, ok := c.spineHeader(c.high, providerReplayUnitMeter()); !ok || stop != c.selection.selected {
					t.Fatal("positive history DATA requires the actual known stop header")
				}
				if tc.name == "history-tail" {
					c.historyNext = 33
				}
			case "close":
				c.phase = "close"
			}
			before := *c
			spine := append([]FinalizedHeader(nil), c.spine...)
			effects := append([]providerHistoryCollectionEffect(nil), c.effects...)
			got, ok := c.admissionCountLocked()
			if !ok || got != tc.want || !reflect.DeepEqual(before, *c) || !reflect.DeepEqual(spine, c.spine) || !reflect.DeepEqual(effects, c.effects) {
				t.Fatal("scalar count differs from handwritten grammar or mutated its cursor", got, ok)
			}
			if table := historyCollectionPlan(t, c, b, capture); table.count != tc.want {
				t.Fatal("scalar quote differs from the independent observed request builder")
			}
		})
	}
}

func TestProviderHistoryCollectionAdmissionCountRefusesIncompleteData(t *testing.T) {
	for _, mode := range []string{
		"nil", "owner", "selection", "zero-window", "large-window", "zero-content", "large-content", "reversed-window", "wide-window", "above-selected", "below-deployment", "mode", "done", "unknown",
		"ancestry-mode", "ancestry-low", "ancestry-high", "content-low", "content-high", "missing-spine", "wrong-spine-number",
		"effects-mode", "effects-negative", "effects-empty", "effects-past", "state-mode", "grants-mode", "history-mode", "history-zero", "history-past",
	} {
		t.Run(mode, func(t *testing.T) {
			c, _, _ := historyCollectionPlanFixture(t, 13, 4, 2)
			switch mode {
			case "nil":
				c = nil
			case "owner":
				c.owner = nil
			case "selection":
				c.selection.program = "changed DATA"
			case "zero-window":
				c.selection.input.windowBlocks, c.sealed.input.windowBlocks = 0, 0
			case "large-window":
				c.selection.input.windowBlocks, c.sealed.input.windowBlocks = 4097, 4097
			case "zero-content":
				c.selection.input.contentBlocks, c.sealed.input.contentBlocks = 0, 0
			case "large-content":
				c.selection.input.contentBlocks, c.sealed.input.contentBlocks = 11, 11
			case "reversed-window":
				c.low = c.high + 1
			case "wide-window":
				c.low = c.high - c.selection.input.windowBlocks
			case "above-selected":
				c.high, c.low = c.selection.selected.Number+1, c.selection.selected.Number+1
			case "below-deployment":
				c.low, c.high = sepoliaDeploymentBlock-1, sepoliaDeploymentBlock
			case "mode":
				c.mode = "unselected"
			case "done", "unknown":
				c.phase = mode
			case "ancestry-mode":
				c.phase, c.mode = "ancestry", "content"
			case "ancestry-low", "ancestry-high":
				c.phase = "ancestry"
				c.next = c.low - 1
				if mode == "ancestry-high" {
					c.next = c.high + 1
				}
			case "content-low", "content-high", "missing-spine", "wrong-spine-number":
				c.mode, c.phase, c.next = "content", "content", c.high
				switch mode {
				case "content-low":
					c.next = c.low - 1
				case "content-high":
					c.next = c.high + 1
				case "wrong-spine-number":
					c.spine = append(c.spine, c.selection.selected)
				}
			case "effects-mode", "effects-negative", "effects-empty", "effects-past":
				c.mode, c.phase = "content", "effects"
				switch mode {
				case "effects-mode":
					c.mode = "ancestry"
				case "effects-negative":
					c.effectNext = -1
				case "effects-past":
					c.effects = make([]providerHistoryCollectionEffect, 1)
					c.effectNext = 1
				}
			case "state-mode", "grants-mode":
				c.phase = "state"
				if mode == "grants-mode" {
					c.phase = "grants"
				}
			case "history-mode", "history-zero", "history-past":
				c.phase, c.mode, c.historyNext, c.stopVersion = "history", "content", 1, 33
				switch mode {
				case "history-mode":
					c.mode = "ancestry"
				case "history-zero":
					c.historyNext = 0
				case "history-past":
					c.historyNext = 34
				}
			}
			if got, ok := c.admissionCountLocked(); ok || got != 0 {
				t.Fatal("incomplete scalar DATA produced a reservable count", got, ok)
			}
		})
	}
}

func TestProviderHistoryCollectionAdmissionQuoteRefinesOnlyCardinalityAndWire(t *testing.T) {
	for _, count := range []int{1, 18, 32} {
		t.Run(strconv.Itoa(count), func(t *testing.T) {
			manifest := historyCollectionManifest(t, historyCollectionInput(t))
			capture := manifest.input.capture
			d := &providerHistoryProgramDerivation{}
			d.owner = d
			u := historyMultistageTestUpper(capture)
			u.execution[0], u.execution[1] = 32, 32*uint64(capture.WireBytes+1)
			b := providerHistoryCollectionAdmissionBounds{d, manifest, u}
			d.admission = b // Inert DATA only: d.snapshot is deliberately zero.
			got, ok := b.forMembers(d, manifest, capture, count)
			want := u
			want.execution[0], want.execution[1] = uint64(count), uint64(count)*uint64(capture.WireBytes+1)
			if !ok || got != want || d.admission != b || b.maximum != u || d.snapshot != (providerHistoryProgramBounds{}) {
				t.Fatal("quote changed another work/storage/clock coordinate or minted a profile")
			}
		})
	}
}

func TestProviderHistoryCollectionAdmissionQuoteRefusesUnboundPlans(t *testing.T) {
	for _, mode := range []string{"nil-derivation", "derivation-owner", "plan-owner", "snapshot", "nil-manifest", "manifest-copy", "manifest-owner", "capture", "invalid-capture", "zero-count", "large-count", "maximum-count", "wire", "work-overflow"} {
		t.Run(mode, func(t *testing.T) {
			manifest := historyCollectionManifest(t, historyCollectionInput(t))
			capture := manifest.input.capture
			d := &providerHistoryProgramDerivation{}
			d.owner = d
			u := historyMultistageTestUpper(capture)
			u.execution[0], u.execution[1] = 32, 32*uint64(capture.WireBytes+1)
			b := providerHistoryCollectionAdmissionBounds{d, manifest, u}
			count := 18
			switch mode {
			case "derivation-owner":
				d.owner = nil
			case "plan-owner":
				b.owner = nil
			case "nil-manifest":
				manifest = nil
			case "manifest-copy":
				copy := *manifest
				copy.owner = &copy
				manifest = &copy
			case "manifest-owner":
				manifest.owner = nil
			case "capture":
				capture.WireBytes--
			case "invalid-capture":
				capture.JSONDepth = 0
			case "zero-count":
				count = 0
			case "large-count":
				count = 33
			case "maximum-count":
				b.maximum.execution[0] = 31
			case "wire":
				b.maximum.execution[1]--
			case "work-overflow":
				b.maximum.execution[10] = ^uint64(0)
			}
			d.admission = b
			switch mode {
			case "snapshot":
				d.admission.maximum.clocks++
			case "nil-derivation":
				d = nil
			}
			if got, ok := b.forMembers(d, manifest, capture, count); ok || got != (providerHistoryGroupUpper{}) {
				t.Fatal("unbound plan gained an admission quote", mode)
			}
		})
	}
}

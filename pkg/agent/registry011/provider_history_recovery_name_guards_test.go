package registry011

import (
	"context"
	"os"
	"strings"
	"testing"
)

// Inert namespace DATA only. No store, original clock, selected checkpoint,
// dispatch or production derivation is installed by this private writer.
func historyRecoveryNameUnitWriter() *providerHistoryRecoveryExpectedCatalog {
	r := &providerHistoryRecoveryOwner{parent: &providerHistoryProgramOwner{}, meter: providerReplayUnitMeter()}
	r.parent.bounds.catalogEntries = 4
	r.parent.bounds.catalogNameBytes = 1024
	return &providerHistoryRecoveryExpectedCatalog{recovery: r, names: make([]providerHistoryRecoveryExpectedName, 0, 4)}
}

func TestProviderHistoryRecoveryNamesRespectNamespaceAndCumulativeCapacity(t *testing.T) {
	for _, mode := range []string{"same-namespace", "other-namespace", "empty", "long", "slots", "bytes"} {
		t.Run(mode, func(t *testing.T) {
			w := historyRecoveryNameUnitWriter()
			if !w.add(false, "kept", false) {
				t.Fatal("ordinary private name precursor")
			}
			name, evidence := "kept", false
			switch mode {
			case "other-namespace":
				evidence = true
			case "empty":
				name = ""
			case "long":
				name = strings.Repeat("x", 256)
			case "slots":
				w.names = w.names[:cap(w.names)]
			case "bytes":
				w.bytes = w.recovery.parent.bounds.catalogNameBytes
			}
			count, bytes := len(w.names), w.bytes
			got := w.add(evidence, name, true)
			if got != (mode == "other-namespace") || w.recovery.meter.failed || w.recovery.catalogResult != nil {
				t.Fatal("invalid or duplicate namespace name accepted", mode, got)
			}
			if !got && (len(w.names) != count || w.bytes != bytes) {
				t.Fatal("refused name consumed a catalog slot or name bytes", mode)
			}
			if got && (len(w.names) != count+1 || w.bytes != bytes+4 || !w.names[count].optional || !w.names[count].evidence) {
				t.Fatal("same text in distinct namespaces lost its role or optionality")
			}
		})
	}
}

func TestProviderHistoryRecoveryNamesObserveCopiesBeforePublishingEntries(t *testing.T) {
	// The four-byte clone costs five units, then the slot copy costs two.
	// No failed copy may publish the partially prepared name or renew fuel.
	for _, fuel := range []uint64{0, 5} {
		w := historyRecoveryNameUnitWriter()
		w.recovery.meter.upper[10] = fuel
		if w.add(false, "kept", false) || !w.recovery.meter.failed || len(w.names) != 0 || w.bytes != 0 || w.recovery.meter.actual[10] != fuel {
			t.Fatal("unfunded name copy published partial namespace DATA", fuel)
		}
		charged := w.recovery.meter.actual
		if w.add(false, "kept", false) || w.recovery.meter.actual != charged {
			t.Fatal("failed private name copy renewed its observer")
		}
	}
	w := historyRecoveryNameUnitWriter()
	// An already enumerated name is inert comparison DATA; the new UNIT
	// observer starts with no allowance. No successful observer is resized.
	w.names = append(w.names, providerHistoryRecoveryExpectedName{name: "kept"})
	w.bytes = 4
	w.recovery.meter.upper[10] = 0
	charged := w.recovery.meter.actual
	if w.add(false, "next", false) || !w.recovery.meter.failed || w.recovery.meter.actual != charged || len(w.names) != 1 || w.bytes != 4 {
		t.Fatal("unfunded duplicate-name search changed the established prefix")
	}
}

func TestProviderHistoryRecoveryOrdinalNamesRequireTheirOwnObservation(t *testing.T) {
	w := historyRecoveryNameUnitWriter()
	if got := w.recovery.catalogOrdinalNameLocked("highwater", 7); got != "highwater-00000000000000000007" {
		t.Fatal("closed ordinal namespace encoding changed", got)
	}
	r := historyRecoveryNameUnitWriter().recovery
	r.meter.upper[10] = 0
	if r.catalogOrdinalNameLocked("highwater", 7) != "" || !r.meter.failed || r.meter.actual != (candidateWorkVector{}) {
		t.Fatal("unobserved ordinal name escaped its private writer")
	}
}

func TestProviderHistoryRecoveryExpectedNamesRequireSelectedFiniteCatalogs(t *testing.T) {
	info, err := os.Stat(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"same", "missing-required", "optional-present", "wrong-namespace", "extra-entry", "unselected", "unfunded"} {
		t.Run(mode, func(t *testing.T) {
			w := historyRecoveryNameUnitWriter()
			w.names = append(w.names, providerHistoryRecoveryExpectedName{name: "kept"}, providerHistoryRecoveryExpectedName{evidence: true, optional: true, name: "maybe"})
			w.bytes = 9
			r := w.recovery
			c := &providerHistoryRecoveryCatalog{recovery: r, complete: true, roots: [2]os.FileInfo{info, info}, names: 4,
				entries: make([]providerHistoryRecoveryCatalogEntry, 1, 4)}
			c.owner, r.catalogBefore = c, c
			c.entries[0] = providerHistoryRecoveryCatalogEntry{name: "kept", info: info}
			switch mode {
			case "missing-required":
				c.entries[0].name = "lost"
			case "optional-present":
				c.entries = append(c.entries, providerHistoryRecoveryCatalogEntry{evidence: true, name: "maybe", info: info})
				c.names += 5
			case "wrong-namespace":
				c.entries[0].evidence = true
			case "extra-entry":
				c.entries = append(c.entries, providerHistoryRecoveryCatalogEntry{name: "extra"}, providerHistoryRecoveryCatalogEntry{name: "other"})
			case "unselected":
				r.catalogBefore = nil
			}
			if mode == "unfunded" {
				r.meter.upper[10] = 0
			}
			before := r.meter.actual
			got := w.matches(c)
			if got != (mode == "same" || mode == "optional-present") || r.catalogResult != nil || r.active || r.used || r.store != nil {
				t.Fatal("private namespace comparison accepted inconsistent DATA or gained a recovery role", mode, got)
			}
			if mode == "unfunded" && (!r.meter.failed || r.meter.actual != before) {
				t.Fatal("unfunded name comparison renewed work")
			}
		})
	}
}

func TestProviderHistoryRecoveryNameDerivationRejectsUnselectedInputs(t *testing.T) {
	w := historyRecoveryNameUnitWriter()
	before := w.recovery.meter.actual
	if out, err := w.recovery.expectedCatalogLocked(context.Background(), -1); err == nil || out != nil {
		t.Fatal("invalid model pass produced expected names")
	}
	if w.addCheckpoint(nil, true) || w.addMemberAt(nil, 0, true) || w.addMember(0, true) || w.recovery.meter.actual != before || len(w.names) != 0 {
		t.Fatal("unselected checkpoint/member produced names or spent work")
	}
	c := &providerHistoryRecoveryCheckpoint{}
	if w.addMemberAt(c, 0, false) || len(w.names) != 0 || w.recovery.catalogResult != nil {
		t.Fatal("member without an original selection supplied a namespace name")
	}
}

package registry011

import (
	"context"
	"os"
	"testing"
)

func TestProviderHistoryRecoveryCatalogComparisonRejectsChangedIdentitiesAndMissingEntries(t *testing.T) {
	root, err := os.Stat(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	other, err := os.Stat(t.TempDir())
	if err != nil || os.SameFile(root, other) {
		t.Fatal("two distinct owned UNIT directories", err)
	}
	// These are inert comparison DATA, not selected catalogs or a recovery
	// result. No path from these entries is opened or used as a selector.
	for _, mode := range []string{"same", "root-identity", "missing-entry", "entry-identity"} {
		t.Run(mode, func(t *testing.T) {
			r := &providerHistoryRecoveryOwner{meter: providerReplayUnitMeter()}
			a := &providerHistoryRecoveryCatalog{recovery: r, complete: true, names: 4, roots: [2]os.FileInfo{root, root},
				entries: []providerHistoryRecoveryCatalogEntry{{name: "kept", info: root}}}
			b := &providerHistoryRecoveryCatalog{recovery: r, complete: true, names: 4, roots: [2]os.FileInfo{root, root},
				entries: []providerHistoryRecoveryCatalogEntry{{name: "kept", info: root}}}
			a.owner, b.owner = a, b
			switch mode {
			case "root-identity":
				b.roots[1] = other
			case "missing-entry":
				b.entries[0].name = "lost"
			case "entry-identity":
				b.entries[0].info = other
			}
			if got := r.catalogsEqualLocked(a, b); got != (mode == "same") || r.meter.failed ||
				a.entries[0].name != "kept" || a.entries[0].info != root || a.roots != ([2]os.FileInfo{root, root}) || r.catalogResult != nil {
				t.Fatal("catalog comparison accepted changed DATA, mutated its predecessor or created a result", mode, got)
			}
		})
	}
}

func TestProviderHistoryRecoveryTailComparisonAndCancelledInputKeepUnpublishedData(t *testing.T) {
	for _, mode := range []string{"same", "attempt-result", "entry-index", "entry-digest"} {
		t.Run(mode, func(t *testing.T) {
			r := &providerHistoryRecoveryOwner{meter: providerReplayUnitMeter()}
			a := providerHistoryRecoveryTailData{count: 1}
			a.entries[0] = providerHistoryRecoveryTailInput{kind: providerHistoryRecoveryHead, state: providerHistoryRecoveryWholeInput,
				bytes: 1, digest: admissionTestDigest("a"), whole: true}
			b := a
			switch mode {
			case "attempt-result":
				b.attemptResult = "unverified-unit-result"
			case "entry-index":
				b.entries[0].index++
			case "entry-digest":
				b.entries[0].digest = admissionTestDigest("b")
			}
			before := a
			if got := r.tailDataEqualLocked(&a, &b); got != (mode == "same") || a != before || r.meter.failed || r.tailModels != ([2]*providerHistoryRecoveryTailPayloads{}) {
				t.Fatal("tail comparison accepted inconsistent DATA or published a model", mode, got)
			}
		})
	}
	t.Run("cancelled-original-input", func(t *testing.T) {
		o, r := historyRecoveryInputFixture(t)
		r.store.mu.Lock()
		defer r.store.mu.Unlock()
		old, clocks, pin, held := o.meter.actual, o.store.clock.read, o.pin, o.store.stored
		files, bytes := r.inputFiles, r.inputBytes
		original := r.confirmed.admission
		if original == nil || !original.shapeLocked(r, r.confirmed) {
			t.Fatal("genuine admitted and consumed checkpoint precursor")
		}
		// Read-only shape DATA only: no copied descriptor is installed in
		// the selected chain or used to grant a recovery/execution role.
		descriptor, admitted := *original, *original.checkpoint
		descriptor.owner, descriptor.checkpoint = &descriptor, &admitted
		admitted.owner, admitted.admission = &admitted, &descriptor
		if !descriptor.shapeLocked(r, &admitted) {
			t.Fatal("ordinary admitted-checkpoint shape")
		}
		descriptor.purpose = "unsupported-unit-purpose"
		if descriptor.shapeLocked(r, &admitted) || !original.shapeLocked(r, r.confirmed) {
			t.Fatal("unknown admission purpose acquired a valid shape or changed the original")
		}
		descriptor.purpose = original.purpose
		descriptor.requests.bytes = 0
		admitted.inventory.entries[0] = descriptor.requests
		if descriptor.shapeLocked(r, &admitted) || !original.shapeLocked(r, r.confirmed) {
			t.Fatal("zero-byte request DATA acquired an admission shape or changed its original")
		}
		model := providerHistoryRecoveryTailPayloads{recovery: r}
		plan := providerHistoryRecoveryTailPlan{checkpoint: r.initial, kind: providerHistoryRecoveryAnchorCustody, bytes: 1, raw: "x"}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		out, whole, err := model.read(ctx, plan)
		if err == nil || out != nil || whole || model.data != (providerHistoryRecoveryTailData{}) || r.active || !r.failed || !r.closed ||
			r.inputFiles != files || r.inputBytes != bytes || o.meter.actual != old || o.store.clock.read != clocks || o.pin != pin || o.store.stored != held {
			t.Fatal("cancelled original read exposed tail DATA, performed I/O or revived execution", err)
		}
		charged := r.meter.actual
		if out, whole, err := model.read(ctx, plan); err == nil || out != nil || whole || r.meter.actual != charged || model.data != (providerHistoryRecoveryTailData{}) {
			t.Fatal("cancelled inspection resumed or published a partial tail")
		}
	})
}

func TestProviderHistoryRecoveryFirstFinalizedOriginalRequiresSelectedProvenance(t *testing.T) {
	var absentRecovery *providerHistoryRecoveryOwner
	if absentRecovery.prepareInitialDescriptorLocked(nil) != nil || (&providerHistoryRecoveryOwner{}).prepareInitialDescriptorLocked(nil) != nil {
		t.Fatal("missing initializer/checkpoint created a selected descriptor")
	}
	var absent *providerHistoryRecoveryInitialModel
	missingRecovery := &providerHistoryRecoveryInitialModel{}
	missingRecovery.owner = missingRecovery
	unselected := &providerHistoryRecoveryInitialModel{recovery: &providerHistoryRecoveryOwner{meter: providerReplayUnitMeter()}}
	unselected.owner = unselected
	for _, model := range []*providerHistoryRecoveryInitialModel{absent, missingRecovery, unselected} {
		if raw := model.firstFinalizedOriginalObserved("request", admissionTestDigest("a")); raw != nil {
			t.Fatal("missing selected provenance exposed an original")
		}
	}
	if unselected.recovery.meter.actual != (candidateWorkVector{}) || unselected.recovery.meter.failed {
		t.Fatal("unselected original lookup consumed or reset an observer")
	}
}

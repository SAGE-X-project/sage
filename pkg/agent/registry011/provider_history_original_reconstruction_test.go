package registry011

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestProviderHistoryFailedCollectionInspectionRequiresTheOriginalTerminalGroup(t *testing.T) {
	group, _ := historyRecoveryGroupFixture(t, 1)
	p, reader, store := group.parent, group.parent.recovery, group.store
	work, clocks, held, ack := group.meter.actual, store.clock.read, store.stored, group.last
	if out, err := p.inspectFailedCollectionCandidate(context.Background()); err == nil || out != nil || reader.used || reader.active ||
		reader.meter.actual != (candidateWorkVector{}) || group.meter.actual != work || store.clock.read != clocks || store.stored != held {
		t.Fatal("a healthy collector entered the failed inspection allowance", err)
	}
	store.mu.Lock()
	group.failLocked()
	store.objects, store.frontierReplay, store.bindings = nil, nil, nil
	store.mu.Unlock()
	out, err := p.inspectFailedCollectionCandidate(context.Background())
	if err != nil || out == nil || !out.selectedLocked(reader) || !reader.used || !reader.active || reader.failed || reader.closed ||
		group.meter.actual != work || store.clock.read != clocks || store.stored != held || group.last != ack || store.activeMeter.Load() != nil ||
		!p.failed || !group.failed || !group.meter.closed || !group.meter.failed {
		t.Fatal("the original failed collection did not complete read-only inspection", err)
	}
	readWork, files, bytes := reader.meter.actual, reader.inputFiles, reader.inputBytes
	if out, err := p.inspectFailedCollectionCandidate(context.Background()); err == nil || out != nil || reader.meter.actual != readWork || reader.inputFiles != files || reader.inputBytes != bytes {
		t.Fatal("failed collection inspection was retried with a renewed allowance", err)
	}
	var missing *providerHistoryProgramOwner
	if out, err := missing.inspectFailedCollectionCandidate(context.Background()); err == nil || out != nil {
		t.Fatal("missing original parent created an inspection caller")
	}
}

func TestProviderHistoryFailedCollectionInspectionClosesCancelledReconstruction(t *testing.T) {
	group, _ := historyRecoveryGroupFixture(t, 1)
	p, reader, store := group.parent, group.parent.recovery, group.store
	work, clocks, held, ack := group.meter.actual, store.clock.read, store.stored, group.last
	store.mu.Lock()
	group.failLocked()
	store.objects, store.frontierReplay, store.bindings = nil, nil, nil
	store.mu.Unlock()
	base, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx := &historyOriginalCancellationContext{Context: base, cancel: cancel, reader: reader, phase: "inventory-start"}
	if out, err := p.inspectFailedCollectionCandidate(ctx); err != errProviderIntegration || out != nil || !ctx.reached ||
		!reader.used || reader.active || !reader.closed || !reader.failed || !reader.meter.closed || !reader.meter.failed ||
		reader.catalogBefore == nil || reader.catalogAfter != nil || reader.catalogResult != nil {
		t.Fatal("cancelled original inspection returned partial data or retained its lane", err)
	}
	if group.meter.actual != work || store.clock.read != clocks || store.stored != held || group.last != ack || store.activeMeter.Load() != nil {
		t.Fatal("cancelled inspection borrowed execution, clocks or the original reservation")
	}
	readWork, files, bytes := reader.meter.actual, reader.inputFiles, reader.inputBytes
	if out, err := p.inspectFailedCollectionCandidate(context.Background()); err != errProviderIntegration || out != nil ||
		reader.meter.actual != readWork || reader.inputFiles != files || reader.inputBytes != bytes || reader.active || !reader.closed {
		t.Fatal("fresh context renewed a failed original inspection", err)
	}
}

func TestProviderHistoryOriginalReconstructionOwnsBothCompletePasses(t *testing.T) {
	for _, mode := range []string{"consumed", "settled", "pending-admission"} {
		t.Run(mode, func(t *testing.T) {
			group, reader, _ := historyRecoveryCatalogFixture(t, mode)
			store := group.store
			work, clocks, held, ack, pin := group.meter.actual, store.clock.read, store.stored, group.last, group.pin
			originals := providerHistoryProvisionedOriginalBytes(t, store.config)
			out, err := reader.reconstructOriginalsCandidate(context.Background())
			if err != nil || out == nil || !out.selectedLocked(reader) || reader.catalogBefore == reader.catalogAfter ||
				reader.groupModels[0] == reader.groupModels[1] || reader.initialModels[0] == reader.initialModels[1] ||
				!reader.groupModels[0].selectedLocked(reader) || !reader.groupModels[1].selectedLocked(reader) ||
				!reader.completedMemberModels[0].selectedLocked(reader) || !reader.completedMemberModels[1].selectedLocked(reader) ||
				!reader.tailModels[0].selectedLocked(reader) || !reader.tailModels[1].selectedLocked(reader) {
				t.Fatal("original reconstruction omitted a fresh pass or final catalog", mode, err)
			}
			if group.meter.actual != work || store.clock.read != clocks || store.stored != held || group.last != ack || group.pin != pin ||
				store.activeMeter.Load() != nil || !group.failed || !group.meter.closed || !reader.active || reader.closed || reader.failed ||
				store.objects != nil || store.frontierReplay != nil || store.bindings != nil ||
				!reflect.DeepEqual(originals, providerHistoryProvisionedOriginalBytes(t, store.config)) {
				t.Fatal("read-only reconstruction changed old execution, originals or reservation")
			}
			measured, files, bytes := reader.meter.actual, reader.inputFiles, reader.inputBytes
			if again, err := reader.reconstructOriginalsCandidate(context.Background()); err == nil || again != nil ||
				reader.meter.actual != measured || reader.inputFiles != files || reader.inputBytes != bytes || !reader.active {
				t.Fatal("a repeated reconstruction reset or consumed the original lane", err)
			}
		})
	}
}

func TestProviderHistoryOriginalReconstructionRejectsDetachedAndCancelledEntry(t *testing.T) {
	group, reader, _ := historyRecoveryCatalogFixture(t, "consumed")
	work, clocks := reader.meter.actual, group.store.clock.read
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	var absent context.Context
	for _, ctx := range []context.Context{absent, cancelled} {
		if out, err := reader.reconstructOriginalsCandidate(ctx); err == nil || out != nil || reader.meter.actual != work ||
			reader.catalogBefore != nil || reader.inputFiles != 0 || !reader.active || group.store.clock.read != clocks {
			t.Fatal("cancelled entry performed original observation or consumed a second lane")
		}
	}
	var missing *providerHistoryRecoveryOwner
	if out, err := missing.reconstructOriginalsCandidate(context.Background()); err == nil || out != nil {
		t.Fatal("detached reader returned a reconstruction")
	}
	if missing.reconstructOriginalPassCandidate(context.Background(), 0) == nil || missing.drainOriginalInventoryCandidate(context.Background(), nil) == nil {
		t.Fatal("detached original traversal entered a child")
	}
}

func TestProviderHistoryOriginalReconstructionSelectsTheOriginalCollectionHeader(t *testing.T) {
	manifest, recorded := historyCollectionRecordedRunManifest(t)
	group, reader, _ := historyFirstFinalizedFixtureWithManifest(t, string(recorded["canonical-deployment"]), manifest)
	clocks, work, held := group.store.clock.read, group.meter.actual, group.store.stored
	out, err := reader.reconstructOriginalsCandidate(context.Background())
	if err != nil || out == nil || !out.selectedLocked(reader) {
		t.Fatal("actual original collection reconstruction", err)
	}
	for pass := range 2 {
		initial := reader.initialModels[pass]
		if !initial.finalizedSelected || initial.firstFinalized != manifest.input.journal.Deployment ||
			reader.groupModels[pass].ledger.collection == nil || !reader.groupModels[pass].ledger.collection.initial() {
			t.Fatal("reconstruction omitted this pass's original selected header", pass)
		}
	}
	if group.store.clock.read != clocks || group.meter.actual != work || group.store.stored != held || group.store.activeMeter.Load() != nil {
		t.Fatal("collection reconstruction renewed an execution or clock role")
	}
}

// A valid caller context cancels at an actual monotone reconstruction
// boundary. Only this owned UNIT context selects the boundary; no production
// callback, replacement source, clock or native hook is introduced.
type historyOriginalCancellationContext struct {
	context.Context
	cancel  context.CancelFunc
	reader  *providerHistoryRecoveryOwner
	phase   string
	reached bool
}

func (c *historyOriginalCancellationContext) Err() error {
	r, reached := c.reader, false
	switch c.phase {
	case "inventory-start":
		reached = r.catalogBefore != nil
	case "inventory-object":
		reached = r.inventoryWalks[0] != nil && r.inventoryWalks[0].current != nil
	case "inventory-end":
		w := r.inventoryWalks[0]
		reached = w != nil && w.remaining == 0 && w.current == nil
	case "scan":
		reached = r.inventoryWalks[0] != nil && r.inventoryWalks[0].complete
	case "legacy":
		reached = r.scanWalks[0] != nil && r.scanWalks[0].complete
	case "initial":
		reached = r.legacyPrefixes[0] != nil
	case "group":
		reached = r.initialModels[0] != nil
	case "members":
		reached = r.groupModels[0] != nil
	case "tail":
		reached = r.completedMemberModels[0] != nil
	case "catalog-end":
		reached = r.tailModels[1] != nil
	}
	if reached {
		c.reached = true
		c.cancel()
	}
	return c.Context.Err()
}

func TestProviderHistoryOriginalReconstructionCancellationClosesTheSameLane(t *testing.T) {
	for _, phase := range []string{"inventory-start", "inventory-object", "inventory-end", "scan", "legacy", "initial", "group", "members", "tail", "catalog-end"} {
		t.Run(phase, func(t *testing.T) {
			group, reader, _ := historyRecoveryCatalogFixture(t, "consumed")
			base, cancel := context.WithCancel(context.Background())
			defer cancel()
			ctx := &historyOriginalCancellationContext{Context: base, cancel: cancel, reader: reader, phase: phase}
			work, clocks, held, ack := group.meter.actual, group.store.clock.read, group.store.stored, group.last
			if out, err := reader.reconstructOriginalsCandidate(ctx); err == nil || out != nil || !ctx.reached || reader.active || !reader.closed || !reader.failed ||
				!reader.meter.closed || !reader.meter.failed || reader.catalogResult != nil || reader.catalogAfter != nil {
				t.Fatal("cancellation returned a partial result or left an entered lane active", phase, err)
			}
			if group.meter.actual != work || group.store.clock.read != clocks || group.store.stored != held || group.last != ack || group.store.activeMeter.Load() != nil {
				t.Fatal("cancelled reconstruction changed the failed execution or original hold")
			}
		})
	}
}

func TestProviderHistoryOriginalReconstructionClosesAnEnteredNamespaceFailure(t *testing.T) {
	group, reader, _ := historyRecoveryCatalogFixture(t, "consumed")
	store := group.store
	if err := os.WriteFile(filepath.Join(reader.config.evidenceDirectory, "owned-unexpected-original"), []byte("ordinary owned UNIT data"), 0600); err != nil {
		t.Fatal(err)
	}
	work, clocks, held, ack := group.meter.actual, store.clock.read, store.stored, group.last
	if out, err := reader.reconstructOriginalsCandidate(context.Background()); err == nil || out != nil || reader.active || !reader.closed || !reader.failed ||
		!reader.meter.closed || !reader.meter.failed || reader.catalogBefore == nil || reader.catalogResult != nil || reader.catalogAfter != nil {
		t.Fatal("entered namespace failure returned a complete reconstruction", err)
	}
	if group.meter.actual != work || store.clock.read != clocks || store.stored != held || group.last != ack || store.activeMeter.Load() != nil {
		t.Fatal("reconstruction failure borrowed execution, clock or acknowledgment")
	}
}

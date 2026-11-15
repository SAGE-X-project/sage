package registry011

import (
	"context"
	"reflect"
	"testing"
)

// Cancellation belongs to a standard context. Its bounded check count is
// selected before this call from an unchanged positive read of the same owned
// directory. No clock, observer counter or native file result is rewritten.
type historyRecoveryCatalogCancelContext struct {
	context.Context
	cancel        context.CancelFunc
	checks, limit int
}

func (c *historyRecoveryCatalogCancelContext) Err() error {
	c.checks++
	if c.limit > 0 && c.checks == c.limit {
		c.cancel()
	}
	return c.Context.Err()
}

func TestProviderHistoryRecoveryCatalogPrivateBoundsAndCancellationKeepOriginalRoles(t *testing.T) {
	var absent *providerHistoryRecoveryOwner
	if out, err := absent.beginCatalogCandidate(context.Background()); err == nil || out != nil {
		t.Fatal("absent recovery owner created an original catalog")
	}
	o, recovery, _ := historyRecoveryCatalogFixture(t, "consumed")
	store := o.store
	store.mu.Lock()
	control := &providerHistoryRecoveryCatalog{recovery: recovery,
		entries: make([]providerHistoryRecoveryCatalogEntry, 0, int(recovery.parent.bounds.catalogEntries))}
	rootContext, cancelRoot := context.WithCancel(context.Background())
	rootProbe := &historyRecoveryCatalogCancelContext{Context: rootContext, cancel: cancelRoot}
	err := control.readRootLocked(rootProbe, 0, recovery.custody)
	cancelRoot()
	if err != nil || len(control.entries) == 0 || control.roots[0] == nil ||
		rootProbe.checks != len(control.entries)+3 || uint64(len(control.entries)) >= recovery.parent.bounds.catalogEntries {
		store.mu.Unlock()
		t.Fatal("positive owned-root entry and final-close prerequisite", err)
	}
	wholeContext, cancelWhole := context.WithCancel(context.Background())
	wholeProbe := &historyRecoveryCatalogCancelContext{Context: wholeContext, cancel: cancelWhole}
	whole, err := recovery.readCatalogLocked(wholeProbe)
	cancelWhole()
	if err != nil || whole == nil || !whole.complete || whole.roots[0] == nil || whole.roots[1] == nil ||
		wholeProbe.checks != len(whole.entries)+8 || recovery.catalogBefore != nil || recovery.catalogAfter != nil {
		store.mu.Unlock()
		t.Fatal("positive two-root native catalog without registering C0", err)
	}
	store.mu.Unlock()

	for _, mode := range []string{"slots", "name-bytes", "duplicate", "nil-context", "loop-cancel", "root-end-cancel", "catalog-end-cancel"} {
		t.Run(mode, func(t *testing.T) {
			store.mu.Lock()
			defer store.mu.Unlock()
			c := &providerHistoryRecoveryCatalog{recovery: recovery,
				entries: make([]providerHistoryRecoveryCatalogEntry, 0, int(recovery.parent.bounds.catalogEntries))}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			probe := &historyRecoveryCatalogCancelContext{Context: ctx, cancel: cancel}
			var input context.Context = probe
			switch mode {
			case "slots":
				c.entries = c.entries[:cap(c.entries)]
			case "name-bytes":
				c.names = recovery.parent.bounds.catalogNameBytes
			case "duplicate":
				c.entries = append(c.entries, control.entries...)
				c.names = control.names
			case "nil-context":
				input = nil
			case "loop-cancel":
				probe.limit = 2
			case "root-end-cancel":
				probe.limit = rootProbe.checks
			case "catalog-end-cancel":
				probe.limit = wholeProbe.checks
			}
			// Only this unselected catalog's bookkeeping DATA differs. The
			// genuine read-only owner and all original allowances stay fixed.
			entries, names := len(c.entries), c.names
			work, files, bytes, bounds := recovery.meter.actual, recovery.inputFiles, recovery.inputBytes, recovery.bounds
			old, reads, pin, ack, held := o.meter.actual, store.clock.read, o.pin, o.last, store.stored
			originals := providerHistoryProvisionedOriginalBytes(t, store.config)
			var out *providerHistoryRecoveryCatalog
			var err error
			if mode == "catalog-end-cancel" {
				out, err = recovery.readCatalogLocked(input)
			} else {
				err = c.readRootLocked(input, 0, recovery.custody)
			}
			if err == nil || out != nil || c.owner != nil || c.complete || c.roots[0] != nil || c.roots[1] != nil ||
				recovery.catalogBefore != nil || recovery.catalogAfter != nil || recovery.catalogResult != nil ||
				!recovery.activeLocked(recovery.meter) || recovery.meter.failed || recovery.bounds != bounds ||
				recovery.inputFiles != files || recovery.inputBytes != bytes || o.meter.actual != old || store.clock.read != reads ||
				o.pin != pin || o.last != ack || store.stored != held || store.activeMeter.Load() != nil ||
				!reflect.DeepEqual(originals, providerHistoryProvisionedOriginalBytes(t, store.config)) {
				t.Fatal("private catalog refusal installed C0, renewed a role or changed original execution", mode, err)
			}
			for coordinate, before := range work {
				if recovery.meter.actual[coordinate] < before {
					t.Fatal("catalog refusal refunded actual original observation", mode, coordinate)
				}
			}
			if mode == "slots" || mode == "name-bytes" || mode == "duplicate" {
				if len(c.entries) != entries || c.names != names || probe.checks != 2 || recovery.meter.actual[10] <= work[10] {
					t.Fatal("private capacity or duplicate refusal skipped the first actual directory entry", mode)
				}
			} else if mode == "nil-context" {
				if recovery.meter.actual != work || probe.checks != 0 {
					t.Fatal("missing context reached native directory observation")
				}
			} else if probe.checks != probe.limit || probe.Context.Err() != context.Canceled || recovery.meter.actual[10] <= work[10] ||
				mode == "root-end-cancel" && len(c.entries) != len(control.entries) || mode == "loop-cancel" && len(c.entries) != 0 {
				t.Fatal("cancellation failed before its positively observed directory boundary", mode, probe.checks, probe.limit)
			}
		})
	}
}

func TestProviderHistoryRecoveryCatalogInputQuotesRejectUnselectedPrivateData(t *testing.T) {
	r := historyRecoveryNameUnitWriter().recovery
	if upper, ok := r.catalogInputUpperLocked(1); !ok || upper[10] == 0 {
		t.Fatal("positive finite input quotation before any catalog")
	}
	// Neither catalog nor owner is selected. This is pure comparison DATA,
	// not a replacement for a real read-only owner or its original C0.
	r.catalogBefore = &providerHistoryRecoveryCatalog{}
	before := r.meter.actual
	if upper, ok := r.catalogInputUpperLocked(1); ok || upper != (candidateWorkVector{}) ||
		r.catalogInputMatchesLocked(nil, "unselected", nil, nil) || r.meter.actual != before ||
		r.owner != nil || r.store != nil || r.active || r.used || r.catalogAfter != nil || r.catalogResult != nil {
		t.Fatal("unselected private catalog supplied input permission or spent original work")
	}
}

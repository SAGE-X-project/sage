package registry011

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestProviderHistoryRecoveryReadsInitialAndCompletedMemberNamespace(t *testing.T) {
	o, d := historyMemberDispatchFixtureWithOptions(t, 1, nil, true)
	historyRetainNormalPrefix(t, o, d, 1)
	r, s := o.parent.recovery, o.store
	o.store.mu.Lock()
	o.failLocked()
	o.store.mu.Unlock()
	if r.beginInspectionCandidate(context.Background(), o) != nil || r.bounds.maximumFiles != 128 || r.completed != 1 {
		t.Fatal("genuine pre-H0 namespace profile and original completed member")
	}
	n := r.initialDescriptor
	data := *d // Filename-only DATA, never installed as a live dispatch.
	data.owner, data.meter = &data, nil
	items := []struct {
		label    string
		kind     providerHistoryRecoveryFileKind
		index    int
		evidence bool
		name     string
	}{
		{"anchor-custody", providerHistoryRecoveryAnchorCustody, 0, false, "ANCHOR"},
		{"anchor-evidence", providerHistoryRecoveryAnchorEvidence, 0, true, "ANCHOR"},
		{"legacy-intent", providerHistoryRecoveryLegacyIntent, 1, false, providerCustodyName("intent", 1)},
		{"legacy-commit", providerHistoryRecoveryLegacyCommit, 1, false, providerCustodyName("commit", 1)},
		{"legacy-frontier", providerHistoryRecoveryLegacyFrontier, 1, false, providerCustodyName("frontier", 1)},
		{"legacy-pending", providerHistoryRecoveryLegacyPending, 1, true, providerCustodyName("pending", 1)},
		{"legacy-highwater", providerHistoryRecoveryLegacyHighwater, 0, false, providerCustodyName("highwater", 0)},
		{"setup-intent", providerHistoryRecoverySetupIntent, 0, false, providerCustodyName("setup-intent", 0)},
		{"setup-commit", providerHistoryRecoverySetupCommit, 0, false, providerCustodyName("setup-commit", 0)},
		{"setup-claim", providerHistoryRecoverySetupClaim, 0, true, providerCustodyName("setup-claim", 0)},
		{"scan-head-custody", providerHistoryRecoveryScanHeadCustody, 0, false, "SCAN-HEAD"},
		{"scan-head-evidence", providerHistoryRecoveryScanHeadEvidence, 0, true, "SCAN-HEAD"},
		{"scan-highwater-custody", providerHistoryRecoveryScanHighwaterCustody, 0, false, providerCustodyName("scan-highwater", 0)},
		{"scan-highwater-evidence", providerHistoryRecoveryScanHighwaterEvidence, 0, true, providerCustodyName("scan-highwater", 0)},
		{"scan-intent", providerHistoryRecoveryScanIntent, 1, false, providerCustodyName("scan-intent", 1)},
		{"scan-commit", providerHistoryRecoveryScanCommit, 1, false, providerCustodyName("scan-commit", 1)},
		{"scan-claim", providerHistoryRecoveryScanClaim, 1, true, providerCustodyName("scan-claim", 1)},
		{"member-response", providerHistoryRecoveryMemberResponse, 0, true, data.originalNameLocked(0, "response")},
		{"member-tls", providerHistoryRecoveryMemberTLS, 0, true, data.originalNameLocked(0, "tls")},
		{"last-chunk-custody", providerHistoryRecoveryScanLastChunkCustody, 0, false, providerScanReceiptName("chunk", n.lastChunk, nil)},
		{"last-chunk-evidence", providerHistoryRecoveryScanLastChunkEvidence, 0, true, providerScanReceiptName("chunk", n.lastChunk, nil)},
		{"initial-inventory", providerHistoryRecoveryInitialInventory, 0, true, "inventory-" + n.objects[2:]},
	}
	old, reads, held, pin, ack := o.meter.actual, s.clock.read, s.stored, o.pin, o.last
	before := providerHistoryProvisionedOriginalBytes(t, s.config)
	var bytes uint64
	for _, item := range items {
		t.Run(item.label, func(t *testing.T) {
			checkpoint, directory := r.initial, s.config.custodyDirectory
			if item.evidence {
				directory = s.config.evidenceDirectory
			}
			if item.kind == providerHistoryRecoveryMemberResponse || item.kind == providerHistoryRecoveryMemberTLS {
				checkpoint = r.consumption
			}
			want, err := os.ReadFile(filepath.Join(directory, item.name))
			if err != nil || len(want) == 0 {
				t.Fatal("actual retained namespace precursor", item.label, err)
			}
			out, err := r.readCheckpointInputCandidate(context.Background(), checkpoint, item.kind, item.index)
			if err != nil || out == nil || out.state != providerHistoryRecoveryWholeInput || !reflect.DeepEqual(out.raw, want) {
				t.Fatal("closed purpose selected a different original/root or omitted EOF", item.label, err)
			}
			bytes += uint64(len(out.raw))
		})
	}
	if r.inputFiles != uint64(len(items)) || r.inputBytes != bytes || o.meter.actual != old || s.clock.read != reads || s.stored != held || o.pin != pin || o.last != ack ||
		s.activeMeter.Load() != nil || !r.active || !o.failed || r.completed != 1 || r.sent != 1 || !reflect.DeepEqual(before, providerHistoryProvisionedOriginalBytes(t, s.config)) {
		t.Fatal("namespace observation changed old execution, logical state, originals or cumulative counters")
	}
}

func TestProviderHistoryRecoveryNamespaceRefusesUnselectedOrdinalAndRoleBeforeIO(t *testing.T) {
	o, r := historyRecoveryInputFixture(t)
	for _, mode := range []string{"wrong-initial", "legacy-zero", "legacy-overflow", "setup-overflow", "scan-zero", "scan-overflow", "anchor-index", "chunk-index", "inventory-index", "member-wrong-c", "member-not-started", "member-future"} {
		t.Run(mode, func(t *testing.T) {
			c, kind, index := r.initial, providerHistoryRecoveryLegacyIntent, 1
			switch mode {
			case "wrong-initial":
				c, kind, index = r.confirmed, providerHistoryRecoverySetupIntent, 0
			case "legacy-zero":
				index = 0
			case "legacy-overflow":
				index = int(r.initialDescriptor.main.sequence + 1)
			case "setup-overflow":
				kind, index = providerHistoryRecoverySetupClaim, int(r.initialDescriptor.setups)
			case "scan-zero":
				kind, index = providerHistoryRecoveryScanIntent, 0
			case "scan-overflow":
				kind, index = providerHistoryRecoveryScanHighwaterEvidence, int(r.initialDescriptor.scan.ordinal+1)
			case "anchor-index":
				kind = providerHistoryRecoveryAnchorCustody
			case "chunk-index":
				kind = providerHistoryRecoveryScanLastChunkEvidence
			case "inventory-index":
				kind = providerHistoryRecoveryInitialInventory
			case "member-wrong-c":
				kind, index = providerHistoryRecoveryMemberResponse, 0
			case "member-not-started":
				c, kind, index = r.consumption, providerHistoryRecoveryMemberResponse, 0
			case "member-future":
				c, kind = r.consumption, providerHistoryRecoveryMemberTLS
			}
			actual, files, bytes, old, reads, held := r.meter.actual, r.inputFiles, r.inputBytes, o.meter.actual, o.store.clock.read, o.store.stored
			if out, err := r.readCheckpointInputCandidate(context.Background(), c, kind, index); err == nil || out != nil || !r.active ||
				r.meter.actual != actual || r.inputFiles != files || r.inputBytes != bytes || o.meter.actual != old || o.store.clock.read != reads || o.store.stored != held {
				t.Fatal("unselected namespace role or ordinal reached observation", mode, err)
			}
		})
	}
	if out, err := r.readCheckpointInputCandidate(context.Background(), r.initial, providerHistoryRecoveryAnchorCustody, 0); err != nil || out == nil || out.state != providerHistoryRecoveryWholeInput {
		t.Fatal("otherwise-valid original namespace after isolated refusals", err)
	}
}

func TestProviderHistoryRecoveryMemberNamespaceKeepsUncertainPhysicalPrefixWithoutRetry(t *testing.T) {
	for _, mode := range []string{"original", "expected-name"} {
		t.Run(mode, func(t *testing.T) { historyRecoveryMemberNamespaceUncertainty(t, mode) })
	}
}

func historyRecoveryMemberNamespaceUncertainty(t *testing.T, mode string) {
	t.Helper()
	o, d := historyMemberDispatchFixture(t, 1)
	s, r := o.store, o.parent.recovery
	data := *d
	data.owner, data.meter = &data, nil
	name := data.originalNameLocked(0, "response")
	syncFile, reached := s.evidence.syncFile, false
	s.evidence.syncFile = func(file *os.File) error {
		if err := syncFile(file); err != nil {
			return err
		}
		if filepath.Base(file.Name()) == name {
			reached = true
			return errors.New("owned unit original response sync uncertainty")
		}
		return nil
	}
	member := d.coordinator.ticket.table.members[0]
	conn := providerIntegrationOwnedTLS(t, member.request.body, `{"jsonrpc":"2.0","id":1,"result":"0xaa36a7"}`)
	if out, err := o.retainMemberCandidateWithConnection(context.Background(), d, 0, conn); err == nil || out != nil || !reached ||
		!o.failed || r.memberAttempt == nil || r.memberCapture == nil || r.completed != 0 || r.sent != 1 || d.attempt == nil || d.attempt.files != 0 ||
		r.beginInspectionCandidate(context.Background(), o) != nil {
		t.Fatal("actual uncertain response spool and sealed partial-attempt precursor", err)
	}
	old, reads, held, pin, ack := o.meter.actual, s.clock.read, s.stored, o.pin, o.last
	if mode == "expected-name" {
		// Inert unit DATA fault after genuine response sync uncertainty.
		// The original C/ordinal-derived filename and actual files are intact.
		r.memberAttempt.responseName = "inert expected-name mismatch"
		out, err := r.readCheckpointInputCandidate(context.Background(), r.consumption, providerHistoryRecoveryMemberResponse, 0)
		if err == nil || out != nil || r.active || !r.closed || !r.failed || r.inputFiles != 0 || r.inputBytes != 0 || r.meter.actual[10] == 0 ||
			r.completed != 0 || r.sent != 1 || o.pin != pin || o.last != ack || o.meter.actual != old || s.clock.read != reads || s.stored != held {
			t.Fatal("changed sealed name reached a file probe or renewed logical authority", err)
		}
		return
	}
	response, err := r.readCheckpointInputCandidate(context.Background(), r.consumption, providerHistoryRecoveryMemberResponse, 0)
	if err != nil || response == nil || response.state != providerHistoryRecoveryWholeInput || len(response.raw) == 0 {
		t.Fatal("actual response file existed before uncertain sync completion", err)
	}
	tls, err := r.readCheckpointInputCandidate(context.Background(), r.consumption, providerHistoryRecoveryMemberTLS, 0)
	if err != nil || tls == nil || tls.state != providerHistoryRecoveryMissingInput || len(tls.raw) != 0 || r.inputFiles != 2 || r.inputBytes != uint64(len(response.raw)) ||
		r.completed != 0 || r.sent != 1 || d.attempt.files != 0 || o.pin != pin || o.last != ack || o.meter.actual != old || s.clock.read != reads || s.stored != held ||
		s.activeMeter.Load() != nil || !r.active {
		t.Fatal("partial physical member files advanced completion, retried capture or renewed a hold/ACK", err)
	}
}

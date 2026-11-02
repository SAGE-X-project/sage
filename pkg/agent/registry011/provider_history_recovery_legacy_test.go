package registry011

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func historyRecoveryLegacySource(t *testing.T, r *providerHistoryRecoveryOwner, pass int, want map[string]providerIntegrationObjectRef) uint64 {
	t.Helper()
	bytes := historyRecoveryScanSource(t, r, pass, want)
	w, err := r.newScanWalkCandidate(context.Background(), pass)
	if err != nil {
		t.Fatal("genuine independent scan precursor", err)
	}
	for !w.complete {
		if _, err := w.nextRecordCandidate(context.Background()); err != nil {
			t.Fatal("actual original scan chain", err)
		}
	}
	return bytes + w.controlBytes + w.result.metadata
}

func TestProviderHistoryRecoveryLegacyReconstructsIndependentControlAndEnrollmentPrefixes(t *testing.T) {
	o, r, want := historyRecoveryInventoryFixtureWithLegacy(t, true)
	if r.bounds.maximumFiles != 256 || r.bounds.maximumBytes != 4<<20 || r.bounds.work[2] != 8<<20 || r.bounds.work[10] != 1<<44 {
		t.Fatal("genuine separate pre-H0 legacy UNIT profile")
	}
	s := o.store
	old, reads, held, pin, ack := o.meter.actual, s.clock.read, s.stored, o.pin, o.last
	before := providerHistoryProvisionedOriginalBytes(t, s.config)
	var bytes uint64
	for pass := range 2 {
		bytes += historyRecoveryLegacySource(t, r, pass, want)
		prefix, err := r.replayLegacyPrefixCandidate(context.Background(), pass)
		if err != nil || prefix == nil || !prefix.selectedLocked(r) || prefix.pin != r.initialDescriptor.main || len(prefix.setups) != int(r.initialDescriptor.setups) {
			t.Fatal("genuine fresh typed legacy prefix", pass, err)
		}
		for i, record := range prefix.records {
			if record.root == nil || record.after.sequence != uint64(i+1) || record.before.sequence != uint64(i) || record.kind != [4]string{"enroll", "admit", "consume", "settle"}[i] ||
				record.frontier.enrollment != record.root.digest || record.frontier.objects != prefix.source.frames[i].inventory.digest {
				t.Fatal("typed initial enrollment/control sequence was skipped or rebound", i)
			}
			for _, item := range []struct{ name, raw string }{{"intent", record.intent}, {"commit", record.commit}, {"frontier", record.rawFrontier}} {
				actual, err := os.ReadFile(filepath.Join(r.config.custodyDirectory, providerCustodyName(item.name, uint64(i+1))))
				if err != nil || string(actual) != item.raw {
					t.Fatal("record differs from actual original", err)
				}
			}
		}
		if pass == 1 && (prefix == r.legacyPrefixes[0] || prefix.records[0].root == r.legacyPrefixes[0].records[0].root || prefix.source.frames[0] == r.legacyPrefixes[0].source.frames[0]) {
			t.Fatal("independent prefixes share mutable typed roots or actual-frame state")
		}
		bytes += prefix.bytes
	}
	n := r.initialDescriptor
	legacyFiles := 2 + 3*n.setups + (n.main.ordinal + 1) + 4*n.main.sequence
	wantFiles := 2 * (4 + uint64(len(want)) + 4 + 5*n.scan.ordinal + 4*n.records + legacyFiles)
	if r.inputFiles != wantFiles || r.inputBytes != bytes || r.inputFiles > r.bounds.maximumFiles || o.meter.actual != old || s.clock.read != reads || s.stored != held || o.pin != pin || o.last != ack ||
		s.activeMeter.Load() != nil || s.objects != nil || !o.failed || !o.meter.closed || !r.active || r.closed || r.failed || !reflect.DeepEqual(before, providerHistoryProvisionedOriginalBytes(t, s.config)) {
		t.Fatal("typed prefix replay renewed old execution, changed originals or reset the cumulative row")
	}
	if json.Unmarshal([]byte(`{}`), &providerHistoryRecoveryLegacyPrefix{}) == nil {
		t.Fatal("serialized DATA manufactured a selected prefix")
	}
}

func TestProviderHistoryRecoveryLegacyRefusesIncompleteAndDetachedSourcesBeforeIO(t *testing.T) {
	_, r, want := historyRecoveryInventoryFixtureWithLegacy(t, true)
	work := r.meter.actual
	if out, err := r.replayLegacyPrefixCandidate(context.Background(), 0); err == nil || out != nil || r.meter.actual != work || r.inputFiles != 0 || !r.active {
		t.Fatal("prefix selected without actual source", err)
	}
	historyRecoveryLegacySource(t, r, 0, want)
	historyRecoveryLegacySource(t, r, 1, want)
	for _, mode := range []string{"missing-frame", "incomplete-frame", "detached-source", "second-before-first", "negative-pass", "excess-pass"} {
		t.Run(mode, func(t *testing.T) {
			actual, files, bytes := r.meter.actual, r.inputFiles, r.inputBytes
			var out *providerHistoryRecoveryLegacyPrefix
			var err error
			switch mode {
			case "missing-frame":
				frame := r.inventoryWalks[0].frames[0]
				r.inventoryWalks[0].frames[0] = nil
				out, err = r.replayLegacyPrefixCandidate(context.Background(), 0)
				r.inventoryWalks[0].frames[0] = frame
			case "incomplete-frame":
				frame := r.inventoryWalks[0].frames[0]
				frame.complete = false
				out, err = r.replayLegacyPrefixCandidate(context.Background(), 0)
				frame.complete = true
			case "detached-source":
				original := r.scanWalks[0].source
				copy := *original
				copy.owner = &copy
				r.scanWalks[0].source = &copy
				out, err = r.replayLegacyPrefixCandidate(context.Background(), 0)
				r.scanWalks[0].source = original
			case "second-before-first":
				out, err = r.replayLegacyPrefixCandidate(context.Background(), 1)
			case "negative-pass":
				out, err = r.replayLegacyPrefixCandidate(context.Background(), -1)
			case "excess-pass":
				out, err = r.replayLegacyPrefixCandidate(context.Background(), 2)
			}
			if err == nil || out != nil || r.meter.actual != actual || r.inputFiles != files || r.inputBytes != bytes || !r.active || r.closed {
				t.Fatal("unselected prefix source reached observation", mode, err)
			}
		})
	}
	positive, err := r.replayLegacyPrefixCandidate(context.Background(), 0)
	if err != nil || positive == nil || !positive.selectedLocked(r) {
		t.Fatal("otherwise-valid prefix after isolated source refusals", err)
	}
	copy := *positive
	copy.owner = &copy
	if copy.selectedLocked(r) {
		t.Fatal("copied prefix replaced the exact registered original")
	}
	work, files, bytes := r.meter.actual, r.inputFiles, r.inputBytes
	if out, err := r.replayLegacyPrefixCandidate(context.Background(), 0); err == nil || out != nil || r.meter.actual != work || r.inputFiles != files || r.inputBytes != bytes || !r.active {
		t.Fatal("same prefix pass renewed its row", err)
	}
}

func TestProviderHistoryRecoveryLegacyRejectsChangedActualOriginalWithoutPublishingPrefix(t *testing.T) {
	for _, mode := range []string{"anchor-custody", "anchor-evidence", "setup-claim", "setup-intent", "setup-commit", "highwater-zero", "highwater-one", "highwater-final", "intent", "commit", "frontier", "pending"} {
		t.Run(mode, func(t *testing.T) {
			o, r, want := historyRecoveryInventoryFixtureWithLegacy(t, true)
			historyRecoveryLegacySource(t, r, 0, want)
			directory, name := r.config.custodyDirectory, "ANCHOR"
			switch mode {
			case "anchor-evidence":
				directory = r.config.evidenceDirectory
			case "setup-claim":
				directory, name = r.config.evidenceDirectory, providerCustodyName("setup-claim", 0)
			case "setup-intent":
				name = providerCustodyName("setup-intent", 0)
			case "setup-commit":
				name = providerCustodyName("setup-commit", 0)
			case "highwater-zero":
				name = providerCustodyName("highwater", 0)
			case "highwater-one":
				name = providerCustodyName("highwater", 1)
			case "highwater-final":
				name = providerCustodyName("highwater", r.initialDescriptor.main.ordinal)
			case "intent":
				name = providerCustodyName("intent", 1)
			case "commit":
				name = providerCustodyName("commit", 2)
			case "frontier":
				name = providerCustodyName("frontier", 3)
			case "pending":
				directory, name = r.config.evidenceDirectory, providerCustodyName("pending", 4)
			}
			path := filepath.Join(directory, name)
			raw, err := os.ReadFile(path)
			if err != nil || len(raw) == 0 {
				t.Fatal("actual original prefix precursor", err)
			}
			// Exclusive inert unit fixture files; no wire/host/symlink/race helper.
			for i := range raw {
				raw[i] = '!'
			}
			if err := os.WriteFile(path, raw, 0600); err != nil {
				t.Fatal(err)
			}
			old, reads, held, pin, ack, files := o.meter.actual, o.store.clock.read, o.store.stored, o.pin, o.last, r.inputFiles
			out, err := r.replayLegacyPrefixCandidate(context.Background(), 0)
			if err == nil || out != nil || r.legacyPrefixes[0] != nil || r.active || !r.closed || !r.failed || r.inputFiles <= files || o.meter.actual != old || o.store.clock.read != reads || o.store.stored != held || o.pin != pin || o.last != ack || o.store.activeMeter.Load() != nil {
				t.Fatal("changed original published a partial prefix or revived execution", mode, err)
			}
		})
	}
}

func TestProviderHistoryRecoveryLegacyCancellationStopsBeforeNextEffect(t *testing.T) {
	for _, mode := range []string{"nil", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			o, r, want := historyRecoveryInventoryFixtureWithLegacy(t, true)
			historyRecoveryLegacySource(t, r, 0, want)
			var ctx context.Context
			if mode == "cancel" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(context.Background())
				cancel()
			}
			work, files, bytes, old, reads := r.meter.actual, r.inputFiles, r.inputBytes, o.meter.actual, o.store.clock.read
			out, err := r.replayLegacyPrefixCandidate(ctx, 0)
			if err == nil || out != nil || r.active || !r.closed || !r.failed || r.legacyPrefixes[0] != nil || r.meter.actual != work || r.inputFiles != files || r.inputBytes != bytes || o.meter.actual != old || o.store.clock.read != reads {
				t.Fatal("cancelled valid prefix reached an effect or renewed execution", err)
			}
		})
	}
}

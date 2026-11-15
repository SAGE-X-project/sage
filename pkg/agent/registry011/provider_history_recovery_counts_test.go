package registry011

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestProviderHistoryRecoveryReconstructsHistoricalRepeatedReadsFromEachActualPrefix(t *testing.T) {
	o, r, want := historyRecoveryInventoryFixtureWithLegacy(t, true)
	// The completed producer's counters, receipt arrays and reader are not
	// provenance for this reconstruction. Keep only the original owner seal.
	b := r.initialDescriptor.first.body
	b.objects, b.rawBytes, b.receipts, b.reader = ^uint64(0), ^uint64(0), nil, nil
	old, reads, held, ack, pin := o.meter.actual, o.store.clock.read, o.store.stored, o.last, o.pin
	before := providerHistoryProvisionedOriginalBytes(t, o.store.config)
	var previous [2]providerHistoryRecoveryReadCount
	for pass := range 2 {
		historyRecoveryModelPrefix(t, r, pass, want)
		files, bytes := r.inputFiles, r.inputBytes
		m, err := r.replayInitialModelCandidate(context.Background(), pass)
		if err != nil || m == nil || !m.selectedLocked(r) || !m.scanReadsVerified || r.inputFiles != files+2 || r.inputBytes != bytes {
			t.Fatal("two actual stable empty-lock reads and complete historical counters", pass, err)
		}
		counts, scan, frame := m.scanReadCounts, m.prefix.scan, m.prefix.source.frames[0]
		// The source path performs ten counted setup reads. E0 reads all fresh
		// objects plus 29 repeated/control/zero-lock target reads; SCAN and
		// retained command/chunk controls are separately excluded.
		if counts[0].objects != 10 || counts[1].objects != uint64(len(frame.objects))+29 ||
			counts[0].objects != scan.records[1].objects || counts[1].objects != scan.records[0].objects ||
			counts[0].bytes != scan.records[1].bytes || counts[1].bytes != scan.records[0].bytes ||
			counts[0].objects+counts[1].objects != scan.result.objects || counts[0].bytes+counts[1].bytes != scan.result.bytes ||
			scan.result.objects == uint64(len(want)) || scan.result.objects == r.inputFiles || scan.result.bytes == r.inputBytes {
			t.Fatal("historical repeated-read counters were replaced by unique objects or recovery I/O")
		}
		if pass == 1 && (counts != previous || r.initialModels[0].prefix == m.prefix || r.initialModels[0].frontier == m.frontier) {
			t.Fatal("two actual historical reconstructions disagree or share their mutable model")
		}
		previous = counts
	}
	if o.meter.actual != old || o.store.clock.read != reads || o.store.stored != held || o.last != ack || o.pin != pin ||
		o.store.activeMeter.Load() != nil || b.objects != ^uint64(0) || b.rawBytes != ^uint64(0) || b.receipts != nil || b.reader != nil ||
		!reflect.DeepEqual(before, providerHistoryProvisionedOriginalBytes(t, o.store.config)) {
		t.Fatal("historical counter reconstruction used or restored a producer cache, clock or execution role")
	}
}

func TestProviderHistoryRecoveryRejectsInconsistentHistoricalReadDiagnostics(t *testing.T) {
	for _, mode := range []string{"setup-objects", "setup-bytes", "main-objects", "main-bytes", "balanced-objects", "balanced-bytes", "post-receipt-pin", "recovery-counts"} {
		t.Run(mode, func(t *testing.T) {
			o, r, want := historyRecoveryInventoryFixtureWithLegacy(t, true)
			historyRecoveryModelPrefix(t, r, 0, want)
			scan := r.scanWalks[0]
			// These are private intermediate diagnostic faults after genuine
			// original receipt and control verification, never network edits.
			switch mode {
			case "setup-objects":
				scan.records[1].objects++
				scan.result.objects++
			case "setup-bytes":
				scan.records[1].bytes++
				scan.result.bytes++
			case "main-objects":
				scan.records[0].objects--
				scan.result.objects--
			case "main-bytes":
				scan.records[0].bytes--
				scan.result.bytes--
			case "balanced-objects":
				scan.records[1].objects++
				scan.records[0].objects--
			case "balanced-bytes":
				scan.records[1].bytes++
				scan.records[0].bytes--
			case "post-receipt-pin":
				scan.records[0].objects += 2
				scan.result.objects += 2
				extra := 2 * r.legacyPrefixes[0].highwaterOriginalBytes[1]
				scan.records[0].bytes += extra
				scan.result.bytes += extra
			case "recovery-counts":
				scan.result.objects, scan.result.bytes = r.inputFiles, r.inputBytes
			}
			files, bytes, old, reads, held, ack := r.inputFiles, r.inputBytes, o.meter.actual, o.store.clock.read, o.store.stored, o.last
			m, err := r.replayInitialModelCandidate(context.Background(), 0)
			if err == nil || m != nil || r.initialModels[0] != nil || r.inputFiles != files || r.inputBytes != bytes || r.active || !r.closed || !r.failed ||
				o.meter.actual != old || o.store.clock.read != reads || o.store.stored != held || o.last != ack || o.store.activeMeter.Load() != nil {
				t.Fatal("inconsistent historical counts published a model or reached lock I/O", mode, err)
			}
		})
	}
}

func TestProviderHistoryRecoveryRequiresActualEmptyOriginalLocksForZeroByteReads(t *testing.T) {
	for _, evidence := range []bool{false, true} {
		name := "custody"
		if evidence {
			name = "evidence"
		}
		t.Run(name, func(t *testing.T) {
			o, r, want := historyRecoveryInventoryFixtureWithLegacy(t, true)
			historyRecoveryModelPrefix(t, r, 0, want)
			root := r.config.custodyDirectory
			if evidence {
				root = r.config.evidenceDirectory
			}
			path := filepath.Join(root, "AUTHORITY.LOCK")
			if raw, err := os.ReadFile(path); err != nil || len(raw) != 0 || os.WriteFile(path, []byte("!"), 0600) != nil {
				t.Fatal("owned normal original lock precursor", err)
			}
			files, old, reads, held := r.inputFiles, o.meter.actual, o.store.clock.read, o.store.stored
			out, err := r.replayInitialModelCandidate(context.Background(), 0)
			probes := uint64(1)
			if evidence {
				probes = 2
			}
			if err == nil || out != nil || r.initialModels[0] != nil || r.inputFiles != files+probes || r.active || !r.closed || !r.failed ||
				o.meter.actual != old || o.store.clock.read != reads || o.store.stored != held {
				t.Fatal("a nonempty original lock was treated as a successful zero-byte read", err)
			}
		})
	}
}

func TestProviderHistoryRecoveryReadCounterArithmeticRejectsOverflowAndOriginalChunkBounds(t *testing.T) {
	_, r, want := historyRecoveryInventoryFixtureWithLegacy(t, true)
	historyRecoveryModelPrefix(t, r, 0, want)
	m, err := r.replayInitialModelCandidate(context.Background(), 0)
	if err != nil || m == nil || !m.selectedLocked(r) {
		t.Fatal("genuine positive model before isolated arithmetic cases", err)
	}
	for _, mode := range []string{"product", "objects", "bytes", "chunk-objects", "chunk-bytes"} {
		t.Run(mode, func(t *testing.T) {
			out, copies, bytes := providerHistoryRecoveryReadCount{}, uint64(1), uint64(1)
			switch mode {
			case "product":
				copies, bytes = ^uint64(0), 2
			case "objects":
				out.objects = ^uint64(0)
			case "bytes":
				out.bytes = ^uint64(0)
			case "chunk-objects":
				out.objects = r.config.bounds.scan.chunkObjects
			case "chunk-bytes":
				out.bytes = r.config.bounds.scan.chunkRawBytes
			}
			before, work, files := out, r.meter.actual, r.inputFiles
			if m.addScanReads(&out, copies, bytes) || out != before || r.meter.actual != work || r.inputFiles != files || !r.active {
				t.Fatal("refused arithmetic changed its result or consumed an input", mode)
			}
		})
	}
}

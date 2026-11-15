package registry011

import (
	"context"
	"testing"
)

// Fixed private reconstruction faults only. Each case obtains its own real
// original prefix/read-only lane first; no peer, file or selected limit is
// replaced. The temporary DATA edits are restored under the original lock,
// and no failed owner, clock, observation or allowance is renewed.
func TestProviderHistoryRecoveryReadCountsRejectIncompleteOriginalDerivations(t *testing.T) {
	for _, mode := range []string{"record-cut", "missing-enrollment", "missing-highwater", "anchor-capacity", "setup-overflow",
		"anchor-byte-mismatch", "missing-anchor", "missing-intent", "missing-mutation", "missing-reference"} {
		t.Run(mode, func(t *testing.T) {
			o, r, originals := historyRecoveryInventoryFixtureWithLegacy(t, true)
			historyRecoveryModelPrefix(t, r, 0, originals)
			model, err := r.replayInitialModelCandidate(context.Background(), 0)
			if err != nil || model == nil || !model.selectedLocked(r) {
				t.Fatal("original complete semantic model precursor", err)
			}
			r.store.mu.Lock()
			defer r.store.mu.Unlock()
			old, clocks, pin, held := o.meter.actual, o.store.clock.read, o.pin, o.store.stored
			files, bytes := r.inputFiles, r.inputBytes
			working, frontier := *model, *model.frontier
			working.owner, working.frontier = &working, &frontier
			frontier.owner = &frontier
			frontier.anchors = append([]providerIntegrationObject(nil), frontier.anchors...)
			working.complete, working.scanReadsVerified = false, false
			working.scanReadCounts = [2]providerHistoryRecoveryReadCount{}
			if got, ok := working.deriveScanReadCounts(); !ok || got != model.scanReadCounts {
				t.Fatal("fresh historical counts do not match the verified original prefix")
			}
			prefix := working.prefix
			previous := *prefix
			frame := prefix.source.frames[0]
			objects := frame.objects
			defer func() {
				*prefix, frame.objects = previous, objects
				if !model.selectedLocked(r) || !r.activeLocked(r.meter) || o.meter.actual != old || o.store.clock.read != clocks || o.pin != pin || o.store.stored != held ||
					r.inputFiles != files || r.inputBytes != bytes {
					t.Error("read-count UNIT changed original selection, I/O, clock or execution")
				}
			}()
			switch mode {
			case "record-cut":
				working.through = 3
			case "missing-enrollment":
				prefix.records[0].root = nil
			case "missing-highwater":
				prefix.highwaterOriginalBytes = append([]uint64(nil), prefix.highwaterOriginalBytes...)
				prefix.highwaterOriginalBytes[0] = 0
			case "anchor-capacity":
				prefix.anchorBytes = ^uint64(0) - 1
			case "setup-overflow":
				prefix.setupOriginalBytes = append([]uint64(nil), prefix.setupOriginalBytes...)
				prefix.setupOriginalBytes[0] = ^uint64(0)
			case "anchor-byte-mismatch":
				frontier.anchors[0].raw = "unverified-unit-anchor-bytes"
			case "missing-anchor":
				frontier.anchors = frontier.anchors[:2]
			case "missing-intent":
				prefix.records[0].intent = ""
			case "missing-mutation":
				frame.objects = make([]providerIntegrationObject, len(objects))
				copy(frame.objects, objects)
				found := false
				for i := range frame.objects {
					if frame.objects[i].kind == "index-mutation" {
						frame.objects[i].kind = "unverified-unit-kind"
						found = true
					}
				}
				if !found {
					t.Fatal("ordinary initial index-mutation precursor missing")
				}
			case "missing-reference":
				root := *prefix.records[0].root
				root.digest = admissionTestDigest("e")
				prefix.records[0].root = &root
			}
			if _, ok := working.deriveScanReadCounts(); ok || working.scanReadsVerified || working.scanReadCounts != ([2]providerHistoryRecoveryReadCount{}) || working.complete {
				t.Fatal("incomplete historical derivation was marked verified", mode)
			}
			// A false result may contain private partial arithmetic DATA. It
			// is discarded, never adopted as verified scan counts or a model.
		})
	}
}

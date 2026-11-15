package registry011

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func historyReplayReadsFixture(t *testing.T) (*providerHistoryGroupOwner, *providerHistoryReplayReads) {
	t.Helper()
	o, _ := historyDispatchReplayFixture(t, 1)
	o.phase = "preparing-dispatch"
	o.store.activeMeter.Store(o.meter)
	p, err := newProviderHistoryReplayReadsLocked(context.Background(), o, 0)
	if err != nil || p == nil {
		t.Fatal("live original C and observer required", err)
	}
	o.meter.replayReads = p
	return o, p
}

func TestProviderHistoryDispatchRecordsEachIndependentOriginalRead(t *testing.T) {
	for _, count := range []int{1, 3, providerHistoryGroupMembers} {
		name := "one"
		switch count {
		case 3:
			name = "several"
		case providerHistoryGroupMembers:
			name = "maximum"
		}
		t.Run(name, func(t *testing.T) {
			o, ack := historyDispatchReplayFixture(t, count)
			d, err := o.prepareDispatchCandidate(context.Background(), ack)
			if err != nil || d == nil || o.meter.replayReads != nil || d.replayReads[0] == nil || d.replayReads[1] == nil {
				t.Fatal("both fully interpreted passes need their own completed transcript", err)
			}
			left, right := d.replayReads[0], d.replayReads[1]
			if left == right || !left.complete || !right.complete || left.failed || right.failed || left.context != nil || right.context != nil || left.role != 0 || right.role != 1 ||
				len(left.rows) < 4 || len(left.rows) != len(right.rows) || &left.rows[0] == &right.rows[0] || left.root == right.root ||
				left.model != d.coordinator.head || right.model != d.custodian.head {
				t.Fatal("roles, arrays, full interpretation or model provenance was shared")
			}
			for _, p := range []*providerHistoryReplayReads{left, right} {
				if p.owner != p || p.group != o || p.meter != o.meter || p.ack != ack || p.cutoff != o.stageBefore ||
					p.current != o.pin || p.initial != o.start.main || p.dataRootObserved(nil) != p.root {
					t.Fatal("recording is not bound to the exact original C and independent pass")
				}
				for ordinal, row := range p.rows {
					if row != right.rows[ordinal] || row.maximum == 0 || row.bytes == 0 || row.bytes > row.maximum {
						t.Fatal("ordered original read identity changed", ordinal)
					}
					root := o.store.config.custodyDirectory
					if row.root == 1 {
						root = o.store.config.evidenceDirectory
					}
					raw, err := os.ReadFile(filepath.Join(root, row.name))
					if err != nil {
						t.Fatal(err)
					}
					sum := sha256.Sum256(raw)
					if uint64(len(raw)) != row.bytes || "0x"+hex.EncodeToString(sum[:]) != row.digest {
						t.Fatal("descriptor did not come from the actual original bytes", ordinal)
					}
				}
				a, c := o.admission.claim, ack.claim
				metadata := o.store.config.bounds.maximumMetadataPerCommit
				head, frame := uint64(candidateLedgerMetadataLimit), uint64(providerHistoryCustodyFrameBytes)
				expected := []struct {
					root    uint8
					name    string
					maximum uint64
				}{
					{0, "HEAD", metadata}, {0, o.store.name("highwater", o.start.main.ordinal), metadata},
					{0, o.store.name("frontier", o.start.main.sequence), metadata}, {0, o.store.name("commit", o.start.main.sequence), metadata},
					{0, "HEAD", head}, {0, "HEAD", head}, {0, o.store.name("highwater", o.stageBefore.pin.ordinal), head},
					{0, o.store.name("highwater", o.admission.pin.ordinal), head},
					{1, providerHistoryStoredName(a.request, nil), a.request.bytes}, {1, providerHistoryStoredName(a.ticket, nil), a.ticket.bytes},
					{1, providerHistoryInventoryName(a.inventory.digest, nil), metadata},
					{0, o.store.name("intent", a.frame.after.sequence), frame}, {1, o.store.name("history-pending", a.frame.after.sequence), frame},
					{0, o.store.name("commit", a.frame.after.sequence), frame}, {0, o.store.name("frontier", a.frame.after.sequence), frame},
					{0, "HEAD", head}, {0, o.store.name("highwater", o.admission.pin.ordinal), head}, {0, o.store.name("highwater", ack.pin.ordinal), head},
					{1, providerHistoryStoredName(c.consumption, nil), c.consumption.bytes}, {1, providerHistoryInventoryName(c.inventory.digest, nil), metadata},
					{0, o.store.name("intent", c.frame.after.sequence), frame}, {1, o.store.name("history-pending", c.frame.after.sequence), frame},
					{0, o.store.name("commit", c.frame.after.sequence), frame}, {0, o.store.name("frontier", c.frame.after.sequence), frame},
				}
				if len(p.rows) != len(expected) {
					t.Fatal("legacy first-stage original sequence changed", len(p.rows), len(expected))
				}
				for ordinal, want := range expected {
					if p.rows[ordinal].root != want.root || p.rows[ordinal].name != want.name || p.rows[ordinal].maximum != want.maximum {
						t.Fatal("original read order or repeated HEAD/highwater input was omitted", ordinal)
					}
				}
			}
			before := right.root
			left.rows[0].name = "inert-unit-different-name"
			if left.dataRootObserved(nil) == left.root || right.dataRootObserved(nil) != before {
				t.Fatal("one role's descriptor mutation changed or authenticated the other role")
			}
		})
	}
}

func TestProviderHistoryReplayReadRecordingRefusesDetachedAndSpentData(t *testing.T) {
	for _, mode := range []string{"nil-recorder", "copied-recorder", "nil-store", "different-root", "empty-name", "wide-name", "zero-maximum", "empty-input", "wide-input", "full", "complete", "failed", "wrong-role", "different-ack", "different-cutoff", "different-program", "different-costs", "inactive", "cancelled", "spent"} {
		t.Run(mode, func(t *testing.T) {
			o, p := historyReplayReadsFixture(t)
			s, disk, name, maximum, raw := o.store, o.store.custody, "unit-original", int64(8), []byte("data")
			m := o.meter
			switch mode {
			case "nil-recorder":
				p = nil
			case "copied-recorder":
				copy := *p
				p = &copy
			case "nil-store":
				s = nil
			case "different-root":
				disk = &retentionDisk{}
			case "empty-name":
				name = ""
			case "wide-name":
				name = strings.Repeat("x", 256)
			case "zero-maximum":
				maximum = 0
			case "empty-input":
				raw = nil
			case "wide-input":
				maximum = 1
			case "full":
				p.maximum = 1
				p.rows = []providerHistoryReplayRead{{}}
			case "complete":
				p.complete = true
			case "failed":
				p.failed = true
			case "wrong-role":
				p.role = 2
			case "different-ack":
				p.ack = o.admission
			case "different-cutoff":
				p.cutoffPin.sequence++
			case "different-program":
				p.program = admissionTestDigest("e")
			case "different-costs":
				p.costs = admissionTestDigest("f")
			case "inactive":
				s.activeMeter.Store(nil)
			case "cancelled":
				ctx, cancel := context.WithCancel(context.Background())
				p.context = ctx
				cancel()
			case "spent":
				m.upper[10] = m.actual[10]
			}
			before := m.actual
			if p.recordLocked(s, disk, name, maximum, raw) || p != nil && p.complete && !p.failed {
				t.Fatal("detached or incomplete recording became usable", mode)
			}
			for i, n := range before {
				if m.actual[i] < n {
					t.Fatal("refusal refunded original consumption", i)
				}
			}
		})
	}
}

func TestProviderHistoryReplayReadCommitmentBindsEveryOrderedRowField(t *testing.T) {
	for _, mode := range []string{"order", "drop", "root", "name", "digest", "maximum", "length", "role", "model", "copied", "empty", "invalid-root", "invalid-digest", "invalid-length"} {
		t.Run(mode, func(t *testing.T) {
			o, ack := historyDispatchReplayFixture(t, 1)
			d, err := o.prepareDispatchCandidate(context.Background(), ack)
			if err != nil || d == nil {
				t.Fatal(err)
			}
			p := d.replayReads[0]
			root := p.root
			switch mode {
			case "order":
				p.rows[0], p.rows[1] = p.rows[1], p.rows[0]
			case "drop":
				p.rows = p.rows[:len(p.rows)-1]
			case "root":
				p.rows[0].root = 1
			case "name":
				p.rows[0].name = "inert-unit-other-original"
			case "digest":
				p.rows[0].digest = admissionTestDigest("d")
			case "maximum":
				p.rows[0].maximum++
			case "length":
				p.rows[0].bytes--
			case "role":
				p.role = 1
			case "model":
				p.model = admissionTestDigest("e")
			case "copied":
				copy := *p
				p = &copy
			case "empty":
				p.rows = nil
			case "invalid-root":
				p.rows[0].root = 2
			case "invalid-digest":
				p.rows[0].digest = "not-a-digest"
			case "invalid-length":
				p.rows[0].bytes = p.rows[0].maximum + 1
			}
			if p.dataRootObserved(nil) == root {
				t.Fatal("changed ordered read DATA retained its commitment", mode)
			}
		})
	}
}

func TestProviderHistoryFailedReplayNeverCompletesOrLeavesRecorderAttached(t *testing.T) {
	o, ack := historyDispatchReplayFixture(t, 1)
	s, before := o.store, o.meter.actual
	name := providerHistoryStoredName(o.admission.claim.request, nil)
	if err := os.WriteFile(filepath.Join(s.config.evidenceDirectory, name), []byte("inert unit original mismatch"), 0600); err != nil {
		t.Fatal(err)
	}
	if d, err := o.prepareDispatchCandidate(context.Background(), ack); err == nil || d != nil || o.dispatch != nil ||
		o.meter.replayReads != nil || !o.failed || !o.meter.failed || !o.meter.closed || s.activeMeter.Load() != nil {
		t.Fatal("failed semantic replay produced a completed recording or kept its input recorder")
	}
	for i, n := range before {
		if o.meter.actual[i] < n {
			t.Fatal("failed full replay refunded its prefix", i)
		}
	}
	var p providerHistoryReplayReads
	if json.Unmarshal([]byte(`{}`), &p) == nil {
		t.Fatal("read transcript restored from JSON")
	}
}

func TestProviderHistoryReplayReadRootRetainsLateObserverFailure(t *testing.T) {
	o, ack := historyDispatchReplayFixture(t, 1)
	d, err := o.prepareDispatchCandidate(context.Background(), ack)
	if err != nil || d == nil {
		t.Fatal(err)
	}
	p := d.replayReads[0]
	probe := providerHistorySemanticUnitMeter()
	if p.dataRootObserved(probe) != p.root || probe.actual[10] == 0 || probe.actual[2] != 0 || probe.failed {
		t.Fatal("ordinary encoded/hashed DATA precursor failed")
	}
	// Required quota includes conservative encoder preflight, not just the
	// actual units finally consumed. Discover the minimum on separate UNIT
	// observers; no operational meter, hold or original allowance is reset.
	low := uint64(0)
	high := probe.actual[10] + uint64(candidateLedgerMetadataLimit)*uint64(candidateLedgerMetadataLimit) + 1
	bounded := providerHistorySemanticUnitMeter()
	bounded.upper[10] = high
	if p.dataRootObserved(bounded) != p.root || bounded.failed || bounded.actual != probe.actual {
		t.Fatal("independent UNIT search bracket did not fund the complete root")
	}
	for low < high {
		middle := low + (high-low)/2
		meter := providerHistorySemanticUnitMeter()
		meter.upper[10] = middle
		if p.dataRootObserved(meter) == p.root {
			high = middle
		} else {
			low = middle + 1
		}
	}
	if low == 0 {
		t.Fatal("nonempty original transcript required no logical work")
	}
	for _, mode := range []string{"exact", "one-short"} {
		t.Run(mode, func(t *testing.T) {
			meter := providerHistorySemanticUnitMeter()
			meter.upper[10] = low
			if mode == "one-short" {
				meter.upper[10]--
			}
			root := p.dataRootObserved(meter)
			if mode == "exact" {
				if root != p.root || meter.failed || meter.actual != probe.actual {
					t.Fatal("minimum sufficient DATA allowance refused or changed actual consumption")
				}
				return
			}
			if root != "" || !meter.failed || meter.actual[10] == 0 {
				t.Fatal("late quota refusal completed a root or dropped its consumed prefix")
			}
			before := meter.actual
			if p.dataRootObserved(meter) != "" || meter.actual != before {
				t.Fatal("spent observer was replaced, refunded or restored")
			}
		})
	}
}

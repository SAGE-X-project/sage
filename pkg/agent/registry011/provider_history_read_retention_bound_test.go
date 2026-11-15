package registry011

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"strconv"
	"testing"
)

// Independent ordinary JSON/SHA reference for the closed ASCII metadata
// retained by the actual fixture. It does not call the product root/encoder
// helper. Work is actual encoded bytes, never the8KiB preflight maximum.
func historyReplayReadRootReference(t *testing.T, p *providerHistoryReplayReads) (string, candidateWorkVector, uint64) {
	t.Helper()
	var work candidateWorkVector
	encode := func(domain string, value map[string]string) (string, uint64) {
		raw, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		n := uint64(len(raw))
		actual := n*n + 1 + n + uint64(len(domain)) + 2
		work[10] += actual
		sum := sha256.Sum256(append([]byte(domain+"\x00"), raw...))
		return "0x" + hex.EncodeToString(sum[:]), actual
	}
	root, base := encode("sage-provider-history-replay-reads-v1", map[string]string{
		"format": "sage-provider-history-replay-reads-v1", "role": strconv.Itoa(int(p.role)),
		"program": p.program, "profile": p.config.profile, "cost_revision": p.costs,
		"initial_frontier": p.initial.frontier, "initial_commit": p.initial.commit,
		"current_frontier": p.current.frontier, "current_commit": p.current.commit,
		"cutoff_frontier": p.cutoffPin.frontier, "cutoff_commit": p.cutoffPin.commit,
		"model": p.model, "rows": strconv.Itoa(len(p.rows)), "maximum_rows": strconv.FormatUint(p.maximum, 10),
	})
	for ordinal, row := range p.rows {
		root, _ = encode("sage-provider-history-replay-read-v1", map[string]string{
			"previous": root, "ordinal": strconv.Itoa(ordinal), "root": strconv.Itoa(int(row.root)),
			"name": row.name, "maximum": strconv.FormatUint(row.maximum, 10),
			"bytes": strconv.FormatUint(row.bytes, 10), "digest": row.digest,
		})
	}
	return root, work, base
}

func TestProviderHistoryReadRetentionUsesIndependentCountAndByteLiterals(t *testing.T) {
	for _, tc := range []struct {
		input                                                    providerHistoryReadRetentionInput
		maximum, recorders, recorded, refreshed, calls, rootRows uint64
		recordedBytes, refreshedBytes                            uint64
		recording, refresh, commitments, comparisons, total      uint64
	}{
		{providerHistoryReadRetentionInput{1, 1, 1, 1024, 4096, 1024, 4096, 4096}, 34, 2, 68, 68, 8, 272, 393216, 393216, 436872, 402328, 18792786608, 1330, 18793627138},
		{providerHistoryReadRetentionInput{1, 32, 64, 1024, 64 << 20, 65536, 1 << 20, 262144}, 97, 2, 194, 6208, 132, 12804, 139329536, 4458545152, 139454084, 4459377024, 868226741052, 26068, 872825598228},
		{providerHistoryReadRetentionInput{2, 3, 64, 1024, 64 << 20, 65536, 1 << 20, 262144}, 124, 4, 442, 682, 20, 2248, 286916608, 433192960, 287200372, 433284348, 152221571480, 3458, 152942059658},
		{providerHistoryReadRetentionInput{3, 96, 64, 1024, 64 << 20, 65536, 1 << 20, 262144}, 271, 6, 1104, 35328, 396, 72864, 501743616, 16055795712, 502452384, 16060529664, 4916998380096, 78204, 4933561440348},
	} {
		got, ok := providerHistoryReadRetentionUpper(tc.input)
		fixed := tc.recorders * (uint64(reflect.TypeFor[providerHistoryReplayReads]().Size()) + 1)
		if !ok || got.capacity != tc.input.catalogEntries+tc.input.firstObjects+128 || got.maximumRows != tc.maximum ||
			got.recorders != tc.recorders || got.recordedRows != tc.recorded || got.refreshedRows != tc.refreshed || got.rootCalls != tc.calls || got.rootRows != tc.rootRows ||
			got.recordedBytes != tc.recordedBytes || got.refreshedBytes != tc.refreshedBytes ||
			got.initialization != (candidateWorkVector{10: fixed}) || got.recording != (candidateWorkVector{10: tc.recording}) ||
			got.refresh != (candidateWorkVector{10: tc.refresh}) || got.commitments != (candidateWorkVector{10: tc.commitments}) ||
			got.comparisons != (candidateWorkVector{10: tc.comparisons}) || got.total != (candidateWorkVector{10: tc.total + fixed}) {
			t.Fatal("original read-retention upper differs from independent literals", tc, got, ok)
		}
	}
}

func TestProviderHistoryOriginalReadsRetainDuplicatesAndExactAdditionalCharge(t *testing.T) {
	o, p := historyReplayReadsFixture(t)
	m := o.meter
	for index := range 2 {
		before := m.actual
		raw, err := o.store.read(o.store.custody, "HEAD", candidateLedgerMetadataLimit)
		if err != nil || len(raw) == 0 || len(p.rows) != index+1 || p.failed || p.complete || m.failed {
			t.Fatal("genuine original HEAD read/record precursor", err)
		}
		sum := sha256.Sum256(raw)
		row := p.rows[index]
		want := before
		// Actual read n+1 is distinct from additional rawSHA n+1,
		// name copies2*(4+1) and explicit descriptor129.
		want[10] += 2*uint64(len(raw)) + 141
		if row.root != 0 || row.name != "HEAD" || row.maximum != candidateLedgerMetadataLimit || row.bytes != uint64(len(raw)) ||
			row.digest != "0x"+hex.EncodeToString(sum[:]) || m.actual != want {
			t.Fatal("duplicate read was deduplicated or raw/name/descriptor charge changed", row, m.actual, want)
		}
		raw[0] ^= 1
		if p.rows[index] != row {
			t.Fatal("retained digest/name descriptor owns a mutable caller buffer")
		}
	}
	if p.rows[0] != p.rows[1] || p.meter != m || m.replayReads != p {
		t.Fatal("ordered duplicate original input or observer was replaced")
	}
}

func TestProviderHistoryReadCommitmentsMatchIndependentOriginalJSONAndSHA(t *testing.T) {
	for _, count := range []int{1, providerHistoryGroupMembers} {
		t.Run(strconv.Itoa(count), func(t *testing.T) {
			o, ack := historyDispatchReplayFixture(t, count)
			d, err := o.prepareDispatchCandidate(context.Background(), ack)
			if err != nil || d == nil {
				t.Fatal("two actual complete independent original passes", err)
			}
			before := o.meter.actual
			for role, p := range d.replayReads {
				wantRoot, wantWork, _ := historyReplayReadRootReference(t, p)
				m := providerReplayUnitMeter() // DATA inspection; not a live replay owner.
				got := p.dataRootObserved(m)
				upper := uint64(67117096) + uint64(len(p.rows))*67117095
				if got != wantRoot || got != p.root || m.actual != wantWork || m.failed || m.actual[10] > upper || p.role != uint8(role) ||
					!p.complete || p.context != nil || p.failed || p.meter != o.meter || o.meter.actual != before {
					t.Fatal("rolling original root, weighted actuals or original observer changed", role, got, wantRoot, m.actual, wantWork)
				}
			}
			if d.replayReads[0] == d.replayReads[1] || d.replayReads[0].root == d.replayReads[1].root {
				t.Fatal("role-bound original transcripts were shared")
			}
		})
	}
}

func TestProviderHistoryReadCommitmentLatePreflightKeepsOriginalPrefix(t *testing.T) {
	o, ack := historyDispatchReplayFixture(t, 1)
	d, err := o.prepareDispatchCandidate(context.Background(), ack)
	if err != nil || d == nil {
		t.Fatal("complete genuine original transcript", err)
	}
	p := d.replayReads[0]
	_, _, baseActual := historyReplayReadRootReference(t, p)
	if len(p.rows) == 0 || baseActual == 0 || baseActual >= 67108865 {
		t.Fatal("ordinary original root must fund its first row preflight")
	}
	m := providerReplayUnitMeter()
	m.upper[10] = baseActual + 67108865 - 1
	if root := p.dataRootObserved(m); root != "" || !m.failed || m.actual != (candidateWorkVector{10: baseActual}) {
		t.Fatal("next real row preflight was not refused after the genuine base prefix", root, m.actual, baseActual)
	}
	before := m.actual
	if p.dataRootObserved(m) != "" || m.actual != before || p.failed || !p.complete || p.meter != o.meter {
		t.Fatal("spent DATA observer rewound or changed the retained original pass")
	}
}

func TestProviderHistoryReadRecorderChargesBeforeAllocationAndRejectsClosedObserver(t *testing.T) {
	o, _ := historyDispatchReplayFixture(t, 1)
	o.phase = "preparing-dispatch"
	o.store.activeMeter.Store(o.meter)
	m, before := o.meter, o.meter.actual
	p, err := newProviderHistoryReplayReadsLocked(context.Background(), o, 0)
	want := before
	want[10] += uint64(reflect.TypeFor[providerHistoryReplayReads]().Size()) + 1
	if err != nil || p == nil || p.owner != p || p.group != o || p.meter != m || p.role != 0 || p.maximum != o.parent.bounds.catalogEntries+128 ||
		len(p.rows) != 0 || p.complete || p.failed || m.actual != want || m.replayReads != nil {
		t.Fatal("normal original recorder allocation/observation changed", err, m.actual, want)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if bad, err := newProviderHistoryReplayReadsLocked(ctx, o, 1); err == nil || bad != nil || m.actual != want {
		t.Fatal("cancelled recorder performed an allocation event")
	}
	// A closed observer cannot allocate another recorder. Preserve its real
	// earlier prefix and original allowance; no synthetic encoder fault.
	m.closed = true
	if bad, err := newProviderHistoryReplayReadsLocked(context.Background(), o, 1); err == nil || bad != nil || m.actual != want || !m.closed {
		t.Fatal("closed original observer allocated another recorder")
	}
}

func TestProviderHistoryReadRetentionRejectsInvalidProspectiveDomains(t *testing.T) {
	base := providerHistoryReadRetentionInput{1, 1, 64, 1024, 64 << 20, 65536, 1 << 20, 262144}
	for _, mode := range []string{"groups", "requests", "too-many-requests", "objects", "catalog", "wide-catalog", "raw", "wide-raw", "metadata", "wide-metadata", "object", "wide-object", "wire", "wide-wire", "recording-capacity"} {
		t.Run(mode, func(t *testing.T) {
			in := base
			switch mode {
			case "groups":
				in.groups = 0
			case "requests":
				in.requests = 0
			case "too-many-requests":
				in.requests = 33
			case "objects":
				in.firstObjects = 0
			case "catalog":
				in.catalogEntries = 0
			case "wide-catalog":
				in.catalogEntries = providerHistoryCatalogEntriesMaximum + 1
			case "raw":
				in.firstRaw = 0
			case "wide-raw":
				in.firstRaw = 64<<20 + 1
			case "metadata":
				in.metadata = 0
			case "wide-metadata":
				in.metadata = 1<<20 + 1
			case "object":
				in.object = 0
			case "wide-object":
				in.object = 32<<20 + 1
			case "wire":
				in.wire = 0
			case "wide-wire":
				in.wire = 16<<20 + 1
			case "recording-capacity":
				in.groups, in.requests, in.catalogEntries = 10, 320, 1
			}
			if got, ok := providerHistoryReadRetentionUpper(in); ok || got != (providerHistoryReadRetentionCost{}) {
				t.Fatal("invalid source limits returned partial read-retention work", mode, got, ok)
			}
		})
	}
}

func TestProviderHistoryReadRetentionLateLogicalOverflowReturnsWholeZero(t *testing.T) {
	// All scalar domains and maximum-row capacity are valid. Independent
	// root commitments alone require131435929309500420000 >uint64, while
	// raw-byte totals62805917040640000/2009789345300480000 remain finite.
	in := providerHistoryReadRetentionInput{10000, 320000, 1 << 20, 900000, 64 << 20, 1 << 20, 32 << 20, 16 << 20}
	if got, ok := providerHistoryReadRetentionUpper(in); ok || got != (providerHistoryReadRetentionCost{}) {
		t.Fatal("late rolling-root overflow leaked finite earlier fields or a partial vector", got, ok)
	}
}

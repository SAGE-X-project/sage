package registry011

import "testing"

func TestProviderHistoryReconstructionIndexMatchesOriginalCallLedger(t *testing.T) {
	got := providerHistoryReconstructionIndexUpper()
	if [5]uint64{got.preparations, got.generated, got.checked, got.segmentKeys, got.valueComparisons} != [5]uint64{16, 36, 18, 24, 90} ||
		got.normal != (candidateWorkVector{2: 958464, 10: 37787799836}) ||
		got.incoming != (candidateWorkVector{2: 65536, 10: 4294967297}) ||
		got.envelope != (candidateWorkVector{2: 1024000, 10: 42082767133}) {
		t.Fatal("two original index prefixes lost preparations, repeated nodes, comparisons or original guards", got)
	}
	// Independently enumerated constructor /prepare /latest-node /segment /
	// mutation /wire-check /value-comparison rows. No measured claim follows.
	want := [7][2]uint64{{24576, 201375942}, {131072, 1076233312}, {360448, 17985536032},
		{0, 402752304}, {294912, 4832429328}, {147456, 1208402568}, {0, 12081070350}}
	for i, part := range [...]candidateWorkVector{got.anchors, got.preparation, got.ledgerNodes, got.segments, got.mutations, got.validation, got.values} {
		if part != (candidateWorkVector{2: want[i][0], 10: want[i][1]}) {
			t.Fatal("original initial index component", i, part, want[i])
		}
	}
	if !providerHistoryReadWorkDomain(got.envelope) {
		t.Fatal("initial index DATA acquired another read/execution resource")
	}
}

func TestProviderHistoryReconstructionIndexFundsFreshTypedPrefixes(t *testing.T) {
	for _, outcome := range []string{"complete", "failed"} {
		t.Run(outcome, func(t *testing.T) {
			quote := providerHistoryReconstructionIndexUpper()
			// Separate candidate arithmetic UNIT only, with no live read/store
			// role. The same inert meter remains installed for both prefixes.
			// Ledger transitions are observed too, but outside index snapshots;
			// their work is deliberately NOT certified by the index-only row.
			meter := providerHistoryContentUnitMeter(candidateWorkVector{2: 64 << 20, 10: 1 << 50})
			var measured candidateWorkVector
			var firstIndexes *providerLedgerIndexes
			var firstLedger *CandidateAdmissionLedger
			for pass := range 2 {
				m, _ := providerPrivateLedgerFixture(t, 8, 32)
				if m == nil || m.meter != nil || m.continuation == nil || m.continuation.auxiliaryMetadata != 0 {
					t.Fatal("fresh unbound candidate ledger precursor")
				}
				id := m.intent.identity
				v := admissionTestRootObject(t)
				v["policy"] = id.Policy
				root, err := ParseCandidateEnrollmentRoot(&ProviderCandidateContext{identity: id, bound: true}, providerPolicyTestBytes(t, v), 64<<10)
				if err != nil || root == nil || root.digest != m.intent.anchor {
					t.Fatal("matching initial candidate root", err)
				}
				m.meter = meter // This fresh DATA ledger has never owned a live role.
				before := meter.actual
				s, anchors, err := newProviderLedgerIndexesObserved(id, m.intent.fence, m.continuation.bounds, meter)
				if err != nil || s == nil || len(anchors) != 3 {
					t.Fatal("fresh observed candidate index constructor", err)
				}
				if s.meter != nil || m.meter != meter || meter.owner != meter || meter.failed || meter.closed ||
					meter.authority || meter.clocks != 0 || meter.integration == nil || meter.integration.store != nil ||
					meter.scan != nil || meter.scanBody != nil || meter.firstAdmission != nil || meter.firstOperation != nil ||
					meter.guardian != nil || meter.continuing != nil || meter.historyGroup != nil || meter.historyInputGroup != nil ||
					meter.historyRecovery != nil || meter.historyFinal != nil || meter.replayReads != nil || meter.parent != nil {
					t.Fatal("fresh candidate index and ledger must share an inert observer without a live role")
				}
				s.meter = meter // Connect this new DATA index once, preserving constructor observations.
				var decoded uint64
				for _, anchor := range anchors {
					decoded += uint64(len(anchor.raw))
				}
				if meter.actual[2] < before[2] || meter.actual[2]-before[2] != decoded || decoded == 0 || meter.actual[10] <= before[10] {
					t.Fatal("actual constructor codecs did not match its three original anchor values")
				}
				measured[2] += decoded
				measured[10] += meter.actual[10] - before[10]
				apply := func(kind, entryKind string, raw []byte, nodes int) {
					if kind != "enroll" {
						root = providerIndexRoot(t, root, m)
					}
					entry, err := newProviderIntegrationObject(entryKind, raw, 64<<10)
					if err != nil {
						t.Fatal("bounded entry precursor outside index interval", err)
					}
					operation, ledger := entry.digest, m
					if kind == "enroll" {
						operation, ledger = admissionTestDigest("a"), nil
					}
					var latest uint64
					if ledger != nil {
						if len(m.current.nodes) == 0 {
							t.Fatal("actual candidate ledger node precursor")
						}
						latest = uint64(len(m.current.nodes[len(m.current.nodes)-1]))
					}
					start := meter.actual
					delta, err := s.prepare(kind, operation, uint64(len(s.operations))+1, entry, root, ledger)
					if err != nil || delta == nil || len(delta.mutations) != nodes {
						t.Fatal("original typed index preparation", kind, err)
					}
					wantDecoded := 2*uint64(len(root.raw)) + 2*latest
					if kind == "settle" {
						wantDecoded += 2 * uint64(len(entry.raw))
					}
					for _, node := range delta.mutations {
						wantDecoded += 3 * uint64(len(node.object.raw))
					}
					if err := s.apply(delta); err != nil || !delta.used || meter.actual[2] < start[2] ||
						meter.actual[2]-start[2] != wantDecoded || meter.actual[10] <= start[10] {
						t.Fatal("apply skipped its independent preparation or exact node decoding", kind, err)
					}
					measured[2] += wantDecoded
					measured[10] += meter.actual[10] - start[10]
				}
				apply("enroll", "enrollment", []byte(root.raw), 1)
				if err := s.bindLedger(m); err != nil {
					t.Fatal("fresh candidate ledger/index binding", err)
				}
				first := m.lastTicket
				apply("admit", "ticket", []byte(first.raw), 3)
				consumed := ledgerTestConsumption(t, first, nil, 10)
				if _, err := m.AppendCandidateConsumption(consumed); err != nil {
					t.Fatal("candidate consumption outside index interval", err)
				}
				apply("consume", "consumption", []byte(consumed.raw), 2)
				settled := ledgerTestSettlement(t, first, outcome, 20)
				if _, err := m.AppendCandidateSettlement(settled); err != nil {
					t.Fatal("candidate settlement outside index interval", err)
				}
				apply("settle", "settlement", []byte(settled.raw), 3)
				key := candidateAdmissionSegment(m.intent.digest, first.segment)
				if s.counts != [3]uint64{3, 4, 2} || s.tickets[first.digest].consumption != consumed.digest ||
					s.tickets[first.digest].settlement != settled.digest || s.segments[key].totals.charged != first.work ||
					s.segments[key].totals.outstanding != (candidateWorkVector{}) || s == firstIndexes || m == firstLedger {
					t.Fatal("fresh typed index prefix lost its terminal state or shared an earlier model", pass)
				}
				if pass == 0 {
					firstIndexes, firstLedger = s, m
				}
			}
			if measured[2] == 0 || measured[2] > quote.normal[2] || measured[10] > quote.normal[10] || meter.failed ||
				meter.authority || meter.clocks != 0 || meter.historyRecovery != nil || meter.historyFinal != nil || meter.historyGroup != nil ||
				meter.actual[2] < measured[2] || meter.actual[10] < measured[10] {
				t.Fatal("index component exceeded its quote, borrowed other work or acquired a live role", measured)
			}
		})
	}
}

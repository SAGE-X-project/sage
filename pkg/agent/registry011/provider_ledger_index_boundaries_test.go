package registry011

import "testing"

// All modified values in this file are inert index/ledger unit data. They are
// never written as a live frontier or submitted to an execution/transport path.
func TestProviderLedgerIndexConstructionAndBindingAreFunded(t *testing.T) {
	m, _ := providerPrivateLedgerFixture(t, 8, 32)
	b := m.continuation.bounds
	if _, _, err := newProviderLedgerIndexes(m.intent.identity, 0, b); err == nil {
		t.Fatal("zero fence constructed indexes")
	}
	b.maximumResidentBytes = 1
	if _, _, err := newProviderLedgerIndexes(m.intent.identity, 1, b); err == nil {
		t.Fatal("initial codec/maps ignored resident capacity")
	}
	for _, mode := range []string{"metadata-overflow", "payload-cap"} {
		t.Run(mode, func(t *testing.T) {
			u := providerFrontierUnitFixture(t)
			ledger, _ := providerPrivateLedgerFixture(t, 8, 32)
			prior := providerFrontierRetention{metadata: ^uint64(0)}
			if mode == "payload-cap" {
				prior = providerFrontierRetention{payload: ^uint64(0)}
			}
			before := ledger.current
			if u.indexes.bindRetainedLedger(ledger, prior) == nil || u.indexes.ledger != nil || ledger.current != before {
				t.Fatal("failed initial debit partially bound indexes")
			}
		})
	}
}

func TestProviderLedgerIndexPreparationRejectsInvalidTypedStateBeforeMutation(t *testing.T) {
	for _, mode := range []string{"root", "operation", "empty-entry", "entry-hash", "root-hash", "metadata", "resident", "missing-ledger", "pending", "missing-seal", "wrong-receipt", "missing-consumption", "missing-ticket", "missing-node", "wrong-node", "invalid-ticket", "segment-totals"} {
		t.Run(mode, func(t *testing.T) {
			s, m, root := providerIndexFixture(t)
			entry, err := newProviderIntegrationObject("ticket", m.lastTicket.CanonicalBytes(), 64<<10)
			if err != nil {
				t.Fatal(err)
			}
			next := providerIndexRoot(t, root, m)
			kind, operation := "admit", entry.digest
			switch mode {
			case "root":
				next = nil
			case "operation":
				operation = admissionTestDigest("f")
			case "empty-entry":
				entry.raw = ""
			case "entry-hash":
				entry.digest, operation = admissionTestDigest("f"), admissionTestDigest("f")
			case "root-hash":
				next.digest = admissionTestDigest("f")
			case "metadata":
				s.bounds.maximumMetadataBytes, s.bounds.pageMetadataBytes = s.schemaBytes, s.schemaBytes
			case "resident":
				s.bounds.maximumResidentBytes = 1
			case "missing-ledger":
				s.ledger = nil
			case "pending":
				m.continuation.pending = &providerLedgerBoundary{}
			case "missing-seal":
				kind = "seal"
			case "wrong-receipt":
				kind = "verify"
			case "missing-consumption":
				kind = "consume"
			case "missing-ticket":
				delete(m.tickets, m.lastTicket.digest)
			case "missing-node":
				m.current.nodes = nil
			case "wrong-node":
				m.current.nodes = []string{`{}`}
			case "invalid-ticket":
				state := m.tickets[m.lastTicket.digest]
				state.ticket = nil
				m.tickets[m.lastTicket.digest] = state
			case "segment-totals":
				m.segments[0] = candidateBudgetTotals{}
			}
			heads, counts, before := s.heads, s.counts, m.current
			if _, err := s.prepare(kind, operation, 2, entry, next, m); err == nil {
				t.Fatal("invalid typed state produced a prepared delta", mode)
			}
			if s.heads != heads || s.counts != counts || m.current != before || len(s.tickets) != 0 || len(s.operations) != 1 {
				t.Fatal("refused preparation partially mutated indexes")
			}
		})
	}
}

func TestProviderLedgerIndexMutationChecksTypedValuesAndExpectedOldEntry(t *testing.T) {
	for _, mode := range []string{"shape", "hash", "ticket-value", "ticket-key", "operation-value", "duplicate-operation", "segment-value", "segment-key", "old-value"} {
		t.Run(mode, func(t *testing.T) {
			s, m, root := providerIndexFixture(t)
			entry, err := newProviderIntegrationObject("ticket", m.lastTicket.CanonicalBytes(), 64<<10)
			if err != nil {
				t.Fatal(err)
			}
			d, err := s.prepare("admit", entry.digest, 2, entry, providerIndexRoot(t, root, m), m)
			if err != nil {
				t.Fatal(err)
			}
			node := d.mutations[0]
			if mode == "operation-value" || mode == "duplicate-operation" {
				node = d.mutations[2]
			}
			if mode == "segment-value" || mode == "segment-key" {
				node = d.mutations[1]
			}
			if s.checkMutation(node, m.intent.digest) != nil {
				t.Fatal("normal typed mutation data")
			}
			switch mode {
			case "shape":
				node.object.raw = `{}`
			case "hash":
				node.object.digest = admissionTestDigest("f")
			case "ticket-value", "operation-value", "segment-value":
				node.value = map[string]any{}
			case "ticket-key", "segment-key":
				node.key = admissionTestDigest("f")
			case "duplicate-operation":
				for key := range s.operations {
					node.key = key
				}
				node.value = providerOperationIndexValue{node.key, "admit", 2}.object()
			case "old-value":
				node.expected = map[string]string{"unexpected": "prior"}
			}
			if mode != "shape" && mode != "hash" {
				// Re-encode only finite data to reach the typed-value/old-entry
				// comparison; matching hashes alone must not bypass those rules.
				value := contentJSON([]byte(node.object.raw)).(map[string]any)
				value["key"], value["expected"], value["value"] = node.key, node.expected, node.value
				node.object, err = newProviderIntegrationObject("index-mutation", providerIntegrationJSON(value), candidateLedgerMetadataLimit)
				if err != nil {
					t.Fatal(err)
				}
			}
			if s.checkMutation(node, m.intent.digest) == nil {
				t.Fatal("hash/shape replaced typed old-state validation", mode)
			}
		})
	}
}

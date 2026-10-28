package registry011

import "testing"

// These are finite private data comparisons. No corrupted state is retained,
// dispatched, serialized as authority or presented as a hostile runtime peer.
func TestProviderFirstOperationModelsKeepIndependentPrivateAccounting(t *testing.T) {
	for _, mode := range []string{"same", "missing-store", "missing-proposal", "aliased-index", "aliased-ledger", "missing-ticket", "current-digest", "policy-original", "node-original", "missing-consumption", "consumption-original", "ticket-index", "operation-index", "segment-index", "ticket-state", "segment-state", "continuation-profile", "continuation-record"} {
		t.Run(mode, func(t *testing.T) {
			s, o := providerFirstOperationFixture(t)
			s.mu.Lock()
			defer s.mu.Unlock()
			proposed, retained := o.indexes, s.frontierReplay.indexes
			a, b := proposed.ledger, retained.ledger
			selected := s
			switch mode {
			case "missing-store":
				selected = nil
			case "missing-proposal":
				proposed = nil
			case "aliased-index":
				proposed = retained
			case "aliased-ledger":
				proposed.ledger = b
			case "missing-ticket":
				a.lastTicket = nil
			case "current-digest":
				a.current.digest = admissionTestDigest("f")
			case "policy-original":
				a.policy += " "
			case "node-original":
				a.current.nodes[0] += " "
			case "missing-consumption":
				a.lastConsumption = &CandidateConsumption{}
			case "consumption-original":
				a.lastConsumption = &CandidateConsumption{digest: admissionTestDigest("a"), raw: "a"}
				b.lastConsumption = &CandidateConsumption{digest: admissionTestDigest("a"), raw: "b"}
			case "ticket-index":
				for key, value := range proposed.tickets {
					value.input = admissionTestDigest("f")
					proposed.tickets[key] = value
				}
			case "operation-index":
				for key, value := range proposed.operations {
					value.kind = "settle"
					proposed.operations[key] = value
					break
				}
			case "segment-index":
				for key, value := range proposed.segments {
					value.totals.charged[10]++
					proposed.segments[key] = value
				}
			case "ticket-state":
				for key, value := range a.tickets {
					value.terminal = admissionTestDigest("f")
					a.tickets[key] = value
				}
			case "segment-state":
				for key, value := range a.segments {
					value.charged[10]++
					a.segments[key] = value
				}
			case "continuation-profile":
				a.continuation.bounds.maximumResidentBytes--
			case "continuation-record":
				a.continuation.records[0].entry = admissionTestDigest("f")
			}
			matched := providerFirstOperationModelsEqual(selected, proposed, retained, o.execution)
			if matched != (mode == "same") || s.sequence != 2 || s.last != nil || o.execution.actual[0] != 0 || o.execution.actual[9] != 0 {
				t.Fatal("private model mismatch was accepted or comparison dispatched/persisted", mode)
			}
		})
	}
}

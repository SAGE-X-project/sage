package registry011

import (
	"context"
	"encoding/json"
	"testing"
)

// Ordinary owned TLS exchanges only. Each child finishes its original T/C/S
// before this helper constructs the next request; there is no parent permit.
func providerContinuingPinReadyFixture(t *testing.T) (*providerCustodyStore, *providerGuardianOwner, [4]string) {
	t.Helper()
	s, g := providerContinuingRPCReadyWithSelectionFixture(t, &providerScanFixtureSelection{pin: true})
	selected := admissionTestDigest("a")
	requests, err := providerPinRequests(selected)
	if err != nil {
		t.Fatal(err)
	}
	results := providerPinResultFixture(t)
	var digests [4]string
	for i, request := range requests {
		if i == 2 {
			if _, err := sealProviderContinuingPageCandidate(s); err != nil {
				t.Fatal("second original page", err)
			}
		}
		var params any
		if json.Unmarshal([]byte(request.params), &params) != nil {
			t.Fatal("normal fixed request params")
		}
		o, err := newProviderContinuingRPCCandidate(s, request.method, params)
		if err != nil {
			t.Fatalf("pin child %d admission: %v G=%+v", i, err, g.meter)
		}
		connection := providerIntegrationOwnedTLS(t, o.request.body, `{"jsonrpc":"2.0","id":1,"result":`+string(results[i])+`}`)
		data, err := o.executeRPCCandidateWithConnection(context.Background(), connection)
		if err != nil || data == nil {
			t.Fatalf("pin child %d: %v E=%+v R=%+v", i, err, o.execution, o.remaining)
		}
		digests[i] = data.exchange
	}
	if _, err := sealProviderContinuingPageCandidate(s); err != nil {
		t.Fatal("third original page", err)
	}
	if s.sequence != 22 || g.ordinal != 8 || g.integration.ledger.continuation.page != 3 || len(g.integration.ledger.continuation.records) != 0 {
		t.Fatal("four closed child/page boundaries")
	}
	return s, g, digests
}

func TestProviderContinuingPinRevalidatesFourOriginalClosuresWithDistinctWork(t *testing.T) {
	t.Parallel()
	s, g, digests := providerContinuingPinReadyFixture(t)
	original, initial, bootstrap := s.setupOrigin, g.integration.firstExchange, g.integration.bootstrap
	oldE, oldR := g.first.execution.actual, g.first.remaining.actual
	before, ordinal := s.stored, g.ordinal
	o, err := newProviderContinuingPinCandidate(s, "example-agent", digests)
	if err != nil {
		t.Fatalf("new local pin T: %v G=%+v", err, g.meter)
	}
	if o.ticket.kind != "local-work" || o.request != nil || o.pin == nil || o.pin.parent != o || o.pin.exchanges != digests || o.ticket.input != o.pin.input.digest || o.state != providerContinuingPrepared || g.ordinal != ordinal+1 || s.stored != before+o.plan.physical {
		t.Fatal("local admission borrowed a child permit or lost its own hold")
	}
	data, err := o.executePinCandidate(context.Background())
	if err != nil || data == nil {
		t.Fatalf("independent pin C/S: %v state=%d E=%+v R=%+v", err, o.state, o.execution, o.remaining)
	}
	if data.exchanges != digests || data.selected != admissionTestDigest("a") || data.pin != g.integration.pin || data.bootstrap != bootstrap || data.ticket != o.ticket.digest || data.input != o.ticket.input || data.highwater != s.frontierFiles.pin || o.state != providerContinuingDone || s.sequence != 25 || g.ordinal != 9 || !g.ready || g.active || g.continuing != nil || s.activeMeter.Load() != nil {
		t.Fatal("pin escaped before independent retained S or changed its exact inputs")
	}
	if s.setupOrigin != original || g.original != original || g.integration.firstExchange != initial || g.integration.bootstrap != bootstrap || g.first.execution.actual != oldE || g.first.remaining.actual != oldR || s.stored != before+o.plan.physical || data.execution[0] != 0 || data.execution[1] != 0 || data.execution[3] != 1 || data.remaining[0] != 0 || data.remaining[1] != 0 || data.remaining[3] != 0 || o.clocks > 11 || !o.execution.closed || !o.remaining.closed || !o.permit.used || o.permit.ack != nil {
		t.Fatal("pin reset original lifetime, reused child work, invented network work or refunded storage")
	}
	for i, v := range data.execution {
		if v > o.plan.reservation.execution[i] || data.remaining[i] > o.plan.reservation.remaining[i] {
			t.Fatal("observed cost exceeded its separate prospective hold", i)
		}
	}
	stored := s.stored
	if value, err := o.executePinCandidate(context.Background()); err == nil || value != nil || s.failed || s.stored != stored {
		t.Fatal("completed private command reused or refunded")
	}
	if value, err := newProviderContinuingPinCandidate(s, "example-agent", digests); err == nil || value != nil || s.failed || s.stored != stored {
		t.Fatal("already pinned lifetime admitted another command")
	}
}

func TestProviderContinuingPinRejectsMissingAndDuplicateInputWithoutAuthority(t *testing.T) {
	t.Parallel()
	for _, value := range []any{&providerContinuingPinData{}, &providerContinuingPinCommand{}} {
		if json.Unmarshal([]byte(`{}`), value) == nil {
			t.Fatal("JSON restored a private pin role")
		}
	}
	digests := [4]string{admissionTestDigest("a"), admissionTestDigest("b"), admissionTestDigest("c"), admissionTestDigest("d")}
	for _, s := range []*providerCustodyStore{nil, {}} {
		if value, err := newProviderContinuingPinCandidate(s, "example-agent", digests); err == nil || value != nil {
			t.Fatal("missing original store admitted pin")
		}
	}
	if value, err := (*providerContinuingOwner)(nil).executePinCandidate(context.Background()); err == nil || value != nil {
		t.Fatal("missing role executed pin")
	}
	if _, ok := providerContinuingPinPlan(nil, nil); ok {
		t.Fatal("missing guardian produced pin reservation")
	}
	if _, ok := providerContinuingPinMemoryUpper(nil, nil, providerFirstOperationStorage{}, 0, 0, [4]uint64{}); ok {
		t.Fatal("missing guardian produced resident quote")
	}
	if _, _, err := providerContinuingPinSelected(nil, nil, nil); err == nil {
		t.Fatal("missing original bootstrap selected a block")
	}
	if _, err := (*providerCustodyReplay)(nil).continuingPinResults(nil, digests, admissionTestDigest("a")); err == nil {
		t.Fatal("unowned replay produced pin results")
	}
	if _, err := (*providerCustodyReplay)(nil).continuingPinSettlement(nil, digests[0], nil); err == nil {
		t.Fatal("unowned replay selected a settlement")
	}
	if _, err := providerContinuingPinObject(digests[0], digests[1], digests, nil); err == nil {
		t.Fatal("unobserved pin encoding succeeded")
	}
	s, _ := providerContinuingRPCReadyFixture(t)
	stored, tip := s.stored, s.frontierFiles.pin
	for _, mode := range []string{"empty-agent", "bad-hash", "duplicate"} {
		agent, selected := "example-agent", digests
		switch mode {
		case "empty-agent":
			agent = ""
		case "bad-hash":
			selected[0] = "bad"
		case "duplicate":
			selected[3] = selected[0]
		}
		if value, err := newProviderContinuingPinCandidate(s, agent, selected); err == nil || value != nil || s.failed || s.stored != stored || s.frontierFiles.pin != tip {
			t.Fatal("invalid lookup DATA started an original claim", mode)
		}
	}
}

func TestProviderContinuingPinUnretainedChildrenLeaveOnlyFailedSettlement(t *testing.T) {
	t.Parallel()
	s, g := providerContinuingRPCReadyWithSelectionFixture(t, &providerScanFixtureSelection{pin: true})
	// Finite nonexistent lookup hints are never written as original X/S and
	// never sent over a connection. C must still be irreversibly consumed.
	digests := [4]string{admissionTestDigest("a"), admissionTestDigest("b"), admissionTestDigest("c"), admissionTestDigest("d")}
	o, err := newProviderContinuingPinCandidate(s, "example-agent", digests)
	if err != nil {
		t.Fatalf("separately funded local admission: %v G=%+v", err, g.meter)
	}
	held := s.stored
	data, err := o.executePinCandidate(context.Background())
	if err == nil || data != nil || !s.failed || !g.failed || g.ready || g.integration.pin != "" || !o.consumption || !o.settlement || o.permit == nil || !o.permit.used || o.permit.ack != nil || s.sequence != 11 || s.stored != held || o.execution.actual[0] != 0 || o.execution.actual[1] != 0 {
		t.Fatalf("incomplete child replay returned usable pin DATA or refunded a hold: state=%d sequence=%d E=%+v R=%+v", o.state, s.sequence, o.execution, o.remaining)
	}
	entry := s.frontierReplay.body.ledger.tickets[o.ticket.digest]
	raw, readErr := s.evidence.read(providerObjectName("settlement", entry.terminal), candidateLedgerMetadataLimit)
	value := contentObject(contentJSON(raw), "format", "ticket_digest", "outcome", "actual_work", "retained_root", "clock_end")
	if readErr != nil || value == nil || value["outcome"] != "failed" || value["actual_work"] != nil || value["retained_root"] != nil {
		t.Fatal("failed local command invented a measured successful pin")
	}
	if data, err := o.executePinCandidate(context.Background()); err == nil || data != nil || s.stored != held || s.sequence != 11 {
		t.Fatal("failed command restarted or refunded")
	}
}

func TestProviderContinuingPinObjectKeepsExactBoundedData(t *testing.T) {
	t.Parallel()
	s, policy, candidate, primitive := providerStoreFixture(t)
	c, err := newProviderIntegration(s, policy, candidate, primitive)
	if err != nil {
		t.Fatal(err)
	}
	sequence, retained := s.sequence, s.stored
	digests := [4]string{admissionTestDigest("a"), admissionTestDigest("b"), admissionTestDigest("c"), admissionTestDigest("d")}
	for _, mode := range []string{"normal", "input", "selected", "digest", "duplicate", "closed", "failed", "encode-budget"} {
		t.Run(mode, func(t *testing.T) {
			input, selected, children := admissionTestDigest("e"), admissionTestDigest("f"), digests
			// This unattached unit observer encodes DATA only. It cannot create a
			// private G/T/C/S, retain originals, dispatch or settle real work.
			meter := newProviderWorkMeter(c, providerOperationReservation{execution: candidateWorkVector{2: 64 << 10, 10: 1 << 40}})
			switch mode {
			case "input":
				input = "invalid"
			case "selected":
				selected = "invalid"
			case "digest":
				children[3] = "invalid"
			case "duplicate":
				children[3] = children[1]
			case "closed":
				meter.closed = true
			case "failed":
				meter.failed = true
			case "encode-budget":
				meter.upper[10] = 1
			}
			object, err := providerContinuingPinObject(input, selected, children, meter)
			if mode == "normal" {
				m := contentObject(contentJSON([]byte(object.raw)), "format", "input", "selected_block_hash", "exchanges")
				if err != nil || object.kind != "pin" || m == nil || m["input"] != input || m["selected_block_hash"] != selected || len(object.raw) > 4096 || !lowerHash(object.digest) || meter.actual[0] != 0 || meter.actual[1] != 0 || s.sequence != sequence || s.stored != retained {
					t.Fatal("bounded unit pin DATA changed or created authority", err)
				}
			} else if err == nil || object != (providerIntegrationObject{}) {
				t.Fatal("incomplete DATA was encoded", mode)
			}
		})
	}
}

package registry011

import "testing"

// Inert candidate-ledger DATA. No store, G, TLS peer or execution permit.
func TestProviderContinuingRPCDoesNotRefundTheOriginalWireBudget(t *testing.T) {
	t.Parallel()
	fixture, policy := providerPrivateLedgerFixture(t, 12, 12)
	raw := contentJSON(fixture.lastTicket.CanonicalBytes()).(map[string]any)
	raw["reservation"].(map[string]any)["wire_bytes"] = "32768"
	first := admissionTestTicket(t, policy, fixture.intent, 0, raw)
	ledger, err := newProviderContinuingLedger(policy, fixture.intent, first, fixture.continuation.bounds)
	if err != nil {
		t.Fatal(err)
	}
	ticket := first
	for i := 0; i < 3; i++ {
		if i != 0 {
			next := ledgerTestNext(t, policy, ticket, 0)
			raw := contentJSON(next.CanonicalBytes()).(map[string]any)
			raw["reservation"].(map[string]any)["wire_bytes"] = "16384"
			ticket = admissionTestTicket(t, policy, fixture.intent, 0, raw)
			if _, err := ledger.AppendCandidateTicket(ticket); err != nil {
				t.Fatal("declared complete-lifetime wire budget", i, err)
			}
		}
		consumption := ledgerTestConsumption(t, ticket, ledger.lastConsumption, int64(10+i*20))
		if _, err := ledger.AppendCandidateConsumption(consumption); err != nil {
			t.Fatal(err)
		}
		object := ledgerTestSettlementObject(ticket, "complete", int64(20+i*20))
		object["actual_work"].(map[string]any)["wire_bytes"] = "1"
		settlement, err := ParseCandidateAdmissionSettlement(ticket, providerPolicyTestBytes(t, object), 64<<10)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ledger.AppendCandidateSettlement(settlement); err != nil {
			t.Fatal(err)
		}
	}
	before := ledger.current
	next := ledgerTestNext(t, policy, ticket, 0)
	if before.totals.bootstrap.charged[1] != 65536 || before.totals.bootstrap.outstanding[1] != 0 || next.work[1] != 16384 {
		t.Fatal("completed small responses refunded their full reservations")
	}
	if _, err := ledger.AppendCandidateTicket(next); err == nil || ledger.current != before || ledger.lastTicket != ticket {
		t.Fatal("an additional scalar reservation spent the original wire budget twice")
	}
}

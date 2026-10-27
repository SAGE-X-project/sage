package registry011

import "testing"

// These units start with a genuine retained, consumed test ticket but never
// dial. Altered permit/result values below are finite inert invalid inputs;
// they must produce neither a completed settlement nor any work credential.
func TestProviderLocalResultRefusalsPreserveConsumedReservation(t *testing.T) {
	for _, mode := range []string{"unknown-kind", "work-limit", "bad-object", "clock-unavailable", "wrong-meter-ticket", "wrong-result-role"} {
		t.Run(mode, func(t *testing.T) {
			c, s := providerPendingIntegrationUnit(t)
			permit, err := c.consume(c.firstTicket, c.first)
			if err != nil || !c.use(permit) {
				t.Fatal("normal consumed unit ticket", err)
			}
			// Removing the request is not a local-work conversion. In particular
			// the final settlement still enforces the original RPC ticket role.
			permit.request = nil
			kind := "pin"
			raw := []byte(`{"format":"sage-provider-owned-pin-v1"}`)
			switch mode {
			case "unknown-kind":
				kind = "unknown"
			case "work-limit":
				permit.meter.upper[10] = 0
			case "bad-object":
				raw = []byte(`{}`)
			case "clock-unavailable":
				c.clock.maximum = c.clock.read
			case "wrong-meter-ticket":
				permit.meter.ticket = admissionTestDigest("f")
			}
			before, sequence := c.ledger.current, s.sequence
			if result, err := c.finishLocalResult(permit, kind, raw); err == nil || result != "" || c.ledger.current != before || s.sequence != sequence || c.ledger.current.totals.bootstrap.outstanding != c.firstTicket.work || c.ledger.current.counts[2] != 0 {
				t.Fatal("invalid local result completed/refunded the original consumed ticket")
			}
			if mode != "unknown-kind" && !c.failed {
				t.Fatal("unknown completion left its coordinator reusable")
			}
		})
	}
}

func TestProviderAbortRetainsFullChargeOrOutstandingWorkWithoutDispatch(t *testing.T) {
	for _, mode := range []string{"normal-abort", "unavailable-clock", "absent-permit"} {
		t.Run(mode, func(t *testing.T) {
			c, _ := providerPendingIntegrationUnit(t)
			permit, err := c.consume(c.firstTicket, c.first)
			if err != nil || !c.use(permit) {
				t.Fatal("normal consumed unit ticket", err)
			}
			switch mode {
			case "unavailable-clock":
				c.clock.maximum = c.clock.read
			case "absent-permit":
				permit = nil
			}
			c.failLocal(permit)
			if !c.failed || c.root.status != "enrolling" || c.root.checkpoint != "" {
				t.Fatal("aborted work left an active coordinator or checkpoint")
			}
			totals := c.ledger.current.totals.bootstrap
			if mode == "normal-abort" {
				if totals.charged != c.firstTicket.work || totals.outstanding != (candidateWorkVector{}) || c.ledger.current.counts[2] != 1 {
					t.Fatal("abort did not retain the whole original charge")
				}
			} else if totals.outstanding != c.firstTicket.work || totals.charged != (candidateWorkVector{}) || c.ledger.current.counts[2] != 0 {
				t.Fatal("unknown abort refunded or fabricated terminal work")
			}
		})
	}
}

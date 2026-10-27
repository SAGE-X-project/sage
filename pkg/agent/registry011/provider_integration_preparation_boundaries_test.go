package registry011

import "testing"

func TestProviderIntegrationInitialObjectsMustFitBeforeEnrollment(t *testing.T) {
	s, policy, candidate, primitive := providerStoreFixture(t)
	s.config.bounds.maximumObjectBytes = 1
	if c, err := newProviderIntegration(s, policy, candidate, primitive); err == nil || c != nil || s.sequence != 0 || s.enrollment != "" {
		t.Fatal("unfunded initial closure installed enrollment or admission")
	}
}

func TestProviderIntegrationPreparationRefusesUnfundedPrimitiveAndEncoder(t *testing.T) {
	for _, mode := range []string{"primitive", "estimate", "root", "node", "metadata", "object"} {
		t.Run(mode, func(t *testing.T) {
			c, s := providerPendingIntegrationUnit(t)
			before, tip, sequence := c.root, s.tip, s.sequence
			var err error
			switch mode {
			case "primitive":
				bounds, capture := providerFrontierBoundsFixture()
				s.config.bounds, c.primitive.capture = bounds, capture
				// Only an inert invalid selector input is evaluated, no admission.
				_, err = c.admitRPC(&providerIntegrationRequest{method: "unknown"}, false)
			case "estimate":
				c.primitive.capture.Timeout = 0
				_, err = c.admitRPC(c.first, false)
			default:
				wide := candidateWorkVector{2: 1 << 20, 10: 1 << 60}
				meter := newProviderWorkMeter(c, providerOperationReservation{execution: wide})
				s.activeMeter.Store(meter)
				defer s.activeMeter.Store(nil)
				var extra []providerIntegrationObjectInput
				switch mode {
				case "root":
					copy := *c.root
					m := contentJSON([]byte(copy.raw)).(map[string]any)
					m["namespace"] = admissionTestDigest("f")
					copy.raw = string(providerIntegrationJSON(m))
					c.root = &copy
				case "node":
					meter.upper[2] = 0
				case "metadata":
					var nodeBytes uint64
					for _, raw := range c.ledger.current.LedgerNodeBytes() {
						nodeBytes += uint64(len(raw))
					}
					meter.upper[2] = nodeBytes
				case "object":
					extra = []providerIntegrationObjectInput{{"unsupported", []byte("{}")}}
				}
				_, err = c.commitChange("admit", c.firstTicket.digest, c.ledger.current, extra)
				c.root = before
			}
			if err == nil || s.sequence != sequence || s.tip != tip || c.root != before || c.ledger.lastConsumption != nil {
				t.Fatal("failed preparation installed a new retained state", mode)
			}
		})
	}
}

func TestProviderRequestNumbersConvertsOnlyTheClosedZeroTracerOption(t *testing.T) {
	input := map[string]any{"reexec": float64(0), "nested": []any{float64(0), float64(1), "0"}}
	got := providerRequestNumbers(input).(map[string]any)
	items := got["nested"].([]any)
	if zero, ok := got["reexec"].(int); !ok || zero != 0 {
		t.Fatal("closed numeric zero did not normalize")
	}
	if zero, ok := items[0].(int); !ok || zero != 0 {
		t.Fatal("nested numeric zero did not normalize")
	}
	if value, ok := items[1].(float64); !ok || value != 1 || items[2] != "0" {
		t.Fatal("unsupported numeric or string value was broadened")
	}
}

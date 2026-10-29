package registry011

import "testing"

func TestProviderBootstrapRefusesUnfundedFinalizationWithoutInstallingStart(t *testing.T) {
	for _, mode := range []string{"block", "clock", "measurement", "remaining"} {
		t.Run(mode, func(t *testing.T) {
			c, s := providerRetainedFirstExchangeFixture(t)
			selected, err := providerDecodeHeader([]byte(c.firstResult))
			if err != nil {
				t.Fatal(err)
			}
			permit, err := c.beginLocal(c.firstExchange)
			if err != nil {
				t.Fatal(err)
			}
			defer c.restoreMeter(permit.meter)
			start, raw, err := c.bootstrapObject(permit.meter, selected)
			if err != nil {
				t.Fatal(err)
			}
			before, sequence := c.ledger.current, s.sequence
			switch mode {
			case "block":
				permit.meter.upper[3] = 0
			case "clock":
				s.clock.failed = true
			case "measurement":
				ticket := *permit.ticket
				ticket.deadline = 0 // Inert expired measurement input, not installed.
				permit.ticket = &ticket
			case "remaining":
				permit.used = false // Inert refusal input; never used to dispatch.
			}
			if err := c.finishBootstrap(permit, selected, start, raw); err == nil || !c.failed || c.bootstrap != "" || c.ledger.providerStart != "" || s.sequence != sequence || c.ledger.current != before {
				t.Fatal("incomplete original work installed or refunded a bootstrap", mode)
			}
			if c.ledger.current.totals.bootstrap.outstanding == (candidateWorkVector{}) {
				t.Fatal("consumed local reservation disappeared")
			}
		})
	}
}

func TestProviderBootstrapLocalPreparationCannotBorrowUnsupportedAllowances(t *testing.T) {
	for _, mode := range []string{"input", "primitive"} {
		t.Run(mode, func(t *testing.T) {
			c, s := providerRetainedFirstExchangeFixture(t)
			input := c.firstExchange
			if mode == "input" {
				input = admissionTestDigest("f")
			} else {
				c.primitive.capture.Timeout = 0
			}
			before, sequence := c.ledger.current, s.sequence
			if permit, err := c.beginLocal(input); err == nil || permit != nil || c.ledger.current != before || s.sequence != sequence {
				t.Fatal("unsupported local preparation installed a ticket", mode)
			}
		})
	}
}

func TestProviderBootstrapObjectCountsRepeatedValidationAndRefusesUnfundedWork(t *testing.T) {
	c, s := providerRetainedFirstExchangeFixture(t)
	// This helper produces DATA only. No capture, complete S, ACK, selected
	// operating principal or Source is derived from these inert inputs.
	selected, err := providerDecodeHeader([]byte(c.firstResult))
	if err != nil {
		t.Fatal(err)
	}
	wide := candidateWorkVector{2: 1 << 20, 10: 1 << 60}
	baseline := newProviderWorkMeter(c, providerOperationReservation{execution: wide})
	object, raw, err := c.bootstrapObject(baseline, selected)
	if err != nil || string(raw) != object.raw || object.kind != "bootstrap" || baseline.actual[0] != 0 || baseline.actual[1] != 0 || baseline.actual[3] != 0 {
		t.Fatal("bounded bootstrap data or its measured scope", err)
	}
	a, b, n := uint64(len(c.policy.raw)), uint64(len(c.intent.raw)), uint64(len(raw))
	// Encoding is a separate weighted work event, not a decoded-byte event.
	if baseline.actual[2] != a+b+n {
		t.Fatal("policy, intent and object decoder bytes not individually counted")
	}
	for _, mode := range []string{"policy", "intent", "identifier-hash", "encoder", "object-size"} {
		t.Run(mode, func(t *testing.T) {
			meter := newProviderWorkMeter(c, providerOperationReservation{execution: wide})
			maximum := s.config.bounds.maximumObjectBytes
			defer func() { s.config.bounds.maximumObjectBytes = maximum }()
			switch mode {
			case "policy":
				meter.upper[2] = a - 1
			case "intent":
				meter.upper[2] = a + b - 1
			case "identifier-hash":
				meter.upper[10] = a*a + b*b + 2
			case "encoder":
				meter.upper[10] = a*a + b*b + 2 + uint64(len("sage-provider-bootstrap-id-v2")+1+len(c.intent.digest)) + 1
			case "object-size":
				s.config.bounds.maximumObjectBytes = n - 1
			}
			got, bytes, err := c.bootstrapObject(meter, selected)
			if err == nil || got != (providerIntegrationObject{}) || bytes != nil {
				t.Fatal("unfunded retained-data preparation succeeded", mode)
			}
			if mode == "object-size" {
				if meter.actual != baseline.actual {
					t.Fatal("refused object refunded preceding validation")
				}
			} else if !meter.failed {
				t.Fatal("unfunded meter remained usable")
			}
		})
	}
	if c.bootstrap != "" || s.sequence != 4 || c.ledger.current.counts != [3]uint64{1, 1, 1} {
		t.Fatal("DATA encoding installed a bootstrap or changed the completed first ticket")
	}
}

package registry011

import (
	"context"
	"testing"
)

func providerPendingIntegrationUnit(t *testing.T) (*providerIntegration, *providerCustodyStore) {
	t.Helper()
	s, policy, candidate, primitive := providerStoreFixture(t)
	c, err := newProviderIntegration(s, policy, candidate, primitive)
	if err != nil {
		t.Fatal(err)
	}
	return c, s
}

func TestProviderPendingAdmissionRefusalPreservesOriginalTicketAndRoots(t *testing.T) {
	for _, mode := range []string{"unconsumed-parent", "foreign-attribution", "invalid-input", "duplicate-operation", "invalid-change"} {
		t.Run(mode, func(t *testing.T) {
			c, s := providerPendingIntegrationUnit(t)
			budget, root, sequence, tip := c.ledger.current, c.root, s.sequence, s.tip
			originalIntent := c.intent
			var err error
			switch mode {
			case "unconsumed-parent":
				request, parseErr := newProviderIntegrationRequest("eth_chainId", []any{})
				if parseErr != nil {
					t.Fatal(parseErr)
				}
				_, err = c.admitRPC(request, false)
			case "foreign-attribution":
				s.activeMeter.Store(newProviderWorkMeter(&providerIntegration{}, providerOperationReservation{}))
				_, err = c.admit("local-work", admissionTestDigest("a"), "", "", providerOperationReservation{}, nil)
			case "invalid-input":
				_, err = c.admit("local-work", "unknown", "", "", providerOperationReservation{}, nil)
			case "duplicate-operation":
				_, err = c.commitChange("admit", c.firstTicket.digest, c.ledger.current, nil)
			case "invalid-change":
				_, err = c.commitChange("admit", admissionTestDigest("a"), &CandidateAdmissionLedgerChange{raw: "{}"}, nil)
			}
			s.activeMeter.Store(nil)
			if err == nil || c.ledger.current != budget || c.root != root || c.intent != originalIntent || s.sequence != sequence || s.tip != tip || c.ledger.lastConsumption != nil {
				t.Fatal("refused preparation changed retained admission or original I")
			}
		})
	}
}

func TestProviderPreparationAttemptsCannotRefreshAnEnrolledInstance(t *testing.T) {
	c, s := providerPendingIntegrationUnit(t)
	before := c.intent
	c.preparations = s.config.bounds.maximumCommits - 1
	if err := c.prepare(); err != nil {
		t.Fatal("last finite preparation", err)
	}
	if c.prepare() == nil || c.preparations != s.config.bounds.maximumCommits || c.intent != before {
		t.Fatal("preparation limit was reset or extended I")
	}
	if _, err := newProviderIntegration(s, c.policy, c.context, c.primitive); err == nil {
		t.Fatal("new handle replenished the same enrolled instance")
	}
}

func TestProviderUnfundedOrChangedConsumedRequestCannotDispatch(t *testing.T) {
	for _, mode := range []string{"wire", "params", "method", "request-budget"} {
		t.Run(mode, func(t *testing.T) {
			c, s := providerPendingIntegrationUnit(t)
			permit, err := c.consume(c.firstTicket, c.first)
			if err != nil {
				t.Fatal(err)
			}
			request := *permit.request // Finite inert substitution, never sent.
			permit.request = &request
			switch mode {
			case "wire":
				request.wire += " "
			case "params":
				request.params = `[]`
			case "method":
				request.method = "eth_chainId"
			case "request-budget":
				permit.meter.upper[0] = 0
			}
			before := c.ledger.current
			if c.use(permit) || !permit.used || c.use(permit) || c.ledger.current != before || c.ledger.current.totals.bootstrap.outstanding != c.firstTicket.work || permit.meter.actual[0] != 0 {
				t.Fatal("changed/unfunded request dispatched or refunded consumption")
			}
			s.activeMeter.Store(nil)
		})
	}
}

func TestProviderCoordinatorRejectsUnsupportedObjectsClientsAndInputs(t *testing.T) {
	for _, mode := range []string{"empty-objects", "unknown-object", "malformed-object", "object-cap", "invalid-setup-clock", "foreign-epoch", "client-body", "client-timeout", "write-method", "non-array", "cancelled", "maximum"} {
		t.Run(mode, func(t *testing.T) {
			c, s := providerPendingIntegrationUnit(t)
			before, intent, root := c.ledger.current, c.intent, c.root
			var err error
			switch mode {
			case "empty-objects":
				_, err = c.objects(nil)
			case "unknown-object":
				_, err = c.objects([]providerIntegrationObjectInput{{"unknown", []byte(`{}`)}})
			case "malformed-object":
				_, err = c.objects([]providerIntegrationObjectInput{{"enrollment", []byte(`{}`)}})
			case "object-cap":
				s.config.bounds.maximumObjectBytes = 1
				_, err = c.objects([]providerIntegrationObjectInput{{"response", []byte(`{}`)}})
			case "invalid-setup-clock":
				s.setupOrigin.WallMS, s.setupOrigin.UncertaintyMS = 1, 2
				err = c.makeIntent(s.tip)
			case "foreign-epoch":
				copy := *c.intent
				copy.clock.Epoch = admissionTestDigest("f")
				c.intent = &copy
				_, err = c.currentClock()
				c.intent = intent
			case "client-body":
				c.primitive.capture.BodyBytes = 0
				_, err = c.readClient()
			case "client-timeout":
				c.primitive.capture.Timeout = 0
				_, err = c.readClient()
			case "write-method":
				_, _, err = c.execute(context.Background(), "eth_sendRawTransaction", []any{}, 1024)
			case "non-array":
				_, _, err = c.execute(context.Background(), "eth_chainId", true, 1024)
			case "cancelled":
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				_, _, err = c.execute(ctx, "eth_chainId", []any{}, 1024)
			case "maximum":
				_, _, err = c.execute(context.Background(), "eth_chainId", []any{}, 0)
			}
			if err == nil || c.ledger.current != before || c.intent != intent || c.root != root || c.ledger.lastConsumption != nil {
				t.Fatal("invalid input refreshed admission or reached consumption")
			}
		})
	}
}

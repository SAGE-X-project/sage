package registry011

import (
	"crypto/tls"
	"strings"
	"testing"
)

// These are finite private DATA scenarios. Their standalone counters cannot
// retain originals, dispatch a request or qualify a Source. The real connected
// owner and its original execution observation remain unchanged throughout.
func TestProviderFirstOperationDataBuildersRefuseMissingOrInconsistentEvidence(t *testing.T) {
	s, o := providerFirstOperationFixture(t)
	c := o.integration
	before, stored := o.execution.actual, s.stored
	for _, mode := range []string{"consumption-ticket", "consumption-clock", "consumption-quota", "settlement-owner", "settlement-result", "settlement-measurement", "failed-result", "exchange-sample", "exchange-request", "exchange-quote", "exchange-policy", "exchange-headers", "exchange-certificate", "exchange-cipher", "change-owner", "change-kind", "change-sequence", "change-root", "change-format", "change-object", "change-node", "change-entry"} {
		t.Run(mode, func(t *testing.T) {
			meter := newProviderWorkMeter(c, providerOperationReservation{execution: candidateWorkVector{2: 256 << 20, 10: 1 << 55}})
			originalConfig, originalPolicy := s.config, c.policy
			originalNodes, originalSequence := c.ledger.current.nodes, s.sequence
			s.activeMeter.Store(meter)
			c.ledger.meter, o.indexes.meter = meter, meter
			defer func() {
				s.config, c.policy, s.sequence = originalConfig, originalPolicy, originalSequence
				c.ledger.current.nodes = originalNodes
				c.ledger.meter, o.indexes.meter = o.execution, o.execution
				s.activeMeter.Store(o.execution)
			}()
			clock := c.firstTicket.intent.clock
			end := ClockSample(clock)
			var err error
			switch {
			case strings.HasPrefix(mode, "consumption-"):
				ticket := c.firstTicket
				switch mode {
				case "consumption-ticket":
					ticket = nil
				case "consumption-clock":
					clock.Epoch = "different-unit-epoch"
				case "consumption-quota":
					meter.upper[10] = 0
				}
				_, err = providerFirstOperationConsumptionData(ticket, clock, meter)
			case strings.HasPrefix(mode, "settlement-") || mode == "failed-result":
				meter.ticket, meter.authority = c.firstTicket.digest, true
				selected, outcome, retained := c, "complete", admissionTestDigest("a")
				switch mode {
				case "settlement-owner":
					selected = nil
				case "settlement-result":
					retained = "missing-result"
				case "failed-result":
					outcome = "failed"
				}
				_, err = providerFirstOperationSettlementData(selected, c.firstTicket, outcome, retained, end, nil, meter)
			case strings.HasPrefix(mode, "exchange-"):
				request := c.first
				sample := &ProviderTransportSample{Endpoint: SepoliaRPCURL, Method: request.method, HTTPVersion: "HTTP/1.1", ServerName: "rpc.sepolia.ethpandaops.io", Complete: true, RequestHead: []byte(request.head), RequestBody: []byte(request.body), ResponseWire: []byte("finite response DATA"), ResponseBody: []byte(`{"jsonrpc":"2.0","id":1,"result":"0x1"}`), Headers: []ProviderHeader{{Name: "content-type", Value: "application/json"}}, TLSVersion: tls.VersionTLS13, CipherSuite: tls.TLS_AES_128_GCM_SHA256, VerifiedChains: [][]string{{admissionTestDigest("b")}}}
				sample.WireBytes = len(request.wire) + len(sample.ResponseWire)
				permit := &providerIntegrationPermit{owner: c, ticket: c.firstTicket, request: request, used: true, meter: meter, clock: end}
				switch mode {
				case "exchange-sample":
					sample = nil
				case "exchange-request":
					sample.RequestHead = []byte("different finite request")
				case "exchange-quote":
					s.config.capture.Timeout = 0
				case "exchange-policy":
					policy := *c.policy
					policy.raw = "null"
					c.policy = &policy
				case "exchange-headers":
					sample.Headers[0].Name = strings.Repeat("h", c.primitive.capture.HeaderBytes+1)
				case "exchange-certificate":
					sample.VerifiedChains = [][]string{{"not-a-digest"}}
				case "exchange-cipher":
					sample.CipherSuite = 0
				}
				_, _, err = providerFirstOperationExchangeData(c, permit, sample, end, meter)
			default:
				selected, root, prior := s, *c.root, s.frontierReplay.current
				kind, operation := "consume", admissionTestDigest("c")
				consumption, buildErr := providerFirstOperationConsumptionData(c.firstTicket, clock, meter)
				if buildErr != nil {
					t.Fatal(buildErr)
				}
				extra := []providerIntegrationObjectInput{{"consumption", []byte(consumption.raw)}}
				switch mode {
				case "change-owner":
					selected = nil
				case "change-kind":
					kind = "unsupported"
				case "change-sequence":
					s.sequence, prior.sequence = ^uint64(0), ^uint64(0)
				case "change-root":
					root.raw = "null"
				case "change-format":
					value := contentJSON([]byte(root.raw)).(map[string]any)
					value["format"] = "unsupported-unit-format"
					root.raw = string(providerIntegrationJSON(value))
				case "change-object":
					extra[0].raw = []byte(`{"format":"unsupported-unit-format"}`)
				case "change-node":
					operation = consumption.digest
					c.ledger.current.nodes = []string{`{"format":"unsupported-unit-format"}`}
				}
				_, err = providerFirstOperationChangeData(selected, &root, c.ledger, o.indexes, prior, kind, operation, c.ledger.current, extra, prior.clock, meter)
			}
			if err == nil || meter.actual[0] != 0 || meter.actual[9] != 0 || s.stored != stored || o.execution.actual != before {
				t.Fatal("finite invalid DATA wrote, dispatched or changed the real observer", mode, err)
			}
		})
	}
	if (&providerFirstOperationChange{}).UnmarshalJSON(nil) == nil {
		t.Fatal("serialized change became a live owner")
	}
}

func TestProviderFirstOperationWireBindingChecksFiniteOriginalData(t *testing.T) {
	for _, mode := range []string{"valid", "format", "exchange", "request", "response", "tls", "different-exchange", "missing", "quota", "closed", "meter"} {
		t.Run(mode, func(t *testing.T) {
			meter := newProviderWorkMeter(&providerIntegration{}, providerOperationReservation{execution: candidateWorkVector{2: 64 << 10, 10: 1 << 40}})
			exchange := admissionTestDigest("a")
			value := map[string]any{"format": "sage-provider-wire-binding-v1", "exchange_digest": exchange, "request_wire_digest": admissionTestDigest("b"), "response_wire_digest": admissionTestDigest("c"), "tls_digest": admissionTestDigest("d")}
			switch mode {
			case "format":
				value["format"] = "unsupported-unit-format"
			case "exchange", "request", "response", "tls":
				key := map[string]string{"exchange": "exchange_digest", "request": "request_wire_digest", "response": "response_wire_digest", "tls": "tls_digest"}[mode]
				value[key] = "not-a-digest"
			case "different-exchange":
				exchange = admissionTestDigest("e")
			case "missing":
				delete(value, "tls_digest")
			case "quota":
				meter.upper[10] = 0
			case "closed":
				meter.closed = true
			case "meter":
				meter = nil
			}
			err := providerFirstOperationBindingExchange(providerIntegrationJSON(value), exchange, meter)
			if (err == nil) != (mode == "valid") {
				t.Fatal("finite original binding accepted inconsistent or unobserved DATA", mode)
			}
		})
	}
	var missing *providerFirstOperationOwner
	if missing.bindRetainedExchangeLocked(nil, nil) == nil || missing.consumeStorageLocked(1) || missing.selectedLocked() {
		t.Fatal("missing live owner accepted retained DATA")
	}
	missing.failLocked()
}

func TestProviderFirstOperationExchangeReferencesRequireOneCompletePair(t *testing.T) {
	for _, mode := range []string{"valid", "short", "missing-binding", "missing-exchange", "duplicate-binding", "duplicate-exchange", "binding-digest", "exchange-digest"} {
		t.Run(mode, func(t *testing.T) {
			fresh := make([]providerIntegrationObject, 11)
			fresh[0] = providerIntegrationObject{kind: "wire-binding", digest: admissionTestDigest("a")}
			fresh[1] = providerIntegrationObject{kind: "exchange", digest: admissionTestDigest("b")}
			switch mode {
			case "short":
				fresh = fresh[:10]
			case "missing-binding":
				fresh[0].kind = "response"
			case "missing-exchange":
				fresh[1].kind = "tls"
			case "duplicate-binding":
				fresh[2] = fresh[0]
			case "duplicate-exchange":
				fresh[2] = fresh[1]
			case "binding-digest":
				fresh[0].digest = "missing"
			case "exchange-digest":
				fresh[1].digest = "missing"
			}
			binding, exchange, err := providerFirstOperationExchangeReferences(fresh)
			if mode == "valid" {
				if err != nil || binding != fresh[0].digest || exchange != fresh[1].digest {
					t.Fatal("unique finite pair changed")
				}
			} else if err == nil || binding != "" || exchange != "" {
				t.Fatal("ambiguous or incomplete finite references were usable", mode)
			}
		})
	}
}

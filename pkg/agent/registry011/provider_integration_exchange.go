package registry011

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"strconv"
)

func (c *providerIntegration) finishRPC(permit *providerIntegrationPermit, sample *ProviderTransportSample, executionErr error) (json.RawMessage, json.RawMessage, error) {
	if !c.valid() || permit == nil || permit.owner != c || !permit.used || permit.request == nil {
		return nil, nil, errProviderIntegration
	}
	clock, err := c.store.sample()
	if err != nil {
		c.failed = true // Full reservation remains outstanding, without a fake end.
		return nil, nil, errProviderIntegration
	}
	outcome, retained := "failed", ""
	var measurement *providerMeasuredWork
	var extra []providerIntegrationObjectInput
	if executionErr == nil && sample != nil && sample.Complete && !candidateLedgerExpired(providerClockDiagnostic(clock), permit.ticket.deadline) {
		exchange, err := c.exchangeObject(permit, sample, clock)
		if err == nil {
			outcome, retained = "complete", exchange.digest
			transport := providerIntegrationJSON(contentJSON([]byte(exchange.raw)).(map[string]any)["transport"])
			responseDigest, _ := providerIntegrationObjectDigest("response", sample.ResponseWire)
			tlsDigest, _ := providerIntegrationObjectDigest("tls", transport)
			binding := providerIntegrationJSON(map[string]string{"format": "sage-provider-wire-binding-v1", "exchange_digest": exchange.digest, "request_wire_digest": permit.ticket.input, "response_wire_digest": responseDigest, "tls_digest": tlsDigest})
			extra = []providerIntegrationObjectInput{{"exchange", []byte(exchange.raw)}, {"response", sample.ResponseWire}, {"tls", transport}, {"wire-binding", binding}}
			for _, input := range extra {
				if !permit.meter.event(providerCopyEvent, uint64(len(input.raw))) || !permit.meter.event(providerHashEvent, uint64(len(input.raw))) {
					outcome, retained = "uncertain", ""
					extra = nil
					break
				}
			}
			clock, err = c.store.sample()
			if err == nil && outcome == "complete" {
				measurement, err = permit.meter.finish(c, permit.ticket, clock)
			}
			if err != nil {
				outcome, retained = "uncertain", ""
				extra = nil
			}
		}
	} else if executionErr == nil {
		outcome = "uncertain"
	}
	// The self-containing settlement/retention/ACK follows the accepted separate
	// finite remaining envelope; it is never reported as already measured work.
	if c.beginRemaining(permit) != nil {
		c.failed = true
		return nil, nil, errProviderIntegration
	}
	if err := c.settle(permit.ticket, outcome, retained, clock, extra, measurement); err != nil {
		c.failed = true
		return nil, nil, err
	}
	if outcome != "complete" {
		c.failed = true
		return nil, nil, errProviderIntegration
	}
	if clock, err := c.currentClock(); err != nil || candidateLedgerExpired(providerClockDiagnostic(clock), permit.ticket.deadline) {
		c.failed = true // Late publication leaves the committed full charge intact.
		return nil, nil, errProviderIntegration
	}
	if permit.ticket == c.firstTicket {
		c.firstExchange, c.firstResult = retained, string(sample.Result)
	}
	c.lastExchange = retained
	// Complete responses already belong to the immutable typed X inventory.
	// Do not accumulate every raw response again in lifetime process memory.
	return append(json.RawMessage(nil), sample.ResponseBody...), append(json.RawMessage(nil), sample.Result...), nil
}

func (c *providerIntegration) exchangeObject(permit *providerIntegrationPermit, sample *ProviderTransportSample, clock ClockSample) (providerIntegrationObject, error) {
	r := permit.request
	if sample.Endpoint != SepoliaRPCURL || sample.Method != r.method || sample.HTTPVersion != "HTTP/1.1" || sample.ServerName != "rpc.sepolia.ethpandaops.io" || !sample.Complete || sample.CaptureError != "" || !bytes.Equal(sample.RequestHead, []byte(r.head)) || !bytes.Equal(sample.RequestBody, []byte(r.body)) || sample.WireBytes > c.primitive.capture.WireBytes || sample.WireBytes <= len(r.wire) || len(sample.VerifiedChains) == 0 || len(sample.VerifiedChains[0]) == 0 || len(sample.Headers) > c.primitive.capture.HeaderBytes/4 || !candidateLedgerClockAdvance(providerClockDiagnostic(permit.clock), providerClockDiagnostic(clock)) {
		return providerIntegrationObject{}, errProviderIntegration
	}
	p := contentJSON([]byte(c.policy.raw)).(map[string]any)
	provider := p["provider"].(map[string]any)
	headers := make([]map[string]string, 0, len(sample.Headers))
	for _, header := range sample.Headers {
		headers = append(headers, map[string]string{"name": header.Name, "value": header.Value})
	}
	transport := map[string]any{"endpoint": SepoliaRPCURL, "method": "POST", "http_version": sample.HTTPVersion, "status": "200", "response_headers": headers, "tls_version": strconv.FormatUint(uint64(sample.TLSVersion), 10), "cipher_suite": strconv.FormatUint(uint64(sample.CipherSuite), 10), "server_name": sample.ServerName, "verified_chain": sample.VerifiedChains[0], "roots_artifact": provider["tls_roots_artifact"], "redirects": "0", "proxy": "none"}
	id := c.intent.identity
	raw := providerIntegrationJSON(map[string]any{"format": "sage-provider-exchange-v2", "policy": id.Policy, "namespace": id.Namespace, "instance_key": id.Instance, "source_id": id.SourceID, "generation": strconv.FormatUint(id.Generation, 10), "ticket_digest": permit.ticket.digest, "provider_digest": id.Provider, "transport": transport, "request": base64.RawURLEncoding.EncodeToString(sample.RequestBody), "response": base64.RawURLEncoding.EncodeToString(sample.ResponseBody), "clock_start": providerClockObject(permit.clock), "clock_end": providerClockObject(clock)})
	return newProviderIntegrationObject("exchange", raw, c.store.config.bounds.maximumObjectBytes)
}

// All outcomes charge the full reservation. Complete records only the private
// owner's measured vector through clock_end; future S/retention/ACK work belongs
// to its already reserved remaining envelope and cannot become a fake measure.
func (c *providerIntegration) settle(ticket *CandidateAdmissionTicket, outcome, retained string, clock ClockSample, extra []providerIntegrationObjectInput, measurement *providerMeasuredWork) error {
	var actual, result any
	if outcome == "complete" {
		work, ok := measurement.claim(c, ticket, clock)
		if !ok {
			return errProviderIntegration
		}
		actual, result = candidateLedgerWorkObject(work), retained
	}
	raw := providerIntegrationJSON(map[string]any{"format": "sage-provider-budget-settlement-v2", "ticket_digest": ticket.digest, "outcome": outcome, "actual_work": actual, "retained_root": result, "clock_end": providerClockObject(clock)})
	settlement, err := ParseCandidateAdmissionSettlement(ticket, raw, 64<<10)
	if err != nil {
		return errProviderIntegration
	}
	change, err := c.ledger.AppendCandidateSettlement(settlement)
	if err != nil {
		return errProviderIntegration
	}
	extra = append(extra, providerIntegrationObjectInput{"settlement", raw})
	_, err = c.commitChange("settle", settlement.digest, change, extra)
	return err
}

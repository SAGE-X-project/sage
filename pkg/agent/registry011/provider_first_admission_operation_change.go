package registry011

import (
	"encoding/base64"
	"strconv"
)

// A coordinator delta is proposed DATA only. The independent physical writer
// must reconstruct its ledger and indexes from newly retained originals. This
// object cannot supply an ACK, reopen F or select an operating principal.
type providerFirstOperationChange struct {
	root                     *CandidateEnrollmentRoot
	delta                    *providerIndexDelta
	fresh                    []providerIntegrationObject
	inventory                *providerIntegrationInventory
	intent, commit, frontier []byte
	after                    providerFrontierHighwater
}

func (*providerFirstOperationChange) UnmarshalJSON([]byte) error { return errProviderIntegration }

// Only the just-retained successful first S can populate this private index.
// Read and hash its original binding, then let independent typed replay check
// the complete X/request/response/TLS closure again. No cached producer bytes
// or restored receipt can stand in for the retained original.
func (o *providerFirstOperationOwner) bindRetainedExchangeLocked(s *providerCustodyStore, fresh []providerIntegrationObject) error {
	if o == nil || !o.selectedLocked() || o.store != s || o.state != providerFirstOperationSettlement || !o.consumption || !o.settlement || s.sequence != 3 || s.activeMeter.Load() != o.remaining || len(fresh) != 11 || s.bindings == nil || len(s.bindings) != 0 {
		return errProviderIntegration
	}
	binding, exchange, err := providerFirstOperationExchangeReferences(fresh)
	if err != nil {
		return errProviderIntegration
	}
	raw, err := s.object("wire-binding", binding)
	if err != nil || providerFirstOperationBindingExchange(raw, exchange, o.remaining) != nil {
		return errProviderIntegration
	}
	s.bindings[exchange] = binding
	return nil
}

// Select only two unique references from the bounded proposed inventory.
// The returned digests remain DATA until their retained originals are checked.
func providerFirstOperationExchangeReferences(fresh []providerIntegrationObject) (string, string, error) {
	if len(fresh) != 11 {
		return "", "", errProviderIntegration
	}
	var binding, exchange string
	for _, object := range fresh {
		switch object.kind {
		case "wire-binding":
			if binding != "" {
				return "", "", errProviderIntegration
			}
			binding = object.digest
		case "exchange":
			if exchange != "" {
				return "", "", errProviderIntegration
			}
			exchange = object.digest
		}
	}
	if !lowerHash(binding) || !lowerHash(exchange) {
		return "", "", errProviderIntegration
	}
	return binding, exchange, nil
}

// This finite DATA validator grants no ACK or execution permission.
func providerFirstOperationBindingExchange(raw []byte, exchange string, meter *providerWorkMeter) error {
	if meter == nil || meter.closed || meter.failed || len(raw) == 0 || len(raw) > candidateLedgerMetadataLimit {
		return errProviderIntegration
	}
	m := contentObject(providerReplayJSON(raw, meter), "format", "exchange_digest", "request_wire_digest", "response_wire_digest", "tls_digest")
	if meter.failed || m == nil || m["format"] != "sage-provider-wire-binding-v1" || !providerCandidateDigest(m["exchange_digest"]) || !providerCandidateDigest(m["request_wire_digest"]) || !providerCandidateDigest(m["response_wire_digest"]) || !providerCandidateDigest(m["tls_digest"]) || !providerReplayTextEqual(contentString(m["exchange_digest"]), exchange, meter) {
		return errProviderIntegration
	}
	return nil
}

// The first consumption has no predecessor. The caller still has to retain
// this original and independently validate it before its one allowed RPC.
func providerFirstOperationConsumptionData(ticket *CandidateAdmissionTicket, clock CandidateClockDiagnostic, meter *providerWorkMeter) (*CandidateConsumption, error) {
	if ticket == nil || ticket.intent == nil || ticket.sequence != 0 || ticket.kind != "rpc-capture" || meter == nil || meter.failed || meter.closed || !candidateLedgerClockAdvance(ticket.intent.clock, clock) || candidateLedgerExpired(clock, ticket.deadline) {
		return nil, errProviderIntegration
	}
	id := ticket.intent.identity
	raw := providerReplayCanonical(map[string]any{"format": "sage-provider-budget-consumption-v1", "ticket_digest": ticket.digest, "namespace": id.Namespace, "instance_key": id.Instance, "source_id": id.SourceID, "generation": strconv.FormatUint(id.Generation, 10), "fence": strconv.FormatUint(ticket.intent.fence, 10), "sequence": "0", "previous_consumption": nil, "kind": ticket.kind, "input_digest": ticket.input, "clock_start": providerDiagnosticClockObject(clock)}, candidateLedgerMetadataLimit, meter)
	return parseCandidateConsumptionObserved(ticket, raw, candidateLedgerMetadataLimit, meter)
}

// The execution observation is frozen before this remaining-phase record is
// constructed. Neither an upper nor an externally supplied vector is actual.
func providerFirstOperationSettlementData(c *providerIntegration, ticket *CandidateAdmissionTicket, outcome, retained string, clock ClockSample, measurement *providerMeasuredWork, meter *providerWorkMeter) (*CandidateAdmissionSettlement, error) {
	if c == nil || ticket == nil || ticket.intent != c.intent || meter == nil || meter.integration != c || meter.scan != nil || !meter.authority || meter.failed || meter.closed || meter.ticket != ticket.digest || c.store.activeMeter.Load() != meter || outcome != "complete" && outcome != "failed" && outcome != "uncertain" {
		return nil, errProviderIntegration
	}
	var actual, result any
	if outcome == "complete" {
		if !lowerHash(retained) {
			return nil, errProviderIntegration
		}
		work, ok := measurement.claim(c, ticket, clock)
		if !ok {
			return nil, errProviderIntegration
		}
		actual, result = candidateLedgerWorkObject(work), retained
	} else if retained != "" || measurement != nil {
		return nil, errProviderIntegration
	}
	raw := providerReplayCanonical(map[string]any{"format": "sage-provider-budget-settlement-v2", "ticket_digest": ticket.digest, "outcome": outcome, "actual_work": actual, "retained_root": result, "clock_end": providerClockObject(clock)}, candidateLedgerMetadataLimit, meter)
	return parseCandidateAdmissionSettlementObserved(ticket, raw, candidateLedgerMetadataLimit, meter)
}

// Construct X and its exact separately retained wire/TLS binding while the
// execution meter is still open. These are diagnostic originals, not a Source
// qualification or permission to publish the observed response to an Agent.
func providerFirstOperationExchangeData(c *providerIntegration, permit *providerIntegrationPermit, sample *ProviderTransportSample, clock ClockSample, meter *providerWorkMeter) ([]providerIntegrationObjectInput, string, error) {
	if !c.valid() || permit == nil || permit.request == nil {
		return nil, "", errProviderIntegration
	}
	quoted, ok := providerFirstOperationStorageUpper(c.store.config.bounds, c.store.config.capture, uint64(len(permit.request.wire)))
	if !ok {
		return nil, "", errProviderIntegration
	}
	return providerExchangeData(c, permit, sample, clock, meter, c.primitive.capture, quoted)
}

// The later live owner supplies its already bound method class, and this
// builder derives storage from that same class without changing global caps.
func providerOperationExchangeData(c *providerIntegration, permit *providerIntegrationPermit, sample *ProviderTransportSample, clock ClockSample, meter *providerWorkMeter, capture ProviderCaptureLimits) ([]providerIntegrationObjectInput, string, error) {
	if !c.valid() || permit == nil || permit.request == nil {
		return nil, "", errProviderIntegration
	}
	quoted, ok := providerOperationStorageUpper(c.store.config.bounds, capture, uint64(len(permit.request.wire)))
	if !ok {
		return nil, "", errProviderIntegration
	}
	return providerExchangeData(c, permit, sample, clock, meter, capture, quoted)
}

// Private construction DATA only. The owning operation still enforces its
// exact original request and bound class before burn, capture and retention.
func providerExchangeData(c *providerIntegration, permit *providerIntegrationPermit, sample *ProviderTransportSample, clock ClockSample, meter *providerWorkMeter, capture ProviderCaptureLimits, quoted providerFirstOperationStorage) ([]providerIntegrationObjectInput, string, error) {
	if !c.valid() || c.store.config.bounds.firstAdmission == nil || !capture.valid() || permit == nil || permit.owner != c || permit.ticket == nil || permit.request == nil || !permit.used || meter == nil || meter != permit.meter || meter.integration != c || meter.scan != nil || meter.authority || meter.closed || meter.failed || c.store.activeMeter.Load() != meter || sample == nil || !sample.Complete || sample.CaptureError != "" || sample.Endpoint != SepoliaRPCURL || sample.Method != permit.request.method || sample.HTTPVersion != "HTTP/1.1" || sample.ServerName != "rpc.sepolia.ethpandaops.io" || sample.WireBytes <= len(permit.request.wire) || sample.WireBytes > capture.WireBytes || len(sample.ResponseWire) != sample.WireBytes-len(permit.request.wire) || len(sample.ResponseBody) > capture.BodyBytes || len(sample.Headers) > min(capture.HeaderBytes/4, 128) || len(sample.VerifiedChains) == 0 || len(sample.VerifiedChains[0]) == 0 || len(sample.VerifiedChains[0]) > c.store.config.bounds.tls.certificates || !candidateLedgerClockAdvance(providerClockDiagnostic(permit.clock), providerClockDiagnostic(clock)) || candidateLedgerExpired(providerClockDiagnostic(clock), permit.ticket.deadline) {
		return nil, "", errProviderIntegration
	}
	if !providerReplayPhaseRawTextEqual(sample.RequestHead, permit.request.head, meter) || !providerReplayPhaseRawTextEqual(sample.RequestBody, permit.request.body, meter) {
		return nil, "", errProviderIntegration
	}
	p, ok := providerReplayJSON(providerReplayPhaseBytes(c.policy.raw, meter), meter).(map[string]any)
	provider, providerOK := p["provider"].(map[string]any)
	if !ok || !providerOK {
		return nil, "", errProviderIntegration
	}
	headers := make([]map[string]string, 0, len(sample.Headers))
	var headerBytes uint64
	for _, header := range sample.Headers {
		var fits bool
		headerBytes, fits = providerUpperSum(headerBytes, uint64(len(header.Name)), uint64(len(header.Value)))
		if !fits || headerBytes > uint64(capture.HeaderBytes) {
			return nil, "", errProviderIntegration
		}
		headers = append(headers, map[string]string{"name": header.Name, "value": header.Value})
	}
	for _, digest := range sample.VerifiedChains[0] {
		if !providerCandidateDigest(digest) {
			return nil, "", errProviderIntegration
		}
	}
	transport := map[string]any{"endpoint": SepoliaRPCURL, "method": "POST", "http_version": sample.HTTPVersion, "status": "200", "response_headers": headers, "tls_version": strconv.FormatUint(uint64(sample.TLSVersion), 10), "cipher_suite": strconv.FormatUint(uint64(sample.CipherSuite), 10), "server_name": sample.ServerName, "verified_chain": sample.VerifiedChains[0], "roots_artifact": provider["tls_roots_artifact"], "redirects": "0", "proxy": "none"}
	transportRaw := providerReplayCanonical(transport, quoted.transport, meter)
	if len(transportRaw) == 0 || !providerIntegrationCipher(transport) || !meter.event(providerEncodeEvent, uint64(len(sample.RequestBody))) || !meter.event(providerEncodeEvent, uint64(len(sample.ResponseBody))) {
		return nil, "", errProviderIntegration
	}
	id := c.intent.identity
	raw := providerReplayCanonical(map[string]any{"format": "sage-provider-exchange-v2", "policy": id.Policy, "namespace": id.Namespace, "instance_key": id.Instance, "source_id": id.SourceID, "generation": strconv.FormatUint(id.Generation, 10), "ticket_digest": permit.ticket.digest, "provider_digest": id.Provider, "transport": transport, "request": base64.RawURLEncoding.EncodeToString(sample.RequestBody), "response": base64.RawURLEncoding.EncodeToString(sample.ResponseBody), "clock_start": providerClockObject(permit.clock), "clock_end": providerClockObject(clock)}, quoted.exchange, meter)
	exchange, err := newProviderIntegrationObjectObserved("exchange", raw, quoted.exchange, meter)
	response, responseErr := providerIntegrationObjectDigestObserved("response", sample.ResponseWire, meter)
	tls, tlsErr := providerIntegrationObjectDigestObserved("tls", transportRaw, meter)
	if err != nil || !responseErr || !tlsErr {
		return nil, "", errProviderIntegration
	}
	binding := providerReplayCanonical(map[string]string{"format": "sage-provider-wire-binding-v1", "exchange_digest": exchange.digest, "request_wire_digest": permit.ticket.input, "response_wire_digest": response, "tls_digest": tls}, candidateLedgerMetadataLimit, meter)
	if len(binding) == 0 || meter.failed {
		return nil, "", errProviderIntegration
	}
	return []providerIntegrationObjectInput{{"exchange", providerReplayPhaseBytes(exchange.raw, meter)}, {"response", sample.ResponseWire}, {"tls", transportRaw}, {"wire-binding", binding}}, exchange.digest, nil
}

// Build one closed C/S delta under the NEW operation's current observer. The
// caller owns the coordinator ledger and keeps store.mu throughout. No write,
// observer transfer, hold or request dispatch occurs in this construction.
func providerFirstOperationChangeData(s *providerCustodyStore, before *CandidateEnrollmentRoot, ledger *CandidateAdmissionLedger, indexes *providerLedgerIndexes, prior providerFrontierRecord, kind, operation string, change *CandidateAdmissionLedgerChange, extra []providerIntegrationObjectInput, clock CandidateClockDiagnostic, meter *providerWorkMeter) (*providerFirstOperationChange, error) {
	if !s.valid() || s.config.bounds.firstAdmission == nil || s.frontierFiles == nil || s.frontierReplay == nil || s.frontierReplay.body == nil || meter == nil || meter.owner != meter || meter.closed || meter.failed || meter.integration == nil || meter.integration.store != s || meter.scan != nil || s.activeMeter.Load() != meter || before == nil || before.digest != s.enrollment || ledger == nil || !ledger.valid() || ledger.meter != meter || change == nil || ledger.current != change || indexes == nil || !indexes.valid() || indexes.meter != meter || indexes.ledger != ledger || indexes.heads != prior.indexes || prior.sequence != s.sequence || prior.commit != s.tip || prior.enrollment != before.digest || prior.objects != s.objectsRoot || prior.admission != ledger.intent.digest || prior.sequence < 2 || !candidateLedgerClockAdvance(prior.clock, clock) || candidateLedgerExpired(clock, prior.deadline) || !lowerHash(operation) || change.admission != ledger.intent.digest || s.frontierReplay.body.ledger == ledger || s.frontierReplay.indexes == indexes {
		return nil, errProviderIntegration
	}
	if kind != "consume" && kind != "settle" || kind == "consume" && (len(extra) != 1 || extra[0].kind != "consumption") || kind == "settle" && (len(extra) != 1 && len(extra) != 5 || extra[len(extra)-1].kind != "settlement") {
		return nil, errProviderIntegration
	}
	if s.sequence == ^uint64(0) || before.sequence == ^uint64(0) || s.frontierFiles.pin.ordinal == ^uint64(0) || s.sequence >= uint64(s.config.bounds.maximumCommits) {
		return nil, errProviderIntegration
	}
	m, ok := providerReplayJSON(providerReplayPhaseBytes(before.raw, meter), meter).(map[string]any)
	if !ok {
		return nil, errProviderIntegration
	}
	m["sequence"], m["previous_root"], m["budget_root"] = strconv.FormatUint(before.sequence+1, 10), before.digest, change.digest
	rootRaw := providerReplayCanonical(m, candidateLedgerMetadataLimit, meter)
	root, err := parseCandidateEnrollmentRootObserved(s.frontierReplay.body.context, rootRaw, candidateLedgerMetadataLimit, meter)
	if err != nil {
		return nil, errProviderIntegration
	}
	fresh := make([]providerIntegrationObject, 0, len(extra)+6)
	for i, input := range extra {
		maximum := uint64(candidateLedgerMetadataLimit)
		if kind == "settle" && len(extra) == 5 && i < 3 {
			if input.kind != []string{"exchange", "response", "tls"}[i] {
				return nil, errProviderIntegration
			}
			maximum = s.config.bounds.maximumObjectBytes
		} else if kind == "settle" && len(extra) == 5 && i == 3 && input.kind != "wire-binding" {
			return nil, errProviderIntegration
		}
		object, objectErr := newProviderIntegrationObjectObserved(input.kind, input.raw, maximum, meter)
		if objectErr != nil {
			return nil, errProviderIntegration
		}
		fresh = append(fresh, object)
	}
	entry := fresh[len(fresh)-1]
	if entry.digest != operation || len(change.nodes) != 1 {
		return nil, errProviderIntegration
	}
	for _, raw := range change.nodes {
		object, objectErr := newProviderIntegrationObjectObserved("ledger-node", providerReplayPhaseBytes(raw, meter), candidateLedgerMetadataLimit, meter)
		if objectErr != nil {
			return nil, errProviderIntegration
		}
		fresh = append(fresh, object)
	}
	for _, input := range []providerIntegrationObjectInput{{"budget", providerReplayPhaseBytes(change.raw, meter)}, {"enrollment", rootRaw}} {
		object, objectErr := newProviderIntegrationObjectObserved(input.kind, input.raw, candidateLedgerMetadataLimit, meter)
		if objectErr != nil {
			return nil, errProviderIntegration
		}
		fresh = append(fresh, object)
	}
	mutations := 3
	if kind == "consume" {
		mutations = 2 // Consumption changes the ticket and operation indexes only.
	}
	delta, err := indexes.prepare(kind, operation, s.sequence+1, entry, root, ledger)
	if err != nil || len(delta.mutations) != mutations {
		return nil, errProviderIntegration
	}
	heads := delta.previous
	for _, mutation := range delta.mutations {
		if len(mutation.object.raw) > candidateLedgerMetadataLimit {
			return nil, errProviderIntegration
		}
		fresh = append(fresh, mutation.object)
		heads[mutation.chain] = mutation.object.digest
	}
	if len(fresh) != len(extra)+3+mutations || len(fresh) > s.config.bounds.maximumEntries || len(s.objects) > s.config.bounds.maximumObjects-len(fresh) {
		return nil, errProviderIntegration
	}
	inventory, err := newProviderFrontierInventoryObserved(s.objectsRoot, fresh, s.config.bounds.maximumEntries, min(s.config.bounds.maximumStoredBytes, 64<<20), meter)
	previous, previousErr := prior.bytesObserved(meter)
	link := providerFrontierLink{identity: s.config.identity, fence: root.fence, sequence: s.sequence, commit: s.tip, setup: s.setupTip, enrollment: before.digest, objects: s.objectsRoot, frontier: providerReplayDigest("sage-provider-custody-frontier-v1", previous, meter)}
	intent, intentErr := providerFrontierIntentDiagnosticObserved(link, root, inventory, kind, operation, clock, s.config.bounds, meter)
	commit := providerFrontierCommitObserved(intent, meter)
	if err != nil || previousErr != nil || intentErr != nil || len(commit) == 0 {
		return nil, errProviderIntegration
	}
	frontier := prior
	frontier.sequence, frontier.previous, frontier.commit = s.sequence+1, link.frontier, providerReplayDigest("sage-provider-custody-commit-v2", commit, meter)
	frontier.objects, frontier.enrollment, frontier.indexes = inventory.digest, root.digest, heads
	frontier.budget, frontier.clock = change.digest, clock
	raw, err := frontier.bytesObserved(meter)
	after := s.frontierFiles.pin
	after.ordinal, after.sequence = after.ordinal+1, frontier.sequence
	after.previous = providerReplayDigest("sage-provider-custody-highwater-v1", providerReplayPhaseBytes(s.frontierFiles.pinRaw, meter), meter)
	after.commit, after.frontier = frontier.commit, providerReplayDigest("sage-provider-custody-frontier-v1", raw, meter)
	if err != nil || !after.validObserved(s.config, meter) || meter.failed {
		return nil, errProviderIntegration
	}
	return &providerFirstOperationChange{root, delta, fresh, inventory, intent, commit, raw, after}, nil
}

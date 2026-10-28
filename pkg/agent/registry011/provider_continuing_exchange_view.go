package registry011

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"strconv"
)

// Reconstruct complete RPC DATA under this separately funded local phase.
// This is checked DATA only: it cannot admit, dispatch, retain or restore a
// role. Each caller phase supplies its own current observer and reads every
// original again. The earlier RPC's completed observer is never reused.
func (r *providerCustodyReplay) historyExchangeResultView(s *providerCustodyStore, digest string, expected *providerIntegrationRequest) (json.RawMessage, error) {
	if r == nil || s == nil || expected == nil || r.meter == nil || r.meter.closed || r.meter.failed || s.activeMeter.Load() != r.meter || r.ledger == nil || r.intent == nil || r.policy == nil || s.config.bounds.methods == nil || !lowerHash(digest) {
		return nil, errProviderIntegration
	}
	capture, err := s.config.bounds.methods.captureObserved(expected, s.config.capture, r.meter)
	if err != nil {
		return nil, errProviderIntegration
	}
	limits, ok := providerOperationStorageUpper(s.config.bounds, capture, uint64(len(expected.wire)))
	if !ok || !r.continuingPinBoundOriginal(s, "exchange", digest, limits.exchange) {
		return nil, errProviderIntegration
	}
	raw, err := s.object("exchange", digest)
	x := contentObject(providerReplayJSON(raw, r.meter), "format", "policy", "namespace", "instance_key", "source_id", "generation", "ticket_digest", "provider_digest", "transport", "request", "response", "clock_start", "clock_end")
	id := r.intent.identity
	if err != nil || x == nil || x["format"] != "sage-provider-exchange-v2" || x["policy"] != id.Policy || x["namespace"] != id.Namespace || x["instance_key"] != id.Instance || x["source_id"] != id.SourceID || x["generation"] != strconv.FormatUint(id.Generation, 10) || x["provider_digest"] != id.Provider {
		return nil, errProviderIntegration
	}
	entry, exists := r.ledger.tickets[contentString(x["ticket_digest"])]
	if !exists || entry.ticket == nil || entry.ticket.digest != x["ticket_digest"] || entry.ticket.kind != "rpc-capture" || entry.ticket.input != expected.inputDigest || entry.ticket.method != expected.method || entry.ticket.params != expected.paramsDigest || entry.consumption == nil || !r.continuingPinBoundOriginal(s, "settlement", entry.terminal, 64<<10) {
		return nil, errProviderIntegration
	}
	raw, err = s.object("settlement", entry.terminal)
	settlement, parseErr := s.parsePinSettlementObserved(entry.ticket, raw, r.meter)
	if err != nil || parseErr != nil || settlement.outcome != "complete" || settlement.retained != digest {
		return nil, errProviderIntegration
	}
	policy, valid := providerReplayJSON(providerReplayPhaseBytes(r.policy.raw, r.meter), r.meter).(map[string]any)
	if !valid {
		return nil, errProviderIntegration
	}
	clock, clockOK := policy["clock"].(map[string]any)
	provider, providerOK := policy["provider"].(map[string]any)
	if !clockOK || !providerOK {
		return nil, errProviderIntegration
	}
	start, a := candidateAdmissionClock(x["clock_start"], clock)
	end, b := candidateAdmissionClock(x["clock_end"], clock)
	if !a || !b || !candidateLedgerClockAdvance(entry.consumption.clock, start) || !candidateLedgerClockAdvance(start, end) || !candidateLedgerClockAdvance(end, settlement.clock) || candidateLedgerExpired(end, entry.ticket.deadline) {
		return nil, errProviderIntegration
	}
	request, err := s.object("request", entry.ticket.input)
	if err != nil || !providerReplayPhaseRawTextEqual(request, expected.wire, r.meter) || !historyViewBase64Matches(x["request"], providerReplayPhaseBytes(expected.body, r.meter), r.meter) {
		return nil, errProviderIntegration
	}
	actualClass, err := providerRetainedMethodCapture(s, entry.ticket, request, r.meter)
	if err != nil || actualClass != capture {
		return nil, errProviderIntegration
	}
	transport := contentObject(x["transport"], "endpoint", "method", "http_version", "status", "response_headers", "tls_version", "cipher_suite", "server_name", "verified_chain", "roots_artifact", "redirects", "proxy")
	if transport == nil || transport["endpoint"] != SepoliaRPCURL || transport["method"] != "POST" || transport["http_version"] != "HTTP/1.1" || transport["status"] != "200" || transport["server_name"] != "rpc.sepolia.ethpandaops.io" || transport["roots_artifact"] != provider["tls_roots_artifact"] || transport["redirects"] != "0" || transport["proxy"] != "none" || !providerIntegrationCipher(transport) {
		return nil, errProviderIntegration
	}
	chain, valid := transport["verified_chain"].([]any)
	if !valid || len(chain) == 0 || len(chain) > 16 {
		return nil, errProviderIntegration
	}
	for _, certificate := range chain {
		if !providerCandidateDigest(certificate) {
			return nil, errProviderIntegration
		}
	}
	bindingDigest := s.bindings[digest]
	if !r.continuingPinBoundOriginal(s, "wire-binding", bindingDigest, 8<<10) {
		return nil, errProviderIntegration
	}
	raw, err = s.object("wire-binding", bindingDigest)
	binding := contentObject(providerReplayJSON(raw, r.meter), "format", "exchange_digest", "request_wire_digest", "response_wire_digest", "tls_digest")
	if err != nil || binding == nil || binding["format"] != "sage-provider-wire-binding-v1" || binding["exchange_digest"] != digest || binding["request_wire_digest"] != expected.inputDigest || !providerCandidateDigest(binding["tls_digest"]) || !providerCandidateDigest(binding["response_wire_digest"]) {
		return nil, errProviderIntegration
	}
	metadata := providerReplayCanonical(transport, 1<<20, r.meter)
	tlsDigest, valid := providerIntegrationObjectDigestObserved("tls", metadata, r.meter)
	if len(metadata) == 0 || !valid || !providerReplayTextEqual(tlsDigest, contentString(binding["tls_digest"]), r.meter) || !r.continuingPinBoundOriginal(s, "tls", tlsDigest, limits.transport) {
		return nil, errProviderIntegration
	}
	retained, err := s.object("tls", tlsDigest)
	if err != nil || !providerReplayEqual(metadata, retained, r.meter) || !r.continuingPinBoundOriginal(s, "response", contentString(binding["response_wire_digest"]), uint64(capture.WireBytes)) {
		return nil, errProviderIntegration
	}
	raw, err = s.object("response", contentString(binding["response_wire_digest"]))
	if err != nil {
		return nil, errProviderIntegration
	}
	_, parsedHeaders, body, result, err := readProviderHTTPObserved(bufio.NewReader(bytes.NewReader(raw)), capture, r.meter, uint64(len(raw)))
	if err != nil || !historyViewBase64Matches(x["response"], body, r.meter) {
		return nil, errProviderIntegration
	}
	items, valid := transport["response_headers"].([]any)
	if !valid || len(items) == 0 || len(items) > capture.HeaderBytes/4 {
		return nil, errProviderIntegration
	}
	values := make([]map[string]string, 0, len(parsedHeaders))
	for _, header := range parsedHeaders {
		values = append(values, map[string]string{"name": header.Name, "value": header.Value})
	}
	if !providerReplayEqual(providerReplayCanonical(items, 1<<20, r.meter), providerReplayCanonical(values, 1<<20, r.meter), r.meter) || r.meter.failed {
		return nil, errProviderIntegration
	}
	// readProviderHTTPObserved has parsed the complete exact RPC envelope at
	// the same class limits. Matching the canonical encoding binds X's body;
	// no second decode of that same body is needed within this one phase.
	return result, nil
}

func historyViewBase64Matches(value any, raw []byte, meter *providerWorkMeter) bool {
	text, valid := value.(string)
	if !valid || len(raw) == 0 || meter == nil || !meter.event(providerEncodeEvent, uint64(len(raw))) {
		return false
	}
	encoded := base64.RawURLEncoding.EncodeToString(raw)
	return providerReplayTextEqual(text, encoded, meter)
}

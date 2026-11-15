package registry011

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"strconv"
	"unicode/utf8"
)

// The same closed HTTP/RPC request contract is reconstructed from this
// recovery pass's actual bytes. No request is sent by this parser.
func (m *providerHistoryRecoveryInitialModel) request(ticket *CandidateAdmissionTicket) error {
	if ticket == nil || ticket.kind != "rpc-capture" || ticket.sequence != 0 {
		return errProviderIntegration
	}
	meter := m.recovery.meter
	wire, err := m.object("request", ticket.input)
	parts := bytes.SplitN(wire, []byte("\r\n\r\n"), 2)
	if err != nil || len(parts) != 2 || !meter.event(providerCompareEvent, uint64(len(parts[1]))) || !utf8.Valid(parts[1]) ||
		!meter.event(providerCompareEvent, uint64(len(parts[1]))) || !validRPCUnicode(parts[1]) || !meter.event(providerCodecEvent, uint64(len(parts[1]))) {
		return errProviderIntegration
	}
	decoder := json.NewDecoder(bytes.NewReader(parts[1]))
	decoder.UseNumber()
	if !readRPCValue(decoder, 0, 10) {
		return errProviderIntegration
	}
	if _, err := decoder.Token(); err != io.EOF {
		return errProviderIntegration
	}
	var body struct {
		Version string `json:"jsonrpc"`
		ID      uint64 `json:"id"`
		Method  string `json:"method"`
		Params  any    `json:"params"`
	}
	if providerReplayUnmarshal(parts[1], &body, meter) != nil || body.Version != "2.0" || body.ID != 1 || body.Method != ticket.method {
		return errProviderIntegration
	}
	body.Params = providerRequestNumbersObserved(body.Params, meter)
	request, err := newProviderIntegrationRequestObserved(body.Method, body.Params, meter)
	if err != nil || !providerReplayEqual(wire, providerReplayPhaseBytes(request.wire, meter), meter) || request.paramsDigest != ticket.params || !request.firstFinalizedObserved(meter) {
		return errProviderIntegration
	}
	paramsDigest, ok := providerIntegrationObjectDigestObserved("params", providerReplayPhaseBytes(request.params, meter), meter)
	params, err := m.object("params", paramsDigest)
	if !ok || err != nil || !providerReplayEqual(params, providerReplayPhaseBytes(request.params, meter), meter) {
		return errProviderIntegration
	}
	return nil
}

func (m *providerHistoryRecoveryInitialModel) exchange(settlement *CandidateAdmissionSettlement) error {
	body, meter := m.frontier.body, m.recovery.meter
	if settlement == nil || settlement.ticket == nil || body.intent == nil || body.ledger == nil {
		return errProviderIntegration
	}
	ticket, id := settlement.ticket, body.intent.identity
	raw, err := m.object("exchange", settlement.retained)
	v := contentObject(providerReplayJSON(raw, meter), "format", "policy", "namespace", "instance_key", "source_id", "generation", "ticket_digest", "provider_digest", "transport", "request", "response", "clock_start", "clock_end")
	if err != nil || v == nil || v["format"] != "sage-provider-exchange-v2" || v["policy"] != id.Policy || v["namespace"] != id.Namespace ||
		v["instance_key"] != id.Instance || v["source_id"] != id.SourceID || v["generation"] != strconv.FormatUint(id.Generation, 10) || v["ticket_digest"] != ticket.digest || v["provider_digest"] != id.Provider {
		return errProviderIntegration
	}
	p, policyOK := providerReplayJSON(providerReplayPhaseBytes(body.policy.raw, meter), meter).(map[string]any)
	clock, clockOK := p["clock"].(map[string]any)
	provider, providerOK := p["provider"].(map[string]any)
	start, a := candidateAdmissionClock(v["clock_start"], clock)
	end, b := candidateAdmissionClock(v["clock_end"], clock)
	state, exists := body.ledger.tickets[ticket.digest]
	if !policyOK || !clockOK || !providerOK || !a || !b || !exists || state.ticket != ticket || state.consumption == nil ||
		!candidateLedgerClockAdvance(state.consumption.clock, start) || !candidateLedgerClockAdvance(start, end) ||
		!candidateLedgerClockAdvance(end, settlement.clock) || candidateLedgerExpired(end, ticket.deadline) {
		return errProviderIntegration
	}
	request, err := m.object("request", ticket.input)
	parts := bytes.SplitN(request, []byte("\r\n\r\n"), 2)
	capture := m.recovery.config.capture
	requestBody, requestOK := providerCanonicalBase64Observed(v["request"], 16<<10, meter)
	response, responseOK := providerCanonicalBase64Observed(v["response"], capture.BodyBytes, meter)
	if err != nil || len(parts) != 2 || !requestOK || !responseOK || !providerReplayEqual(parts[1], requestBody, meter) {
		return errProviderIntegration
	}
	if _, err := decodeRPCResultObserved(response, 1, RPCResponseLimits{capture.BodyBytes, capture.JSONDepth}, meter); err != nil {
		return errProviderIntegration
	}
	transport := contentObject(v["transport"], "endpoint", "method", "http_version", "status", "response_headers", "tls_version", "cipher_suite", "server_name", "verified_chain", "roots_artifact", "redirects", "proxy")
	if transport == nil || transport["endpoint"] != SepoliaRPCURL || transport["method"] != "POST" || transport["http_version"] != "HTTP/1.1" ||
		transport["status"] != "200" || transport["server_name"] != "rpc.sepolia.ethpandaops.io" || transport["roots_artifact"] != provider["tls_roots_artifact"] ||
		transport["redirects"] != "0" || transport["proxy"] != "none" || !providerIntegrationCipher(transport) {
		return errProviderIntegration
	}
	chain, chainOK := transport["verified_chain"].([]any)
	if !chainOK || len(chain) == 0 || len(chain) > 16 {
		return errProviderIntegration
	}
	for _, certificate := range chain {
		if !providerCandidateDigest(certificate) {
			return errProviderIntegration
		}
	}
	metadata := providerReplayCanonical(transport, 1<<20, meter)
	digest, digestOK := providerIntegrationObjectDigestObserved("tls", metadata, meter)
	retained, err := m.object("tls", digest)
	binding, bindingErr := m.wireBinding(settlement.retained)
	if !digestOK || err != nil || bindingErr != nil || binding["request_wire_digest"] != ticket.input || binding["tls_digest"] != digest ||
		!providerReplayEqual(metadata, retained, meter) || m.response(contentString(binding["response_wire_digest"]), transport["response_headers"], response) != nil {
		return errProviderIntegration
	}
	return nil
}

// Bind from actual input objects, never the failed producer's bindings map.
func (m *providerHistoryRecoveryInitialModel) wireBinding(exchange string) (map[string]any, error) {
	meter := m.recovery.meter
	var selected map[string]any
	for i := uint64(0); i < m.through; i++ {
		for _, original := range m.prefix.source.frames[i].objects {
			if !meter.event(providerCompareEvent, uint64(len(original.kind)+len("wire-binding"))) {
				return nil, errProviderIntegration
			}
			if original.kind != "wire-binding" {
				continue
			}
			raw, err := m.object("wire-binding", original.digest)
			value := contentObject(providerReplayJSON(raw, meter), "format", "exchange_digest", "request_wire_digest", "response_wire_digest", "tls_digest")
			if err != nil || value == nil || !providerReplayTextEqual(contentString(value["exchange_digest"]), exchange, meter) || selected != nil {
				return nil, errProviderIntegration
			}
			selected = value
		}
	}
	if selected == nil {
		return nil, errProviderIntegration
	}
	return selected, nil
}

func (m *providerHistoryRecoveryInitialModel) response(digest string, headers any, body []byte) error {
	meter, capture := m.recovery.meter, m.recovery.config.capture
	if !capture.valid() {
		return errProviderIntegration
	}
	wanted := providerReplayCanonical(headers, 1<<20, meter)
	items, ok := headers.([]any)
	if !ok || len(items) == 0 || len(items) > capture.HeaderBytes/4 {
		return errProviderIntegration
	}
	raw, err := m.object("response", digest)
	if err != nil || len(raw) > capture.WireBytes {
		return errProviderIntegration
	}
	_, parsedHeaders, parsedBody, _, parseErr := readProviderHTTPObserved(bufio.NewReader(bytes.NewReader(raw)), capture, meter, uint64(len(raw)))
	if parseErr != nil || !providerReplayEqual(body, parsedBody, meter) || !meter.event(providerCopyEvent, uint64(len(parsedHeaders))) {
		return errProviderIntegration
	}
	values := make([]map[string]string, 0, len(parsedHeaders))
	for _, header := range parsedHeaders {
		values = append(values, map[string]string{"name": header.Name, "value": header.Value})
	}
	if !providerReplayEqual(wanted, providerReplayCanonical(values, 1<<20, meter), meter) {
		return errProviderIntegration
	}
	return nil
}

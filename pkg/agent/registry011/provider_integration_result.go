package registry011

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"strconv"
)

func (r *providerCustodyReplay) checkResult(s *providerCustodyStore, settlement *CandidateAdmissionSettlement) error {
	if settlement.ticket.kind == "rpc-capture" {
		if r.checkExchange(s, settlement) != nil {
			return errProviderIntegration
		}
		if settlement.ticket.sequence == 0 {
			if r.firstExchange != "" {
				return errProviderIntegration
			}
			r.firstExchange = settlement.retained
		}
		return nil
	}
	if r.start == "" && settlement.ticket.input == r.firstExchange && lowerHash(r.firstExchange) {
		if r.checkStart(s, settlement.retained) != nil {
			return errProviderIntegration
		}
		r.start, r.ledger.providerStart = settlement.retained, settlement.retained
		return nil
	}
	if _, exists := s.objects["pin-input:"+settlement.ticket.input]; exists {
		return r.checkPin(s, settlement)
	}
	// Other local result kinds must provide their exact closed reconstruction;
	// an existing hash, copied diagnostic or nil successful result is insufficient.
	return errProviderIntegration
}

func (r *providerCustodyReplay) checkExchange(s *providerCustodyStore, settlement *CandidateAdmissionSettlement) error {
	ticket := settlement.ticket
	raw, err := s.object("exchange", settlement.retained)
	m := contentObject(contentJSON(raw), "format", "policy", "namespace", "instance_key", "source_id", "generation", "ticket_digest", "provider_digest", "transport", "request", "response", "clock_start", "clock_end")
	id := r.intent.identity
	if err != nil || m == nil || m["format"] != "sage-provider-exchange-v2" || m["policy"] != id.Policy || m["namespace"] != id.Namespace || m["instance_key"] != id.Instance || m["source_id"] != id.SourceID || m["generation"] != strconv.FormatUint(id.Generation, 10) || m["ticket_digest"] != ticket.digest || m["provider_digest"] != id.Provider {
		return errProviderIntegration
	}
	p := contentJSON([]byte(r.policy.raw)).(map[string]any)
	start, a := candidateAdmissionClock(m["clock_start"], p["clock"].(map[string]any))
	end, b := candidateAdmissionClock(m["clock_end"], p["clock"].(map[string]any))
	consumption := r.ledger.tickets[ticket.digest].consumption
	if !a || !b || consumption == nil || !candidateLedgerClockAdvance(consumption.clock, start) || !candidateLedgerClockAdvance(start, end) || !candidateLedgerClockAdvance(end, settlement.clock) || candidateLedgerExpired(end, ticket.deadline) {
		return errProviderIntegration
	}
	request, err := s.object("request", ticket.input)
	parts := bytes.SplitN(request, []byte("\r\n\r\n"), 2)
	body, valid := providerCanonicalBase64(m["request"], 16<<10)
	response, responseValid := providerCanonicalBase64(m["response"], s.config.capture.BodyBytes)
	if err != nil || len(parts) != 2 || !valid || !responseValid || !bytes.Equal(parts[1], body) {
		return errProviderIntegration
	}
	if !s.event(providerCodecEvent, uint64(len(response))) {
		return errProviderIntegration
	}
	if _, err := DecodeRPCResult(response, 1, RPCResponseLimits{s.config.capture.BodyBytes, s.config.capture.JSONDepth}); err != nil {
		return errProviderIntegration
	}
	transport := contentObject(m["transport"], "endpoint", "method", "http_version", "status", "response_headers", "tls_version", "cipher_suite", "server_name", "verified_chain", "roots_artifact", "redirects", "proxy")
	if transport == nil || transport["endpoint"] != SepoliaRPCURL || transport["method"] != "POST" || transport["http_version"] != "HTTP/1.1" || transport["status"] != "200" || transport["server_name"] != "rpc.sepolia.ethpandaops.io" || transport["roots_artifact"] != p["provider"].(map[string]any)["tls_roots_artifact"] || transport["redirects"] != "0" || transport["proxy"] != "none" || !providerIntegrationCipher(transport) {
		return errProviderIntegration
	}
	chain, ok := transport["verified_chain"].([]any)
	if !ok || len(chain) == 0 || len(chain) > 16 {
		return errProviderIntegration
	}
	for _, cert := range chain {
		if !providerCandidateDigest(cert) {
			return errProviderIntegration
		}
	}
	metadata := providerIntegrationJSON(transport)
	digest, _ := providerIntegrationObjectDigest("tls", metadata)
	retainedMetadata, err := s.object("tls", digest)
	bindingBytes, bindingErr := s.object("wire-binding", s.bindings[settlement.retained])
	binding := contentObject(contentJSON(bindingBytes), "format", "exchange_digest", "request_wire_digest", "response_wire_digest", "tls_digest")
	if err != nil || bindingErr != nil || binding == nil || binding["exchange_digest"] != settlement.retained || binding["request_wire_digest"] != ticket.input || binding["tls_digest"] != digest || !bytes.Equal(metadata, retainedMetadata) || r.matchResponse(s, contentString(binding["response_wire_digest"]), transport["response_headers"], response) != nil {
		return errProviderIntegration
	}
	return nil
}

func providerCanonicalBase64(value any, maximum int) ([]byte, bool) {
	s, ok := value.(string)
	if !ok || maximum <= 0 || len(s) == 0 || len(s) > base64.RawURLEncoding.EncodedLen(maximum) {
		return nil, false
	}
	raw, err := base64.RawURLEncoding.DecodeString(s)
	return raw, err == nil && len(raw) <= maximum && base64.RawURLEncoding.EncodeToString(raw) == s
}

func providerIntegrationCipher(transport map[string]any) bool {
	version, a := providerCandidateUint(transport["tls_version"], true)
	cipher, b := providerCandidateUint(transport["cipher_suite"], true)
	if !a || !b {
		return false
	}
	if version == tls.VersionTLS13 {
		return cipher == tls.TLS_AES_128_GCM_SHA256 || cipher == tls.TLS_AES_256_GCM_SHA384 || cipher == tls.TLS_CHACHA20_POLY1305_SHA256
	}
	return version == tls.VersionTLS12 && (cipher == tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256 || cipher == tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256 || cipher == tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384 || cipher == tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384 || cipher == tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256 || cipher == tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256)
}

func (r *providerCustodyReplay) matchResponse(s *providerCustodyStore, digest string, headers any, body []byte) error {
	wanted := providerIntegrationJSON(headers)
	items, ok := headers.([]any)
	if !ok || len(items) == 0 || len(items) > s.config.capture.HeaderBytes/4 {
		return errProviderIntegration
	}
	raw, err := s.object("response", digest)
	if err != nil || len(raw) > s.config.capture.WireBytes {
		return errProviderIntegration
	}
	_, parsedHeaders, parsedBody, _, parseErr := readProviderHTTP(bufio.NewReader(bytes.NewReader(raw)), s.config.capture)
	if parseErr != nil || !bytes.Equal(body, parsedBody) {
		return errProviderIntegration
	}
	values := make([]map[string]string, 0, len(parsedHeaders))
	for _, header := range parsedHeaders {
		values = append(values, map[string]string{"name": header.Name, "value": header.Value})
	}
	if !bytes.Equal(wanted, providerIntegrationJSON(values)) {
		return errProviderIntegration
	}
	return nil
}

func (r *providerCustodyReplay) checkStart(s *providerCustodyStore, digest string) error {
	raw, err := s.object("bootstrap", digest)
	m := contentObject(contentJSON(raw), "format", "admission_digest", "policy", "namespace", "instance_key", "source_id", "generation", "bootstrap_id", "enrollment_anchor_root", "selected_finality_exchange", "selected_block_hash", "deployment_block_hash", "prior_published_checkpoint", "initial_clock", "deadline_wall_ms", "limits_digest")
	id := r.intent.identity
	i := contentJSON([]byte(r.intent.raw)).(map[string]any)
	p := contentJSON([]byte(r.policy.raw)).(map[string]any)
	if err != nil || m == nil || m["format"] != "sage-provider-start-v2" || m["admission_digest"] != r.intent.digest || m["policy"] != id.Policy || m["namespace"] != id.Namespace || m["instance_key"] != id.Instance || m["source_id"] != id.SourceID || m["generation"] != strconv.FormatUint(id.Generation, 10) || m["bootstrap_id"] != providerCandidateHash("sage-provider-bootstrap-id-v2", []byte(r.intent.digest)) || m["enrollment_anchor_root"] != r.anchor.digest || m["selected_finality_exchange"] != r.firstExchange || m["deployment_block_hash"] != p["instance"].(map[string]any)["deployment_hash"] || m["prior_published_checkpoint"] != nil || m["deadline_wall_ms"] != i["deadline_wall_ms"] || m["limits_digest"] != i["limits_digest"] || !bytes.Equal(providerIntegrationJSON(m["initial_clock"]), providerIntegrationJSON(i["initial_clock"])) {
		return errProviderIntegration
	}
	x, err := s.object("exchange", r.firstExchange)
	xObject, ok := contentJSON(x).(map[string]any)
	if err != nil || !ok {
		return errProviderIntegration
	}
	response, valid := providerCanonicalBase64(xObject["response"], s.config.capture.BodyBytes)
	if !valid {
		return errProviderIntegration
	}
	result, err := DecodeRPCResult(response, 1, RPCResponseLimits{s.config.capture.BodyBytes, s.config.capture.JSONDepth})
	var header struct{ Hash, ParentHash, Number, Timestamp string }
	if err != nil || json.Unmarshal(result, &header) != nil {
		return errProviderIntegration
	}
	number, a := rpcQuantity(header.Number)
	timestamp, b := rpcQuantity(header.Timestamp)
	if !a || !b || !validHeader(FinalizedHeader{header.Hash, header.ParentHash, number, timestamp}) || number < sepoliaDeploymentBlock || m["selected_block_hash"] != header.Hash {
		return errProviderIntegration
	}
	return nil
}

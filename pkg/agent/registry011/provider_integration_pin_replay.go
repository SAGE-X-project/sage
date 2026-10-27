package registry011

import (
	"bytes"
	"encoding/json"
)

func (r *providerCustodyReplay) pinInput(s *providerCustodyStore, digest string) (string, string, error) {
	raw, err := s.object("pin-input", digest)
	m := contentObject(contentJSON(raw), "format", "bootstrap_start", "registry", "agent")
	agent := contentString(m["agent"])
	if err != nil || r.start == "" || m == nil || !bytes.Equal(raw, providerPinInput(r.start, agent)) {
		return "", "", errProviderIntegration
	}
	start, err := s.object("bootstrap", r.start)
	b, valid := contentJSON(start).(map[string]any)
	if err != nil || !valid || !providerCandidateDigest(b["selected_block_hash"]) {
		return "", "", errProviderIntegration
	}
	return agent, contentString(b["selected_block_hash"]), nil
}

func (r *providerCustodyReplay) checkPin(s *providerCustodyStore, settlement *CandidateAdmissionSettlement) error {
	_, selected, err := r.pinInput(s, settlement.ticket.input)
	if err != nil {
		return err
	}
	raw, err := s.object("pin", settlement.retained)
	m := contentObject(contentJSON(raw), "format", "input", "selected_block_hash", "exchanges")
	if err != nil || m == nil || m["format"] != "sage-provider-owned-pin-v1" || m["input"] != settlement.ticket.input || m["selected_block_hash"] != selected {
		return errProviderIntegration
	}
	values, valid := m["exchanges"].([]any)
	if !valid || len(values) != 4 {
		return errProviderIntegration
	}
	var digests [4]string
	for i, value := range values {
		if !providerCandidateDigest(value) {
			return errProviderIntegration
		}
		digests[i] = contentString(value)
	}
	results, err := r.pinResults(s, digests, selected)
	if err != nil || !providerCheckPinResults(results) {
		return errProviderIntegration
	}
	return nil
}

func (r *providerCustodyReplay) pinResults(s *providerCustodyStore, digests [4]string, selected string) ([4]json.RawMessage, error) {
	var results [4]json.RawMessage
	requests, err := providerPinRequests(selected)
	if err != nil {
		return results, err
	}
	for i, digest := range digests {
		var err error
		results[i], err = r.completedExchangeResult(s, digest, requests[i])
		if err != nil {
			return results, err
		}
	}
	return results, nil
}

func (r *providerCustodyReplay) completedExchangeResult(s *providerCustodyStore, digest string, request *providerIntegrationRequest) (json.RawMessage, error) {
	x, err := s.object("exchange", digest)
	m, valid := contentJSON(x).(map[string]any)
	if err != nil || !valid || r.ledger == nil {
		return nil, errProviderIntegration
	}
	entry, exists := r.ledger.tickets[contentString(m["ticket_digest"])]
	if !exists || entry.ticket == nil || request == nil || entry.ticket.method != request.method || entry.ticket.params != request.paramsDigest || entry.ticket.input != request.inputDigest || entry.consumption == nil || entry.terminal == "" {
		return nil, errProviderIntegration
	}
	raw, err := s.object("settlement", entry.terminal)
	settled, parseErr := ParseCandidateAdmissionSettlement(entry.ticket, raw, 64<<10)
	if err != nil || parseErr != nil || settled.outcome != "complete" || settled.retained != digest {
		return nil, errProviderIntegration
	}
	response, valid := providerCanonicalBase64(m["response"], s.config.capture.BodyBytes)
	if !valid || !s.event(providerCodecEvent, uint64(len(response))) {
		return nil, errProviderIntegration
	}
	return DecodeRPCResult(response, 1, RPCResponseLimits{s.config.capture.BodyBytes, s.config.capture.JSONDepth})
}

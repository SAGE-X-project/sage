package registry011

import (
	"bytes"
	"encoding/json"
)

func (r *providerCustodyReplay) pinInput(s *providerCustodyStore, digest string) (string, string, error) {
	raw, err := s.object("pin-input", digest)
	if err != nil || !s.event(providerCodecEvent, uint64(len(raw))) {
		return "", "", errProviderIntegration
	}
	m := contentObject(contentJSON(raw), "format", "bootstrap_start", "registry", "agent")
	agent := contentString(m["agent"])
	expected := providerPinInput(r.start, agent)
	if r.start == "" || m == nil || !s.event(providerEncodeEvent, uint64(len(expected))) || !s.event(providerCompareEvent, uint64(len(raw)+len(expected))) || !bytes.Equal(raw, expected) {
		return "", "", errProviderIntegration
	}
	start, err := s.object("bootstrap", r.start)
	if err != nil || !s.event(providerCodecEvent, uint64(len(start))) {
		return "", "", errProviderIntegration
	}
	b, valid := contentJSON(start).(map[string]any)
	if !valid || !providerCandidateDigest(b["selected_block_hash"]) {
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
	if err != nil || !s.event(providerCodecEvent, uint64(len(raw))) {
		return errProviderIntegration
	}
	m := contentObject(contentJSON(raw), "format", "input", "selected_block_hash", "exchanges")
	if m == nil || m["format"] != "sage-provider-owned-pin-v1" || m["input"] != settlement.ticket.input || m["selected_block_hash"] != selected {
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
	if err != nil || !s.checkPinResults(results) {
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
	for _, request := range requests {
		// The expected request builder really encodes params/body and hashes
		// both inputs again. Retaining a prior request does not observe this work.
		if !s.event(providerEncodeEvent, uint64(len(request.params))) || !s.event(providerEncodeEvent, uint64(len(request.body))) || !s.event(providerCopyEvent, uint64(len(request.wire))) || !s.event(providerHashEvent, uint64(len(request.wire))) || !s.event(providerHashEvent, uint64(len("sage-provider-params-v1")+1+len(request.params))) {
			return results, errProviderIntegration
		}
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
	if err != nil || !s.event(providerCodecEvent, uint64(len(x))) {
		return nil, errProviderIntegration
	}
	m, valid := contentJSON(x).(map[string]any)
	if !valid || r.ledger == nil {
		return nil, errProviderIntegration
	}
	entry, exists := r.ledger.tickets[contentString(m["ticket_digest"])]
	if !exists || entry.ticket == nil || request == nil || entry.ticket.method != request.method || entry.ticket.params != request.paramsDigest || entry.ticket.input != request.inputDigest || entry.consumption == nil || entry.terminal == "" {
		return nil, errProviderIntegration
	}
	raw, err := s.object("settlement", entry.terminal)
	if err != nil {
		return nil, errProviderIntegration
	}
	settled, parseErr := s.parsePinSettlement(entry.ticket, raw)
	if parseErr != nil || settled.outcome != "complete" || settled.retained != digest {
		return nil, errProviderIntegration
	}
	encoded, valid := m["response"].(string)
	if !valid || !s.event(providerCodecEvent, uint64(len(encoded))) {
		return nil, errProviderIntegration
	}
	response, valid := providerCanonicalBase64(m["response"], s.config.capture.BodyBytes)
	if !valid || !s.event(providerCodecEvent, uint64(len(response))) {
		return nil, errProviderIntegration
	}
	return DecodeRPCResult(response, 1, RPCResponseLimits{s.config.capture.BodyBytes, s.config.capture.JSONDepth})
}

// The fixed settlement decoder reads TWO independently sized JSON inputs:
// the retained settlement and its exact ticket policy. It then hashes/copies
// the settlement. Failure leaves no usable complete measurement or authority.
func (s *providerCustodyStore) parsePinSettlement(ticket *CandidateAdmissionTicket, raw []byte) (*CandidateAdmissionSettlement, error) {
	if ticket == nil || ticket.raw == "" || len(raw) == 0 || len(raw) > 64<<10 {
		return nil, errProviderIntegration
	}
	if !s.event(providerCodecEvent, uint64(len(raw))) || !s.event(providerCodecEvent, uint64(len(ticket.policy))) || !s.event(providerHashEvent, uint64(len("sage-provider-budget-settlement-v2")+1+len(raw))) || !s.event(providerCopyEvent, uint64(len(raw))) {
		return nil, errProviderIntegration
	}
	settled, err := ParseCandidateAdmissionSettlement(ticket, raw, 64<<10)
	if err != nil {
		if meter := s.activeMeter.Load(); meter != nil {
			meter.refuse() // A failed parser may not have completed both decodes.
		}
		return nil, errProviderIntegration
	}
	return settled, nil // Candidate data only, never a measurement or work ACK.
}

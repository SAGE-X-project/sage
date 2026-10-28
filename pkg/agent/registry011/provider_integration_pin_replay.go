package registry011

import "encoding/json"

func (r *providerCustodyReplay) pinInput(s *providerCustodyStore, digest string) (string, string, error) {
	raw, err := s.object("pin-input", digest)
	if err != nil || r.meter == nil && !s.event(providerCodecEvent, uint64(len(raw))) {
		return "", "", errProviderIntegration
	}
	m := contentObject(providerReplayJSON(raw, r.meter), "format", "bootstrap_start", "registry", "agent")
	agent := contentString(m["agent"])
	expected := providerPinInputObserved(r.start, agent, r.meter)
	if r.start == "" || m == nil || r.meter == nil && !s.event(providerEncodeEvent, uint64(len(expected))) || r.meter == nil && !s.event(providerCompareEvent, uint64(len(raw)+len(expected))) || !providerReplayEqual(raw, expected, r.meter) {
		return "", "", errProviderIntegration
	}
	start, err := s.object("bootstrap", r.start)
	if err != nil || r.meter == nil && !s.event(providerCodecEvent, uint64(len(start))) {
		return "", "", errProviderIntegration
	}
	b, valid := providerReplayJSON(start, r.meter).(map[string]any)
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
	if err != nil || r.meter == nil && !s.event(providerCodecEvent, uint64(len(raw))) {
		return errProviderIntegration
	}
	m := contentObject(providerReplayJSON(raw, r.meter), "format", "input", "selected_block_hash", "exchanges")
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
	var results [4]json.RawMessage
	if s.config.bounds.guardian != nil {
		results, err = r.continuingPinResults(s, digests, selected)
	} else {
		results, err = r.pinResults(s, digests, selected)
	}
	if err != nil {
		return errProviderIntegration
	}
	if r.meter == nil {
		if !s.checkPinResults(results) {
			return errProviderIntegration
		}
	} else {
		for part, result := range results {
			if !providerCheckPinPartObserved(part, result, r.meter) {
				return errProviderIntegration
			}
		}
	}
	return nil
}

func (r *providerCustodyReplay) pinResults(s *providerCustodyStore, digests [4]string, selected string) ([4]json.RawMessage, error) {
	var results [4]json.RawMessage
	requests, err := providerPinRequestsObserved(selected, r.meter)
	if err != nil {
		return results, err
	}
	if r.meter == nil {
		for _, request := range requests {
			// The expected request builder really encodes params/body and hashes
			// both inputs again. Retaining a prior request does not observe this work.
			if !s.event(providerEncodeEvent, uint64(len(request.params))) || !s.event(providerEncodeEvent, uint64(len(request.body))) || !s.event(providerCopyEvent, uint64(len(request.wire))) || !s.event(providerHashEvent, uint64(len(request.wire))) || !s.event(providerHashEvent, uint64(len("sage-provider-params-v1")+1+len(request.params))) {
				return results, errProviderIntegration
			}
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
	if err != nil || r.meter == nil && !s.event(providerCodecEvent, uint64(len(x))) {
		return nil, errProviderIntegration
	}
	m, valid := providerReplayJSON(x, r.meter).(map[string]any)
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
	settled, parseErr := s.parsePinSettlementObserved(entry.ticket, raw, r.meter)
	if parseErr != nil || settled.outcome != "complete" || settled.retained != digest {
		return nil, errProviderIntegration
	}
	encoded, valid := m["response"].(string)
	if !valid || r.meter == nil && !s.event(providerCodecEvent, uint64(len(encoded))) {
		return nil, errProviderIntegration
	}
	response, valid := providerCanonicalBase64Observed(m["response"], s.config.capture.BodyBytes, r.meter)
	if !valid || r.meter == nil && !s.event(providerCodecEvent, uint64(len(response))) {
		return nil, errProviderIntegration
	}
	return decodeRPCResultObserved(response, 1, RPCResponseLimits{s.config.capture.BodyBytes, s.config.capture.JSONDepth}, r.meter)
}

// The fixed settlement decoder reads TWO independently sized JSON inputs:
// the retained settlement and its exact ticket policy. It then hashes/copies
// the settlement. Failure leaves no usable complete measurement or authority.
func (s *providerCustodyStore) parsePinSettlement(ticket *CandidateAdmissionTicket, raw []byte) (*CandidateAdmissionSettlement, error) {
	return s.parsePinSettlementObserved(ticket, raw, nil)
}

func (s *providerCustodyStore) parsePinSettlementObserved(ticket *CandidateAdmissionTicket, raw []byte, meter *providerWorkMeter) (*CandidateAdmissionSettlement, error) {
	if meter != nil {
		return parseCandidateAdmissionSettlementObserved(ticket, raw, 64<<10, meter)
	}
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

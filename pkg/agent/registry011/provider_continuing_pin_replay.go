package registry011

import "encoding/json"

// The four hashes are lookup DATA only. Each phase reconstructs the complete
// original X/S/transport/binding/class closure with its own current observer.
func (r *providerCustodyReplay) continuingPinResults(s *providerCustodyStore, digests [4]string, selected string) ([4]json.RawMessage, error) {
	var results [4]json.RawMessage
	if r == nil || s == nil || r.meter == nil || r.meter.closed || r.meter.failed || s.activeMeter.Load() != r.meter || r.ledger == nil || !lowerHash(selected) {
		return results, errProviderIntegration
	}
	for _, digest := range digests {
		if !lowerHash(digest) {
			return results, errProviderIntegration
		}
	}
	if !providerContinuingPinDistinct(digests) {
		return results, errProviderIntegration
	}
	requests, err := providerPinRequestsObserved(selected, r.meter)
	if err != nil {
		return results, err
	}
	for i, digest := range digests {
		settlement, err := r.continuingPinSettlement(s, digest, requests[i])
		if err != nil || r.checkExchange(s, settlement) != nil {
			return results, errProviderIntegration
		}
		results[i], err = r.completedExchangeResult(s, digest, requests[i])
		if err != nil {
			return results, errProviderIntegration
		}
	}
	return results, nil
}

func (r *providerCustodyReplay) continuingPinSettlement(s *providerCustodyStore, digest string, request *providerIntegrationRequest) (*CandidateAdmissionSettlement, error) {
	if r == nil || s == nil || r.meter == nil || s.activeMeter.Load() != r.meter || r.ledger == nil || request == nil || !lowerHash(digest) {
		return nil, errProviderIntegration
	}
	raw, err := s.object("exchange", digest)
	m, valid := providerReplayJSON(raw, r.meter).(map[string]any)
	if err != nil || !valid {
		return nil, errProviderIntegration
	}
	entry, exists := r.ledger.tickets[contentString(m["ticket_digest"])]
	if !exists || entry.ticket == nil || entry.ticket.kind != "rpc-capture" || entry.ticket.method != request.method || entry.ticket.input != request.inputDigest || entry.ticket.params != request.paramsDigest || entry.consumption == nil || !lowerHash(entry.terminal) {
		return nil, errProviderIntegration
	}
	raw, err = s.object("settlement", entry.terminal)
	if err != nil {
		return nil, errProviderIntegration
	}
	settlement, err := s.parsePinSettlementObserved(entry.ticket, raw, r.meter)
	if err != nil || settlement.outcome != "complete" || settlement.retained != digest {
		return nil, errProviderIntegration
	}
	return settlement, nil
}

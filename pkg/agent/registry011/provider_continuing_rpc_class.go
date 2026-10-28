package registry011

import "bytes"

// Derive the class again from the exact independently retained request.
// This returns finite DATA, never a live execution permit or Source.
func providerRetainedMethodCapture(s *providerCustodyStore, ticket *CandidateAdmissionTicket, wire []byte, meter *providerWorkMeter) (ProviderCaptureLimits, error) {
	var empty ProviderCaptureLimits
	if s == nil || ticket == nil || ticket.kind != "rpc-capture" || s.config.bounds.guardian == nil || s.config.bounds.methods == nil || len(wire) == 0 || len(wire) > 16<<10 {
		return empty, errProviderIntegration
	}
	parts := bytes.SplitN(wire, []byte("\r\n\r\n"), 2)
	if len(parts) != 2 {
		return empty, errProviderIntegration
	}
	var body struct {
		Version string `json:"jsonrpc"`
		ID      uint64 `json:"id"`
		Method  string `json:"method"`
		Params  any    `json:"params"`
	}
	if providerReplayUnmarshal(parts[1], &body, meter) != nil || body.Version != "2.0" || body.ID != 1 || body.Method != ticket.method {
		return empty, errProviderIntegration
	}
	params := providerRequestNumbersObserved(body.Params, meter)
	request, err := newProviderIntegrationRequestObserved(body.Method, params, meter)
	if err != nil || request.inputDigest != ticket.input || request.paramsDigest != ticket.params || !providerReplayEqual(wire, providerReplayPhaseBytes(request.wire, meter), meter) {
		return empty, errProviderIntegration
	}
	return s.config.bounds.methods.captureObserved(request, s.config.capture, meter)
}

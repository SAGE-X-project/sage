package registry011

// These are prospective size checks on the original reconstructed index, not
// a successful read or authority. Every ordinary read/hash/parse still follows.
func (r *providerCustodyReplay) continuingPinBoundOriginal(s *providerCustodyStore, kind, digest string, maximum uint64) bool {
	if r == nil || s == nil || r.meter == nil || r.meter.closed || r.meter.failed || s.activeMeter.Load() != r.meter || !lowerHash(digest) || maximum == 0 || maximum > 4<<20 {
		return false
	}
	switch kind {
	case "exchange", "settlement", "wire-binding", "tls", "response":
	default:
		return false
	}
	key, ok := providerReplayRefKey(providerIntegrationObjectRef{kind: kind, digest: digest}, r.meter)
	if !ok {
		return false
	}
	ref, exists := s.objects[key]
	return exists && ref.kind == kind && ref.digest == digest && ref.bytes != 0 && ref.bytes <= maximum && ref.bytes <= s.config.bounds.maximumObjectBytes
}

// Called only AFTER the independent first-X and first-S bounds, locator and
// original complete-S check. This helper bounds each remaining original BEFORE
// the unchanged full checker first reads it, under the same store mutex/meter.
func (r *providerCustodyReplay) continuingPinAncillaryBounds(s *providerCustodyStore, digest string, request *providerIntegrationRequest, exchange map[string]any, capture ProviderCaptureLimits, limits providerFirstOperationStorage) bool {
	if r == nil || s == nil || request == nil || exchange == nil || !capture.valid() {
		return false
	}
	bindingDigest := s.bindings[digest]
	if !r.continuingPinBoundOriginal(s, "wire-binding", bindingDigest, 8<<10) {
		return false
	}
	raw, err := s.object("wire-binding", bindingDigest)
	binding := contentObject(providerReplayJSON(raw, r.meter), "format", "exchange_digest", "request_wire_digest", "response_wire_digest", "tls_digest")
	if err != nil || binding == nil || binding["format"] != "sage-provider-wire-binding-v1" || binding["exchange_digest"] != digest || binding["request_wire_digest"] != request.inputDigest || !providerCandidateDigest(binding["response_wire_digest"]) || !providerCandidateDigest(binding["tls_digest"]) {
		return false
	}
	transport := contentObject(exchange["transport"], "endpoint", "method", "http_version", "status", "response_headers", "tls_version", "cipher_suite", "server_name", "verified_chain", "roots_artifact", "redirects", "proxy")
	if transport == nil {
		return false
	}
	metadata := providerReplayCanonical(transport, 1<<20, r.meter)
	tlsDigest, ok := providerIntegrationObjectDigestObserved("tls", metadata, r.meter)
	if len(metadata) == 0 || !ok || !providerReplayTextEqual(tlsDigest, contentString(binding["tls_digest"]), r.meter) {
		return false
	}
	return r.continuingPinBoundOriginal(s, "tls", tlsDigest, limits.transport) && r.continuingPinBoundOriginal(s, "response", contentString(binding["response_wire_digest"]), uint64(capture.WireBytes))
}

package registry011

import (
	"bufio"
	"bytes"
	"strconv"
)

const providerHistoryTLSOriginalDomain = "sage-provider-history-tls-v1"
const providerHistoryMemberResultDomain = "sage-provider-history-member-result-v1"

// These are reconstructed DATA views, never a protected result or proof of a
// live TLS connection. Their hashes are derived from exact retained originals;
// the future closed owner must bind actual capture and S before publication.
type providerHistoryMemberResult struct {
	digest, response, result string
}

func providerHistoryResultContextObserved(ticket *providerHistoryGroupTicket, consumption *providerHistoryGroupConsumption, ordinal int, meter *providerWorkMeter) bool {
	return providerHistoryGroupTicketValidObserved(ticket, meter) && providerHistoryGroupConsumptionValidObserved(consumption, ticket, meter) &&
		ordinal >= 0 && ordinal < ticket.table.count
}

// Only already captured diagnostic DATA is encoded here. In particular a
// valid JSON TLS record is not a live certificate/origin attestation. Its
// original source has to be bound by the concrete one-use capture owner.
func newProviderHistoryTLSOriginalObserved(ticket *providerHistoryGroupTicket, consumption *providerHistoryGroupConsumption, ordinal int, sample *ProviderTransportSample, roots string, bounds providerTLSBounds, meter *providerWorkMeter) (string, error) {
	if !providerHistoryResultContextObserved(ticket, consumption, ordinal, meter) || sample == nil || !sample.Complete || sample.CaptureError != "" ||
		sample.Endpoint != SepoliaRPCURL || sample.HTTPVersion != "HTTP/1.1" || sample.ServerName != "rpc.sepolia.ethpandaops.io" ||
		!lowerHash(roots) || !bounds.valid() || len(sample.VerifiedChains) == 0 || len(sample.VerifiedChains) > bounds.chains {
		return "", errProviderIntegration
	}
	request := &ticket.table.members[ordinal].request
	if sample.Method != request.method || !providerHistoryRequestBytesMatchObserved(sample, request, meter) {
		return "", errProviderIntegration
	}
	version, cipher := strconv.FormatUint(uint64(sample.TLSVersion), 10), strconv.FormatUint(uint64(sample.CipherSuite), 10)
	if !providerIntegrationCipher(map[string]any{"tls_version": version, "cipher_suite": cipher}) {
		return "", errProviderIntegration
	}
	chains := make([][]string, len(sample.VerifiedChains))
	for i, chain := range sample.VerifiedChains {
		if len(chain) == 0 || len(chain) > bounds.certificates {
			return "", errProviderIntegration
		}
		chains[i] = make([]string, len(chain))
		for j, digest := range chain {
			if !lowerHash(digest) || meter != nil && !meter.event(providerCopyEvent, uint64(len(digest))) {
				return "", errProviderIntegration
			}
			chains[i][j] = digest
		}
	}
	raw := providerReplayCanonical(map[string]any{
		"format": providerHistoryTLSOriginalDomain, "program": ticket.program, "stage": strconv.FormatUint(ticket.table.stage, 10),
		"table": ticket.table.digest, "ticket": ticket.digest, "consumption": consumption.digest, "ordinal": strconv.Itoa(ordinal),
		"request": request.inputDigest, "endpoint": SepoliaRPCURL, "http_version": "HTTP/1.1", "server_name": sample.ServerName,
		"tls_version": version, "cipher_suite": cipher, "verified_chains": chains, "roots_artifact": roots,
	}, 64<<10, meter)
	if len(raw) == 0 {
		return "", errProviderIntegration
	}
	original := providerReplayString(raw, meter)
	if original == "" {
		return "", errProviderIntegration
	}
	return original, nil
}

// A closed byte primitive, not T/C validation or origin authority. Unlike the
// legacy phase helpers, every nonnil DATA observer pays for copies and
// comparisons even when it has no V4 store attached.
func providerHistoryRequestBytesMatchObserved(sample *ProviderTransportSample, request *providerIntegrationRequest, meter *providerWorkMeter) bool {
	if sample == nil || request == nil || len(request.head) == 0 || len(request.head) > 512 || len(request.body) == 0 || len(request.body) > 16<<10 ||
		len(sample.RequestHead) != len(request.head) || len(sample.RequestBody) != len(request.body) {
		return false
	}
	return providerReplayEqual(sample.RequestHead, providerReplayBytes(request.head, meter), meter) &&
		providerReplayEqual(sample.RequestBody, providerReplayBytes(request.body, meter), meter)
}

func parseProviderHistoryTLSOriginalObserved(raw []byte, ticket *providerHistoryGroupTicket, consumption *providerHistoryGroupConsumption, ordinal int, roots string, bounds providerTLSBounds, meter *providerWorkMeter) error {
	if len(raw) == 0 || len(raw) > 64<<10 || !providerHistoryResultContextObserved(ticket, consumption, ordinal, meter) || !bounds.valid() || !lowerHash(roots) {
		return errProviderIntegration
	}
	v := contentObject(providerReplayJSON(raw, meter), "format", "program", "stage", "table", "ticket", "consumption", "ordinal", "request", "endpoint", "http_version", "server_name", "tls_version", "cipher_suite", "verified_chains", "roots_artifact")
	if v == nil || v["format"] != providerHistoryTLSOriginalDomain || v["program"] != ticket.program || v["stage"] != strconv.FormatUint(ticket.table.stage, 10) ||
		v["table"] != ticket.table.digest || v["ticket"] != ticket.digest || v["consumption"] != consumption.digest || v["ordinal"] != strconv.Itoa(ordinal) ||
		v["request"] != ticket.table.members[ordinal].request.inputDigest || v["endpoint"] != SepoliaRPCURL || v["http_version"] != "HTTP/1.1" ||
		v["server_name"] != "rpc.sepolia.ethpandaops.io" || v["roots_artifact"] != roots || !providerIntegrationCipher(v) {
		return errProviderIntegration
	}
	chains, ok := v["verified_chains"].([]any)
	if !ok || len(chains) == 0 || len(chains) > bounds.chains {
		return errProviderIntegration
	}
	for _, item := range chains {
		chain, ok := item.([]any)
		if !ok || len(chain) == 0 || len(chain) > bounds.certificates {
			return errProviderIntegration
		}
		for _, digest := range chain {
			if !providerCandidateDigest(digest) {
				return errProviderIntegration
			}
		}
	}
	if !providerReplayEqual(providerReplayCanonical(v, 64<<10, meter), raw, meter) {
		return errProviderIntegration
	}
	return nil
}

// The selected inventory supplies expected, and T supplies every request,
// class, and the enclosing S replay selects a completed prefix count. This
// accepts no decoded producer result or reported digest.
// It verifies HTTP/RPC framing and bindings; later program-specific checks
// must still validate such facts as chain ID, selected block and call meaning.
func parseProviderHistoryResultPackObserved(raw []byte, expected string, ticket *providerHistoryGroupTicket, consumption *providerHistoryGroupConsumption, count int, roots string, bounds providerTLSBounds, maximum uint64, meter *providerWorkMeter) (*providerHistoryOriginalPack, [providerHistoryGroupMembers]providerHistoryMemberResult, error) {
	var results [providerHistoryGroupMembers]providerHistoryMemberResult
	if !providerHistoryResultContextObserved(ticket, consumption, 0, meter) || count <= 0 || count > ticket.table.count {
		return nil, results, errProviderIntegration
	}
	pack, err := parseProviderHistoryOriginalPackObserved(raw, expected, ticket.table.stage, ticket.table.digest, "results", 0, count, maximum, meter)
	if err != nil {
		return nil, results, errProviderIntegration
	}
	for i := range count {
		_, response, responseOK := pack.original(2 * i)
		_, tlsOriginal, tlsOK := pack.original(2*i + 1)
		member := &ticket.table.members[i]
		if !responseOK || !tlsOK || len(response) == 0 || len(response) > member.capture.WireBytes-len(member.request.wire) ||
			parseProviderHistoryTLSOriginalObserved(providerReplayBytes(tlsOriginal, meter), ticket, consumption, i, roots, bounds, meter) != nil {
			return nil, [providerHistoryGroupMembers]providerHistoryMemberResult{}, errProviderIntegration
		}
		wire := providerReplayBytes(response, meter)
		if len(wire) != len(response) {
			return nil, [providerHistoryGroupMembers]providerHistoryMemberResult{}, errProviderIntegration
		}
		reader := bufio.NewReader(bytes.NewReader(wire))
		_, _, body, result, readErr := readProviderHTTPObserved(reader, member.capture, meter, uint64(len(wire)))
		if readErr != nil {
			return nil, [providerHistoryGroupMembers]providerHistoryMemberResult{}, errProviderIntegration
		}
		responseDigest := providerReplayDigest("", wire, meter)
		tlsDigest := providerReplayDigest("", providerReplayBytes(tlsOriginal, meter), meter)
		binding := providerReplayCanonical(map[string]any{
			"format": providerHistoryMemberResultDomain, "program": ticket.program, "stage": strconv.FormatUint(ticket.table.stage, 10),
			"table": ticket.table.digest, "ticket": ticket.digest, "consumption": consumption.digest, "ordinal": strconv.Itoa(i),
			"request": member.request.inputDigest, "response": responseDigest, "tls": tlsDigest,
		}, candidateLedgerMetadataLimit, meter)
		if responseDigest == "" || tlsDigest == "" || len(binding) == 0 {
			return nil, [providerHistoryGroupMembers]providerHistoryMemberResult{}, errProviderIntegration
		}
		digest := providerReplayDigest(providerHistoryMemberResultDomain, binding, meter)
		responseBody, rpcResult := providerReplayString(body, meter), providerReplayString(result, meter)
		if digest == "" || responseBody == "" || rpcResult == "" {
			return nil, [providerHistoryGroupMembers]providerHistoryMemberResult{}, errProviderIntegration
		}
		results[i] = providerHistoryMemberResult{digest, responseBody, rpcResult}
	}
	return pack, results, nil
}

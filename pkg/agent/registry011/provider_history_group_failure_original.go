package registry011

import (
	"encoding/binary"
	"strconv"
)

const providerHistoryFailureDomain = "sage-provider-history-member-failure-v1"
const providerHistoryFailureMagic = providerHistoryFailureDomain + "\n"
const providerHistoryFailureHeaderBytes = 16 << 10

// Closed failure DATA. Only the concrete original attempt may select it for
// a failed S. Encoding/parsing this object grants no execution or ACK role.
// Binary payloads preserve partial HTTP bytes verbatim, including empty ones.
type providerHistoryFailureOriginal struct {
	stage                      string
	ordinal, written, received int
	raw, digest                string
}

func (*providerHistoryFailureOriginal) UnmarshalJSON([]byte) error { return errProviderIntegration }

// Arithmetic framing only, for physical payload/metadata suballocation. This
// does not parse the header, authenticate the failure or accept a live role.
func providerHistoryFailureLayout(raw string) (uint64, uint64, bool) {
	start := len(providerHistoryFailureMagic) + 4
	if len(raw) < start || len(raw) > providerHistoryOriginalPackBytes || raw[:len(providerHistoryFailureMagic)] != providerHistoryFailureMagic {
		return 0, 0, false
	}
	i := len(providerHistoryFailureMagic)
	header := uint64(raw[i])<<24 | uint64(raw[i+1])<<16 | uint64(raw[i+2])<<8 | uint64(raw[i+3])
	metadata, ok := providerUpperSum(uint64(start), header)
	if !ok || header == 0 || header > providerHistoryFailureHeaderBytes || metadata > uint64(len(raw)) {
		return 0, 0, false
	}
	return metadata, uint64(len(raw)) - metadata, true
}

// A whole failure representation must fit BEFORE a dial. It is distinct from
// the maximum successful whole-result pack. Every arithmetic operation is
// checked; no request table is shortened to fit a later failure record.
func (d *providerHistoryGroupDispatch) failureOriginalFitsLocked() bool {
	if d == nil || d.group == nil || d.group.store == nil || d.coordinator == nil || d.coordinator.ticket == nil || d.order == nil ||
		d.order.count <= 0 || d.order.count > providerHistoryGroupMembers || d.coordinator.ticket.table.count != d.order.count {
		return false
	}
	if d.group.parent == nil || !d.group.parent.tls.valid() {
		return false
	}
	tls := d.group.parent.tls
	certificates, a := providerUpperProduct(uint64(tls.chains), uint64(tls.certificates))
	digests, b := providerUpperProduct(certificates, 70)
	punctuation, c := providerUpperProduct(uint64(tls.chains), 4)
	header, e := providerUpperSum(4096, digests, punctuation)
	if !a || !b || !c || !e || header > providerHistoryFailureHeaderBytes {
		return false
	}
	maximum := min(d.group.store.config.bounds.maximumObjectBytes, uint64(providerHistoryOriginalPackBytes))
	for i := range d.order.count {
		member := d.coordinator.ticket.table.members[i]
		// Written request + retained read (including rejected extra byte) is
		// at most the original HTTP wire allowance plus its funded sentinel.
		total, ok := providerUpperSum(uint64(len(providerHistoryFailureMagic)+4+providerHistoryFailureHeaderBytes), uint64(member.capture.WireBytes), 1)
		if !member.capture.valid() || !ok || total > maximum {
			return false
		}
	}
	return true
}

func providerHistoryFailureTLSObserved(sample *ProviderTransportSample, request *providerIntegrationRequest, bounds providerTLSBounds, meter *providerWorkMeter) (any, bool) {
	if sample == nil || request == nil || !bounds.valid() || sample.Complete || sample.Endpoint != SepoliaRPCURL || sample.Method != request.method ||
		sample.HTTPVersion != "HTTP/1.1" || sample.ServerName != "rpc.sepolia.ethpandaops.io" ||
		!providerHistoryRequestBytesMatchObserved(sample, request, meter) || len(sample.VerifiedChains) == 0 || len(sample.VerifiedChains) > bounds.chains {
		return nil, false
	}
	version, cipher := strconv.FormatUint(uint64(sample.TLSVersion), 10), strconv.FormatUint(uint64(sample.CipherSuite), 10)
	if !providerIntegrationCipher(map[string]any{"tls_version": version, "cipher_suite": cipher}) {
		return nil, false
	}
	chains := make([][]string, len(sample.VerifiedChains))
	for i, chain := range sample.VerifiedChains {
		if len(chain) == 0 || len(chain) > bounds.certificates {
			return nil, false
		}
		chains[i] = make([]string, len(chain))
		for j, digest := range chain {
			if !lowerHash(digest) || meter != nil && !meter.event(providerCopyEvent, uint64(len(digest))) {
				return nil, false
			}
			chains[i][j] = digest
		}
	}
	return map[string]any{"server_name": sample.ServerName, "tls_version": version, "cipher_suite": cipher, "verified_chains": chains}, true
}

func newProviderHistoryFailureOriginalObserved(ticket *providerHistoryGroupTicket, consumption *providerHistoryGroupConsumption, ordinal int, data *providerHistoryCaptureData, roots string, bounds providerTLSBounds, maximum uint64, meter *providerWorkMeter) (*providerHistoryFailureOriginal, error) {
	if !providerHistoryResultContextObserved(ticket, consumption, ordinal, meter) || data == nil || !data.knownFailure || data.uncertain || !lowerHash(roots) || !bounds.valid() ||
		maximum == 0 || maximum > providerHistoryOriginalPackBytes {
		return nil, errProviderIntegration
	}
	member := ticket.table.members[ordinal]
	written, received := data.requestWritten, data.responseRead
	if written < 0 || written > len(member.request.wire) || received < 0 || received > member.capture.WireBytes-written+1 {
		return nil, errProviderIntegration
	}
	var tls any
	var response []byte
	switch data.failureStage {
	case "dial":
		if data.sample != nil || written != 0 || received != 0 {
			return nil, errProviderIntegration
		}
	case "write", "read", "framing":
		if data.sample == nil || data.sample.Complete || received != len(data.sample.ResponseWire) {
			return nil, errProviderIntegration
		}
		if data.failureStage == "write" && received != 0 || data.failureStage != "write" && written != len(member.request.wire) ||
			data.failureStage == "framing" && (received == 0 || received > member.capture.WireBytes-written) {
			return nil, errProviderIntegration
		}
		var ok bool
		tls, ok = providerHistoryFailureTLSObserved(data.sample, &member.request, bounds, meter)
		if !ok {
			return nil, errProviderIntegration
		}
		response = data.sample.ResponseWire
	default:
		return nil, errProviderIntegration
	}
	prefix := providerReplayBytes(member.request.wire[:written], meter)
	requestHash, responseHash := providerReplayDigest("", prefix, meter), providerReplayDigest("", response, meter)
	if requestHash == "" || responseHash == "" {
		return nil, errProviderIntegration
	}
	sent, _ := providerHistoryMemberPrefix(ordinal + 1)
	header := providerReplayCanonical(map[string]any{
		"format": providerHistoryFailureDomain, "program": ticket.program, "stage": strconv.FormatUint(ticket.table.stage, 10), "table": ticket.table.digest,
		"ticket": ticket.digest, "consumption": consumption.digest, "ordinal": strconv.Itoa(ordinal), "request": member.request.inputDigest,
		"failure_stage": data.failureStage, "consumed": strconv.FormatUint(uint64(consumption.consumed), 10), "sent": strconv.FormatUint(uint64(sent), 10), "completed": strconv.Itoa(ordinal),
		"request_written": strconv.Itoa(written), "response_read": strconv.Itoa(received), "request_prefix": requestHash, "response_prefix": responseHash,
		"endpoint": SepoliaRPCURL, "http_version": "HTTP/1.1", "method": member.request.method, "roots_artifact": roots, "tls": tls,
	}, providerHistoryFailureHeaderBytes, meter)
	total, ok := providerUpperSum(uint64(len(providerHistoryFailureMagic)+4), uint64(len(header)), uint64(written), uint64(received))
	if len(header) == 0 || !ok || total > maximum {
		return nil, errProviderIntegration
	}
	raw := make([]byte, int(total))
	start := len(providerHistoryFailureMagic) + 4
	if !providerHistoryOriginalCopy(raw[:len(providerHistoryFailureMagic)], providerHistoryFailureMagic, meter) || meter != nil && !meter.event(providerEncodeEvent, 4) {
		return nil, errProviderIntegration
	}
	binary.BigEndian.PutUint32(raw[len(providerHistoryFailureMagic):start], uint32(len(header)))
	if !providerHistoryOriginalCopy(raw[start:start+len(header)], providerReplayString(header, meter), meter) {
		return nil, errProviderIntegration
	}
	start += len(header)
	if !providerHistoryOriginalCopy(raw[start:start+written], providerReplayString(prefix, meter), meter) {
		return nil, errProviderIntegration
	}
	start += written
	if !providerHistoryOriginalCopy(raw[start:], providerReplayString(response, meter), meter) {
		return nil, errProviderIntegration
	}
	result := &providerHistoryFailureOriginal{stage: data.failureStage, ordinal: ordinal, written: written, received: received, raw: providerReplayString(raw, meter), digest: providerReplayDigest(providerHistoryFailureDomain, raw, meter)}
	if result.raw == "" || result.digest == "" {
		return nil, errProviderIntegration
	}
	return result, nil
}

func parseProviderHistoryFailureOriginalObserved(raw []byte, expected string, ticket *providerHistoryGroupTicket, consumption *providerHistoryGroupConsumption, ordinal int, roots string, bounds providerTLSBounds, maximum uint64, meter *providerWorkMeter) (*providerHistoryFailureOriginal, error) {
	start := len(providerHistoryFailureMagic) + 4
	if maximum == 0 || maximum > providerHistoryOriginalPackBytes || len(raw) < start || uint64(len(raw)) > maximum || !lowerHash(expected) ||
		!providerHistoryResultContextObserved(ticket, consumption, ordinal, meter) {
		return nil, errProviderIntegration
	}
	// Storage layout arithmetic is not an observed parser. Fund the exact
	// magic copy/comparison and four-byte decode on this original observer
	// before examining either framing field, including rejected input.
	if !providerReplayTextEqual(providerReplayString(raw[:len(providerHistoryFailureMagic)], meter), providerHistoryFailureMagic, meter) ||
		meter != nil && !meter.event(providerCodecEvent, 4) {
		return nil, errProviderIntegration
	}
	header := uint64(binary.BigEndian.Uint32(raw[len(providerHistoryFailureMagic):start]))
	if header == 0 || header > providerHistoryFailureHeaderBytes || header > uint64(len(raw)-start) {
		return nil, errProviderIntegration
	}
	metadata := uint64(start) + header
	v := contentObject(providerReplayJSON(raw[start:int(metadata)], meter), "format", "program", "stage", "table", "ticket", "consumption", "ordinal", "request", "failure_stage", "consumed", "sent", "completed", "request_written", "response_read", "request_prefix", "response_prefix", "endpoint", "http_version", "method", "roots_artifact", "tls")
	written, a := providerCandidateUint(v["request_written"], false)
	received, b := providerCandidateUint(v["response_read"], false)
	total, c := providerUpperSum(metadata, written, received)
	member := ticket.table.members[ordinal]
	if v == nil || !a || !b || !c || total != uint64(len(raw)) || written > uint64(len(member.request.wire)) || received > uint64(member.capture.WireBytes)+1 {
		return nil, errProviderIntegration
	}
	prefix := raw[int(metadata):int(metadata+written)]
	if !providerReplayEqual(prefix, providerReplayBytes(member.request.wire[:int(written)], meter), meter) {
		return nil, errProviderIntegration
	}
	data := &providerHistoryCaptureData{failureStage: contentString(v["failure_stage"]), requestWritten: int(written), responseRead: int(received), knownFailure: true}
	if data.failureStage != "dial" {
		tls := contentObject(v["tls"], "server_name", "tls_version", "cipher_suite", "verified_chains")
		version, e := providerCandidateUint(tls["tls_version"], true)
		cipher, f := providerCandidateUint(tls["cipher_suite"], true)
		chains, g := tls["verified_chains"].([]any)
		if tls == nil || !e || !f || !g || version > 65535 || cipher > 65535 || !bounds.valid() || len(chains) == 0 || len(chains) > bounds.chains {
			return nil, errProviderIntegration
		}
		parsed := make([][]string, len(chains))
		for i, item := range chains {
			chain, ok := item.([]any)
			if !ok || len(chain) == 0 || len(chain) > bounds.certificates {
				return nil, errProviderIntegration
			}
			parsed[i] = make([]string, len(chain))
			for j, value := range chain {
				digest, ok := value.(string)
				if !ok || !lowerHash(digest) {
					return nil, errProviderIntegration
				}
				parsed[i][j] = digest
			}
		}
		data.sample = &ProviderTransportSample{Endpoint: SepoliaRPCURL, Method: member.request.method, HTTPVersion: "HTTP/1.1", ServerName: contentString(tls["server_name"]), TLSVersion: uint16(version), CipherSuite: uint16(cipher), VerifiedChains: parsed,
			RequestHead: providerReplayBytes(member.request.head, meter), RequestBody: providerReplayBytes(member.request.body, meter), ResponseWire: raw[int(metadata+written):]}
	}
	rebuilt, err := newProviderHistoryFailureOriginalObserved(ticket, consumption, ordinal, data, roots, bounds, maximum, meter)
	if err != nil || !providerReplayTextEqual(rebuilt.digest, expected, meter) || !providerReplayEqual(raw, providerReplayBytes(rebuilt.raw, meter), meter) {
		return nil, errProviderIntegration
	}
	return rebuilt, nil
}

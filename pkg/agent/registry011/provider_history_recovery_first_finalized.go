package registry011

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"unicode/utf8"
)

// This is an explicit additional DATA derivation from this pass's completed
// original first P. General transport-only UNIT profiles remain valid without
// invoking it. It neither starts a collector nor turns a decoded header into
// a Source, a new selection allowance, an ACK or an execution role. The closed
// pre-H0 collection manifest must still bound the header before collection.
func (m *providerHistoryRecoveryInitialModel) selectFirstFinalizedCandidate(ctx context.Context) (FinalizedHeader, error) {
	if m == nil || m.recovery == nil || m.recovery.store == nil {
		return FinalizedHeader{}, errProviderIntegration
	}
	r := m.recovery
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	if !r.activeLocked(r.meter) || !m.selectedLocked(r) || m.finalizedSelected ||
		m.prefix.pass == 1 && (r.initialModels[0] == nil || !r.initialModels[0].finalizedSelected) {
		return FinalizedHeader{}, errProviderIntegration
	}
	accepted := false
	defer func() {
		if !accepted {
			r.stopLocked(true)
		}
	}()
	if !r.inputReadyLocked(ctx) {
		return FinalizedHeader{}, errProviderIntegration
	}
	body, n := m.frontier.body, r.initialDescriptor
	if !providerReplayTextEqual(body.firstExchange, n.exchange, r.meter) {
		return FinalizedHeader{}, errProviderIntegration
	}
	expected, err := newProviderIntegrationRequestObserved("eth_getBlockByNumber", []any{"finalized", false}, r.meter)
	if err != nil {
		return FinalizedHeader{}, errProviderIntegration
	}
	raw := m.firstFinalizedOriginalObserved("exchange", n.exchange)
	x := contentObject(providerReplayJSON(raw, r.meter), "format", "policy", "namespace", "instance_key", "source_id", "generation", "ticket_digest", "provider_digest", "transport", "request", "response", "clock_start", "clock_end")
	if len(raw) == 0 || x == nil || x["format"] != "sage-provider-exchange-v2" || x["ticket_digest"] != n.ticket {
		return FinalizedHeader{}, errProviderIntegration
	}
	entry, exists := body.ledger.tickets[n.ticket]
	if !exists || entry.ticket == nil || entry.ticket.digest != n.ticket || entry.ticket.kind != "rpc-capture" || entry.ticket.sequence != 0 ||
		entry.ticket.method != expected.method || entry.ticket.params != expected.paramsDigest || entry.ticket.input != expected.inputDigest ||
		entry.consumption == nil || !lowerHash(entry.terminal) || entry.terminal != m.prefix.records[3].operation {
		return FinalizedHeader{}, errProviderIntegration
	}
	request := m.firstFinalizedOriginalObserved("request", entry.ticket.input)
	settled := m.firstFinalizedOriginalObserved("settlement", entry.terminal)
	settlement, parseErr := parseCandidateAdmissionSettlementObserved(entry.ticket, settled, 64<<10, r.meter)
	if !providerReplayPhaseRawTextEqual(request, expected.wire, r.meter) || parseErr != nil || settlement.outcome != "complete" || settlement.retained != n.exchange {
		return FinalizedHeader{}, errProviderIntegration
	}
	// The full initial model already independently checked the actual original
	// HTTP/TLS/request/consumption/S closure. These exact rehashed originals come
	// from its own immutable frames, never store.objects/bindings/frontierReplay
	// or the old live firstResult. Re-decoding X's identical envelope adds no trust.
	response, ok := providerCanonicalBase64Observed(x["response"], r.config.capture.BodyBytes, r.meter)
	if !ok {
		return FinalizedHeader{}, errProviderIntegration
	}
	result, err := decodeRPCResultObserved(response, 1, RPCResponseLimits{r.config.capture.BodyBytes, r.config.capture.JSONDepth}, r.meter)
	selected, parseErr := providerHistoryFirstFinalizedHeaderObserved(result, RPCResponseLimits{r.config.capture.BodyBytes, r.config.capture.JSONDepth}, r.meter)
	if err != nil || parseErr != nil || m.prefix.pass == 1 && selected != r.initialModels[0].firstFinalized ||
		r.parent.bounds.collection != nil && !r.parent.bounds.collection.selectedObserved(selected, r.meter) || !r.inputReadyLocked(ctx) {
		return FinalizedHeader{}, errProviderIntegration
	}
	m.firstFinalized, m.finalizedSelected, accepted = selected, true, true
	return selected, nil
}

// Only three named dependencies of the first P are selected, from the actual
// current pass's original frame closure. Hashing again binds any interpreted
// bytes to their original reference. This adds no filesystem or live role.
func (m *providerHistoryRecoveryInitialModel) firstFinalizedOriginalObserved(kind, digest string) []byte {
	if m == nil || m.recovery == nil || !m.selectedLocked(m.recovery) || !m.recovery.activeLocked(m.recovery.meter) || !lowerHash(digest) {
		return nil
	}
	switch kind {
	case "exchange", "request", "settlement":
	default:
		return nil
	}
	raw := m.prefix.source.frameObjectObserved(kind, digest, 4)
	actual, ok := providerIntegrationObjectDigestObserved(kind, raw, m.recovery.meter)
	if len(raw) == 0 || !ok || !providerReplayTextEqual(actual, digest, m.recovery.meter) {
		return nil
	}
	return raw
}

// Header DATA is not a new finalized observation or a consensus credential.
// The caller must select this result from its genuine retained first P and
// additionally enforce the original manifest's inclusive interval/maxima.
func providerHistoryFirstFinalizedHeaderObserved(raw []byte, limits RPCResponseLimits, meter *providerWorkMeter) (FinalizedHeader, error) {
	header, err := providerHistoryRPCHeaderObserved(raw, limits, meter)
	if err != nil || header.Number < sepoliaDeploymentBlock {
		return FinalizedHeader{}, errProviderIntegration
	}
	return header, nil
}

// Genesis and numeric ancestry use the same strict field/token/quantity parser.
// Only the original first-P wrapper additionally requires deployment or later.
func providerHistoryRPCHeaderObserved(raw []byte, limits RPCResponseLimits, meter *providerWorkMeter) (FinalizedHeader, error) {
	var h struct{ Hash, ParentHash, Number, Timestamp string }
	if limits.MaxBytes <= 0 || limits.MaxBytes > 8<<20 || limits.MaxDepth <= 0 || limits.MaxDepth > 128 ||
		len(raw) == 0 || len(raw) > limits.MaxBytes || meter == nil ||
		!meter.event(providerCompareEvent, uint64(len(raw))) || !utf8.Valid(raw) ||
		!meter.event(providerCompareEvent, uint64(len(raw))) || !validRPCUnicode(raw) || !meter.event(providerCodecEvent, uint64(len(raw))) {
		return FinalizedHeader{}, errProviderIntegration
	}
	// Preserve arbitrary ordinary RPC object field order. Strict token parsing
	// rejects duplicate names/ambiguous input; canonical metadata's field order
	// is not imposed on a provider's original JSON header.
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if !readRPCValue(decoder, 0, limits.MaxDepth) {
		return FinalizedHeader{}, errProviderIntegration
	}
	var fields map[string]json.RawMessage
	if _, err := decoder.Token(); err != io.EOF || providerReplayUnmarshal(raw, &fields, meter) != nil ||
		providerReplayUnmarshal(fields["hash"], &h.Hash, meter) != nil ||
		providerReplayUnmarshal(fields["parentHash"], &h.ParentHash, meter) != nil ||
		providerReplayUnmarshal(fields["number"], &h.Number, meter) != nil ||
		providerReplayUnmarshal(fields["timestamp"], &h.Timestamp, meter) != nil {
		return FinalizedHeader{}, errProviderIntegration
	}
	number, a := rpcQuantityObserved(h.Number, meter)
	stamp, b := rpcQuantityObserved(h.Timestamp, meter)
	header := FinalizedHeader{h.Hash, h.ParentHash, number, stamp}
	if !a || !b || !validHeader(header) ||
		!meter.event(providerCompareEvent, uint64(len(h.Hash)+len(h.ParentHash)+len(h.Number)+len(h.Timestamp))) {
		return FinalizedHeader{}, errProviderIntegration
	}
	return header, nil
}

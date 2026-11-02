package registry011

import (
	"bufio"
	"bytes"
	"strconv"
)

// Each invocation owns a fresh closure of retained originals. No producer map,
// cached replay, callback, old firstResult or restored execution role can
// supply bytes. The original live owner/observer remains the only reader.
type providerHistoryFirstPOriginals struct {
	group   *providerHistoryGroupOwner
	bounds  providerHistoryFirstPReadBounds
	raw     uint64
	objects []providerIntegrationObject
}

func (o *providerHistoryGroupOwner) firstFinalizedInitialRootLocked(current providerFrontierRecord) (string, *providerHistoryCollectionCursor, error) {
	if o == nil || !o.activeLocked(o.meter) || o.firstPReading || o.parent.bounds.collection == nil ||
		!o.parent.collectionMatchesLocked() || o.start.main.sequence != 4 {
		return "", nil, errProviderIntegration
	}
	b := o.parent.collectionInput.firstRead
	if !b.valid() || b.objects > uint64(o.store.config.bounds.maximumObjects) || b.raw > o.store.config.bounds.maximumStoredBytes {
		return "", nil, errProviderIntegration
	}
	o.firstPReading, o.firstPStart = true, o.meter.actual
	defer func() { o.firstPReading = false }()
	if !o.meter.preflightWork(b.work) || !o.identityLocked() {
		return "", nil, errProviderIntegration
	}
	input := &providerHistoryFirstPOriginals{group: o, bounds: b}
	if input.load() != nil {
		return "", nil, errProviderIntegration
	}
	selected, err := input.selected(current)
	if err != nil || !o.parent.bounds.collection.selectedObserved(selected, o.meter) || !o.identityLocked() {
		return "", nil, errProviderIntegration
	}
	root := providerHistoryInitialRootObserved(o.parent.bounds, o.store.config.profile, o.start.main, o.start.exchange, current.budget, current.objects, &selected, o.meter)
	cursor, cursorErr := newProviderHistoryCollectionCursorObserved(o.parent.bounds.program, o.parent.bounds.collection, selected, o.meter)
	if root == "" || cursorErr != nil {
		return "", nil, errProviderIntegration
	}
	return root, cursor, nil
}

func (p *providerHistoryFirstPOriginals) active() bool {
	return p != nil && p.group != nil && p.group.firstPReading && p.group.activeLocked(p.group.meter) &&
		p.group.parent.collectionMatchesLocked() && p.bounds == p.group.parent.collectionInput.firstRead
}

func (p *providerHistoryFirstPOriginals) read(disk *retentionDisk, name string, maximum uint64) ([]byte, error) {
	if !p.active() || maximum == 0 || maximum > p.bounds.raw || !candidateCapacity(0, p.raw, maximum, p.bounds.raw) {
		return nil, errProviderIntegration
	}
	raw, err := p.group.store.read(disk, name, int64(maximum))
	if err != nil || len(raw) == 0 || !candidateCapacity(0, p.raw, uint64(len(raw)), p.bounds.raw) {
		return nil, errProviderIntegration
	}
	p.raw += uint64(len(raw))
	return raw, nil
}

// Exactly four legacy inventories close at the independently derived empty
// anchor. References are selected from these originals, never a directory or
// store.objects. Every object is re-read with its exact declared byte bound.
func (p *providerHistoryFirstPOriginals) load() error {
	if !p.active() {
		return errProviderIntegration
	}
	o, s, meter := p.group, p.group.store, p.group.meter
	cursor := o.start.objects
	for range 4 {
		raw, err := p.read(s.evidence, s.inventoryName(cursor), 64<<10)
		if err != nil {
			return errProviderIntegration
		}
		fields := contentObject(providerReplayJSON(raw, meter), "format", "previous", "entries")
		previous := contentString(fields["previous"])
		inventory, err := parseProviderFrontierInventoryObserved(raw, previous, s.config.bounds.maximumEntries, s.config.bounds.maximumObjectBytes, min(s.config.bounds.maximumStoredBytes, 64<<20), meter)
		if err != nil || !providerReplayTextEqual(inventory.digest, cursor, meter) || len(inventory.entries) == 0 ||
			!candidateCapacity(0, uint64(len(p.objects)), uint64(len(inventory.entries)), p.bounds.objects) {
			return errProviderIntegration
		}
		for _, ref := range inventory.entries {
			for _, old := range p.objects {
				if !meter.event(providerCompareEvent, uint64(len(old.kind)+len(ref.kind)+len(old.digest)+len(ref.digest))) || old.kind == ref.kind && old.digest == ref.digest {
					return errProviderIntegration
				}
			}
			original, err := p.read(s.evidence, s.objectName(ref.kind, ref.digest), ref.bytes)
			if err != nil || uint64(len(original)) != ref.bytes {
				return errProviderIntegration
			}
			object, err := newProviderIntegrationObjectObserved(ref.kind, original, s.config.bounds.maximumObjectBytes, meter)
			if err != nil || !providerReplayTextEqual(object.digest, ref.digest, meter) || !meter.event(providerCopyEvent, 1) {
				return errProviderIntegration
			}
			p.objects = append(p.objects, object)
		}
		cursor = previous
	}
	empty, err := newProviderFrontierInventoryObserved("", nil, s.config.bounds.maximumEntries, min(s.config.bounds.maximumStoredBytes, 64<<20), meter)
	if err != nil || !providerReplayTextEqual(cursor, empty.digest, meter) {
		return errProviderIntegration
	}
	return nil
}

func (p *providerHistoryFirstPOriginals) object(kind, digest string) []byte {
	if !p.active() || !lowerHash(digest) {
		return nil
	}
	meter := p.group.meter
	for _, original := range p.objects {
		if !meter.event(providerCompareEvent, uint64(len(original.kind)+len(kind)+len(original.digest)+len(digest))) {
			return nil
		}
		if original.kind == kind && original.digest == digest {
			raw := providerReplayPhaseBytes(original.raw, meter)
			actual, ok := providerIntegrationObjectDigestObserved(kind, raw, meter)
			if !ok || !providerReplayTextEqual(actual, digest, meter) {
				return nil
			}
			return raw
		}
	}
	return nil
}

func (p *providerHistoryFirstPOriginals) unique(kind string) string {
	if !p.active() {
		return ""
	}
	var selected string
	for _, original := range p.objects {
		if !p.group.meter.event(providerCompareEvent, uint64(len(original.kind)+len(kind))) {
			return ""
		}
		if original.kind == kind {
			if selected != "" {
				return ""
			}
			selected = original.digest
		}
	}
	return selected
}

func (p *providerHistoryFirstPOriginals) selected(current providerFrontierRecord) (FinalizedHeader, error) {
	o, meter := p.group, p.group.meter
	policy, err := parseProviderPolicyCandidateObserved(providerReplayPhaseBytes(o.store.config.policy.raw, meter), 64<<10, meter)
	if err != nil || *policy != *o.store.config.policy {
		return FinalizedHeader{}, errProviderIntegration
	}
	context := &ProviderCandidateContext{identity: o.store.config.identity, bound: true}
	admissionRaw := p.object("admission", o.start.admission)
	fields, ok := providerReplayJSON(admissionRaw, meter).(map[string]any)
	if !ok {
		return FinalizedHeader{}, errProviderIntegration
	}
	anchor, err := parseCandidateEnrollmentRootObserved(context, p.object("enrollment", contentString(fields["enrollment_anchor_root"])), 64<<10, meter)
	if err != nil {
		return FinalizedHeader{}, errProviderIntegration
	}
	intent, err := parseCandidateAdmissionIntentObserved(policy, context, anchor, admissionRaw, 64<<10, meter)
	if err != nil || !providerReplayTextEqual(intent.digest, o.start.admission, meter) {
		return FinalizedHeader{}, errProviderIntegration
	}
	ticket, err := parseCandidateAdmissionTicketObserved(policy, intent, 0, p.object("ticket", o.start.ticket), 64<<10, meter)
	if err != nil || ticket.sequence != 0 || ticket.kind != "rpc-capture" || !providerReplayTextEqual(ticket.digest, o.start.ticket, meter) {
		return FinalizedHeader{}, errProviderIntegration
	}
	expected, err := newProviderIntegrationRequestObserved("eth_getBlockByNumber", []any{"finalized", false}, meter)
	if err != nil || ticket.method != expected.method || ticket.params != expected.paramsDigest || ticket.input != expected.inputDigest ||
		!providerReplayPhaseRawTextEqual(p.object("request", ticket.input), expected.wire, meter) {
		return FinalizedHeader{}, errProviderIntegration
	}
	paramsRaw := providerReplayPhaseBytes(expected.params, meter)
	params, valid := providerIntegrationObjectDigestObserved("params", paramsRaw, meter)
	if !valid || !providerReplayEqual(p.object("params", params), paramsRaw, meter) {
		return FinalizedHeader{}, errProviderIntegration
	}
	consumptionDigest, settlementDigest := p.unique("consumption"), p.unique("settlement")
	consumed, err := parseCandidateConsumptionObserved(ticket, p.object("consumption", consumptionDigest), 64<<10, meter)
	if err != nil || consumed.sequence != 0 || consumed.previous != "" {
		return FinalizedHeader{}, errProviderIntegration
	}
	settled, err := parseCandidateAdmissionSettlementObserved(ticket, p.object("settlement", settlementDigest), 64<<10, meter)
	if err != nil || settled.outcome != "complete" || !providerReplayTextEqual(settled.retained, o.start.exchange, meter) ||
		!p.control(current, consumptionDigest, settlementDigest) {
		return FinalizedHeader{}, errProviderIntegration
	}
	return p.exchange(policy, ticket, consumed, settled)
}

// Bind the unique original C/S to the actual selected commits, not merely to
// membership in an inventory. The third frontier must be the fourth's parent.
func (p *providerHistoryFirstPOriginals) control(last providerFrontierRecord, consumption, settlement string) bool {
	o, s, meter := p.group, p.group.store, p.group.meter
	previousRaw, err := p.read(s.custody, s.name("frontier", 3), s.config.bounds.maximumMetadataPerCommit)
	previous, parseErr := parseProviderFrontierRecordObserved(previousRaw, s.config.identity, meter)
	if err != nil || parseErr != nil || previous.sequence != 3 || last.sequence != 4 ||
		!providerReplayTextEqual(providerReplayDigest("sage-provider-custody-frontier-v1", previousRaw, meter), last.previous, meter) ||
		last.objects != o.start.objects || last.commit != o.start.main.commit {
		return false
	}
	for _, row := range []struct {
		frontier        providerFrontierRecord
		kind, operation string
	}{{previous, "consume", consumption}, {last, "settle", settlement}} {
		intent, a := p.read(s.custody, s.name("intent", row.frontier.sequence), s.config.bounds.maximumMetadataPerCommit)
		commit, b := p.read(s.custody, s.name("commit", row.frontier.sequence), s.config.bounds.maximumMetadataPerCommit)
		value := providerFrontierIntentObjectObserved(intent, meter)
		if a != nil || b != nil || value == nil || value["kind"] != row.kind || value["operation"] != row.operation ||
			value["sequence"] != strconv.FormatUint(row.frontier.sequence, 10) || value["objects_root"] != row.frontier.objects ||
			value["previous_frontier"] != row.frontier.previous || value["new_enrollment_root"] != row.frontier.enrollment ||
			!providerReplayTextEqual(providerReplayDigest("sage-provider-custody-commit-v2", commit, meter), row.frontier.commit, meter) ||
			!providerReplayEqual(providerFrontierCommitObserved(intent, meter), commit, meter) {
			return false
		}
	}
	return true
}

func (p *providerHistoryFirstPOriginals) exchange(policy *ProviderPolicyCandidate, ticket *CandidateAdmissionTicket, consumption *CandidateConsumption, settlement *CandidateAdmissionSettlement) (FinalizedHeader, error) {
	o, meter := p.group, p.group.meter
	id, capture := o.store.config.identity, o.store.config.capture
	x := contentObject(providerReplayJSON(p.object("exchange", o.start.exchange), meter), "format", "policy", "namespace", "instance_key", "source_id", "generation", "ticket_digest", "provider_digest", "transport", "request", "response", "clock_start", "clock_end")
	if x == nil || x["format"] != "sage-provider-exchange-v2" || x["policy"] != id.Policy || x["namespace"] != id.Namespace || x["instance_key"] != id.Instance ||
		x["source_id"] != id.SourceID || x["generation"] != strconv.FormatUint(id.Generation, 10) || x["ticket_digest"] != ticket.digest || x["provider_digest"] != id.Provider {
		return FinalizedHeader{}, errProviderIntegration
	}
	configuration, ok := providerReplayJSON(providerReplayPhaseBytes(policy.raw, meter), meter).(map[string]any)
	clock, clockOK := configuration["clock"].(map[string]any)
	provider, providerOK := configuration["provider"].(map[string]any)
	start, a := candidateAdmissionClock(x["clock_start"], clock)
	end, b := candidateAdmissionClock(x["clock_end"], clock)
	if !ok || !clockOK || !providerOK || !a || !b || !candidateLedgerClockAdvance(consumption.clock, start) ||
		!candidateLedgerClockAdvance(start, end) || !candidateLedgerClockAdvance(end, settlement.clock) || candidateLedgerExpired(end, ticket.deadline) {
		return FinalizedHeader{}, errProviderIntegration
	}
	request := p.object("request", ticket.input)
	parts := bytes.SplitN(request, []byte("\r\n\r\n"), 2)
	requestBody, requestOK := providerCanonicalBase64Observed(x["request"], 16<<10, meter)
	response, responseOK := providerCanonicalBase64Observed(x["response"], capture.BodyBytes, meter)
	if len(parts) != 2 || !requestOK || !responseOK || !providerReplayEqual(parts[1], requestBody, meter) {
		return FinalizedHeader{}, errProviderIntegration
	}
	transport := contentObject(x["transport"], "endpoint", "method", "http_version", "status", "response_headers", "tls_version", "cipher_suite", "server_name", "verified_chain", "roots_artifact", "redirects", "proxy")
	if transport == nil || transport["endpoint"] != SepoliaRPCURL || transport["method"] != "POST" || transport["http_version"] != "HTTP/1.1" || transport["status"] != "200" ||
		transport["server_name"] != "rpc.sepolia.ethpandaops.io" || transport["roots_artifact"] != provider["tls_roots_artifact"] || transport["redirects"] != "0" || transport["proxy"] != "none" || !providerIntegrationCipher(transport) {
		return FinalizedHeader{}, errProviderIntegration
	}
	chain, chainOK := transport["verified_chain"].([]any)
	if !chainOK || len(chain) == 0 || len(chain) > 16 {
		return FinalizedHeader{}, errProviderIntegration
	}
	for _, certificate := range chain {
		if !providerCandidateDigest(certificate) {
			return FinalizedHeader{}, errProviderIntegration
		}
	}
	metadata := providerReplayCanonical(transport, 1<<20, meter)
	tlsDigest, valid := providerIntegrationObjectDigestObserved("tls", metadata, meter)
	binding := contentObject(providerReplayJSON(p.object("wire-binding", p.unique("wire-binding")), meter), "format", "exchange_digest", "request_wire_digest", "response_wire_digest", "tls_digest")
	if len(metadata) == 0 || !valid || !providerReplayEqual(metadata, p.object("tls", tlsDigest), meter) || binding == nil || binding["format"] != "sage-provider-wire-binding-v1" ||
		binding["exchange_digest"] != o.start.exchange || binding["request_wire_digest"] != ticket.input || binding["tls_digest"] != tlsDigest {
		return FinalizedHeader{}, errProviderIntegration
	}
	responseWire := p.object("response", contentString(binding["response_wire_digest"]))
	if len(responseWire) == 0 || len(responseWire)+len(request) > capture.WireBytes {
		return FinalizedHeader{}, errProviderIntegration
	}
	_, headers, parsed, _, err := readProviderHTTPObserved(bufio.NewReader(bytes.NewReader(responseWire)), capture, meter, uint64(len(responseWire)))
	if err != nil || !providerReplayEqual(parsed, response, meter) || !meter.event(providerCopyEvent, uint64(len(headers))) {
		return FinalizedHeader{}, errProviderIntegration
	}
	actualHeaders := make([]map[string]string, 0, len(headers))
	for _, header := range headers {
		actualHeaders = append(actualHeaders, map[string]string{"name": header.Name, "value": header.Value})
	}
	if !providerReplayEqual(providerReplayCanonical(actualHeaders, 1<<20, meter), providerReplayCanonical(transport["response_headers"], 1<<20, meter), meter) {
		return FinalizedHeader{}, errProviderIntegration
	}
	result, err := decodeRPCResultObserved(response, 1, RPCResponseLimits{capture.BodyBytes, capture.JSONDepth}, meter)
	if err != nil {
		return FinalizedHeader{}, errProviderIntegration
	}
	return providerHistoryFirstFinalizedHeaderObserved(result, RPCResponseLimits{capture.BodyBytes, capture.JSONDepth}, meter)
}

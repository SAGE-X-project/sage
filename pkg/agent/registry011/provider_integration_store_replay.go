package registry011

import (
	"bytes"
	"encoding/json"
	"io"
	"unicode/utf8"
)

// This derives the complete typed budget/ledger closure before an ACK. It is
// independent of the live coordinator's provisional model and never trusts
// supplied totals, root presence alone or a prior diagnostic snapshot.
type providerCustodyReplay struct {
	context       *ProviderCandidateContext
	policy        *ProviderPolicyCandidate
	anchor        *CandidateEnrollmentRoot
	intent        *CandidateAdmissionIntent
	ledger        *CandidateAdmissionLedger
	start         string
	firstExchange string
	meter         *providerWorkMeter
}

func (s *providerCustodyStore) object(kind, digest string) ([]byte, error) {
	ref, exists := s.objects[kind+":"+digest]
	if !exists || ref.kind != kind || ref.digest != digest || ref.bytes > s.config.bounds.maximumObjectBytes {
		return nil, errProviderIntegration
	}
	if kind == "request" && ref.bytes > 16<<10 || kind == "params" && ref.bytes > 8<<10 {
		return nil, errProviderIntegration
	}
	raw, err := s.read(s.evidence, providerObjectName(kind, digest), int64(ref.bytes))
	domain, _ := providerIntegrationDomain(kind)
	hashBytes := uint64(len(raw) + len(domain))
	var scanMeter *providerWorkMeter
	if m := s.activeMeter.Load(); m != nil && m.scan != nil && domain != "" {
		// The new scan path observes the actual domain separator as well.
		// Existing non-scan measurement subjects remain unchanged.
		hashBytes++
		scanMeter = m
	}
	if !s.event(providerHashEvent, hashBytes) || scanMeter == nil && domain != "" && !s.event(providerCodecEvent, uint64(len(raw))) {
		return nil, errProviderIntegration
	}
	actual, valid := providerIntegrationObjectDigest(kind, raw)
	if err != nil || !valid || actual != digest || uint64(len(raw)) != ref.bytes {
		return nil, errProviderIntegration
	}
	if domain != "" {
		m, ok := providerReplayJSON(raw, scanMeter).(map[string]any)
		if !ok || m["format"] != domain {
			return nil, errProviderIntegration
		}
	}
	if s.requiredReferences != nil {
		s.requiredReferences[kind+":"+digest] = true
	}
	return raw, nil
}

func (r *providerCustodyReplay) apply(s *providerCustodyStore, intent map[string]any) error {
	if r == nil || s == nil || r.meter != nil && s.activeMeter.Load() != r.meter {
		return errProviderIntegration
	}
	raw, err := s.object("enrollment", contentString(intent["new_enrollment_root"]))
	root, parseErr := parseCandidateEnrollmentRootObserved(r.context, raw, 64<<10, r.meter)
	if err != nil || parseErr != nil {
		return errProviderIntegration
	}
	kind, operation := contentString(intent["kind"]), contentString(intent["operation"])
	if kind == "enroll" {
		if r.anchor != nil || root.sequence != 0 || root.budget != "" || operation != providerCandidateHash("sage-provider-enroll-operation-v1", []byte(s.config.anchor+root.identity.Instance+root.identity.Policy+root.identity.SourceID)) {
			return errProviderIntegration
		}
		selected, exists := s.setupForBoundary(s.tip)
		if !exists {
			return errProviderIntegration
		}
		commit, err := s.object("setup-commit", selected.commit)
		claim, claimErr := s.object("setup", selected.claim)
		if err != nil || claimErr != nil || !bytes.Equal(commit, providerSetupCommit(claim)) {
			return errProviderIntegration
		}
		r.anchor = root
		return nil
	}
	if r.anchor == nil || root.budget == "" {
		return errProviderIntegration
	}
	budget, err := s.object("budget", root.budget)
	if err != nil {
		return errProviderIntegration
	}
	var change *CandidateAdmissionLedgerChange
	switch kind {
	case "admit":
		if r.intent == nil {
			b, ok := providerReplayJSON(budget, r.meter).(map[string]any)
			if !ok || b["format"] != "sage-provider-budget-root-v2" {
				return errProviderIntegration
			}
			i, err := s.object("admission", contentString(b["admission_digest"]))
			if err != nil {
				return errProviderIntegration
			}
			r.intent, err = parseCandidateAdmissionIntentObserved(r.policy, r.context, r.anchor, i, 64<<10, r.meter)
			originalIntent, intentOK := providerReplayJSON(i, r.meter).(map[string]any)
			if err != nil || !intentOK || originalIntent["authority_metadata_root"] != s.tip || r.intent.clock != providerClockDiagnostic(s.setupOrigin) {
				return errProviderIntegration
			}
		}
		payload, err := s.object("ticket", operation)
		if err != nil {
			return errProviderIntegration
		}
		// Segment ordinal is recovered from its exact admission-derived digest,
		// never selected from an untrusted supplied count or a fresh ledger.
		segment := uint64(0)
		if r.ledger != nil {
			segment = r.ledger.lastTicket.segment
			m, ok := providerReplayJSON(payload, r.meter).(map[string]any)
			if !ok {
				return errProviderIntegration
			}
			if m["segment"] != candidateAdmissionSegment(r.intent.digest, segment) {
				segment++
			}
		}
		ticket, err := parseCandidateAdmissionTicketObserved(r.policy, r.intent, segment, payload, 64<<10, r.meter)
		if err != nil || r.checkTicketBytes(s, ticket) != nil {
			return errProviderIntegration
		}
		if r.ledger == nil {
			if s.config.bounds.continuation != nil {
				r.ledger, err = newProviderContinuingLedgerObserved(r.policy, r.intent, ticket, *s.config.bounds.continuation, r.meter)
			} else {
				r.ledger, err = NewCandidateAdmissionLedger(r.policy, r.intent, ticket, s.config.bounds.ledger)
			}
			if err == nil {
				change, err = r.ledger.Current()
			}
		} else {
			change, err = r.ledger.AppendCandidateTicket(ticket)
		}
		if err != nil {
			return errProviderIntegration
		}
	case "consume":
		payload, err := s.object("consumption", operation)
		if err != nil || r.ledger == nil {
			return errProviderIntegration
		}
		m, ok := providerReplayJSON(payload, r.meter).(map[string]any)
		if !ok {
			return errProviderIntegration
		}
		entry, exists := r.ledger.tickets[contentString(m["ticket_digest"])]
		if !exists || entry.ticket == nil {
			return errProviderIntegration
		}
		ticket := entry.ticket
		consumption, err := parseCandidateConsumptionObserved(ticket, payload, 64<<10, r.meter)
		if err != nil {
			return errProviderIntegration
		}
		change, err = r.ledger.AppendCandidateConsumption(consumption)
		if err != nil {
			return errProviderIntegration
		}
	case "settle":
		payload, err := s.object("settlement", operation)
		if err != nil || r.ledger == nil {
			return errProviderIntegration
		}
		m, ok := providerReplayJSON(payload, r.meter).(map[string]any)
		if !ok {
			return errProviderIntegration
		}
		entry, exists := r.ledger.tickets[contentString(m["ticket_digest"])]
		if !exists || entry.ticket == nil {
			return errProviderIntegration
		}
		ticket := entry.ticket
		settlement, err := parseCandidateAdmissionSettlementObserved(ticket, payload, 64<<10, r.meter)
		if err != nil || settlement.outcome == "complete" && r.checkResult(s, settlement) != nil {
			return errProviderIntegration
		}
		change, err = r.ledger.AppendCandidateSettlement(settlement)
		if err != nil {
			return errProviderIntegration
		}
	default:
		// Later typed segment/checkpoint/activation kinds must implement their
		// exact replay before they can receive this ACK. No generic acceptance.
		return errProviderIntegration
	}
	if change == nil || change.digest != root.budget || !bytes.Equal(budget, []byte(change.raw)) {
		return errProviderIntegration
	}
	for _, node := range change.LedgerNodeBytes() {
		digest := providerCandidateHash("sage-provider-ledger-node-v1", node)
		raw, err := s.object("ledger-node", digest)
		if err != nil || !bytes.Equal(raw, node) {
			return errProviderIntegration
		}
	}
	return nil
}

func (r *providerCustodyReplay) checkTicketBytes(s *providerCustodyStore, ticket *CandidateAdmissionTicket) error {
	if ticket.kind == "local-work" {
		if _, exists := s.objects["pin-input:"+ticket.input]; exists {
			_, _, err := r.pinInput(s, ticket.input)
			return err
		}
		// This bundle's local input must be an existing typed immutable object.
		for _, kind := range []string{"exchange", "bootstrap", "pin", "checkpoint", "batch"} {
			if _, ok := s.objects[kind+":"+ticket.input]; ok {
				return nil
			}
		}
		return errProviderIntegration
	}
	wire, err := s.object("request", ticket.input)
	if err != nil {
		return errProviderIntegration
	}
	parts := bytes.SplitN(wire, []byte("\r\n\r\n"), 2)
	if len(parts) != 2 {
		return errProviderIntegration
	}
	if !utf8.Valid(parts[1]) || !validRPCUnicode(parts[1]) {
		return errProviderIntegration
	}
	decoder := json.NewDecoder(bytes.NewReader(parts[1]))
	decoder.UseNumber()
	if !readRPCValue(decoder, 0, 10) {
		return errProviderIntegration
	}
	if _, err := decoder.Token(); err != io.EOF {
		return errProviderIntegration
	}
	var body struct {
		Version string `json:"jsonrpc"`
		ID      uint64 `json:"id"`
		Method  string `json:"method"`
		Params  any    `json:"params"`
	}
	if json.Unmarshal(parts[1], &body) != nil || body.Version != "2.0" || body.ID != 1 || body.Method != ticket.method {
		return errProviderIntegration
	}
	body.Params = providerRequestNumbers(body.Params)
	request, err := newProviderIntegrationRequest(body.Method, body.Params)
	if err != nil || !bytes.Equal(wire, []byte(request.wire)) || request.paramsDigest != ticket.params || ticket.sequence == 0 && !request.firstFinalized() {
		return errProviderIntegration
	}
	paramsDigest, _ := providerIntegrationObjectDigest("params", []byte(request.params))
	params, err := s.object("params", paramsDigest)
	if err != nil || !bytes.Equal(params, []byte(request.params)) {
		return errProviderIntegration
	}
	return nil
}

// Only the exact fixed tracer option numeric0 is supported. Every other JSON
// number remains an unsupported type and the closed request builder refuses.
func providerRequestNumbers(value any) any {
	switch v := value.(type) {
	case float64:
		if v == 0 {
			return int(0)
		}
	case []any:
		for i := range v {
			v[i] = providerRequestNumbers(v[i])
		}
	case map[string]any:
		for key := range v {
			v[key] = providerRequestNumbers(v[key])
		}
	}
	return value
}

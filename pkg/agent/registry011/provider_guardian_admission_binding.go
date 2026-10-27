package registry011

import "strconv"

const providerGuardianAdmissionBindingDomain = "sage-provider-guardian-admission-binding-v1"

// This is retained DATA, not permission. A concrete continuing owner must
// retain this record and its exact T in the same main custody transition,
// check its private composite ACK again before C, and burn before execution.
type providerGuardianAdmissionBinding struct {
	ordinal                                   uint64
	claim, head, previous                     string
	main, frontier, setup, admission          string
	operation, purpose, target, ticket, input string
	kind, method, params                      string
	clock                                     CandidateClockDiagnostic
}

// The target is available before T, so there is no claim -> T -> claim cycle.
// RPC input is the ordinary exact request-wire digest; TicketV2 is unchanged.
func (x *providerScanCodec) guardianTicketTarget(purpose, admission, kind, input, method, params string) (string, error) {
	if x == nil || x.config.bounds.guardian == nil || !lowerHash(admission) || !lowerHash(input) {
		return "", errProviderIntegration
	}
	var methodValue, paramsValue any
	switch kind {
	case "rpc-capture":
		if purpose != "rpc-admission" || !candidateAdmissionReadMethod(method) || !lowerHash(params) {
			return "", errProviderIntegration
		}
		methodValue, paramsValue = method, params
	case "local-work":
		if purpose != "bootstrap-admission" && purpose != "local-admission" && purpose != "publication-inspection" || method != "" || params != "" {
			return "", errProviderIntegration
		}
	default:
		return "", errProviderIntegration
	}
	m := x.guardianIdentity("sage-provider-guardian-ticket-target-v1")
	m["purpose"], m["admission_digest"], m["kind"], m["input_digest"] = purpose, admission, kind, input
	m["method"], m["params_digest"] = methodValue, paramsValue
	raw, err := x.encode(m)
	digest := x.hash("sage-provider-guardian-ticket-target-v1", raw)
	if err != nil || !lowerHash(digest) {
		return "", errProviderIntegration
	}
	return digest, nil
}

func (x *providerScanCodec) guardianAdmissionBinding(v providerGuardianAdmissionBinding) ([]byte, error) {
	if x == nil || x.config.bounds.guardian == nil || v.ordinal == 0 || v.ordinal > x.config.bounds.guardian.attempts || v.clock.ClockID == "" || v.clock.Epoch == "" || !candidateLedgerClockAdvance(v.clock, v.clock) {
		return nil, errProviderIntegration
	}
	for _, digest := range []string{v.claim, v.head, v.previous, v.main, v.frontier, v.setup, v.admission, v.operation, v.target, v.ticket, v.input} {
		if !lowerHash(digest) {
			return nil, errProviderIntegration
		}
	}
	target, err := x.guardianTicketTarget(v.purpose, v.admission, v.kind, v.input, v.method, v.params)
	if err != nil || target != v.target {
		return nil, errProviderIntegration
	}
	m := x.guardianIdentity(providerGuardianAdmissionBindingDomain)
	m["ordinal"], m["guardian_claim_digest"], m["guardian_head_digest"], m["previous_guardian_head"] = strconv.FormatUint(v.ordinal, 10), v.claim, v.head, v.previous
	m["immediate_main_commit"], m["immediate_main_frontier"], m["selected_setup_head"] = v.main, v.frontier, v.setup
	m["admission_digest"], m["original_clock"] = v.admission, providerDiagnosticClockObject(v.clock)
	m["guardian_operation"], m["purpose"], m["target_digest"], m["ticket_digest"] = v.operation, v.purpose, v.target, v.ticket
	m["kind"], m["input_digest"] = v.kind, v.input
	var method, params any
	if v.kind == "rpc-capture" {
		method, params = v.method, v.params
	}
	m["method"], m["params_digest"] = method, params
	return x.encode(m)
}

func (x *providerScanCodec) parseGuardianAdmissionBinding(raw []byte) (providerGuardianAdmissionBinding, error) {
	if x == nil || x.config.bounds.guardian == nil {
		return providerGuardianAdmissionBinding{}, errProviderIntegration
	}
	m := contentObject(x.decode(raw), "format", "namespace", "instance_key", "generation", "fence", "profile", "policy", "source_id", "ordinal", "guardian_claim_digest", "guardian_head_digest", "previous_guardian_head", "immediate_main_commit", "immediate_main_frontier", "selected_setup_head", "admission_digest", "original_clock", "guardian_operation", "purpose", "target_digest", "ticket_digest", "kind", "input_digest", "method", "params_digest")
	n, a := providerCandidateUint(m["ordinal"], true)
	clock, b := providerStoredClock(m["original_clock"])
	if m == nil || !a || !b {
		return providerGuardianAdmissionBinding{}, errProviderIntegration
	}
	v := providerGuardianAdmissionBinding{ordinal: n, claim: contentString(m["guardian_claim_digest"]), head: contentString(m["guardian_head_digest"]), previous: contentString(m["previous_guardian_head"]), main: contentString(m["immediate_main_commit"]), frontier: contentString(m["immediate_main_frontier"]), setup: contentString(m["selected_setup_head"]), admission: contentString(m["admission_digest"]), operation: contentString(m["guardian_operation"]), purpose: contentString(m["purpose"]), target: contentString(m["target_digest"]), ticket: contentString(m["ticket_digest"]), input: contentString(m["input_digest"]), kind: contentString(m["kind"]), method: contentString(m["method"]), params: contentString(m["params_digest"]), clock: clock}
	expected, err := x.guardianAdmissionBinding(v)
	if err != nil || !x.equal(raw, expected) {
		return providerGuardianAdmissionBinding{}, errProviderIntegration
	}
	return v, nil
}

// Canonical binding bytes are built after the G claim and the new T both
// exist as proposed DATA. Their common custody transition is still owed.
func providerGuardianBindingForTicket(x *providerScanCodec, data *providerGuardianClaimData, ticket *CandidateAdmissionTicket) (providerIntegrationObject, error) {
	if x == nil || data == nil || ticket == nil || ticket.intent == nil || ticket.sequence == 0 || ticket.intent.clock != data.attempt.clock || ticket.intent.digest != data.attempt.admission || data.head.ordinal != data.attempt.ordinal {
		return providerIntegrationObject{}, errProviderIntegration
	}
	a := data.attempt
	claimHead, err := x.guardianHead(data.head)
	if err != nil {
		return providerIntegrationObject{}, errProviderIntegration
	}
	v := providerGuardianAdmissionBinding{ordinal: a.ordinal, claim: data.head.claim, head: x.hash(providerGuardianHeadDomain, claimHead), previous: a.previous, main: a.main, frontier: a.frontier, setup: a.setup, admission: a.admission, operation: a.operation, purpose: a.purpose, target: a.target, ticket: ticket.digest, input: ticket.input, kind: ticket.kind, method: ticket.method, params: ticket.params, clock: a.clock}
	raw, err := x.guardianAdmissionBinding(v)
	if err != nil {
		return providerIntegrationObject{}, errProviderIntegration
	}
	return newProviderIntegrationObjectObserved("guardian-admission-binding", raw, candidateLedgerMetadataLimit, x.meter)
}

package registry011

import "strconv"

const providerGuardianSealBindingDomain = "sage-provider-guardian-seal-binding-v1"

// Candidate DATA only. A live metadata-only owner must consume and retain
// its own G claim before building this binding. Neither parsing nor equality
// acknowledges a main write, seals a page, or restores execution authority.
type providerGuardianSealBinding struct {
	ordinal, pageIndex, records      uint64
	claim, head, previous            string
	main, frontier, setup, admission string
	operation, target, budget        string
	page, seal                       string
	indexes                          [3]string
	clock                            CandidateClockDiagnostic
}

func (*providerGuardianSealBinding) UnmarshalJSON([]byte) error { return errProviderIntegration }

// Available before G and page construction: this commits the complete old
// page boundary, rather than introducing a claim -> seal -> claim cycle.
func (x *providerScanCodec) guardianSealTarget(v providerGuardianSealBinding) (string, error) {
	if x == nil || x.config.bounds.guardian == nil || x.config.bounds.continuation == nil ||
		v.clock.ClockID == "" || v.clock.Epoch == "" || !candidateLedgerClockAdvance(v.clock, v.clock) {
		return "", errProviderIntegration
	}
	bounds := x.config.bounds.continuation
	if bounds.maximumPages < 2 || v.pageIndex >= bounds.maximumPages-1 || v.records != bounds.pageRecords || v.records == 0 {
		return "", errProviderIntegration
	}
	for _, digest := range [8]string{v.main, v.frontier, v.setup, v.admission, v.budget, v.indexes[0], v.indexes[1], v.indexes[2]} {
		if !lowerHash(digest) {
			return "", errProviderIntegration
		}
	}
	m := x.guardianIdentity("sage-provider-guardian-seal-target-v1")
	m["purpose"], m["admission_digest"], m["original_clock"] = "ledger-page-seal", v.admission, providerDiagnosticClockObject(v.clock)
	m["immediate_main_commit"], m["immediate_main_frontier"], m["selected_setup_head"] = v.main, v.frontier, v.setup
	m["budget_root"], m["page_index"], m["record_count"] = v.budget, strconv.FormatUint(v.pageIndex, 10), strconv.FormatUint(v.records, 10)
	m["ticket_index_root"], m["operation_index_root"], m["segment_totals_root"] = v.indexes[0], v.indexes[1], v.indexes[2]
	raw, err := x.encode(m)
	digest := x.hash("sage-provider-guardian-seal-target-v1", raw)
	if err != nil || !lowerHash(digest) {
		return "", errProviderIntegration
	}
	return digest, nil
}

func (x *providerScanCodec) guardianSealBinding(v providerGuardianSealBinding) ([]byte, error) {
	if x == nil || x.config.bounds.guardian == nil || v.ordinal == 0 || v.ordinal > x.config.bounds.guardian.attempts {
		return nil, errProviderIntegration
	}
	for _, digest := range [7]string{v.claim, v.head, v.previous, v.operation, v.target, v.page, v.seal} {
		if !lowerHash(digest) {
			return nil, errProviderIntegration
		}
	}
	target, err := x.guardianSealTarget(v)
	if err != nil || target != v.target {
		return nil, errProviderIntegration
	}
	m := x.guardianIdentity(providerGuardianSealBindingDomain)
	m["ordinal"], m["guardian_claim_digest"], m["guardian_head_digest"], m["previous_guardian_head"] = strconv.FormatUint(v.ordinal, 10), v.claim, v.head, v.previous
	m["immediate_main_commit"], m["immediate_main_frontier"], m["selected_setup_head"] = v.main, v.frontier, v.setup
	m["admission_digest"], m["original_clock"] = v.admission, providerDiagnosticClockObject(v.clock)
	m["guardian_operation"], m["purpose"], m["target_digest"] = v.operation, "ledger-page-seal", v.target
	m["budget_root"], m["page_index"], m["record_count"] = v.budget, strconv.FormatUint(v.pageIndex, 10), strconv.FormatUint(v.records, 10)
	m["page_root"], m["seal_digest"] = v.page, v.seal
	m["ticket_index_root"], m["operation_index_root"], m["segment_totals_root"] = v.indexes[0], v.indexes[1], v.indexes[2]
	return x.encode(m)
}

func (x *providerScanCodec) parseGuardianSealBinding(raw []byte) (providerGuardianSealBinding, error) {
	var empty providerGuardianSealBinding
	if x == nil || x.config.bounds.guardian == nil {
		return empty, errProviderIntegration
	}
	m := contentObject(x.decode(raw), "format", "namespace", "instance_key", "generation", "fence", "profile", "policy", "source_id", "ordinal", "guardian_claim_digest", "guardian_head_digest", "previous_guardian_head", "immediate_main_commit", "immediate_main_frontier", "selected_setup_head", "admission_digest", "original_clock", "guardian_operation", "purpose", "target_digest", "budget_root", "page_index", "record_count", "page_root", "seal_digest", "ticket_index_root", "operation_index_root", "segment_totals_root")
	n, a := providerCandidateUint(m["ordinal"], true)
	page, b := providerCandidateUint(m["page_index"], false)
	records, c := providerCandidateUint(m["record_count"], true)
	clock, d := providerStoredClock(m["original_clock"])
	if m == nil || !a || !b || !c || !d || m["purpose"] != "ledger-page-seal" {
		return empty, errProviderIntegration
	}
	v := providerGuardianSealBinding{ordinal: n, pageIndex: page, records: records, claim: contentString(m["guardian_claim_digest"]), head: contentString(m["guardian_head_digest"]), previous: contentString(m["previous_guardian_head"]), main: contentString(m["immediate_main_commit"]), frontier: contentString(m["immediate_main_frontier"]), setup: contentString(m["selected_setup_head"]), admission: contentString(m["admission_digest"]), operation: contentString(m["guardian_operation"]), target: contentString(m["target_digest"]), budget: contentString(m["budget_root"]), page: contentString(m["page_root"]), seal: contentString(m["seal_digest"]), indexes: [3]string{contentString(m["ticket_index_root"]), contentString(m["operation_index_root"]), contentString(m["segment_totals_root"])}, clock: clock}
	expected, err := x.guardianSealBinding(v)
	if err != nil || !x.equal(raw, expected) {
		return empty, errProviderIntegration
	}
	return v, nil
}

// Page/seal are available only after both typed ledgers have independently
// received their own G-original metadata delta and prepared their boundary.
func providerGuardianSealBindingForBoundary(x *providerScanCodec, data *providerGuardianClaimData, v providerGuardianSealBinding) (providerIntegrationObject, error) {
	if x == nil || data == nil || data.attempt.purpose != "ledger-page-seal" || data.attempt.admission != v.admission ||
		data.attempt.clock != v.clock || data.attempt.main != v.main || data.attempt.frontier != v.frontier || data.attempt.setup != v.setup {
		return providerIntegrationObject{}, errProviderIntegration
	}
	a := data.attempt
	head, err := x.guardianHead(data.head)
	if err != nil || data.head.ordinal != a.ordinal {
		return providerIntegrationObject{}, errProviderIntegration
	}
	v.ordinal, v.claim, v.head, v.previous = a.ordinal, data.head.claim, x.hash(providerGuardianHeadDomain, head), a.previous
	v.operation, v.target = a.operation, a.target
	raw, err := x.guardianSealBinding(v)
	if err != nil {
		return providerIntegrationObject{}, errProviderIntegration
	}
	return newProviderIntegrationObjectObserved("guardian-seal-binding", raw, candidateLedgerMetadataLimit, x.meter)
}

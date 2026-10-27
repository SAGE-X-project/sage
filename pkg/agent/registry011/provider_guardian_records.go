package registry011

import "strconv"

const providerGuardianOriginDomain = "sage-provider-guardian-origin-v1"
const providerGuardianHeadDomain = "sage-provider-guardian-head-v1"
const providerGuardianIntentDomain = "sage-provider-guardian-attempt-v1"
const providerGuardianClaimDomain = "sage-provider-guardian-claim-v1"
const providerGuardianCommitDomain = "sage-provider-guardian-commit-v1"

// Immutable DATA only. The private live guardian separately owns the selected
// pins and an irreversible claim; decoding these records cannot restore it.
type providerGuardianHeadData struct {
	origin, e0, frontier, h0, setup    string
	previous, claim, commit, admission string
	ordinal, deadline                  uint64
	clock, lastStart                   CandidateClockDiagnostic
}

type providerGuardianAttemptData struct {
	origin, previous, operation, purpose     string
	main, frontier, setup, target, admission string
	ordinal, deadline                        uint64
	clock, start                             CandidateClockDiagnostic
}

func (x *providerScanCodec) guardianIdentity(domain string) map[string]any {
	m := x.identity(domain)
	m["policy"], m["source_id"] = x.config.identity.Policy, x.config.identity.SourceID
	return m
}

func (x *providerScanCodec) guardianOrigin(h0, setup string, clock CandidateClockDiagnostic) ([]byte, error) {
	if x == nil || x.config.bounds.guardian == nil || !lowerHash(h0) || !lowerHash(setup) || !candidateLedgerClockAdvance(clock, clock) {
		return nil, errProviderIntegration
	}
	bounds, err := x.guardianBoundsBytes()
	digest := x.hash("sage-provider-guardian-bounds-v1", bounds)
	if err != nil || !lowerHash(digest) {
		return nil, errProviderIntegration
	}
	m := x.guardianIdentity(providerGuardianOriginDomain)
	m["selected_anchor"], m["h0_digest"], m["setup0_head"] = x.config.anchor, h0, setup
	m["original_clock"], m["guardian_bounds_digest"] = providerDiagnosticClockObject(clock), digest
	m["initial_guardian_ordinal"], m["previous_guardian_head"] = "0", nil
	m["continuation"] = "same-live-owner-incremental-v1"
	return x.encode(m)
}

func (x *providerScanCodec) guardianHead(v providerGuardianHeadData) ([]byte, error) {
	if x == nil || x.config.bounds.guardian == nil || v.ordinal > x.config.bounds.guardian.attempts || !candidateLedgerClockAdvance(v.clock, v.clock) {
		return nil, errProviderIntegration
	}
	for _, value := range []string{v.origin, v.e0, v.frontier, v.h0, v.setup} {
		if !lowerHash(value) {
			return nil, errProviderIntegration
		}
	}
	if v.ordinal == 0 {
		if v.previous != "" || v.claim != "" || v.commit != "" || v.admission != "" || v.lastStart != (CandidateClockDiagnostic{}) {
			return nil, errProviderIntegration
		}
	} else {
		if !candidateLedgerClockAdvance(v.clock, v.lastStart) || candidateLedgerExpired(v.lastStart, v.deadline) || uint64(v.lastStart.MonotonicMS-v.clock.MonotonicMS) > x.config.bounds.guardian.lifetime.duration {
			return nil, errProviderIntegration
		}
		for _, value := range []string{v.previous, v.claim, v.commit, v.admission} {
			if !lowerHash(value) {
				return nil, errProviderIntegration
			}
		}
	}
	deadline, _, valid := x.deadline(v.clock)
	used, ok := x.config.bounds.guardian.used(v.ordinal)
	if !valid || deadline != v.deadline || !ok {
		return nil, errProviderIntegration
	}
	m := x.guardianIdentity(providerGuardianHeadDomain)
	m["origin_digest"], m["e0_commit"], m["e0_frontier"], m["selected_h0"], m["setup0_head"] = v.origin, v.e0, v.frontier, v.h0, v.setup
	m["sequence"], m["previous"], m["last_claim"], m["last_commit"] = strconv.FormatUint(v.ordinal, 10), providerNullableDigest(v.previous), providerNullableDigest(v.claim), providerNullableDigest(v.commit)
	m["admission_digest"], m["original_clock"], m["deadline_wall_ms"] = providerNullableDigest(v.admission), providerDiagnosticClockObject(v.clock), strconv.FormatUint(v.deadline, 10)
	m["cumulative_claim_upper"] = used
	var lastClock any
	var elapsed, resident uint64
	if v.ordinal != 0 {
		lastClock = providerDiagnosticClockObject(v.lastStart)
		elapsed = uint64(v.lastStart.MonotonicMS - v.clock.MonotonicMS)
		resident = x.config.bounds.guardian.claim.resident
	}
	m["last_claim_clock"], m["elapsed_from_origin_ms"], m["resident_peak_upper"] = lastClock, strconv.FormatUint(elapsed, 10), strconv.FormatUint(resident, 10)
	return x.encode(m)
}

func (b providerGuardianBounds) used(ordinal uint64) (map[string]any, bool) {
	if ordinal > b.attempts {
		return nil, false
	}
	var work candidateWorkVector
	for i, n := range b.claim.work {
		var ok bool
		work[i], ok = providerUpperProduct(n, ordinal)
		if !ok || work[i] > b.lifetime.work[i] {
			return nil, false
		}
	}
	m := map[string]any{"work_11": candidateLedgerWorkObject(work)}
	for key, pair := range map[string][2]uint64{
		"retained_bytes": {b.claim.retained, b.lifetime.retained},
		"live_bytes":     {b.claim.live, b.lifetime.live},
		"clock_reads":    {b.claim.clocks, b.lifetime.clocks},
		"records":        {b.claim.records, b.lifetime.records},
		"objects":        {b.claim.objects, b.lifetime.objects},
		"raw_bytes":      {b.claim.raw, b.lifetime.raw},
		"metadata_bytes": {b.claim.metadata, b.lifetime.metadata},
	} {
		n, ok := providerUpperProduct(pair[0], ordinal)
		if !ok || n > pair[1] {
			return nil, false
		}
		m[key] = strconv.FormatUint(n, 10)
	}
	commits, commitsOK := providerUpperSum(ordinal, 1)
	files, filesOK := providerUpperProduct(ordinal, 5)
	files, filesSumOK := providerUpperSum(files, 2)
	if !commitsOK || !filesOK || !filesSumOK || commits > b.commits || files > b.catalogEntries {
		return nil, false
	}
	m["guardian_committed_records"], m["guardian_file_allocations"] = strconv.FormatUint(commits, 10), strconv.FormatUint(files, 10)
	return m, true
}

func (x *providerScanCodec) guardianAttempt(v providerGuardianAttemptData) ([]byte, error) {
	if x == nil || x.config.bounds.guardian == nil || v.ordinal == 0 || v.ordinal > x.config.bounds.guardian.attempts || !providerGuardianPurpose(v.purpose) {
		return nil, errProviderIntegration
	}
	for _, value := range []string{v.origin, v.previous, v.operation, v.main, v.frontier, v.setup, v.target, v.admission} {
		if !lowerHash(value) {
			return nil, errProviderIntegration
		}
	}
	deadline, _, ok := x.deadline(v.clock)
	if !ok || deadline != v.deadline || !candidateLedgerClockAdvance(v.clock, v.start) || candidateLedgerExpired(v.start, deadline) || uint64(v.start.MonotonicMS-v.clock.MonotonicMS) > x.config.bounds.guardian.lifetime.duration {
		return nil, errProviderIntegration
	}
	m := x.guardianIdentity(providerGuardianIntentDomain)
	m["origin_digest"], m["previous_guardian_head"], m["selected_guardian_head"] = v.origin, v.previous, v.previous
	m["ordinal"], m["operation_id"], m["purpose"] = strconv.FormatUint(v.ordinal, 10), v.operation, v.purpose
	m["selected_main_commit"], m["selected_main_frontier"], m["selected_setup_head"] = v.main, v.frontier, v.setup
	m["target_digest"], m["admission_digest"] = v.target, v.admission
	m["original_clock"], m["deadline_wall_ms"], m["claim_upper"] = providerDiagnosticClockObject(v.clock), strconv.FormatUint(v.deadline, 10), x.config.bounds.guardian.claim.object()
	m["claim_start_clock"] = providerDiagnosticClockObject(v.start)
	return x.encode(m)
}

func providerGuardianPurpose(purpose string) bool {
	return purpose == "bootstrap-admission" || purpose == "rpc-admission" || purpose == "local-admission" || purpose == "publication-inspection" || purpose == "ledger-page-seal"
}

func (x *providerScanCodec) guardianOperation(v providerGuardianAttemptData) (string, error) {
	if x == nil || x.config.bounds.guardian == nil || v.ordinal == 0 || v.ordinal > x.config.bounds.guardian.attempts || !providerGuardianPurpose(v.purpose) {
		return "", errProviderIntegration
	}
	for _, digest := range []string{v.target, v.previous, v.main, v.admission} {
		if !lowerHash(digest) {
			return "", errProviderIntegration
		}
	}
	m := x.guardianIdentity("sage-provider-guardian-operation-v1")
	m["ordinal"], m["target"], m["previous"], m["purpose"] = strconv.FormatUint(v.ordinal, 10), v.target, v.previous, v.purpose
	m["main_commit"], m["admission"] = v.main, v.admission
	raw, err := x.encode(m)
	digest := x.hash("sage-provider-guardian-operation-v1", raw)
	if err != nil || !lowerHash(digest) {
		return "", errProviderIntegration
	}
	return digest, nil
}

// Strict DATA parsing re-derives every selected upper and canonical field.
// This can verify an original record; it never recreates the live G owner.
func (x *providerScanCodec) parseGuardianAttempt(raw []byte) (providerGuardianAttemptData, error) {
	if x == nil || x.config.bounds.guardian == nil {
		return providerGuardianAttemptData{}, errProviderIntegration
	}
	m := contentObject(x.decode(raw), "format", "namespace", "instance_key", "generation", "fence", "profile", "policy", "source_id", "origin_digest", "previous_guardian_head", "selected_guardian_head", "ordinal", "operation_id", "purpose", "selected_main_commit", "selected_main_frontier", "selected_setup_head", "target_digest", "admission_digest", "original_clock", "deadline_wall_ms", "claim_upper", "claim_start_clock")
	ordinal, a := providerCandidateUint(m["ordinal"], true)
	deadline, b := providerCandidateUint(m["deadline_wall_ms"], true)
	clock, c := providerStoredClock(m["original_clock"])
	start, d := providerStoredClock(m["claim_start_clock"])
	if m == nil || !a || !b || !c || !d {
		return providerGuardianAttemptData{}, errProviderIntegration
	}
	v := providerGuardianAttemptData{origin: contentString(m["origin_digest"]), previous: contentString(m["previous_guardian_head"]), operation: contentString(m["operation_id"]), purpose: contentString(m["purpose"]), main: contentString(m["selected_main_commit"]), frontier: contentString(m["selected_main_frontier"]), setup: contentString(m["selected_setup_head"]), target: contentString(m["target_digest"]), admission: contentString(m["admission_digest"]), ordinal: ordinal, deadline: deadline, clock: clock, start: start}
	operation, opErr := x.guardianOperation(v)
	expected, err := x.guardianAttempt(v)
	if opErr != nil || operation != v.operation || err != nil || !x.equal(raw, expected) {
		return providerGuardianAttemptData{}, errProviderIntegration
	}
	return v, nil
}

func (x *providerScanCodec) guardianClaim(intent []byte, v providerGuardianAttemptData) ([]byte, error) {
	expected, err := x.guardianAttempt(v)
	if err != nil || !x.equal(intent, expected) {
		return nil, errProviderIntegration
	}
	m := x.guardianIdentity(providerGuardianClaimDomain)
	m["intent_digest"], m["operation_id"], m["ordinal"] = x.hash(providerGuardianIntentDomain, intent), v.operation, strconv.FormatUint(v.ordinal, 10)
	m["original_clock"], m["status"], m["claim_upper"] = providerDiagnosticClockObject(v.clock), "consumed", x.config.bounds.guardian.claim.object()
	return x.encode(m)
}

func (x *providerScanCodec) guardianCommit(intent, claim []byte, v providerGuardianAttemptData) ([]byte, error) {
	expected, err := x.guardianClaim(intent, v)
	if err != nil {
		return nil, errProviderIntegration
	}
	used, ok := x.config.bounds.guardian.used(v.ordinal)
	if !ok || !x.equal(claim, expected) {
		return nil, errProviderIntegration
	}
	m := x.guardianIdentity(providerGuardianCommitDomain)
	m["intent_digest"], m["claim_digest"], m["previous_guardian_head"] = x.hash(providerGuardianIntentDomain, intent), x.hash(providerGuardianClaimDomain, claim), v.previous
	m["ordinal"], m["cumulative_claim_upper"] = strconv.FormatUint(v.ordinal, 10), used
	return x.encode(m)
}

package registry011

import "strconv"

const (
	providerScanIntentDomain    = "sage-provider-scan-attempt-intent-v1"
	providerScanCommitDomain    = "sage-provider-scan-attempt-commit-v1"
	providerScanHighwaterDomain = "sage-provider-scan-attempt-highwater-v1"
	providerScanHeadDomain      = "sage-provider-scan-head-v1"
)

// These closed records describe candidate retained data. A parsed chain cannot
// create the selected root-operation principal, an execution ACK or a Source.
type providerScanHighwater struct {
	ordinal                          uint64
	previous, target, intent, commit string
	held                             providerScanUpper
}

type providerScanAttempt struct {
	ordinal                                                            uint64
	previousCommit, previousHighwater, setup, custody, target, purpose string
	clock                                                              CandidateClockDiagnostic
	deadline                                                           uint64
	preparation, body                                                  providerScanUpper
}

func (x *providerScanCodec) highwater(h providerScanHighwater) ([]byte, error) {
	config := x.config
	m := x.identity(providerScanHighwaterDomain)
	if m == nil || h.ordinal > config.bounds.scan.maximumAttempts {
		return nil, errProviderIntegration
	}
	if h.ordinal == 0 {
		if h.previous != "" || h.target != "" || h.intent != "" || h.commit != "" || h.held != (providerScanUpper{}) {
			return nil, errProviderIntegration
		}
	} else if !lowerHash(h.previous) || !lowerHash(h.target) || !lowerHash(h.intent) || !lowerHash(h.commit) || h.held.retained == 0 || h.held.clocks == 0 || !config.bounds.scan.fits(providerScanOpenUpper(config), h.held) {
		return nil, errProviderIntegration
	}
	m["ordinal"], m["previous_highwater"], m["target"] = strconv.FormatUint(h.ordinal, 10), providerNullableDigest(h.previous), providerNullableDigest(h.target)
	m["attempt_intent"], m["attempt_commit"] = providerNullableDigest(h.intent), providerNullableDigest(h.commit)
	m["held_work"], m["held_retained"], m["held_clock_reads"] = candidateLedgerWorkObject(h.held.work), strconv.FormatUint(h.held.retained, 10), strconv.FormatUint(h.held.clocks, 10)
	return x.encode(m)
}

func (x *providerScanCodec) parseHighwater(raw []byte) (providerScanHighwater, error) {
	config := x.config
	if len(raw) == 0 || len(raw) > candidateLedgerMetadataLimit || uint64(len(raw)) > config.bounds.maximumMetadataPerCommit {
		return providerScanHighwater{}, errProviderIntegration
	}
	m := contentObject(x.decode(raw), "format", "namespace", "instance_key", "generation", "fence", "profile", "ordinal", "previous_highwater", "target", "attempt_intent", "attempt_commit", "held_work", "held_retained", "held_clock_reads")
	ordinal, a := providerCandidateUint(m["ordinal"], false)
	held, b := providerScanUpperValue(map[string]any{"work": m["held_work"], "retained_bytes": m["held_retained"], "clock_reads": m["held_clock_reads"]})
	if m == nil || !a || !b || !providerIndexNullableDigest(m["previous_highwater"]) || !providerIndexNullableDigest(m["target"]) || !providerIndexNullableDigest(m["attempt_intent"]) || !providerIndexNullableDigest(m["attempt_commit"]) {
		return providerScanHighwater{}, errProviderIntegration
	}
	h := providerScanHighwater{ordinal, contentString(m["previous_highwater"]), contentString(m["target"]), contentString(m["attempt_intent"]), contentString(m["attempt_commit"]), held}
	want, err := x.highwater(h)
	if err != nil || !x.equal(raw, want) {
		return providerScanHighwater{}, errProviderIntegration
	}
	return h, nil
}

func providerScanOpenUpper(config providerCustodyConfig) providerScanUpper {
	b := config.bounds
	return providerScanUpper{b.setupWork, b.setupRetained, b.setupMaximumClockReads}
}

func (x *providerScanCodec) head(h providerScanHighwater) ([]byte, error) {
	raw, err := x.highwater(h)
	m := x.identity(providerScanHeadDomain)
	if err != nil || m == nil {
		return nil, errProviderIntegration
	}
	digest := x.hash(providerScanHighwaterDomain, raw)
	if !lowerHash(digest) {
		return nil, errProviderIntegration
	}
	m["ordinal"], m["highwater"] = strconv.FormatUint(h.ordinal, 10), digest
	return x.encode(m)
}

func (x *providerScanCodec) attempt(a providerScanAttempt) ([]byte, error) {
	config := x.config
	m := x.identity(providerScanIntentDomain)
	preparation, prepErr := providerScanPreparationUpper(config, a.ordinal)
	body, bodyErr := providerScanBodyUpper(config)
	if prepErr != nil || bodyErr != nil || a.preparation != preparation || a.body != body {
		return nil, errProviderIntegration
	}
	if m == nil || a.ordinal == 0 || a.ordinal > config.bounds.scan.maximumAttempts || (a.ordinal == 1) != (a.previousCommit == "") || a.previousCommit != "" && !lowerHash(a.previousCommit) || !lowerHash(a.previousHighwater) || !lowerHash(a.setup) || !lowerHash(a.custody) || !lowerHash(a.target) || (a.purpose != "initial-inspection" && a.purpose != "recovery-inspection") || !candidateLedgerClockAdvance(a.clock, a.clock) || candidateLedgerExpired(a.clock, a.deadline) || a.preparation.retained == 0 || a.preparation.clocks == 0 || a.body.retained == 0 || a.body.clocks == 0 {
		return nil, errProviderIntegration
	}
	joined, ok := providerScanUpperSum(a.preparation, a.body)
	if !ok || joined.work[0] != 0 || joined.work[1] != 0 || joined.work[2] == 0 || joined.work[9] == 0 || joined.work[10] == 0 || !config.bounds.scan.fits(providerScanOpenUpper(config), joined) {
		return nil, errProviderIntegration
	}
	m["attempt"], m["previous_scan_commit"], m["previous_highwater"] = strconv.FormatUint(a.ordinal, 10), providerNullableDigest(a.previousCommit), a.previousHighwater
	m["expected_setup_head"], m["expected_custody_commit"], m["target"], m["purpose"] = a.setup, a.custody, a.target, a.purpose
	m["original_clock"], m["deadline_wall_ms"] = providerDiagnosticClockObject(a.clock), strconv.FormatUint(a.deadline, 10)
	m["preparation_upper"], m["body_upper"] = a.preparation.object(), a.body.object()
	return x.encode(m)
}

func (x *providerScanCodec) parseAttempt(raw []byte) (providerScanAttempt, error) {
	config := x.config
	if len(raw) == 0 || len(raw) > candidateLedgerMetadataLimit || uint64(len(raw)) > config.bounds.maximumMetadataPerCommit {
		return providerScanAttempt{}, errProviderIntegration
	}
	m := contentObject(x.decode(raw), "format", "namespace", "instance_key", "generation", "fence", "profile", "attempt", "previous_scan_commit", "previous_highwater", "expected_setup_head", "expected_custody_commit", "target", "purpose", "original_clock", "deadline_wall_ms", "preparation_upper", "body_upper")
	n, nOK := providerCandidateUint(m["attempt"], true)
	deadline, dOK := providerCandidateUint(m["deadline_wall_ms"], true)
	clock, cOK := providerStoredClock(m["original_clock"])
	preparation, pOK := providerScanUpperValue(m["preparation_upper"])
	body, bOK := providerScanUpperValue(m["body_upper"])
	if m == nil || !nOK || !dOK || !cOK || !pOK || !bOK || !providerIndexNullableDigest(m["previous_scan_commit"]) {
		return providerScanAttempt{}, errProviderIntegration
	}
	a := providerScanAttempt{n, contentString(m["previous_scan_commit"]), contentString(m["previous_highwater"]), contentString(m["expected_setup_head"]), contentString(m["expected_custody_commit"]), contentString(m["target"]), contentString(m["purpose"]), clock, deadline, preparation, body}
	want, err := x.attempt(a)
	if err != nil || !x.equal(want, raw) {
		return providerScanAttempt{}, errProviderIntegration
	}
	return a, nil
}

// Reproduce Qj exactly from Qj-1 and both disjoint reservations. No field from
// a supplied commit can choose a smaller charge, ordinal or predecessor.
func (x *providerScanCodec) successor(before providerScanHighwater, attempt providerScanAttempt) (providerScanHighwater, []byte, error) {
	config := x.config
	previous, err := x.highwater(before)
	intent, intentErr := x.attempt(attempt)
	if err != nil || intentErr != nil || before.ordinal == ^uint64(0) || attempt.ordinal != before.ordinal+1 || attempt.previousHighwater != x.hash(providerScanHighwaterDomain, previous) || attempt.previousCommit != before.commit {
		return providerScanHighwater{}, nil, errProviderIntegration
	}
	held, ok := providerScanUpperSum(before.held, attempt.preparation, attempt.body)
	if !ok || !config.bounds.scan.fits(providerScanOpenUpper(config), held) {
		return providerScanHighwater{}, nil, errProviderIntegration
	}
	m := x.identity(providerScanCommitDomain)
	m["attempt"], m["previous_scan_commit"], m["previous_highwater"] = strconv.FormatUint(attempt.ordinal, 10), providerNullableDigest(before.commit), attempt.previousHighwater
	m["intent_digest"] = x.hash(providerScanIntentDomain, intent)
	m["new_cumulative_held_work"], m["new_cumulative_held_retained"], m["new_cumulative_held_clock_reads"] = candidateLedgerWorkObject(held.work), strconv.FormatUint(held.retained, 10), strconv.FormatUint(held.clocks, 10)
	commit, commitErr := x.encode(m)
	after := providerScanHighwater{attempt.ordinal, attempt.previousHighwater, attempt.target, x.hash(providerScanIntentDomain, intent), x.hash(providerScanCommitDomain, commit), held}
	if commitErr != nil {
		return providerScanHighwater{}, nil, errProviderIntegration
	}
	if _, err := x.highwater(after); err != nil {
		return providerScanHighwater{}, nil, errProviderIntegration
	}
	return after, commit, nil
}

// Pure metadata wrappers have no owner or authority. Production control paths
// use one metered codec instead, including every repeated encoding and hash.
func (h providerScanHighwater) bytes(config providerCustodyConfig) ([]byte, error) {
	x, err := newProviderScanCodec(config, nil)
	if err != nil {
		return nil, err
	}
	return x.highwater(h)
}
func (h providerScanHighwater) head(config providerCustodyConfig) ([]byte, error) {
	x, err := newProviderScanCodec(config, nil)
	if err != nil {
		return nil, err
	}
	return x.head(h)
}
func (a providerScanAttempt) bytes(config providerCustodyConfig) ([]byte, error) {
	x, err := newProviderScanCodec(config, nil)
	if err != nil {
		return nil, err
	}
	return x.attempt(a)
}
func parseProviderScanHighwater(raw []byte, config providerCustodyConfig) (providerScanHighwater, error) {
	x, err := newProviderScanCodec(config, nil)
	if err != nil {
		return providerScanHighwater{}, err
	}
	return x.parseHighwater(raw)
}
func parseProviderScanAttempt(raw []byte, config providerCustodyConfig) (providerScanAttempt, error) {
	x, err := newProviderScanCodec(config, nil)
	if err != nil {
		return providerScanAttempt{}, err
	}
	return x.parseAttempt(raw)
}
func providerScanSuccessor(config providerCustodyConfig, before providerScanHighwater, a providerScanAttempt) (providerScanHighwater, []byte, error) {
	x, err := newProviderScanCodec(config, nil)
	if err != nil {
		return providerScanHighwater{}, nil, err
	}
	return x.successor(before, a)
}

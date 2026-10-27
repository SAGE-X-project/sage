package registry011

import "strconv"

const providerScanAdmissionBindingDomain = "sage-provider-scan-admission-binding-v1"

// This closed backward-only record is DATA. Parsing it cannot reconstruct the
// live finishing owner, select a current pin, produce an ACK or run a request.
type providerScanAdmissionBinding struct {
	e0, frontier, setup, head, highwater, attempt string
	target, lastChunk, admission                  string
	ordinal, deadline, live                       uint64
	clock                                         CandidateClockDiagnostic
	open, held, total                             providerScanUpper
}

func (*providerScanAdmissionBinding) UnmarshalJSON([]byte) error { return errProviderIntegration }

func (x *providerScanCodec) admissionBinding(v providerScanAdmissionBinding) ([]byte, error) {
	if x == nil || x.config.bounds.firstAdmission == nil || x.config.bounds.scan == nil || v.ordinal == 0 || v.ordinal > x.config.bounds.scan.maximumAttempts || v.held.retained == 0 || v.held.clocks == 0 || v.open != providerScanOpenUpper(x.config) || !x.config.bounds.scan.fits(v.open, v.held) || !candidateLedgerClockAdvance(v.clock, v.clock) {
		return nil, errProviderIntegration
	}
	for _, digest := range []string{v.e0, v.frontier, v.setup, v.head, v.highwater, v.attempt, v.target, v.lastChunk, v.admission} {
		if !lowerHash(digest) {
			return nil, errProviderIntegration
		}
	}
	total, ok := providerScanUpperSum(v.open, v.held)
	deadline, duration, clockOK := x.deadline(v.clock)
	if !ok || total != v.total || v.live != total.retained || !clockOK || deadline != v.deadline || x.config.bounds.scan.totalDurationMS > duration || !x.preflight(providerEncodeEvent, x.config.bounds.maximumMetadataPerCommit) {
		return nil, errProviderIntegration
	}
	m := x.identity(providerScanAdmissionBindingDomain)
	m["e0_commit"], m["e0_frontier"], m["setup_head"] = v.e0, v.frontier, v.setup
	m["scan_head"], m["scan_highwater"], m["scan_attempt"] = v.head, v.highwater, v.attempt
	m["target"], m["last_chunk"], m["admission_digest"] = v.target, v.lastChunk, v.admission
	m["scan_ordinal"], m["deadline_wall_ms"], m["pre_i_live_upper"] = strconv.FormatUint(v.ordinal, 10), strconv.FormatUint(v.deadline, 10), strconv.FormatUint(v.live, 10)
	m["original_clock"] = providerDiagnosticClockObject(v.clock)
	m["open_upper"], m["cumulative_scan_held"], m["total_pre_i_upper"] = v.open.object(), v.held.object(), v.total.object()
	return x.encode(m)
}

func (x *providerScanCodec) parseAdmissionBinding(raw []byte) (providerScanAdmissionBinding, error) {
	m := contentObject(x.decode(raw), "format", "namespace", "instance_key", "generation", "fence", "profile", "e0_commit", "e0_frontier", "setup_head", "scan_head", "scan_highwater", "scan_attempt", "scan_ordinal", "target", "last_chunk", "admission_digest", "original_clock", "deadline_wall_ms", "open_upper", "cumulative_scan_held", "total_pre_i_upper", "pre_i_live_upper")
	ordinal, a := providerCandidateUint(m["scan_ordinal"], true)
	deadline, b := providerCandidateUint(m["deadline_wall_ms"], true)
	live, c := providerCandidateUint(m["pre_i_live_upper"], true)
	clock, d := providerStoredClock(m["original_clock"])
	open, e := providerScanUpperValue(m["open_upper"])
	held, f := providerScanUpperValue(m["cumulative_scan_held"])
	total, g := providerScanUpperValue(m["total_pre_i_upper"])
	if m == nil || !a || !b || !c || !d || !e || !f || !g {
		return providerScanAdmissionBinding{}, errProviderIntegration
	}
	v := providerScanAdmissionBinding{contentString(m["e0_commit"]), contentString(m["e0_frontier"]), contentString(m["setup_head"]), contentString(m["scan_head"]), contentString(m["scan_highwater"]), contentString(m["scan_attempt"]), contentString(m["target"]), contentString(m["last_chunk"]), contentString(m["admission_digest"]), ordinal, deadline, live, clock, open, held, total}
	want, err := x.admissionBinding(v)
	if err != nil || !x.equal(raw, want) {
		return providerScanAdmissionBinding{}, errProviderIntegration
	}
	return v, nil
}

// Reconstruct original H0..H_k under the caller's current owned finishing or
// admitted meter. Historical H_k is independently derived from its immutable
// files; a later SCAN-HEAD is neither substituted nor treated as a permit.
// The caller holds store.mu and still owes target/chunk/I/E1 typed validation.
func replayProviderAdmissionControls(s *providerCustodyStore, meter *providerWorkMeter, binding providerScanAdmissionBinding) (*providerLedgerInitialCharge, error) {
	owned := meter != nil && (meter.scan != nil && meter.scan.store == s || meter.integration != nil && meter.integration.store == s)
	if !s.valid() || meter == nil || meter.owner != meter || meter.failed || meter.closed || s.activeMeter.Load() != meter || !owned || binding.clock != providerClockDiagnostic(s.setupOrigin) {
		return nil, errProviderIntegration
	}
	x, err := newProviderScanCodec(s.config, meter)
	if err != nil {
		return nil, errProviderIntegration
	}
	if _, err := x.admissionBinding(binding); err != nil {
		return nil, errProviderIntegration
	}
	c := &providerScanControl{store: s} // Reader only; owner is deliberately nil.
	h := providerScanHighwater{}
	raw, rawErr := x.highwater(h)
	head, headErr := x.head(h)
	if rawErr != nil || headErr != nil || x.hash(providerScanHeadDomain, head) != s.config.expectedScanHead {
		return nil, errProviderIntegration
	}
	for _, disk := range []*retentionDisk{s.custody, s.evidence} {
		if !c.readExact(disk, s.name("scan-highwater", 0), raw) {
			return nil, errProviderIntegration
		}
	}
	var last providerScanAttempt
	for ordinal := uint64(1); ordinal <= binding.ordinal; ordinal++ {
		original, readErr := s.read(s.custody, s.name("scan-intent", ordinal), int64(s.config.bounds.maximumMetadataPerCommit))
		a, parseErr := x.parseAttempt(original)
		if readErr != nil || parseErr != nil || a.ordinal != ordinal || a.purpose != "initial-inspection" || a.clock != binding.clock || a.deadline != binding.deadline || !c.readExact(s.evidence, s.name("scan-claim", ordinal), original) {
			return nil, errProviderIntegration
		}
		after, commit, successorErr := x.successor(h, a)
		next, nextErr := x.highwater(after)
		if successorErr != nil || nextErr != nil || !c.readExact(s.custody, s.name("scan-commit", ordinal), commit) {
			return nil, errProviderIntegration
		}
		for _, disk := range []*retentionDisk{s.custody, s.evidence} {
			if !c.readExact(disk, s.name("scan-highwater", ordinal), next) {
				return nil, errProviderIntegration
			}
		}
		h, last = after, a
	}
	raw, rawErr = x.highwater(h)
	head, headErr = x.head(h)
	total, ok := providerScanUpperSum(providerScanOpenUpper(s.config), h.held)
	capacity := &providerCustodyStore{config: s.config}
	if rawErr != nil || headErr != nil || !ok || h.held != binding.held || total != binding.total || total.retained != binding.live || h.ordinal != binding.ordinal || h.target != binding.target || h.intent != binding.attempt || last.setup != binding.setup || last.custody != binding.e0 || x.hash(providerScanHighwaterDomain, raw) != binding.highwater || x.hash(providerScanHeadDomain, head) != binding.head || !capacity.firstAdmissionLimits(x) || !providerFirstAdmissionCapacity(capacity, total) || meter.failed {
		return nil, errProviderIntegration
	}
	return &providerLedgerInitialCharge{total.work, total.retained}, nil
}

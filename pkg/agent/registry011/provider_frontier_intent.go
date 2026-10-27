package registry011

import "strconv"

// The custodian constructs this from its selected, fully verified old state.
// A parsed link or record has no independent authority and cannot open a store.
type providerFrontierLink struct {
	identity                                     ProviderCandidateIdentity
	fence, sequence                              uint64
	commit, frontier, setup, enrollment, objects string
}

func (p providerFrontierLink) valid() bool {
	if !lowerHash(p.identity.Namespace) || !lowerHash(p.identity.Instance) || !lowerHash(p.identity.Policy) || p.fence == 0 || p.sequence == ^uint64(0) || !lowerHash(p.commit) || !lowerHash(p.setup) || !lowerHash(p.objects) {
		return false
	}
	if p.sequence == 0 {
		return p.frontier == "" && p.enrollment == ""
	}
	return lowerHash(p.frontier) && lowerHash(p.enrollment)
}

func providerFrontierIntent(p providerFrontierLink, root *CandidateEnrollmentRoot, inventory *providerIntegrationInventory, kind, operation string, clock ClockSample, bounds providerIntegrationBounds) ([]byte, error) {
	if !p.valid() || bounds.continuation == nil || !bounds.valid() || !providerIndexedCustodyKind(kind) || !lowerHash(operation) || root == nil || root.identity != p.identity || root.fence != p.fence || root.raw == "" || inventory == nil || inventory.previous != p.objects || (p.sequence == 0) != (kind == "enroll") {
		return nil, errProviderIntegration
	}
	parsed, err := ParseCandidateEnrollmentRoot(&ProviderCandidateContext{identity: p.identity, bound: true}, []byte(root.raw), 64<<10)
	if err != nil || parsed.digest != root.digest || parsed.sequence != root.sequence || parsed.budget != root.budget || parsed.fence != root.fence {
		return nil, errProviderIntegration
	}
	if p.sequence == 0 && (root.sequence != 0 || root.budget != "" || root.status != "enrolling") {
		return nil, errProviderIntegration
	}
	if p.sequence != 0 && (root.sequence != p.sequence || root.previous != p.enrollment) {
		return nil, errProviderIntegration
	}
	checked, err := parseProviderFrontierInventory([]byte(inventory.raw), p.objects, bounds.maximumEntries, bounds.maximumObjectBytes, min(bounds.maximumStoredBytes, 64<<20))
	if err != nil || checked.digest != inventory.digest {
		return nil, errProviderIntegration
	}
	sample := providerClockDiagnostic(clock)
	if !candidateLedgerClockAdvance(sample, sample) || sample.ClockID == "" || !lowerHash(sample.Epoch) {
		return nil, errProviderIntegration
	}
	raw := providerIntegrationJSON(map[string]any{
		"format": "sage-provider-custody-intent-v2", "namespace": p.identity.Namespace, "instance_key": p.identity.Instance,
		"generation": strconv.FormatUint(p.identity.Generation, 10), "fence": strconv.FormatUint(p.fence, 10), "sequence": strconv.FormatUint(p.sequence+1, 10),
		"previous_commit": p.commit, "expected_enrollment_root": providerNullableDigest(p.enrollment), "new_enrollment_root": root.digest,
		"objects_root": inventory.digest, "operation": operation, "kind": kind, "policy": p.identity.Policy, "clock": providerClockObject(clock),
		"metadata_reservation": providerCustodyMetadata(bounds), "previous_frontier": providerNullableDigest(p.frontier), "setup_head": p.setup,
	})
	if len(raw) == 0 || uint64(len(raw)) > bounds.maximumMetadataPerCommit {
		return nil, errProviderIntegration
	}
	return raw, nil
}

// Closed format checking is deliberately followed by independently deriving
// the exact intent from the old private link, typed root and inventory.
func checkProviderFrontierIntent(raw []byte, p providerFrontierLink, root *CandidateEnrollmentRoot, inventory *providerIntegrationInventory, kind, operation string, bounds providerIntegrationBounds) bool {
	if len(raw) == 0 || uint64(len(raw)) > bounds.maximumMetadataPerCommit {
		return false
	}
	m := providerFrontierIntentObject(raw)
	if m == nil {
		return false
	}
	clock, ok := providerStoredClock(m["clock"])
	if !ok {
		return false
	}
	want, err := providerFrontierIntent(p, root, inventory, kind, operation, ClockSample{ClockID: clock.ClockID, Epoch: clock.Epoch, WallMS: clock.WallMS, MonotonicMS: clock.MonotonicMS, UncertaintyMS: clock.UncertaintyMS}, bounds)
	return err == nil && string(raw) == string(want)
}

func providerFrontierIntentObject(raw []byte) map[string]any {
	if len(raw) == 0 || len(raw) > 1<<20 {
		return nil
	}
	m := contentObject(contentJSON(raw), "format", "namespace", "instance_key", "generation", "fence", "sequence", "previous_commit", "expected_enrollment_root", "new_enrollment_root", "objects_root", "operation", "kind", "policy", "clock", "metadata_reservation", "previous_frontier", "setup_head")
	if m == nil || m["format"] != "sage-provider-custody-intent-v2" {
		return nil
	}
	return m
}

func providerFrontierCommit(intent []byte) []byte {
	m := providerFrontierIntentObject(intent)
	if m == nil {
		return nil
	}
	commit := make(map[string]any, 13)
	for _, name := range []string{"namespace", "instance_key", "generation", "fence", "sequence", "previous_commit", "new_enrollment_root", "objects_root", "operation", "previous_frontier", "setup_head"} {
		commit[name] = m[name]
	}
	commit["format"], commit["intent_digest"] = "sage-provider-custody-commit-v2", providerCandidateHash("sage-provider-custody-intent-v2", intent)
	return providerIntegrationJSON(commit)
}

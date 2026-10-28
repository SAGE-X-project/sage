package registry011

import "strconv"

// Closed retained DATA only. Neither parsing this value nor comparing two
// records creates the separately selected live custodian principal/private pin.
// The concrete CAS and exact readback remain required before any private ACK.
type providerFrontierHighwater struct {
	identity                                   ProviderCandidateIdentity
	fence, ordinal, sequence                   uint64
	previous, profile, commit, setup, frontier string
}

func (h providerFrontierHighwater) valid(config providerCustodyConfig) bool {
	return h.validObserved(config, nil)
}

func (h providerFrontierHighwater) validObserved(config providerCustodyConfig, meter *providerWorkMeter) bool {
	if config.bounds.continuation == nil || !config.bounds.valid() || !config.capture.valid() || !config.bounds.methods.fits(config.capture) || h.identity != config.identity || !lowerHash(h.identity.Namespace) || !lowerHash(h.identity.Instance) || !lowerHash(h.identity.Policy) || !lowerHash(h.identity.SourceID) || !lowerHash(h.identity.Provider) || h.fence == 0 || h.profile != config.profile || h.profile != providerIntegrationProfileObserved(config.bounds, config.capture, meter) || !lowerHash(h.commit) || !lowerHash(h.setup) || h.sequence > uint64(config.bounds.maximumCommits) {
		return false
	}
	maxOrdinal, ok := providerUpperSum(uint64(config.bounds.maximumCommits), uint64(config.bounds.maximumSetupRecoveries))
	if !ok || h.ordinal > maxOrdinal || h.ordinal < h.sequence || h.ordinal-h.sequence > uint64(config.bounds.maximumSetupRecoveries) || (h.ordinal == 0) != (h.previous == "") || h.previous != "" && !lowerHash(h.previous) {
		return false
	}
	if h.sequence == 0 {
		return h.frontier == "" && h.commit == config.anchor && h.fence == 1 && h.identity.Generation == 0 && h.ordinal <= uint64(config.bounds.maximumSetupRecoveries)
	}
	return lowerHash(h.frontier)
}

func (h providerFrontierHighwater) bytes(config providerCustodyConfig) ([]byte, error) {
	return h.bytesObserved(config, nil)
}

func (h providerFrontierHighwater) bytesObserved(config providerCustodyConfig, meter *providerWorkMeter) ([]byte, error) {
	if !h.validObserved(config, meter) {
		return nil, errProviderIntegration
	}
	raw := providerReplayCanonical(map[string]any{
		"format": "sage-provider-custody-highwater-v1", "namespace": h.identity.Namespace, "instance_key": h.identity.Instance,
		"generation": strconv.FormatUint(h.identity.Generation, 10), "fence": strconv.FormatUint(h.fence, 10), "ordinal": strconv.FormatUint(h.ordinal, 10),
		"previous_highwater": providerNullableDigest(h.previous), "profile": h.profile, "sequence": strconv.FormatUint(h.sequence, 10),
		"custody_commit": h.commit, "setup_head": h.setup, "frontier": providerNullableDigest(h.frontier),
	}, candidateLedgerMetadataLimit, meter)
	if len(raw) == 0 || len(raw) > candidateLedgerMetadataLimit {
		return nil, errProviderIntegration
	}
	return raw, nil
}

func parseProviderFrontierHighwater(raw []byte, config providerCustodyConfig) (providerFrontierHighwater, error) {
	return parseProviderFrontierHighwaterObserved(raw, config, nil)
}

func parseProviderFrontierHighwaterObserved(raw []byte, config providerCustodyConfig, meter *providerWorkMeter) (providerFrontierHighwater, error) {
	if len(raw) == 0 || len(raw) > candidateLedgerMetadataLimit {
		return providerFrontierHighwater{}, errProviderIntegration
	}
	m := contentObject(providerReplayJSON(raw, meter), "format", "namespace", "instance_key", "generation", "fence", "ordinal", "previous_highwater", "profile", "sequence", "custody_commit", "setup_head", "frontier")
	fence, a := providerCandidateUint(m["fence"], true)
	ordinal, b := providerCandidateUint(m["ordinal"], false)
	sequence, c := providerCandidateUint(m["sequence"], false)
	if m == nil || m["format"] != "sage-provider-custody-highwater-v1" || m["namespace"] != config.identity.Namespace || m["instance_key"] != config.identity.Instance || m["generation"] != strconv.FormatUint(config.identity.Generation, 10) || !a || !b || !c || !providerIndexNullableDigest(m["previous_highwater"]) || !providerIndexNullableDigest(m["frontier"]) {
		return providerFrontierHighwater{}, errProviderIntegration
	}
	h := providerFrontierHighwater{identity: config.identity, fence: fence, ordinal: ordinal, sequence: sequence, previous: contentString(m["previous_highwater"]), profile: contentString(m["profile"]), commit: contentString(m["custody_commit"]), setup: contentString(m["setup_head"]), frontier: contentString(m["frontier"])}
	canonical, err := h.bytesObserved(config, meter)
	if err != nil || !providerReplayEqual(canonical, raw, meter) {
		return providerFrontierHighwater{}, errProviderIntegration
	}
	return h, nil
}

func checkProviderHighwaterPredecessor(before, after providerFrontierHighwater, config providerCustodyConfig) bool {
	return checkProviderHighwaterPredecessorObserved(before, after, config, nil)
}

func checkProviderHighwaterPredecessorObserved(before, after providerFrontierHighwater, config providerCustodyConfig, meter *providerWorkMeter) bool {
	if !before.validObserved(config, meter) || !after.validObserved(config, meter) || before.identity != after.identity || before.fence != after.fence || before.profile != after.profile || before.ordinal == ^uint64(0) || after.ordinal != before.ordinal+1 {
		return false
	}
	raw, err := before.bytesObserved(config, meter)
	return err == nil && after.previous == providerReplayDigest("sage-provider-custody-highwater-v1", raw, meter)
}

func checkProviderHighwaterMain(before, after providerFrontierHighwater, config providerCustodyConfig, intent, commit, frontier []byte) bool {
	return checkProviderHighwaterMainObserved(before, after, config, intent, commit, frontier, nil)
}

func checkProviderHighwaterMainObserved(before, after providerFrontierHighwater, config providerCustodyConfig, intent, commit, frontier []byte, meter *providerWorkMeter) bool {
	if !checkProviderHighwaterPredecessorObserved(before, after, config, meter) || before.sequence == ^uint64(0) || after.sequence != before.sequence+1 || after.setup != before.setup || len(intent) == 0 || uint64(len(intent)) > config.bounds.maximumMetadataPerCommit || len(commit) == 0 || uint64(len(commit)) > config.bounds.maximumMetadataPerCommit {
		return false
	}
	m := providerFrontierIntentObjectObserved(intent, meter)
	f, err := parseProviderFrontierRecordObserved(frontier, config.identity, meter)
	if m == nil || err != nil || m["policy"] != config.identity.Policy || m["previous_commit"] != before.commit || m["previous_frontier"] != providerNullableDigest(before.frontier) || m["setup_head"] != before.setup || m["namespace"] != config.identity.Namespace || m["instance_key"] != config.identity.Instance || m["generation"] != strconv.FormatUint(config.identity.Generation, 10) || m["fence"] != strconv.FormatUint(before.fence, 10) || m["sequence"] != strconv.FormatUint(after.sequence, 10) || !providerReplayEqual(providerFrontierCommitObserved(intent, meter), commit, meter) {
		return false
	}
	canonical, err := f.bytesObserved(meter)
	clock, clockOK := providerStoredClock(m["clock"])
	return err == nil && clockOK && f.clock == clock && providerIndexedCustodyKind(contentString(m["kind"])) && providerCandidateDigest(m["operation"]) && providerReplayEqual(providerReplayCanonical(m["metadata_reservation"], candidateLedgerMetadataLimit, meter), providerReplayCanonical(providerCustodyMetadata(config.bounds), candidateLedgerMetadataLimit, meter), meter) && providerReplayEqual(providerReplayCanonical(m, config.bounds.maximumMetadataPerCommit, meter), intent, meter) && providerReplayEqual(canonical, frontier, meter) && f.fence == after.fence && f.sequence == after.sequence && f.commit == after.commit && f.commit == providerReplayDigest("sage-provider-custody-commit-v2", commit, meter) && f.setup == after.setup && f.previous == before.frontier && f.objects == m["objects_root"] && f.enrollment == m["new_enrollment_root"] && after.frontier == providerReplayDigest("sage-provider-custody-frontier-v1", frontier, meter)
}

func checkProviderHighwaterSetup(before, after providerFrontierHighwater, config providerCustodyConfig, claim, commit []byte) bool {
	return checkProviderHighwaterSetupObserved(before, after, config, claim, commit, nil)
}

func checkProviderHighwaterSetupObserved(before, after providerFrontierHighwater, config providerCustodyConfig, claim, commit []byte, meter *providerWorkMeter) bool {
	if !checkProviderHighwaterPredecessorObserved(before, after, config, meter) || after.sequence != before.sequence || after.commit != before.commit || after.frontier != before.frontier || after.setup == before.setup || len(claim) == 0 || uint64(len(claim)) > config.bounds.maximumMetadataPerCommit || len(commit) == 0 || uint64(len(commit)) > config.bounds.maximumMetadataPerCommit {
		return false
	}
	m, ok := providerReplayJSON(claim, meter).(map[string]any)
	attempt, a := providerCandidateUint(m["attempt"], false)
	clock, b := providerStoredClock(m["clock"])
	// Bound the native int conversion here as well as in the validated profile.
	if !ok || !a || !b || attempt == 0 || attempt > 64 || attempt != after.ordinal-after.sequence || attempt > uint64(config.bounds.maximumSetupRecoveries) || m["expected_tip"] != before.commit || m["previous_setup_commit"] != before.setup {
		return false
	}
	copy := config
	copy.attempt, copy.expectedTip, copy.expectedSetupTip = int(attempt), before.commit, before.setup
	return providerReplayEqual(claim, providerCustodySetupDiagnosticObserved(copy, clock, meter), meter) && providerReplayEqual(commit, providerSetupCommitObserved(claim, meter), meter) && after.setup == providerReplayDigest("sage-provider-authority-setup-commit-v1", commit, meter)
}

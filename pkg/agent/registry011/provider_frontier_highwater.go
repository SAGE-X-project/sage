package registry011

import (
	"bytes"
	"strconv"
)

// Closed retained DATA only. Neither parsing this value nor comparing two
// records creates the separately selected live custodian principal/private pin.
// The concrete CAS and exact readback remain required before any private ACK.
type providerFrontierHighwater struct {
	identity                                   ProviderCandidateIdentity
	fence, ordinal, sequence                   uint64
	previous, profile, commit, setup, frontier string
}

func (h providerFrontierHighwater) valid(config providerCustodyConfig) bool {
	if config.bounds.continuation == nil || !config.bounds.valid() || !config.capture.valid() || !config.bounds.methods.fits(config.capture) || h.identity != config.identity || !lowerHash(h.identity.Namespace) || !lowerHash(h.identity.Instance) || !lowerHash(h.identity.Policy) || !lowerHash(h.identity.SourceID) || !lowerHash(h.identity.Provider) || h.fence == 0 || h.profile != config.profile || h.profile != providerIntegrationProfile(config.bounds, config.capture) || !lowerHash(h.commit) || !lowerHash(h.setup) || h.sequence > uint64(config.bounds.maximumCommits) {
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
	if !h.valid(config) {
		return nil, errProviderIntegration
	}
	raw := providerIntegrationJSON(map[string]any{
		"format": "sage-provider-custody-highwater-v1", "namespace": h.identity.Namespace, "instance_key": h.identity.Instance,
		"generation": strconv.FormatUint(h.identity.Generation, 10), "fence": strconv.FormatUint(h.fence, 10), "ordinal": strconv.FormatUint(h.ordinal, 10),
		"previous_highwater": providerNullableDigest(h.previous), "profile": h.profile, "sequence": strconv.FormatUint(h.sequence, 10),
		"custody_commit": h.commit, "setup_head": h.setup, "frontier": providerNullableDigest(h.frontier),
	})
	if len(raw) == 0 || len(raw) > candidateLedgerMetadataLimit {
		return nil, errProviderIntegration
	}
	return raw, nil
}

func parseProviderFrontierHighwater(raw []byte, config providerCustodyConfig) (providerFrontierHighwater, error) {
	if len(raw) == 0 || len(raw) > candidateLedgerMetadataLimit {
		return providerFrontierHighwater{}, errProviderIntegration
	}
	m := contentObject(contentJSON(raw), "format", "namespace", "instance_key", "generation", "fence", "ordinal", "previous_highwater", "profile", "sequence", "custody_commit", "setup_head", "frontier")
	fence, a := providerCandidateUint(m["fence"], true)
	ordinal, b := providerCandidateUint(m["ordinal"], false)
	sequence, c := providerCandidateUint(m["sequence"], false)
	if m == nil || m["format"] != "sage-provider-custody-highwater-v1" || m["namespace"] != config.identity.Namespace || m["instance_key"] != config.identity.Instance || m["generation"] != strconv.FormatUint(config.identity.Generation, 10) || !a || !b || !c || !providerIndexNullableDigest(m["previous_highwater"]) || !providerIndexNullableDigest(m["frontier"]) {
		return providerFrontierHighwater{}, errProviderIntegration
	}
	h := providerFrontierHighwater{identity: config.identity, fence: fence, ordinal: ordinal, sequence: sequence, previous: contentString(m["previous_highwater"]), profile: contentString(m["profile"]), commit: contentString(m["custody_commit"]), setup: contentString(m["setup_head"]), frontier: contentString(m["frontier"])}
	canonical, err := h.bytes(config)
	if err != nil || !bytes.Equal(canonical, raw) {
		return providerFrontierHighwater{}, errProviderIntegration
	}
	return h, nil
}

func checkProviderHighwaterPredecessor(before, after providerFrontierHighwater, config providerCustodyConfig) bool {
	if !before.valid(config) || !after.valid(config) || before.identity != after.identity || before.fence != after.fence || before.profile != after.profile || before.ordinal == ^uint64(0) || after.ordinal != before.ordinal+1 {
		return false
	}
	raw, err := before.bytes(config)
	return err == nil && after.previous == providerCandidateHash("sage-provider-custody-highwater-v1", raw)
}

func checkProviderHighwaterMain(before, after providerFrontierHighwater, config providerCustodyConfig, intent, commit, frontier []byte) bool {
	if !checkProviderHighwaterPredecessor(before, after, config) || before.sequence == ^uint64(0) || after.sequence != before.sequence+1 || after.setup != before.setup || len(intent) == 0 || uint64(len(intent)) > config.bounds.maximumMetadataPerCommit || len(commit) == 0 || uint64(len(commit)) > config.bounds.maximumMetadataPerCommit {
		return false
	}
	m := providerFrontierIntentObject(intent)
	f, err := parseProviderFrontierRecord(frontier, config.identity)
	if m == nil || err != nil || m["policy"] != config.identity.Policy || m["previous_commit"] != before.commit || m["previous_frontier"] != providerNullableDigest(before.frontier) || m["setup_head"] != before.setup || m["namespace"] != config.identity.Namespace || m["instance_key"] != config.identity.Instance || m["generation"] != strconv.FormatUint(config.identity.Generation, 10) || m["fence"] != strconv.FormatUint(before.fence, 10) || m["sequence"] != strconv.FormatUint(after.sequence, 10) || !bytes.Equal(providerFrontierCommit(intent), commit) {
		return false
	}
	canonical, err := f.bytes()
	clock, clockOK := providerStoredClock(m["clock"])
	return err == nil && clockOK && f.clock == clock && providerIndexedCustodyKind(contentString(m["kind"])) && providerCandidateDigest(m["operation"]) && bytes.Equal(providerIntegrationJSON(m["metadata_reservation"]), providerIntegrationJSON(providerCustodyMetadata(config.bounds))) && bytes.Equal(providerIntegrationJSON(m), intent) && bytes.Equal(canonical, frontier) && f.fence == after.fence && f.sequence == after.sequence && f.commit == after.commit && f.commit == providerCandidateHash("sage-provider-custody-commit-v2", commit) && f.setup == after.setup && f.previous == before.frontier && f.objects == m["objects_root"] && f.enrollment == m["new_enrollment_root"] && after.frontier == providerCandidateHash("sage-provider-custody-frontier-v1", frontier)
}

func checkProviderHighwaterSetup(before, after providerFrontierHighwater, config providerCustodyConfig, claim, commit []byte) bool {
	if !checkProviderHighwaterPredecessor(before, after, config) || after.sequence != before.sequence || after.commit != before.commit || after.frontier != before.frontier || after.setup == before.setup || len(claim) == 0 || uint64(len(claim)) > config.bounds.maximumMetadataPerCommit || len(commit) == 0 || uint64(len(commit)) > config.bounds.maximumMetadataPerCommit {
		return false
	}
	m, ok := contentJSON(claim).(map[string]any)
	attempt, a := providerCandidateUint(m["attempt"], false)
	clock, b := providerStoredClock(m["clock"])
	if !ok || !a || !b || attempt == 0 || attempt != after.ordinal-after.sequence || attempt > uint64(config.bounds.maximumSetupRecoveries) || m["expected_tip"] != before.commit || m["previous_setup_commit"] != before.setup {
		return false
	}
	copy := config
	copy.attempt, copy.expectedTip, copy.expectedSetupTip = int(attempt), before.commit, before.setup
	return bytes.Equal(claim, providerCustodySetupDiagnostic(copy, clock)) && bytes.Equal(commit, providerSetupCommit(claim)) && after.setup == providerCandidateHash("sage-provider-authority-setup-commit-v1", commit)
}

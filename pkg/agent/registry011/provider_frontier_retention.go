package registry011

import "strconv"

// These exact byte counts are derived from an already constructed new delta.
// They are not measurements of physical allocation, a reservation, a current
// tip or an acknowledgment. The independent custodian must derive them again
// from retained original bytes before accepting any frontier.
type providerFrontierRetention struct {
	payload, metadata, stored, objects uint64
}

func providerFrontierRetentionCost(id ProviderCandidateIdentity, inventory *providerIntegrationInventory, fresh []providerIntegrationObject, intent, commit, frontier []byte, bounds providerIntegrationBounds) (providerFrontierRetention, error) {
	if inventory == nil || bounds.continuation == nil || !bounds.valid() || len(fresh) == 0 || len(fresh) > bounds.maximumEntries || len(fresh) != len(inventory.entries) || len(inventory.raw) == 0 || len(inventory.raw) > 64<<10 || len(intent) == 0 || uint64(len(intent)) > bounds.maximumMetadataPerCommit || len(commit) == 0 || uint64(len(commit)) > bounds.maximumMetadataPerCommit || len(frontier) == 0 || len(frontier) > 64<<10 {
		return providerFrontierRetention{}, errProviderIntegration
	}
	// Check the entire finite new-delta byte/codec upper before hashing, parsing
	// or constructing a second inventory. Resident global maps and old objects
	// have their own additional preflight at the concrete custodian boundary.
	metadata, ok := providerUpperSum(uint64(len(inventory.raw)), uint64(2*len(intent)+len(commit)+len(frontier)))
	if !ok {
		return providerFrontierRetention{}, errProviderIntegration
	}
	var rawBytes, payload, extra uint64
	for _, object := range fresh {
		if len(object.raw) == 0 || uint64(len(object.raw)) > bounds.maximumObjectBytes {
			return providerFrontierRetention{}, errProviderIntegration
		}
		rawBytes, ok = providerUpperSum(rawBytes, uint64(len(object.raw)))
		if !ok || rawBytes > 64<<20 {
			return providerFrontierRetention{}, errProviderIntegration
		}
		class := providerFrontierRetentionClass(object.kind)
		switch class {
		case 1:
			payload, ok = providerUpperSum(payload, uint64(len(object.raw)))
		case 2:
			extra, ok = providerUpperSum(extra, uint64(len(object.raw)))
		case 0: // Already charged by exact typed ledger/index/page reconstruction.
		default:
			return providerFrontierRetention{}, errProviderIntegration
		}
		if !ok {
			return providerFrontierRetention{}, errProviderIntegration
		}
	}
	scratch, ok := providerUpperSum(rawBytes, metadata)
	if !ok {
		return providerFrontierRetention{}, errProviderIntegration
	}
	if _, ok = bounds.continuation.residentUpper(0, 0, 0, uint64(len(fresh)), scratch, scratch); !ok {
		return providerFrontierRetention{}, errProviderIntegration
	}
	derived, err := newProviderFrontierInventory(inventory.previous, fresh, bounds.maximumEntries, min(bounds.maximumStoredBytes, 64<<20))
	if err != nil || derived.raw != inventory.raw || derived.digest != inventory.digest || derived.payloadBytes != inventory.payloadBytes {
		return providerFrontierRetention{}, errProviderIntegration
	}
	m := providerFrontierIntentObject(intent)
	f, err := parseProviderFrontierRecord(frontier, id)
	if m == nil || err != nil || m["namespace"] != id.Namespace || m["instance_key"] != id.Instance || m["generation"] != strconv.FormatUint(id.Generation, 10) || m["sequence"] != strconv.FormatUint(f.sequence, 10) || m["fence"] != strconv.FormatUint(f.fence, 10) || m["objects_root"] != inventory.digest || string(providerFrontierCommit(intent)) != string(commit) || f.commit != providerCandidateHash("sage-provider-custody-commit-v2", commit) || f.objects != inventory.digest || f.enrollment != m["new_enrollment_root"] || f.setup != m["setup_head"] || providerNullableDigest(f.previous) != m["previous_frontier"] {
		return providerFrontierRetention{}, errProviderIntegration
	}
	canonical, err := f.bytes()
	if err != nil || string(canonical) != string(frontier) || string(providerIntegrationJSON(m)) != string(intent) {
		return providerFrontierRetention{}, errProviderIntegration
	}
	metadata, ok = providerUpperSum(metadata, extra)
	if !ok || payload > bounds.continuation.maximumPayloadBytes || metadata > bounds.continuation.maximumMetadataBytes {
		return providerFrontierRetention{}, errProviderIntegration
	}
	// Five new metadata files: intent, pending, commit, inventory and frontier.
	files, ok := providerUpperSum(uint64(len(fresh)), 5)
	allocation, mulOK := providerUpperProduct(files, bounds.fileOverheadBytes)
	stored, sumOK := providerUpperSum(rawBytes, uint64(len(inventory.raw)), uint64(2*len(intent)+len(commit)+len(frontier)), allocation)
	if !ok || !mulOK || !sumOK || stored > bounds.maximumStoredBytes {
		return providerFrontierRetention{}, errProviderIntegration
	}
	return providerFrontierRetention{payload: payload, metadata: metadata, stored: stored, objects: uint64(len(fresh))}, nil
}

// Closed accounting classes prevent a new kind from silently becoming free.
// Class0 is charged exactly once by the indicated independent typed owner.
func providerFrontierRetentionClass(kind string) int {
	switch kind {
	case "ticket", "consumption", "settlement", "ledger-node", "budget", "ledger-page", "ledger-seal", "index-empty", "index-mutation":
		return 0
	case "request", "params", "response", "tls":
		return 1
	case "setup", "setup-commit", "enrollment", "admission", "exchange", "wire-binding", "bootstrap", "pin-input", "pin", "segment", "batch", "checkpoint", "verification-receipt":
		return 2
	}
	return -1
}

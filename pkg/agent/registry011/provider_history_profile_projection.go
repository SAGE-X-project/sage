package registry011

// Immutable serialized DATA for the final profile schema, including the
// selected collection admission maximum. This is not a source derivation,
// feasibility certificate, owner or permission to open the selected paths.
// Compiler work, copies and backing/peak/lifetime costs remain to be funded
// by the enclosing complete compiler before it may create a real derivation.
type providerHistoryProfileProjection struct {
	raw, domain string
}

// Project the bytes that the ordinary serializer must produce AFTER a real
// derivation is constructed. Quoting the unminted ordinary profile would omit
// collection_admission_maximum and price a different schema. No temporary or
// synthetic derivation is installed to reach the final schema here.
func providerHistoryProjectFinalProfile(config providerCustodyConfig, maximum providerHistoryGroupUpper) (providerHistoryProfileProjection, bool) {
	var empty providerHistoryProfileProjection
	b := config.bounds
	if b.history == nil || !lowerHash(b.history.program) || !lowerHash(b.history.costs) {
		return empty, false
	}
	if _, ok := providerHistorySourceLimitsFromConfig(config); !ok || !maximum.valid(providerHistoryGroupMembers) {
		return empty, false
	}
	wire, wireOK := providerUpperSum(uint64(config.capture.WireBytes), 1)
	wire, totalOK := providerUpperProduct(providerHistoryGroupMembers, wire)
	if !wireOK || !totalOK || maximum.execution[1] != wire {
		return empty, false
	}
	store := b.profileObjectWithHistoryAdmission(&maximum)
	raw, domain := providerIntegrationProfileBytesForStore(b, config.capture, store)
	if len(raw) == 0 || uint64(len(raw)) > b.maximumMetadataPerCommit {
		return empty, false
	}
	return providerHistoryProfileProjection{raw: string(raw), domain: domain}, true
}

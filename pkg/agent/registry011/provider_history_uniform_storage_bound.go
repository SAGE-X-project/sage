package registry011

// Storage assigned by the current uniform T admission: every T retains its
// complete 32-member storage maximum, even when forMembers narrows RPC/wire.
// Successful S storage and its terminal-failure addition are both retained.
// These numeric rows are not a reservation, source witness or native profile.
type providerHistoryUniformStorageCost struct {
	groups     uint64
	successful providerHistoryCollectionWriteCost
	terminal   providerHistoryStorageCharge
	stage      providerHistoryStorageCharge
	future     providerHistoryStorageCharge
}

// Compile the existing structural writer/allocation components into the FULL
// admitted-stage SUM. Do not substitute grammar.requests for 32*groups: the
// current forMembers leaves all storage coordinates at their original maximum.
// Initial/pre-I storage, uncertain failures outside these components, native
// filesystem overhead qualification, RAM, work, clocks and time remain separate.
func providerHistoryCollectionUniformStorageUpper(manifest *providerHistoryCollectionManifest, methods providerMethodBounds, certificates providerTLSBounds, maximum, metadata, overhead uint64) (providerHistoryUniformStorageCost, bool) {
	var empty providerHistoryUniformStorageCost
	if !manifest.validSnapshot() || manifest.counts.groups == 0 || manifest.counts.groups > (1<<20)/3 {
		return empty, false
	}
	successful, successOK := providerHistoryCollectionSuccessfulWriteUpper(1, providerHistoryGroupMembers, maximum, uint64(manifest.input.capture.WireBytes), overhead)
	transport, transportOK := providerHistoryCollectionTransportRoleUpper(manifest, methods, certificates, maximum, metadata, overhead)
	if !successOK || !transportOK {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	l, r := successful.storage, transport.failedStorage
	stage := providerHistoryStorageCharge{
		physical: a.sum(l.physical, r.physical), raw: a.sum(l.raw, r.raw),
		payload: a.sum(l.payload, r.payload), metadata: a.sum(l.metadata, r.metadata),
		objects: a.sum(l.objects, r.objects), files: a.sum(l.files, r.files),
	}
	if a.failed {
		return empty, false
	}
	future, ok := providerHistoryStorageRepeatedUpper(stage, manifest.counts.groups)
	if !ok {
		return empty, false
	}
	return providerHistoryUniformStorageCost{manifest.counts.groups, successful, transport.failedStorage, stage, future}, true
}

// Checked arithmetic DATA only. A finite result can exceed a selected profile
// or the unchanged absolute limits, and is therefore not feasibility evidence.
func providerHistoryStorageRepeatedUpper(stage providerHistoryStorageCharge, groups uint64) (providerHistoryStorageCharge, bool) {
	if groups == 0 || groups > (1<<20)/3 {
		return providerHistoryStorageCharge{}, false
	}
	var a providerHistoryReadArithmetic
	out := providerHistoryStorageCharge{
		physical: a.product(stage.physical, groups), raw: a.product(stage.raw, groups),
		payload: a.product(stage.payload, groups), metadata: a.product(stage.metadata, groups),
		objects: a.product(stage.objects, groups), files: a.product(stage.files, groups),
	}
	if a.failed {
		return providerHistoryStorageCharge{}, false
	}
	return out, true
}

// Only the assigned storage component is compared. Passing does not account
// for the other source components or supply the missing production derivation.
// In particular maximum.valid keeps the existing absolute storage ceilings.
func (c providerHistoryUniformStorageCost) within(maximum providerHistoryGroupTotals) bool {
	if c.groups == 0 || !maximum.valid() {
		return false
	}
	_, ok := (providerHistoryStorageCharge{}).add(c.future, maximum)
	return ok
}

package registry011

import "context"

// Private source-derived per-group maximum. It is part of the original
// nonserializable pre-H0 derivation, not a caller budget or a collection permit.
// No production constructor exists until the full source/native/RAM/final
// compiler is complete. Tests may select explicit synthetic inputs before H0.
type providerHistoryCollectionAdmissionBounds struct {
	owner    *providerHistoryProgramDerivation
	manifest *providerHistoryCollectionManifest
	maximum  providerHistoryGroupUpper
}

func (*providerHistoryCollectionAdmissionBounds) UnmarshalJSON([]byte) error { return ErrConfiguration }

// Only RPC/cardinality-dependent wire are refined to the exact original table
// count. Every other coordinate/storage/clock row keeps its complete maximum;
// unused work remains inside the one irrevocable original whole-program hold.
func (b providerHistoryCollectionAdmissionBounds) forMembers(d *providerHistoryProgramDerivation, manifest *providerHistoryCollectionManifest, capture ProviderCaptureLimits, count int) (providerHistoryGroupUpper, bool) {
	if d == nil || d.owner != d || b.owner != d || d.admission != b || manifest == nil || b.manifest != manifest || !manifest.validSnapshot() ||
		!capture.valid() || manifest.input.capture != capture || count < 1 || count > providerHistoryGroupMembers || !b.maximum.valid(providerHistoryGroupMembers) {
		return providerHistoryGroupUpper{}, false
	}
	perMember, a := providerUpperSum(uint64(capture.WireBytes), 1)
	maximum, c := providerUpperProduct(providerHistoryGroupMembers, perMember)
	wire, e := providerUpperProduct(uint64(count), perMember)
	if !a || !c || !e || b.maximum.execution[1] != maximum {
		return providerHistoryGroupUpper{}, false
	}
	upper := b.maximum
	upper.execution[0], upper.execution[1] = uint64(count), wire
	return upper, upper.valid(count)
}

func (o *providerHistoryGroupOwner) collectionAdmissionUpperLocked() (int, providerHistoryGroupUpper, bool) {
	if o == nil || o.parent == nil || o.parent.bounds.collection == nil || !o.parent.matchesLocked() ||
		o.coordinator == nil || o.custodian == nil || o.coordinator.collection == nil || o.custodian.collection == nil ||
		o.coordinator.collection == o.custodian.collection {
		return 0, providerHistoryGroupUpper{}, false
	}
	left, a := o.coordinator.collection.admissionCountLocked()
	right, c := o.custodian.collection.admissionCountLocked()
	if !a || !c || left != right {
		return 0, providerHistoryGroupUpper{}, false
	}
	upper, ok := o.parent.collectionAdmission.forMembers(o.parent.bounds.derivation, o.parent.bounds.collection, o.store.config.capture, left)
	return left, upper, ok
}

// This occurs INSIDE the original admitted T window, after its initial sample.
// Each pass reads the complete actual initial/prior T/C/S independently. No
// cached cursor, caller purpose, response-derived list or hash is authority.
func (o *providerHistoryGroupOwner) collectionAdmissionTableLocked(ctx context.Context, count int) (*providerHistoryGroupTable, error) {
	if ctx == nil || ctx.Err() != nil || o == nil || o.parent == nil || o.parent.bounds.collection == nil ||
		o.phase != "preparing-admit" || !o.activeLocked(o.meter) || !o.parent.matchesLocked() || count < 1 || count > providerHistoryGroupMembers {
		return nil, errProviderIntegration
	}
	m := o.meter
	var models [2]*providerHistoryGroupLedger
	var tables [2]*providerHistoryGroupTable
	for role := range 2 {
		prefix, err := o.replayHealthyPrefixLocked(ctx, o.stageBefore, o.store.frontierFiles.pinRaw)
		if err != nil || prefix == nil || prefix.ledger == nil || prefix.ledger.collection == nil ||
			prefix.pin != o.pin || prefix.inventory != o.inventory || !prefix.ledger.validObserved(m) ||
			!o.coordinator.validObserved(m) || !o.custodian.validObserved(m) ||
			!providerReplayTextEqual(prefix.ledger.head, o.coordinator.head, m) || !providerReplayTextEqual(prefix.ledger.head, o.custodian.head, m) {
			return nil, errProviderIntegration
		}
		table, err := prefix.ledger.collection.nextTableObserved(o.nextStageLocked(), *o.store.config.bounds.methods, o.store.config.capture, m)
		if err != nil || table == nil || table.count != count || table.stage != o.nextStageLocked() {
			return nil, errProviderIntegration
		}
		models[role], tables[role] = prefix.ledger, table
	}
	if models[0] == models[1] || models[0].collection == models[1].collection || models[0].collection.semantic == models[1].collection.semantic ||
		tables[0] == tables[1] || !providerHistoryCollectionTableEqualObserved(tables[0], tables[1], m) || ctx.Err() != nil ||
		!o.parent.matchesLocked() || !o.identityLocked() {
		return nil, errProviderIntegration
	}
	return tables[0], nil
}

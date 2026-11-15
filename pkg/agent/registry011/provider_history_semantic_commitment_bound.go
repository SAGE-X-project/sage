package registry011

// Counts are prospective maxima for two separately owned lifetime indexes.
// They do not identify an actual fold or authorize a read/execution observer.
type providerHistorySemanticCommitmentLimits struct {
	ConfirmedClaims, ConfirmedCreated, WorkingClaims, WorkingCreated, Queue uint64
	Window, Pending                                                         bool
}

// One original stream root under the inspected owned-state field/cardinality
// envelope. Universal state serialization bounds include entered refusals;
// successful raw output caps apply only to the later domain hashes.
func providerHistorySemanticStreamRootUpper(claims, created uint64) (candidateWorkVector, bool) {
	var empty candidateWorkVector
	claimRoot, claimOK := providerHistorySemanticIndexUpper(claims, true)
	createdRoot, createdOK := providerHistorySemanticIndexUpper(created, false)
	record, recordOK := providerHistoryStateSerializationUpper(providerHistoryStateSerializationLimits{Keys: 128, Services: 16, Record: true})
	grants, grantsOK := providerHistoryStateSerializationUpper(providerHistoryStateSerializationLimits{Grants: 128})
	if !claimOK || !createdOK || !recordOK || !grantsOK {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	var work candidateWorkVector
	for _, component := range [...]candidateWorkVector{claimRoot, createdRoot, record, grants} {
		for coordinate, units := range component {
			work[coordinate] = a.sum(work[coordinate], units)
		}
	}
	// Original generic metadata preflight and actual encode/hash, plus kind
	// copy<=32. This is conservative for each inspected fixed root schema.
	metadata := uint64(candidateLedgerMetadataLimit)
	value := a.sum(33, a.product(metadata, metadata), 1, metadata, uint64(len(providerHistoryCollectionStateDomain)), 2)
	// Both decimal preflights and successful record/grant domain hashes.
	work[10] = a.sum(work[10], 2*401, 65536, uint64(len("semantic-record")), 2,
		262144, uint64(len("semantic-grants")), 2, value)
	if a.failed {
		return empty, false
	}
	return work, true
}

// One complete semantic DATA commitment, including both state/index roots,
// actual queue/pending byte roots and final window/fold metadata. Index upper
// rows already include their canonical-preflight padding. Snapshot/cursor copy,
// repeated calls, native CPU/backing RAM and factory authority are separate.
func providerHistorySemanticCommitmentUpper(l providerHistorySemanticCommitmentLimits) (candidateWorkVector, bool) {
	var empty candidateWorkVector
	if l.Queue > 4096 || !l.Window && (l.WorkingClaims != 0 || l.WorkingCreated != 0 || l.Queue != 0 || l.Pending) {
		return empty, false
	}
	work, ok := providerHistorySemanticStreamRootUpper(l.ConfirmedClaims, l.ConfirmedCreated)
	if !ok {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	metadata := uint64(candidateLedgerMetadataLimit)
	value := a.sum(33, a.product(metadata, metadata), 1, metadata, uint64(len(providerHistoryCollectionStateDomain)), 2)
	if l.Window {
		working, ok := providerHistorySemanticStreamRootUpper(l.WorkingClaims, l.WorkingCreated)
		if !ok {
			return empty, false
		}
		for coordinate, units := range working {
			work[coordinate] = a.sum(work[coordinate], units)
		}
		bytesRoot := uint64(2*262144 + len("semantic-bytes") + 3)
		// Initial queue metadata exists even for zero items. The working
		// stream's own decimal/hash/state row is already in its component.
		work[10] = a.sum(work[10], 2*401, value,
			a.product(l.Queue, a.sum(401, a.product(2, bytesRoot), value)), value)
		if l.Pending {
			work[10] = a.sum(work[10], bytesRoot)
		}
	}
	work[10] = a.sum(work[10], value) // final semantic-fold metadata
	if a.failed {
		return empty, false
	}
	return work, true
}

// Parts of ONE original stream-root call on the parsed/applied owned path.
// normal bounds successful entered work only. envelope covers the complete
// caller closure and every original preflight, including deferred siblings
// after a serializer's late output-cap refusal. It is not a per-stage bound,
// reusable slack, whole-cost certificate, runtime observer or authority.
type providerHistoryAppliedStreamRootCost struct{ normal, envelope candidateWorkVector }

func providerHistoryAppliedStreamRootUpper(claims, created uint64) (providerHistoryAppliedStreamRootCost, bool) {
	var empty providerHistoryAppliedStreamRootCost
	record, recordOK := providerHistoryAppliedStateSerializationUpper(providerHistoryStateSerializationLimits{Keys: 128, Services: 16, Record: true})
	grants, grantsOK := providerHistoryAppliedStateSerializationUpper(providerHistoryStateSerializationLimits{Grants: 128})
	claimActual, claimActualOK := providerHistorySemanticIndexActualUpper(claims, true)
	createdActual, createdActualOK := providerHistorySemanticIndexActualUpper(created, false)
	claimEnvelope, claimEnvelopeOK := providerHistorySemanticIndexUpper(claims, true)
	createdEnvelope, createdEnvelopeOK := providerHistorySemanticIndexUpper(created, false)
	if !recordOK || !grantsOK || !claimActualOK || !createdActualOK || !claimEnvelopeOK || !createdEnvelopeOK {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	var out providerHistoryAppliedStreamRootCost
	for c := range out.normal {
		out.normal[c] = a.sum(record.normal[c], grants.normal[c], claimActual[c], createdActual[c])
		out.envelope[c] = a.sum(record.envelope[c], grants.envelope[c], claimEnvelope[c], createdEnvelope[c])
	}
	// The exact semantic-state schema has only generated hex hashes, uint64
	// decimal strings, this fixed domain/kind and one four-field header. Its
	// ASCII canonical maximum is697 bytes; original encode8192 stays intact.
	const raw uint64 = 697
	const domain = uint64(len(providerHistoryCollectionStateDomain))
	const copyKind = uint64(len("semantic-state")) + 1
	metadata := uint64(candidateLedgerMetadataLimit)
	normalValue := a.sum(copyKind, a.product(raw, raw), 1, raw, domain, 2)
	envelopeValue := a.sum(copyKind, a.product(metadata, metadata), 1, raw, domain, 2)
	// Both serializers run before their bools are combined. Version/admitted
	// and BOTH index roots run before that combined result is tested. A late
	// size refusal does not itself poison the meter. Keep this WHOLE closure,
	// not just the first failed primitive. Hash/value tails are conservative
	// even though a serializer refusal returns before reaching those tails.
	hashes := a.sum(record.normalBytes, uint64(len("semantic-record")), 2,
		grants.normalBytes, uint64(len("semantic-grants")), 2)
	out.normal[10] = a.sum(out.normal[10], 2*401, hashes, normalValue)
	out.envelope[10] = a.sum(out.envelope[10], 2*401, hashes, envelopeValue)
	if a.failed {
		return empty, false
	}
	return out, true
}

// Parts of ONE original fold commitment on its owned parsed/applied route.
// historyBytes is the SAME selected window's History.MaxBytes, not a report
// of queued actuals. These numeric parts do not validate or select that route.
// Every successful retain charges record+grants+entry+command before publishing
// immutable entry/command copies. Snapshot preserves those bytes and the charge;
// a failed retain may charge more, but cannot publish an uncharged queue item.
// Thus all queued entry+command bytes together are bounded by historyBytes.
// Pending command bytes are separate and retain their original262144 guard.
// The complete caller envelope includes queue/pending work deferred after a
// working-state late refusal. Repeated calls, snapshot/cursor work, native RAM,
// pointwise enclosing headroom and factory authority remain separate.
type providerHistoryAppliedCommitmentCost struct {
	normal, envelope candidateWorkVector
	queueBytes       uint64
}

func providerHistoryAppliedCommitmentUpper(l providerHistorySemanticCommitmentLimits, historyBytes uint64) (providerHistoryAppliedCommitmentCost, bool) {
	var empty providerHistoryAppliedCommitmentCost
	if l.Queue > 4096 || !l.Window && (l.WorkingClaims != 0 || l.WorkingCreated != 0 || l.Queue != 0 || l.Pending || historyBytes != 0) ||
		l.Window && (historyBytes == 0 || historyBytes > 8<<20) {
		return empty, false
	}
	confirmed, ok := providerHistoryAppliedStreamRootUpper(l.ConfirmedClaims, l.ConfirmedCreated)
	if !ok {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	out := providerHistoryAppliedCommitmentCost{normal: confirmed.normal, envelope: confirmed.envelope}
	// Closed escape-free schemas: queue91, expected-item370, window664 and
	// fold879. Hashes have66 bytes, uint64 decimal strings20, queue/ordinal4;
	// signed int counters use19 digits. Existing encode8192 guards stay intact.
	valueParts := func(kind string, raw uint64) (uint64, uint64) {
		tail := a.sum(uint64(len(kind)), 1, raw, uint64(len(providerHistoryCollectionStateDomain)), 2)
		metadata := uint64(candidateLedgerMetadataLimit)
		return a.sum(tail, a.product(raw, raw), 1), a.sum(tail, a.product(metadata, metadata), 1)
	}
	if l.Window {
		working, ok := providerHistoryAppliedStreamRootUpper(l.WorkingClaims, l.WorkingCreated)
		if !ok {
			return empty, false
		}
		for c := range out.normal {
			out.normal[c] = a.sum(out.normal[c], working.normal[c])
			out.envelope[c] = a.sum(out.envelope[c], working.envelope[c])
		}
		out.queueBytes = min(historyBytes, a.product(l.Queue, 2*262144))
		// Both byte-root fields execute before the item condition. The fixed
		// overhead is two events' +1 and the hash domain separator's +1.
		queueRoots := a.sum(a.product(2, out.queueBytes), a.product(2*l.Queue, uint64(len("semantic-bytes"))+3))
		queueNormal, queueEnvelope := valueParts("semantic-queue", 91)
		itemNormal, itemEnvelope := valueParts("semantic-expected-item", 370)
		windowNormal, windowEnvelope := valueParts("semantic-window", 664)
		out.normal[10] = a.sum(out.normal[10], 2*401, queueNormal, queueRoots, a.product(l.Queue, a.sum(401, itemNormal)), windowNormal)
		out.envelope[10] = a.sum(out.envelope[10], 2*401, queueEnvelope, queueRoots, a.product(l.Queue, a.sum(401, itemEnvelope)), windowEnvelope)
		if l.Pending {
			pendingRoot := uint64(2*262144 + len("semantic-bytes") + 3)
			out.normal[10] = a.sum(out.normal[10], pendingRoot)
			out.envelope[10] = a.sum(out.envelope[10], pendingRoot)
		}
	}
	foldNormal, foldEnvelope := valueParts("semantic-fold", 879)
	out.normal[10] = a.sum(out.normal[10], foldNormal)
	out.envelope[10] = a.sum(out.envelope[10], foldEnvelope)
	if a.failed {
		return empty, false
	}
	return out, true
}

// Conditional source facts, not producer claims or runtime selectors. Empty
// means the actual original version-zero branch with its empty record/grants,
// not merely a private object or reported version. Lifetime indexes remain
// independently bounded: other-subject commands can populate them. Actual
// ancestry has no semantic window/index entries; nonempty support is unchanged.
type providerHistoryAppliedStateKinds struct{ ConfirmedEmpty, WorkingEmpty bool }

func providerHistoryAppliedEmptyStreamRootUpper(claims, created uint64) (providerHistoryAppliedStreamRootCost, bool) {
	var empty providerHistoryAppliedStreamRootCost
	claimActual, a := providerHistorySemanticIndexActualUpper(claims, true)
	createdActual, b := providerHistorySemanticIndexActualUpper(created, false)
	claimEnvelope, c := providerHistorySemanticIndexUpper(claims, true)
	createdEnvelope, d := providerHistorySemanticIndexUpper(created, false)
	if !a || !b || !c || !d {
		return empty, false
	}
	var arithmetic providerHistoryReadArithmetic
	var out providerHistoryAppliedStreamRootCost
	for coordinate := range out.normal {
		out.normal[coordinate] = arithmetic.sum(claimActual[coordinate], createdActual[coordinate])
		out.envelope[coordinate] = arithmetic.sum(claimEnvelope[coordinate], createdEnvelope[coordinate])
	}
	// stateBytesObserved takes only its unchanged zero-work preflight; no
	// serializer or record/grant hash is entered. Both decimals/index roots
	// and the original metadata8192 guard remain.697 is conservatively kept
	// from the closed nonempty schema rather than claiming an exact empty size.
	const raw uint64 = 697
	tail := uint64(len("semantic-state")) + 1 + raw + uint64(len(providerHistoryCollectionStateDomain)) + 2
	out.normal[10] = arithmetic.sum(out.normal[10], 2*401, raw*raw+1, tail)
	out.envelope[10] = arithmetic.sum(out.envelope[10], 2*401, uint64(candidateLedgerMetadataLimit)*uint64(candidateLedgerMetadataLimit)+1, tail)
	if arithmetic.failed {
		return empty, false
	}
	return out, true
}

// Same ONE original fold closure, with source-proven empty branches removing
// only work those branches never enter. The existing full helper is unchanged.
// No smaller eligible target, interval, role or successful normal allowance is
// selected here; a future complete factory must prove every conditional fact.
func providerHistoryAppliedCommitmentKindsUpper(l providerHistorySemanticCommitmentLimits, historyBytes uint64, kinds providerHistoryAppliedStateKinds) (providerHistoryAppliedCommitmentCost, bool) {
	var empty providerHistoryAppliedCommitmentCost
	if kinds.WorkingEmpty && !l.Window {
		return empty, false
	}
	out, ok := providerHistoryAppliedCommitmentUpper(l, historyBytes)
	if !ok {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	replace := func(claims, created uint64) bool {
		full, fullOK := providerHistoryAppliedStreamRootUpper(claims, created)
		zero, zeroOK := providerHistoryAppliedEmptyStreamRootUpper(claims, created)
		if !fullOK || !zeroOK {
			return false
		}
		for coordinate := range out.normal {
			out.normal[coordinate] = a.sum(a.difference(out.normal[coordinate], full.normal[coordinate]), zero.normal[coordinate])
			out.envelope[coordinate] = a.sum(a.difference(out.envelope[coordinate], full.envelope[coordinate]), zero.envelope[coordinate])
		}
		return !a.failed
	}
	if kinds.ConfirmedEmpty && !replace(l.ConfirmedClaims, l.ConfirmedCreated) || kinds.WorkingEmpty && !replace(l.WorkingClaims, l.WorkingCreated) {
		return empty, false
	}
	return out, true
}

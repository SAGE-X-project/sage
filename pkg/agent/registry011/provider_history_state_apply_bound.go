package registry011

import "reflect"

// One original applyObserved call after the actual selected Registry/command
// parser route. Universal schema ceilings cover every intermediate array,
// not merely initial or successful output counts. Counts and this scalar
// quote confer no caller, native curve/URI, RAM or production-factory authority.
func providerHistoryStateApplyUpper() (candidateWorkVector, bool) {
	var empty candidateWorkVector
	clone, cloneOK := providerHistorySemanticCloneUpper(providerHistorySemanticCloneLimits{Keys: 128, Services: 16, Grants: 128})
	record, recordOK := providerHistoryStateSerializationUpper(providerHistoryStateSerializationLimits{Keys: 128, Services: 16, Record: true})
	grants, grantsOK := providerHistoryStateSerializationUpper(providerHistoryStateSerializationLimits{Grants: 128})
	entry, entryOK := providerHistoryParsedStateEntryUpper(128)
	proof, proofOK := providerHistoryNewKeyProofUpper(128, 128)
	if !cloneOK || !recordOK || !grantsOK || !entryOK || !proofOK {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	text := uint64(reflect.TypeFor[string]().Size())
	key := uint64(reflect.TypeFor[replayKey]().Size())
	service := uint64(reflect.TypeFor[replayService]().Size())
	grant := uint64(reflect.TypeFor[replayGrant]().Size())
	recordSize := uint64(reflect.TypeFor[replayRecord]().Size())
	header := uint64(reflect.TypeFor[[]replayGrant]().Size())
	const k, t, g uint64 = 128, 16, 128
	keyClone := a.sum(key, a.product(6, text), 808)
	keyArray := a.sum(1, a.product(k, a.sum(a.product(3, key), a.product(6, text), 809)))
	serviceArray := a.sum(1, a.product(t, a.sum(a.product(2, service), 4295)))
	create := a.sum(a.product(57, k), keyArray, serviceArray, 1, 1, 133, 86, recordSize, header, 1)
	authorization := a.sum(28, 208, a.product(120, g))
	activate := a.sum(24, a.product(57, k), text, 1)
	deactivate := a.sum(text, 1)
	insert := a.sum(23, a.product(217, k), a.product(65, t), keyClone,
		a.product(k, key), 1, a.product(k, a.sum(key, 1)), header, 1)
	revoke := a.sum(a.product(147, k), a.product(2, a.sum(text, 1)))
	// URI parsing is entered again by mutation, independently of the earlier
	// command parser. Keep its codec2048 bytes as well as logical work.
	services := a.sum(header, 25, a.product(t, a.sum(a.product(2, service), 4194483, a.product(65, k))))
	grantChange := a.sum(header, 2235, a.product(g, a.sum(a.product(2, grant), 236)))
	// Mutually exclusive branches are deliberately summed, not asserted to
	// execute together. This also bounds all entered mutation refusal prefixes.
	mutation := a.sum(25, create, authorization, activate, deactivate, insert, revoke, services, grantChange)
	split := a.sum(a.product(g, a.sum(a.product(2, grant), 29)), header, 4)
	post := a.sum(182, 2*401, 17, text, 1, 55, split, recordSize, a.product(2, text), 3,
		262145, uint64(reflect.TypeFor[HistoryContentStep]().Size()), 1, uint64(reflect.TypeFor[replayState]().Size()), 1)
	// The shared clone quote includes unused stream/empty-index baseline;
	// retaining that overhead is conservative for this actual state-only clone.
	work := clone
	// The second record is required terminal/version-exhaustion headroom,
	// even when it is not returned. Do not count only the first serializer.
	for _, component := range [...]candidateWorkVector{record, record, grants, entry, proof} {
		for coordinate, units := range component {
			work[coordinate] = a.sum(work[coordinate], units)
		}
	}
	work[2] = a.sum(work[2], 2048*t, 40) // service URI and grant address codecs
	work[10] = a.sum(work[10], mutation, post)
	if a.failed {
		return empty, false
	}
	return work, true
}

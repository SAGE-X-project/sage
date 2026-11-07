package registry011

import "reflect"

// Prospective cardinalities for one successful shared state/index clone.
// These counts are DATA, not an observed stream, source or admission witness.
type providerHistorySemanticCloneLimits struct {
	Claims, Created, Keys, Services, Grants uint64
}

// Match only the original explicit logical copy/text/compare events, including
// nil and nonnil-empty storage. Reflect sizes describe those source events;
// they do not qualify map buckets, allocation, backing RAM, GC or native CPU.
// Repeated clones, index roots, serialization and whole-program cost remain
// separate. No original clone, meter, lifetime bound or normalization changes.
func providerHistorySemanticCloneUpper(l providerHistorySemanticCloneLimits) (candidateWorkVector, bool) {
	var empty candidateWorkVector
	if l.Claims > 1<<20 || l.Created > 1<<20 || l.Keys > 128 || l.Services > 16 || l.Grants > 128 {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	text := uint64(reflect.TypeFor[string]().Size())
	key := uint64(reflect.TypeFor[replayKey]().Size())
	service := uint64(reflect.TypeFor[replayService]().Size())
	grant := uint64(reflect.TypeFor[replayGrant]().Size())
	claimSlot := a.sum(uint64(reflect.TypeFor[replayClaimID]().Size()), uint64(reflect.TypeFor[replayClaimState]().Size()))
	createdSlot := a.sum(text, uint64(reflect.TypeFor[bool]().Size()))
	stream := a.sum(uint64(reflect.TypeFor[replayStream]().Size()), 1)
	state := a.sum(uint64(reflect.TypeFor[replayState]().Size()), 1, 546, a.product(4, text))
	// One array event, then key clone/put plus six compare/copy/set fields.
	// Every prospective key includes the optional eight-byte expiry copy.
	keys := a.sum(1, a.product(l.Keys, a.sum(a.product(3, key), a.product(6, text), 809)))
	// Services and grants put whole values; they have no separate setString.
	services := a.sum(1, a.product(l.Services, a.sum(a.product(2, service), 4295)))
	grants := a.sum(1, a.product(l.Grants, a.sum(a.product(2, grant), 121)))
	// Both map buffers share one event. Keep consumed/false tombstones too.
	maps := a.sum(1, a.product(l.Claims, a.sum(a.product(2, claimSlot), 162)), a.product(l.Created, a.sum(a.product(2, createdSlot), 196)))
	units := a.sum(stream, state, keys, services, grants, maps)
	if a.failed {
		return empty, false
	}
	var work candidateWorkVector
	work[10] = units
	return work, true
}

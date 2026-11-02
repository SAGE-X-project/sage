package registry011

import "reflect"

// Successful collection expected-window DATA only. This is not the legacy
// nil-history path, an original profile or an executable reservation. ABI,
// decoded commands/claims, state/proofs and serializers have separate rows.
type providerHistoryEventQueueWork struct {
	changed, consume, events, queue, stop candidateWorkVector
	begin, block, finish, total           candidateWorkVector
	windowOpens, eventSlots, subjects     uint64
	maximumQueue, queueMoves, oldSlots    uint64
	queueBytes, stopBytes                 uint64
}

func providerHistoryEventQueueUpper(l providerHistoryCollectionSemanticLimits, logs, evidence, claims, created uint64) (providerHistoryEventQueueWork, bool) {
	var empty providerHistoryEventQueueWork
	calls, ok := providerHistoryCollectionSemanticCallsBound(l)
	if !ok || logs == 0 || logs > 65536 || evidence == 0 || evidence > 64<<20 ||
		claims == 0 || claims > 1<<20 || created == 0 || created > 1<<20 {
		return empty, false
	}
	clone, ok := providerHistorySemanticCloneUpper(providerHistorySemanticCloneLimits{
		Claims: claims, Created: created, Keys: 128, Services: 16, Grants: 128,
	})
	if !ok {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	id := uint64(reflect.TypeFor[replayClaimID]().Size())
	claim := uint64(reflect.TypeFor[replayClaimState]().Size())
	text := uint64(reflect.TypeFor[string]().Size())
	flag := uint64(reflect.TypeFor[bool]().Size())
	item := uint64(reflect.TypeFor[providerHistoryExpectedItem]().Size())
	array := uint64(reflect.TypeFor[[]providerHistoryExpectedItem]().Size())
	integer := uint64(reflect.TypeFor[int]().Size())
	var out providerHistoryEventQueueWork
	out.windowOpens, out.subjects = calls.windowOpens, calls.selectedCommands
	out.eventSlots = a.product(out.windowOpens, logs)
	out.maximumQueue = min(l.Versions, uint64(4096), logs)
	// This is one optional create consumption, including BOTH funded index
	// writes. It never deletes the consumed claim or another subject's create.
	out.consume[10] = a.sum(a.product(2, id), a.product(2, claim), text, flag, 359)
	// DID join/copy/hash, actor topic, two topic checks, event DID/operation,
	// command-hash equality and selected-agent comparison. Include the extra
	// one of command hash here; its actual raw length is pooled below.
	out.changed = candidateWorkVector{2: 40, 10: 2745}
	h := a.sum(a.product(6, evidence), a.product(3136, logs))
	for i := range out.events {
		out.events[i] = a.product(out.eventSlots, a.sum(out.changed[i], out.consume[i]))
	}
	out.events[10] = a.sum(out.events[10], a.product(out.windowOpens, h))

	// For an original queue of m<=u items, sum(2q+3)=m*(m+2).
	// Global selected-command count J bounds all original windows/journeys.
	out.queueMoves = a.product(out.subjects, a.sum(out.maximumQueue, 2))
	if out.maximumQueue > 0 {
		out.oldSlots = a.product(out.subjects, out.maximumQueue-1) / 2
	}
	// prepare+retain+history comparison and record/grant hashes need
	// 8R+4X+Y+Z. retainExpected charges R+X+Y+Z to the SAME replayBudget;
	// history comparison charges R+X again. Also apply independent caps
	// R/X<=262144,Y<=65536,Z<=262144. No producer length is authoritative.
	out.queueBytes = min(a.product(a.product(8, out.windowOpens), evidence), a.product(3473408, out.subjects))
	out.queue[10] = a.sum(a.product(item, out.queueMoves), out.oldSlots, out.queueBytes,
		a.product(out.subjects, a.sum(a.product(2, text), a.product(2, flag), array, integer, 197)))

	// Actual stop comparison occurs once per window, even if absent. The
	// existing state component funds both present serializer calls separately.
	// stopSerializations/2 is exactly the present-stop comparison count.
	out.stopBytes = a.product(calls.stopSerializations/2, min(uint64(655360), a.product(2, evidence)))
	out.stop[10] = a.sum(a.product(21, out.windowOpens), out.stopBytes)
	// These actual begin clones are DISTINCT from earlier cursor snapshots.
	// Constructor/configuration is already in cursor-initialization costs.
	out.begin = clone
	out.begin[10] = a.sum(out.begin[10], uint64(reflect.TypeFor[providerHistoryExpectedWindow]().Size()), uint64(reflect.TypeFor[replayRun]().Size()), 1)
	out.block[10] = a.sum(uint64(reflect.TypeFor[FinalizedHeader]().Size()), integer, 9)
	out.finish[10] = a.sum(uint64(reflect.TypeFor[replayStream]().Size()), integer,
		uint64(reflect.TypeFor[*providerHistoryExpectedWindow]().Size()), 2)
	for i := range out.total {
		out.total[i] = a.sum(out.events[i], out.queue[i], out.stop[i],
			a.product(out.windowOpens, out.begin[i]), a.product(calls.blockApplies, out.block[i]),
			a.product(calls.windowCloses, out.finish[i]))
	}
	if a.failed {
		return empty, false
	}
	return out, true
}

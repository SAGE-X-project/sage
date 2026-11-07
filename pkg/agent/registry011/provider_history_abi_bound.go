package registry011

import (
	"reflect"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// Original prospective per-window evidence/log limits must be bound by the
// eventual complete factory. This component covers only successful observed
// ABI work and commitObserved; native CPU/allocations and command JSON/tuple
// generation/consumer packing, outer RecordChanged, state/proofs and final/failure
// remain separate. Positive numeric DATA supplies no execution authority.
type providerHistoryABIWork struct {
	fixed, selector, framing, changed, mutation, commit candidateWorkVector
	event, canonicalWindow, window, total               candidateWorkVector
	initializations, eventSlots                         uint64
}

// Fixed ABI and one numeric window; no original role count is inferred here.
func providerHistoryABIWindowUpper(logs, evidence uint64) (providerHistoryABIWork, bool) {
	var empty providerHistoryABIWork
	if logs == 0 || logs > 65536 || evidence == 0 || evidence > 64<<20 {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	argument := uint64(reflect.TypeFor[abi.Argument]().Size())
	method := uint64(reflect.TypeFor[abi.Method]().Size())
	claim := uint64(reflect.TypeFor[replayClaimID]().Size())
	state := uint64(reflect.TypeFor[replayClaimState]().Size())
	var out providerHistoryABIWork
	contract, outcomes := uint64(len(replayContractABI)), uint64(len(replayOutcomesABI))
	out.fixed[2] = a.sum(contract, outcomes)
	out.fixed[10] = a.sum(a.product(contract, contract), contract, 2, a.product(outcomes, outcomes), outcomes, 2)
	out.initializations = 1
	out.eventSlots = logs

	// Eight selector candidates each copy one Method then compare4/4 bytes.
	out.selector[10] = a.product(8, a.sum(method, 10))
	// The pinned revealClaim graph dominates the other seven mutation
	// graphs: up to128 seven-field key tuples and16 three-field services.
	// Every offset/length word is114 units/8 decoded; padding<=63; ASCII
	// compares cost2 per byte. Argument storage/assignment adds2*n*A+n+1.
	out.framing[2] = 12208
	out.framing[10] = a.sum(a.product(2176, argument), 364678)
	// RecordChanged: eight nonindexed fields, two bounded dynamic strings,
	// three uint64s and three bytes32s. Canonical round-trips are below.
	out.changed[2] = 32
	out.changed[10] = a.sum(a.product(16, argument), 1038)
	for coordinate := range out.mutation {
		out.mutation[coordinate] = a.sum(out.selector[coordinate], out.framing[coordinate], out.changed[coordinate])
	}
	// COMPLETE commitObserved, including actor/topic/height checks and the
	// final claim store. Its own fixed32-byte encoder is not in the pool.
	out.commit[2] = 40
	out.commit[10] = a.sum(method, a.product(2, argument), a.product(2, claim), state, 3189)

	// One frame can produce only one accepted Registry event. Mutation and
	// commit are alternative branches; retain their coordinatewise maximum.
	for coordinate := range out.event {
		out.event[coordinate] = max(out.mutation[coordinate], out.commit[coordinate])
	}
	// The SAME per-window replayBudget charges each frame.Input and log.Data
	// before event replay. Unique frames therefore give sum(n)<=evidence for
	// BOTH mutation canonical calls, each n<=148800, at most2*logs calls.
	// Sum(2*n*n+2*n+3) <= (2*148800+2)*evidence +6*logs.
	// This pool is a reservation, never an observed codec output or a reset.
	out.canonicalWindow[2] = evidence
	out.canonicalWindow[10] = a.sum(a.product(297602, evidence), a.product(6, logs))
	for coordinate := range out.window {
		out.window[coordinate] = a.sum(a.product(logs, out.event[coordinate]), out.canonicalWindow[coordinate])
		out.total[coordinate] = a.sum(a.product(out.initializations, out.fixed[coordinate]),
			out.window[coordinate])
	}
	if a.failed {
		return empty, false
	}
	return out, true
}

// Preserve the original whole-program result and all original scalar guards.
func providerHistoryABIUpper(l providerHistoryCollectionSemanticLimits, logs, evidence uint64) (providerHistoryABIWork, bool) {
	var empty providerHistoryABIWork
	calls, ok := providerHistoryCollectionSemanticCallsBound(l)
	if !ok {
		return empty, false
	}
	out, ok := providerHistoryABIWindowUpper(logs, evidence)
	if !ok {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	out.initializations = calls.observedABIPairs
	out.eventSlots = a.product(calls.windowOpens, logs)
	for c := range out.total {
		out.total[c] = a.sum(a.product(out.initializations, out.fixed[c]), a.product(calls.windowOpens, out.window[c]))
	}
	if a.failed {
		return empty, false
	}
	return out, true
}

// One first-refusing ABI callback prefix. Typed closed framing
// precedes Unpack: canonical mutation bytes <=148800 and changed bytes <=512.
// Command generation/consumer/claim and outer/state/queue remain separate.
// This numeric DATA creates no source or original role/observer authority.
func providerHistoryABIRefusalUpper() (providerHistoryControlPair, bool) {
	var empty providerHistoryControlPair
	var a providerHistoryReadArithmetic
	argument := uint64(reflect.TypeFor[abi.Argument]().Size())
	method := uint64(reflect.TypeFor[abi.Method]().Size())
	claim := uint64(reflect.TypeFor[replayClaimID]().Size())
	state := uint64(reflect.TypeFor[replayClaimState]().Size())
	mutation := candidateWorkVector{2: 12240, 10: a.sum(a.product(2192, argument), a.product(8, method), 365796)}
	commit := candidateWorkVector{2: 40, 10: a.sum(method, a.product(2, argument), a.product(2, claim), state, 3189)}
	var out providerHistoryControlPair
	for c := range out.normal {
		out.normal[c] = max(mutation[c], commit[c])
	}
	const input, changed uint64 = 148800, 512
	// One codec then Pack under an original n-byte preflight: observed
	// encoded m<=n, and comparing the two buffers costs <=2*n.
	out.normal[2] = a.sum(out.normal[2], input, changed)
	out.normal[10] = a.sum(out.normal[10], a.product(2, a.product(input, input)), a.product(2, a.product(changed, changed)), a.product(2, input), a.product(2, changed), 6)
	out.envelope = out.normal
	// One serial outstanding encode preflight, never a renewed full budget.
	out.envelope[10] = a.sum(out.normal[10], a.product(input, input), 1)
	if a.failed {
		return empty, false
	}
	return out, true
}

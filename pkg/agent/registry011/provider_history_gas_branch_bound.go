package registry011

import (
	"math/big"
	"reflect"
	"strconv"

	"github.com/ethereum/go-ethereum/core/types"
)

type providerHistoryGasBranchCost struct {
	strict, normalization, assembly candidateWorkVector
	actual, preflight, total        candidateWorkVector
	journeys, windows, native       uint64
	strictPool, originalPool        uint64
	strictSlots, originalSlots      uint64
}

// Successful original post-fork gas acquisition and BOTH strict gas checks.
// E/T/V are shared original persistent-window bounds, C selected capture bytes,
// native whole post-fork blocks. Profile selection is already bound by the
// original input; this DATA neither chooses nor falls back to another profile.
// Native CPU/backing/RAM, frames/effects, entered/deferred/failing prefixes and
// complete factory/final admission remain separate. No authority is created.
func providerHistoryGasBranchUpper(l providerHistoryCollectionSemanticLimits, evidence, transactions, bodyBytes, valueBytes, native uint64, profile string) (providerHistoryGasBranchCost, bool) {
	out, ok := providerHistoryGasBranchActual(l, evidence, transactions, bodyBytes, valueBytes, native, profile)
	if !ok {
		return providerHistoryGasBranchCost{}, false
	}
	var a providerHistoryReadArithmetic
	out.preflight[10] = a.product(out.windows, 4503633987174401)
	for coordinate := range out.total {
		out.total[coordinate] = a.sum(out.actual[coordinate], out.preflight[coordinate])
	}
	if a.failed {
		return providerHistoryGasBranchCost{}, false
	}
	return out, true
}

// Both strict passes, normalization and assembly as entered-work DATA only.
// Zero preflight/total are deliberate: original-scope admission is separate,
// including composite remaining/closure rows and native/RAM qualification.
func providerHistoryGasBranchActual(l providerHistoryCollectionSemanticLimits, evidence, transactions, bodyBytes, valueBytes, native uint64, profile string) (providerHistoryGasBranchCost, bool) {
	var empty providerHistoryGasBranchCost
	_, ok := providerHistoryCollectionSemanticCallsBound(l)
	if !ok || evidence < 65536 || evidence > 64<<20 || transactions == 0 || transactions > 16384 ||
		bodyBytes < 4 || bodyBytes > 8<<20 || valueBytes == 0 || valueBytes > min(evidence, bodyBytes) || native > l.Blocks {
		return empty, false
	}
	root := false
	switch profile {
	case SepoliaAmsterdamGasProfile, SepoliaAmsterdamNethermindGasProfile:
	case SepoliaAmsterdamRootRegularGasProfile, SepoliaAmsterdamRootExecutionGasProfile:
		root = true
	default:
		return empty, false
	}
	var a providerHistoryReadArithmetic
	var out providerHistoryGasBranchCost
	out.journeys = a.sum(a.product(14, l.Groups), 1)
	// Only windows containing a native block enter these paths. Every such
	// window contains at least one of the original at-most-native blocks.
	nw := min(l.Windows, native)
	out.windows, out.native = a.product(out.journeys, nw), a.product(out.journeys, native)
	c, v := bodyBytes, valueBytes
	h := min(evidence, c, uint64(65536))
	w := uint64(strconv.IntSize / 8)
	i := uint64(reflect.TypeFor[big.Int]().Size())
	gas := uint64(reflect.TypeFor[ReplayGasEvidence]().Size())
	// Native/debug transactions may retain a network sidecar. Every hash
	// then copies these two structs, not their referenced blob payloads.
	// Each original copy event also contributes one logical unit.
	sidecar := a.sum(uint64(reflect.TypeFor[types.Transaction]().Size()), uint64(reflect.TypeFor[types.BlobTx]().Size()), 2)
	nt := a.product(nw, transactions)
	// The later successful ORIGINAL persistent budget charges the final
	// strict trace. Root normalization's original result is charged only to
	// its temporary per-block budget and cannot borrow that window pool.
	out.strictPool = min(a.product(c, native), a.product(evidence, nw))
	out.originalPool = out.strictPool
	if root {
		out.originalPool = a.product(min(c, evidence), native)
	}
	out.strictSlots = min(a.product(native, transactions), out.strictPool)
	out.originalSlots = min(a.product(native, transactions), out.originalPool)
	canonical := max(a.sum(a.product(2, v), 260), h+1)
	perTransaction := a.sum(a.product(256, v), i, a.product(4, w), 87130, sidecar)
	window := a.sum(a.product(canonical, evidence), evidence/23, a.product(perTransaction, transactions))
	oneStrict := a.sum(a.product(a.sum(a.product(6, c), 10), out.strictPool), a.product(4497, native),
		a.product(a.product(3, w), out.strictSlots), a.product(nw, window))
	passes := a.product(2, out.journeys)
	out.strict[2] = a.product(passes, a.sum(a.product(6, out.strictPool), a.product(161, nt)))
	out.strict[10] = a.product(passes, oneStrict)
	if root {
		// Original top-only context has15 distinct allowed names/121 name
		// bytes. Its object membership constant including codec/scan is3857.
		// Sender/hash/data terms share the same original canonical E.
		perTx := a.sum(a.product(256, v), a.product(9, w), 103734, sidecar)
		window := a.sum(a.product(a.sum(a.product(2, v), 261), evidence), a.product(perTx, transactions))
		out.normalization[2] = a.product(out.journeys, a.sum(a.product(8, out.originalPool), a.product(257, nt)))
		out.normalization[10] = a.product(out.journeys, a.sum(a.product(a.sum(a.product(9, c), 8), out.originalPool),
			a.product(5, native), a.product(a.product(3, w), out.originalSlots), a.product(nw, window)))
	} else {
		// Original native profile classification inspects only the first
		// row's closed field shape after its complete raw-array parse.
		out.normalization[2] = a.product(out.journeys, a.product(3, out.originalPool))
		out.normalization[10] = a.product(out.journeys, a.sum(a.product(a.sum(a.product(3, c), 2), out.originalPool),
			a.product(663, native), a.product(a.product(3, w), out.originalSlots), a.product(a.sum(a.product(3, w), 1), nt)))
	}
	// Initial original-trace copy, exact selected-profile comparison and gas
	// descriptor copy. Immutable retain/replay trace copies are already in
	// the separate binary-wrapper component and are not added again.
	out.assembly[10] = a.product(out.journeys, a.sum(out.originalPool,
		a.product(a.sum(a.product(2, uint64(len(profile))), gas, 3), native)))
	for coordinate := range out.actual {
		out.actual[coordinate] = a.sum(out.strict[coordinate], out.normalization[coordinate], out.assembly[coordinate])
	}
	if a.failed {
		return empty, false
	}
	return out, true
}

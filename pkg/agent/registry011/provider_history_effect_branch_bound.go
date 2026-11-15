package registry011

import (
	"math/big"
	"reflect"
	"strconv"

	"github.com/ethereum/go-ethereum/core/types"
)

type providerHistoryEffectBranchLimits struct {
	Evidence, Transactions, Frames, Logs uint64
	BodyBytes, ValueBytes, Effects       uint64
	ContentBlocks                        uint64
}

type providerHistoryEffectBranchCost struct {
	normalization, execution, collection candidateWorkVector
	actual, preflight, total             candidateWorkVector
	journeys, windows, selected          uint64
	frames, logs, children, traceBytes   uint64
	blockingBatches                      uint64
}

// Complete successful selected trace normalization, transaction-bound frame
// replay and original content/effect transition control. Binary acquisition,
// immutable block retention, gas, cursor roots/copies and command/state/event
// processing remain disjoint. Native CPU/backing/RAM, entered/deferred/failing
// prefixes, final/page and factory admission are not certified by this DATA.
func providerHistoryEffectBranchUpper(l providerHistoryCollectionSemanticLimits, p providerHistoryEffectBranchLimits) (providerHistoryEffectBranchCost, bool) {
	out, ok := providerHistoryEffectBranchActual(l, p)
	if !ok {
		return providerHistoryEffectBranchCost{}, false
	}
	var a providerHistoryReadArithmetic
	out.preflight[10] = a.product(out.windows, 4503633987174401)
	for coordinate := range out.total {
		out.total[coordinate] = a.sum(out.actual[coordinate], out.preflight[coordinate])
	}
	if a.failed {
		return providerHistoryEffectBranchCost{}, false
	}
	return out, true
}

// Actual-only selected normalization, execution and transition DATA. The
// original observer/ledger never consumes this row directly. Zero preflight
// and total do not waive any original primitive, composite or failure bound.
func providerHistoryEffectBranchActual(l providerHistoryCollectionSemanticLimits, p providerHistoryEffectBranchLimits) (providerHistoryEffectBranchCost, bool) {
	var empty providerHistoryEffectBranchCost
	_, ok := providerHistoryCollectionSemanticCallsBound(l)
	if !ok || p.Evidence < 65536 || p.Evidence > 64<<20 || p.Transactions == 0 || p.Transactions > 16384 ||
		p.Frames == 0 || p.Frames > 65536 || p.Logs == 0 || p.Logs > 65536 ||
		p.BodyBytes < 4 || p.BodyBytes > 8<<20 || p.ValueBytes == 0 || p.ValueBytes > min(p.Evidence, p.BodyBytes) ||
		p.ContentBlocks == 0 || p.ContentBlocks > 10 {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	if p.Effects > a.product(l.Windows, p.Transactions) {
		return empty, false
	}
	var out providerHistoryEffectBranchCost
	out.journeys = a.sum(a.product(14, l.Groups), 1)
	// A selected receipt needs a transaction, a Registry log and at least
	// one replay frame. These successful original capacities only refine an
	// existing maximum; they neither remove a subject nor renew a window.
	q := min(p.Effects, a.product(l.Windows, min(p.Transactions, p.Frames, p.Logs)))
	n := min(l.Windows, q)
	af, logs := a.product(n, p.Frames), a.product(n, p.Logs)
	// Prospective Q is an upper, not the exact number of roots. Subtracting
	// Q from a frame maximum would underfund a trace when fewer roots enter.
	children := a.product(n, p.Frames-1)
	raw := a.product(min(p.BodyBytes, p.Evidence), q)
	out.windows, out.selected = a.product(out.journeys, n), a.product(out.journeys, q)
	out.frames, out.logs = a.product(out.journeys, af), a.product(out.journeys, logs)
	out.children, out.traceBytes = a.product(out.journeys, children), a.product(out.journeys, raw)
	w := uint64(strconv.IntSize / 8)
	integer := uint64(reflect.TypeFor[big.Int]().Size())
	logSize := uint64(reflect.TypeFor[types.Log]().Size())
	rowSize := uint64(reflect.TypeFor[replayLog]().Size())
	blockSize := uint64(reflect.TypeFor[providerHistoryCollectionBlock]().Size())
	effectSize := uint64(reflect.TypeFor[providerHistoryCollectionEffect]().Size())
	executionSize := uint64(reflect.TypeFor[ReplayExecution]().Size())
	stringSize := uint64(reflect.TypeFor[string]().Size())
	sidecar := a.sum(uint64(reflect.TypeFor[types.Transaction]().Size()), uint64(reflect.TypeFor[types.BlobTx]().Size()), 2)
	c, v, e := p.BodyBytes, p.ValueBytes, p.Evidence
	if n != 0 {
		// Original frame depth0..32 gives at most33 appearances of a byte
		// in recursive frame objects; nonrecursive log raw objects share one
		// pool. Ignored diagnostic/whitespace bytes do not share semantic E,
		// so the complete raw result pool remains min(C,E)*Q, not n*E.
		out.normalization[2] = a.product(out.journeys, a.sum(a.product(138, raw), a.product(176, af), a.product(296, logs)))
		out.normalization[5], out.normalization[6] = out.frames, out.logs
		out.normalization[10] = a.product(out.journeys, a.sum(
			a.product(a.sum(a.product(138, c), a.product(198, w), 170), raw), a.product(12906, af),
			a.product(a.sum(a.product(38, w), 2), children), a.product(a.sum(18765, a.product(42, w)), logs), a.product(4, q)))
		// Enclosing execution independently hashes a header per selected
		// transaction, recovers its sender, binds root/value/input/recipient,
		// walks all frames and joins every returned log to its receipt.
		h := min(e, c, uint64(65536))
		perExecution := a.sum(a.product(h, h), h, a.product(256, v), integer, w, 100976, a.product(16, rowSize))
		perLog := a.sum(logSize, 1801, a.product(7, rowSize))
		out.execution[2] = a.product(out.journeys, a.sum(a.product(161, q), a.product(64, af), a.product(40, logs)))
		out.execution[5], out.execution[6] = out.frames, out.logs
		out.execution[10] = a.product(out.journeys, a.sum(
			a.product(a.product(a.sum(a.product(2, v), 266), n), e), a.product(4533, af),
			a.product(perExecution, q), a.product(perLog, logs)))
	}
	// Pending arrays exist even when no Registry receipt selects a trace.
	// Effects reset after each content batch of at most K original blocks:
	// cumulative descriptor initialization is <=K*Q, never B*Q globally.
	// Each extra blocked scan belongs to one nonempty32-member trace group.
	qmax := min(p.Transactions, q)
	out.blockingBatches = a.sum(qmax, 31) / 32
	perSelected := a.sum(a.product(p.ContentBlocks, effectSize), a.product(3, stringSize), executionSize, sidecar, 4363,
		a.product(9, a.sum(p.ContentBlocks, out.blockingBatches)))
	out.collection[10] = a.product(out.journeys, a.sum(
		a.product(a.product(40, l.Windows), p.Logs), a.product(a.product(2, l.Windows), p.Transactions),
		a.product(a.sum(a.product(2, blockSize), 137), l.Blocks), raw,
		a.product(a.product(v+1, n), e), a.product(perSelected, q)))
	for coordinate := range out.actual {
		out.actual[coordinate] = a.sum(out.normalization[coordinate], out.execution[coordinate], out.collection[coordinate])
	}
	if a.failed {
		return empty, false
	}
	return out, true
}

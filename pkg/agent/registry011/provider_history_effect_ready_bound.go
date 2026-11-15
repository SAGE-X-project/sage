package registry011

import (
	"reflect"

	"github.com/ethereum/go-ethereum/core/types"
)

type providerHistoryEffectReadyLimits struct {
	Windows, Blocks, PendingBlocks uint64
	EvidenceBytes, BodyBytes       uint64
	TransactionBytes, ValueBytes   uint64
	Transactions, Logs, Effects    uint64
}

type providerHistoryEffectReadyCost struct {
	selection, arrays, hashes, pending, traceHeaders, readyScans candidateWorkVector
	normal, envelope                                             candidateWorkVector
	selectedRows, canonicalPool, effectArraySlots                uint64
	waitGroups, scanEntries                                      uint64
}

// Conditional DATA for the assigned effect-selection, pending allocation,
// string-header and ready-loop operations in ONE original monotone journey.
// All source counts and BEFORE-use canonical/native bounds must belong to the
// same original reader; scalar DATA creates no source or admission authority.
// Canonical decoding, gas, trace execution, retained raw bytes, spine lookup,
// cursor copies/commitments and all other original caller work are separate.
//
// Each pending batch has at most K blocks. Its selected effects can be copied
// at most K times, and its ready blocks scan that batch's complete effect list.
// Extra blocked scans are bounded by its ORIGINAL fixed32-result groups, not
// arbitrary callbacks or failed-S retries. Empty effects still retain pending
// allocation/release and empty trace-array invocation events.
//
// This does not replace full stage reservations or qualify native CPU,
// physical/backing/peak/lifetime RAM, resources, feasibility or the factory.
func providerHistoryEffectReadyUpper(p providerHistoryEffectReadyLimits) (providerHistoryEffectReadyCost, bool) {
	var empty providerHistoryEffectReadyCost
	if p.Blocks == 0 || p.Blocks > 1<<20 || p.Windows == 0 || p.Windows > p.Blocks ||
		p.PendingBlocks == 0 || p.PendingBlocks > 10 || p.EvidenceBytes == 0 || p.EvidenceBytes > 64<<20 ||
		p.BodyBytes < 4 || p.BodyBytes > 8<<20 || p.TransactionBytes == 0 || p.TransactionBytes > 64<<20 ||
		p.ValueBytes == 0 || p.ValueBytes > min(p.EvidenceBytes, p.BodyBytes, p.TransactionBytes) ||
		p.Transactions == 0 || p.Transactions > 16384 || p.Logs == 0 || p.Logs > 65536 || p.Effects > 1<<20 {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	var out providerHistoryEffectReadyCost
	blocks, k, effects := p.Blocks, p.PendingBlocks, p.Effects
	one := a.sum(p.Windows, k)
	txs := min(a.product(blocks, p.Transactions), a.product(one, p.Transactions))
	logs := min(a.product(blocks, p.Logs), a.product(one, p.Logs))
	out.selectedRows = min(effects, txs)
	out.canonicalPool = min(a.product(one, p.EvidenceBytes), a.product(blocks, min(p.BodyBytes, p.EvidenceBytes)), a.product(out.selectedRows, p.ValueBytes))
	out.effectArraySlots = a.product(k, effects)
	maximumBatchEffects := min(effects, a.product(k, p.Transactions))
	const group = uint64(providerHistoryGroupMembers) // The original nonzero constant32.
	out.waitGroups = a.sum(maximumBatchEffects, group-1) / group
	out.scanEntries = a.product(effects, a.sum(k, out.waitGroups))
	effect := uint64(reflect.TypeFor[providerHistoryCollectionEffect]().Size())
	text := uint64(reflect.TypeFor[string]().Size())
	block := uint64(reflect.TypeFor[providerHistoryCollectionBlock]().Size())
	sidecar := a.sum(uint64(reflect.TypeFor[types.Transaction]().Size()), uint64(reflect.TypeFor[types.BlobTx]().Size()), 2)
	out.selection[10] = a.sum(a.product(40, logs), a.product(2, txs))
	out.arrays[10] = a.sum(a.product(effect, out.effectArraySlots), blocks)
	out.hashes[10] = a.sum(a.product(a.sum(p.ValueBytes, 1), out.canonicalPool), a.product(a.sum(sidecar, 4359), out.selectedRows))
	out.pending[10] = a.product(2, a.product(a.sum(block, 1), blocks))
	out.traceHeaders[10] = a.sum(a.product(3, a.product(text, effects)), a.product(2, effects), blocks)
	out.readyScans[10] = a.product(9, out.scanEntries)
	for i := range out.normal {
		out.normal[i] = a.sum(out.selection[i], out.arrays[i], out.hashes[i], out.pending[i], out.traceHeaders[i], out.readyScans[i])
	}
	out.envelope = out.normal
	if out.selectedRows != 0 {
		out.envelope[10] = a.sum(out.normal[10], a.product(p.TransactionBytes, p.TransactionBytes), 1)
	}
	if a.failed {
		return empty, false
	}
	return out, true
}

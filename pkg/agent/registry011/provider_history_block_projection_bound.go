package registry011

import (
	"reflect"
	"strconv"
)

type providerHistoryBlockProjectionLimits struct {
	Windows, Blocks, PendingBlocks   uint64
	EvidenceBytes, BodyBytes         uint64
	TransactionBytes                 uint64
	Transactions, Logs, NativeBlocks uint64
}

type providerHistoryBlockProjectionCost struct {
	successfulCommon, successfulMaterialization, successfulBuilder  candidateWorkVector
	successfulBranch, successfulExtraTries                          candidateWorkVector
	failedCommon, failedMaterialization, failedBuilder              candidateWorkVector
	failedBranch, failedExtraTries, failedJSONAdditional            candidateWorkVector
	retention, decoder, normal, envelope                            candidateWorkVector
	body, header, bodyPool, headerPool, canonicalPool               uint64
	txCount, logCount, extremalBlock, extremalReceipt               uint64
	successfulWithdrawals, failedWithdrawals, failedTrieWithdrawals uint64
	incomingMaximum                                                 uint64
}

// Conditional numeric DATA for the entire original JSON/debug block/receipt
// projection and immutable retain/rebuild work in ONE monotone journey. The
// separately closed content decoder is included once for BOTH acquisition and
// semantic application. Gas, trace, effect selection, cursor/control, other
// original readers and physical native CPU/backing/RAM remain separate.
//
// Successful conversions share (W+K) canonical pools: cleared batches have
// passed their original semantic budgets, and ONE batch can remain pending.
// ONE complete additional failed-call allowance retains early native work,
// unclosed typed JSON, eager quantity calls, unknown fields and debug withdrawal
// encodings BEFORE the later trie-count guard. This overestimates actual calls;
// it does not authorize another call, a failed-S retry or a renewed quota.
//
// Every input must be bound to the original range, capture, replay and native
// selector BEFORE use. Scalars grant no source/admission authority. The result
// does not replace initializer plus SUM FULL admitted stage reservations,
// establish whole-program feasibility, or provide a production cost factory.
func providerHistoryBlockProjectionUpper(p providerHistoryBlockProjectionLimits, encoding string) (providerHistoryBlockProjectionCost, bool) {
	var empty providerHistoryBlockProjectionCost
	if p.Blocks == 0 || p.Blocks > 1<<20 || p.Windows == 0 || p.Windows > p.Blocks ||
		p.PendingBlocks == 0 || p.PendingBlocks > 10 || p.EvidenceBytes == 0 || p.EvidenceBytes > 64<<20 ||
		p.BodyBytes < 4 || p.BodyBytes > 8<<20 || p.TransactionBytes < p.BodyBytes+1024 || p.TransactionBytes > 64<<20 ||
		p.Transactions == 0 || p.Transactions > 16384 || p.Logs == 0 || p.Logs > 65536 || p.NativeBlocks > p.Blocks {
		return empty, false
	}
	json := false
	switch encoding {
	case EthereumJSONBlocks:
		json = true
	case "debug-raw-rlp-v1":
	default:
		return empty, false
	}
	var a providerHistoryReadArithmetic
	var out providerHistoryBlockProjectionCost
	windows, blocks, k := p.Windows, p.Blocks, p.PendingBlocks
	evidence, c, hNative, txs, logs := p.EvidenceBytes, p.BodyBytes, p.TransactionBytes, p.Transactions, p.Logs
	w := uint64(strconv.IntSize / 8)
	bodyCopy := uint64(reflect.TypeFor[providerHistoryNativeBlockBody]().Size())
	gasCopy := uint64(reflect.TypeFor[ReplayGasEvidence]().Size())
	one := a.sum(windows, k)
	out.body = min(c, evidence)
	out.header = min(out.body, uint64(65536))
	out.canonicalPool = a.product(one, evidence)
	out.txCount = min(a.product(blocks, txs), a.product(one, txs))
	out.logCount = min(a.product(blocks, logs), a.product(one, logs))
	out.bodyPool = a.product(blocks, out.body)
	out.headerPool = min(a.product(blocks, out.header), out.canonicalPool)
	out.extremalBlock = a.product(blocks, out.body)
	out.extremalReceipt = a.product(blocks, min(out.body, a.difference(evidence, out.body)))
	out.successfulWithdrawals, out.failedWithdrawals = 16384, min(uint64(65536), out.body/5)
	if json {
		out.bodyPool = min(out.bodyPool, a.sum(a.product(one, a.sum(evidence, a.product(5, txs))), a.product(800, blocks)))
		out.successfulWithdrawals, out.failedWithdrawals = 16, 16
	}
	out.failedTrieWithdrawals = min(out.failedWithdrawals, uint64(16384))
	trie := func(count uint64) uint64 {
		return a.sum(4, a.product(a.sum(4067, providerHistoryTrieNodeLogical), count))
	}
	common := func(count, pool, headerPool, withdrawals uint64) candidateWorkVector {
		perBlock := a.sum(a.product(2, c), 9128, a.product(withdrawals, a.sum(2402, a.product(3, w))))
		return candidateWorkVector{
			2: a.sum(a.product(4, pool), headerPool),
			10: a.sum(a.product(6, a.product(out.body, pool)), a.product(4, a.product(out.header, headerPool)),
				a.product(3, pool), a.product(3, headerPool), a.product(count, perBlock)),
		}
	}
	out.successfulCommon = common(blocks, out.bodyPool, out.headerPool, out.successfulWithdrawals)
	out.failedCommon = common(1, out.body, out.header, out.failedWithdrawals)
	out.successfulMaterialization[10] = a.sum(a.product(out.body, out.canonicalPool), a.product(a.sum(a.product(3, w), 1), out.txCount))
	out.failedMaterialization[10] = a.sum(a.product(out.body, out.body), a.product(a.sum(a.product(3, w), 1), txs))
	if json {
		out.successfulBuilder[10] = a.sum(a.product(out.body, out.bodyPool), a.product(blocks, a.sum(bodyCopy, 2)))
		out.failedBuilder[10] = a.sum(a.product(out.body, out.body), bodyCopy, 2)
		branch := func(count, rawBlock, rawReceipt, transactions, logCount, nativePool, header, value uint64) candidateWorkVector {
			canonicalCoefficient := max(a.sum(a.product(5, value), 261), a.sum(a.product(3, header), 3))
			transactionCoefficient := a.sum(a.product(header, header), header, a.product(256, value), a.product(14, w), 3283839)
			return candidateWorkVector{
				2: a.sum(a.product(24, rawBlock), a.product(12, rawReceipt), a.product(35136, count), a.product(2305, transactions), a.product(48, logCount)),
				3: count, 4: transactions, 6: logCount,
				10: a.sum(a.product(a.sum(a.product(25, c), a.product(18, w), 13875), rawBlock),
					a.product(a.sum(a.product(12, c), a.product(12, w), 19), rawReceipt),
					a.product(a.sum(53813875, a.product(128, w)), count), a.product(canonicalCoefficient, nativePool),
					a.product(transactionCoefficient, transactions), a.product(a.sum(14895, a.product(24, w)), logCount)),
			}
		}
		out.successfulBranch = branch(blocks, out.extremalBlock, out.extremalReceipt, out.txCount, out.logCount, out.canonicalPool, out.header, out.body)
		// Earlier rows passed their original per-row binary guard before the
		// next row. Only ONE current row can add the pre-canonical1024 spare.
		failedNativePool := a.sum(a.product(2, out.body), 1024, 65536)
		out.failedBranch = branch(1, out.body, min(out.body, a.difference(evidence, out.body)), txs, logs, failedNativePool, 65536, hNative)
		// The successful binder's s+1024 bound cannot price a mismatching
		// typed projection. Retain the complete generic original output cap.
		lastJSON := a.sum(a.product(2, out.body), 65536)
		out.failedJSONAdditional[2] = a.sum(a.product(9, out.body), a.product(2, lastJSON))
		out.failedJSONAdditional[10] = a.sum(a.product(out.body, out.body), a.product(3, a.product(lastJSON, lastJSON)),
			a.product(5, out.body), a.product(6, lastJSON), 8490, a.product(29, out.body), 327,
			a.product(8, a.sum(a.product(out.body, out.body), out.body, 3)))
		out.successfulExtraTries[10] = a.sum(a.product(a.sum(out.body, 34), out.canonicalPool),
			a.product(providerHistoryTrieNodeLogical, out.txCount), a.product(4, blocks), a.product(2, a.product(blocks, trie(16))))
		out.failedExtraTries[10] = a.sum(a.product(out.body, out.body), a.product(34, out.body), a.product(providerHistoryTrieNodeLogical, txs), 4, a.product(2, trie(16)))
		out.incomingMaximum = max(uint64(65536), lastJSON, a.sum(hNative, 256))
	} else {
		debug := func(count, transactions uint64) candidateWorkVector {
			slots := min(txs, (c-1)/2)
			perBlock := a.sum(a.difference(a.product(5, a.product(c, c)), a.product(6, c)), 20)
			return candidateWorkVector{2: a.product(a.difference(a.product(5, c), 4), count),
				10: a.sum(a.product(perBlock, count), a.product(a.sum(a.product(6, w), 1), a.product(count, slots)), a.product(a.sum(a.product(3, w), 1), transactions))}
		}
		out.successfulBranch, out.failedBranch = debug(blocks, out.txCount), debug(1, txs)
		out.successfulExtraTries[10] = a.product(blocks, trie(16384))
		out.failedExtraTries[10] = trie(out.failedTrieWithdrawals)
		out.incomingMaximum = max(uint64(65536), c)
	}
	// Canonical bytes and gas share E after clearing. One unfinished batch
	// may still own independent temporary canonical-E and gas-E per block.
	out.retention[10] = a.sum(a.product(2, a.product(a.sum(windows, a.product(2, k)), evidence)),
		a.product(a.sum(a.product(10, w), 4), out.txCount), a.product(6, blocks), a.product(a.sum(gasCopy, 3), p.NativeBlocks))
	decoder, ok := providerHistoryContentDecodePoolUpper(providerHistoryContentDecodePoolLimits{
		windows, blocks, k, evidence, out.body, txs, logs,
	})
	if !ok {
		return empty, false
	}
	out.decoder = decoder.normal
	for i := range out.normal {
		out.normal[i] = a.sum(out.successfulCommon[i], out.successfulMaterialization[i], out.successfulBuilder[i], out.successfulBranch[i], out.successfulExtraTries[i],
			out.failedCommon[i], out.failedMaterialization[i], out.failedBuilder[i], out.failedBranch[i], out.failedExtraTries[i], out.failedJSONAdditional[i], out.retention[i], out.decoder[i])
	}
	out.envelope = out.normal
	out.envelope[10] = a.sum(out.normal[10], a.product(out.incomingMaximum, out.incomingMaximum), 1)
	if a.failed {
		return empty, false
	}
	return out, true
}

package registry011

import (
	"reflect"
	"strconv"

	"github.com/ethereum/go-ethereum/core/types"
)

// Numeric DATA for ONE complete original nodeRootGas invocation. C is the
// complete original raw trace ceiling; H is the transaction ceiling selected
// by the original native owner, with signing ceiling M=H+256. T is the original
// temporary per-block MaxTransactions, never a count of successful rows or a
// later semantic-window allowance. These numbers authenticate no source/owner.
//
// Original witness row spans share C. In each row txHash/result are disjoint;
// the fixed witness/root schemas, FULL root context, four dimensions and
// continued native hash/text calls give decoded6u+161 and logical
// 6u²+29u+H²+M²+2H+S+37690+3w. S is the two original sidecar struct copies
// including their event constants; w is the Go word size. The raw codecs,
// array headers, result initialization and final encoding complete the row
// below. A failed parse, unconsumed preflight or later failure keeps all earlier
// entered work; an unconsumed preflight is not another recorded actual event.
//
// Raw preallocated slots may exceed actual rows, including a whitespace-padded
// empty array. Retain min(T,(C-1)/2) slots and separately overbound entered rows
// by the same maximum. An empty root object of two/three bytes performs only
// the five-unit invalid scalar prefix, covered by the retained root constant.
//
// This whole normalization E replaces its overlapping assigned scope. BOTH
// strict gas checks, block acquisition/retention, semantic execution, complete
// role/failure counts, native CPU/backing/peak/lifetime RAM, resources, factory
// and operating authority remain separate. Arithmetic success is not feasibility
// or permission to multiply incoming envelopes as recorded actual work.
func providerHistoryRootGasNormalizationRefusalUpper(bodyBytes, transactionBytes, transactions uint64) (candidateWorkVector, bool) {
	var empty candidateWorkVector
	if bodyBytes < 4 || bodyBytes > 8<<20 || transactionBytes == 0 || transactionBytes > 64<<20 ||
		transactions == 0 || transactions > 16384 {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	c, h, m := bodyBytes, transactionBytes, transactionBytes+256
	w := uint64(strconv.IntSize / 8)
	slots := min(transactions, (c-1)/2)
	sidecar := a.sum(uint64(reflect.TypeFor[types.Transaction]().Size()), uint64(reflect.TypeFor[types.BlobTx]().Size()), 2)
	native := a.sum(a.product(h, h), a.product(m, m), a.product(2, h), sidecar, 37690, a.product(3, w))
	var out candidateWorkVector
	out[2] = a.sum(a.product(8, c), a.product(161, slots))
	out[10] = a.sum(a.product(9, a.product(c, c)), a.product(29, c),
		a.product(a.sum(a.product(9, w), 1), slots), a.product(slots, native), 5)
	if a.failed {
		return empty, false
	}
	return out, true
}

type providerHistoryRootGasNormalizationCost struct {
	normal, envelope candidateWorkVector
}

// Alternative matched N/E DATA for the SAME complete original nodeRootGas
// invocation. Q bounds the SUM of standalone canonical transaction bytes at
// distinct original block indexes; V bounds each such encoding. The original
// complete block/capture and temporary replay evidence budgets must establish
// Q/V before use. Numeric arguments neither establish this premise nor bind an
// owner. The collection factory and complete phase/role composition are separate.
//
// Canonical block transactions share Q even when the root trace has C bytes.
// Sum b_i²+b_i+2 <= VQ+Q+2s. Signing lists have r_i<=b_i+256,
// giving sum(r_i²+2r_i+5)<=(V+258)(Q+256s)+5s. Native input
// comparison bytes share Q as well. The old loose row included signing4 and
// transaction-hash2: remove those six units before replacing BOTH native rows.
// All remaining raw, recovery/address, destination, quantity and output rows
// retain their original envelope, so this N also overbounds every actual prefix.
//
// E retains one widest UNCONSUMED original native encoding preflight M²+1
// above N. Neither native preflight is reduced; repeated successful preflights
// are not recorded actual work. Both fields have the same whole-normalizer
// scope and replace its overlapping assigned scope, never add to the loose E.
// Native CPU/accessor/backing/peak/lifetime RAM, strict gas checks, all-eleven
// feasibility, operating authority and the production factory remain separate.
func providerHistoryRootGasNormalizationPoolUpper(bodyBytes, canonicalPool, valueBytes, transactionBytes, transactions uint64) (providerHistoryRootGasNormalizationCost, bool) {
	var empty providerHistoryRootGasNormalizationCost
	if bodyBytes < 4 || bodyBytes > 8<<20 || canonicalPool == 0 || canonicalPool > 64<<20 ||
		transactionBytes == 0 || transactionBytes > 64<<20 || valueBytes == 0 || valueBytes > canonicalPool || valueBytes > transactionBytes ||
		transactions == 0 || transactions > 16384 {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	c, q, v, m := bodyBytes, canonicalPool, valueBytes, transactionBytes+256
	w := uint64(strconv.IntSize / 8)
	slots := min(transactions, (c-1)/2)
	sidecar := a.sum(uint64(reflect.TypeFor[types.Transaction]().Size()), uint64(reflect.TypeFor[types.BlobTx]().Size()), 2)
	fixed := a.sum(sidecar, 37691, a.product(3, w))
	signing := a.product(a.sum(v, 258), a.sum(q, a.product(256, slots)))
	var out providerHistoryRootGasNormalizationCost
	out.normal[2] = a.sum(a.product(8, c), a.product(161, slots))
	out.normal[10] = a.sum(a.product(9, a.product(c, c)), a.product(29, c),
		a.product(a.sum(a.product(9, w), 1), slots), 5, a.product(slots, fixed),
		a.product(v, q), a.product(2, q), signing)
	out.envelope = out.normal
	out.envelope[10] = a.sum(out.normal[10], a.product(m, m), 1)
	if a.failed {
		return empty, false
	}
	return out, true
}

package registry011

import (
	"math/big"
	"reflect"
	"strconv"

	"github.com/ethereum/go-ethereum/core/types"
)

type providerHistoryGasCheckCost struct {
	normal, envelope candidateWorkVector
}

// Matched N/E DATA for ONE complete original replayCheckGas invocation.
// Original canonical header bytes are at most64KiB; gas.BlockHash must be
// the original selected canonical66-byte text. Original complete canonical
// transaction bytes at distinct block indexes share Q, with each size<=V.
// The original source/capture/temporary replay budget must establish these
// premises. Numeric arguments create no source identity, owner or authority.
//
// C bounds the complete original trace, H is the original native owner ceiling
// and T is its positive temporary MaxTransactions, never actual successful rows
// or later semantic-window allowance. The raw parsers retain their whole-prefix
// envelopes. Native hash/signing actuals share Q; value-copy events keep the
// original selected rounded-H plus big.Int structure ceiling A. Access-list
// entry counts share canonical bytes Q. Fixed21078 retains the original sender
// 13788 and replaces only the old hash2/signing4 with pooled hash2/signing5.
//
// E adds one unconsumed original (H+256)²+1 encoding preflight above N.
// Header incoming room is already covered by its original whole-header row;
// original value-copy incoming room stays in its retained value row. No guard
// is reduced and no unconsumed preflight is recorded as actual. BOTH strict gas
// passes, surrounding normalization/acquisition/semantics and role counts need
// their own matched composition. CPU/accessors/backing/peak/lifetime RAM,
// complete all-eleven feasibility, production factory and authority are separate.
func providerHistoryGasCheckPoolUpper(bodyBytes, canonicalPool, valueBytes, transactionBytes, transactions uint64) (providerHistoryGasCheckCost, bool) {
	var empty providerHistoryGasCheckCost
	if bodyBytes < 4 || bodyBytes > 8<<20 || canonicalPool == 0 || canonicalPool > 64<<20 ||
		transactionBytes == 0 || transactionBytes > 64<<20 || valueBytes == 0 || valueBytes > canonicalPool || valueBytes > transactionBytes ||
		transactions == 0 || transactions > 16384 {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	c, q, v, h := bodyBytes, canonicalPool, valueBytes, transactionBytes
	m, w := transactionBytes+256, uint64(strconv.IntSize/8)
	slots := min(transactions, (c-1)/2)
	sidecar := a.sum(uint64(reflect.TypeFor[types.Transaction]().Size()), uint64(reflect.TypeFor[types.BlobTx]().Size()), 2)
	value := a.sum(a.product(a.sum(h, w-1)/w, w), uint64(reflect.TypeFor[big.Int]().Size()))
	fixed := a.sum(sidecar, value, 21078)
	signing := a.product(a.sum(v, 258), a.sum(q, a.product(256, slots)))
	const header = uint64(65536)
	var out providerHistoryGasCheckCost
	out.normal[2] = a.sum(a.product(6, c), a.product(97, slots))
	out.normal[10] = a.sum(a.product(header, header), header, 4497,
		a.product(6, a.product(c, c)), a.product(16, c),
		a.product(a.sum(a.product(6, w), 1), slots), a.product(slots, fixed),
		a.product(v, q), a.product(2, q), signing)
	out.envelope = out.normal
	out.envelope[10] = a.sum(out.normal[10], a.product(m, m), 1)
	if a.failed {
		return empty, false
	}
	return out, true
}

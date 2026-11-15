package registry011

import (
	"math/big"
	"reflect"
	"strconv"

	"github.com/ethereum/go-ethereum/core/types"
)

type providerHistoryExecutionReplayLimits struct {
	BodyBytes, ValueBytes, TransactionBytes uint64
	Frames, Logs, ReceiptBytes              uint64
}

type providerHistoryExecutionReplayCost struct {
	normal, envelope candidateWorkVector
}

// Numeric N/E DATA for ONE complete original replayExecutionObserved, including
// its four eager hash/text calls, sender/root/recipient, every entered frame,
// log construction/output growth and final receipt-log join. The caller must
// establish the same complete original normalized tree, canonical66 hash fields,
// original header<=64KiB, canonical transaction<=V and native owner ceiling H.
// Frame/log limits are original whole-window caps, never renewed per root.
// ReceiptBytes bounds original canonical receipt data through the temporary
// replay evidence charge; no later successful-window allowance is borrowed.
//
// Successful original normalization bounds addresses42, types12, values66,
// topics4 and the separate input/log-data byte pools by original raw C. Original
// value-copy A=roundUp(H,w)+sizeof(big.Int) is retained. Complete output growth
// copies at most16+7L logical replayLog slots with at most3L event constants;
// Receipt joins also fund all four 65-unit topic comparisons per log; a
// first receipt mismatch still consumes its original comparison bytes.
// Source-qualified shapes/pools are premises, not credentials this helper mints.
//
// N includes all original nonnative/header/value/CREATE incoming rows. E adds
// one unconsumed native signing encoding preflight (H+256)²+1; all original
// guards and native preflights remain unchanged. This whole execution scope
// replaces its overlapping assignment. Normalization, enclosing role/failure
// counts, all-eleven feasibility, native CPU/accessors/backing/peak/lifetime RAM,
// production factory and operating authority remain separate.
func providerHistoryExecutionReplayUpper(p providerHistoryExecutionReplayLimits) (providerHistoryExecutionReplayCost, bool) {
	var empty providerHistoryExecutionReplayCost
	if p.BodyBytes < 4 || p.BodyBytes > 8<<20 || p.TransactionBytes == 0 || p.TransactionBytes > 64<<20 ||
		p.ValueBytes == 0 || p.ValueBytes > p.TransactionBytes || p.Frames == 0 || p.Frames > 65536 ||
		p.Logs == 0 || p.Logs > 65536 || p.ReceiptBytes == 0 || p.ReceiptBytes > 64<<20 {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	v, h, w := p.ValueBytes, p.TransactionBytes, uint64(strconv.IntSize/8)
	m := h + 256
	value := a.sum(a.product(a.sum(h, w-1)/w, w), uint64(reflect.TypeFor[big.Int]().Size()))
	sidecar := a.sum(uint64(reflect.TypeFor[types.Transaction]().Size()), uint64(reflect.TypeFor[types.BlobTx]().Size()), 2)
	row := uint64(reflect.TypeFor[replayLog]().Size())
	log := uint64(reflect.TypeFor[types.Log]().Size())
	signing := a.product(a.sum(v, 258), a.sum(v, 256))
	const header = uint64(65536)
	var out providerHistoryExecutionReplayCost
	out.normal[2] = a.sum(161, a.product(64, p.Frames), a.product(40, p.Logs))
	out.normal[5], out.normal[6] = p.Frames, p.Logs
	out.normal[10] = a.sum(a.product(header, header), header, a.product(v, v), a.product(2, v),
		sidecar, signing, 34899, a.product(2, value), a.product(3, p.BodyBytes), p.ReceiptBytes,
		a.product(a.sum(16, a.product(7, p.Logs)), row), a.product(4433, p.Frames),
		a.product(a.sum(log, 2315), p.Logs))
	out.envelope = out.normal
	out.envelope[10] = a.sum(out.normal[10], a.product(m, m), 1)
	if a.failed {
		return empty, false
	}
	return out, true
}

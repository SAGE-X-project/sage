package registry011

import (
	"context"
	"reflect"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

// The exact external RLP body of pinned go-ethereum v1.17.5. This transient
// object borrows only this decoder's own header/body until synchronous encoding
// returns. It is never retained as a cursor or shared across original readers.
// Avoiding NewBlockWithHeader/WithBody removes their implicit header/body copies;
// it does not qualify RLP allocator internals or the whole native RAM envelope.
type providerHistoryNativeBlockBody struct {
	Header      *types.Header
	Txs         []*types.Transaction
	Uncles      []*types.Header
	Withdrawals []*types.Withdrawal `rlp:"optional"`
}

func providerHistoryNativeBlockBytesObserved(header *types.Header, txs []*types.Transaction, withdrawals types.Withdrawals, maximum int, meter *providerWorkMeter) ([]byte, error) {
	if header == nil || meter == nil || maximum < 1 || maximum > 64<<20 || len(txs) > 16384 || len(withdrawals) > 16 ||
		!meter.event(providerCopyEvent, uint64(reflect.TypeFor[providerHistoryNativeBlockBody]().Size())) {
		return nil, ErrReplayUnavailable
	}
	body := providerHistoryNativeBlockBody{Header: header, Txs: txs, Withdrawals: withdrawals}
	if !meter.preflight(providerEncodeEvent, uint64(maximum)) {
		return nil, ErrReplayUnavailable
	}
	raw, err := rlp.EncodeToBytes(&body)
	if err != nil || len(raw) > maximum {
		meter.refuse()
		return nil, ErrReplayUnavailable
	}
	if !meter.event(providerEncodeEvent, uint64(len(raw))) {
		return nil, ErrReplayUnavailable
	}
	return raw, nil
}

// Decode the first canonical RLP element instead of calling Block.Header,
// whose pinned implementation allocates a deep native CopyHeader. The complete
// body is independently decoded/canonicalized by the caller. This header must
// still match that caller's selected spine; it supplies DATA, never authority.
func providerHistoryNativeHeaderObserved(ctx context.Context, raw []byte, budget *replayBudget) (*types.Header, error) {
	if ctx == nil || ctx.Err() != nil || budget == nil || budget.observer == nil || len(raw) == 0 || len(raw) > 64<<20 || len(raw) > budget.limits.MaxEvidenceBytes || !budget.decodeObserved(raw) {
		return nil, ErrReplayUnavailable
	}
	body, rest, err := rlp.SplitList(raw)
	if err != nil || len(rest) != 0 || !budget.decodeObserved(body) {
		return nil, ErrReplay
	}
	kind, _, tail, err := rlp.Split(body)
	if err != nil || kind != rlp.List {
		return nil, ErrReplay
	}
	headerRaw := body[:len(body)-len(tail)]
	if len(headerRaw) == 0 || len(headerRaw) > 65536 || !budget.decodeObserved(headerRaw) {
		return nil, ErrReplayUnavailable
	}
	var header types.Header
	if rlp.DecodeBytes(headerRaw, &header) != nil || header.Number == nil || !header.Number.IsUint64() || header.Time > 1<<53-1 ||
		!budget.encodePreflight(len(headerRaw)) {
		return nil, ErrReplay
	}
	canonical, err := rlp.EncodeToBytes(&header)
	if err != nil || !budget.encodedObserved(canonical, len(headerRaw)) || !budget.equalObserved(canonical, headerRaw) || ctx.Err() != nil {
		return nil, ErrReplay
	}
	return &header, nil
}

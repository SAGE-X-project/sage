package registry011

import (
	"context"
	"strconv"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

func nodeJSONScanObserved(raw []byte, meter *providerWorkMeter) bool {
	return meter == nil || meter.event(providerCompareEvent, uint64(len(raw)))
}

func nodeJSONSliceObserved(count, words int, meter *providerWorkMeter) bool {
	return count >= 0 && count <= 65536 && words >= 1 && words <= 3 && (meter == nil || meter.event(providerCopyEvent, uint64(count*words*(strconv.IntSize/8))))
}

func nodeJSONSelectedObserved(header *types.Header, selected FinalizedHeader, meter *providerWorkMeter) bool {
	if header == nil || header.Number == nil || !header.Number.IsUint64() {
		return false
	}
	hash, ok := nodeHeaderHashObserved(header, meter)
	if !ok {
		return false
	}
	h, hashOK := nodeHashTextObserved(hash, meter)
	p, parentOK := nodeHashTextObserved(header.ParentHash, meter)
	return hashOK && parentOK && providerReplayTextEqual(h, selected.Hash, meter) && providerReplayTextEqual(p, selected.ParentHash, meter) && header.Number.Uint64() == selected.Number && header.Time == selected.Timestamp
}

// Only the closed JSON reader supplies these typed values. Its original JSON
// contains every consensus field, so the binary encoding is no larger than
// that original. Check the derived ceiling before and after encoding.
func nodeJSONRLPObserved(value any, maximum int, meter *providerWorkMeter) ([]byte, error) {
	if maximum < 1 || maximum > 64<<20 || meter != nil && !meter.preflight(providerEncodeEvent, uint64(maximum)) {
		return nil, ErrReplayUnavailable
	}
	var raw []byte
	var err error
	switch v := value.(type) {
	case *types.Transaction:
		if v == nil {
			return nil, ErrReplayUnavailable
		}
		raw, err = v.MarshalBinary()
	case *types.Receipt:
		if v == nil {
			return nil, ErrReplayUnavailable
		}
		raw, err = v.MarshalBinary()
	case *types.Block:
		if v == nil {
			return nil, ErrReplayUnavailable
		}
		raw, err = rlp.EncodeToBytes(v)
	case *types.Withdrawal:
		if v == nil {
			return nil, ErrReplayUnavailable
		}
		raw, err = rlp.EncodeToBytes(value)
	default:
		return nil, ErrReplayUnavailable
	}
	if err != nil || len(raw) > maximum {
		if meter != nil {
			meter.refuse()
		}
		return nil, ErrReplayUnavailable
	}
	if meter != nil && !meter.event(providerEncodeEvent, uint64(len(raw))) {
		return nil, ErrReplayUnavailable
	}
	return raw, nil
}

func nodeJSONRootsObserved(ctx context.Context, txs, withdrawals [][]byte, header *types.Header, meter *providerWorkMeter) bool {
	if header == nil || header.WithdrawalsHash == nil {
		return false
	}
	w, err := providerHistoryContentRootObserved(ctx, withdrawals, meter)
	if err != nil || !providerReplayEqual(w[:], header.WithdrawalsHash[:], meter) {
		return false
	}
	t, err := providerHistoryContentRootObserved(ctx, txs, meter)
	return err == nil && providerReplayEqual(t[:], header.TxHash[:], meter)
}

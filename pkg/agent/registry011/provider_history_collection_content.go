package registry011

import (
	"context"
	"reflect"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

// Only immutable consensus bytes cross S boundaries. Native decoded objects
// are temporary and must never become shared mutable state between readers.
// This representation does not qualify native allocation or resident RAM.
type providerHistoryCollectionBlock struct {
	header                 FinalizedHeader
	transactionCount       int
	headerRLP              string
	transactions, receipts []string
	gasProfile, gasTrace   string
}

func providerHistoryCollectionDecodeBlockObserved(ctx context.Context, blockResult, receiptResult string, want FinalizedHeader, in providerHistoryCollectionInput, meter *providerWorkMeter) (ReplayBlock, replayDecodedBlock, error) {
	var empty ReplayBlock
	var zero replayDecodedBlock
	if ctx == nil || ctx.Err() != nil || meter == nil || len(blockResult) == 0 || len(receiptResult) == 0 ||
		len(blockResult) > in.capture.BodyBytes || len(receiptResult) > in.capture.BodyBytes || !replayLimits(in.journal.Replay) {
		return empty, zero, ErrReplayUnavailable
	}
	// This temporary decoder has the selected shape/byte limits, not a fresh
	// provider allowance. All duplicate interpretation uses the same original
	// observer; semantic application additionally retains its whole-window
	// replay budget. Sufficient cost/native/RAM qualification is still required.
	budget := replayBudget{limits: in.journal.Replay, observer: meter}
	blockRaw, receiptsRaw := providerReplayBytes(blockResult, meter), providerReplayBytes(receiptResult, meter)
	if len(blockRaw) != len(blockResult) || len(receiptsRaw) != len(receiptResult) {
		return empty, zero, ErrReplayUnavailable
	}
	var binary []byte
	var err error
	switch in.encoding {
	case EthereumJSONBlocks:
		binary, err = nodeJSONBlock(ctx, blockRaw, want, &budget)
	case "debug-raw-rlp-v1":
		binary, err = rpcDataObserved(blockRaw, meter)
	default:
		return empty, zero, ErrReplayUnavailable
	}
	if err != nil || len(binary) == 0 || len(binary) > budget.limits.MaxEvidenceBytes ||
		!budget.decodeObserved(binary) || !nodeBlockCount(binary, budget.limits.MaxTransactions) {
		return empty, zero, ErrReplayUnavailable
	}
	header, headerErr := providerHistoryNativeHeaderObserved(ctx, binary, &budget)
	if headerErr != nil || !nodeJSONSelectedObserved(header, want, meter) {
		return empty, zero, ErrReplay
	}
	var block types.Block
	if !budget.decodeObserved(binary) || rlp.DecodeBytes(binary, &block) != nil {
		return empty, zero, ErrReplay
	}
	canonical, err := nodeJSONRLPObserved(&block, len(binary), meter)
	if err != nil || !budget.equalObserved(binary, canonical) ||
		len(block.Transactions()) > budget.limits.MaxTransactions || !budget.encodePreflight(len(binary)) {
		return empty, zero, ErrReplay
	}
	uncles, err := rlp.EncodeToBytes(block.Uncles())
	if err != nil || !budget.encodedObserved(uncles, len(binary)) || !meter.event(providerHashEvent, uint64(len(uncles))) {
		return empty, zero, ErrReplayUnavailable
	}
	uncleHash := crypto.Keccak256Hash(uncles)
	if !budget.equalObserved(uncleHash[:], header.UncleHash[:]) {
		return empty, zero, ErrReplay
	}
	withdrawals := block.Withdrawals()
	if header.WithdrawalsHash == nil {
		if withdrawals != nil {
			return empty, zero, ErrReplay
		}
	} else {
		if !nodeJSONSliceObserved(len(withdrawals), 3, meter) {
			return empty, zero, ErrReplayUnavailable
		}
		rows := make([][]byte, len(withdrawals))
		for i, withdrawal := range withdrawals {
			if ctx.Err() != nil {
				return empty, zero, ErrReplayUnavailable
			}
			rows[i], err = nodeJSONRLPObserved(withdrawal, len(binary), meter)
			if err != nil {
				return empty, zero, err
			}
		}
		root, rootErr := providerHistoryContentRootObserved(ctx, rows, meter)
		if rootErr != nil || !budget.equalObserved(root[:], header.WithdrawalsHash[:]) {
			return empty, zero, ErrReplay
		}
	}
	if !budget.encodePreflight(65536) {
		return empty, zero, ErrReplayUnavailable
	}
	out := ReplayBlock{}
	out.HeaderRLP, err = rlp.EncodeToBytes(header)
	if err != nil || !budget.encodedObserved(out.HeaderRLP, 65536) || !nodeJSONSliceObserved(len(block.Transactions()), 3, meter) {
		return empty, zero, ErrReplayUnavailable
	}
	out.Transactions = make([][]byte, len(block.Transactions()))
	for i, tx := range block.Transactions() {
		if ctx.Err() != nil {
			return empty, zero, ErrReplayUnavailable
		}
		out.Transactions[i], err = nodeJSONRLPObserved(tx, len(binary), meter)
		if err != nil {
			return empty, zero, err
		}
	}
	if in.encoding == EthereumJSONBlocks {
		out.Receipts, err = nodeJSONReceiptsHeaderObserved(ctx, receiptsRaw, &block, header, &budget)
	} else {
		rows, ok := nodeRawArrayObserved(receiptsRaw, budget.limits.MaxTransactions, meter)
		if !ok || len(rows) != len(out.Transactions) || !nodeJSONSliceObserved(len(rows), 3, meter) {
			return empty, zero, ErrReplay
		}
		out.Receipts = make([][]byte, len(rows))
		for i, row := range rows {
			if ctx.Err() != nil {
				return empty, zero, ErrReplayUnavailable
			}
			out.Receipts[i], err = rpcDataObserved(row, meter)
			if err != nil {
				return empty, zero, err
			}
		}
	}
	if err != nil || ctx.Err() != nil {
		return empty, zero, ErrReplayUnavailable
	}
	decoded, err := replayDecodeContent(ctx, out, &budget)
	return out, decoded, err
}

func providerHistoryCollectionRetainBlockObserved(raw ReplayBlock, header FinalizedHeader, meter *providerWorkMeter) (providerHistoryCollectionBlock, bool) {
	var zero providerHistoryCollectionBlock
	if meter == nil || len(raw.HeaderRLP) == 0 || len(raw.HeaderRLP) > 65536 || len(raw.Transactions) != len(raw.Receipts) ||
		!nodeJSONSliceObserved(len(raw.Transactions), 2, meter) || !nodeJSONSliceObserved(len(raw.Receipts), 2, meter) {
		return zero, false
	}
	out := providerHistoryCollectionBlock{header: header, transactionCount: len(raw.Transactions), headerRLP: providerReplayString(raw.HeaderRLP, meter),
		transactions: make([]string, len(raw.Transactions)), receipts: make([]string, len(raw.Receipts))}
	if len(out.headerRLP) != len(raw.HeaderRLP) {
		return zero, false
	}
	for i := range raw.Transactions {
		out.transactions[i], out.receipts[i] = providerReplayString(raw.Transactions[i], meter), providerReplayString(raw.Receipts[i], meter)
		if len(out.transactions[i]) != len(raw.Transactions[i]) || len(out.receipts[i]) != len(raw.Receipts[i]) {
			return zero, false
		}
	}
	if raw.Gas != nil {
		out.gasProfile, out.gasTrace = raw.Gas.Profile, providerReplayString(raw.Gas.Trace, meter)
		if len(out.gasTrace) != len(raw.Gas.Trace) {
			return zero, false
		}
	}
	return out, true
}

func (b providerHistoryCollectionBlock) replayObserved(meter *providerWorkMeter) (ReplayBlock, bool) {
	var zero ReplayBlock
	if meter == nil || len(b.headerRLP) == 0 || len(b.headerRLP) > 65536 || len(b.transactions) != len(b.receipts) || len(b.transactions) != b.transactionCount ||
		!nodeJSONSliceObserved(len(b.transactions), 3, meter) || !nodeJSONSliceObserved(len(b.receipts), 3, meter) {
		return zero, false
	}
	out := ReplayBlock{HeaderRLP: providerReplayBytes(b.headerRLP, meter), Transactions: make([][]byte, len(b.transactions)), Receipts: make([][]byte, len(b.receipts))}
	if len(out.HeaderRLP) != len(b.headerRLP) {
		return zero, false
	}
	for i := range b.transactions {
		out.Transactions[i], out.Receipts[i] = providerReplayBytes(b.transactions[i], meter), providerReplayBytes(b.receipts[i], meter)
		if len(out.Transactions[i]) != len(b.transactions[i]) || len(out.Receipts[i]) != len(b.receipts[i]) {
			return zero, false
		}
	}
	if b.gasProfile != "" {
		if !meter.event(providerCopyEvent, uint64(reflect.TypeFor[ReplayGasEvidence]().Size())) {
			return zero, false
		}
		out.Gas = &ReplayGasEvidence{b.gasProfile, b.header.Hash, providerReplayBytes(b.gasTrace, meter)}
		if len(out.Gas.Trace) != len(b.gasTrace) {
			return zero, false
		}
	} else if b.gasTrace != "" {
		return zero, false
	}
	return out, true
}

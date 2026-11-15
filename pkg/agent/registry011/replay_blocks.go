package registry011

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

var (
	ErrReplay            = errors.New("registry011: inconsistent command replay evidence")
	ErrReplayUnavailable = errors.New("registry011: replay evidence or work budget unavailable")
)

// ReplayLimits are explicit source-acquisition/work budgets, NOT protocol caps.
// Exhaustion returns no partial result. There are deliberately no defaults.
type ReplayLimits struct {
	History                                                                     HistoryReadLimits
	MaxBlocks, MaxTransactions, MaxFrames, MaxLogs, MaxProofs, MaxEvidenceBytes int
}

// ReplayBlock carries the COMPLETE ordered canonical block body and receipts.
// Header identity/finality, deployment provenance, code/ABI and the trace
// normalizer must be separately qualified by the Source. Trie equality alone
// does not authenticate an RPC or establish the header's consensus validity.
type ReplayBlock struct {
	HeaderRLP              []byte
	Transactions, Receipts [][]byte
	Executions             []ReplayExecution
	Gas                    *ReplayGasEvidence
}

type replayBudget struct {
	limits                                      ReplayLimits
	bytes, txs, frames, logs, traceLogs, proofs int
	observer                                    *providerWorkMeter
}

func (b *replayBudget) charge(n int) bool {
	if n < 0 || n > b.limits.MaxEvidenceBytes-b.bytes {
		return false
	}
	b.bytes += n
	return true
}
func replayLimits(l ReplayLimits) bool {
	return l.History.MaxItems > 0 && l.History.MaxItems <= 4096 && l.History.MaxBytes > 0 && l.History.MaxBytes <= 8<<20 &&
		l.MaxBlocks > 0 && l.MaxBlocks <= 4096 && l.MaxTransactions > 0 && l.MaxTransactions <= 16384 &&
		l.MaxFrames > 0 && l.MaxFrames <= 65536 && l.MaxLogs > 0 && l.MaxLogs <= 65536 &&
		l.MaxProofs > 0 && l.MaxProofs <= 4096 && l.MaxEvidenceBytes > 0 && l.MaxEvidenceBytes <= 64<<20
}

type replayDecodedBlock struct {
	header   *types.Header
	txs      types.Transactions
	receipts types.Receipts
}

// replayDecodeBlock is the strict command-replay boundary. Pinned fork rules
// require complete gas evidence after activation; content acquisition cannot
// bypass this check. Provider gas assertions are not consensus validation.
func replayDecodeBlock(ctx context.Context, block ReplayBlock, budget *replayBudget) (replayDecodedBlock, error) {
	result, err := replayDecodeContent(ctx, block, budget)
	if err != nil {
		return result, err
	}
	if err := replayCheckGas(ctx, block, result, budget); err != nil {
		return result, err
	}
	return result, nil
}

// replayDecodeContent verifies canonical encodings, complete transaction/receipt
// commitments and their internal consistency. It does not validate the fork's
// block gas accounting or consensus validity and cannot authorize command replay.
func replayDecodeContent(ctx context.Context, block ReplayBlock, budget *replayBudget) (replayDecodedBlock, error) {
	var result replayDecodedBlock
	if err := ctx.Err(); err != nil {
		return result, err
	}
	bound, eligible := providerHistoryNativeInputBoundObserved(budget.observer)
	if !eligible {
		return result, ErrReplayUnavailable
	}
	if !budget.charge(len(block.HeaderRLP)) || len(block.HeaderRLP) > 65536 || len(block.Transactions) != len(block.Receipts) || len(block.Transactions) > budget.limits.MaxTransactions-budget.txs {
		return result, ErrReplayUnavailable
	}
	budget.txs += len(block.Transactions)
	for i, raw := range block.Transactions {
		// Every retained reconstruction rechecks actual original byte lengths
		// before any native transaction decode; a cursor/hash is not a size proof.
		if bound.collection && uint64(len(raw)) > bound.body {
			budget.observer.refuse()
			return result, ErrReplayUnavailable
		}
		if !budget.charge(len(raw)) || !budget.charge(len(block.Receipts[i])) {
			return result, ErrReplayUnavailable
		}
	}
	var header types.Header
	if !budget.blockObserved() || !budget.decodeObserved(block.HeaderRLP) || rlp.DecodeBytes(block.HeaderRLP, &header) != nil || header.Number == nil || !header.Number.IsUint64() || header.Time > 1<<53-1 {
		return result, ErrReplayUnavailable
	}
	if !budget.encodePreflight(len(block.HeaderRLP)) {
		return result, ErrReplayUnavailable
	}
	encoded, e := rlp.EncodeToBytes(&header)
	if e != nil || !budget.encodedObserved(encoded, len(block.HeaderRLP)) || !budget.equalObserved(encoded, block.HeaderRLP) {
		return result, ErrReplay
	}
	result.header = &header
	result.txs = make(types.Transactions, len(block.Transactions))
	result.receipts = make(types.Receipts, len(block.Receipts))
	var gas uint64
	for i, raw := range block.Transactions {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		tx := new(types.Transaction)
		if !budget.transactionObserved() || !budget.decodeObserved(raw) || tx.UnmarshalBinary(raw) != nil || !budget.encodePreflight(len(raw)) {
			return result, ErrReplayUnavailable
		}
		canonical, e := tx.MarshalBinary()
		if e != nil || !budget.encodedObserved(canonical, len(raw)) || !budget.equalObserved(canonical, raw) {
			return result, ErrReplay
		}
		receipt := new(types.Receipt)
		if !budget.decodeObserved(block.Receipts[i]) || receipt.UnmarshalBinary(block.Receipts[i]) != nil || !budget.encodePreflight(len(block.Receipts[i])) {
			return result, ErrReplayUnavailable
		}
		canonical, e = receipt.MarshalBinary()
		if e != nil || !budget.encodedObserved(canonical, len(block.Receipts[i])) || !budget.equalObserved(canonical, block.Receipts[i]) || receipt.Type != tx.Type() {
			return result, ErrReplay
		}
		// The enrolled Sepolia deployment is post-Byzantium; no post-state/status
		// coercion, failed-receipt logs or inconsistent gas is accepted.
		if len(receipt.PostState) != 0 || receipt.Status > 1 || (receipt.Status == 0 && len(receipt.Logs) != 0) || receipt.CumulativeGasUsed < gas {
			return result, ErrReplay
		}
		gas = receipt.CumulativeGasUsed
		if len(receipt.Logs) > budget.limits.MaxLogs-budget.logs || !budget.logsObserved(len(receipt.Logs)) {
			return result, ErrReplayUnavailable
		}
		budget.logs += len(receipt.Logs)
		result.txs[i] = tx
		result.receipts[i] = receipt
	}
	if budget.observer != nil {
		txRoot, txErr := providerHistoryContentRootObserved(ctx, block.Transactions, budget.observer)
		if txErr != nil || !budget.equalObserved(txRoot[:], header.TxHash[:]) {
			return result, ErrReplay
		}
		receiptRoot, receiptErr := providerHistoryContentRootObserved(ctx, block.Receipts, budget.observer)
		if receiptErr != nil || !budget.equalObserved(receiptRoot[:], header.ReceiptHash[:]) {
			return result, ErrReplay
		}
	} else if types.DeriveSha(result.txs, trie.NewStackTrie(nil)) != header.TxHash || types.DeriveSha(result.receipts, trie.NewStackTrie(nil)) != header.ReceiptHash {
		return result, ErrReplay
	}
	return result, nil
}

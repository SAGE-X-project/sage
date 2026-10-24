package registry011

import (
	"bytes"
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
	if !budget.charge(len(block.HeaderRLP)) || len(block.HeaderRLP) > 65536 || len(block.Transactions) != len(block.Receipts) || len(block.Transactions) > budget.limits.MaxTransactions-budget.txs {
		return result, ErrReplayUnavailable
	}
	budget.txs += len(block.Transactions)
	for i, raw := range block.Transactions {
		if !budget.charge(len(raw)) || !budget.charge(len(block.Receipts[i])) {
			return result, ErrReplayUnavailable
		}
	}
	var header types.Header
	if rlp.DecodeBytes(block.HeaderRLP, &header) != nil || header.Number == nil || !header.Number.IsUint64() || header.Time > 1<<53-1 {
		return result, ErrReplayUnavailable
	}
	encoded, e := rlp.EncodeToBytes(&header)
	if e != nil || !bytes.Equal(encoded, block.HeaderRLP) {
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
		if tx.UnmarshalBinary(raw) != nil {
			return result, ErrReplayUnavailable
		}
		canonical, e := tx.MarshalBinary()
		if e != nil || !bytes.Equal(canonical, raw) {
			return result, ErrReplay
		}
		receipt := new(types.Receipt)
		if receipt.UnmarshalBinary(block.Receipts[i]) != nil {
			return result, ErrReplayUnavailable
		}
		canonical, e = receipt.MarshalBinary()
		if e != nil || !bytes.Equal(canonical, block.Receipts[i]) || receipt.Type != tx.Type() {
			return result, ErrReplay
		}
		// The enrolled Sepolia deployment is post-Byzantium; no post-state/status
		// coercion, failed-receipt logs or inconsistent gas is accepted.
		if len(receipt.PostState) != 0 || receipt.Status > 1 || (receipt.Status == 0 && len(receipt.Logs) != 0) || receipt.CumulativeGasUsed < gas {
			return result, ErrReplay
		}
		gas = receipt.CumulativeGasUsed
		if len(receipt.Logs) > budget.limits.MaxLogs-budget.logs {
			return result, ErrReplayUnavailable
		}
		budget.logs += len(receipt.Logs)
		result.txs[i] = tx
		result.receipts[i] = receipt
	}
	if types.DeriveSha(result.txs, trie.NewStackTrie(nil)) != header.TxHash || types.DeriveSha(result.receipts, trie.NewStackTrie(nil)) != header.ReceiptHash {
		return result, ErrReplay
	}
	return result, nil
}

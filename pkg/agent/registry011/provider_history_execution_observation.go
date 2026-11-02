package registry011

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
)

// These are private normalized DATA checks, not a producer-controlled live
// adapter. An original collector must separately authenticate complete raw
// call/log coverage, retain it and bind its one original observer beforehand.
func replayExecutionObserved(ctx context.Context, e *ReplayExecution, block replayDecodedBlock, index int, b *replayBudget) ([]replayLog, error) {
	if ctx == nil || ctx.Err() != nil || e == nil || b == nil || b.observer == nil || block.header == nil || index < 0 || index >= len(block.txs) || index >= len(block.receipts) {
		return nil, ErrReplayUnavailable
	}
	m := b.observer
	tx, receipt := block.txs[index], block.receipts[index]
	if tx == nil || receipt == nil || !m.preflightWork(candidateWorkVector{}) {
		return nil, ErrReplayUnavailable
	}
	hash, headerOK := nodeHeaderHashObserved(block.header, m)
	txHash, txOK := nodeTransactionHashObserved(tx, m)
	headerText, headerTextOK := nodeHashTextObserved(hash, m)
	txText, txTextOK := nodeHashTextObserved(txHash, m)
	if !headerOK || !txOK || !headerTextOK || !txTextOK || e.TransactionIndex != uint64(index) || !providerReplayTextEqual(e.BlockHash, headerText, m) || !providerReplayTextEqual(e.TransactionHash, txText, m) || receipt.Status != 1 {
		return nil, ErrReplay
	}
	// Keep the existing stricter command-execution type/protection gate. The
	// separate sender/hash codecs supporting more types do not widen this API.
	if tx.Type() > types.DynamicFeeTxType || !tx.Protected() || tx.ChainId().Cmp(big.NewInt(11155111)) != 0 {
		return nil, ErrReplayUnavailable
	}
	sender, err := nodeSenderObserved(tx, m)
	if err != nil {
		return nil, err
	}
	address, addressOK := nodeAddressTextObserved(sender, m)
	root := &e.Root
	value, valueOK := nodeValueObserved(root.Value, m)
	amount, amountOK := nodeTransactionValueObserved(tx, m)
	if !addressOK || !valueOK || !amountOK || !providerReplayTextEqual(root.From, address, m) || !m.event(providerCompareEvent, uint64((value.BitLen()+7)/8+(amount.BitLen()+7)/8)) || value.Cmp(amount) != 0 || !providerReplayEqual(root.Input, tx.Data(), m) || root.Error != "" {
		return nil, ErrReplay
	}
	to, toOK := nodeTransactionToObserved(tx, m)
	if !toOK {
		return nil, ErrReplayUnavailable
	}
	kind := "CALL"
	target := sender
	if to == nil {
		kind = "CREATE"
		created, ok := nodeCreateAddressObserved(sender, tx.Nonce(), m)
		if !ok {
			return nil, ErrReplayUnavailable
		}
		target = created
	} else {
		target = *to
	}
	targetText, targetOK := nodeAddressTextObserved(target, m)
	if !targetOK || !providerReplayTextEqual(root.Type, kind, m) || !providerReplayTextEqual(root.To, targetText, m) {
		return nil, ErrReplay
	}
	logs, err := replayFrameObserved(ctx, root, "", false, true, 0, b)
	if err != nil {
		return nil, err
	}
	if len(logs) != len(receipt.Logs) {
		return nil, ErrReplay
	}
	for i, l := range logs {
		r := receipt.Logs[i]
		if r == nil || l.log == nil || !providerReplayEqual(l.log.Address[:], r.Address[:], m) || !providerReplayEqual(l.log.Data, r.Data, m) || len(l.log.Topics) != len(r.Topics) {
			return nil, ErrReplay
		}
		for j := range l.log.Topics {
			if !providerReplayEqual(l.log.Topics[j][:], r.Topics[j][:], m) {
				return nil, ErrReplay
			}
		}
	}
	if ctx.Err() != nil || !m.preflightWork(candidateWorkVector{}) {
		return nil, ErrReplayUnavailable
	}
	return logs, nil
}

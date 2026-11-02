package registry011

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

// Match the pinned Sepolia signer preimage for each supported transaction
// type. No supplied hash, cached signer, implicit network or producer-chosen
// output upper is accepted. Accessor allocations remain in full-cost scope.
func nodeSigningHashObserved(tx *types.Transaction, meter *providerWorkMeter) (common.Hash, error) {
	if tx == nil || tx.Type() > types.SetCodeTxType || meter == nil {
		return common.Hash{}, ErrReplayUnavailable
	}
	bound, eligible := providerHistoryNativeInputBoundObserved(meter)
	if !eligible || !meter.preflight(providerEncodeEvent, bound.signing) {
		return common.Hash{}, ErrReplayUnavailable
	}
	chain := big.NewInt(11155111)
	var fields []any
	if tx.Type() == types.LegacyTxType {
		fields = []any{tx.Nonce(), tx.GasPrice(), tx.Gas(), tx.To(), tx.Value(), tx.Data()}
		if tx.Protected() {
			if !tx.ChainId().IsUint64() || tx.ChainId().Uint64() != 11155111 {
				return common.Hash{}, ErrReplayUnavailable
			}
			fields = append(fields, chain, uint(0), uint(0))
		}
	} else {
		if !tx.ChainId().IsUint64() || tx.ChainId().Uint64() != 11155111 {
			return common.Hash{}, ErrReplayUnavailable
		}
		switch tx.Type() {
		case types.AccessListTxType:
			fields = []any{chain, tx.Nonce(), tx.GasPrice(), tx.Gas(), tx.To(), tx.Value(), tx.Data(), tx.AccessList()}
		case types.DynamicFeeTxType:
			fields = []any{chain, tx.Nonce(), tx.GasTipCap(), tx.GasFeeCap(), tx.Gas(), tx.To(), tx.Value(), tx.Data(), tx.AccessList()}
		case types.BlobTxType:
			fields = []any{chain, tx.Nonce(), tx.GasTipCap(), tx.GasFeeCap(), tx.Gas(), tx.To(), tx.Value(), tx.Data(), tx.AccessList(), tx.BlobGasFeeCap(), tx.BlobHashes()}
		case types.SetCodeTxType:
			fields = []any{chain, tx.Nonce(), tx.GasTipCap(), tx.GasFeeCap(), tx.Gas(), tx.To(), tx.Value(), tx.Data(), tx.AccessList(), tx.SetCodeAuthorizations()}
		}
	}
	// The original decoded transaction has its proved input ceiling. Omitting its
	// signature and inserting the fixed Sepolia chain/zero tuple can add only
	// constant RLP overhead.256 bytes includes the type and every length prefix.
	if !meter.preflight(providerEncodeEvent, bound.signing) {
		return common.Hash{}, ErrReplayUnavailable
	}
	raw, err := rlp.EncodeToBytes(fields)
	if err != nil || uint64(len(raw)) > bound.signing-1 {
		meter.refuse()
		return common.Hash{}, ErrReplayUnavailable
	}
	if !meter.event(providerEncodeEvent, uint64(len(raw))) {
		return common.Hash{}, ErrReplayUnavailable
	}
	if tx.Type() != types.LegacyTxType {
		if !meter.event(providerCopyEvent, uint64(len(raw)+1)) {
			return common.Hash{}, ErrReplayUnavailable
		}
		typed := make([]byte, len(raw)+1)
		typed[0] = tx.Type()
		copy(typed[1:], raw)
		raw = typed
	}
	if !meter.event(providerHashEvent, uint64(len(raw))) {
		return common.Hash{}, ErrReplayUnavailable
	}
	return crypto.Keccak256Hash(raw), nil
}

package registry011

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// A private DATA reader for already bounded, completely decoded transactions.
// The nil entry keeps the library API. The observed entry deliberately does
// not reuse its sender cache: every actual recovery and address hash is visible
// to the original observer. The fixed native recovery primitive still needs
// full source-cost, CPU and backing-memory qualification before admission.
func nodeSenderObserved(tx *types.Transaction, meter *providerWorkMeter) (common.Address, error) {
	if tx == nil {
		return common.Address{}, ErrReplayUnavailable
	}
	if meter == nil {
		return types.Sender(types.LatestSignerForChainID(big.NewInt(11155111)), tx)
	}
	if tx.Type() > types.SetCodeTxType || !meter.preflight(providerCompareEvent, 65) {
		return common.Address{}, ErrReplayUnavailable
	}
	v, r, s := tx.RawSignatureValues()
	if v == nil || r == nil || s == nil {
		return common.Address{}, ErrReplayUnavailable
	}
	if v.Sign() < 0 || r.Sign() < 0 || s.Sign() < 0 || v.BitLen() > 256 || r.BitLen() > 256 || s.BitLen() > 256 {
		return common.Address{}, ErrReplay
	}
	var recovery byte
	if tx.Type() == types.LegacyTxType && !tx.Protected() {
		if !v.IsUint64() || v.Uint64() < 27 || v.Uint64() > 28 {
			return common.Address{}, ErrReplay
		}
		recovery = byte(v.Uint64() - 27)
	} else {
		chain := tx.ChainId()
		if chain == nil || !chain.IsUint64() || chain.Uint64() != 11155111 || !v.IsUint64() {
			return common.Address{}, ErrReplayUnavailable
		}
		n := v.Uint64()
		if tx.Type() == types.LegacyTxType {
			if n < 2*11155111+35 || n > 2*11155111+36 {
				return common.Address{}, ErrReplay
			}
			n -= 2*11155111 + 35
		}
		if n > 1 {
			return common.Address{}, ErrReplay
		}
		recovery = byte(n)
	}
	// The primitive compares the fixed32-byte R,32-byte S and recovery byte.
	if !meter.event(providerCompareEvent, 65) {
		return common.Address{}, ErrReplayUnavailable
	}
	if !crypto.ValidateSignatureValues(recovery, r, s, true) {
		return common.Address{}, ErrReplay
	}
	hash, err := nodeSigningHashObserved(tx, meter)
	if err != nil {
		return common.Address{}, err
	}
	if !meter.event(providerEncodeEvent, 65) {
		return common.Address{}, ErrReplayUnavailable
	}
	var signature [65]byte
	r.FillBytes(signature[:32])
	s.FillBytes(signature[32:64])
	signature[64] = recovery
	// This is one concrete binary recovery codec with97 actual input bytes,
	// not a claimed cost, generic callback or synthetic Proofs increment.
	if !meter.event(providerCodecEvent, 97) {
		return common.Address{}, ErrReplayUnavailable
	}
	public, err := crypto.Ecrecover(hash[:], signature[:])
	if err != nil {
		return common.Address{}, ErrReplay
	}
	return nodeRecoveredPublicKeyAddressObserved(public, meter)
}

// The original signature reader alone obtains a recovered public key. This
// private byte-to-address stage supplies DATA, never account authorization.
func nodeRecoveredPublicKeyAddressObserved(public []byte, meter *providerWorkMeter) (common.Address, error) {
	if len(public) != 65 || public[0] != 4 {
		return common.Address{}, ErrReplay
	}
	if !meter.event(providerHashEvent, 64) {
		return common.Address{}, ErrReplayUnavailable
	}
	digest := crypto.Keccak256Hash(public[1:])
	if !meter.event(providerCopyEvent, 20) {
		return common.Address{}, ErrReplayUnavailable
	}
	return common.BytesToAddress(digest[12:]), nil
}

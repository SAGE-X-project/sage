package registry011

import (
	"encoding/hex"
	"reflect"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

func nodeHashTextObserved(hash common.Hash, meter *providerWorkMeter) (string, bool) {
	if meter != nil && !meter.event(providerEncodeEvent, 66) {
		return "", false
	}
	return hash.Hex(), true
}

func nodeAddressTextObserved(address common.Address, meter *providerWorkMeter) (string, bool) {
	if meter == nil {
		return replayAddress(address), true
	}
	if !meter.event(providerEncodeEvent, 42) {
		return "", false
	}
	// The existing result is lower-case; do not compute and then discard an
	// EIP-55 checksum just to obtain those same lower-case address bytes.
	return "0x" + hex.EncodeToString(address[:]), true
}

func nodeHeaderHashObserved(header *types.Header, meter *providerWorkMeter) (common.Hash, bool) {
	if header == nil {
		return common.Hash{}, false
	}
	if meter == nil {
		return header.Hash(), true
	}
	// The original binary reader caps headers at64KiB. JSON headers have23
	// bounded fields. This is an encoding preflight upper, not a reported size.
	if !meter.preflight(providerEncodeEvent, 65536) {
		return common.Hash{}, false
	}
	raw, err := rlp.EncodeToBytes(header)
	if err != nil || len(raw) > 65536 {
		meter.refuse()
		return common.Hash{}, false
	}
	return nodeEncodedHashObserved(raw, meter)
}

func nodeTransactionHashObserved(tx *types.Transaction, meter *providerWorkMeter) (common.Hash, bool) {
	if tx == nil {
		return common.Hash{}, false
	}
	if meter == nil {
		return tx.Hash(), true
	}
	// The original collection callers guard actual raw bytes before decode;
	// their closed JSON size proof also covers pre-canonical rejecting rows.
	// Other original roles keep the existing64MiB encoding ceiling.
	bound, eligible := providerHistoryNativeInputBoundObserved(meter)
	if !eligible || !meter.preflight(providerEncodeEvent, bound.transaction) {
		return common.Hash{}, false
	}
	// A network blob sidecar is not in the transaction hash. This fixed
	// library operation copies the Transaction and BlobTx structs only; its
	// immutable payload references remain shared. It does not copy blob data.
	if tx.BlobTxSidecar() != nil {
		if !meter.event(providerCopyEvent, uint64(reflect.TypeFor[types.Transaction]().Size())) ||
			!meter.event(providerCopyEvent, uint64(reflect.TypeFor[types.BlobTx]().Size())) {
			return common.Hash{}, false
		}
		tx = tx.WithoutBlobTxSidecar()
	}
	raw, err := tx.MarshalBinary()
	if err != nil || uint64(len(raw)) > bound.transaction {
		meter.refuse()
		return common.Hash{}, false
	}
	return nodeEncodedHashObserved(raw, meter)
}

func nodeCreateAddressObserved(sender common.Address, nonce uint64, meter *providerWorkMeter) (common.Address, bool) {
	if meter == nil {
		return crypto.CreateAddress(sender, nonce), true
	}
	if !meter.preflight(providerEncodeEvent, 64) {
		return common.Address{}, false
	}
	raw, err := rlp.EncodeToBytes([]any{sender, nonce})
	if err != nil || len(raw) > 64 {
		meter.refuse()
		return common.Address{}, false
	}
	hash, ok := nodeEncodedHashObserved(raw, meter)
	if !ok {
		return common.Address{}, false
	}
	if !meter.event(providerCopyEvent, 20) {
		return common.Address{}, false
	}
	return common.BytesToAddress(hash[12:]), true
}

// An encoding stage over already bounded bytes. Callers keep their original
// prospective preflight and native input checks; this helper grants no permit.
func nodeEncodedHashObserved(raw []byte, meter *providerWorkMeter) (common.Hash, bool) {
	if !meter.event(providerEncodeEvent, uint64(len(raw))) || !meter.event(providerHashEvent, uint64(len(raw))) {
		return common.Hash{}, false
	}
	return crypto.Keccak256Hash(raw), true
}

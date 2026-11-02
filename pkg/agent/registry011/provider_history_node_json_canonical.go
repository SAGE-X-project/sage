package registry011

import (
	"encoding/json"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

// This is a closed typed JSON binding, not an arbitrary caller-supplied
// marshaler. Header/transaction JSON methods hide an extra commitment hash;
// build their pinned field values with an explicitly observed fresh hash.
// Accessor allocations and native codec internals still need the separate
// complete cost/RAM qualification before any collector can be admitted.
func nodeJSONCanonicalObserved(fields map[string]json.RawMessage, value any, extras string, sourceBytes int, meter *providerWorkMeter) bool {
	if meter == nil {
		return nodeJSONCanonical(fields, value, extras)
	}
	if sourceBytes < 1 || sourceBytes > 64<<20 || !meter.preflightWork(candidateWorkVector{}) {
		return false
	}
	expected, ok := nodeJSONTypedValueObserved(value, meter)
	if !ok {
		return false
	}
	// All supported values contain only fixed keys, hex strings, bounded
	// numbers and their typed lists. Canonical output cannot grow beyond twice
	// the original JSON plus the fixed optional-key allowance. This is an
	// execution preflight, never a claimed/measured output length.
	maximum := 2*sourceBytes + 65536
	if !meter.preflight(providerEncodeEvent, uint64(maximum)) {
		return false
	}
	raw, err := json.Marshal(expected)
	if err != nil || len(raw) > maximum {
		return meter.refuse()
	}
	if !meter.event(providerEncodeEvent, uint64(len(raw))) {
		return false
	}
	var canonical map[string]json.RawMessage
	if providerReplayUnmarshal(raw, &canonical, meter) != nil || canonical == nil {
		return false
	}
	for key := range fields {
		if !meter.event(providerCompareEvent, uint64(len(key))) {
			return false
		}
		if _, known := canonical[key]; !known && !nodeJSONExtraObserved(key, extras, meter) {
			return false
		}
	}
	for key, want := range canonical {
		if !meter.event(providerCompareEvent, uint64(len(key))) {
			return false
		}
		got, present := fields[key]
		if !present && nodeLiteralObserved(want, "null", meter) || key == "yParity" && !present && len(fields["v"]) != 0 {
			continue
		}
		var a, b any
		if !present || providerReplayUnmarshal(got, &a, meter) != nil || providerReplayUnmarshal(want, &b, meter) != nil || !nodeJSONValueEqualObserved(a, b, 0, meter) {
			return false
		}
	}
	return meter.preflightWork(candidateWorkVector{})
}

func nodeJSONExtraObserved(key, extras string, meter *providerWorkMeter) bool {
	for _, allowed := range strings.Fields(extras) {
		if !meter.event(providerCompareEvent, uint64(len(key)+len(allowed))) {
			return false
		}
		if key == allowed {
			return true
		}
	}
	return false
}

func nodeJSONTypedValueObserved(value any, meter *providerWorkMeter) (map[string]any, bool) {
	switch v := value.(type) {
	case types.Header:
		return nodeJSONHeaderValueObserved(&v, meter)
	case *types.Transaction:
		return nodeJSONTxValueObserved(v, meter)
	case types.Withdrawal:
		return map[string]any{"index": hexutil.Uint64(v.Index), "validatorIndex": hexutil.Uint64(v.Validator), "address": v.Address, "amount": hexutil.Uint64(v.Amount)}, true
	default:
		return nil, meter.refuse()
	}
}

func nodeJSONHeaderValueObserved(h *types.Header, meter *providerWorkMeter) (map[string]any, bool) {
	hash, ok := nodeHeaderHashObserved(h, meter)
	if !ok {
		return nil, false
	}
	return map[string]any{
		"parentHash": h.ParentHash, "sha3Uncles": h.UncleHash, "miner": h.Coinbase,
		"stateRoot": h.Root, "transactionsRoot": h.TxHash, "receiptsRoot": h.ReceiptHash, "logsBloom": h.Bloom,
		"difficulty": (*hexutil.Big)(h.Difficulty), "number": (*hexutil.Big)(h.Number), "gasLimit": hexutil.Uint64(h.GasLimit), "gasUsed": hexutil.Uint64(h.GasUsed), "timestamp": hexutil.Uint64(h.Time),
		"extraData": hexutil.Bytes(h.Extra), "mixHash": h.MixDigest, "nonce": h.Nonce, "baseFeePerGas": (*hexutil.Big)(h.BaseFee), "withdrawalsRoot": h.WithdrawalsHash,
		"blobGasUsed": (*hexutil.Uint64)(h.BlobGasUsed), "excessBlobGas": (*hexutil.Uint64)(h.ExcessBlobGas), "parentBeaconBlockRoot": h.ParentBeaconRoot,
		"requestsHash": h.RequestsHash, "blockAccessListHash": h.BlockAccessListHash, "slotNumber": (*hexutil.Uint64)(h.SlotNumber), "hash": hash,
	}, true
}

func nodeJSONTxValueObserved(tx *types.Transaction, meter *providerWorkMeter) (map[string]any, bool) {
	if tx == nil || tx.Type() > types.SetCodeTxType || tx.BlobTxSidecar() != nil {
		return nil, meter.refuse()
	}
	hash, ok := nodeTransactionHashObserved(tx, meter)
	if !ok {
		return nil, false
	}
	v, r, s := tx.RawSignatureValues()
	fields := map[string]any{
		"type": hexutil.Uint64(tx.Type()), "nonce": hexutil.Uint64(tx.Nonce()), "to": tx.To(), "gas": hexutil.Uint64(tx.Gas()),
		"gasPrice": nil, "maxPriorityFeePerGas": nil, "maxFeePerGas": nil, "value": (*hexutil.Big)(tx.Value()), "input": hexutil.Bytes(tx.Data()),
		"v": (*hexutil.Big)(v), "r": (*hexutil.Big)(r), "s": (*hexutil.Big)(s), "hash": hash,
	}
	if tx.Type() == types.LegacyTxType {
		fields["gasPrice"] = (*hexutil.Big)(tx.GasPrice())
		if tx.Protected() {
			fields["chainId"] = (*hexutil.Big)(tx.ChainId())
		}
		return fields, true
	}
	list := tx.AccessList()
	fields["chainId"], fields["accessList"], fields["yParity"] = (*hexutil.Big)(tx.ChainId()), &list, hexutil.Uint64(v.Uint64())
	if tx.Type() == types.AccessListTxType {
		fields["gasPrice"] = (*hexutil.Big)(tx.GasPrice())
	} else {
		fields["maxPriorityFeePerGas"], fields["maxFeePerGas"] = (*hexutil.Big)(tx.GasTipCap()), (*hexutil.Big)(tx.GasFeeCap())
	}
	if tx.Type() == types.BlobTxType {
		fields["maxFeePerBlobGas"] = (*hexutil.Big)(tx.BlobGasFeeCap())
		if hashes := tx.BlobHashes(); len(hashes) != 0 {
			fields["blobVersionedHashes"] = hashes
		}
	}
	if tx.Type() == types.SetCodeTxType {
		if authorizations := tx.SetCodeAuthorizations(); len(authorizations) != 0 {
			fields["authorizationList"] = authorizations
		}
	}
	return fields, true
}

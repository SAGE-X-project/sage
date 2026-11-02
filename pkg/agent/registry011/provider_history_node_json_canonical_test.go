package registry011

import (
	"encoding/json"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

func providerHistoryNodeJSONUnitFields(t *testing.T, v any) (map[string]json.RawMessage, []byte) {
	t.Helper()
	raw, err := json.Marshal(v)
	var fields map[string]json.RawMessage
	if err != nil || json.Unmarshal(raw, &fields) != nil || fields == nil {
		t.Fatal("pinned typed JSON precursor", err)
	}
	return fields, raw
}

func providerHistoryNodeJSONUnitTransactions() []*types.Transaction {
	address := common.HexToAddress(replayTestWallet)
	return []*types.Transaction{
		types.NewTx(&types.LegacyTx{Nonce: 1, GasPrice: big.NewInt(2), Gas: 21000, To: &address, Value: big.NewInt(3), Data: []byte{0, 1}, V: big.NewInt(27), R: big.NewInt(1), S: big.NewInt(1)}),
		types.NewTx(&types.LegacyTx{GasPrice: big.NewInt(2), Gas: 21000, Value: big.NewInt(3), V: big.NewInt(11155111*2 + 35), R: big.NewInt(1), S: big.NewInt(1)}),
		types.NewTx(&types.AccessListTx{ChainID: big.NewInt(11155111), GasPrice: big.NewInt(2), Gas: 21000, To: &address, Value: big.NewInt(3), AccessList: types.AccessList{{Address: address, StorageKeys: []common.Hash{{1}}}}, V: big.NewInt(1), R: big.NewInt(1), S: big.NewInt(1)}),
		types.NewTx(&types.DynamicFeeTx{ChainID: big.NewInt(11155111), GasTipCap: big.NewInt(1), GasFeeCap: big.NewInt(2), Gas: 21000, To: &address, Value: big.NewInt(3), V: big.NewInt(1), R: big.NewInt(1), S: big.NewInt(1)}),
		types.NewTx(&types.BlobTx{ChainID: uint256.NewInt(11155111), GasTipCap: uint256.NewInt(1), GasFeeCap: uint256.NewInt(2), Gas: 21000, To: address, Value: uint256.NewInt(3), BlobFeeCap: uint256.NewInt(1), BlobHashes: []common.Hash{{1}}, V: uint256.NewInt(1), R: uint256.NewInt(1), S: uint256.NewInt(1)}),
		types.NewTx(&types.SetCodeTx{ChainID: uint256.NewInt(11155111), GasTipCap: uint256.NewInt(1), GasFeeCap: uint256.NewInt(2), Gas: 21000, To: address, Value: uint256.NewInt(3), AuthList: []types.SetCodeAuthorization{{}}, V: uint256.NewInt(1), R: uint256.NewInt(1), S: uint256.NewInt(1)}),
		types.NewTx(&types.BlobTx{ChainID: uint256.NewInt(11155111), GasTipCap: uint256.NewInt(1), GasFeeCap: uint256.NewInt(2), Gas: 21000, To: address, Value: uint256.NewInt(3), BlobFeeCap: uint256.NewInt(1), V: uint256.NewInt(1), R: uint256.NewInt(1), S: uint256.NewInt(1)}),
		types.NewTx(&types.SetCodeTx{ChainID: uint256.NewInt(11155111), GasTipCap: uint256.NewInt(1), GasFeeCap: uint256.NewInt(2), Gas: 21000, To: address, Value: uint256.NewInt(3), V: uint256.NewInt(1), R: uint256.NewInt(1), S: uint256.NewInt(1)}),
	}
}

func TestProviderHistoryNodeJSONTypedBindingsMatchPinnedCanonicalFields(t *testing.T) {
	// Inert unsigned hash/codec values. They are not authorization, valid
	// signatures, blob commitments or network/operating subjects.
	for i, tx := range providerHistoryNodeJSONUnitTransactions() {
		fields, raw := providerHistoryNodeJSONUnitFields(t, tx)
		m := providerReplayUnitMeter()
		if !nodeJSONCanonical(fields, tx, "") || !nodeJSONCanonicalObserved(fields, tx, "", len(raw), m) || m.actual[2] == 0 || m.actual[10] == 0 {
			t.Fatal("pinned transaction canonical contract", i, m.actual)
		}
		if !nodeJSONCanonicalObserved(fields, tx, "", len(raw), nil) {
			t.Fatal("legacy canonical entry", i)
		}
		if _, present := fields["yParity"]; present {
			delete(fields, "yParity")
			if !nodeJSONCanonicalObserved(fields, tx, "", len(raw), providerReplayUnitMeter()) {
				t.Fatal("existing explicit-v alias", i)
			}
		}
	}
	input := replayTestInput(t).Blocks[0]
	var header types.Header
	if rlp.DecodeBytes(input.HeaderRLP, &header) != nil {
		t.Fatal("canonical binary header precursor")
	}
	for _, value := range []any{header, types.Withdrawal{Index: 1, Validator: 2, Address: common.Address{1}, Amount: 3}} {
		fields, raw := providerHistoryNodeJSONUnitFields(t, value)
		if !nodeJSONCanonicalObserved(fields, value, "", len(raw), providerReplayUnitMeter()) {
			t.Fatal("header/withdrawal canonical contract", reflect.TypeOf(value))
		}
		for key, rawValue := range fields {
			if string(rawValue) == "null" {
				delete(fields, key)
			}
		}
		if !nodeJSONCanonicalObserved(fields, value, "", len(raw), providerReplayUnitMeter()) {
			t.Fatal("omitted null-only optional fields")
		}
	}
}

func TestProviderHistoryNodeJSONCanonicalRefusesChangesAndUnsupportedValues(t *testing.T) {
	v := types.Withdrawal{Index: 1, Validator: 2, Address: common.Address{1}, Amount: 3}
	for _, tc := range []struct {
		name string
		edit func(map[string]json.RawMessage)
	}{
		{"unknown", func(m map[string]json.RawMessage) { m["extra"] = json.RawMessage("true") }},
		{"missing", func(m map[string]json.RawMessage) { delete(m, "amount") }},
		{"value", func(m map[string]json.RawMessage) { m["amount"] = json.RawMessage(`"0x4"`) }},
		{"invalid-json", func(m map[string]json.RawMessage) { m["amount"] = json.RawMessage(`"`) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fields, raw := providerHistoryNodeJSONUnitFields(t, v)
			tc.edit(fields)
			if nodeJSONCanonicalObserved(fields, v, "", len(raw), providerReplayUnitMeter()) {
				t.Fatal("changed/missing canonical field accepted")
			}
		})
	}
	fields, raw := providerHistoryNodeJSONUnitFields(t, v)
	fields["providerField"] = json.RawMessage(`"retained separately"`)
	if !nodeJSONCanonicalObserved(fields, v, "providerField", len(raw), providerReplayUnitMeter()) {
		t.Fatal("explicit unchanged extras contract")
	}
	if nodeJSONCanonicalObserved(fields, v, "another", len(raw), providerReplayUnitMeter()) {
		t.Fatal("different extra field")
	}
	for _, size := range []int{0, -1, (64 << 20) + 1} {
		if nodeJSONCanonicalObserved(fields, v, "providerField", size, providerReplayUnitMeter()) {
			t.Fatal("closed source byte ceiling", size)
		}
	}
	for _, value := range []any{true, (*types.Transaction)(nil)} {
		m := providerReplayUnitMeter()
		if nodeJSONCanonicalObserved(fields, value, "", len(raw), m) || !m.failed {
			t.Fatal("unsupported typed binding")
		}
	}
	sidecar := providerHistoryNodeJSONUnitTransactions()[4].WithBlobTxSidecar(&types.BlobTxSidecar{})
	sidecarFields, sidecarRaw := providerHistoryNodeJSONUnitFields(t, sidecar)
	if nodeJSONCanonicalObserved(sidecarFields, sidecar, "", len(sidecarRaw), providerReplayUnitMeter()) {
		t.Fatal("closed JSON block codec cannot acquire a network sidecar")
	}
	m := providerHistoryContentUnitMeter(candidateWorkVector{})
	if nodeJSONCanonicalObserved(fields, v, "providerField", len(raw), m) || !m.failed {
		t.Fatal("unfunded canonical encoder")
	}
	before := m.actual
	if nodeJSONCanonicalObserved(fields, v, "providerField", len(raw), m) || m.actual != before {
		t.Fatal("original failed meter restarted")
	}
}

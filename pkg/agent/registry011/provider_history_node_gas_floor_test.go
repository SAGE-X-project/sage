package registry011

import (
	"math/big"
	"reflect"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestProviderHistoryNodeGasFloorUsesFreshRecordedSenderAndSameRule(t *testing.T) {
	_, block := providerHistoryNodeRecordedContent(t)
	for i, tx := range block.txs {
		want, err := replayGasFloor(tx)
		if err != nil {
			t.Fatal("normal fixed-profile precursor", i, err)
		}
		legacy, err := replayGasFloorObserved(tx, nil)
		if err != nil || legacy != want {
			t.Fatal("legacy gas-floor entry", i, err)
		}
		m := providerReplayUnitMeter()
		got, err := replayGasFloorObserved(tx, m)
		if err != nil || got != want || m.actual[2] != 97 || m.failed {
			t.Fatal("gas-floor rule or original fresh recovery", i, err, got, want, m.actual)
		}
	}
	if _, err := replayGasFloorObserved(nil, providerReplayUnitMeter()); err == nil {
		t.Fatal("missing typed transaction")
	}
	if _, err := replayGasFloorObserved(block.txs[0], providerHistoryContentUnitMeter(candidateWorkVector{})); err == nil {
		t.Fatal("unfunded original recovery")
	}
}

func TestProviderHistoryNodeDerivedGasFloorArithmeticBranches(t *testing.T) {
	transactions := providerHistoryNodeJSONUnitTransactions()
	address := common.HexToAddress(replayTestWallet)
	other := common.Address{1}
	for _, tc := range []struct {
		name string
		tx   *types.Transaction
		from common.Address
		want uint64
	}{
		{"create", transactions[1], other, 24000},
		{"self", transactions[0], address, 12000 + 64*2},
		{"other-value", transactions[0], other, 21000 + 64*2},
		{"access-list", transactions[2], other, 21000 + 64*(20+32)},
		{"other-zero", types.NewTx(&types.LegacyTx{GasPrice: big.NewInt(1), Gas: 21000, To: &address, Value: new(big.Int)}), other, 15000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := providerReplayUnitMeter()
			got, err := replayGasFloorDerivedObserved(tc.tx, tc.from, m)
			if err != nil || got != tc.want || m.actual[2] != 0 {
				t.Fatal("owned arithmetic DATA after separate sender recovery", got, tc.want, err, m.actual)
			}
		})
	}
	if _, err := replayGasFloorDerivedObserved(nil, other, providerReplayUnitMeter()); err == nil {
		t.Fatal("missing derived transaction")
	}
	if _, err := replayGasFloorDerivedObserved(transactions[0], other, nil); err == nil {
		t.Fatal("missing original observer")
	}
	if _, err := replayGasFloorDerivedObserved(transactions[0], other, providerHistoryContentUnitMeter(candidateWorkVector{})); err == nil {
		t.Fatal("unfunded derived arithmetic")
	}
}

func TestProviderHistoryNodeAccessorActualCopiesAreNotTheirPreflightMaxima(t *testing.T) {
	for _, tx := range providerHistoryNodeJSONUnitTransactions() {
		m := providerReplayUnitMeter()
		to, ok := nodeTransactionToObserved(tx, m)
		want := uint64(0)
		if to != nil {
			want = 21
		}
		if !ok || m.actual[10] != want || m.actual[2] != 0 {
			t.Fatal("recipient actual vs preflight", m.actual, want)
		}
		m = providerReplayUnitMeter()
		value, valid := nodeTransactionValueObserved(tx, m)
		if !valid {
			t.Fatal("funded value copy precursor")
		}
		n := uint64(reflect.TypeFor[big.Int]().Size()) + uint64(len(value.Bits()))*uint64(strconv.IntSize/8)
		if !valid || m.actual[10] != n+1 || m.actual[10] >= 64<<20 || m.actual[2] != 0 {
			t.Fatal("value actual vs64MiB preflight", m.actual, n)
		}
	}
}

func TestProviderHistoryNodeTransactionAccessorCopiesKeepOriginalValues(t *testing.T) {
	for _, tx := range providerHistoryNodeJSONUnitTransactions() {
		m := providerReplayUnitMeter()
		to, ok := nodeTransactionToObserved(tx, m)
		want := tx.To()
		if !ok || (to == nil) != (want == nil) || to != nil && *to != *want {
			t.Fatal("recipient accessor value")
		}
		value, valid := nodeTransactionValueObserved(tx, m)
		if !valid || value.Cmp(tx.Value()) != 0 {
			t.Fatal("value accessor copy")
		}
		if _, ok := nodeTransactionToObserved(tx, providerHistoryContentUnitMeter(candidateWorkVector{})); ok {
			t.Fatal("unfunded recipient accessor")
		}
		if _, ok := nodeTransactionValueObserved(tx, providerHistoryContentUnitMeter(candidateWorkVector{})); ok {
			t.Fatal("unfunded value accessor")
		}
	}
	for _, tx := range []*types.Transaction{nil} {
		if _, ok := nodeTransactionToObserved(tx, providerReplayUnitMeter()); ok {
			t.Fatal("missing recipient subject")
		}
		if _, ok := nodeTransactionValueObserved(tx, providerReplayUnitMeter()); ok {
			t.Fatal("missing value subject")
		}
	}
	tx := providerHistoryNodeJSONUnitTransactions()[0]
	if _, ok := nodeTransactionToObserved(tx, nil); ok {
		t.Fatal("missing original recipient observer")
	}
	if _, ok := nodeTransactionValueObserved(tx, nil); ok {
		t.Fatal("missing original value observer")
	}
}

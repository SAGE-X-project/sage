package registry011

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestProviderHistoryNodeSigningHashesMatchPinnedSupportedTypes(t *testing.T) {
	// The existing finite unsigned DATA values test only signing-preimage
	// encodings; they claim no account authorization or valid blob proof.
	for i, tx := range providerHistoryNodeJSONUnitTransactions() {
		signer := types.LatestSignerForChainID(big.NewInt(11155111))
		if tx.Type() == types.LegacyTxType && !tx.Protected() {
			signer = types.HomesteadSigner{}
		}
		want := signer.Hash(tx)
		m := providerReplayUnitMeter()
		got, err := nodeSigningHashObserved(tx, m)
		if err != nil || got != want || m.actual[2] != 0 || m.actual[10] == 0 {
			t.Fatal("pinned signing preimage", i, err, got, want, m.actual)
		}
		before := m.actual
		if got, err := nodeSigningHashObserved(tx, m); err != nil || got != want || m.actual[10] != 2*before[10] {
			t.Fatal("fresh repeated signing hash", i, err, m.actual)
		}
	}
	if _, err := nodeSigningHashObserved(nil, providerReplayUnitMeter()); err == nil {
		t.Fatal("missing typed transaction")
	}
	if _, err := nodeSigningHashObserved(providerHistoryNodeJSONUnitTransactions()[0], nil); err == nil {
		t.Fatal("missing original observer")
	}
	if _, err := nodeSigningHashObserved(providerHistoryNodeJSONUnitTransactions()[0], providerHistoryContentUnitMeter(candidateWorkVector{})); err == nil {
		t.Fatal("unfunded original encoder")
	}
}

func TestProviderHistoryNodeSenderScalarGuardsPrecedeNativeRecovery(t *testing.T) {
	makeTx := func(v, r, s *big.Int) *types.Transaction {
		return types.NewTx(&types.DynamicFeeTx{ChainID: big.NewInt(11155111), GasTipCap: big.NewInt(1), GasFeeCap: big.NewInt(2), Gas: 21000, Value: new(big.Int), V: v, R: r, S: s})
	}
	normal := makeTx(big.NewInt(0), big.NewInt(1), big.NewInt(1))
	want, err := types.Sender(types.LatestSignerForChainID(big.NewInt(11155111)), normal)
	if err != nil {
		t.Fatal("finite scalar recovery precursor", err)
	}
	if got, err := nodeSenderObserved(normal, providerReplayUnitMeter()); err != nil || got != want {
		t.Fatal("normal fresh precursor", err)
	}
	highS := new(big.Int).Add(new(big.Int).Rsh(new(big.Int).Set(crypto.S256().Params().N), 1), big.NewInt(1))
	for _, tc := range []struct {
		name    string
		v, r, s *big.Int
	}{
		{"parity", big.NewInt(2), big.NewInt(1), big.NewInt(1)},
		{"negative-v", big.NewInt(-1), big.NewInt(1), big.NewInt(1)},
		{"negative-r", big.NewInt(0), big.NewInt(-1), big.NewInt(1)},
		{"negative-s", big.NewInt(0), big.NewInt(1), big.NewInt(-1)},
		{"oversized-v", new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1), big.NewInt(1)},
		{"oversized-r", big.NewInt(0), new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1)},
		{"oversized-s", big.NewInt(0), big.NewInt(1), new(big.Int).Lsh(big.NewInt(1), 256)},
		{"high-s", big.NewInt(0), big.NewInt(1), highS},
		{"zero-r", big.NewInt(0), big.NewInt(0), big.NewInt(1)},
		{"zero-s", big.NewInt(0), big.NewInt(1), big.NewInt(0)},
		{"r-range", big.NewInt(0), new(big.Int).Set(crypto.S256().Params().N), big.NewInt(1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := providerReplayUnitMeter()
			if got, err := nodeSenderObserved(makeTx(tc.v, tc.r, tc.s), m); err == nil || got != (common.Address{}) || m.actual[2] != 0 {
				t.Fatal("invalid scalar entered recovery or returned account", err, m.actual)
			}
		})
	}
	for _, v := range []int64{0, 1} {
		legacy := types.NewTx(&types.LegacyTx{GasPrice: big.NewInt(1), Gas: 21000, Value: new(big.Int), V: big.NewInt(v), R: big.NewInt(1), S: big.NewInt(1)})
		if got, err := nodeSenderObserved(legacy, providerReplayUnitMeter()); err == nil || got != (common.Address{}) {
			t.Fatal("unprotected legacy needs27/28")
		}
	}
}

func TestProviderHistoryNodeSenderNativeFailureReturnsNoAccount(t *testing.T) {
	// Finite signature-decoder DATA only. Compare native recovery outcomes
	// with the pinned library; no key, authorization request or network exists.
	success, failure := false, false
	for r := int64(1); r <= 32; r++ {
		tx := types.NewTx(&types.DynamicFeeTx{ChainID: big.NewInt(11155111), GasTipCap: big.NewInt(1), GasFeeCap: big.NewInt(2), Gas: 21000, Value: new(big.Int), V: big.NewInt(0), R: big.NewInt(r), S: big.NewInt(1)})
		want, oldErr := types.Sender(types.LatestSignerForChainID(big.NewInt(11155111)), tx)
		m := providerReplayUnitMeter()
		got, err := nodeSenderObserved(tx, m)
		if (err != nil) != (oldErr != nil) || err == nil && got != want || err != nil && got != (common.Address{}) || m.actual[2] != 97 {
			t.Fatal("bounded native recovery oracle", r, err, oldErr, m.actual)
		}
		if err != nil {
			failure = true
		} else {
			success = true
		}
	}
	if !success || !failure {
		t.Fatal("finite positive and native-error precursors required")
	}
}

func TestProviderHistoryNodeSenderReconstructsRecordedAccountsWithoutCacheAuthority(t *testing.T) {
	_, block := providerHistoryNodeRecordedContent(t)
	signer := types.LatestSignerForChainID(big.NewInt(11155111))
	for i, tx := range block.txs {
		want, err := types.Sender(signer, tx) // Warm cache before the private reader.
		if err != nil {
			t.Fatal("recorded normal signed transaction", i, err)
		}
		legacy, err := nodeSenderObserved(tx, nil)
		if err != nil || legacy != want {
			t.Fatal("legacy sender entry", i, err)
		}
		m := providerReplayUnitMeter()
		got, err := nodeSenderObserved(tx, m)
		if err != nil || got != want || m.failed || m.actual[2] != 97 || m.actual[7] != 0 {
			t.Fatal("fresh observed account recovery", i, err, got, want, m.actual)
		}
		before := m.actual
		if got, err := nodeSenderObserved(tx, m); err != nil || got != want || m.actual[2] != 194 || m.actual[10] != 2*before[10] {
			t.Fatal("warm library cache hid repeat recovery", i, err, m.actual)
		}
	}
}

func TestProviderHistoryNodeSenderRefusesMissingOriginalQuotaAndInvalidInput(t *testing.T) {
	_, block := providerHistoryNodeRecordedContent(t)
	tx := block.txs[0]
	if _, err := nodeSenderObserved(tx, providerReplayUnitMeter()); err != nil {
		t.Fatal("normal signed precursor", err)
	}
	for _, upper := range []candidateWorkVector{{}, {2: 96, 10: 1 << 60}} {
		m := providerHistoryContentUnitMeter(upper)
		if got, err := nodeSenderObserved(tx, m); err == nil || got != (common.Address{}) || !m.failed {
			t.Fatal("original recovery allowance", err, got, m.actual)
		}
		if upper[2] == 96 && (m.actual[2] != 0 || m.actual[10] == 0) {
			t.Fatal("late original byte shortage discarded signing/encoding prefix")
		}
		before := m.actual
		if got, err := nodeSenderObserved(tx, m); err == nil || got != (common.Address{}) || m.actual != before {
			t.Fatal("failed original resumed")
		}
	}
	if _, err := nodeSenderObserved(nil, providerReplayUnitMeter()); err == nil {
		t.Fatal("missing transaction")
	}
	invalid := types.NewTx(&types.DynamicFeeTx{ChainID: big.NewInt(11155111), GasTipCap: big.NewInt(1), GasFeeCap: big.NewInt(2), Gas: 21000, Value: new(big.Int)})
	if got, err := nodeSenderObserved(invalid, providerReplayUnitMeter()); err == nil || got != (common.Address{}) {
		t.Fatal("missing signature")
	}
	otherChain := types.NewTx(&types.DynamicFeeTx{ChainID: big.NewInt(1), GasTipCap: big.NewInt(1), GasFeeCap: big.NewInt(2), Gas: 21000, Value: new(big.Int), V: big.NewInt(0), R: big.NewInt(1), S: big.NewInt(1)})
	if got, err := nodeSenderObserved(otherChain, providerReplayUnitMeter()); err == nil || got != (common.Address{}) {
		t.Fatal("different chain")
	}
	if _, err := nodeSigningHashObserved(otherChain, providerReplayUnitMeter()); err == nil {
		t.Fatal("different-chain signing preimage")
	}
}

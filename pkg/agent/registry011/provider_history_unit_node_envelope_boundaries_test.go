package registry011

import (
	"context"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

func TestProviderHistoryNativeHeaderRejectsNonHeaderChildren(t *testing.T) {
	// Finite local RLP decoder subjects, never an RPC response or accepted
	// consensus block. Neither a byte child nor an empty list is a header.
	for _, raw := range [][]byte{{0xc1, 0x01}, {0xc1, 0xc0}} {
		m := providerReplayUnitMeter()
		budget := replayBudget{limits: replayTestLimits(), observer: m}
		if out, err := providerHistoryNativeHeaderObserved(context.Background(), raw, &budget); err == nil || out != nil || m.actual[2] < uint64(len(raw)) {
			t.Fatal("non-header child escaped the original decoder", err, m.actual)
		}
	}
}

func TestProviderHistoryNativeHashRejectsEncodingErrorsWithoutPartialCommitments(t *testing.T) {
	// Negative native integers are local encoder-error DATA. They are never
	// signed, transmitted or used to claim account authorization.
	address := common.HexToAddress(replayTestWallet)
	tx := types.NewTx(&types.LegacyTx{GasPrice: big.NewInt(1), Gas: 21000, To: &address,
		Value: big.NewInt(-1), V: big.NewInt(2*11155111 + 35), R: big.NewInt(1), S: big.NewInt(1)})
	for _, mode := range []string{"transaction", "signing"} {
		t.Run(mode, func(t *testing.T) {
			m := providerReplayUnitMeter()
			var out common.Hash
			ok := false
			if mode == "transaction" {
				out, ok = nodeTransactionHashObserved(tx, m)
			} else {
				var err error
				out, err = nodeSigningHashObserved(tx, m)
				ok = err == nil
			}
			if ok || out != (common.Hash{}) || !m.failed || m.actual != (candidateWorkVector{}) || tx.Value().Sign() != -1 {
				t.Fatal("native error returned a commitment, hid refusal or changed the subject", m.actual)
			}
		})
	}
	other := types.NewTx(&types.LegacyTx{GasPrice: big.NewInt(1), Gas: 21000, Value: new(big.Int),
		V: big.NewInt(37), R: big.NewInt(1), S: big.NewInt(1)})
	if out, err := nodeSigningHashObserved(other, providerReplayUnitMeter()); err == nil || out != (common.Hash{}) {
		t.Fatal("another protected legacy chain supplied a Sepolia signing hash")
	}
}

func TestProviderHistoryHeaderHashIncludesItsFinalHashingBudget(t *testing.T) {
	input := replayTestInput(t).Blocks[0]
	var header types.Header
	if rlp.DecodeBytes(input.HeaderRLP, &header) != nil {
		t.Fatal("normal native header precursor")
	}
	// This is a native encoder ceiling subject, not a claim that a consensus
	// header with this extra-data length is valid. Its finite64KiB RLP stays
	// within the helper's own input envelope and cannot exceed a byte bound.
	header.Extra = make([]byte, 65536)
	var raw []byte
	for attempts := 0; attempts < 4; attempts++ {
		var err error
		raw, err = rlp.EncodeToBytes(&header)
		if err != nil {
			t.Fatal("finite header encoding precursor", err)
		}
		if len(raw) == 65536 {
			break
		}
		next := len(header.Extra) + 65536 - len(raw)
		if next < 0 || next > 65536 {
			t.Fatal("header ceiling fixture exceeded its finite input bound", next)
		}
		header.Extra = make([]byte, next)
	}
	if len(raw) != 65536 {
		t.Fatal("exact native RLP ceiling precursor", len(raw))
	}
	before := types.CopyHeader(&header)
	const encoded = uint64(65536*65536 + 1)
	m := providerReplayUnitMeter()
	m.upper[10] = encoded
	if out, ok := nodeHeaderHashObserved(&header, m); ok || out != (common.Hash{}) || !m.failed ||
		m.actual != (candidateWorkVector{10: encoded}) || !reflect.DeepEqual(&header, before) {
		t.Fatal("encoding alone was treated as a completed header hash", m.actual)
	}
	prefix := m.actual
	if out, ok := nodeHeaderHashObserved(&header, m); ok || out != (common.Hash{}) || m.actual != prefix {
		t.Fatal("refused native hash resumed its original observer")
	}
	full := providerReplayUnitMeter()
	full.upper[10] = encoded + 65537
	if out, ok := nodeHeaderHashObserved(&header, full); !ok || out != header.Hash() || full.failed ||
		full.actual != (candidateWorkVector{10: encoded + 65537}) {
		t.Fatal("exact encode-and-hash budget did not preserve the native commitment", full.actual)
	}
}

func TestProviderHistoryDerivedGasFloorRefusesBeforeLaterArithmetic(t *testing.T) {
	address := common.HexToAddress(replayTestWallet)
	tx := types.NewTx(&types.LegacyTx{GasPrice: big.NewInt(1), Gas: 21000, To: &address, Value: big.NewInt(1)})
	for _, tc := range []struct {
		fuel, prefix uint64
	}{
		{21, 21}, // Recipient copy21, then destination comparison41 cannot enter.
		{62, 62}, // Both completed; the original value-copy preflight cannot enter.
	} {
		m := providerReplayUnitMeter()
		m.upper[10] = tc.fuel
		if out, err := replayGasFloorDerivedObserved(tx, common.Address{}, m); err == nil || out != 0 || !m.failed ||
			m.actual != (candidateWorkVector{10: tc.prefix}) {
			t.Fatal("unfunded derived gas arithmetic returned a partial floor", err, m.actual)
		}
		before := m.actual
		if out, err := replayGasFloorDerivedObserved(tx, common.Address{}, m); err == nil || out != 0 || m.actual != before {
			t.Fatal("refused gas-floor observer resumed")
		}
	}
}

func TestProviderHistoryExplicitJSONExtraStillRequiresItsComparisonBudget(t *testing.T) {
	m := providerReplayUnitMeter()
	m.upper[10] = 0
	if nodeJSONExtraObserved("displayName", "displayName", m) || !m.failed || m.actual != (candidateWorkVector{}) {
		t.Fatal("declared extension bypassed its original comparison budget")
	}
	if !nodeJSONExtraObserved("displayName", "displayName", providerReplayUnitMeter()) {
		t.Fatal("normal explicit extension comparison")
	}
}

func TestProviderHistoryUnprotectedLegacySenderMatchesItsOriginalSigner(t *testing.T) {
	for _, parity := range []int64{27, 28} {
		tx := types.NewTx(&types.LegacyTx{GasPrice: big.NewInt(1), Gas: 21000, Value: new(big.Int),
			V: big.NewInt(parity), R: big.NewInt(1), S: big.NewInt(1)})
		want, err := types.Sender(types.HomesteadSigner{}, tx)
		if err != nil {
			t.Fatal("finite legacy scalar recovery precursor", err)
		}
		m := providerReplayUnitMeter()
		if out, err := nodeSenderObserved(tx, m); err != nil || out != want || m.failed || m.actual[2] != 97 {
			t.Fatal("unprotected legacy recovery changed its signer or original observation", parity, err, m.actual)
		}
	}
}

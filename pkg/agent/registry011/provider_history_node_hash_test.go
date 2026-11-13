package registry011

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

func TestProviderHistoryNodeHashObservationUsesFreshCanonicalBytes(t *testing.T) {
	raw := replayTestInput(t).Blocks[0]
	var header types.Header
	var tx types.Transaction
	if rlp.DecodeBytes(raw.HeaderRLP, &header) != nil || tx.UnmarshalBinary(raw.Transactions[0]) != nil {
		t.Fatal("normal canonical precursor")
	}
	wantHeader, wantTx := header.Hash(), tx.Hash() // Warm library caches are not authority.
	m := providerReplayUnitMeter()
	h, ok := nodeHeaderHashObserved(&header, m)
	x, valid := nodeTransactionHashObserved(&tx, m)
	if !ok || !valid || h != wantHeader || x != wantTx || m.actual[2] != 0 {
		t.Fatal("canonical header/transaction commitment", h, x, ok, valid)
	}
	before := m.actual
	if h, ok := nodeHeaderHashObserved(&header, m); !ok || h != wantHeader {
		t.Fatal("repeat header reconstruction")
	}
	if x, ok := nodeTransactionHashObserved(&tx, m); !ok || x != wantTx || m.actual[10] != 2*before[10] {
		t.Fatal("library hash cache hid the actual second encode/hash", m.actual, before)
	}
	if h, ok := nodeHeaderHashObserved(&header, nil); !ok || h != wantHeader {
		t.Fatal("legacy header hash")
	}
	if x, ok := nodeTransactionHashObserved(&tx, nil); !ok || x != wantTx {
		t.Fatal("legacy transaction hash")
	}
	if text, ok := nodeHashTextObserved(wantHeader, m); !ok || text != wantHeader.Hex() {
		t.Fatal("hash hex representation")
	}
	address := common.HexToAddress(SepoliaRegistryAddress)
	if text, ok := nodeAddressTextObserved(address, m); !ok || text != replayAddress(address) {
		t.Fatal("lower-case address representation")
	}
	if text, ok := nodeHashTextObserved(wantHeader, nil); !ok || text != wantHeader.Hex() {
		t.Fatal("legacy hash representation")
	}
	if text, ok := nodeAddressTextObserved(address, nil); !ok || text != replayAddress(address) {
		t.Fatal("legacy address representation")
	}
}

func TestProviderHistoryNodeTransactionHashExcludesNetworkBlobSidecar(t *testing.T) {
	// These finite unsigned envelopes are hash-codec subjects only. No blob
	// proof is asserted valid, recovered as an account or sent to a service.
	for _, tc := range []struct {
		name    string
		sidecar *types.BlobTxSidecar
	}{
		{"empty", &types.BlobTxSidecar{}},
		{"one-blob", &types.BlobTxSidecar{Blobs: make([]kzg4844.Blob, 1), Commitments: make([]kzg4844.Commitment, 1), Proofs: make([]kzg4844.Proof, 1)}},
	} {
		for _, warm := range []bool{false, true} {
			name := "cold"
			if warm {
				name = "warm"
			}
			t.Run(tc.name+"/"+name, func(t *testing.T) {
				tx := types.NewTx(&types.BlobTx{
					ChainID: uint256.NewInt(11155111), Nonce: 1, Gas: 21000,
					GasTipCap: uint256.NewInt(1), GasFeeCap: uint256.NewInt(2),
					To: common.HexToAddress(replayTestWallet), Value: new(uint256.Int),
					BlobFeeCap: uint256.NewInt(1), BlobHashes: []common.Hash{{1}},
					V: new(uint256.Int), R: uint256.NewInt(1), S: uint256.NewInt(1),
					Sidecar: tc.sidecar,
				})
				before, err := tx.MarshalBinary()
				if err != nil || tx.BlobTxSidecar() == nil {
					t.Fatal("normal sidecar serialization precursor", err)
				}
				if warm && tx.Size() == 0 {
					t.Fatal("warm-size precursor")
				}
				want := tx.Hash()
				m := providerReplayUnitMeter()
				got, ok := nodeTransactionHashObserved(tx, m)
				after, err := tx.MarshalBinary()
				if !ok || got != want || err != nil || !bytes.Equal(before, after) || tx.BlobTxSidecar() == nil || m.actual[2] != 0 || len(tx.BlobTxSidecar().Blobs) != len(tc.sidecar.Blobs) {
					t.Fatal("sidecar entered the hash or its original bytes changed", got, want, err)
				}
				// The warm nonempty case traverses sidecar
				// length metadata in WithoutBlobTxSidecar. Those scalar loops
				// still require their separate complete native-cost treatment.
			})
		}
	}
}

func TestProviderHistoryNodeCreateAddressObservationMatchesNonceBoundaries(t *testing.T) {
	address := common.HexToAddress(replayTestWallet)
	for _, nonce := range []uint64{0, 1, 127, 128, 255, 256, ^uint64(0)} {
		m := providerReplayUnitMeter()
		got, ok := nodeCreateAddressObserved(address, nonce, m)
		legacy, oldOK := nodeCreateAddressObserved(address, nonce, nil)
		want := crypto.CreateAddress(address, nonce)
		if !ok || !oldOK || got != want || legacy != want || m.actual[2] != 0 || m.actual[10] == 0 {
			t.Fatal("RLP sender/nonce or final20byte selection", nonce, got, want)
		}
	}
}

func TestProviderHistoryNodeHashRefusalKeepsOriginalMeterFailed(t *testing.T) {
	raw := replayTestInput(t).Blocks[0]
	var header types.Header
	var tx types.Transaction
	if rlp.DecodeBytes(raw.HeaderRLP, &header) != nil || tx.UnmarshalBinary(raw.Transactions[0]) != nil {
		t.Fatal("normal precursor")
	}
	for _, mode := range []string{"header", "transaction", "create", "hash-text", "address-text"} {
		m := providerHistoryContentUnitMeter(candidateWorkVector{})
		ok := false
		switch mode {
		case "header":
			_, ok = nodeHeaderHashObserved(&header, m)
		case "transaction":
			_, ok = nodeTransactionHashObserved(&tx, m)
		case "create":
			_, ok = nodeCreateAddressObserved(common.Address{}, 0, m)
		case "hash-text":
			_, ok = nodeHashTextObserved(common.Hash{}, m)
		case "address-text":
			_, ok = nodeAddressTextObserved(common.Address{}, m)
		}
		if ok || !m.failed || m.actual != (candidateWorkVector{}) {
			t.Fatal("unfunded hash primitive ran or supplied a result", mode, m.actual)
		}
	}
	if _, ok := nodeHeaderHashObserved(nil, providerReplayUnitMeter()); ok {
		t.Fatal("missing header")
	}
	if _, ok := nodeTransactionHashObserved(nil, providerReplayUnitMeter()); ok {
		t.Fatal("missing transaction")
	}
	invalid := header
	invalid.Difficulty = big.NewInt(-1)
	m := providerReplayUnitMeter()
	if _, ok := nodeHeaderHashObserved(&invalid, m); ok || !m.failed {
		t.Fatal("encoder failure retained a reusable original meter")
	}
}

package registry011

import (
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestNodeCreationIdentityChecks(t *testing.T) {
	identity, contract, err := nodeIdentityManifest(nodeIdentityJSON, nodeABIJSON)
	if err != nil {
		t.Fatal(err)
	}
	key, err := crypto.HexToECDSA(strings.Repeat("0", 63) + "1")
	if err != nil {
		t.Fatal(err)
	}
	sender := crypto.PubkeyToAddress(key.PublicKey)
	identity.Deployer = replayAddress(sender)
	identity.Registry = replayAddress(crypto.CreateAddress(sender, 8))
	args := []any{}
	for _, c := range identity.Contracts[:8] {
		args = append(args, common.HexToAddress(c.Address))
	}
	encoded, err := contract.Constructor.Inputs.Pack(args...)
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"normal", "missing", "nonce", "value", "to", "chain", "legacy", "failed", "sender", "created-address", "short-arguments", "arguments-highbits", "dependency"} {
		t.Run(mode, func(t *testing.T) {
			id := replayTestClone(t, identity)
			data := append([]byte{0x60}, encoded...)
			nonce := uint64(8)
			chain := big.NewInt(11155111)
			value := new(big.Int)
			var to *common.Address
			switch mode {
			case "nonce":
				nonce = 9
			case "value":
				value = big.NewInt(1)
			case "to":
				a := common.HexToAddress(replayTestWallet)
				to = &a
			case "chain":
				chain = big.NewInt(1)
			case "sender":
				id.Deployer = replayTestWallet
			case "created-address":
				id.Registry = replayTestWallet
			case "short-arguments":
				data = []byte{1}
			case "arguments-highbits":
				data[1] = 1
			case "dependency":
				data[32] ^= 1
			}
			var tx *types.Transaction
			if mode == "legacy" {
				tx = types.NewTx(&types.LegacyTx{Nonce: nonce, Gas: 1000000, GasPrice: big.NewInt(1), Value: value, Data: data})
			} else {
				tx = types.NewTx(&types.DynamicFeeTx{ChainID: chain, Nonce: nonce, Gas: 1000000, GasFeeCap: big.NewInt(2), GasTipCap: big.NewInt(1), Value: value, To: to, Data: data})
			}
			tx, err = types.SignTx(tx, types.LatestSignerForChainID(chain), key)
			if err != nil {
				t.Fatal(err)
			}
			id.Deployment.Hash = tx.Hash().Hex()
			if mode == "missing" {
				id.Deployment.Hash = testBlock
			}
			receipt := &types.Receipt{Status: 1}
			if mode == "failed" {
				receipt.Status = 0
			}
			b := replayDecodedBlock{txs: types.Transactions{tx}, receipts: types.Receipts{receipt}}
			i, e := nodeCreation(id, contract, b)
			if mode == "normal" {
				if e != nil || i != 0 {
					t.Fatal(i, e)
				}
			} else if e == nil {
				t.Fatal("accepted", mode)
			}
		})
	}
}

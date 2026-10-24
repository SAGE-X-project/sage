package registry011

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

const sepoliaDeploymentBlock uint64 = 11974670

//go:embed node_identity.json
var nodeIdentityJSON []byte

//go:embed node_deployment_abi.json
var nodeABIJSON []byte

type nodeCodeIdentity struct {
	Name, Address string
	SHA256        string `json:"code_sha256"`
}
type nodeIdentity struct {
	Registry   string `json:"registry_address"`
	Contracts  []nodeCodeIdentity
	Deployer   string
	Deployment struct {
		Hash        string
		BlockHash   string `json:"block_hash"`
		BlockNumber uint64 `json:"block_number"`
	}
}

// DeploymentEvidence establishes only the thirteen exact runtime identities at
// Anchor and one signed Registry creation at Deployment. It does not re-prove
// all nine historical creation transactions, consensus, full Registry history,
// live mutation log behavior, custody or Source readiness. Exchanges retain all
// successful raw response envelopes for independent audit; no boolean promotes
// this bounded observation into a Source credential.
type DeploymentEvidence struct {
	ClientVersion      string
	Anchor, Deployment FinalizedHeader
	ABIHash            string
	CodeHashes         map[string]string
	Creation           ReplayExecution
	Block              ReplayBlock
	Exchanges          []NodeExchange
}

func nodeIdentityManifest(rawIdentity, rawABI []byte) (nodeIdentity, abi.ABI, error) {
	var identity nodeIdentity
	var parsed abi.ABI
	if json.Unmarshal(rawIdentity, &identity) != nil || len(identity.Contracts) != 13 || identity.Registry != SepoliaRegistryAddress || identity.Deployment.BlockNumber != sepoliaDeploymentBlock || hex.EncodeToString(sha256Digest(rawABI)) != "7000f92bdcd26d84c63d2b71a12c0db12c61e9909b7f4bedb76774381339678e" {
		return identity, parsed, ErrConfiguration
	}
	parsed, err := abi.JSON(bytes.NewReader(rawABI))
	if err != nil {
		return identity, parsed, ErrConfiguration
	}
	return identity, parsed, nil
}

func sha256Digest(b []byte) []byte { h := sha256.Sum256(b); return h[:] }

// ObserveDeployment uses the already selected official RPC and block hash; all
// runtime reads use EIP-1898 requireCanonical. No new transaction is submitted.
// Supplied budgets describe this acquisition, not an approved operating policy.
func (r *PinnedAgentReads) ObserveDeployment(ctx context.Context, limits NodeLimits) (*DeploymentEvidence, error) {
	a, cancel, err := r.nodeStart(ctx, limits)
	if err != nil {
		return nil, err
	}
	defer cancel()
	if err := a.network(); err != nil {
		return nil, err
	}
	identity, contract, err := nodeIdentityManifest(nodeIdentityJSON, nodeABIJSON)
	if err != nil {
		return nil, err
	}
	out := &DeploymentEvidence{Anchor: r.header, ABIHash: "7000f92bdcd26d84c63d2b71a12c0db12c61e9909b7f4bedb76774381339678e", CodeHashes: map[string]string{}}
	for _, expected := range identity.Contracts {
		result, err := a.call("eth_getCode", []any{expected.Address, r.block()})
		if err != nil {
			return nil, err
		}
		code, ok := nodeData(result)
		if !ok || len(code) == 0 || hex.EncodeToString(sha256Digest(code)) != expected.SHA256 {
			return nil, ErrRPCState
		}
		out.CodeHashes[expected.Address] = expected.SHA256
	}
	e, b, err := a.block(sepoliaDeploymentBlock)
	if err != nil {
		return nil, err
	}
	if e.Header.Hash != identity.Deployment.BlockHash {
		return nil, ErrAncestry
	}
	index, err := nodeCreation(identity, contract, b)
	if err != nil {
		return nil, err
	}
	creation, err := a.execution(b, index)
	if err != nil {
		return nil, err
	}
	if err := a.recheck(e.Header); err != nil {
		return nil, err
	}
	out.Deployment = e.Header
	out.Creation = creation
	out.Block = e.Block
	out.Exchanges = a.exchanges
	out.ClientVersion = a.clientVersion
	return out, nil
}

func nodeCreation(identity nodeIdentity, contract abi.ABI, b replayDecodedBlock) (int, error) {
	index := -1
	for i, tx := range b.txs {
		if strings.EqualFold(tx.Hash().Hex(), identity.Deployment.Hash) {
			index = i
			break
		}
	}
	if index < 0 {
		return -1, ErrReplayUnavailable
	}
	tx := b.txs[index]
	if tx.To() != nil || tx.Nonce() != 8 || tx.Value().Sign() != 0 || tx.Type() != types.DynamicFeeTxType || tx.ChainId().Cmp(big.NewInt(11155111)) != 0 || b.receipts[index].Status != 1 {
		return -1, ErrReplay
	}
	sender, err := types.Sender(types.LatestSignerForChainID(big.NewInt(11155111)), tx)
	if err != nil || replayAddress(sender) != identity.Deployer || replayAddress(crypto.CreateAddress(sender, tx.Nonce())) != identity.Registry {
		return -1, ErrReplay
	}
	// The pinned transaction hash commits to all initcode, not just this suffix.
	// Compare all eight constructor dependencies to the independently pinned ABI.
	if len(tx.Data()) < 8*32 {
		return -1, ErrReplay
	}
	args := tx.Data()[len(tx.Data())-8*32:]
	values, err := contract.Constructor.Inputs.Unpack(args)
	if err != nil || len(values) != 8 {
		return -1, ErrReplay
	}
	encoded, err := contract.Constructor.Inputs.Pack(values...)
	if err != nil || !bytes.Equal(encoded, args) {
		return -1, ErrReplay
	}
	for i := 0; i < 8; i++ {
		if !strings.EqualFold(nodeHex(args[i*32+12:(i+1)*32]), identity.Contracts[i].Address) {
			return -1, ErrReplay
		}
	}

	return index, nil
}

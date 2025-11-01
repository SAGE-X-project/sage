// SAGE - Secure Agent Guarantee Engine
// Copyright (C) 2025 SAGE-X-project
//
// This file is part of SAGE.
//
// SAGE is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// SAGE is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with SAGE. If not, see <https://www.gnu.org/licenses/>.

package ethereum

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	sagecrypto "github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/did"
)

// EthereumClient implements DID registry operations for Ethereum V2 contracts.
//
// DEPRECATED: This client is for legacy V2 contracts only and is no longer actively maintained.
// V2 contracts have incompatible signature verification with the current architecture.
// For new deployments, use EthereumClientV4 (clientv4.go) which supports:
//   - Multi-key management (ECDSA + Ed25519)
//   - Compatible signature verification
//   - Update functionality
//   - Better security features
//
// Migration: Replace NewEthereumClient() calls with NewEthereumClientV4().
type EthereumClient struct {
	client          *ethclient.Client
	contract        *bind.BoundContract
	contractABI     abi.ABI
	contractAddress common.Address
	privateKey      *ecdsa.PrivateKey
	chainID         *big.Int
	config          *did.RegistryConfig
}

// init registers the Ethereum client creator with the factory
func init() {
	did.RegisterEthereumClientCreator(func(config *did.RegistryConfig) (did.Client, error) {
		return NewEthereumClient(config)
	})
}

// NewEthereumClient creates a new Ethereum DID client for V2 contracts.
//
// DEPRECATED: Use NewEthereumClientV4() for new deployments.
// V2 contracts are no longer maintained and have compatibility issues.
func NewEthereumClient(config *did.RegistryConfig) (*EthereumClient, error) {
	client, err := ethclient.Dial(config.RPCEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum node: %w", err)
	}

	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get network ID: %w", err)
	}

	var privateKey *ecdsa.PrivateKey
	if config.PrivateKey != "" {
		privateKey, err = crypto.HexToECDSA(config.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("invalid private key: %w", err)
		}
	}

	contractAddress := common.HexToAddress(config.ContractAddress)

	// Parse the contract ABI
	contractABI, err := abi.JSON(strings.NewReader(AgentCardRegistryABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse contract ABI: %w", err)
	}

	contract := bind.NewBoundContract(contractAddress, contractABI, client, client, client)

	return &EthereumClient{
		client:          client,
		contract:        contract,
		contractABI:     contractABI,
		contractAddress: contractAddress,
		privateKey:      privateKey,
		chainID:         chainID,
		config:          config,
	}, nil
}

// Register registers a new agent on Ethereum
func (c *EthereumClient) Register(ctx context.Context, req *did.RegistrationRequest) (*did.RegistrationResult, error) {
	// Validate key type first (before checking client initialization)
	if req.KeyPair.Type() != sagecrypto.KeyTypeSecp256k1 {
		return nil, fmt.Errorf("ethereum requires Secp256k1 keys")
	}

	// Validate client is initialized
	if c.contract == nil {
		return nil, fmt.Errorf("ethereum client not properly initialized: contract is nil")
	}

	// Get the Ethereum address directly from public key (no provider dependency needed)
	ecdsaPubKey, ok := req.KeyPair.PublicKey().(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("invalid public key type for Ethereum, expected *ecdsa.PublicKey")
	}
	ethAddress := crypto.PubkeyToAddress(*ecdsaPubKey)
	addressValue := ethAddress.Hex()

	// Prepare the message to sign
	message := c.prepareRegistrationMessage(req, addressValue)
	messageHash := crypto.Keccak256([]byte(message))

	// Sign the message
	signature, err := req.KeyPair.Sign(messageHash)
	if err != nil {
		return nil, fmt.Errorf("failed to sign registration: %w", err)
	}

	// Prepare capabilities as JSON string
	capabilitiesJSON, err := json.Marshal(req.Capabilities)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal capabilities: %w", err)
	}

	// Get public key bytes
	publicKeyBytes, err := did.MarshalPublicKey(req.KeyPair.PublicKey())
	if err != nil {
		return nil, err
	}

	// V2 contract requires 65-byte uncompressed format (0x04 prefix + 64 bytes)
	// MarshalPublicKey returns 64 bytes for secp256k1, so we need to add the prefix for V2
	// Note: This is V2-specific. V4 (clientv4.go) accepts both 64 and 65 byte formats
	if len(publicKeyBytes) == 64 {
		prefixedKey := make([]byte, 65)
		prefixedKey[0] = 0x04 // uncompressed key prefix
		copy(prefixedKey[1:], publicKeyBytes)
		publicKeyBytes = prefixedKey
	}

	// Prepare transaction options
	auth, err := c.getTransactOpts(ctx)
	if err != nil {
		return nil, err
	}

	// Call the contract
	tx, err := c.contract.Transact(auth, "registerAgent",
		string(req.DID),
		req.Name,
		req.Description,
		req.Endpoint,
		publicKeyBytes,
		string(capabilitiesJSON),
		signature,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to register agent: %w", err)
	}

	// Wait for transaction confirmation
	receipt, err := c.waitForTransaction(ctx, tx)
	if err != nil {
		return nil, err
	}

	return &did.RegistrationResult{
		TransactionHash: tx.Hash().Hex(),
		BlockNumber:     receipt.BlockNumber.Uint64(),
		Timestamp:       time.Now(),
		GasUsed:         receipt.GasUsed,
	}, nil
}

// Resolve retrieves agent metadata from Ethereum
// Resolve retrieves agent metadata (ECDSA pub via keyHashes→getKey, KEM X25519 via kemPublicKey).
func (c *EthereumClient) Resolve(ctx context.Context, agentDID did.AgentDID) (*did.AgentMetadata, error) {
	// --- 로컬 on-chain 호환 구조체 (컨트랙트와 동일 순서/이름) ---
	type agentMetadataLocal struct {
		Did          string         `abi:"did"`
		Name         string         `abi:"name"`
		Description  string         `abi:"description"`
		Endpoint     string         `abi:"endpoint"`
		KeyHashes    [][32]byte     `abi:"keyHashes"` // bytes32[]
		Capabilities string         `abi:"capabilities"`
		Owner        common.Address `abi:"owner"`
		RegisteredAt *big.Int       `abi:"registeredAt"`
		UpdatedAt    *big.Int       `abi:"updatedAt"`
		Active       bool           `abi:"active"`
		ChainID      *big.Int       `abi:"chainId"`
		KEMPublicKey []byte         `abi:"kemPublicKey"` // raw 32B X25519
	}
	type agentKeyLocal struct {
		KeyType      uint8    `abi:"keyType"` // 0=ECDSA, 1=Ed25519, 2=X25519
		KeyData      []byte   `abi:"keyData"`
		Signature    []byte   `abi:"signature"`
		Verified     bool     `abi:"verified"`
		RegisteredAt *big.Int `abi:"registeredAt"`
	}

	// helper: []byte(플랫) → [][32]byte 변환
	chunk32s := func(b []byte) ([][32]byte, error) {
		if len(b)%32 != 0 {
			return nil, fmt.Errorf("flat keyHashes length=%d not multiple of 32", len(b))
		}
		out := make([][32]byte, len(b)/32)
		for i := range out {
			copy(out[i][:], b[i*32:(i+1)*32])
		}
		return out, nil
	}

	// helper: getKey 결과 해석 (struct or flat)
	coerceKey := func(vals []interface{}, out *agentKeyLocal) error {
		if len(vals) == 1 && reflect.ValueOf(vals[0]).Kind() == reflect.Struct {
			v := reflect.ValueOf(vals[0])
			out.KeyType = v.Field(0).Interface().(uint8)
			out.KeyData = v.Field(1).Interface().([]byte)
			out.Signature = v.Field(2).Interface().([]byte)
			out.Verified = v.Field(3).Interface().(bool)
			out.RegisteredAt = v.Field(4).Interface().(*big.Int)
			return nil
		}
		if len(vals) != 5 {
			return fmt.Errorf("unexpected getKey outputs len=%d", len(vals))
		}
		out.KeyType = vals[0].(uint8)
		out.KeyData = vals[1].([]byte)
		out.Signature = vals[2].([]byte)
		out.Verified = vals[3].(bool)
		out.RegisteredAt = vals[4].(*big.Int)
		return nil
	}

	// helper: getAgentByDID 결과 해석 (struct or flat), keyHashes 타입 보정까지
	coerceAgent := func(vals []interface{}, out *agentMetadataLocal) error {
		// struct 한 덩이로 오는 경우
		if len(vals) == 1 && reflect.ValueOf(vals[0]).Kind() == reflect.Struct {
			v := reflect.ValueOf(vals[0])
			out.Did = v.Field(0).Interface().(string)
			out.Name = v.Field(1).Interface().(string)
			out.Description = v.Field(2).Interface().(string)
			out.Endpoint = v.Field(3).Interface().(string)

			kh := v.Field(4).Interface()
			switch t := kh.(type) {
			case [][32]byte:
				out.KeyHashes = t
			case [][]uint8:
				out.KeyHashes = make([][32]byte, len(t))
				for i := range t {
					if len(t[i]) != 32 {
						return fmt.Errorf("keyHash[%d] len=%d", i, len(t[i]))
					}
					copy(out.KeyHashes[i][:], t[i])
				}
			case []byte:
				conv, err := chunk32s(t)
				if err != nil {
					return err
				}
				out.KeyHashes = conv
			default:
				return fmt.Errorf("unsupported keyHashes type %T", kh)
			}

			out.Capabilities = v.Field(5).Interface().(string)
			out.Owner = v.Field(6).Interface().(common.Address)
			out.RegisteredAt = v.Field(7).Interface().(*big.Int)
			out.UpdatedAt = v.Field(8).Interface().(*big.Int)
			out.Active = v.Field(9).Interface().(bool)
			out.ChainID = v.Field(10).Interface().(*big.Int)
			out.KEMPublicKey = v.Field(11).Interface().([]byte)
			return nil
		}

		// 평탄화(12개)로 오는 경우
		if len(vals) != 12 {
			return fmt.Errorf("unexpected getAgentByDID outputs len=%d", len(vals))
		}
		out.Did = vals[0].(string)
		out.Name = vals[1].(string)
		out.Description = vals[2].(string)
		out.Endpoint = vals[3].(string)
		switch kh := vals[4].(type) {
		case [][32]byte:
			out.KeyHashes = kh
		case [][]uint8:
			out.KeyHashes = make([][32]byte, len(kh))
			for i := range kh {
				if len(kh[i]) != 32 {
					return fmt.Errorf("keyHash[%d] len=%d", i, len(kh[i]))
				}
				copy(out.KeyHashes[i][:], kh[i])
			}
		case []byte:
			conv, err := chunk32s(kh)
			if err != nil {
				return err
			}
			out.KeyHashes = conv
		default:
			return fmt.Errorf("unsupported keyHashes type %T", vals[4])
		}
		out.Capabilities = vals[5].(string)
		out.Owner = vals[6].(common.Address)
		out.RegisteredAt = vals[7].(*big.Int)
		out.UpdatedAt = vals[8].(*big.Int)
		out.Active = vals[9].(bool)
		out.ChainID = vals[10].(*big.Int)
		out.KEMPublicKey = vals[11].([]byte)
		return nil
	}

	// 1) pack call
	callData, err := c.contractABI.Pack("getAgentByDID", string(agentDID))
	if err != nil {
		return nil, fmt.Errorf("pack getAgentByDID: %w", err)
	}

	// 2) call
	output, err := c.client.CallContract(ctx, ethereum.CallMsg{
		To:   &c.contractAddress,
		Data: callData,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("call getAgentByDID: %w", err)
	}

	// 3) unpack → coerce
	vals, err := c.contractABI.Unpack("getAgentByDID", output) // []interface{}, error  (go-ethereum v1.16.5)
	if err != nil {
		// 이 에러가 "into string"이면 ABI 스펙 불일치(여전히 구 ABI)임
		return nil, fmt.Errorf("unpack getAgentByDID: %w", err)
	}
	var on agentMetadataLocal
	if err := coerceAgent(vals, &on); err != nil {
		return nil, fmt.Errorf("coerce getAgentByDID: %w", err)
	}

	// 4) 존재 확인
	if on.Did == "" || on.Owner == (common.Address{}) {
		return nil, did.ErrDIDNotFound
	}

	// 5) capabilities JSON → map
	var caps map[string]interface{}
	if s := strings.TrimSpace(on.Capabilities); s != "" {
		if err := json.Unmarshal([]byte(s), &caps); err != nil {
			return nil, fmt.Errorf("unmarshal capabilities: %w", err)
		}
	}

	// 6) keyHashes → getKey(bytes32)로 ECDSA 키 하나 찾아 파싱
	var publicKey interface{} // secp256k1 (65B uncompressed)
	for _, kh := range on.KeyHashes {
		cdKey, err := c.contractABI.Pack("getKey", kh)
		if err != nil {
			return nil, fmt.Errorf("pack getKey: %w", err)
		}
		outKey, err := c.client.CallContract(ctx, ethereum.CallMsg{
			To:   &c.contractAddress,
			Data: cdKey,
		}, nil)
		if err != nil {
			return nil, fmt.Errorf("call getKey: %w", err)
		}
		kvals, err := c.contractABI.Unpack("getKey", outKey)
		if err != nil {
			return nil, fmt.Errorf("unpack getKey: %w", err)
		}
		var k agentKeyLocal
		if err := coerceKey(kvals, &k); err != nil {
			return nil, fmt.Errorf("coerce getKey: %w", err)
		}
		if k.KeyType == 0 { // ECDSA
			pk, err := did.UnmarshalPublicKey(k.KeyData, "secp256k1")
			if err != nil {
				return nil, fmt.Errorf("unmarshal ECDSA pubkey: %w", err)
			}
			publicKey = pk
			break
		}
	}

	// 7) 결과 구성 (kemPublicKey는 원시 32바이트 그대로)
	return &did.AgentMetadata{
		DID:          agentDID,
		Name:         on.Name,
		Description:  on.Description,
		Endpoint:     on.Endpoint,
		PublicKey:    publicKey, // 없을 수 있음
		Capabilities: caps,
		Owner:        on.Owner.Hex(),
		IsActive:     on.Active,
		PublicKEMKey: on.KEMPublicKey, // raw X25519 (32B)
		CreatedAt:    time.Unix(on.RegisteredAt.Int64(), 0),
		UpdatedAt:    time.Unix(on.UpdatedAt.Int64(), 0),
	}, nil
}

// Update updates agent metadata on Ethereum
func (c *EthereumClient) Update(ctx context.Context, agentDID did.AgentDID, updates map[string]interface{}, keyPair sagecrypto.KeyPair) error {
	// Prepare update message
	message := c.prepareUpdateMessage(agentDID, updates)
	messageHash := crypto.Keccak256([]byte(message))

	// Sign the message
	signature, err := keyPair.Sign(messageHash)
	if err != nil {
		return fmt.Errorf("failed to sign update: %w", err)
	}

	// Prepare transaction options
	auth, err := c.getTransactOpts(ctx)
	if err != nil {
		return err
	}

	// Extract update fields
	name, _ := updates["name"].(string)
	description, _ := updates["description"].(string)
	endpoint, _ := updates["endpoint"].(string)

	capabilitiesJSON := ""
	if capabilities, ok := updates["capabilities"]; ok {
		capBytes, err := json.Marshal(capabilities)
		if err != nil {
			return fmt.Errorf("failed to marshal capabilities: %w", err)
		}
		capabilitiesJSON = string(capBytes)
	}

	// Generate agentId from DID (keccak256 hash)
	agentId := crypto.Keccak256Hash([]byte(string(agentDID)))

	// Call the contract
	tx, err := c.contract.Transact(auth, "updateAgent",
		agentId,
		name,
		description,
		endpoint,
		capabilitiesJSON,
		signature,
	)
	if err != nil {
		return fmt.Errorf("failed to update agent: %w", err)
	}

	// Wait for confirmation
	_, err = c.waitForTransaction(ctx, tx)
	return err
}

// Deactivate deactivates an agent on Ethereum
func (c *EthereumClient) Deactivate(ctx context.Context, agentDID did.AgentDID, keyPair sagecrypto.KeyPair) error {
	// Note: The new contract's deactivateAgent doesn't require a signature
	// Prepare transaction options
	auth, err := c.getTransactOpts(ctx)
	if err != nil {
		return err
	}

	// Generate agentId from DID (keccak256 hash)
	agentId := crypto.Keccak256Hash([]byte(string(agentDID)))

	// Call the contract
	tx, err := c.contract.Transact(auth, "deactivateAgent", agentId)
	if err != nil {
		return fmt.Errorf("failed to deactivate agent: %w", err)
	}

	// Wait for confirmation
	_, err = c.waitForTransaction(ctx, tx)
	return err
}

// Helper methods

func (c *EthereumClient) getTransactOpts(ctx context.Context) (*bind.TransactOpts, error) {
	if c.privateKey == nil {
		return nil, fmt.Errorf("private key required for transactions")
	}

	auth, err := bind.NewKeyedTransactorWithChainID(c.privateKey, c.chainID)
	if err != nil {
		return nil, err
	}

	auth.Context = ctx

	// Set gas price if configured
	if c.config.GasPrice > 0 {
		const maxInt64 = 1<<63 - 1
		if c.config.GasPrice > maxInt64 {
			return nil, fmt.Errorf("gas price overflow: %d exceeds maximum int64 value", c.config.GasPrice)
		}
		auth.GasPrice = big.NewInt(int64(c.config.GasPrice)) // #nosec G115 - overflow checked above
	}

	return auth, nil
}

func (c *EthereumClient) waitForTransaction(ctx context.Context, tx *types.Transaction) (*types.Receipt, error) {
	// Wait for transaction to be mined
	for i := 0; i < c.config.MaxRetries; i++ {
		receipt, err := c.client.TransactionReceipt(ctx, tx.Hash())
		if err == nil {
			if receipt.Status == types.ReceiptStatusFailed {
				return nil, fmt.Errorf("transaction failed")
			}

			// Wait for confirmations
			if c.config.ConfirmationBlocks > 0 {
				currentBlock, err := c.client.BlockNumber(ctx)
				if err != nil {
					return nil, err
				}

				if c.config.ConfirmationBlocks < 0 {
					return nil, fmt.Errorf("confirmation blocks must be non-negative: %d", c.config.ConfirmationBlocks)
				}
				confirmations := currentBlock - receipt.BlockNumber.Uint64()
				// #nosec G115 -- ConfirmationBlocks is validated non-negative above
				if confirmations < uint64(c.config.ConfirmationBlocks) {
					time.Sleep(5 * time.Second)
					continue
				}
			}

			return receipt, nil
		}

		time.Sleep(5 * time.Second)
	}

	return nil, fmt.Errorf("transaction timeout")
}

func (c *EthereumClient) prepareRegistrationMessage(req *did.RegistrationRequest, address string) string {
	return fmt.Sprintf("Register agent:\nDID: %s\nName: %s\nEndpoint: %s\nAddress: %s",
		req.DID, req.Name, req.Endpoint, address)
}

func (c *EthereumClient) prepareUpdateMessage(agentDID did.AgentDID, updates map[string]interface{}) string {
	return fmt.Sprintf("Update agent: %s\nUpdates: %v", agentDID, updates)
}

func toKeyHashes(x interface{}) ([][32]byte, error) {
	switch t := x.(type) {
	case [][32]byte:
		return t, nil
	case []common.Hash:
		out := make([][32]byte, len(t))
		for i := range t {
			out[i] = t[i]
		}
		return out, nil
	case [][]byte:
		out := make([][32]byte, len(t))
		for i := range t {
			if len(t[i]) != 32 {
				return nil, fmt.Errorf("keyHash[%d] length=%d", i, len(t[i]))
			}
			copy(out[i][:], t[i])
		}
		return out, nil
	case []byte:
		// 평탄화된 bytes (32바이트씩 잘라서 복원)
		if len(t)%32 != 0 {
			return nil, fmt.Errorf("flat keyHashes length=%d not multiple of 32", len(t))
		}
		n := len(t) / 32
		out := make([][32]byte, n)
		for i := 0; i < n; i++ {
			copy(out[i][:], t[i*32:(i+1)*32])
		}
		return out, nil
	}

	// 리플렉션fallback: []uint8, [][]uint8, [][32]uint8 등 대응
	rv := reflect.ValueOf(x)
	if rv.Kind() == reflect.Slice {
		// slice elem이 uint8이면 []byte로 간주하고 평탄화 처리
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			buf := make([]byte, rv.Len())
			for i := 0; i < rv.Len(); i++ {
				buf[i] = byte(rv.Index(i).Uint())
			}
			if len(buf)%32 != 0 {
				return nil, fmt.Errorf("flat keyHashes(length=%d) not multiple of 32", len(buf))
			}
			n := len(buf) / 32
			out := make([][32]byte, n)
			for i := 0; i < n; i++ {
				copy(out[i][:], buf[i*32:(i+1)*32])
			}
			return out, nil
		}

		// 그 외: 요소별로 [32]byte/[]byte/common.Hash 등 처리
		out := make([][32]byte, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			elem := rv.Index(i).Interface()
			switch e := elem.(type) {
			case [32]byte:
				out[i] = e
			case common.Hash:
				out[i] = [32]byte(e)
			case []byte:
				if len(e) != 32 {
					return nil, fmt.Errorf("keyHash[%d] length=%d", i, len(e))
				}
				copy(out[i][:], e)
			default:
				ev := reflect.ValueOf(elem)
				if ev.Kind() == reflect.Array && ev.Len() == 32 && ev.Type().Elem().Kind() == reflect.Uint8 {
					for j := 0; j < 32; j++ {
						out[i][j] = byte(ev.Index(j).Uint())
					}
				} else {
					return nil, fmt.Errorf("unsupported keyHash elem type %T", elem)
				}
			}
		}
		return out, nil
	}

	// 배열(flat)도 지원
	if rv.Kind() == reflect.Array && rv.Type().Elem().Kind() == reflect.Uint8 {
		if rv.Len()%32 != 0 {
			return nil, fmt.Errorf("flat array keyHashes length=%d not multiple of 32", rv.Len())
		}
		n := rv.Len() / 32
		out := make([][32]byte, n)
		for i := 0; i < n; i++ {
			for j := 0; j < 32; j++ {
				out[i][j] = byte(rv.Index(i*32 + j).Uint())
			}
		}
		return out, nil
	}

	return nil, fmt.Errorf("unsupported keyHashes type %T", x)
}

// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package agentcardregistry

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// AgentCardStorageAgentKey is an auto generated low-level Go binding around an user-defined struct.
type AgentCardStorageAgentKey struct {
	KeyType      uint8
	KeyData      []byte
	Signature    []byte
	Verified     bool
	RegisteredAt *big.Int
}

// AgentCardStorageAgentMetadata is an auto generated low-level Go binding around an user-defined struct.
type AgentCardStorageAgentMetadata struct {
	Did          string
	Name         string
	Description  string
	Endpoint     string
	KeyHashes    [][32]byte
	Capabilities string
	Owner        common.Address
	RegisteredAt *big.Int
	UpdatedAt    *big.Int
	Active       bool
	ChainId      *big.Int
	KmePublicKey []byte
}

// AgentCardStorageRegistrationParams is an auto generated low-level Go binding around an user-defined struct.
type AgentCardStorageRegistrationParams struct {
	Did          string
	Name         string
	Description  string
	Endpoint     string
	Capabilities string
	Keys         [][]byte
	KeyTypes     []uint8
	Signatures   [][]byte
	Salt         [32]byte
}

// IERC8004IdentityRegistryAgentInfo is an auto generated low-level Go binding around an user-defined struct.
type IERC8004IdentityRegistryAgentInfo struct {
	AgentId      string
	AgentAddress common.Address
	Endpoint     string
	IsActive     bool
	RegisteredAt *big.Int
}

// AgentCardRegistryMetaData contains all meta data concerning the AgentCardRegistry contract.
var AgentCardRegistryMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_verifyHook\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"EnforcedPause\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ExpectedPause\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"OwnableInvalidOwner\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"OwnableUnauthorizedAccount\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ReentrancyGuardReentrantCall\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"}],\"name\":\"AgentActivated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"string\",\"name\":\"agentId\",\"type\":\"string\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"agentAddress\",\"type\":\"address\"}],\"name\":\"AgentDeactivated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"}],\"name\":\"AgentDeactivatedByHash\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"string\",\"name\":\"agentId\",\"type\":\"string\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"oldEndpoint\",\"type\":\"string\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"newEndpoint\",\"type\":\"string\"}],\"name\":\"AgentEndpointUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"string\",\"name\":\"did\",\"type\":\"string\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"}],\"name\":\"AgentRegistered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"string\",\"name\":\"agentId\",\"type\":\"string\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"agentAddress\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"endpoint\",\"type\":\"string\"}],\"name\":\"AgentRegistered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"}],\"name\":\"AgentUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"operator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"approved\",\"type\":\"bool\"}],\"name\":\"ApprovalForAgent\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"committer\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"commitHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"}],\"name\":\"CommitmentRecorded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"keyHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"}],\"name\":\"KMEKeyUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"keyHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"enumAgentCardStorage.KeyType\",\"name\":\"keyType\",\"type\":\"uint8\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"}],\"name\":\"KeyAdded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"keyHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"}],\"name\":\"KeyRevoked\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferStarted\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"Paused\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"Unpaused\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"acceptOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"}],\"name\":\"activateAgent\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"activationDelay\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"keyData\",\"type\":\"bytes\"},{\"internalType\":\"enumAgentCardStorage.KeyType\",\"name\":\"keyType\",\"type\":\"uint8\"},{\"internalType\":\"bytes\",\"name\":\"signature\",\"type\":\"bytes\"}],\"name\":\"addKey\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"agentActivationTime\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"agentNonce\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"agentOperators\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"agentReputations\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"successfulInteractions\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"failedInteractions\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"reputationScore\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"verified\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"agentStakes\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"commitHash\",\"type\":\"bytes32\"}],\"name\":\"commitRegistration\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"agentId\",\"type\":\"string\"}],\"name\":\"deactivateAgent\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"}],\"name\":\"deactivateAgentByHash\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"name\":\"didToAgentId\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"}],\"name\":\"getAgent\",\"outputs\":[{\"components\":[{\"internalType\":\"string\",\"name\":\"did\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"description\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"endpoint\",\"type\":\"string\"},{\"internalType\":\"bytes32[]\",\"name\":\"keyHashes\",\"type\":\"bytes32[]\"},{\"internalType\":\"string\",\"name\":\"capabilities\",\"type\":\"string\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"registeredAt\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"updatedAt\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"active\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"chainId\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"kmePublicKey\",\"type\":\"bytes\"}],\"internalType\":\"structAgentCardStorage.AgentMetadata\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"did\",\"type\":\"string\"}],\"name\":\"getAgentByDID\",\"outputs\":[{\"components\":[{\"internalType\":\"string\",\"name\":\"did\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"description\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"endpoint\",\"type\":\"string\"},{\"internalType\":\"bytes32[]\",\"name\":\"keyHashes\",\"type\":\"bytes32[]\"},{\"internalType\":\"string\",\"name\":\"capabilities\",\"type\":\"string\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"registeredAt\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"updatedAt\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"active\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"chainId\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"kmePublicKey\",\"type\":\"bytes\"}],\"internalType\":\"structAgentCardStorage.AgentMetadata\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"getAgentsByOwner\",\"outputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"\",\"type\":\"bytes32[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"}],\"name\":\"getKMEKey\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"keyHash\",\"type\":\"bytes32\"}],\"name\":\"getKey\",\"outputs\":[{\"components\":[{\"internalType\":\"enumAgentCardStorage.KeyType\",\"name\":\"keyType\",\"type\":\"uint8\"},{\"internalType\":\"bytes\",\"name\":\"keyData\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"signature\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"verified\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"registeredAt\",\"type\":\"uint256\"}],\"internalType\":\"structAgentCardStorage.AgentKey\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"agentId\",\"type\":\"string\"}],\"name\":\"isAgentActive\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"operator\",\"type\":\"address\"}],\"name\":\"isApprovedOperator\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"pause\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"paused\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"pendingOwner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"agentId\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"endpoint\",\"type\":\"string\"}],\"name\":\"registerAgent\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"string\",\"name\":\"did\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"description\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"endpoint\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"capabilities\",\"type\":\"string\"},{\"internalType\":\"bytes[]\",\"name\":\"keys\",\"type\":\"bytes[]\"},{\"internalType\":\"enumAgentCardStorage.KeyType[]\",\"name\":\"keyTypes\",\"type\":\"uint8[]\"},{\"internalType\":\"bytes[]\",\"name\":\"signatures\",\"type\":\"bytes[]\"},{\"internalType\":\"bytes32\",\"name\":\"salt\",\"type\":\"bytes32\"}],\"internalType\":\"structAgentCardStorage.RegistrationParams\",\"name\":\"params\",\"type\":\"tuple\"}],\"name\":\"registerAgentWithParams\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"registrationCommitments\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"commitHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"revealed\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"registrationStake\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"agentId\",\"type\":\"string\"}],\"name\":\"resolveAgent\",\"outputs\":[{\"components\":[{\"internalType\":\"string\",\"name\":\"agentId\",\"type\":\"string\"},{\"internalType\":\"address\",\"name\":\"agentAddress\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"endpoint\",\"type\":\"string\"},{\"internalType\":\"bool\",\"name\":\"isActive\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"registeredAt\",\"type\":\"uint256\"}],\"internalType\":\"structIERC8004IdentityRegistry.AgentInfo\",\"name\":\"info\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"agentAddress\",\"type\":\"address\"}],\"name\":\"resolveAgentByAddress\",\"outputs\":[{\"components\":[{\"internalType\":\"string\",\"name\":\"agentId\",\"type\":\"string\"},{\"internalType\":\"address\",\"name\":\"agentAddress\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"endpoint\",\"type\":\"string\"},{\"internalType\":\"bool\",\"name\":\"isActive\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"registeredAt\",\"type\":\"uint256\"}],\"internalType\":\"structIERC8004IdentityRegistry.AgentInfo\",\"name\":\"info\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"keyHash\",\"type\":\"bytes32\"}],\"name\":\"revokeKey\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"newDelay\",\"type\":\"uint256\"}],\"name\":\"setActivationDelay\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"operator\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"approved\",\"type\":\"bool\"}],\"name\":\"setApprovalForAgent\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"newStake\",\"type\":\"uint256\"}],\"name\":\"setRegistrationStake\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newHook\",\"type\":\"address\"}],\"name\":\"setVerifyHook\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"unpause\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"},{\"internalType\":\"string\",\"name\":\"endpoint\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"capabilities\",\"type\":\"string\"}],\"name\":\"updateAgent\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"agentId\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"newEndpoint\",\"type\":\"string\"}],\"name\":\"updateAgentEndpoint\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"newKmeKey\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"signature\",\"type\":\"bytes\"}],\"name\":\"updateKMEKey\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"verifyHook\",\"outputs\":[{\"internalType\":\"contractAgentCardVerifyHook\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// AgentCardRegistryABI is the input ABI used to generate the binding from.
// Deprecated: Use AgentCardRegistryMetaData.ABI instead.
var AgentCardRegistryABI = AgentCardRegistryMetaData.ABI

// AgentCardRegistry is an auto generated Go binding around an Ethereum contract.
type AgentCardRegistry struct {
	AgentCardRegistryCaller     // Read-only binding to the contract
	AgentCardRegistryTransactor // Write-only binding to the contract
	AgentCardRegistryFilterer   // Log filterer for contract events
}

// AgentCardRegistryCaller is an auto generated read-only Go binding around an Ethereum contract.
type AgentCardRegistryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AgentCardRegistryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type AgentCardRegistryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AgentCardRegistryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type AgentCardRegistryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AgentCardRegistrySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type AgentCardRegistrySession struct {
	Contract     *AgentCardRegistry // Generic contract binding to set the session for
	CallOpts     bind.CallOpts      // Call options to use throughout this session
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// AgentCardRegistryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type AgentCardRegistryCallerSession struct {
	Contract *AgentCardRegistryCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts            // Call options to use throughout this session
}

// AgentCardRegistryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type AgentCardRegistryTransactorSession struct {
	Contract     *AgentCardRegistryTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts            // Transaction auth options to use throughout this session
}

// AgentCardRegistryRaw is an auto generated low-level Go binding around an Ethereum contract.
type AgentCardRegistryRaw struct {
	Contract *AgentCardRegistry // Generic contract binding to access the raw methods on
}

// AgentCardRegistryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type AgentCardRegistryCallerRaw struct {
	Contract *AgentCardRegistryCaller // Generic read-only contract binding to access the raw methods on
}

// AgentCardRegistryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type AgentCardRegistryTransactorRaw struct {
	Contract *AgentCardRegistryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewAgentCardRegistry creates a new instance of AgentCardRegistry, bound to a specific deployed contract.
func NewAgentCardRegistry(address common.Address, backend bind.ContractBackend) (*AgentCardRegistry, error) {
	contract, err := bindAgentCardRegistry(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &AgentCardRegistry{AgentCardRegistryCaller: AgentCardRegistryCaller{contract: contract}, AgentCardRegistryTransactor: AgentCardRegistryTransactor{contract: contract}, AgentCardRegistryFilterer: AgentCardRegistryFilterer{contract: contract}}, nil
}

// NewAgentCardRegistryCaller creates a new read-only instance of AgentCardRegistry, bound to a specific deployed contract.
func NewAgentCardRegistryCaller(address common.Address, caller bind.ContractCaller) (*AgentCardRegistryCaller, error) {
	contract, err := bindAgentCardRegistry(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &AgentCardRegistryCaller{contract: contract}, nil
}

// NewAgentCardRegistryTransactor creates a new write-only instance of AgentCardRegistry, bound to a specific deployed contract.
func NewAgentCardRegistryTransactor(address common.Address, transactor bind.ContractTransactor) (*AgentCardRegistryTransactor, error) {
	contract, err := bindAgentCardRegistry(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &AgentCardRegistryTransactor{contract: contract}, nil
}

// NewAgentCardRegistryFilterer creates a new log filterer instance of AgentCardRegistry, bound to a specific deployed contract.
func NewAgentCardRegistryFilterer(address common.Address, filterer bind.ContractFilterer) (*AgentCardRegistryFilterer, error) {
	contract, err := bindAgentCardRegistry(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &AgentCardRegistryFilterer{contract: contract}, nil
}

// bindAgentCardRegistry binds a generic wrapper to an already deployed contract.
func bindAgentCardRegistry(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := AgentCardRegistryMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_AgentCardRegistry *AgentCardRegistryRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _AgentCardRegistry.Contract.AgentCardRegistryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_AgentCardRegistry *AgentCardRegistryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AgentCardRegistry.Contract.AgentCardRegistryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_AgentCardRegistry *AgentCardRegistryRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _AgentCardRegistry.Contract.AgentCardRegistryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_AgentCardRegistry *AgentCardRegistryCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _AgentCardRegistry.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_AgentCardRegistry *AgentCardRegistryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AgentCardRegistry.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_AgentCardRegistry *AgentCardRegistryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _AgentCardRegistry.Contract.contract.Transact(opts, method, params...)
}

// ActivationDelay is a free data retrieval call binding the contract method 0x3a8c0786.
//
// Solidity: function activationDelay() view returns(uint256)
func (_AgentCardRegistry *AgentCardRegistryCaller) ActivationDelay(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _AgentCardRegistry.contract.Call(opts, &out, "activationDelay")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ActivationDelay is a free data retrieval call binding the contract method 0x3a8c0786.
//
// Solidity: function activationDelay() view returns(uint256)
func (_AgentCardRegistry *AgentCardRegistrySession) ActivationDelay() (*big.Int, error) {
	return _AgentCardRegistry.Contract.ActivationDelay(&_AgentCardRegistry.CallOpts)
}

// ActivationDelay is a free data retrieval call binding the contract method 0x3a8c0786.
//
// Solidity: function activationDelay() view returns(uint256)
func (_AgentCardRegistry *AgentCardRegistryCallerSession) ActivationDelay() (*big.Int, error) {
	return _AgentCardRegistry.Contract.ActivationDelay(&_AgentCardRegistry.CallOpts)
}

// AgentActivationTime is a free data retrieval call binding the contract method 0xd37818a9.
//
// Solidity: function agentActivationTime(bytes32 ) view returns(uint256)
func (_AgentCardRegistry *AgentCardRegistryCaller) AgentActivationTime(opts *bind.CallOpts, arg0 [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _AgentCardRegistry.contract.Call(opts, &out, "agentActivationTime", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// AgentActivationTime is a free data retrieval call binding the contract method 0xd37818a9.
//
// Solidity: function agentActivationTime(bytes32 ) view returns(uint256)
func (_AgentCardRegistry *AgentCardRegistrySession) AgentActivationTime(arg0 [32]byte) (*big.Int, error) {
	return _AgentCardRegistry.Contract.AgentActivationTime(&_AgentCardRegistry.CallOpts, arg0)
}

// AgentActivationTime is a free data retrieval call binding the contract method 0xd37818a9.
//
// Solidity: function agentActivationTime(bytes32 ) view returns(uint256)
func (_AgentCardRegistry *AgentCardRegistryCallerSession) AgentActivationTime(arg0 [32]byte) (*big.Int, error) {
	return _AgentCardRegistry.Contract.AgentActivationTime(&_AgentCardRegistry.CallOpts, arg0)
}

// AgentNonce is a free data retrieval call binding the contract method 0x6073c341.
//
// Solidity: function agentNonce(bytes32 ) view returns(uint256)
func (_AgentCardRegistry *AgentCardRegistryCaller) AgentNonce(opts *bind.CallOpts, arg0 [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _AgentCardRegistry.contract.Call(opts, &out, "agentNonce", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// AgentNonce is a free data retrieval call binding the contract method 0x6073c341.
//
// Solidity: function agentNonce(bytes32 ) view returns(uint256)
func (_AgentCardRegistry *AgentCardRegistrySession) AgentNonce(arg0 [32]byte) (*big.Int, error) {
	return _AgentCardRegistry.Contract.AgentNonce(&_AgentCardRegistry.CallOpts, arg0)
}

// AgentNonce is a free data retrieval call binding the contract method 0x6073c341.
//
// Solidity: function agentNonce(bytes32 ) view returns(uint256)
func (_AgentCardRegistry *AgentCardRegistryCallerSession) AgentNonce(arg0 [32]byte) (*big.Int, error) {
	return _AgentCardRegistry.Contract.AgentNonce(&_AgentCardRegistry.CallOpts, arg0)
}

// AgentOperators is a free data retrieval call binding the contract method 0x0633d3d3.
//
// Solidity: function agentOperators(bytes32 , address ) view returns(bool)
func (_AgentCardRegistry *AgentCardRegistryCaller) AgentOperators(opts *bind.CallOpts, arg0 [32]byte, arg1 common.Address) (bool, error) {
	var out []interface{}
	err := _AgentCardRegistry.contract.Call(opts, &out, "agentOperators", arg0, arg1)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// AgentOperators is a free data retrieval call binding the contract method 0x0633d3d3.
//
// Solidity: function agentOperators(bytes32 , address ) view returns(bool)
func (_AgentCardRegistry *AgentCardRegistrySession) AgentOperators(arg0 [32]byte, arg1 common.Address) (bool, error) {
	return _AgentCardRegistry.Contract.AgentOperators(&_AgentCardRegistry.CallOpts, arg0, arg1)
}

// AgentOperators is a free data retrieval call binding the contract method 0x0633d3d3.
//
// Solidity: function agentOperators(bytes32 , address ) view returns(bool)
func (_AgentCardRegistry *AgentCardRegistryCallerSession) AgentOperators(arg0 [32]byte, arg1 common.Address) (bool, error) {
	return _AgentCardRegistry.Contract.AgentOperators(&_AgentCardRegistry.CallOpts, arg0, arg1)
}

// AgentReputations is a free data retrieval call binding the contract method 0xd0c11aaa.
//
// Solidity: function agentReputations(address ) view returns(uint256 successfulInteractions, uint256 failedInteractions, uint256 reputationScore, bool verified)
func (_AgentCardRegistry *AgentCardRegistryCaller) AgentReputations(opts *bind.CallOpts, arg0 common.Address) (struct {
	SuccessfulInteractions *big.Int
	FailedInteractions     *big.Int
	ReputationScore        *big.Int
	Verified               bool
}, error) {
	var out []interface{}
	err := _AgentCardRegistry.contract.Call(opts, &out, "agentReputations", arg0)

	outstruct := new(struct {
		SuccessfulInteractions *big.Int
		FailedInteractions     *big.Int
		ReputationScore        *big.Int
		Verified               bool
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.SuccessfulInteractions = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.FailedInteractions = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.ReputationScore = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.Verified = *abi.ConvertType(out[3], new(bool)).(*bool)

	return *outstruct, err

}

// AgentReputations is a free data retrieval call binding the contract method 0xd0c11aaa.
//
// Solidity: function agentReputations(address ) view returns(uint256 successfulInteractions, uint256 failedInteractions, uint256 reputationScore, bool verified)
func (_AgentCardRegistry *AgentCardRegistrySession) AgentReputations(arg0 common.Address) (struct {
	SuccessfulInteractions *big.Int
	FailedInteractions     *big.Int
	ReputationScore        *big.Int
	Verified               bool
}, error) {
	return _AgentCardRegistry.Contract.AgentReputations(&_AgentCardRegistry.CallOpts, arg0)
}

// AgentReputations is a free data retrieval call binding the contract method 0xd0c11aaa.
//
// Solidity: function agentReputations(address ) view returns(uint256 successfulInteractions, uint256 failedInteractions, uint256 reputationScore, bool verified)
func (_AgentCardRegistry *AgentCardRegistryCallerSession) AgentReputations(arg0 common.Address) (struct {
	SuccessfulInteractions *big.Int
	FailedInteractions     *big.Int
	ReputationScore        *big.Int
	Verified               bool
}, error) {
	return _AgentCardRegistry.Contract.AgentReputations(&_AgentCardRegistry.CallOpts, arg0)
}

// AgentStakes is a free data retrieval call binding the contract method 0x64983f18.
//
// Solidity: function agentStakes(bytes32 ) view returns(uint256)
func (_AgentCardRegistry *AgentCardRegistryCaller) AgentStakes(opts *bind.CallOpts, arg0 [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _AgentCardRegistry.contract.Call(opts, &out, "agentStakes", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// AgentStakes is a free data retrieval call binding the contract method 0x64983f18.
//
// Solidity: function agentStakes(bytes32 ) view returns(uint256)
func (_AgentCardRegistry *AgentCardRegistrySession) AgentStakes(arg0 [32]byte) (*big.Int, error) {
	return _AgentCardRegistry.Contract.AgentStakes(&_AgentCardRegistry.CallOpts, arg0)
}

// AgentStakes is a free data retrieval call binding the contract method 0x64983f18.
//
// Solidity: function agentStakes(bytes32 ) view returns(uint256)
func (_AgentCardRegistry *AgentCardRegistryCallerSession) AgentStakes(arg0 [32]byte) (*big.Int, error) {
	return _AgentCardRegistry.Contract.AgentStakes(&_AgentCardRegistry.CallOpts, arg0)
}

// DidToAgentId is a free data retrieval call binding the contract method 0xf0944df4.
//
// Solidity: function didToAgentId(string ) view returns(bytes32)
func (_AgentCardRegistry *AgentCardRegistryCaller) DidToAgentId(opts *bind.CallOpts, arg0 string) ([32]byte, error) {
	var out []interface{}
	err := _AgentCardRegistry.contract.Call(opts, &out, "didToAgentId", arg0)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// DidToAgentId is a free data retrieval call binding the contract method 0xf0944df4.
//
// Solidity: function didToAgentId(string ) view returns(bytes32)
func (_AgentCardRegistry *AgentCardRegistrySession) DidToAgentId(arg0 string) ([32]byte, error) {
	return _AgentCardRegistry.Contract.DidToAgentId(&_AgentCardRegistry.CallOpts, arg0)
}

// DidToAgentId is a free data retrieval call binding the contract method 0xf0944df4.
//
// Solidity: function didToAgentId(string ) view returns(bytes32)
func (_AgentCardRegistry *AgentCardRegistryCallerSession) DidToAgentId(arg0 string) ([32]byte, error) {
	return _AgentCardRegistry.Contract.DidToAgentId(&_AgentCardRegistry.CallOpts, arg0)
}

// GetAgent is a free data retrieval call binding the contract method 0xa6c2af01.
//
// Solidity: function getAgent(bytes32 agentId) view returns((string,string,string,string,bytes32[],string,address,uint256,uint256,bool,uint256,bytes))
func (_AgentCardRegistry *AgentCardRegistryCaller) GetAgent(opts *bind.CallOpts, agentId [32]byte) (AgentCardStorageAgentMetadata, error) {
	var out []interface{}
	err := _AgentCardRegistry.contract.Call(opts, &out, "getAgent", agentId)

	if err != nil {
		return *new(AgentCardStorageAgentMetadata), err
	}

	out0 := *abi.ConvertType(out[0], new(AgentCardStorageAgentMetadata)).(*AgentCardStorageAgentMetadata)

	return out0, err

}

// GetAgent is a free data retrieval call binding the contract method 0xa6c2af01.
//
// Solidity: function getAgent(bytes32 agentId) view returns((string,string,string,string,bytes32[],string,address,uint256,uint256,bool,uint256,bytes))
func (_AgentCardRegistry *AgentCardRegistrySession) GetAgent(agentId [32]byte) (AgentCardStorageAgentMetadata, error) {
	return _AgentCardRegistry.Contract.GetAgent(&_AgentCardRegistry.CallOpts, agentId)
}

// GetAgent is a free data retrieval call binding the contract method 0xa6c2af01.
//
// Solidity: function getAgent(bytes32 agentId) view returns((string,string,string,string,bytes32[],string,address,uint256,uint256,bool,uint256,bytes))
func (_AgentCardRegistry *AgentCardRegistryCallerSession) GetAgent(agentId [32]byte) (AgentCardStorageAgentMetadata, error) {
	return _AgentCardRegistry.Contract.GetAgent(&_AgentCardRegistry.CallOpts, agentId)
}

// GetAgentByDID is a free data retrieval call binding the contract method 0xe45d486d.
//
// Solidity: function getAgentByDID(string did) view returns((string,string,string,string,bytes32[],string,address,uint256,uint256,bool,uint256,bytes))
func (_AgentCardRegistry *AgentCardRegistryCaller) GetAgentByDID(opts *bind.CallOpts, did string) (AgentCardStorageAgentMetadata, error) {
	var out []interface{}
	err := _AgentCardRegistry.contract.Call(opts, &out, "getAgentByDID", did)

	if err != nil {
		return *new(AgentCardStorageAgentMetadata), err
	}

	out0 := *abi.ConvertType(out[0], new(AgentCardStorageAgentMetadata)).(*AgentCardStorageAgentMetadata)

	return out0, err

}

// GetAgentByDID is a free data retrieval call binding the contract method 0xe45d486d.
//
// Solidity: function getAgentByDID(string did) view returns((string,string,string,string,bytes32[],string,address,uint256,uint256,bool,uint256,bytes))
func (_AgentCardRegistry *AgentCardRegistrySession) GetAgentByDID(did string) (AgentCardStorageAgentMetadata, error) {
	return _AgentCardRegistry.Contract.GetAgentByDID(&_AgentCardRegistry.CallOpts, did)
}

// GetAgentByDID is a free data retrieval call binding the contract method 0xe45d486d.
//
// Solidity: function getAgentByDID(string did) view returns((string,string,string,string,bytes32[],string,address,uint256,uint256,bool,uint256,bytes))
func (_AgentCardRegistry *AgentCardRegistryCallerSession) GetAgentByDID(did string) (AgentCardStorageAgentMetadata, error) {
	return _AgentCardRegistry.Contract.GetAgentByDID(&_AgentCardRegistry.CallOpts, did)
}

// GetAgentsByOwner is a free data retrieval call binding the contract method 0x1ab6f888.
//
// Solidity: function getAgentsByOwner(address owner) view returns(bytes32[])
func (_AgentCardRegistry *AgentCardRegistryCaller) GetAgentsByOwner(opts *bind.CallOpts, owner common.Address) ([][32]byte, error) {
	var out []interface{}
	err := _AgentCardRegistry.contract.Call(opts, &out, "getAgentsByOwner", owner)

	if err != nil {
		return *new([][32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([][32]byte)).(*[][32]byte)

	return out0, err

}

// GetAgentsByOwner is a free data retrieval call binding the contract method 0x1ab6f888.
//
// Solidity: function getAgentsByOwner(address owner) view returns(bytes32[])
func (_AgentCardRegistry *AgentCardRegistrySession) GetAgentsByOwner(owner common.Address) ([][32]byte, error) {
	return _AgentCardRegistry.Contract.GetAgentsByOwner(&_AgentCardRegistry.CallOpts, owner)
}

// GetAgentsByOwner is a free data retrieval call binding the contract method 0x1ab6f888.
//
// Solidity: function getAgentsByOwner(address owner) view returns(bytes32[])
func (_AgentCardRegistry *AgentCardRegistryCallerSession) GetAgentsByOwner(owner common.Address) ([][32]byte, error) {
	return _AgentCardRegistry.Contract.GetAgentsByOwner(&_AgentCardRegistry.CallOpts, owner)
}

// GetKMEKey is a free data retrieval call binding the contract method 0x16ba1d5c.
//
// Solidity: function getKMEKey(bytes32 agentId) view returns(bytes)
func (_AgentCardRegistry *AgentCardRegistryCaller) GetKMEKey(opts *bind.CallOpts, agentId [32]byte) ([]byte, error) {
	var out []interface{}
	err := _AgentCardRegistry.contract.Call(opts, &out, "getKMEKey", agentId)

	if err != nil {
		return *new([]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([]byte)).(*[]byte)

	return out0, err

}

// GetKMEKey is a free data retrieval call binding the contract method 0x16ba1d5c.
//
// Solidity: function getKMEKey(bytes32 agentId) view returns(bytes)
func (_AgentCardRegistry *AgentCardRegistrySession) GetKMEKey(agentId [32]byte) ([]byte, error) {
	return _AgentCardRegistry.Contract.GetKMEKey(&_AgentCardRegistry.CallOpts, agentId)
}

// GetKMEKey is a free data retrieval call binding the contract method 0x16ba1d5c.
//
// Solidity: function getKMEKey(bytes32 agentId) view returns(bytes)
func (_AgentCardRegistry *AgentCardRegistryCallerSession) GetKMEKey(agentId [32]byte) ([]byte, error) {
	return _AgentCardRegistry.Contract.GetKMEKey(&_AgentCardRegistry.CallOpts, agentId)
}

// GetKey is a free data retrieval call binding the contract method 0x12aaac70.
//
// Solidity: function getKey(bytes32 keyHash) view returns((uint8,bytes,bytes,bool,uint256))
func (_AgentCardRegistry *AgentCardRegistryCaller) GetKey(opts *bind.CallOpts, keyHash [32]byte) (AgentCardStorageAgentKey, error) {
	var out []interface{}
	err := _AgentCardRegistry.contract.Call(opts, &out, "getKey", keyHash)

	if err != nil {
		return *new(AgentCardStorageAgentKey), err
	}

	out0 := *abi.ConvertType(out[0], new(AgentCardStorageAgentKey)).(*AgentCardStorageAgentKey)

	return out0, err

}

// GetKey is a free data retrieval call binding the contract method 0x12aaac70.
//
// Solidity: function getKey(bytes32 keyHash) view returns((uint8,bytes,bytes,bool,uint256))
func (_AgentCardRegistry *AgentCardRegistrySession) GetKey(keyHash [32]byte) (AgentCardStorageAgentKey, error) {
	return _AgentCardRegistry.Contract.GetKey(&_AgentCardRegistry.CallOpts, keyHash)
}

// GetKey is a free data retrieval call binding the contract method 0x12aaac70.
//
// Solidity: function getKey(bytes32 keyHash) view returns((uint8,bytes,bytes,bool,uint256))
func (_AgentCardRegistry *AgentCardRegistryCallerSession) GetKey(keyHash [32]byte) (AgentCardStorageAgentKey, error) {
	return _AgentCardRegistry.Contract.GetKey(&_AgentCardRegistry.CallOpts, keyHash)
}

// IsAgentActive is a free data retrieval call binding the contract method 0xefe2d336.
//
// Solidity: function isAgentActive(string agentId) view returns(bool)
func (_AgentCardRegistry *AgentCardRegistryCaller) IsAgentActive(opts *bind.CallOpts, agentId string) (bool, error) {
	var out []interface{}
	err := _AgentCardRegistry.contract.Call(opts, &out, "isAgentActive", agentId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsAgentActive is a free data retrieval call binding the contract method 0xefe2d336.
//
// Solidity: function isAgentActive(string agentId) view returns(bool)
func (_AgentCardRegistry *AgentCardRegistrySession) IsAgentActive(agentId string) (bool, error) {
	return _AgentCardRegistry.Contract.IsAgentActive(&_AgentCardRegistry.CallOpts, agentId)
}

// IsAgentActive is a free data retrieval call binding the contract method 0xefe2d336.
//
// Solidity: function isAgentActive(string agentId) view returns(bool)
func (_AgentCardRegistry *AgentCardRegistryCallerSession) IsAgentActive(agentId string) (bool, error) {
	return _AgentCardRegistry.Contract.IsAgentActive(&_AgentCardRegistry.CallOpts, agentId)
}

// IsApprovedOperator is a free data retrieval call binding the contract method 0xe04569a1.
//
// Solidity: function isApprovedOperator(bytes32 agentId, address operator) view returns(bool)
func (_AgentCardRegistry *AgentCardRegistryCaller) IsApprovedOperator(opts *bind.CallOpts, agentId [32]byte, operator common.Address) (bool, error) {
	var out []interface{}
	err := _AgentCardRegistry.contract.Call(opts, &out, "isApprovedOperator", agentId, operator)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsApprovedOperator is a free data retrieval call binding the contract method 0xe04569a1.
//
// Solidity: function isApprovedOperator(bytes32 agentId, address operator) view returns(bool)
func (_AgentCardRegistry *AgentCardRegistrySession) IsApprovedOperator(agentId [32]byte, operator common.Address) (bool, error) {
	return _AgentCardRegistry.Contract.IsApprovedOperator(&_AgentCardRegistry.CallOpts, agentId, operator)
}

// IsApprovedOperator is a free data retrieval call binding the contract method 0xe04569a1.
//
// Solidity: function isApprovedOperator(bytes32 agentId, address operator) view returns(bool)
func (_AgentCardRegistry *AgentCardRegistryCallerSession) IsApprovedOperator(agentId [32]byte, operator common.Address) (bool, error) {
	return _AgentCardRegistry.Contract.IsApprovedOperator(&_AgentCardRegistry.CallOpts, agentId, operator)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_AgentCardRegistry *AgentCardRegistryCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _AgentCardRegistry.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_AgentCardRegistry *AgentCardRegistrySession) Owner() (common.Address, error) {
	return _AgentCardRegistry.Contract.Owner(&_AgentCardRegistry.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_AgentCardRegistry *AgentCardRegistryCallerSession) Owner() (common.Address, error) {
	return _AgentCardRegistry.Contract.Owner(&_AgentCardRegistry.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_AgentCardRegistry *AgentCardRegistryCaller) Paused(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _AgentCardRegistry.contract.Call(opts, &out, "paused")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_AgentCardRegistry *AgentCardRegistrySession) Paused() (bool, error) {
	return _AgentCardRegistry.Contract.Paused(&_AgentCardRegistry.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_AgentCardRegistry *AgentCardRegistryCallerSession) Paused() (bool, error) {
	return _AgentCardRegistry.Contract.Paused(&_AgentCardRegistry.CallOpts)
}

// PendingOwner is a free data retrieval call binding the contract method 0xe30c3978.
//
// Solidity: function pendingOwner() view returns(address)
func (_AgentCardRegistry *AgentCardRegistryCaller) PendingOwner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _AgentCardRegistry.contract.Call(opts, &out, "pendingOwner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// PendingOwner is a free data retrieval call binding the contract method 0xe30c3978.
//
// Solidity: function pendingOwner() view returns(address)
func (_AgentCardRegistry *AgentCardRegistrySession) PendingOwner() (common.Address, error) {
	return _AgentCardRegistry.Contract.PendingOwner(&_AgentCardRegistry.CallOpts)
}

// PendingOwner is a free data retrieval call binding the contract method 0xe30c3978.
//
// Solidity: function pendingOwner() view returns(address)
func (_AgentCardRegistry *AgentCardRegistryCallerSession) PendingOwner() (common.Address, error) {
	return _AgentCardRegistry.Contract.PendingOwner(&_AgentCardRegistry.CallOpts)
}

// RegistrationCommitments is a free data retrieval call binding the contract method 0xadd0b94e.
//
// Solidity: function registrationCommitments(address ) view returns(bytes32 commitHash, uint256 timestamp, bool revealed)
func (_AgentCardRegistry *AgentCardRegistryCaller) RegistrationCommitments(opts *bind.CallOpts, arg0 common.Address) (struct {
	CommitHash [32]byte
	Timestamp  *big.Int
	Revealed   bool
}, error) {
	var out []interface{}
	err := _AgentCardRegistry.contract.Call(opts, &out, "registrationCommitments", arg0)

	outstruct := new(struct {
		CommitHash [32]byte
		Timestamp  *big.Int
		Revealed   bool
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.CommitHash = *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)
	outstruct.Timestamp = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.Revealed = *abi.ConvertType(out[2], new(bool)).(*bool)

	return *outstruct, err

}

// RegistrationCommitments is a free data retrieval call binding the contract method 0xadd0b94e.
//
// Solidity: function registrationCommitments(address ) view returns(bytes32 commitHash, uint256 timestamp, bool revealed)
func (_AgentCardRegistry *AgentCardRegistrySession) RegistrationCommitments(arg0 common.Address) (struct {
	CommitHash [32]byte
	Timestamp  *big.Int
	Revealed   bool
}, error) {
	return _AgentCardRegistry.Contract.RegistrationCommitments(&_AgentCardRegistry.CallOpts, arg0)
}

// RegistrationCommitments is a free data retrieval call binding the contract method 0xadd0b94e.
//
// Solidity: function registrationCommitments(address ) view returns(bytes32 commitHash, uint256 timestamp, bool revealed)
func (_AgentCardRegistry *AgentCardRegistryCallerSession) RegistrationCommitments(arg0 common.Address) (struct {
	CommitHash [32]byte
	Timestamp  *big.Int
	Revealed   bool
}, error) {
	return _AgentCardRegistry.Contract.RegistrationCommitments(&_AgentCardRegistry.CallOpts, arg0)
}

// RegistrationStake is a free data retrieval call binding the contract method 0x73e6b4a1.
//
// Solidity: function registrationStake() view returns(uint256)
func (_AgentCardRegistry *AgentCardRegistryCaller) RegistrationStake(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _AgentCardRegistry.contract.Call(opts, &out, "registrationStake")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// RegistrationStake is a free data retrieval call binding the contract method 0x73e6b4a1.
//
// Solidity: function registrationStake() view returns(uint256)
func (_AgentCardRegistry *AgentCardRegistrySession) RegistrationStake() (*big.Int, error) {
	return _AgentCardRegistry.Contract.RegistrationStake(&_AgentCardRegistry.CallOpts)
}

// RegistrationStake is a free data retrieval call binding the contract method 0x73e6b4a1.
//
// Solidity: function registrationStake() view returns(uint256)
func (_AgentCardRegistry *AgentCardRegistryCallerSession) RegistrationStake() (*big.Int, error) {
	return _AgentCardRegistry.Contract.RegistrationStake(&_AgentCardRegistry.CallOpts)
}

// ResolveAgent is a free data retrieval call binding the contract method 0x39299d99.
//
// Solidity: function resolveAgent(string agentId) view returns((string,address,string,bool,uint256) info)
func (_AgentCardRegistry *AgentCardRegistryCaller) ResolveAgent(opts *bind.CallOpts, agentId string) (IERC8004IdentityRegistryAgentInfo, error) {
	var out []interface{}
	err := _AgentCardRegistry.contract.Call(opts, &out, "resolveAgent", agentId)

	if err != nil {
		return *new(IERC8004IdentityRegistryAgentInfo), err
	}

	out0 := *abi.ConvertType(out[0], new(IERC8004IdentityRegistryAgentInfo)).(*IERC8004IdentityRegistryAgentInfo)

	return out0, err

}

// ResolveAgent is a free data retrieval call binding the contract method 0x39299d99.
//
// Solidity: function resolveAgent(string agentId) view returns((string,address,string,bool,uint256) info)
func (_AgentCardRegistry *AgentCardRegistrySession) ResolveAgent(agentId string) (IERC8004IdentityRegistryAgentInfo, error) {
	return _AgentCardRegistry.Contract.ResolveAgent(&_AgentCardRegistry.CallOpts, agentId)
}

// ResolveAgent is a free data retrieval call binding the contract method 0x39299d99.
//
// Solidity: function resolveAgent(string agentId) view returns((string,address,string,bool,uint256) info)
func (_AgentCardRegistry *AgentCardRegistryCallerSession) ResolveAgent(agentId string) (IERC8004IdentityRegistryAgentInfo, error) {
	return _AgentCardRegistry.Contract.ResolveAgent(&_AgentCardRegistry.CallOpts, agentId)
}

// ResolveAgentByAddress is a free data retrieval call binding the contract method 0x88282bc8.
//
// Solidity: function resolveAgentByAddress(address agentAddress) view returns((string,address,string,bool,uint256) info)
func (_AgentCardRegistry *AgentCardRegistryCaller) ResolveAgentByAddress(opts *bind.CallOpts, agentAddress common.Address) (IERC8004IdentityRegistryAgentInfo, error) {
	var out []interface{}
	err := _AgentCardRegistry.contract.Call(opts, &out, "resolveAgentByAddress", agentAddress)

	if err != nil {
		return *new(IERC8004IdentityRegistryAgentInfo), err
	}

	out0 := *abi.ConvertType(out[0], new(IERC8004IdentityRegistryAgentInfo)).(*IERC8004IdentityRegistryAgentInfo)

	return out0, err

}

// ResolveAgentByAddress is a free data retrieval call binding the contract method 0x88282bc8.
//
// Solidity: function resolveAgentByAddress(address agentAddress) view returns((string,address,string,bool,uint256) info)
func (_AgentCardRegistry *AgentCardRegistrySession) ResolveAgentByAddress(agentAddress common.Address) (IERC8004IdentityRegistryAgentInfo, error) {
	return _AgentCardRegistry.Contract.ResolveAgentByAddress(&_AgentCardRegistry.CallOpts, agentAddress)
}

// ResolveAgentByAddress is a free data retrieval call binding the contract method 0x88282bc8.
//
// Solidity: function resolveAgentByAddress(address agentAddress) view returns((string,address,string,bool,uint256) info)
func (_AgentCardRegistry *AgentCardRegistryCallerSession) ResolveAgentByAddress(agentAddress common.Address) (IERC8004IdentityRegistryAgentInfo, error) {
	return _AgentCardRegistry.Contract.ResolveAgentByAddress(&_AgentCardRegistry.CallOpts, agentAddress)
}

// VerifyHook is a free data retrieval call binding the contract method 0xa3793f44.
//
// Solidity: function verifyHook() view returns(address)
func (_AgentCardRegistry *AgentCardRegistryCaller) VerifyHook(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _AgentCardRegistry.contract.Call(opts, &out, "verifyHook")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// VerifyHook is a free data retrieval call binding the contract method 0xa3793f44.
//
// Solidity: function verifyHook() view returns(address)
func (_AgentCardRegistry *AgentCardRegistrySession) VerifyHook() (common.Address, error) {
	return _AgentCardRegistry.Contract.VerifyHook(&_AgentCardRegistry.CallOpts)
}

// VerifyHook is a free data retrieval call binding the contract method 0xa3793f44.
//
// Solidity: function verifyHook() view returns(address)
func (_AgentCardRegistry *AgentCardRegistryCallerSession) VerifyHook() (common.Address, error) {
	return _AgentCardRegistry.Contract.VerifyHook(&_AgentCardRegistry.CallOpts)
}

// AcceptOwnership is a paid mutator transaction binding the contract method 0x79ba5097.
//
// Solidity: function acceptOwnership() returns()
func (_AgentCardRegistry *AgentCardRegistryTransactor) AcceptOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AgentCardRegistry.contract.Transact(opts, "acceptOwnership")
}

// AcceptOwnership is a paid mutator transaction binding the contract method 0x79ba5097.
//
// Solidity: function acceptOwnership() returns()
func (_AgentCardRegistry *AgentCardRegistrySession) AcceptOwnership() (*types.Transaction, error) {
	return _AgentCardRegistry.Contract.AcceptOwnership(&_AgentCardRegistry.TransactOpts)
}

// AcceptOwnership is a paid mutator transaction binding the contract method 0x79ba5097.
//
// Solidity: function acceptOwnership() returns()
func (_AgentCardRegistry *AgentCardRegistryTransactorSession) AcceptOwnership() (*types.Transaction, error) {
	return _AgentCardRegistry.Contract.AcceptOwnership(&_AgentCardRegistry.TransactOpts)
}

// ActivateAgent is a paid mutator transaction binding the contract method 0xb015813b.
//
// Solidity: function activateAgent(bytes32 agentId) returns()
func (_AgentCardRegistry *AgentCardRegistryTransactor) ActivateAgent(opts *bind.TransactOpts, agentId [32]byte) (*types.Transaction, error) {
	return _AgentCardRegistry.contract.Transact(opts, "activateAgent", agentId)
}

// ActivateAgent is a paid mutator transaction binding the contract method 0xb015813b.
//
// Solidity: function activateAgent(bytes32 agentId) returns()
func (_AgentCardRegistry *AgentCardRegistrySession) ActivateAgent(agentId [32]byte) (*types.Transaction, error) {
	return _AgentCardRegistry.Contract.ActivateAgent(&_AgentCardRegistry.TransactOpts, agentId)
}

// ActivateAgent is a paid mutator transaction binding the contract method 0xb015813b.
//
// Solidity: function activateAgent(bytes32 agentId) returns()
func (_AgentCardRegistry *AgentCardRegistryTransactorSession) ActivateAgent(agentId [32]byte) (*types.Transaction, error) {
	return _AgentCardRegistry.Contract.ActivateAgent(&_AgentCardRegistry.TransactOpts, agentId)
}

// AddKey is a paid mutator transaction binding the contract method 0xbe624bad.
//
// Solidity: function addKey(bytes32 agentId, bytes keyData, uint8 keyType, bytes signature) returns()
func (_AgentCardRegistry *AgentCardRegistryTransactor) AddKey(opts *bind.TransactOpts, agentId [32]byte, keyData []byte, keyType uint8, signature []byte) (*types.Transaction, error) {
	return _AgentCardRegistry.contract.Transact(opts, "addKey", agentId, keyData, keyType, signature)
}

// AddKey is a paid mutator transaction binding the contract method 0xbe624bad.
//
// Solidity: function addKey(bytes32 agentId, bytes keyData, uint8 keyType, bytes signature) returns()
func (_AgentCardRegistry *AgentCardRegistrySession) AddKey(agentId [32]byte, keyData []byte, keyType uint8, signature []byte) (*types.Transaction, error) {
	return _AgentCardRegistry.Contract.AddKey(&_AgentCardRegistry.TransactOpts, agentId, keyData, keyType, signature)
}

// AddKey is a paid mutator transaction binding the contract method 0xbe624bad.
//
// Solidity: function addKey(bytes32 agentId, bytes keyData, uint8 keyType, bytes signature) returns()
func (_AgentCardRegistry *AgentCardRegistryTransactorSession) AddKey(agentId [32]byte, keyData []byte, keyType uint8, signature []byte) (*types.Transaction, error) {
	return _AgentCardRegistry.Contract.AddKey(&_AgentCardRegistry.TransactOpts, agentId, keyData, keyType, signature)
}

// CommitRegistration is a paid mutator transaction binding the contract method 0xc43b5b1e.
//
// Solidity: function commitRegistration(bytes32 commitHash) payable returns()
func (_AgentCardRegistry *AgentCardRegistryTransactor) CommitRegistration(opts *bind.TransactOpts, commitHash [32]byte) (*types.Transaction, error) {
	return _AgentCardRegistry.contract.Transact(opts, "commitRegistration", commitHash)
}

// CommitRegistration is a paid mutator transaction binding the contract method 0xc43b5b1e.
//
// Solidity: function commitRegistration(bytes32 commitHash) payable returns()
func (_AgentCardRegistry *AgentCardRegistrySession) CommitRegistration(commitHash [32]byte) (*types.Transaction, error) {
	return _AgentCardRegistry.Contract.CommitRegistration(&_AgentCardRegistry.TransactOpts, commitHash)
}

// CommitRegistration is a paid mutator transaction binding the contract method 0xc43b5b1e.
//
// Solidity: function commitRegistration(bytes32 commitHash) payable returns()
func (_AgentCardRegistry *AgentCardRegistryTransactorSession) CommitRegistration(commitHash [32]byte) (*types.Transaction, error) {
	return _AgentCardRegistry.Contract.CommitRegistration(&_AgentCardRegistry.TransactOpts, commitHash)
}

// DeactivateAgent is a paid mutator transaction binding the contract method 0x18697b75.
//
// Solidity: function deactivateAgent(string agentId) returns(bool success)
func (_AgentCardRegistry *AgentCardRegistryTransactor) DeactivateAgent(opts *bind.TransactOpts, agentId string) (*types.Transaction, error) {
	return _AgentCardRegistry.contract.Transact(opts, "deactivateAgent", agentId)
}

// DeactivateAgent is a paid mutator transaction binding the contract method 0x18697b75.
//
// Solidity: function deactivateAgent(string agentId) returns(bool success)
func (_AgentCardRegistry *AgentCardRegistrySession) DeactivateAgent(agentId string) (*types.Transaction, error) {
	return _AgentCardRegistry.Contract.DeactivateAgent(&_AgentCardRegistry.TransactOpts, agentId)
}

// DeactivateAgent is a paid mutator transaction binding the contract method 0x18697b75.
//
// Solidity: function deactivateAgent(string agentId) returns(bool success)
func (_AgentCardRegistry *AgentCardRegistryTransactorSession) DeactivateAgent(agentId string) (*types.Transaction, error) {
	return _AgentCardRegistry.Contract.DeactivateAgent(&_AgentCardRegistry.TransactOpts, agentId)
}

// DeactivateAgentByHash is a paid mutator transaction binding the contract method 0x019e639f.
//
// Solidity: function deactivateAgentByHash(bytes32 agentId) returns()
func (_AgentCardRegistry *AgentCardRegistryTransactor) DeactivateAgentByHash(opts *bind.TransactOpts, agentId [32]byte) (*types.Transaction, error) {
	return _AgentCardRegistry.contract.Transact(opts, "deactivateAgentByHash", agentId)
}

// DeactivateAgentByHash is a paid mutator transaction binding the contract method 0x019e639f.
//
// Solidity: function deactivateAgentByHash(bytes32 agentId) returns()
func (_AgentCardRegistry *AgentCardRegistrySession) DeactivateAgentByHash(agentId [32]byte) (*types.Transaction, error) {
	return _AgentCardRegistry.Contract.DeactivateAgentByHash(&_AgentCardRegistry.TransactOpts, agentId)
}

// DeactivateAgentByHash is a paid mutator transaction binding the contract method 0x019e639f.
//
// Solidity: function deactivateAgentByHash(bytes32 agentId) returns()
func (_AgentCardRegistry *AgentCardRegistryTransactorSession) DeactivateAgentByHash(agentId [32]byte) (*types.Transaction, error) {
	return _AgentCardRegistry.Contract.DeactivateAgentByHash(&_AgentCardRegistry.TransactOpts, agentId)
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_AgentCardRegistry *AgentCardRegistryTransactor) Pause(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AgentCardRegistry.contract.Transact(opts, "pause")
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_AgentCardRegistry *AgentCardRegistrySession) Pause() (*types.Transaction, error) {
	return _AgentCardRegistry.Contract.Pause(&_AgentCardRegistry.TransactOpts)
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_AgentCardRegistry *AgentCardRegistryTransactorSession) Pause() (*types.Transaction, error) {
	return _AgentCardRegistry.Contract.Pause(&_AgentCardRegistry.TransactOpts)
}

// RegisterAgent is a paid mutator transaction binding the contract method 0xff17aca4.
//
// Solidity: function registerAgent(string agentId, string endpoint) returns(bool success)
func (_AgentCardRegistry *AgentCardRegistryTransactor) RegisterAgent(opts *bind.TransactOpts, agentId string, endpoint string) (*types.Transaction, error) {
	return _AgentCardRegistry.contract.Transact(opts, "registerAgent", agentId, endpoint)
}

// RegisterAgent is a paid mutator transaction binding the contract method 0xff17aca4.
//
// Solidity: function registerAgent(string agentId, string endpoint) returns(bool success)
func (_AgentCardRegistry *AgentCardRegistrySession) RegisterAgent(agentId string, endpoint string) (*types.Transaction, error) {
	return _AgentCardRegistry.Contract.RegisterAgent(&_AgentCardRegistry.TransactOpts, agentId, endpoint)
}

// RegisterAgent is a paid mutator transaction binding the contract method 0xff17aca4.
//
// Solidity: function registerAgent(string agentId, string endpoint) returns(bool success)
func (_AgentCardRegistry *AgentCardRegistryTransactorSession) RegisterAgent(agentId string, endpoint string) (*types.Transaction, error) {
	return _AgentCardRegistry.Contract.RegisterAgent(&_AgentCardRegistry.TransactOpts, agentId, endpoint)
}

// RegisterAgentWithParams is a paid mutator transaction binding the contract method 0x03528bf8.
//
// Solidity: function registerAgentWithParams((string,string,string,string,string,bytes[],uint8[],bytes[],bytes32) params) returns(bytes32 agentId)
func (_AgentCardRegistry *AgentCardRegistryTransactor) RegisterAgentWithParams(opts *bind.TransactOpts, params AgentCardStorageRegistrationParams) (*types.Transaction, error) {
	return _AgentCardRegistry.contract.Transact(opts, "registerAgentWithParams", params)
}

// RegisterAgentWithParams is a paid mutator transaction binding the contract method 0x03528bf8.
//
// Solidity: function registerAgentWithParams((string,string,string,string,string,bytes[],uint8[],bytes[],bytes32) params) returns(bytes32 agentId)
func (_AgentCardRegistry *AgentCardRegistrySession) RegisterAgentWithParams(params AgentCardStorageRegistrationParams) (*types.Transaction, error) {
	return _AgentCardRegistry.Contract.RegisterAgentWithParams(&_AgentCardRegistry.TransactOpts, params)
}

// RegisterAgentWithParams is a paid mutator transaction binding the contract method 0x03528bf8.
//
// Solidity: function registerAgentWithParams((string,string,string,string,string,bytes[],uint8[],bytes[],bytes32) params) returns(bytes32 agentId)
func (_AgentCardRegistry *AgentCardRegistryTransactorSession) RegisterAgentWithParams(params AgentCardStorageRegistrationParams) (*types.Transaction, error) {
	return _AgentCardRegistry.Contract.RegisterAgentWithParams(&_AgentCardRegistry.TransactOpts, params)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_AgentCardRegistry *AgentCardRegistryTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AgentCardRegistry.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_AgentCardRegistry *AgentCardRegistrySession) RenounceOwnership() (*types.Transaction, error) {
	return _AgentCardRegistry.Contract.RenounceOwnership(&_AgentCardRegistry.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_AgentCardRegistry *AgentCardRegistryTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _AgentCardRegistry.Contract.RenounceOwnership(&_AgentCardRegistry.TransactOpts)
}

// RevokeKey is a paid mutator transaction binding the contract method 0x1a9cb151.
//
// Solidity: function revokeKey(bytes32 agentId, bytes32 keyHash) returns()
func (_AgentCardRegistry *AgentCardRegistryTransactor) RevokeKey(opts *bind.TransactOpts, agentId [32]byte, keyHash [32]byte) (*types.Transaction, error) {
	return _AgentCardRegistry.contract.Transact(opts, "revokeKey", agentId, keyHash)
}

// RevokeKey is a paid mutator transaction binding the contract method 0x1a9cb151.
//
// Solidity: function revokeKey(bytes32 agentId, bytes32 keyHash) returns()
func (_AgentCardRegistry *AgentCardRegistrySession) RevokeKey(agentId [32]byte, keyHash [32]byte) (*types.Transaction, error) {
	return _AgentCardRegistry.Contract.RevokeKey(&_AgentCardRegistry.TransactOpts, agentId, keyHash)
}

// RevokeKey is a paid mutator transaction binding the contract method 0x1a9cb151.
//
// Solidity: function revokeKey(bytes32 agentId, bytes32 keyHash) returns()
func (_AgentCardRegistry *AgentCardRegistryTransactorSession) RevokeKey(agentId [32]byte, keyHash [32]byte) (*types.Transaction, error) {
	return _AgentCardRegistry.Contract.RevokeKey(&_AgentCardRegistry.TransactOpts, agentId, keyHash)
}

// SetActivationDelay is a paid mutator transaction binding the contract method 0x22d672d7.
//
// Solidity: function setActivationDelay(uint256 newDelay) returns()
func (_AgentCardRegistry *AgentCardRegistryTransactor) SetActivationDelay(opts *bind.TransactOpts, newDelay *big.Int) (*types.Transaction, error) {
	return _AgentCardRegistry.contract.Transact(opts, "setActivationDelay", newDelay)
}

// SetActivationDelay is a paid mutator transaction binding the contract method 0x22d672d7.
//
// Solidity: function setActivationDelay(uint256 newDelay) returns()
func (_AgentCardRegistry *AgentCardRegistrySession) SetActivationDelay(newDelay *big.Int) (*types.Transaction, error) {
	return _AgentCardRegistry.Contract.SetActivationDelay(&_AgentCardRegistry.TransactOpts, newDelay)
}

// SetActivationDelay is a paid mutator transaction binding the contract method 0x22d672d7.
//
// Solidity: function setActivationDelay(uint256 newDelay) returns()
func (_AgentCardRegistry *AgentCardRegistryTransactorSession) SetActivationDelay(newDelay *big.Int) (*types.Transaction, error) {
	return _AgentCardRegistry.Contract.SetActivationDelay(&_AgentCardRegistry.TransactOpts, newDelay)
}

// SetApprovalForAgent is a paid mutator transaction binding the contract method 0xc16bf107.
//
// Solidity: function setApprovalForAgent(bytes32 agentId, address operator, bool approved) returns()
func (_AgentCardRegistry *AgentCardRegistryTransactor) SetApprovalForAgent(opts *bind.TransactOpts, agentId [32]byte, operator common.Address, approved bool) (*types.Transaction, error) {
	return _AgentCardRegistry.contract.Transact(opts, "setApprovalForAgent", agentId, operator, approved)
}

// SetApprovalForAgent is a paid mutator transaction binding the contract method 0xc16bf107.
//
// Solidity: function setApprovalForAgent(bytes32 agentId, address operator, bool approved) returns()
func (_AgentCardRegistry *AgentCardRegistrySession) SetApprovalForAgent(agentId [32]byte, operator common.Address, approved bool) (*types.Transaction, error) {
	return _AgentCardRegistry.Contract.SetApprovalForAgent(&_AgentCardRegistry.TransactOpts, agentId, operator, approved)
}

// SetApprovalForAgent is a paid mutator transaction binding the contract method 0xc16bf107.
//
// Solidity: function setApprovalForAgent(bytes32 agentId, address operator, bool approved) returns()
func (_AgentCardRegistry *AgentCardRegistryTransactorSession) SetApprovalForAgent(agentId [32]byte, operator common.Address, approved bool) (*types.Transaction, error) {
	return _AgentCardRegistry.Contract.SetApprovalForAgent(&_AgentCardRegistry.TransactOpts, agentId, operator, approved)
}

// SetRegistrationStake is a paid mutator transaction binding the contract method 0xe85a43ab.
//
// Solidity: function setRegistrationStake(uint256 newStake) returns()
func (_AgentCardRegistry *AgentCardRegistryTransactor) SetRegistrationStake(opts *bind.TransactOpts, newStake *big.Int) (*types.Transaction, error) {
	return _AgentCardRegistry.contract.Transact(opts, "setRegistrationStake", newStake)
}

// SetRegistrationStake is a paid mutator transaction binding the contract method 0xe85a43ab.
//
// Solidity: function setRegistrationStake(uint256 newStake) returns()
func (_AgentCardRegistry *AgentCardRegistrySession) SetRegistrationStake(newStake *big.Int) (*types.Transaction, error) {
	return _AgentCardRegistry.Contract.SetRegistrationStake(&_AgentCardRegistry.TransactOpts, newStake)
}

// SetRegistrationStake is a paid mutator transaction binding the contract method 0xe85a43ab.
//
// Solidity: function setRegistrationStake(uint256 newStake) returns()
func (_AgentCardRegistry *AgentCardRegistryTransactorSession) SetRegistrationStake(newStake *big.Int) (*types.Transaction, error) {
	return _AgentCardRegistry.Contract.SetRegistrationStake(&_AgentCardRegistry.TransactOpts, newStake)
}

// SetVerifyHook is a paid mutator transaction binding the contract method 0xd6b42db2.
//
// Solidity: function setVerifyHook(address newHook) returns()
func (_AgentCardRegistry *AgentCardRegistryTransactor) SetVerifyHook(opts *bind.TransactOpts, newHook common.Address) (*types.Transaction, error) {
	return _AgentCardRegistry.contract.Transact(opts, "setVerifyHook", newHook)
}

// SetVerifyHook is a paid mutator transaction binding the contract method 0xd6b42db2.
//
// Solidity: function setVerifyHook(address newHook) returns()
func (_AgentCardRegistry *AgentCardRegistrySession) SetVerifyHook(newHook common.Address) (*types.Transaction, error) {
	return _AgentCardRegistry.Contract.SetVerifyHook(&_AgentCardRegistry.TransactOpts, newHook)
}

// SetVerifyHook is a paid mutator transaction binding the contract method 0xd6b42db2.
//
// Solidity: function setVerifyHook(address newHook) returns()
func (_AgentCardRegistry *AgentCardRegistryTransactorSession) SetVerifyHook(newHook common.Address) (*types.Transaction, error) {
	return _AgentCardRegistry.Contract.SetVerifyHook(&_AgentCardRegistry.TransactOpts, newHook)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_AgentCardRegistry *AgentCardRegistryTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _AgentCardRegistry.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_AgentCardRegistry *AgentCardRegistrySession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _AgentCardRegistry.Contract.TransferOwnership(&_AgentCardRegistry.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_AgentCardRegistry *AgentCardRegistryTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _AgentCardRegistry.Contract.TransferOwnership(&_AgentCardRegistry.TransactOpts, newOwner)
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_AgentCardRegistry *AgentCardRegistryTransactor) Unpause(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AgentCardRegistry.contract.Transact(opts, "unpause")
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_AgentCardRegistry *AgentCardRegistrySession) Unpause() (*types.Transaction, error) {
	return _AgentCardRegistry.Contract.Unpause(&_AgentCardRegistry.TransactOpts)
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_AgentCardRegistry *AgentCardRegistryTransactorSession) Unpause() (*types.Transaction, error) {
	return _AgentCardRegistry.Contract.Unpause(&_AgentCardRegistry.TransactOpts)
}

// UpdateAgent is a paid mutator transaction binding the contract method 0xa13b7e1f.
//
// Solidity: function updateAgent(bytes32 agentId, string endpoint, string capabilities) returns()
func (_AgentCardRegistry *AgentCardRegistryTransactor) UpdateAgent(opts *bind.TransactOpts, agentId [32]byte, endpoint string, capabilities string) (*types.Transaction, error) {
	return _AgentCardRegistry.contract.Transact(opts, "updateAgent", agentId, endpoint, capabilities)
}

// UpdateAgent is a paid mutator transaction binding the contract method 0xa13b7e1f.
//
// Solidity: function updateAgent(bytes32 agentId, string endpoint, string capabilities) returns()
func (_AgentCardRegistry *AgentCardRegistrySession) UpdateAgent(agentId [32]byte, endpoint string, capabilities string) (*types.Transaction, error) {
	return _AgentCardRegistry.Contract.UpdateAgent(&_AgentCardRegistry.TransactOpts, agentId, endpoint, capabilities)
}

// UpdateAgent is a paid mutator transaction binding the contract method 0xa13b7e1f.
//
// Solidity: function updateAgent(bytes32 agentId, string endpoint, string capabilities) returns()
func (_AgentCardRegistry *AgentCardRegistryTransactorSession) UpdateAgent(agentId [32]byte, endpoint string, capabilities string) (*types.Transaction, error) {
	return _AgentCardRegistry.Contract.UpdateAgent(&_AgentCardRegistry.TransactOpts, agentId, endpoint, capabilities)
}

// UpdateAgentEndpoint is a paid mutator transaction binding the contract method 0x2511a19e.
//
// Solidity: function updateAgentEndpoint(string agentId, string newEndpoint) returns(bool success)
func (_AgentCardRegistry *AgentCardRegistryTransactor) UpdateAgentEndpoint(opts *bind.TransactOpts, agentId string, newEndpoint string) (*types.Transaction, error) {
	return _AgentCardRegistry.contract.Transact(opts, "updateAgentEndpoint", agentId, newEndpoint)
}

// UpdateAgentEndpoint is a paid mutator transaction binding the contract method 0x2511a19e.
//
// Solidity: function updateAgentEndpoint(string agentId, string newEndpoint) returns(bool success)
func (_AgentCardRegistry *AgentCardRegistrySession) UpdateAgentEndpoint(agentId string, newEndpoint string) (*types.Transaction, error) {
	return _AgentCardRegistry.Contract.UpdateAgentEndpoint(&_AgentCardRegistry.TransactOpts, agentId, newEndpoint)
}

// UpdateAgentEndpoint is a paid mutator transaction binding the contract method 0x2511a19e.
//
// Solidity: function updateAgentEndpoint(string agentId, string newEndpoint) returns(bool success)
func (_AgentCardRegistry *AgentCardRegistryTransactorSession) UpdateAgentEndpoint(agentId string, newEndpoint string) (*types.Transaction, error) {
	return _AgentCardRegistry.Contract.UpdateAgentEndpoint(&_AgentCardRegistry.TransactOpts, agentId, newEndpoint)
}

// UpdateKMEKey is a paid mutator transaction binding the contract method 0x32d5d493.
//
// Solidity: function updateKMEKey(bytes32 agentId, bytes newKmeKey, bytes signature) returns()
func (_AgentCardRegistry *AgentCardRegistryTransactor) UpdateKMEKey(opts *bind.TransactOpts, agentId [32]byte, newKmeKey []byte, signature []byte) (*types.Transaction, error) {
	return _AgentCardRegistry.contract.Transact(opts, "updateKMEKey", agentId, newKmeKey, signature)
}

// UpdateKMEKey is a paid mutator transaction binding the contract method 0x32d5d493.
//
// Solidity: function updateKMEKey(bytes32 agentId, bytes newKmeKey, bytes signature) returns()
func (_AgentCardRegistry *AgentCardRegistrySession) UpdateKMEKey(agentId [32]byte, newKmeKey []byte, signature []byte) (*types.Transaction, error) {
	return _AgentCardRegistry.Contract.UpdateKMEKey(&_AgentCardRegistry.TransactOpts, agentId, newKmeKey, signature)
}

// UpdateKMEKey is a paid mutator transaction binding the contract method 0x32d5d493.
//
// Solidity: function updateKMEKey(bytes32 agentId, bytes newKmeKey, bytes signature) returns()
func (_AgentCardRegistry *AgentCardRegistryTransactorSession) UpdateKMEKey(agentId [32]byte, newKmeKey []byte, signature []byte) (*types.Transaction, error) {
	return _AgentCardRegistry.Contract.UpdateKMEKey(&_AgentCardRegistry.TransactOpts, agentId, newKmeKey, signature)
}

// AgentCardRegistryAgentActivatedIterator is returned from FilterAgentActivated and is used to iterate over the raw logs and unpacked data for AgentActivated events raised by the AgentCardRegistry contract.
type AgentCardRegistryAgentActivatedIterator struct {
	Event *AgentCardRegistryAgentActivated // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *AgentCardRegistryAgentActivatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AgentCardRegistryAgentActivated)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(AgentCardRegistryAgentActivated)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *AgentCardRegistryAgentActivatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AgentCardRegistryAgentActivatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AgentCardRegistryAgentActivated represents a AgentActivated event raised by the AgentCardRegistry contract.
type AgentCardRegistryAgentActivated struct {
	AgentId   [32]byte
	Timestamp *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterAgentActivated is a free log retrieval operation binding the contract event 0x0865b389859e9da765f37427076dccde87d891446c7283dade7653efa4ca133b.
//
// Solidity: event AgentActivated(bytes32 indexed agentId, uint256 timestamp)
func (_AgentCardRegistry *AgentCardRegistryFilterer) FilterAgentActivated(opts *bind.FilterOpts, agentId [][32]byte) (*AgentCardRegistryAgentActivatedIterator, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}

	logs, sub, err := _AgentCardRegistry.contract.FilterLogs(opts, "AgentActivated", agentIdRule)
	if err != nil {
		return nil, err
	}
	return &AgentCardRegistryAgentActivatedIterator{contract: _AgentCardRegistry.contract, event: "AgentActivated", logs: logs, sub: sub}, nil
}

// WatchAgentActivated is a free log subscription operation binding the contract event 0x0865b389859e9da765f37427076dccde87d891446c7283dade7653efa4ca133b.
//
// Solidity: event AgentActivated(bytes32 indexed agentId, uint256 timestamp)
func (_AgentCardRegistry *AgentCardRegistryFilterer) WatchAgentActivated(opts *bind.WatchOpts, sink chan<- *AgentCardRegistryAgentActivated, agentId [][32]byte) (event.Subscription, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}

	logs, sub, err := _AgentCardRegistry.contract.WatchLogs(opts, "AgentActivated", agentIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AgentCardRegistryAgentActivated)
				if err := _AgentCardRegistry.contract.UnpackLog(event, "AgentActivated", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseAgentActivated is a log parse operation binding the contract event 0x0865b389859e9da765f37427076dccde87d891446c7283dade7653efa4ca133b.
//
// Solidity: event AgentActivated(bytes32 indexed agentId, uint256 timestamp)
func (_AgentCardRegistry *AgentCardRegistryFilterer) ParseAgentActivated(log types.Log) (*AgentCardRegistryAgentActivated, error) {
	event := new(AgentCardRegistryAgentActivated)
	if err := _AgentCardRegistry.contract.UnpackLog(event, "AgentActivated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AgentCardRegistryAgentDeactivatedIterator is returned from FilterAgentDeactivated and is used to iterate over the raw logs and unpacked data for AgentDeactivated events raised by the AgentCardRegistry contract.
type AgentCardRegistryAgentDeactivatedIterator struct {
	Event *AgentCardRegistryAgentDeactivated // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *AgentCardRegistryAgentDeactivatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AgentCardRegistryAgentDeactivated)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(AgentCardRegistryAgentDeactivated)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *AgentCardRegistryAgentDeactivatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AgentCardRegistryAgentDeactivatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AgentCardRegistryAgentDeactivated represents a AgentDeactivated event raised by the AgentCardRegistry contract.
type AgentCardRegistryAgentDeactivated struct {
	AgentId      common.Hash
	AgentAddress common.Address
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterAgentDeactivated is a free log retrieval operation binding the contract event 0x93a9b474019118d48d48eb70cc1fe2337fec1a4cf516207432544a4065b7cc54.
//
// Solidity: event AgentDeactivated(string indexed agentId, address indexed agentAddress)
func (_AgentCardRegistry *AgentCardRegistryFilterer) FilterAgentDeactivated(opts *bind.FilterOpts, agentId []string, agentAddress []common.Address) (*AgentCardRegistryAgentDeactivatedIterator, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}
	var agentAddressRule []interface{}
	for _, agentAddressItem := range agentAddress {
		agentAddressRule = append(agentAddressRule, agentAddressItem)
	}

	logs, sub, err := _AgentCardRegistry.contract.FilterLogs(opts, "AgentDeactivated", agentIdRule, agentAddressRule)
	if err != nil {
		return nil, err
	}
	return &AgentCardRegistryAgentDeactivatedIterator{contract: _AgentCardRegistry.contract, event: "AgentDeactivated", logs: logs, sub: sub}, nil
}

// WatchAgentDeactivated is a free log subscription operation binding the contract event 0x93a9b474019118d48d48eb70cc1fe2337fec1a4cf516207432544a4065b7cc54.
//
// Solidity: event AgentDeactivated(string indexed agentId, address indexed agentAddress)
func (_AgentCardRegistry *AgentCardRegistryFilterer) WatchAgentDeactivated(opts *bind.WatchOpts, sink chan<- *AgentCardRegistryAgentDeactivated, agentId []string, agentAddress []common.Address) (event.Subscription, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}
	var agentAddressRule []interface{}
	for _, agentAddressItem := range agentAddress {
		agentAddressRule = append(agentAddressRule, agentAddressItem)
	}

	logs, sub, err := _AgentCardRegistry.contract.WatchLogs(opts, "AgentDeactivated", agentIdRule, agentAddressRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AgentCardRegistryAgentDeactivated)
				if err := _AgentCardRegistry.contract.UnpackLog(event, "AgentDeactivated", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseAgentDeactivated is a log parse operation binding the contract event 0x93a9b474019118d48d48eb70cc1fe2337fec1a4cf516207432544a4065b7cc54.
//
// Solidity: event AgentDeactivated(string indexed agentId, address indexed agentAddress)
func (_AgentCardRegistry *AgentCardRegistryFilterer) ParseAgentDeactivated(log types.Log) (*AgentCardRegistryAgentDeactivated, error) {
	event := new(AgentCardRegistryAgentDeactivated)
	if err := _AgentCardRegistry.contract.UnpackLog(event, "AgentDeactivated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AgentCardRegistryAgentDeactivatedByHashIterator is returned from FilterAgentDeactivatedByHash and is used to iterate over the raw logs and unpacked data for AgentDeactivatedByHash events raised by the AgentCardRegistry contract.
type AgentCardRegistryAgentDeactivatedByHashIterator struct {
	Event *AgentCardRegistryAgentDeactivatedByHash // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *AgentCardRegistryAgentDeactivatedByHashIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AgentCardRegistryAgentDeactivatedByHash)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(AgentCardRegistryAgentDeactivatedByHash)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *AgentCardRegistryAgentDeactivatedByHashIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AgentCardRegistryAgentDeactivatedByHashIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AgentCardRegistryAgentDeactivatedByHash represents a AgentDeactivatedByHash event raised by the AgentCardRegistry contract.
type AgentCardRegistryAgentDeactivatedByHash struct {
	AgentId   [32]byte
	Timestamp *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterAgentDeactivatedByHash is a free log retrieval operation binding the contract event 0xb744c2ed9952c7041822a155d9cf761a7456f4876bdf9df34784d690c8647684.
//
// Solidity: event AgentDeactivatedByHash(bytes32 indexed agentId, uint256 timestamp)
func (_AgentCardRegistry *AgentCardRegistryFilterer) FilterAgentDeactivatedByHash(opts *bind.FilterOpts, agentId [][32]byte) (*AgentCardRegistryAgentDeactivatedByHashIterator, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}

	logs, sub, err := _AgentCardRegistry.contract.FilterLogs(opts, "AgentDeactivatedByHash", agentIdRule)
	if err != nil {
		return nil, err
	}
	return &AgentCardRegistryAgentDeactivatedByHashIterator{contract: _AgentCardRegistry.contract, event: "AgentDeactivatedByHash", logs: logs, sub: sub}, nil
}

// WatchAgentDeactivatedByHash is a free log subscription operation binding the contract event 0xb744c2ed9952c7041822a155d9cf761a7456f4876bdf9df34784d690c8647684.
//
// Solidity: event AgentDeactivatedByHash(bytes32 indexed agentId, uint256 timestamp)
func (_AgentCardRegistry *AgentCardRegistryFilterer) WatchAgentDeactivatedByHash(opts *bind.WatchOpts, sink chan<- *AgentCardRegistryAgentDeactivatedByHash, agentId [][32]byte) (event.Subscription, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}

	logs, sub, err := _AgentCardRegistry.contract.WatchLogs(opts, "AgentDeactivatedByHash", agentIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AgentCardRegistryAgentDeactivatedByHash)
				if err := _AgentCardRegistry.contract.UnpackLog(event, "AgentDeactivatedByHash", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseAgentDeactivatedByHash is a log parse operation binding the contract event 0xb744c2ed9952c7041822a155d9cf761a7456f4876bdf9df34784d690c8647684.
//
// Solidity: event AgentDeactivatedByHash(bytes32 indexed agentId, uint256 timestamp)
func (_AgentCardRegistry *AgentCardRegistryFilterer) ParseAgentDeactivatedByHash(log types.Log) (*AgentCardRegistryAgentDeactivatedByHash, error) {
	event := new(AgentCardRegistryAgentDeactivatedByHash)
	if err := _AgentCardRegistry.contract.UnpackLog(event, "AgentDeactivatedByHash", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AgentCardRegistryAgentEndpointUpdatedIterator is returned from FilterAgentEndpointUpdated and is used to iterate over the raw logs and unpacked data for AgentEndpointUpdated events raised by the AgentCardRegistry contract.
type AgentCardRegistryAgentEndpointUpdatedIterator struct {
	Event *AgentCardRegistryAgentEndpointUpdated // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *AgentCardRegistryAgentEndpointUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AgentCardRegistryAgentEndpointUpdated)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(AgentCardRegistryAgentEndpointUpdated)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *AgentCardRegistryAgentEndpointUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AgentCardRegistryAgentEndpointUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AgentCardRegistryAgentEndpointUpdated represents a AgentEndpointUpdated event raised by the AgentCardRegistry contract.
type AgentCardRegistryAgentEndpointUpdated struct {
	AgentId     common.Hash
	OldEndpoint string
	NewEndpoint string
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterAgentEndpointUpdated is a free log retrieval operation binding the contract event 0xd0a8f62b0046d0af4a22b74ca9dd7d0ea2c775b754c10b914b5ae5e19c5dcfab.
//
// Solidity: event AgentEndpointUpdated(string indexed agentId, string oldEndpoint, string newEndpoint)
func (_AgentCardRegistry *AgentCardRegistryFilterer) FilterAgentEndpointUpdated(opts *bind.FilterOpts, agentId []string) (*AgentCardRegistryAgentEndpointUpdatedIterator, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}

	logs, sub, err := _AgentCardRegistry.contract.FilterLogs(opts, "AgentEndpointUpdated", agentIdRule)
	if err != nil {
		return nil, err
	}
	return &AgentCardRegistryAgentEndpointUpdatedIterator{contract: _AgentCardRegistry.contract, event: "AgentEndpointUpdated", logs: logs, sub: sub}, nil
}

// WatchAgentEndpointUpdated is a free log subscription operation binding the contract event 0xd0a8f62b0046d0af4a22b74ca9dd7d0ea2c775b754c10b914b5ae5e19c5dcfab.
//
// Solidity: event AgentEndpointUpdated(string indexed agentId, string oldEndpoint, string newEndpoint)
func (_AgentCardRegistry *AgentCardRegistryFilterer) WatchAgentEndpointUpdated(opts *bind.WatchOpts, sink chan<- *AgentCardRegistryAgentEndpointUpdated, agentId []string) (event.Subscription, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}

	logs, sub, err := _AgentCardRegistry.contract.WatchLogs(opts, "AgentEndpointUpdated", agentIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AgentCardRegistryAgentEndpointUpdated)
				if err := _AgentCardRegistry.contract.UnpackLog(event, "AgentEndpointUpdated", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseAgentEndpointUpdated is a log parse operation binding the contract event 0xd0a8f62b0046d0af4a22b74ca9dd7d0ea2c775b754c10b914b5ae5e19c5dcfab.
//
// Solidity: event AgentEndpointUpdated(string indexed agentId, string oldEndpoint, string newEndpoint)
func (_AgentCardRegistry *AgentCardRegistryFilterer) ParseAgentEndpointUpdated(log types.Log) (*AgentCardRegistryAgentEndpointUpdated, error) {
	event := new(AgentCardRegistryAgentEndpointUpdated)
	if err := _AgentCardRegistry.contract.UnpackLog(event, "AgentEndpointUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AgentCardRegistryAgentRegisteredIterator is returned from FilterAgentRegistered and is used to iterate over the raw logs and unpacked data for AgentRegistered events raised by the AgentCardRegistry contract.
type AgentCardRegistryAgentRegisteredIterator struct {
	Event *AgentCardRegistryAgentRegistered // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *AgentCardRegistryAgentRegisteredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AgentCardRegistryAgentRegistered)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(AgentCardRegistryAgentRegistered)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *AgentCardRegistryAgentRegisteredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AgentCardRegistryAgentRegisteredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AgentCardRegistryAgentRegistered represents a AgentRegistered event raised by the AgentCardRegistry contract.
type AgentCardRegistryAgentRegistered struct {
	AgentId   [32]byte
	Did       common.Hash
	Owner     common.Address
	Timestamp *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterAgentRegistered is a free log retrieval operation binding the contract event 0x1d1edfbe93d381ab532cea862c0ee3fde2e6e88803b77f675fe3946635608ade.
//
// Solidity: event AgentRegistered(bytes32 indexed agentId, string indexed did, address indexed owner, uint256 timestamp)
func (_AgentCardRegistry *AgentCardRegistryFilterer) FilterAgentRegistered(opts *bind.FilterOpts, agentId [][32]byte, did []string, owner []common.Address) (*AgentCardRegistryAgentRegisteredIterator, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}
	var didRule []interface{}
	for _, didItem := range did {
		didRule = append(didRule, didItem)
	}
	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _AgentCardRegistry.contract.FilterLogs(opts, "AgentRegistered", agentIdRule, didRule, ownerRule)
	if err != nil {
		return nil, err
	}
	return &AgentCardRegistryAgentRegisteredIterator{contract: _AgentCardRegistry.contract, event: "AgentRegistered", logs: logs, sub: sub}, nil
}

// WatchAgentRegistered is a free log subscription operation binding the contract event 0x1d1edfbe93d381ab532cea862c0ee3fde2e6e88803b77f675fe3946635608ade.
//
// Solidity: event AgentRegistered(bytes32 indexed agentId, string indexed did, address indexed owner, uint256 timestamp)
func (_AgentCardRegistry *AgentCardRegistryFilterer) WatchAgentRegistered(opts *bind.WatchOpts, sink chan<- *AgentCardRegistryAgentRegistered, agentId [][32]byte, did []string, owner []common.Address) (event.Subscription, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}
	var didRule []interface{}
	for _, didItem := range did {
		didRule = append(didRule, didItem)
	}
	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _AgentCardRegistry.contract.WatchLogs(opts, "AgentRegistered", agentIdRule, didRule, ownerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AgentCardRegistryAgentRegistered)
				if err := _AgentCardRegistry.contract.UnpackLog(event, "AgentRegistered", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseAgentRegistered is a log parse operation binding the contract event 0x1d1edfbe93d381ab532cea862c0ee3fde2e6e88803b77f675fe3946635608ade.
//
// Solidity: event AgentRegistered(bytes32 indexed agentId, string indexed did, address indexed owner, uint256 timestamp)
func (_AgentCardRegistry *AgentCardRegistryFilterer) ParseAgentRegistered(log types.Log) (*AgentCardRegistryAgentRegistered, error) {
	event := new(AgentCardRegistryAgentRegistered)
	if err := _AgentCardRegistry.contract.UnpackLog(event, "AgentRegistered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AgentCardRegistryAgentRegistered0Iterator is returned from FilterAgentRegistered0 and is used to iterate over the raw logs and unpacked data for AgentRegistered0 events raised by the AgentCardRegistry contract.
type AgentCardRegistryAgentRegistered0Iterator struct {
	Event *AgentCardRegistryAgentRegistered0 // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *AgentCardRegistryAgentRegistered0Iterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AgentCardRegistryAgentRegistered0)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(AgentCardRegistryAgentRegistered0)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *AgentCardRegistryAgentRegistered0Iterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AgentCardRegistryAgentRegistered0Iterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AgentCardRegistryAgentRegistered0 represents a AgentRegistered0 event raised by the AgentCardRegistry contract.
type AgentCardRegistryAgentRegistered0 struct {
	AgentId      common.Hash
	AgentAddress common.Address
	Endpoint     string
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterAgentRegistered0 is a free log retrieval operation binding the contract event 0xc43f3529413bf4994b5fe9d3ce632ca9fb8a6bb4e9c0662994bb7994f1f5b5e8.
//
// Solidity: event AgentRegistered(string indexed agentId, address indexed agentAddress, string endpoint)
func (_AgentCardRegistry *AgentCardRegistryFilterer) FilterAgentRegistered0(opts *bind.FilterOpts, agentId []string, agentAddress []common.Address) (*AgentCardRegistryAgentRegistered0Iterator, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}
	var agentAddressRule []interface{}
	for _, agentAddressItem := range agentAddress {
		agentAddressRule = append(agentAddressRule, agentAddressItem)
	}

	logs, sub, err := _AgentCardRegistry.contract.FilterLogs(opts, "AgentRegistered0", agentIdRule, agentAddressRule)
	if err != nil {
		return nil, err
	}
	return &AgentCardRegistryAgentRegistered0Iterator{contract: _AgentCardRegistry.contract, event: "AgentRegistered0", logs: logs, sub: sub}, nil
}

// WatchAgentRegistered0 is a free log subscription operation binding the contract event 0xc43f3529413bf4994b5fe9d3ce632ca9fb8a6bb4e9c0662994bb7994f1f5b5e8.
//
// Solidity: event AgentRegistered(string indexed agentId, address indexed agentAddress, string endpoint)
func (_AgentCardRegistry *AgentCardRegistryFilterer) WatchAgentRegistered0(opts *bind.WatchOpts, sink chan<- *AgentCardRegistryAgentRegistered0, agentId []string, agentAddress []common.Address) (event.Subscription, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}
	var agentAddressRule []interface{}
	for _, agentAddressItem := range agentAddress {
		agentAddressRule = append(agentAddressRule, agentAddressItem)
	}

	logs, sub, err := _AgentCardRegistry.contract.WatchLogs(opts, "AgentRegistered0", agentIdRule, agentAddressRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AgentCardRegistryAgentRegistered0)
				if err := _AgentCardRegistry.contract.UnpackLog(event, "AgentRegistered0", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseAgentRegistered0 is a log parse operation binding the contract event 0xc43f3529413bf4994b5fe9d3ce632ca9fb8a6bb4e9c0662994bb7994f1f5b5e8.
//
// Solidity: event AgentRegistered(string indexed agentId, address indexed agentAddress, string endpoint)
func (_AgentCardRegistry *AgentCardRegistryFilterer) ParseAgentRegistered0(log types.Log) (*AgentCardRegistryAgentRegistered0, error) {
	event := new(AgentCardRegistryAgentRegistered0)
	if err := _AgentCardRegistry.contract.UnpackLog(event, "AgentRegistered0", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AgentCardRegistryAgentUpdatedIterator is returned from FilterAgentUpdated and is used to iterate over the raw logs and unpacked data for AgentUpdated events raised by the AgentCardRegistry contract.
type AgentCardRegistryAgentUpdatedIterator struct {
	Event *AgentCardRegistryAgentUpdated // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *AgentCardRegistryAgentUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AgentCardRegistryAgentUpdated)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(AgentCardRegistryAgentUpdated)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *AgentCardRegistryAgentUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AgentCardRegistryAgentUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AgentCardRegistryAgentUpdated represents a AgentUpdated event raised by the AgentCardRegistry contract.
type AgentCardRegistryAgentUpdated struct {
	AgentId   [32]byte
	Timestamp *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterAgentUpdated is a free log retrieval operation binding the contract event 0x7a8c7d2cea9391cb6922e32d2c81a85e5b2307519a0f23f37665800328e42253.
//
// Solidity: event AgentUpdated(bytes32 indexed agentId, uint256 timestamp)
func (_AgentCardRegistry *AgentCardRegistryFilterer) FilterAgentUpdated(opts *bind.FilterOpts, agentId [][32]byte) (*AgentCardRegistryAgentUpdatedIterator, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}

	logs, sub, err := _AgentCardRegistry.contract.FilterLogs(opts, "AgentUpdated", agentIdRule)
	if err != nil {
		return nil, err
	}
	return &AgentCardRegistryAgentUpdatedIterator{contract: _AgentCardRegistry.contract, event: "AgentUpdated", logs: logs, sub: sub}, nil
}

// WatchAgentUpdated is a free log subscription operation binding the contract event 0x7a8c7d2cea9391cb6922e32d2c81a85e5b2307519a0f23f37665800328e42253.
//
// Solidity: event AgentUpdated(bytes32 indexed agentId, uint256 timestamp)
func (_AgentCardRegistry *AgentCardRegistryFilterer) WatchAgentUpdated(opts *bind.WatchOpts, sink chan<- *AgentCardRegistryAgentUpdated, agentId [][32]byte) (event.Subscription, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}

	logs, sub, err := _AgentCardRegistry.contract.WatchLogs(opts, "AgentUpdated", agentIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AgentCardRegistryAgentUpdated)
				if err := _AgentCardRegistry.contract.UnpackLog(event, "AgentUpdated", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseAgentUpdated is a log parse operation binding the contract event 0x7a8c7d2cea9391cb6922e32d2c81a85e5b2307519a0f23f37665800328e42253.
//
// Solidity: event AgentUpdated(bytes32 indexed agentId, uint256 timestamp)
func (_AgentCardRegistry *AgentCardRegistryFilterer) ParseAgentUpdated(log types.Log) (*AgentCardRegistryAgentUpdated, error) {
	event := new(AgentCardRegistryAgentUpdated)
	if err := _AgentCardRegistry.contract.UnpackLog(event, "AgentUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AgentCardRegistryApprovalForAgentIterator is returned from FilterApprovalForAgent and is used to iterate over the raw logs and unpacked data for ApprovalForAgent events raised by the AgentCardRegistry contract.
type AgentCardRegistryApprovalForAgentIterator struct {
	Event *AgentCardRegistryApprovalForAgent // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *AgentCardRegistryApprovalForAgentIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AgentCardRegistryApprovalForAgent)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(AgentCardRegistryApprovalForAgent)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *AgentCardRegistryApprovalForAgentIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AgentCardRegistryApprovalForAgentIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AgentCardRegistryApprovalForAgent represents a ApprovalForAgent event raised by the AgentCardRegistry contract.
type AgentCardRegistryApprovalForAgent struct {
	AgentId  [32]byte
	Owner    common.Address
	Operator common.Address
	Approved bool
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterApprovalForAgent is a free log retrieval operation binding the contract event 0x5f0fcbccc201a658d8ff8a6739186b23cce22dd2c40436fbd54d325bf28b1983.
//
// Solidity: event ApprovalForAgent(bytes32 indexed agentId, address indexed owner, address indexed operator, bool approved)
func (_AgentCardRegistry *AgentCardRegistryFilterer) FilterApprovalForAgent(opts *bind.FilterOpts, agentId [][32]byte, owner []common.Address, operator []common.Address) (*AgentCardRegistryApprovalForAgentIterator, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}
	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var operatorRule []interface{}
	for _, operatorItem := range operator {
		operatorRule = append(operatorRule, operatorItem)
	}

	logs, sub, err := _AgentCardRegistry.contract.FilterLogs(opts, "ApprovalForAgent", agentIdRule, ownerRule, operatorRule)
	if err != nil {
		return nil, err
	}
	return &AgentCardRegistryApprovalForAgentIterator{contract: _AgentCardRegistry.contract, event: "ApprovalForAgent", logs: logs, sub: sub}, nil
}

// WatchApprovalForAgent is a free log subscription operation binding the contract event 0x5f0fcbccc201a658d8ff8a6739186b23cce22dd2c40436fbd54d325bf28b1983.
//
// Solidity: event ApprovalForAgent(bytes32 indexed agentId, address indexed owner, address indexed operator, bool approved)
func (_AgentCardRegistry *AgentCardRegistryFilterer) WatchApprovalForAgent(opts *bind.WatchOpts, sink chan<- *AgentCardRegistryApprovalForAgent, agentId [][32]byte, owner []common.Address, operator []common.Address) (event.Subscription, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}
	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var operatorRule []interface{}
	for _, operatorItem := range operator {
		operatorRule = append(operatorRule, operatorItem)
	}

	logs, sub, err := _AgentCardRegistry.contract.WatchLogs(opts, "ApprovalForAgent", agentIdRule, ownerRule, operatorRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AgentCardRegistryApprovalForAgent)
				if err := _AgentCardRegistry.contract.UnpackLog(event, "ApprovalForAgent", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseApprovalForAgent is a log parse operation binding the contract event 0x5f0fcbccc201a658d8ff8a6739186b23cce22dd2c40436fbd54d325bf28b1983.
//
// Solidity: event ApprovalForAgent(bytes32 indexed agentId, address indexed owner, address indexed operator, bool approved)
func (_AgentCardRegistry *AgentCardRegistryFilterer) ParseApprovalForAgent(log types.Log) (*AgentCardRegistryApprovalForAgent, error) {
	event := new(AgentCardRegistryApprovalForAgent)
	if err := _AgentCardRegistry.contract.UnpackLog(event, "ApprovalForAgent", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AgentCardRegistryCommitmentRecordedIterator is returned from FilterCommitmentRecorded and is used to iterate over the raw logs and unpacked data for CommitmentRecorded events raised by the AgentCardRegistry contract.
type AgentCardRegistryCommitmentRecordedIterator struct {
	Event *AgentCardRegistryCommitmentRecorded // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *AgentCardRegistryCommitmentRecordedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AgentCardRegistryCommitmentRecorded)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(AgentCardRegistryCommitmentRecorded)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *AgentCardRegistryCommitmentRecordedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AgentCardRegistryCommitmentRecordedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AgentCardRegistryCommitmentRecorded represents a CommitmentRecorded event raised by the AgentCardRegistry contract.
type AgentCardRegistryCommitmentRecorded struct {
	Committer  common.Address
	CommitHash [32]byte
	Timestamp  *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterCommitmentRecorded is a free log retrieval operation binding the contract event 0xf76059fd91b15b2d465b41fe5d794955f8ac948e38e126713fbfb120585ff6bc.
//
// Solidity: event CommitmentRecorded(address indexed committer, bytes32 commitHash, uint256 timestamp)
func (_AgentCardRegistry *AgentCardRegistryFilterer) FilterCommitmentRecorded(opts *bind.FilterOpts, committer []common.Address) (*AgentCardRegistryCommitmentRecordedIterator, error) {

	var committerRule []interface{}
	for _, committerItem := range committer {
		committerRule = append(committerRule, committerItem)
	}

	logs, sub, err := _AgentCardRegistry.contract.FilterLogs(opts, "CommitmentRecorded", committerRule)
	if err != nil {
		return nil, err
	}
	return &AgentCardRegistryCommitmentRecordedIterator{contract: _AgentCardRegistry.contract, event: "CommitmentRecorded", logs: logs, sub: sub}, nil
}

// WatchCommitmentRecorded is a free log subscription operation binding the contract event 0xf76059fd91b15b2d465b41fe5d794955f8ac948e38e126713fbfb120585ff6bc.
//
// Solidity: event CommitmentRecorded(address indexed committer, bytes32 commitHash, uint256 timestamp)
func (_AgentCardRegistry *AgentCardRegistryFilterer) WatchCommitmentRecorded(opts *bind.WatchOpts, sink chan<- *AgentCardRegistryCommitmentRecorded, committer []common.Address) (event.Subscription, error) {

	var committerRule []interface{}
	for _, committerItem := range committer {
		committerRule = append(committerRule, committerItem)
	}

	logs, sub, err := _AgentCardRegistry.contract.WatchLogs(opts, "CommitmentRecorded", committerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AgentCardRegistryCommitmentRecorded)
				if err := _AgentCardRegistry.contract.UnpackLog(event, "CommitmentRecorded", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseCommitmentRecorded is a log parse operation binding the contract event 0xf76059fd91b15b2d465b41fe5d794955f8ac948e38e126713fbfb120585ff6bc.
//
// Solidity: event CommitmentRecorded(address indexed committer, bytes32 commitHash, uint256 timestamp)
func (_AgentCardRegistry *AgentCardRegistryFilterer) ParseCommitmentRecorded(log types.Log) (*AgentCardRegistryCommitmentRecorded, error) {
	event := new(AgentCardRegistryCommitmentRecorded)
	if err := _AgentCardRegistry.contract.UnpackLog(event, "CommitmentRecorded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AgentCardRegistryKMEKeyUpdatedIterator is returned from FilterKMEKeyUpdated and is used to iterate over the raw logs and unpacked data for KMEKeyUpdated events raised by the AgentCardRegistry contract.
type AgentCardRegistryKMEKeyUpdatedIterator struct {
	Event *AgentCardRegistryKMEKeyUpdated // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *AgentCardRegistryKMEKeyUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AgentCardRegistryKMEKeyUpdated)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(AgentCardRegistryKMEKeyUpdated)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *AgentCardRegistryKMEKeyUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AgentCardRegistryKMEKeyUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AgentCardRegistryKMEKeyUpdated represents a KMEKeyUpdated event raised by the AgentCardRegistry contract.
type AgentCardRegistryKMEKeyUpdated struct {
	AgentId   [32]byte
	KeyHash   [32]byte
	Timestamp *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterKMEKeyUpdated is a free log retrieval operation binding the contract event 0xe81db0f174ab8a1015816d79768cb3de679372bea7420f1c1d25df6562c612ef.
//
// Solidity: event KMEKeyUpdated(bytes32 indexed agentId, bytes32 indexed keyHash, uint256 timestamp)
func (_AgentCardRegistry *AgentCardRegistryFilterer) FilterKMEKeyUpdated(opts *bind.FilterOpts, agentId [][32]byte, keyHash [][32]byte) (*AgentCardRegistryKMEKeyUpdatedIterator, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}
	var keyHashRule []interface{}
	for _, keyHashItem := range keyHash {
		keyHashRule = append(keyHashRule, keyHashItem)
	}

	logs, sub, err := _AgentCardRegistry.contract.FilterLogs(opts, "KMEKeyUpdated", agentIdRule, keyHashRule)
	if err != nil {
		return nil, err
	}
	return &AgentCardRegistryKMEKeyUpdatedIterator{contract: _AgentCardRegistry.contract, event: "KMEKeyUpdated", logs: logs, sub: sub}, nil
}

// WatchKMEKeyUpdated is a free log subscription operation binding the contract event 0xe81db0f174ab8a1015816d79768cb3de679372bea7420f1c1d25df6562c612ef.
//
// Solidity: event KMEKeyUpdated(bytes32 indexed agentId, bytes32 indexed keyHash, uint256 timestamp)
func (_AgentCardRegistry *AgentCardRegistryFilterer) WatchKMEKeyUpdated(opts *bind.WatchOpts, sink chan<- *AgentCardRegistryKMEKeyUpdated, agentId [][32]byte, keyHash [][32]byte) (event.Subscription, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}
	var keyHashRule []interface{}
	for _, keyHashItem := range keyHash {
		keyHashRule = append(keyHashRule, keyHashItem)
	}

	logs, sub, err := _AgentCardRegistry.contract.WatchLogs(opts, "KMEKeyUpdated", agentIdRule, keyHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AgentCardRegistryKMEKeyUpdated)
				if err := _AgentCardRegistry.contract.UnpackLog(event, "KMEKeyUpdated", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseKMEKeyUpdated is a log parse operation binding the contract event 0xe81db0f174ab8a1015816d79768cb3de679372bea7420f1c1d25df6562c612ef.
//
// Solidity: event KMEKeyUpdated(bytes32 indexed agentId, bytes32 indexed keyHash, uint256 timestamp)
func (_AgentCardRegistry *AgentCardRegistryFilterer) ParseKMEKeyUpdated(log types.Log) (*AgentCardRegistryKMEKeyUpdated, error) {
	event := new(AgentCardRegistryKMEKeyUpdated)
	if err := _AgentCardRegistry.contract.UnpackLog(event, "KMEKeyUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AgentCardRegistryKeyAddedIterator is returned from FilterKeyAdded and is used to iterate over the raw logs and unpacked data for KeyAdded events raised by the AgentCardRegistry contract.
type AgentCardRegistryKeyAddedIterator struct {
	Event *AgentCardRegistryKeyAdded // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *AgentCardRegistryKeyAddedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AgentCardRegistryKeyAdded)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(AgentCardRegistryKeyAdded)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *AgentCardRegistryKeyAddedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AgentCardRegistryKeyAddedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AgentCardRegistryKeyAdded represents a KeyAdded event raised by the AgentCardRegistry contract.
type AgentCardRegistryKeyAdded struct {
	AgentId   [32]byte
	KeyHash   [32]byte
	KeyType   uint8
	Timestamp *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterKeyAdded is a free log retrieval operation binding the contract event 0x11f138c8931fc92ab4fbeb5dd32df17d56c9411a543739c3526ed0265d8fad13.
//
// Solidity: event KeyAdded(bytes32 indexed agentId, bytes32 indexed keyHash, uint8 keyType, uint256 timestamp)
func (_AgentCardRegistry *AgentCardRegistryFilterer) FilterKeyAdded(opts *bind.FilterOpts, agentId [][32]byte, keyHash [][32]byte) (*AgentCardRegistryKeyAddedIterator, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}
	var keyHashRule []interface{}
	for _, keyHashItem := range keyHash {
		keyHashRule = append(keyHashRule, keyHashItem)
	}

	logs, sub, err := _AgentCardRegistry.contract.FilterLogs(opts, "KeyAdded", agentIdRule, keyHashRule)
	if err != nil {
		return nil, err
	}
	return &AgentCardRegistryKeyAddedIterator{contract: _AgentCardRegistry.contract, event: "KeyAdded", logs: logs, sub: sub}, nil
}

// WatchKeyAdded is a free log subscription operation binding the contract event 0x11f138c8931fc92ab4fbeb5dd32df17d56c9411a543739c3526ed0265d8fad13.
//
// Solidity: event KeyAdded(bytes32 indexed agentId, bytes32 indexed keyHash, uint8 keyType, uint256 timestamp)
func (_AgentCardRegistry *AgentCardRegistryFilterer) WatchKeyAdded(opts *bind.WatchOpts, sink chan<- *AgentCardRegistryKeyAdded, agentId [][32]byte, keyHash [][32]byte) (event.Subscription, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}
	var keyHashRule []interface{}
	for _, keyHashItem := range keyHash {
		keyHashRule = append(keyHashRule, keyHashItem)
	}

	logs, sub, err := _AgentCardRegistry.contract.WatchLogs(opts, "KeyAdded", agentIdRule, keyHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AgentCardRegistryKeyAdded)
				if err := _AgentCardRegistry.contract.UnpackLog(event, "KeyAdded", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseKeyAdded is a log parse operation binding the contract event 0x11f138c8931fc92ab4fbeb5dd32df17d56c9411a543739c3526ed0265d8fad13.
//
// Solidity: event KeyAdded(bytes32 indexed agentId, bytes32 indexed keyHash, uint8 keyType, uint256 timestamp)
func (_AgentCardRegistry *AgentCardRegistryFilterer) ParseKeyAdded(log types.Log) (*AgentCardRegistryKeyAdded, error) {
	event := new(AgentCardRegistryKeyAdded)
	if err := _AgentCardRegistry.contract.UnpackLog(event, "KeyAdded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AgentCardRegistryKeyRevokedIterator is returned from FilterKeyRevoked and is used to iterate over the raw logs and unpacked data for KeyRevoked events raised by the AgentCardRegistry contract.
type AgentCardRegistryKeyRevokedIterator struct {
	Event *AgentCardRegistryKeyRevoked // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *AgentCardRegistryKeyRevokedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AgentCardRegistryKeyRevoked)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(AgentCardRegistryKeyRevoked)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *AgentCardRegistryKeyRevokedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AgentCardRegistryKeyRevokedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AgentCardRegistryKeyRevoked represents a KeyRevoked event raised by the AgentCardRegistry contract.
type AgentCardRegistryKeyRevoked struct {
	AgentId   [32]byte
	KeyHash   [32]byte
	Timestamp *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterKeyRevoked is a free log retrieval operation binding the contract event 0x209fb85e2522622566ffdf13e48258218f4c155aefc75703539e1a971380cd3f.
//
// Solidity: event KeyRevoked(bytes32 indexed agentId, bytes32 indexed keyHash, uint256 timestamp)
func (_AgentCardRegistry *AgentCardRegistryFilterer) FilterKeyRevoked(opts *bind.FilterOpts, agentId [][32]byte, keyHash [][32]byte) (*AgentCardRegistryKeyRevokedIterator, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}
	var keyHashRule []interface{}
	for _, keyHashItem := range keyHash {
		keyHashRule = append(keyHashRule, keyHashItem)
	}

	logs, sub, err := _AgentCardRegistry.contract.FilterLogs(opts, "KeyRevoked", agentIdRule, keyHashRule)
	if err != nil {
		return nil, err
	}
	return &AgentCardRegistryKeyRevokedIterator{contract: _AgentCardRegistry.contract, event: "KeyRevoked", logs: logs, sub: sub}, nil
}

// WatchKeyRevoked is a free log subscription operation binding the contract event 0x209fb85e2522622566ffdf13e48258218f4c155aefc75703539e1a971380cd3f.
//
// Solidity: event KeyRevoked(bytes32 indexed agentId, bytes32 indexed keyHash, uint256 timestamp)
func (_AgentCardRegistry *AgentCardRegistryFilterer) WatchKeyRevoked(opts *bind.WatchOpts, sink chan<- *AgentCardRegistryKeyRevoked, agentId [][32]byte, keyHash [][32]byte) (event.Subscription, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}
	var keyHashRule []interface{}
	for _, keyHashItem := range keyHash {
		keyHashRule = append(keyHashRule, keyHashItem)
	}

	logs, sub, err := _AgentCardRegistry.contract.WatchLogs(opts, "KeyRevoked", agentIdRule, keyHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AgentCardRegistryKeyRevoked)
				if err := _AgentCardRegistry.contract.UnpackLog(event, "KeyRevoked", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseKeyRevoked is a log parse operation binding the contract event 0x209fb85e2522622566ffdf13e48258218f4c155aefc75703539e1a971380cd3f.
//
// Solidity: event KeyRevoked(bytes32 indexed agentId, bytes32 indexed keyHash, uint256 timestamp)
func (_AgentCardRegistry *AgentCardRegistryFilterer) ParseKeyRevoked(log types.Log) (*AgentCardRegistryKeyRevoked, error) {
	event := new(AgentCardRegistryKeyRevoked)
	if err := _AgentCardRegistry.contract.UnpackLog(event, "KeyRevoked", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AgentCardRegistryOwnershipTransferStartedIterator is returned from FilterOwnershipTransferStarted and is used to iterate over the raw logs and unpacked data for OwnershipTransferStarted events raised by the AgentCardRegistry contract.
type AgentCardRegistryOwnershipTransferStartedIterator struct {
	Event *AgentCardRegistryOwnershipTransferStarted // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *AgentCardRegistryOwnershipTransferStartedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AgentCardRegistryOwnershipTransferStarted)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(AgentCardRegistryOwnershipTransferStarted)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *AgentCardRegistryOwnershipTransferStartedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AgentCardRegistryOwnershipTransferStartedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AgentCardRegistryOwnershipTransferStarted represents a OwnershipTransferStarted event raised by the AgentCardRegistry contract.
type AgentCardRegistryOwnershipTransferStarted struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferStarted is a free log retrieval operation binding the contract event 0x38d16b8cac22d99fc7c124b9cd0de2d3fa1faef420bfe791d8c362d765e22700.
//
// Solidity: event OwnershipTransferStarted(address indexed previousOwner, address indexed newOwner)
func (_AgentCardRegistry *AgentCardRegistryFilterer) FilterOwnershipTransferStarted(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*AgentCardRegistryOwnershipTransferStartedIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _AgentCardRegistry.contract.FilterLogs(opts, "OwnershipTransferStarted", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &AgentCardRegistryOwnershipTransferStartedIterator{contract: _AgentCardRegistry.contract, event: "OwnershipTransferStarted", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferStarted is a free log subscription operation binding the contract event 0x38d16b8cac22d99fc7c124b9cd0de2d3fa1faef420bfe791d8c362d765e22700.
//
// Solidity: event OwnershipTransferStarted(address indexed previousOwner, address indexed newOwner)
func (_AgentCardRegistry *AgentCardRegistryFilterer) WatchOwnershipTransferStarted(opts *bind.WatchOpts, sink chan<- *AgentCardRegistryOwnershipTransferStarted, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _AgentCardRegistry.contract.WatchLogs(opts, "OwnershipTransferStarted", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AgentCardRegistryOwnershipTransferStarted)
				if err := _AgentCardRegistry.contract.UnpackLog(event, "OwnershipTransferStarted", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseOwnershipTransferStarted is a log parse operation binding the contract event 0x38d16b8cac22d99fc7c124b9cd0de2d3fa1faef420bfe791d8c362d765e22700.
//
// Solidity: event OwnershipTransferStarted(address indexed previousOwner, address indexed newOwner)
func (_AgentCardRegistry *AgentCardRegistryFilterer) ParseOwnershipTransferStarted(log types.Log) (*AgentCardRegistryOwnershipTransferStarted, error) {
	event := new(AgentCardRegistryOwnershipTransferStarted)
	if err := _AgentCardRegistry.contract.UnpackLog(event, "OwnershipTransferStarted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AgentCardRegistryOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the AgentCardRegistry contract.
type AgentCardRegistryOwnershipTransferredIterator struct {
	Event *AgentCardRegistryOwnershipTransferred // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *AgentCardRegistryOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AgentCardRegistryOwnershipTransferred)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(AgentCardRegistryOwnershipTransferred)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *AgentCardRegistryOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AgentCardRegistryOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AgentCardRegistryOwnershipTransferred represents a OwnershipTransferred event raised by the AgentCardRegistry contract.
type AgentCardRegistryOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_AgentCardRegistry *AgentCardRegistryFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*AgentCardRegistryOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _AgentCardRegistry.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &AgentCardRegistryOwnershipTransferredIterator{contract: _AgentCardRegistry.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_AgentCardRegistry *AgentCardRegistryFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *AgentCardRegistryOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _AgentCardRegistry.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AgentCardRegistryOwnershipTransferred)
				if err := _AgentCardRegistry.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseOwnershipTransferred is a log parse operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_AgentCardRegistry *AgentCardRegistryFilterer) ParseOwnershipTransferred(log types.Log) (*AgentCardRegistryOwnershipTransferred, error) {
	event := new(AgentCardRegistryOwnershipTransferred)
	if err := _AgentCardRegistry.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AgentCardRegistryPausedIterator is returned from FilterPaused and is used to iterate over the raw logs and unpacked data for Paused events raised by the AgentCardRegistry contract.
type AgentCardRegistryPausedIterator struct {
	Event *AgentCardRegistryPaused // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *AgentCardRegistryPausedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AgentCardRegistryPaused)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(AgentCardRegistryPaused)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *AgentCardRegistryPausedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AgentCardRegistryPausedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AgentCardRegistryPaused represents a Paused event raised by the AgentCardRegistry contract.
type AgentCardRegistryPaused struct {
	Account common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterPaused is a free log retrieval operation binding the contract event 0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258.
//
// Solidity: event Paused(address account)
func (_AgentCardRegistry *AgentCardRegistryFilterer) FilterPaused(opts *bind.FilterOpts) (*AgentCardRegistryPausedIterator, error) {

	logs, sub, err := _AgentCardRegistry.contract.FilterLogs(opts, "Paused")
	if err != nil {
		return nil, err
	}
	return &AgentCardRegistryPausedIterator{contract: _AgentCardRegistry.contract, event: "Paused", logs: logs, sub: sub}, nil
}

// WatchPaused is a free log subscription operation binding the contract event 0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258.
//
// Solidity: event Paused(address account)
func (_AgentCardRegistry *AgentCardRegistryFilterer) WatchPaused(opts *bind.WatchOpts, sink chan<- *AgentCardRegistryPaused) (event.Subscription, error) {

	logs, sub, err := _AgentCardRegistry.contract.WatchLogs(opts, "Paused")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AgentCardRegistryPaused)
				if err := _AgentCardRegistry.contract.UnpackLog(event, "Paused", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParsePaused is a log parse operation binding the contract event 0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258.
//
// Solidity: event Paused(address account)
func (_AgentCardRegistry *AgentCardRegistryFilterer) ParsePaused(log types.Log) (*AgentCardRegistryPaused, error) {
	event := new(AgentCardRegistryPaused)
	if err := _AgentCardRegistry.contract.UnpackLog(event, "Paused", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AgentCardRegistryUnpausedIterator is returned from FilterUnpaused and is used to iterate over the raw logs and unpacked data for Unpaused events raised by the AgentCardRegistry contract.
type AgentCardRegistryUnpausedIterator struct {
	Event *AgentCardRegistryUnpaused // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *AgentCardRegistryUnpausedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AgentCardRegistryUnpaused)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(AgentCardRegistryUnpaused)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *AgentCardRegistryUnpausedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AgentCardRegistryUnpausedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AgentCardRegistryUnpaused represents a Unpaused event raised by the AgentCardRegistry contract.
type AgentCardRegistryUnpaused struct {
	Account common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterUnpaused is a free log retrieval operation binding the contract event 0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa.
//
// Solidity: event Unpaused(address account)
func (_AgentCardRegistry *AgentCardRegistryFilterer) FilterUnpaused(opts *bind.FilterOpts) (*AgentCardRegistryUnpausedIterator, error) {

	logs, sub, err := _AgentCardRegistry.contract.FilterLogs(opts, "Unpaused")
	if err != nil {
		return nil, err
	}
	return &AgentCardRegistryUnpausedIterator{contract: _AgentCardRegistry.contract, event: "Unpaused", logs: logs, sub: sub}, nil
}

// WatchUnpaused is a free log subscription operation binding the contract event 0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa.
//
// Solidity: event Unpaused(address account)
func (_AgentCardRegistry *AgentCardRegistryFilterer) WatchUnpaused(opts *bind.WatchOpts, sink chan<- *AgentCardRegistryUnpaused) (event.Subscription, error) {

	logs, sub, err := _AgentCardRegistry.contract.WatchLogs(opts, "Unpaused")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AgentCardRegistryUnpaused)
				if err := _AgentCardRegistry.contract.UnpackLog(event, "Unpaused", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseUnpaused is a log parse operation binding the contract event 0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa.
//
// Solidity: event Unpaused(address account)
func (_AgentCardRegistry *AgentCardRegistryFilterer) ParseUnpaused(log types.Log) (*AgentCardRegistryUnpaused, error) {
	event := new(AgentCardRegistryUnpaused)
	if err := _AgentCardRegistry.contract.UnpackLog(event, "Unpaused", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

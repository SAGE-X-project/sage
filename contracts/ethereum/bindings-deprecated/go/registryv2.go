// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package registryv2

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

// ISageRegistryAgentMetadata is an auto generated low-level Go binding around an user-defined struct.
type ISageRegistryAgentMetadata struct {
	Did          string
	Name         string
	Description  string
	Endpoint     string
	PublicKey    []byte
	Capabilities string
	Owner        common.Address
	RegisteredAt *big.Int
	UpdatedAt    *big.Int
	Active       bool
}

// SageRegistryV2MetaData contains all meta data concerning the SageRegistryV2 contract.
var SageRegistryV2MetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"AfterRegisterHook\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"oldHook\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newHook\",\"type\":\"address\"}],\"name\":\"AfterRegisterHookUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"}],\"name\":\"AgentDeactivated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"did\",\"type\":\"string\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"}],\"name\":\"AgentRegistered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"}],\"name\":\"AgentUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"BeforeRegisterHook\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"oldHook\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newHook\",\"type\":\"address\"}],\"name\":\"BeforeRegisterHookUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"hook\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"reason\",\"type\":\"string\"}],\"name\":\"HookFailed\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"keyHash\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"KeyRevoked\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"keyHash\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"KeyValidated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferStarted\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"Paused\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"Unpaused\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"acceptOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"afterRegisterHook\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"beforeRegisterHook\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"}],\"name\":\"deactivateAgent\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"did\",\"type\":\"string\"}],\"name\":\"deactivateAgentByDID\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"}],\"name\":\"getAgent\",\"outputs\":[{\"components\":[{\"internalType\":\"string\",\"name\":\"did\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"description\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"endpoint\",\"type\":\"string\"},{\"internalType\":\"bytes\",\"name\":\"publicKey\",\"type\":\"bytes\"},{\"internalType\":\"string\",\"name\":\"capabilities\",\"type\":\"string\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"registeredAt\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"updatedAt\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"active\",\"type\":\"bool\"}],\"internalType\":\"structISageRegistry.AgentMetadata\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"did\",\"type\":\"string\"}],\"name\":\"getAgentByDID\",\"outputs\":[{\"components\":[{\"internalType\":\"string\",\"name\":\"did\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"description\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"endpoint\",\"type\":\"string\"},{\"internalType\":\"bytes\",\"name\":\"publicKey\",\"type\":\"bytes\"},{\"internalType\":\"string\",\"name\":\"capabilities\",\"type\":\"string\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"registeredAt\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"updatedAt\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"active\",\"type\":\"bool\"}],\"internalType\":\"structISageRegistry.AgentMetadata\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"ownerAddress\",\"type\":\"address\"}],\"name\":\"getAgentsByOwner\",\"outputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"\",\"type\":\"bytes32[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"}],\"name\":\"isAgentActive\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"publicKey\",\"type\":\"bytes\"}],\"name\":\"isKeyValid\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"pause\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"paused\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"pendingOwner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"did\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"description\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"endpoint\",\"type\":\"string\"},{\"internalType\":\"bytes\",\"name\":\"publicKey\",\"type\":\"bytes\"},{\"internalType\":\"string\",\"name\":\"capabilities\",\"type\":\"string\"},{\"internalType\":\"bytes\",\"name\":\"signature\",\"type\":\"bytes\"}],\"name\":\"registerAgent\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"publicKey\",\"type\":\"bytes\"}],\"name\":\"revokeKey\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"hook\",\"type\":\"address\"}],\"name\":\"setAfterRegisterHook\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"hook\",\"type\":\"address\"}],\"name\":\"setBeforeRegisterHook\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"unpause\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"description\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"endpoint\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"capabilities\",\"type\":\"string\"},{\"internalType\":\"bytes\",\"name\":\"signature\",\"type\":\"bytes\"}],\"name\":\"updateAgent\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"claimedOwner\",\"type\":\"address\"}],\"name\":\"verifyAgentOwnership\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// SageRegistryV2ABI is the input ABI used to generate the binding from.
// Deprecated: Use SageRegistryV2MetaData.ABI instead.
var SageRegistryV2ABI = SageRegistryV2MetaData.ABI

// SageRegistryV2 is an auto generated Go binding around an Ethereum contract.
type SageRegistryV2 struct {
	SageRegistryV2Caller     // Read-only binding to the contract
	SageRegistryV2Transactor // Write-only binding to the contract
	SageRegistryV2Filterer   // Log filterer for contract events
}

// SageRegistryV2Caller is an auto generated read-only Go binding around an Ethereum contract.
type SageRegistryV2Caller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SageRegistryV2Transactor is an auto generated write-only Go binding around an Ethereum contract.
type SageRegistryV2Transactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SageRegistryV2Filterer is an auto generated log filtering Go binding around an Ethereum contract events.
type SageRegistryV2Filterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SageRegistryV2Session is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type SageRegistryV2Session struct {
	Contract     *SageRegistryV2   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// SageRegistryV2CallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type SageRegistryV2CallerSession struct {
	Contract *SageRegistryV2Caller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// SageRegistryV2TransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type SageRegistryV2TransactorSession struct {
	Contract     *SageRegistryV2Transactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// SageRegistryV2Raw is an auto generated low-level Go binding around an Ethereum contract.
type SageRegistryV2Raw struct {
	Contract *SageRegistryV2 // Generic contract binding to access the raw methods on
}

// SageRegistryV2CallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type SageRegistryV2CallerRaw struct {
	Contract *SageRegistryV2Caller // Generic read-only contract binding to access the raw methods on
}

// SageRegistryV2TransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type SageRegistryV2TransactorRaw struct {
	Contract *SageRegistryV2Transactor // Generic write-only contract binding to access the raw methods on
}

// NewSageRegistryV2 creates a new instance of SageRegistryV2, bound to a specific deployed contract.
func NewSageRegistryV2(address common.Address, backend bind.ContractBackend) (*SageRegistryV2, error) {
	contract, err := bindSageRegistryV2(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &SageRegistryV2{SageRegistryV2Caller: SageRegistryV2Caller{contract: contract}, SageRegistryV2Transactor: SageRegistryV2Transactor{contract: contract}, SageRegistryV2Filterer: SageRegistryV2Filterer{contract: contract}}, nil
}

// NewSageRegistryV2Caller creates a new read-only instance of SageRegistryV2, bound to a specific deployed contract.
func NewSageRegistryV2Caller(address common.Address, caller bind.ContractCaller) (*SageRegistryV2Caller, error) {
	contract, err := bindSageRegistryV2(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &SageRegistryV2Caller{contract: contract}, nil
}

// NewSageRegistryV2Transactor creates a new write-only instance of SageRegistryV2, bound to a specific deployed contract.
func NewSageRegistryV2Transactor(address common.Address, transactor bind.ContractTransactor) (*SageRegistryV2Transactor, error) {
	contract, err := bindSageRegistryV2(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &SageRegistryV2Transactor{contract: contract}, nil
}

// NewSageRegistryV2Filterer creates a new log filterer instance of SageRegistryV2, bound to a specific deployed contract.
func NewSageRegistryV2Filterer(address common.Address, filterer bind.ContractFilterer) (*SageRegistryV2Filterer, error) {
	contract, err := bindSageRegistryV2(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &SageRegistryV2Filterer{contract: contract}, nil
}

// bindSageRegistryV2 binds a generic wrapper to an already deployed contract.
func bindSageRegistryV2(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := SageRegistryV2MetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SageRegistryV2 *SageRegistryV2Raw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SageRegistryV2.Contract.SageRegistryV2Caller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SageRegistryV2 *SageRegistryV2Raw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SageRegistryV2.Contract.SageRegistryV2Transactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SageRegistryV2 *SageRegistryV2Raw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SageRegistryV2.Contract.SageRegistryV2Transactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SageRegistryV2 *SageRegistryV2CallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SageRegistryV2.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SageRegistryV2 *SageRegistryV2TransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SageRegistryV2.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SageRegistryV2 *SageRegistryV2TransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SageRegistryV2.Contract.contract.Transact(opts, method, params...)
}

// AfterRegisterHook is a free data retrieval call binding the contract method 0x24e6c522.
//
// Solidity: function afterRegisterHook() view returns(address)
func (_SageRegistryV2 *SageRegistryV2Caller) AfterRegisterHook(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SageRegistryV2.contract.Call(opts, &out, "afterRegisterHook")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// AfterRegisterHook is a free data retrieval call binding the contract method 0x24e6c522.
//
// Solidity: function afterRegisterHook() view returns(address)
func (_SageRegistryV2 *SageRegistryV2Session) AfterRegisterHook() (common.Address, error) {
	return _SageRegistryV2.Contract.AfterRegisterHook(&_SageRegistryV2.CallOpts)
}

// AfterRegisterHook is a free data retrieval call binding the contract method 0x24e6c522.
//
// Solidity: function afterRegisterHook() view returns(address)
func (_SageRegistryV2 *SageRegistryV2CallerSession) AfterRegisterHook() (common.Address, error) {
	return _SageRegistryV2.Contract.AfterRegisterHook(&_SageRegistryV2.CallOpts)
}

// BeforeRegisterHook is a free data retrieval call binding the contract method 0x8051d5ea.
//
// Solidity: function beforeRegisterHook() view returns(address)
func (_SageRegistryV2 *SageRegistryV2Caller) BeforeRegisterHook(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SageRegistryV2.contract.Call(opts, &out, "beforeRegisterHook")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// BeforeRegisterHook is a free data retrieval call binding the contract method 0x8051d5ea.
//
// Solidity: function beforeRegisterHook() view returns(address)
func (_SageRegistryV2 *SageRegistryV2Session) BeforeRegisterHook() (common.Address, error) {
	return _SageRegistryV2.Contract.BeforeRegisterHook(&_SageRegistryV2.CallOpts)
}

// BeforeRegisterHook is a free data retrieval call binding the contract method 0x8051d5ea.
//
// Solidity: function beforeRegisterHook() view returns(address)
func (_SageRegistryV2 *SageRegistryV2CallerSession) BeforeRegisterHook() (common.Address, error) {
	return _SageRegistryV2.Contract.BeforeRegisterHook(&_SageRegistryV2.CallOpts)
}

// GetAgent is a free data retrieval call binding the contract method 0xa6c2af01.
//
// Solidity: function getAgent(bytes32 agentId) view returns((string,string,string,string,bytes,string,address,uint256,uint256,bool))
func (_SageRegistryV2 *SageRegistryV2Caller) GetAgent(opts *bind.CallOpts, agentId [32]byte) (ISageRegistryAgentMetadata, error) {
	var out []interface{}
	err := _SageRegistryV2.contract.Call(opts, &out, "getAgent", agentId)

	if err != nil {
		return *new(ISageRegistryAgentMetadata), err
	}

	out0 := *abi.ConvertType(out[0], new(ISageRegistryAgentMetadata)).(*ISageRegistryAgentMetadata)

	return out0, err

}

// GetAgent is a free data retrieval call binding the contract method 0xa6c2af01.
//
// Solidity: function getAgent(bytes32 agentId) view returns((string,string,string,string,bytes,string,address,uint256,uint256,bool))
func (_SageRegistryV2 *SageRegistryV2Session) GetAgent(agentId [32]byte) (ISageRegistryAgentMetadata, error) {
	return _SageRegistryV2.Contract.GetAgent(&_SageRegistryV2.CallOpts, agentId)
}

// GetAgent is a free data retrieval call binding the contract method 0xa6c2af01.
//
// Solidity: function getAgent(bytes32 agentId) view returns((string,string,string,string,bytes,string,address,uint256,uint256,bool))
func (_SageRegistryV2 *SageRegistryV2CallerSession) GetAgent(agentId [32]byte) (ISageRegistryAgentMetadata, error) {
	return _SageRegistryV2.Contract.GetAgent(&_SageRegistryV2.CallOpts, agentId)
}

// GetAgentByDID is a free data retrieval call binding the contract method 0xe45d486d.
//
// Solidity: function getAgentByDID(string did) view returns((string,string,string,string,bytes,string,address,uint256,uint256,bool))
func (_SageRegistryV2 *SageRegistryV2Caller) GetAgentByDID(opts *bind.CallOpts, did string) (ISageRegistryAgentMetadata, error) {
	var out []interface{}
	err := _SageRegistryV2.contract.Call(opts, &out, "getAgentByDID", did)

	if err != nil {
		return *new(ISageRegistryAgentMetadata), err
	}

	out0 := *abi.ConvertType(out[0], new(ISageRegistryAgentMetadata)).(*ISageRegistryAgentMetadata)

	return out0, err

}

// GetAgentByDID is a free data retrieval call binding the contract method 0xe45d486d.
//
// Solidity: function getAgentByDID(string did) view returns((string,string,string,string,bytes,string,address,uint256,uint256,bool))
func (_SageRegistryV2 *SageRegistryV2Session) GetAgentByDID(did string) (ISageRegistryAgentMetadata, error) {
	return _SageRegistryV2.Contract.GetAgentByDID(&_SageRegistryV2.CallOpts, did)
}

// GetAgentByDID is a free data retrieval call binding the contract method 0xe45d486d.
//
// Solidity: function getAgentByDID(string did) view returns((string,string,string,string,bytes,string,address,uint256,uint256,bool))
func (_SageRegistryV2 *SageRegistryV2CallerSession) GetAgentByDID(did string) (ISageRegistryAgentMetadata, error) {
	return _SageRegistryV2.Contract.GetAgentByDID(&_SageRegistryV2.CallOpts, did)
}

// GetAgentsByOwner is a free data retrieval call binding the contract method 0x1ab6f888.
//
// Solidity: function getAgentsByOwner(address ownerAddress) view returns(bytes32[])
func (_SageRegistryV2 *SageRegistryV2Caller) GetAgentsByOwner(opts *bind.CallOpts, ownerAddress common.Address) ([][32]byte, error) {
	var out []interface{}
	err := _SageRegistryV2.contract.Call(opts, &out, "getAgentsByOwner", ownerAddress)

	if err != nil {
		return *new([][32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([][32]byte)).(*[][32]byte)

	return out0, err

}

// GetAgentsByOwner is a free data retrieval call binding the contract method 0x1ab6f888.
//
// Solidity: function getAgentsByOwner(address ownerAddress) view returns(bytes32[])
func (_SageRegistryV2 *SageRegistryV2Session) GetAgentsByOwner(ownerAddress common.Address) ([][32]byte, error) {
	return _SageRegistryV2.Contract.GetAgentsByOwner(&_SageRegistryV2.CallOpts, ownerAddress)
}

// GetAgentsByOwner is a free data retrieval call binding the contract method 0x1ab6f888.
//
// Solidity: function getAgentsByOwner(address ownerAddress) view returns(bytes32[])
func (_SageRegistryV2 *SageRegistryV2CallerSession) GetAgentsByOwner(ownerAddress common.Address) ([][32]byte, error) {
	return _SageRegistryV2.Contract.GetAgentsByOwner(&_SageRegistryV2.CallOpts, ownerAddress)
}

// IsAgentActive is a free data retrieval call binding the contract method 0x8a92792b.
//
// Solidity: function isAgentActive(bytes32 agentId) view returns(bool)
func (_SageRegistryV2 *SageRegistryV2Caller) IsAgentActive(opts *bind.CallOpts, agentId [32]byte) (bool, error) {
	var out []interface{}
	err := _SageRegistryV2.contract.Call(opts, &out, "isAgentActive", agentId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsAgentActive is a free data retrieval call binding the contract method 0x8a92792b.
//
// Solidity: function isAgentActive(bytes32 agentId) view returns(bool)
func (_SageRegistryV2 *SageRegistryV2Session) IsAgentActive(agentId [32]byte) (bool, error) {
	return _SageRegistryV2.Contract.IsAgentActive(&_SageRegistryV2.CallOpts, agentId)
}

// IsAgentActive is a free data retrieval call binding the contract method 0x8a92792b.
//
// Solidity: function isAgentActive(bytes32 agentId) view returns(bool)
func (_SageRegistryV2 *SageRegistryV2CallerSession) IsAgentActive(agentId [32]byte) (bool, error) {
	return _SageRegistryV2.Contract.IsAgentActive(&_SageRegistryV2.CallOpts, agentId)
}

// IsKeyValid is a free data retrieval call binding the contract method 0xb47a9025.
//
// Solidity: function isKeyValid(bytes publicKey) view returns(bool)
func (_SageRegistryV2 *SageRegistryV2Caller) IsKeyValid(opts *bind.CallOpts, publicKey []byte) (bool, error) {
	var out []interface{}
	err := _SageRegistryV2.contract.Call(opts, &out, "isKeyValid", publicKey)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsKeyValid is a free data retrieval call binding the contract method 0xb47a9025.
//
// Solidity: function isKeyValid(bytes publicKey) view returns(bool)
func (_SageRegistryV2 *SageRegistryV2Session) IsKeyValid(publicKey []byte) (bool, error) {
	return _SageRegistryV2.Contract.IsKeyValid(&_SageRegistryV2.CallOpts, publicKey)
}

// IsKeyValid is a free data retrieval call binding the contract method 0xb47a9025.
//
// Solidity: function isKeyValid(bytes publicKey) view returns(bool)
func (_SageRegistryV2 *SageRegistryV2CallerSession) IsKeyValid(publicKey []byte) (bool, error) {
	return _SageRegistryV2.Contract.IsKeyValid(&_SageRegistryV2.CallOpts, publicKey)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_SageRegistryV2 *SageRegistryV2Caller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SageRegistryV2.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_SageRegistryV2 *SageRegistryV2Session) Owner() (common.Address, error) {
	return _SageRegistryV2.Contract.Owner(&_SageRegistryV2.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_SageRegistryV2 *SageRegistryV2CallerSession) Owner() (common.Address, error) {
	return _SageRegistryV2.Contract.Owner(&_SageRegistryV2.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_SageRegistryV2 *SageRegistryV2Caller) Paused(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _SageRegistryV2.contract.Call(opts, &out, "paused")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_SageRegistryV2 *SageRegistryV2Session) Paused() (bool, error) {
	return _SageRegistryV2.Contract.Paused(&_SageRegistryV2.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_SageRegistryV2 *SageRegistryV2CallerSession) Paused() (bool, error) {
	return _SageRegistryV2.Contract.Paused(&_SageRegistryV2.CallOpts)
}

// PendingOwner is a free data retrieval call binding the contract method 0xe30c3978.
//
// Solidity: function pendingOwner() view returns(address)
func (_SageRegistryV2 *SageRegistryV2Caller) PendingOwner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SageRegistryV2.contract.Call(opts, &out, "pendingOwner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// PendingOwner is a free data retrieval call binding the contract method 0xe30c3978.
//
// Solidity: function pendingOwner() view returns(address)
func (_SageRegistryV2 *SageRegistryV2Session) PendingOwner() (common.Address, error) {
	return _SageRegistryV2.Contract.PendingOwner(&_SageRegistryV2.CallOpts)
}

// PendingOwner is a free data retrieval call binding the contract method 0xe30c3978.
//
// Solidity: function pendingOwner() view returns(address)
func (_SageRegistryV2 *SageRegistryV2CallerSession) PendingOwner() (common.Address, error) {
	return _SageRegistryV2.Contract.PendingOwner(&_SageRegistryV2.CallOpts)
}

// VerifyAgentOwnership is a free data retrieval call binding the contract method 0x745e8f81.
//
// Solidity: function verifyAgentOwnership(bytes32 agentId, address claimedOwner) view returns(bool)
func (_SageRegistryV2 *SageRegistryV2Caller) VerifyAgentOwnership(opts *bind.CallOpts, agentId [32]byte, claimedOwner common.Address) (bool, error) {
	var out []interface{}
	err := _SageRegistryV2.contract.Call(opts, &out, "verifyAgentOwnership", agentId, claimedOwner)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// VerifyAgentOwnership is a free data retrieval call binding the contract method 0x745e8f81.
//
// Solidity: function verifyAgentOwnership(bytes32 agentId, address claimedOwner) view returns(bool)
func (_SageRegistryV2 *SageRegistryV2Session) VerifyAgentOwnership(agentId [32]byte, claimedOwner common.Address) (bool, error) {
	return _SageRegistryV2.Contract.VerifyAgentOwnership(&_SageRegistryV2.CallOpts, agentId, claimedOwner)
}

// VerifyAgentOwnership is a free data retrieval call binding the contract method 0x745e8f81.
//
// Solidity: function verifyAgentOwnership(bytes32 agentId, address claimedOwner) view returns(bool)
func (_SageRegistryV2 *SageRegistryV2CallerSession) VerifyAgentOwnership(agentId [32]byte, claimedOwner common.Address) (bool, error) {
	return _SageRegistryV2.Contract.VerifyAgentOwnership(&_SageRegistryV2.CallOpts, agentId, claimedOwner)
}

// AcceptOwnership is a paid mutator transaction binding the contract method 0x79ba5097.
//
// Solidity: function acceptOwnership() returns()
func (_SageRegistryV2 *SageRegistryV2Transactor) AcceptOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SageRegistryV2.contract.Transact(opts, "acceptOwnership")
}

// AcceptOwnership is a paid mutator transaction binding the contract method 0x79ba5097.
//
// Solidity: function acceptOwnership() returns()
func (_SageRegistryV2 *SageRegistryV2Session) AcceptOwnership() (*types.Transaction, error) {
	return _SageRegistryV2.Contract.AcceptOwnership(&_SageRegistryV2.TransactOpts)
}

// AcceptOwnership is a paid mutator transaction binding the contract method 0x79ba5097.
//
// Solidity: function acceptOwnership() returns()
func (_SageRegistryV2 *SageRegistryV2TransactorSession) AcceptOwnership() (*types.Transaction, error) {
	return _SageRegistryV2.Contract.AcceptOwnership(&_SageRegistryV2.TransactOpts)
}

// DeactivateAgent is a paid mutator transaction binding the contract method 0x59b5acf3.
//
// Solidity: function deactivateAgent(bytes32 agentId) returns()
func (_SageRegistryV2 *SageRegistryV2Transactor) DeactivateAgent(opts *bind.TransactOpts, agentId [32]byte) (*types.Transaction, error) {
	return _SageRegistryV2.contract.Transact(opts, "deactivateAgent", agentId)
}

// DeactivateAgent is a paid mutator transaction binding the contract method 0x59b5acf3.
//
// Solidity: function deactivateAgent(bytes32 agentId) returns()
func (_SageRegistryV2 *SageRegistryV2Session) DeactivateAgent(agentId [32]byte) (*types.Transaction, error) {
	return _SageRegistryV2.Contract.DeactivateAgent(&_SageRegistryV2.TransactOpts, agentId)
}

// DeactivateAgent is a paid mutator transaction binding the contract method 0x59b5acf3.
//
// Solidity: function deactivateAgent(bytes32 agentId) returns()
func (_SageRegistryV2 *SageRegistryV2TransactorSession) DeactivateAgent(agentId [32]byte) (*types.Transaction, error) {
	return _SageRegistryV2.Contract.DeactivateAgent(&_SageRegistryV2.TransactOpts, agentId)
}

// DeactivateAgentByDID is a paid mutator transaction binding the contract method 0x93f0fa17.
//
// Solidity: function deactivateAgentByDID(string did) returns()
func (_SageRegistryV2 *SageRegistryV2Transactor) DeactivateAgentByDID(opts *bind.TransactOpts, did string) (*types.Transaction, error) {
	return _SageRegistryV2.contract.Transact(opts, "deactivateAgentByDID", did)
}

// DeactivateAgentByDID is a paid mutator transaction binding the contract method 0x93f0fa17.
//
// Solidity: function deactivateAgentByDID(string did) returns()
func (_SageRegistryV2 *SageRegistryV2Session) DeactivateAgentByDID(did string) (*types.Transaction, error) {
	return _SageRegistryV2.Contract.DeactivateAgentByDID(&_SageRegistryV2.TransactOpts, did)
}

// DeactivateAgentByDID is a paid mutator transaction binding the contract method 0x93f0fa17.
//
// Solidity: function deactivateAgentByDID(string did) returns()
func (_SageRegistryV2 *SageRegistryV2TransactorSession) DeactivateAgentByDID(did string) (*types.Transaction, error) {
	return _SageRegistryV2.Contract.DeactivateAgentByDID(&_SageRegistryV2.TransactOpts, did)
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_SageRegistryV2 *SageRegistryV2Transactor) Pause(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SageRegistryV2.contract.Transact(opts, "pause")
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_SageRegistryV2 *SageRegistryV2Session) Pause() (*types.Transaction, error) {
	return _SageRegistryV2.Contract.Pause(&_SageRegistryV2.TransactOpts)
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_SageRegistryV2 *SageRegistryV2TransactorSession) Pause() (*types.Transaction, error) {
	return _SageRegistryV2.Contract.Pause(&_SageRegistryV2.TransactOpts)
}

// RegisterAgent is a paid mutator transaction binding the contract method 0x22b7a307.
//
// Solidity: function registerAgent(string did, string name, string description, string endpoint, bytes publicKey, string capabilities, bytes signature) returns(bytes32)
func (_SageRegistryV2 *SageRegistryV2Transactor) RegisterAgent(opts *bind.TransactOpts, did string, name string, description string, endpoint string, publicKey []byte, capabilities string, signature []byte) (*types.Transaction, error) {
	return _SageRegistryV2.contract.Transact(opts, "registerAgent", did, name, description, endpoint, publicKey, capabilities, signature)
}

// RegisterAgent is a paid mutator transaction binding the contract method 0x22b7a307.
//
// Solidity: function registerAgent(string did, string name, string description, string endpoint, bytes publicKey, string capabilities, bytes signature) returns(bytes32)
func (_SageRegistryV2 *SageRegistryV2Session) RegisterAgent(did string, name string, description string, endpoint string, publicKey []byte, capabilities string, signature []byte) (*types.Transaction, error) {
	return _SageRegistryV2.Contract.RegisterAgent(&_SageRegistryV2.TransactOpts, did, name, description, endpoint, publicKey, capabilities, signature)
}

// RegisterAgent is a paid mutator transaction binding the contract method 0x22b7a307.
//
// Solidity: function registerAgent(string did, string name, string description, string endpoint, bytes publicKey, string capabilities, bytes signature) returns(bytes32)
func (_SageRegistryV2 *SageRegistryV2TransactorSession) RegisterAgent(did string, name string, description string, endpoint string, publicKey []byte, capabilities string, signature []byte) (*types.Transaction, error) {
	return _SageRegistryV2.Contract.RegisterAgent(&_SageRegistryV2.TransactOpts, did, name, description, endpoint, publicKey, capabilities, signature)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_SageRegistryV2 *SageRegistryV2Transactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SageRegistryV2.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_SageRegistryV2 *SageRegistryV2Session) RenounceOwnership() (*types.Transaction, error) {
	return _SageRegistryV2.Contract.RenounceOwnership(&_SageRegistryV2.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_SageRegistryV2 *SageRegistryV2TransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _SageRegistryV2.Contract.RenounceOwnership(&_SageRegistryV2.TransactOpts)
}

// RevokeKey is a paid mutator transaction binding the contract method 0x953909f8.
//
// Solidity: function revokeKey(bytes publicKey) returns()
func (_SageRegistryV2 *SageRegistryV2Transactor) RevokeKey(opts *bind.TransactOpts, publicKey []byte) (*types.Transaction, error) {
	return _SageRegistryV2.contract.Transact(opts, "revokeKey", publicKey)
}

// RevokeKey is a paid mutator transaction binding the contract method 0x953909f8.
//
// Solidity: function revokeKey(bytes publicKey) returns()
func (_SageRegistryV2 *SageRegistryV2Session) RevokeKey(publicKey []byte) (*types.Transaction, error) {
	return _SageRegistryV2.Contract.RevokeKey(&_SageRegistryV2.TransactOpts, publicKey)
}

// RevokeKey is a paid mutator transaction binding the contract method 0x953909f8.
//
// Solidity: function revokeKey(bytes publicKey) returns()
func (_SageRegistryV2 *SageRegistryV2TransactorSession) RevokeKey(publicKey []byte) (*types.Transaction, error) {
	return _SageRegistryV2.Contract.RevokeKey(&_SageRegistryV2.TransactOpts, publicKey)
}

// SetAfterRegisterHook is a paid mutator transaction binding the contract method 0xda7d9d8f.
//
// Solidity: function setAfterRegisterHook(address hook) returns()
func (_SageRegistryV2 *SageRegistryV2Transactor) SetAfterRegisterHook(opts *bind.TransactOpts, hook common.Address) (*types.Transaction, error) {
	return _SageRegistryV2.contract.Transact(opts, "setAfterRegisterHook", hook)
}

// SetAfterRegisterHook is a paid mutator transaction binding the contract method 0xda7d9d8f.
//
// Solidity: function setAfterRegisterHook(address hook) returns()
func (_SageRegistryV2 *SageRegistryV2Session) SetAfterRegisterHook(hook common.Address) (*types.Transaction, error) {
	return _SageRegistryV2.Contract.SetAfterRegisterHook(&_SageRegistryV2.TransactOpts, hook)
}

// SetAfterRegisterHook is a paid mutator transaction binding the contract method 0xda7d9d8f.
//
// Solidity: function setAfterRegisterHook(address hook) returns()
func (_SageRegistryV2 *SageRegistryV2TransactorSession) SetAfterRegisterHook(hook common.Address) (*types.Transaction, error) {
	return _SageRegistryV2.Contract.SetAfterRegisterHook(&_SageRegistryV2.TransactOpts, hook)
}

// SetBeforeRegisterHook is a paid mutator transaction binding the contract method 0x783f054c.
//
// Solidity: function setBeforeRegisterHook(address hook) returns()
func (_SageRegistryV2 *SageRegistryV2Transactor) SetBeforeRegisterHook(opts *bind.TransactOpts, hook common.Address) (*types.Transaction, error) {
	return _SageRegistryV2.contract.Transact(opts, "setBeforeRegisterHook", hook)
}

// SetBeforeRegisterHook is a paid mutator transaction binding the contract method 0x783f054c.
//
// Solidity: function setBeforeRegisterHook(address hook) returns()
func (_SageRegistryV2 *SageRegistryV2Session) SetBeforeRegisterHook(hook common.Address) (*types.Transaction, error) {
	return _SageRegistryV2.Contract.SetBeforeRegisterHook(&_SageRegistryV2.TransactOpts, hook)
}

// SetBeforeRegisterHook is a paid mutator transaction binding the contract method 0x783f054c.
//
// Solidity: function setBeforeRegisterHook(address hook) returns()
func (_SageRegistryV2 *SageRegistryV2TransactorSession) SetBeforeRegisterHook(hook common.Address) (*types.Transaction, error) {
	return _SageRegistryV2.Contract.SetBeforeRegisterHook(&_SageRegistryV2.TransactOpts, hook)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_SageRegistryV2 *SageRegistryV2Transactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _SageRegistryV2.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_SageRegistryV2 *SageRegistryV2Session) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _SageRegistryV2.Contract.TransferOwnership(&_SageRegistryV2.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_SageRegistryV2 *SageRegistryV2TransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _SageRegistryV2.Contract.TransferOwnership(&_SageRegistryV2.TransactOpts, newOwner)
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_SageRegistryV2 *SageRegistryV2Transactor) Unpause(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SageRegistryV2.contract.Transact(opts, "unpause")
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_SageRegistryV2 *SageRegistryV2Session) Unpause() (*types.Transaction, error) {
	return _SageRegistryV2.Contract.Unpause(&_SageRegistryV2.TransactOpts)
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_SageRegistryV2 *SageRegistryV2TransactorSession) Unpause() (*types.Transaction, error) {
	return _SageRegistryV2.Contract.Unpause(&_SageRegistryV2.TransactOpts)
}

// UpdateAgent is a paid mutator transaction binding the contract method 0x3ae37799.
//
// Solidity: function updateAgent(bytes32 agentId, string name, string description, string endpoint, string capabilities, bytes signature) returns()
func (_SageRegistryV2 *SageRegistryV2Transactor) UpdateAgent(opts *bind.TransactOpts, agentId [32]byte, name string, description string, endpoint string, capabilities string, signature []byte) (*types.Transaction, error) {
	return _SageRegistryV2.contract.Transact(opts, "updateAgent", agentId, name, description, endpoint, capabilities, signature)
}

// UpdateAgent is a paid mutator transaction binding the contract method 0x3ae37799.
//
// Solidity: function updateAgent(bytes32 agentId, string name, string description, string endpoint, string capabilities, bytes signature) returns()
func (_SageRegistryV2 *SageRegistryV2Session) UpdateAgent(agentId [32]byte, name string, description string, endpoint string, capabilities string, signature []byte) (*types.Transaction, error) {
	return _SageRegistryV2.Contract.UpdateAgent(&_SageRegistryV2.TransactOpts, agentId, name, description, endpoint, capabilities, signature)
}

// UpdateAgent is a paid mutator transaction binding the contract method 0x3ae37799.
//
// Solidity: function updateAgent(bytes32 agentId, string name, string description, string endpoint, string capabilities, bytes signature) returns()
func (_SageRegistryV2 *SageRegistryV2TransactorSession) UpdateAgent(agentId [32]byte, name string, description string, endpoint string, capabilities string, signature []byte) (*types.Transaction, error) {
	return _SageRegistryV2.Contract.UpdateAgent(&_SageRegistryV2.TransactOpts, agentId, name, description, endpoint, capabilities, signature)
}

// SageRegistryV2AfterRegisterHookIterator is returned from FilterAfterRegisterHook and is used to iterate over the raw logs and unpacked data for AfterRegisterHook events raised by the SageRegistryV2 contract.
type SageRegistryV2AfterRegisterHookIterator struct {
	Event *SageRegistryV2AfterRegisterHook // Event containing the contract specifics and raw log

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
func (it *SageRegistryV2AfterRegisterHookIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SageRegistryV2AfterRegisterHook)
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
		it.Event = new(SageRegistryV2AfterRegisterHook)
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
func (it *SageRegistryV2AfterRegisterHookIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SageRegistryV2AfterRegisterHookIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SageRegistryV2AfterRegisterHook represents a AfterRegisterHook event raised by the SageRegistryV2 contract.
type SageRegistryV2AfterRegisterHook struct {
	AgentId [32]byte
	Owner   common.Address
	Data    []byte
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterAfterRegisterHook is a free log retrieval operation binding the contract event 0x3cb7ebe7deaec2743a657dc0d45c0ae4aaae6befdb78e20ca9f93d6d0023893c.
//
// Solidity: event AfterRegisterHook(bytes32 indexed agentId, address indexed owner, bytes data)
func (_SageRegistryV2 *SageRegistryV2Filterer) FilterAfterRegisterHook(opts *bind.FilterOpts, agentId [][32]byte, owner []common.Address) (*SageRegistryV2AfterRegisterHookIterator, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}
	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _SageRegistryV2.contract.FilterLogs(opts, "AfterRegisterHook", agentIdRule, ownerRule)
	if err != nil {
		return nil, err
	}
	return &SageRegistryV2AfterRegisterHookIterator{contract: _SageRegistryV2.contract, event: "AfterRegisterHook", logs: logs, sub: sub}, nil
}

// WatchAfterRegisterHook is a free log subscription operation binding the contract event 0x3cb7ebe7deaec2743a657dc0d45c0ae4aaae6befdb78e20ca9f93d6d0023893c.
//
// Solidity: event AfterRegisterHook(bytes32 indexed agentId, address indexed owner, bytes data)
func (_SageRegistryV2 *SageRegistryV2Filterer) WatchAfterRegisterHook(opts *bind.WatchOpts, sink chan<- *SageRegistryV2AfterRegisterHook, agentId [][32]byte, owner []common.Address) (event.Subscription, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}
	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _SageRegistryV2.contract.WatchLogs(opts, "AfterRegisterHook", agentIdRule, ownerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SageRegistryV2AfterRegisterHook)
				if err := _SageRegistryV2.contract.UnpackLog(event, "AfterRegisterHook", log); err != nil {
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

// ParseAfterRegisterHook is a log parse operation binding the contract event 0x3cb7ebe7deaec2743a657dc0d45c0ae4aaae6befdb78e20ca9f93d6d0023893c.
//
// Solidity: event AfterRegisterHook(bytes32 indexed agentId, address indexed owner, bytes data)
func (_SageRegistryV2 *SageRegistryV2Filterer) ParseAfterRegisterHook(log types.Log) (*SageRegistryV2AfterRegisterHook, error) {
	event := new(SageRegistryV2AfterRegisterHook)
	if err := _SageRegistryV2.contract.UnpackLog(event, "AfterRegisterHook", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SageRegistryV2AfterRegisterHookUpdatedIterator is returned from FilterAfterRegisterHookUpdated and is used to iterate over the raw logs and unpacked data for AfterRegisterHookUpdated events raised by the SageRegistryV2 contract.
type SageRegistryV2AfterRegisterHookUpdatedIterator struct {
	Event *SageRegistryV2AfterRegisterHookUpdated // Event containing the contract specifics and raw log

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
func (it *SageRegistryV2AfterRegisterHookUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SageRegistryV2AfterRegisterHookUpdated)
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
		it.Event = new(SageRegistryV2AfterRegisterHookUpdated)
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
func (it *SageRegistryV2AfterRegisterHookUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SageRegistryV2AfterRegisterHookUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SageRegistryV2AfterRegisterHookUpdated represents a AfterRegisterHookUpdated event raised by the SageRegistryV2 contract.
type SageRegistryV2AfterRegisterHookUpdated struct {
	OldHook common.Address
	NewHook common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterAfterRegisterHookUpdated is a free log retrieval operation binding the contract event 0x226d1a327a26320137574d82b07495e502e285211e430f461d1ead649cb207d1.
//
// Solidity: event AfterRegisterHookUpdated(address indexed oldHook, address indexed newHook)
func (_SageRegistryV2 *SageRegistryV2Filterer) FilterAfterRegisterHookUpdated(opts *bind.FilterOpts, oldHook []common.Address, newHook []common.Address) (*SageRegistryV2AfterRegisterHookUpdatedIterator, error) {

	var oldHookRule []interface{}
	for _, oldHookItem := range oldHook {
		oldHookRule = append(oldHookRule, oldHookItem)
	}
	var newHookRule []interface{}
	for _, newHookItem := range newHook {
		newHookRule = append(newHookRule, newHookItem)
	}

	logs, sub, err := _SageRegistryV2.contract.FilterLogs(opts, "AfterRegisterHookUpdated", oldHookRule, newHookRule)
	if err != nil {
		return nil, err
	}
	return &SageRegistryV2AfterRegisterHookUpdatedIterator{contract: _SageRegistryV2.contract, event: "AfterRegisterHookUpdated", logs: logs, sub: sub}, nil
}

// WatchAfterRegisterHookUpdated is a free log subscription operation binding the contract event 0x226d1a327a26320137574d82b07495e502e285211e430f461d1ead649cb207d1.
//
// Solidity: event AfterRegisterHookUpdated(address indexed oldHook, address indexed newHook)
func (_SageRegistryV2 *SageRegistryV2Filterer) WatchAfterRegisterHookUpdated(opts *bind.WatchOpts, sink chan<- *SageRegistryV2AfterRegisterHookUpdated, oldHook []common.Address, newHook []common.Address) (event.Subscription, error) {

	var oldHookRule []interface{}
	for _, oldHookItem := range oldHook {
		oldHookRule = append(oldHookRule, oldHookItem)
	}
	var newHookRule []interface{}
	for _, newHookItem := range newHook {
		newHookRule = append(newHookRule, newHookItem)
	}

	logs, sub, err := _SageRegistryV2.contract.WatchLogs(opts, "AfterRegisterHookUpdated", oldHookRule, newHookRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SageRegistryV2AfterRegisterHookUpdated)
				if err := _SageRegistryV2.contract.UnpackLog(event, "AfterRegisterHookUpdated", log); err != nil {
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

// ParseAfterRegisterHookUpdated is a log parse operation binding the contract event 0x226d1a327a26320137574d82b07495e502e285211e430f461d1ead649cb207d1.
//
// Solidity: event AfterRegisterHookUpdated(address indexed oldHook, address indexed newHook)
func (_SageRegistryV2 *SageRegistryV2Filterer) ParseAfterRegisterHookUpdated(log types.Log) (*SageRegistryV2AfterRegisterHookUpdated, error) {
	event := new(SageRegistryV2AfterRegisterHookUpdated)
	if err := _SageRegistryV2.contract.UnpackLog(event, "AfterRegisterHookUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SageRegistryV2AgentDeactivatedIterator is returned from FilterAgentDeactivated and is used to iterate over the raw logs and unpacked data for AgentDeactivated events raised by the SageRegistryV2 contract.
type SageRegistryV2AgentDeactivatedIterator struct {
	Event *SageRegistryV2AgentDeactivated // Event containing the contract specifics and raw log

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
func (it *SageRegistryV2AgentDeactivatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SageRegistryV2AgentDeactivated)
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
		it.Event = new(SageRegistryV2AgentDeactivated)
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
func (it *SageRegistryV2AgentDeactivatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SageRegistryV2AgentDeactivatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SageRegistryV2AgentDeactivated represents a AgentDeactivated event raised by the SageRegistryV2 contract.
type SageRegistryV2AgentDeactivated struct {
	AgentId   [32]byte
	Owner     common.Address
	Timestamp *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterAgentDeactivated is a free log retrieval operation binding the contract event 0x529469922704beaa9a686518bed28d19385256536629b42365b7f4d9caca13f1.
//
// Solidity: event AgentDeactivated(bytes32 indexed agentId, address indexed owner, uint256 timestamp)
func (_SageRegistryV2 *SageRegistryV2Filterer) FilterAgentDeactivated(opts *bind.FilterOpts, agentId [][32]byte, owner []common.Address) (*SageRegistryV2AgentDeactivatedIterator, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}
	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _SageRegistryV2.contract.FilterLogs(opts, "AgentDeactivated", agentIdRule, ownerRule)
	if err != nil {
		return nil, err
	}
	return &SageRegistryV2AgentDeactivatedIterator{contract: _SageRegistryV2.contract, event: "AgentDeactivated", logs: logs, sub: sub}, nil
}

// WatchAgentDeactivated is a free log subscription operation binding the contract event 0x529469922704beaa9a686518bed28d19385256536629b42365b7f4d9caca13f1.
//
// Solidity: event AgentDeactivated(bytes32 indexed agentId, address indexed owner, uint256 timestamp)
func (_SageRegistryV2 *SageRegistryV2Filterer) WatchAgentDeactivated(opts *bind.WatchOpts, sink chan<- *SageRegistryV2AgentDeactivated, agentId [][32]byte, owner []common.Address) (event.Subscription, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}
	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _SageRegistryV2.contract.WatchLogs(opts, "AgentDeactivated", agentIdRule, ownerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SageRegistryV2AgentDeactivated)
				if err := _SageRegistryV2.contract.UnpackLog(event, "AgentDeactivated", log); err != nil {
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

// ParseAgentDeactivated is a log parse operation binding the contract event 0x529469922704beaa9a686518bed28d19385256536629b42365b7f4d9caca13f1.
//
// Solidity: event AgentDeactivated(bytes32 indexed agentId, address indexed owner, uint256 timestamp)
func (_SageRegistryV2 *SageRegistryV2Filterer) ParseAgentDeactivated(log types.Log) (*SageRegistryV2AgentDeactivated, error) {
	event := new(SageRegistryV2AgentDeactivated)
	if err := _SageRegistryV2.contract.UnpackLog(event, "AgentDeactivated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SageRegistryV2AgentRegisteredIterator is returned from FilterAgentRegistered and is used to iterate over the raw logs and unpacked data for AgentRegistered events raised by the SageRegistryV2 contract.
type SageRegistryV2AgentRegisteredIterator struct {
	Event *SageRegistryV2AgentRegistered // Event containing the contract specifics and raw log

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
func (it *SageRegistryV2AgentRegisteredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SageRegistryV2AgentRegistered)
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
		it.Event = new(SageRegistryV2AgentRegistered)
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
func (it *SageRegistryV2AgentRegisteredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SageRegistryV2AgentRegisteredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SageRegistryV2AgentRegistered represents a AgentRegistered event raised by the SageRegistryV2 contract.
type SageRegistryV2AgentRegistered struct {
	AgentId   [32]byte
	Owner     common.Address
	Did       string
	Timestamp *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterAgentRegistered is a free log retrieval operation binding the contract event 0x848b086b4ab56ffb70fbcbb34fd5e8f35d1dd5347ee5344efbe6c0f5b97c70f4.
//
// Solidity: event AgentRegistered(bytes32 indexed agentId, address indexed owner, string did, uint256 timestamp)
func (_SageRegistryV2 *SageRegistryV2Filterer) FilterAgentRegistered(opts *bind.FilterOpts, agentId [][32]byte, owner []common.Address) (*SageRegistryV2AgentRegisteredIterator, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}
	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _SageRegistryV2.contract.FilterLogs(opts, "AgentRegistered", agentIdRule, ownerRule)
	if err != nil {
		return nil, err
	}
	return &SageRegistryV2AgentRegisteredIterator{contract: _SageRegistryV2.contract, event: "AgentRegistered", logs: logs, sub: sub}, nil
}

// WatchAgentRegistered is a free log subscription operation binding the contract event 0x848b086b4ab56ffb70fbcbb34fd5e8f35d1dd5347ee5344efbe6c0f5b97c70f4.
//
// Solidity: event AgentRegistered(bytes32 indexed agentId, address indexed owner, string did, uint256 timestamp)
func (_SageRegistryV2 *SageRegistryV2Filterer) WatchAgentRegistered(opts *bind.WatchOpts, sink chan<- *SageRegistryV2AgentRegistered, agentId [][32]byte, owner []common.Address) (event.Subscription, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}
	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _SageRegistryV2.contract.WatchLogs(opts, "AgentRegistered", agentIdRule, ownerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SageRegistryV2AgentRegistered)
				if err := _SageRegistryV2.contract.UnpackLog(event, "AgentRegistered", log); err != nil {
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

// ParseAgentRegistered is a log parse operation binding the contract event 0x848b086b4ab56ffb70fbcbb34fd5e8f35d1dd5347ee5344efbe6c0f5b97c70f4.
//
// Solidity: event AgentRegistered(bytes32 indexed agentId, address indexed owner, string did, uint256 timestamp)
func (_SageRegistryV2 *SageRegistryV2Filterer) ParseAgentRegistered(log types.Log) (*SageRegistryV2AgentRegistered, error) {
	event := new(SageRegistryV2AgentRegistered)
	if err := _SageRegistryV2.contract.UnpackLog(event, "AgentRegistered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SageRegistryV2AgentUpdatedIterator is returned from FilterAgentUpdated and is used to iterate over the raw logs and unpacked data for AgentUpdated events raised by the SageRegistryV2 contract.
type SageRegistryV2AgentUpdatedIterator struct {
	Event *SageRegistryV2AgentUpdated // Event containing the contract specifics and raw log

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
func (it *SageRegistryV2AgentUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SageRegistryV2AgentUpdated)
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
		it.Event = new(SageRegistryV2AgentUpdated)
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
func (it *SageRegistryV2AgentUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SageRegistryV2AgentUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SageRegistryV2AgentUpdated represents a AgentUpdated event raised by the SageRegistryV2 contract.
type SageRegistryV2AgentUpdated struct {
	AgentId   [32]byte
	Owner     common.Address
	Timestamp *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterAgentUpdated is a free log retrieval operation binding the contract event 0xb28fb12b8366d2fb9a1adf15f6b59fcccc9e3b377eb5db8dcdc758c055dde5e5.
//
// Solidity: event AgentUpdated(bytes32 indexed agentId, address indexed owner, uint256 timestamp)
func (_SageRegistryV2 *SageRegistryV2Filterer) FilterAgentUpdated(opts *bind.FilterOpts, agentId [][32]byte, owner []common.Address) (*SageRegistryV2AgentUpdatedIterator, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}
	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _SageRegistryV2.contract.FilterLogs(opts, "AgentUpdated", agentIdRule, ownerRule)
	if err != nil {
		return nil, err
	}
	return &SageRegistryV2AgentUpdatedIterator{contract: _SageRegistryV2.contract, event: "AgentUpdated", logs: logs, sub: sub}, nil
}

// WatchAgentUpdated is a free log subscription operation binding the contract event 0xb28fb12b8366d2fb9a1adf15f6b59fcccc9e3b377eb5db8dcdc758c055dde5e5.
//
// Solidity: event AgentUpdated(bytes32 indexed agentId, address indexed owner, uint256 timestamp)
func (_SageRegistryV2 *SageRegistryV2Filterer) WatchAgentUpdated(opts *bind.WatchOpts, sink chan<- *SageRegistryV2AgentUpdated, agentId [][32]byte, owner []common.Address) (event.Subscription, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}
	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _SageRegistryV2.contract.WatchLogs(opts, "AgentUpdated", agentIdRule, ownerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SageRegistryV2AgentUpdated)
				if err := _SageRegistryV2.contract.UnpackLog(event, "AgentUpdated", log); err != nil {
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

// ParseAgentUpdated is a log parse operation binding the contract event 0xb28fb12b8366d2fb9a1adf15f6b59fcccc9e3b377eb5db8dcdc758c055dde5e5.
//
// Solidity: event AgentUpdated(bytes32 indexed agentId, address indexed owner, uint256 timestamp)
func (_SageRegistryV2 *SageRegistryV2Filterer) ParseAgentUpdated(log types.Log) (*SageRegistryV2AgentUpdated, error) {
	event := new(SageRegistryV2AgentUpdated)
	if err := _SageRegistryV2.contract.UnpackLog(event, "AgentUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SageRegistryV2BeforeRegisterHookIterator is returned from FilterBeforeRegisterHook and is used to iterate over the raw logs and unpacked data for BeforeRegisterHook events raised by the SageRegistryV2 contract.
type SageRegistryV2BeforeRegisterHookIterator struct {
	Event *SageRegistryV2BeforeRegisterHook // Event containing the contract specifics and raw log

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
func (it *SageRegistryV2BeforeRegisterHookIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SageRegistryV2BeforeRegisterHook)
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
		it.Event = new(SageRegistryV2BeforeRegisterHook)
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
func (it *SageRegistryV2BeforeRegisterHookIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SageRegistryV2BeforeRegisterHookIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SageRegistryV2BeforeRegisterHook represents a BeforeRegisterHook event raised by the SageRegistryV2 contract.
type SageRegistryV2BeforeRegisterHook struct {
	AgentId [32]byte
	Owner   common.Address
	Data    []byte
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterBeforeRegisterHook is a free log retrieval operation binding the contract event 0xe9e7066ed0bb4551380e108afced4a59ed1503dccf6c69f572e8f0b2686b7e6d.
//
// Solidity: event BeforeRegisterHook(bytes32 indexed agentId, address indexed owner, bytes data)
func (_SageRegistryV2 *SageRegistryV2Filterer) FilterBeforeRegisterHook(opts *bind.FilterOpts, agentId [][32]byte, owner []common.Address) (*SageRegistryV2BeforeRegisterHookIterator, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}
	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _SageRegistryV2.contract.FilterLogs(opts, "BeforeRegisterHook", agentIdRule, ownerRule)
	if err != nil {
		return nil, err
	}
	return &SageRegistryV2BeforeRegisterHookIterator{contract: _SageRegistryV2.contract, event: "BeforeRegisterHook", logs: logs, sub: sub}, nil
}

// WatchBeforeRegisterHook is a free log subscription operation binding the contract event 0xe9e7066ed0bb4551380e108afced4a59ed1503dccf6c69f572e8f0b2686b7e6d.
//
// Solidity: event BeforeRegisterHook(bytes32 indexed agentId, address indexed owner, bytes data)
func (_SageRegistryV2 *SageRegistryV2Filterer) WatchBeforeRegisterHook(opts *bind.WatchOpts, sink chan<- *SageRegistryV2BeforeRegisterHook, agentId [][32]byte, owner []common.Address) (event.Subscription, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}
	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _SageRegistryV2.contract.WatchLogs(opts, "BeforeRegisterHook", agentIdRule, ownerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SageRegistryV2BeforeRegisterHook)
				if err := _SageRegistryV2.contract.UnpackLog(event, "BeforeRegisterHook", log); err != nil {
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

// ParseBeforeRegisterHook is a log parse operation binding the contract event 0xe9e7066ed0bb4551380e108afced4a59ed1503dccf6c69f572e8f0b2686b7e6d.
//
// Solidity: event BeforeRegisterHook(bytes32 indexed agentId, address indexed owner, bytes data)
func (_SageRegistryV2 *SageRegistryV2Filterer) ParseBeforeRegisterHook(log types.Log) (*SageRegistryV2BeforeRegisterHook, error) {
	event := new(SageRegistryV2BeforeRegisterHook)
	if err := _SageRegistryV2.contract.UnpackLog(event, "BeforeRegisterHook", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SageRegistryV2BeforeRegisterHookUpdatedIterator is returned from FilterBeforeRegisterHookUpdated and is used to iterate over the raw logs and unpacked data for BeforeRegisterHookUpdated events raised by the SageRegistryV2 contract.
type SageRegistryV2BeforeRegisterHookUpdatedIterator struct {
	Event *SageRegistryV2BeforeRegisterHookUpdated // Event containing the contract specifics and raw log

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
func (it *SageRegistryV2BeforeRegisterHookUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SageRegistryV2BeforeRegisterHookUpdated)
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
		it.Event = new(SageRegistryV2BeforeRegisterHookUpdated)
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
func (it *SageRegistryV2BeforeRegisterHookUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SageRegistryV2BeforeRegisterHookUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SageRegistryV2BeforeRegisterHookUpdated represents a BeforeRegisterHookUpdated event raised by the SageRegistryV2 contract.
type SageRegistryV2BeforeRegisterHookUpdated struct {
	OldHook common.Address
	NewHook common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterBeforeRegisterHookUpdated is a free log retrieval operation binding the contract event 0x386eb72043b00f44a4f50ac829f94d9c8bc42b34f698951c56a3eb06d9833daf.
//
// Solidity: event BeforeRegisterHookUpdated(address indexed oldHook, address indexed newHook)
func (_SageRegistryV2 *SageRegistryV2Filterer) FilterBeforeRegisterHookUpdated(opts *bind.FilterOpts, oldHook []common.Address, newHook []common.Address) (*SageRegistryV2BeforeRegisterHookUpdatedIterator, error) {

	var oldHookRule []interface{}
	for _, oldHookItem := range oldHook {
		oldHookRule = append(oldHookRule, oldHookItem)
	}
	var newHookRule []interface{}
	for _, newHookItem := range newHook {
		newHookRule = append(newHookRule, newHookItem)
	}

	logs, sub, err := _SageRegistryV2.contract.FilterLogs(opts, "BeforeRegisterHookUpdated", oldHookRule, newHookRule)
	if err != nil {
		return nil, err
	}
	return &SageRegistryV2BeforeRegisterHookUpdatedIterator{contract: _SageRegistryV2.contract, event: "BeforeRegisterHookUpdated", logs: logs, sub: sub}, nil
}

// WatchBeforeRegisterHookUpdated is a free log subscription operation binding the contract event 0x386eb72043b00f44a4f50ac829f94d9c8bc42b34f698951c56a3eb06d9833daf.
//
// Solidity: event BeforeRegisterHookUpdated(address indexed oldHook, address indexed newHook)
func (_SageRegistryV2 *SageRegistryV2Filterer) WatchBeforeRegisterHookUpdated(opts *bind.WatchOpts, sink chan<- *SageRegistryV2BeforeRegisterHookUpdated, oldHook []common.Address, newHook []common.Address) (event.Subscription, error) {

	var oldHookRule []interface{}
	for _, oldHookItem := range oldHook {
		oldHookRule = append(oldHookRule, oldHookItem)
	}
	var newHookRule []interface{}
	for _, newHookItem := range newHook {
		newHookRule = append(newHookRule, newHookItem)
	}

	logs, sub, err := _SageRegistryV2.contract.WatchLogs(opts, "BeforeRegisterHookUpdated", oldHookRule, newHookRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SageRegistryV2BeforeRegisterHookUpdated)
				if err := _SageRegistryV2.contract.UnpackLog(event, "BeforeRegisterHookUpdated", log); err != nil {
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

// ParseBeforeRegisterHookUpdated is a log parse operation binding the contract event 0x386eb72043b00f44a4f50ac829f94d9c8bc42b34f698951c56a3eb06d9833daf.
//
// Solidity: event BeforeRegisterHookUpdated(address indexed oldHook, address indexed newHook)
func (_SageRegistryV2 *SageRegistryV2Filterer) ParseBeforeRegisterHookUpdated(log types.Log) (*SageRegistryV2BeforeRegisterHookUpdated, error) {
	event := new(SageRegistryV2BeforeRegisterHookUpdated)
	if err := _SageRegistryV2.contract.UnpackLog(event, "BeforeRegisterHookUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SageRegistryV2HookFailedIterator is returned from FilterHookFailed and is used to iterate over the raw logs and unpacked data for HookFailed events raised by the SageRegistryV2 contract.
type SageRegistryV2HookFailedIterator struct {
	Event *SageRegistryV2HookFailed // Event containing the contract specifics and raw log

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
func (it *SageRegistryV2HookFailedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SageRegistryV2HookFailed)
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
		it.Event = new(SageRegistryV2HookFailed)
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
func (it *SageRegistryV2HookFailedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SageRegistryV2HookFailedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SageRegistryV2HookFailed represents a HookFailed event raised by the SageRegistryV2 contract.
type SageRegistryV2HookFailed struct {
	Hook   common.Address
	Reason string
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterHookFailed is a free log retrieval operation binding the contract event 0xfc062a14be4303d562bf8415af1444dc4bba4de2470abec80b31a0e55255a6d8.
//
// Solidity: event HookFailed(address indexed hook, string reason)
func (_SageRegistryV2 *SageRegistryV2Filterer) FilterHookFailed(opts *bind.FilterOpts, hook []common.Address) (*SageRegistryV2HookFailedIterator, error) {

	var hookRule []interface{}
	for _, hookItem := range hook {
		hookRule = append(hookRule, hookItem)
	}

	logs, sub, err := _SageRegistryV2.contract.FilterLogs(opts, "HookFailed", hookRule)
	if err != nil {
		return nil, err
	}
	return &SageRegistryV2HookFailedIterator{contract: _SageRegistryV2.contract, event: "HookFailed", logs: logs, sub: sub}, nil
}

// WatchHookFailed is a free log subscription operation binding the contract event 0xfc062a14be4303d562bf8415af1444dc4bba4de2470abec80b31a0e55255a6d8.
//
// Solidity: event HookFailed(address indexed hook, string reason)
func (_SageRegistryV2 *SageRegistryV2Filterer) WatchHookFailed(opts *bind.WatchOpts, sink chan<- *SageRegistryV2HookFailed, hook []common.Address) (event.Subscription, error) {

	var hookRule []interface{}
	for _, hookItem := range hook {
		hookRule = append(hookRule, hookItem)
	}

	logs, sub, err := _SageRegistryV2.contract.WatchLogs(opts, "HookFailed", hookRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SageRegistryV2HookFailed)
				if err := _SageRegistryV2.contract.UnpackLog(event, "HookFailed", log); err != nil {
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

// ParseHookFailed is a log parse operation binding the contract event 0xfc062a14be4303d562bf8415af1444dc4bba4de2470abec80b31a0e55255a6d8.
//
// Solidity: event HookFailed(address indexed hook, string reason)
func (_SageRegistryV2 *SageRegistryV2Filterer) ParseHookFailed(log types.Log) (*SageRegistryV2HookFailed, error) {
	event := new(SageRegistryV2HookFailed)
	if err := _SageRegistryV2.contract.UnpackLog(event, "HookFailed", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SageRegistryV2KeyRevokedIterator is returned from FilterKeyRevoked and is used to iterate over the raw logs and unpacked data for KeyRevoked events raised by the SageRegistryV2 contract.
type SageRegistryV2KeyRevokedIterator struct {
	Event *SageRegistryV2KeyRevoked // Event containing the contract specifics and raw log

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
func (it *SageRegistryV2KeyRevokedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SageRegistryV2KeyRevoked)
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
		it.Event = new(SageRegistryV2KeyRevoked)
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
func (it *SageRegistryV2KeyRevokedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SageRegistryV2KeyRevokedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SageRegistryV2KeyRevoked represents a KeyRevoked event raised by the SageRegistryV2 contract.
type SageRegistryV2KeyRevoked struct {
	KeyHash [32]byte
	Owner   common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterKeyRevoked is a free log retrieval operation binding the contract event 0x607ba79db6282926774611ed828760d8d8a7d1266ad896b9aaf81f6ba883192c.
//
// Solidity: event KeyRevoked(bytes32 indexed keyHash, address indexed owner)
func (_SageRegistryV2 *SageRegistryV2Filterer) FilterKeyRevoked(opts *bind.FilterOpts, keyHash [][32]byte, owner []common.Address) (*SageRegistryV2KeyRevokedIterator, error) {

	var keyHashRule []interface{}
	for _, keyHashItem := range keyHash {
		keyHashRule = append(keyHashRule, keyHashItem)
	}
	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _SageRegistryV2.contract.FilterLogs(opts, "KeyRevoked", keyHashRule, ownerRule)
	if err != nil {
		return nil, err
	}
	return &SageRegistryV2KeyRevokedIterator{contract: _SageRegistryV2.contract, event: "KeyRevoked", logs: logs, sub: sub}, nil
}

// WatchKeyRevoked is a free log subscription operation binding the contract event 0x607ba79db6282926774611ed828760d8d8a7d1266ad896b9aaf81f6ba883192c.
//
// Solidity: event KeyRevoked(bytes32 indexed keyHash, address indexed owner)
func (_SageRegistryV2 *SageRegistryV2Filterer) WatchKeyRevoked(opts *bind.WatchOpts, sink chan<- *SageRegistryV2KeyRevoked, keyHash [][32]byte, owner []common.Address) (event.Subscription, error) {

	var keyHashRule []interface{}
	for _, keyHashItem := range keyHash {
		keyHashRule = append(keyHashRule, keyHashItem)
	}
	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _SageRegistryV2.contract.WatchLogs(opts, "KeyRevoked", keyHashRule, ownerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SageRegistryV2KeyRevoked)
				if err := _SageRegistryV2.contract.UnpackLog(event, "KeyRevoked", log); err != nil {
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

// ParseKeyRevoked is a log parse operation binding the contract event 0x607ba79db6282926774611ed828760d8d8a7d1266ad896b9aaf81f6ba883192c.
//
// Solidity: event KeyRevoked(bytes32 indexed keyHash, address indexed owner)
func (_SageRegistryV2 *SageRegistryV2Filterer) ParseKeyRevoked(log types.Log) (*SageRegistryV2KeyRevoked, error) {
	event := new(SageRegistryV2KeyRevoked)
	if err := _SageRegistryV2.contract.UnpackLog(event, "KeyRevoked", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SageRegistryV2KeyValidatedIterator is returned from FilterKeyValidated and is used to iterate over the raw logs and unpacked data for KeyValidated events raised by the SageRegistryV2 contract.
type SageRegistryV2KeyValidatedIterator struct {
	Event *SageRegistryV2KeyValidated // Event containing the contract specifics and raw log

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
func (it *SageRegistryV2KeyValidatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SageRegistryV2KeyValidated)
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
		it.Event = new(SageRegistryV2KeyValidated)
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
func (it *SageRegistryV2KeyValidatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SageRegistryV2KeyValidatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SageRegistryV2KeyValidated represents a KeyValidated event raised by the SageRegistryV2 contract.
type SageRegistryV2KeyValidated struct {
	KeyHash [32]byte
	Owner   common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterKeyValidated is a free log retrieval operation binding the contract event 0xdb3f29cf64872daa542625c243dda32c68005c7267a9541527826dd54044a0b1.
//
// Solidity: event KeyValidated(bytes32 indexed keyHash, address indexed owner)
func (_SageRegistryV2 *SageRegistryV2Filterer) FilterKeyValidated(opts *bind.FilterOpts, keyHash [][32]byte, owner []common.Address) (*SageRegistryV2KeyValidatedIterator, error) {

	var keyHashRule []interface{}
	for _, keyHashItem := range keyHash {
		keyHashRule = append(keyHashRule, keyHashItem)
	}
	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _SageRegistryV2.contract.FilterLogs(opts, "KeyValidated", keyHashRule, ownerRule)
	if err != nil {
		return nil, err
	}
	return &SageRegistryV2KeyValidatedIterator{contract: _SageRegistryV2.contract, event: "KeyValidated", logs: logs, sub: sub}, nil
}

// WatchKeyValidated is a free log subscription operation binding the contract event 0xdb3f29cf64872daa542625c243dda32c68005c7267a9541527826dd54044a0b1.
//
// Solidity: event KeyValidated(bytes32 indexed keyHash, address indexed owner)
func (_SageRegistryV2 *SageRegistryV2Filterer) WatchKeyValidated(opts *bind.WatchOpts, sink chan<- *SageRegistryV2KeyValidated, keyHash [][32]byte, owner []common.Address) (event.Subscription, error) {

	var keyHashRule []interface{}
	for _, keyHashItem := range keyHash {
		keyHashRule = append(keyHashRule, keyHashItem)
	}
	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _SageRegistryV2.contract.WatchLogs(opts, "KeyValidated", keyHashRule, ownerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SageRegistryV2KeyValidated)
				if err := _SageRegistryV2.contract.UnpackLog(event, "KeyValidated", log); err != nil {
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

// ParseKeyValidated is a log parse operation binding the contract event 0xdb3f29cf64872daa542625c243dda32c68005c7267a9541527826dd54044a0b1.
//
// Solidity: event KeyValidated(bytes32 indexed keyHash, address indexed owner)
func (_SageRegistryV2 *SageRegistryV2Filterer) ParseKeyValidated(log types.Log) (*SageRegistryV2KeyValidated, error) {
	event := new(SageRegistryV2KeyValidated)
	if err := _SageRegistryV2.contract.UnpackLog(event, "KeyValidated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SageRegistryV2OwnershipTransferStartedIterator is returned from FilterOwnershipTransferStarted and is used to iterate over the raw logs and unpacked data for OwnershipTransferStarted events raised by the SageRegistryV2 contract.
type SageRegistryV2OwnershipTransferStartedIterator struct {
	Event *SageRegistryV2OwnershipTransferStarted // Event containing the contract specifics and raw log

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
func (it *SageRegistryV2OwnershipTransferStartedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SageRegistryV2OwnershipTransferStarted)
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
		it.Event = new(SageRegistryV2OwnershipTransferStarted)
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
func (it *SageRegistryV2OwnershipTransferStartedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SageRegistryV2OwnershipTransferStartedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SageRegistryV2OwnershipTransferStarted represents a OwnershipTransferStarted event raised by the SageRegistryV2 contract.
type SageRegistryV2OwnershipTransferStarted struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferStarted is a free log retrieval operation binding the contract event 0x38d16b8cac22d99fc7c124b9cd0de2d3fa1faef420bfe791d8c362d765e22700.
//
// Solidity: event OwnershipTransferStarted(address indexed previousOwner, address indexed newOwner)
func (_SageRegistryV2 *SageRegistryV2Filterer) FilterOwnershipTransferStarted(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*SageRegistryV2OwnershipTransferStartedIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _SageRegistryV2.contract.FilterLogs(opts, "OwnershipTransferStarted", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &SageRegistryV2OwnershipTransferStartedIterator{contract: _SageRegistryV2.contract, event: "OwnershipTransferStarted", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferStarted is a free log subscription operation binding the contract event 0x38d16b8cac22d99fc7c124b9cd0de2d3fa1faef420bfe791d8c362d765e22700.
//
// Solidity: event OwnershipTransferStarted(address indexed previousOwner, address indexed newOwner)
func (_SageRegistryV2 *SageRegistryV2Filterer) WatchOwnershipTransferStarted(opts *bind.WatchOpts, sink chan<- *SageRegistryV2OwnershipTransferStarted, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _SageRegistryV2.contract.WatchLogs(opts, "OwnershipTransferStarted", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SageRegistryV2OwnershipTransferStarted)
				if err := _SageRegistryV2.contract.UnpackLog(event, "OwnershipTransferStarted", log); err != nil {
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
func (_SageRegistryV2 *SageRegistryV2Filterer) ParseOwnershipTransferStarted(log types.Log) (*SageRegistryV2OwnershipTransferStarted, error) {
	event := new(SageRegistryV2OwnershipTransferStarted)
	if err := _SageRegistryV2.contract.UnpackLog(event, "OwnershipTransferStarted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SageRegistryV2OwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the SageRegistryV2 contract.
type SageRegistryV2OwnershipTransferredIterator struct {
	Event *SageRegistryV2OwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *SageRegistryV2OwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SageRegistryV2OwnershipTransferred)
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
		it.Event = new(SageRegistryV2OwnershipTransferred)
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
func (it *SageRegistryV2OwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SageRegistryV2OwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SageRegistryV2OwnershipTransferred represents a OwnershipTransferred event raised by the SageRegistryV2 contract.
type SageRegistryV2OwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_SageRegistryV2 *SageRegistryV2Filterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*SageRegistryV2OwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _SageRegistryV2.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &SageRegistryV2OwnershipTransferredIterator{contract: _SageRegistryV2.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_SageRegistryV2 *SageRegistryV2Filterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *SageRegistryV2OwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _SageRegistryV2.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SageRegistryV2OwnershipTransferred)
				if err := _SageRegistryV2.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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
func (_SageRegistryV2 *SageRegistryV2Filterer) ParseOwnershipTransferred(log types.Log) (*SageRegistryV2OwnershipTransferred, error) {
	event := new(SageRegistryV2OwnershipTransferred)
	if err := _SageRegistryV2.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SageRegistryV2PausedIterator is returned from FilterPaused and is used to iterate over the raw logs and unpacked data for Paused events raised by the SageRegistryV2 contract.
type SageRegistryV2PausedIterator struct {
	Event *SageRegistryV2Paused // Event containing the contract specifics and raw log

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
func (it *SageRegistryV2PausedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SageRegistryV2Paused)
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
		it.Event = new(SageRegistryV2Paused)
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
func (it *SageRegistryV2PausedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SageRegistryV2PausedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SageRegistryV2Paused represents a Paused event raised by the SageRegistryV2 contract.
type SageRegistryV2Paused struct {
	Account common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterPaused is a free log retrieval operation binding the contract event 0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258.
//
// Solidity: event Paused(address account)
func (_SageRegistryV2 *SageRegistryV2Filterer) FilterPaused(opts *bind.FilterOpts) (*SageRegistryV2PausedIterator, error) {

	logs, sub, err := _SageRegistryV2.contract.FilterLogs(opts, "Paused")
	if err != nil {
		return nil, err
	}
	return &SageRegistryV2PausedIterator{contract: _SageRegistryV2.contract, event: "Paused", logs: logs, sub: sub}, nil
}

// WatchPaused is a free log subscription operation binding the contract event 0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258.
//
// Solidity: event Paused(address account)
func (_SageRegistryV2 *SageRegistryV2Filterer) WatchPaused(opts *bind.WatchOpts, sink chan<- *SageRegistryV2Paused) (event.Subscription, error) {

	logs, sub, err := _SageRegistryV2.contract.WatchLogs(opts, "Paused")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SageRegistryV2Paused)
				if err := _SageRegistryV2.contract.UnpackLog(event, "Paused", log); err != nil {
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
func (_SageRegistryV2 *SageRegistryV2Filterer) ParsePaused(log types.Log) (*SageRegistryV2Paused, error) {
	event := new(SageRegistryV2Paused)
	if err := _SageRegistryV2.contract.UnpackLog(event, "Paused", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SageRegistryV2UnpausedIterator is returned from FilterUnpaused and is used to iterate over the raw logs and unpacked data for Unpaused events raised by the SageRegistryV2 contract.
type SageRegistryV2UnpausedIterator struct {
	Event *SageRegistryV2Unpaused // Event containing the contract specifics and raw log

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
func (it *SageRegistryV2UnpausedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SageRegistryV2Unpaused)
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
		it.Event = new(SageRegistryV2Unpaused)
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
func (it *SageRegistryV2UnpausedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SageRegistryV2UnpausedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SageRegistryV2Unpaused represents a Unpaused event raised by the SageRegistryV2 contract.
type SageRegistryV2Unpaused struct {
	Account common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterUnpaused is a free log retrieval operation binding the contract event 0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa.
//
// Solidity: event Unpaused(address account)
func (_SageRegistryV2 *SageRegistryV2Filterer) FilterUnpaused(opts *bind.FilterOpts) (*SageRegistryV2UnpausedIterator, error) {

	logs, sub, err := _SageRegistryV2.contract.FilterLogs(opts, "Unpaused")
	if err != nil {
		return nil, err
	}
	return &SageRegistryV2UnpausedIterator{contract: _SageRegistryV2.contract, event: "Unpaused", logs: logs, sub: sub}, nil
}

// WatchUnpaused is a free log subscription operation binding the contract event 0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa.
//
// Solidity: event Unpaused(address account)
func (_SageRegistryV2 *SageRegistryV2Filterer) WatchUnpaused(opts *bind.WatchOpts, sink chan<- *SageRegistryV2Unpaused) (event.Subscription, error) {

	logs, sub, err := _SageRegistryV2.contract.WatchLogs(opts, "Unpaused")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SageRegistryV2Unpaused)
				if err := _SageRegistryV2.contract.UnpackLog(event, "Unpaused", log); err != nil {
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
func (_SageRegistryV2 *SageRegistryV2Filterer) ParseUnpaused(log types.Log) (*SageRegistryV2Unpaused, error) {
	event := new(SageRegistryV2Unpaused)
	if err := _SageRegistryV2.contract.UnpackLog(event, "Unpaused", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

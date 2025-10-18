// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package registryv4

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

// ISageRegistryV4AgentKey is an auto generated low-level Go binding around an user-defined struct.
type ISageRegistryV4AgentKey struct {
	KeyType      uint8
	KeyData      []byte
	Signature    []byte
	Verified     bool
	RegisteredAt *big.Int
}

// ISageRegistryV4AgentMetadata is an auto generated low-level Go binding around an user-defined struct.
type ISageRegistryV4AgentMetadata struct {
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
}

// ISageRegistryV4RegistrationParams is an auto generated low-level Go binding around an user-defined struct.
type ISageRegistryV4RegistrationParams struct {
	Did          string
	Name         string
	Description  string
	Endpoint     string
	KeyTypes     []uint8
	KeyData      [][]byte
	Signatures   [][]byte
	Capabilities string
}

// SageRegistryV4MetaData contains all meta data concerning the SageRegistryV4 contract.
var SageRegistryV4MetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"caller\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"hookData\",\"type\":\"bytes\"}],\"name\":\"AfterRegisterHook\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"}],\"name\":\"AgentDeactivated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"did\",\"type\":\"string\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"}],\"name\":\"AgentRegistered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"}],\"name\":\"AgentUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"caller\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"hookData\",\"type\":\"bytes\"}],\"name\":\"BeforeRegisterHook\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"keyHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"}],\"name\":\"Ed25519KeyApproved\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"keyHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"enumISageRegistryV4.KeyType\",\"name\":\"keyType\",\"type\":\"uint8\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"}],\"name\":\"KeyAdded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"keyHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"}],\"name\":\"KeyRevoked\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"OWNER\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"},{\"internalType\":\"enumISageRegistryV4.KeyType\",\"name\":\"keyType\",\"type\":\"uint8\"},{\"internalType\":\"bytes\",\"name\":\"keyData\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"signature\",\"type\":\"bytes\"}],\"name\":\"addKey\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"afterRegisterHook\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"keyHash\",\"type\":\"bytes32\"}],\"name\":\"approveEd25519Key\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"beforeRegisterHook\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"}],\"name\":\"deactivateAgent\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"}],\"name\":\"getAgent\",\"outputs\":[{\"components\":[{\"internalType\":\"string\",\"name\":\"did\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"description\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"endpoint\",\"type\":\"string\"},{\"internalType\":\"bytes32[]\",\"name\":\"keyHashes\",\"type\":\"bytes32[]\"},{\"internalType\":\"string\",\"name\":\"capabilities\",\"type\":\"string\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"registeredAt\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"updatedAt\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"active\",\"type\":\"bool\"}],\"internalType\":\"structISageRegistryV4.AgentMetadata\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"did\",\"type\":\"string\"}],\"name\":\"getAgentByDID\",\"outputs\":[{\"components\":[{\"internalType\":\"string\",\"name\":\"did\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"description\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"endpoint\",\"type\":\"string\"},{\"internalType\":\"bytes32[]\",\"name\":\"keyHashes\",\"type\":\"bytes32[]\"},{\"internalType\":\"string\",\"name\":\"capabilities\",\"type\":\"string\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"registeredAt\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"updatedAt\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"active\",\"type\":\"bool\"}],\"internalType\":\"structISageRegistryV4.AgentMetadata\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"}],\"name\":\"getAgentKeys\",\"outputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"\",\"type\":\"bytes32[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"ownerAddress\",\"type\":\"address\"}],\"name\":\"getAgentsByOwner\",\"outputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"\",\"type\":\"bytes32[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"keyHash\",\"type\":\"bytes32\"}],\"name\":\"getKey\",\"outputs\":[{\"components\":[{\"internalType\":\"enumISageRegistryV4.KeyType\",\"name\":\"keyType\",\"type\":\"uint8\"},{\"internalType\":\"bytes\",\"name\":\"keyData\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"signature\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"verified\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"registeredAt\",\"type\":\"uint256\"}],\"internalType\":\"structISageRegistryV4.AgentKey\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"}],\"name\":\"isAgentActive\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"string\",\"name\":\"did\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"description\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"endpoint\",\"type\":\"string\"},{\"internalType\":\"enumISageRegistryV4.KeyType[]\",\"name\":\"keyTypes\",\"type\":\"uint8[]\"},{\"internalType\":\"bytes[]\",\"name\":\"keyData\",\"type\":\"bytes[]\"},{\"internalType\":\"bytes[]\",\"name\":\"signatures\",\"type\":\"bytes[]\"},{\"internalType\":\"string\",\"name\":\"capabilities\",\"type\":\"string\"}],\"internalType\":\"structISageRegistryV4.RegistrationParams\",\"name\":\"params\",\"type\":\"tuple\"}],\"name\":\"registerAgent\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"keyHash\",\"type\":\"bytes32\"}],\"name\":\"revokeKey\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"hook\",\"type\":\"address\"}],\"name\":\"setAfterRegisterHook\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"hook\",\"type\":\"address\"}],\"name\":\"setBeforeRegisterHook\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"description\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"endpoint\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"capabilities\",\"type\":\"string\"},{\"internalType\":\"bytes\",\"name\":\"signature\",\"type\":\"bytes\"}],\"name\":\"updateAgent\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"claimedOwner\",\"type\":\"address\"}],\"name\":\"verifyAgentOwnership\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// SageRegistryV4ABI is the input ABI used to generate the binding from.
// Deprecated: Use SageRegistryV4MetaData.ABI instead.
var SageRegistryV4ABI = SageRegistryV4MetaData.ABI

// SageRegistryV4 is an auto generated Go binding around an Ethereum contract.
type SageRegistryV4 struct {
	SageRegistryV4Caller     // Read-only binding to the contract
	SageRegistryV4Transactor // Write-only binding to the contract
	SageRegistryV4Filterer   // Log filterer for contract events
}

// SageRegistryV4Caller is an auto generated read-only Go binding around an Ethereum contract.
type SageRegistryV4Caller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SageRegistryV4Transactor is an auto generated write-only Go binding around an Ethereum contract.
type SageRegistryV4Transactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SageRegistryV4Filterer is an auto generated log filtering Go binding around an Ethereum contract events.
type SageRegistryV4Filterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SageRegistryV4Session is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type SageRegistryV4Session struct {
	Contract     *SageRegistryV4   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// SageRegistryV4CallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type SageRegistryV4CallerSession struct {
	Contract *SageRegistryV4Caller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// SageRegistryV4TransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type SageRegistryV4TransactorSession struct {
	Contract     *SageRegistryV4Transactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// SageRegistryV4Raw is an auto generated low-level Go binding around an Ethereum contract.
type SageRegistryV4Raw struct {
	Contract *SageRegistryV4 // Generic contract binding to access the raw methods on
}

// SageRegistryV4CallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type SageRegistryV4CallerRaw struct {
	Contract *SageRegistryV4Caller // Generic read-only contract binding to access the raw methods on
}

// SageRegistryV4TransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type SageRegistryV4TransactorRaw struct {
	Contract *SageRegistryV4Transactor // Generic write-only contract binding to access the raw methods on
}

// NewSageRegistryV4 creates a new instance of SageRegistryV4, bound to a specific deployed contract.
func NewSageRegistryV4(address common.Address, backend bind.ContractBackend) (*SageRegistryV4, error) {
	contract, err := bindSageRegistryV4(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &SageRegistryV4{SageRegistryV4Caller: SageRegistryV4Caller{contract: contract}, SageRegistryV4Transactor: SageRegistryV4Transactor{contract: contract}, SageRegistryV4Filterer: SageRegistryV4Filterer{contract: contract}}, nil
}

// NewSageRegistryV4Caller creates a new read-only instance of SageRegistryV4, bound to a specific deployed contract.
func NewSageRegistryV4Caller(address common.Address, caller bind.ContractCaller) (*SageRegistryV4Caller, error) {
	contract, err := bindSageRegistryV4(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &SageRegistryV4Caller{contract: contract}, nil
}

// NewSageRegistryV4Transactor creates a new write-only instance of SageRegistryV4, bound to a specific deployed contract.
func NewSageRegistryV4Transactor(address common.Address, transactor bind.ContractTransactor) (*SageRegistryV4Transactor, error) {
	contract, err := bindSageRegistryV4(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &SageRegistryV4Transactor{contract: contract}, nil
}

// NewSageRegistryV4Filterer creates a new log filterer instance of SageRegistryV4, bound to a specific deployed contract.
func NewSageRegistryV4Filterer(address common.Address, filterer bind.ContractFilterer) (*SageRegistryV4Filterer, error) {
	contract, err := bindSageRegistryV4(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &SageRegistryV4Filterer{contract: contract}, nil
}

// bindSageRegistryV4 binds a generic wrapper to an already deployed contract.
func bindSageRegistryV4(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := SageRegistryV4MetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SageRegistryV4 *SageRegistryV4Raw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SageRegistryV4.Contract.SageRegistryV4Caller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SageRegistryV4 *SageRegistryV4Raw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SageRegistryV4.Contract.SageRegistryV4Transactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SageRegistryV4 *SageRegistryV4Raw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SageRegistryV4.Contract.SageRegistryV4Transactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SageRegistryV4 *SageRegistryV4CallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SageRegistryV4.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SageRegistryV4 *SageRegistryV4TransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SageRegistryV4.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SageRegistryV4 *SageRegistryV4TransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SageRegistryV4.Contract.contract.Transact(opts, method, params...)
}

// OWNER is a free data retrieval call binding the contract method 0x117803e3.
//
// Solidity: function OWNER() view returns(address)
func (_SageRegistryV4 *SageRegistryV4Caller) OWNER(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SageRegistryV4.contract.Call(opts, &out, "OWNER")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OWNER is a free data retrieval call binding the contract method 0x117803e3.
//
// Solidity: function OWNER() view returns(address)
func (_SageRegistryV4 *SageRegistryV4Session) OWNER() (common.Address, error) {
	return _SageRegistryV4.Contract.OWNER(&_SageRegistryV4.CallOpts)
}

// OWNER is a free data retrieval call binding the contract method 0x117803e3.
//
// Solidity: function OWNER() view returns(address)
func (_SageRegistryV4 *SageRegistryV4CallerSession) OWNER() (common.Address, error) {
	return _SageRegistryV4.Contract.OWNER(&_SageRegistryV4.CallOpts)
}

// AfterRegisterHook is a free data retrieval call binding the contract method 0x24e6c522.
//
// Solidity: function afterRegisterHook() view returns(address)
func (_SageRegistryV4 *SageRegistryV4Caller) AfterRegisterHook(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SageRegistryV4.contract.Call(opts, &out, "afterRegisterHook")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// AfterRegisterHook is a free data retrieval call binding the contract method 0x24e6c522.
//
// Solidity: function afterRegisterHook() view returns(address)
func (_SageRegistryV4 *SageRegistryV4Session) AfterRegisterHook() (common.Address, error) {
	return _SageRegistryV4.Contract.AfterRegisterHook(&_SageRegistryV4.CallOpts)
}

// AfterRegisterHook is a free data retrieval call binding the contract method 0x24e6c522.
//
// Solidity: function afterRegisterHook() view returns(address)
func (_SageRegistryV4 *SageRegistryV4CallerSession) AfterRegisterHook() (common.Address, error) {
	return _SageRegistryV4.Contract.AfterRegisterHook(&_SageRegistryV4.CallOpts)
}

// BeforeRegisterHook is a free data retrieval call binding the contract method 0x8051d5ea.
//
// Solidity: function beforeRegisterHook() view returns(address)
func (_SageRegistryV4 *SageRegistryV4Caller) BeforeRegisterHook(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SageRegistryV4.contract.Call(opts, &out, "beforeRegisterHook")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// BeforeRegisterHook is a free data retrieval call binding the contract method 0x8051d5ea.
//
// Solidity: function beforeRegisterHook() view returns(address)
func (_SageRegistryV4 *SageRegistryV4Session) BeforeRegisterHook() (common.Address, error) {
	return _SageRegistryV4.Contract.BeforeRegisterHook(&_SageRegistryV4.CallOpts)
}

// BeforeRegisterHook is a free data retrieval call binding the contract method 0x8051d5ea.
//
// Solidity: function beforeRegisterHook() view returns(address)
func (_SageRegistryV4 *SageRegistryV4CallerSession) BeforeRegisterHook() (common.Address, error) {
	return _SageRegistryV4.Contract.BeforeRegisterHook(&_SageRegistryV4.CallOpts)
}

// GetAgent is a free data retrieval call binding the contract method 0xa6c2af01.
//
// Solidity: function getAgent(bytes32 agentId) view returns((string,string,string,string,bytes32[],string,address,uint256,uint256,bool))
func (_SageRegistryV4 *SageRegistryV4Caller) GetAgent(opts *bind.CallOpts, agentId [32]byte) (ISageRegistryV4AgentMetadata, error) {
	var out []interface{}
	err := _SageRegistryV4.contract.Call(opts, &out, "getAgent", agentId)

	if err != nil {
		return *new(ISageRegistryV4AgentMetadata), err
	}

	out0 := *abi.ConvertType(out[0], new(ISageRegistryV4AgentMetadata)).(*ISageRegistryV4AgentMetadata)

	return out0, err

}

// GetAgent is a free data retrieval call binding the contract method 0xa6c2af01.
//
// Solidity: function getAgent(bytes32 agentId) view returns((string,string,string,string,bytes32[],string,address,uint256,uint256,bool))
func (_SageRegistryV4 *SageRegistryV4Session) GetAgent(agentId [32]byte) (ISageRegistryV4AgentMetadata, error) {
	return _SageRegistryV4.Contract.GetAgent(&_SageRegistryV4.CallOpts, agentId)
}

// GetAgent is a free data retrieval call binding the contract method 0xa6c2af01.
//
// Solidity: function getAgent(bytes32 agentId) view returns((string,string,string,string,bytes32[],string,address,uint256,uint256,bool))
func (_SageRegistryV4 *SageRegistryV4CallerSession) GetAgent(agentId [32]byte) (ISageRegistryV4AgentMetadata, error) {
	return _SageRegistryV4.Contract.GetAgent(&_SageRegistryV4.CallOpts, agentId)
}

// GetAgentByDID is a free data retrieval call binding the contract method 0xe45d486d.
//
// Solidity: function getAgentByDID(string did) view returns((string,string,string,string,bytes32[],string,address,uint256,uint256,bool))
func (_SageRegistryV4 *SageRegistryV4Caller) GetAgentByDID(opts *bind.CallOpts, did string) (ISageRegistryV4AgentMetadata, error) {
	var out []interface{}
	err := _SageRegistryV4.contract.Call(opts, &out, "getAgentByDID", did)

	if err != nil {
		return *new(ISageRegistryV4AgentMetadata), err
	}

	out0 := *abi.ConvertType(out[0], new(ISageRegistryV4AgentMetadata)).(*ISageRegistryV4AgentMetadata)

	return out0, err

}

// GetAgentByDID is a free data retrieval call binding the contract method 0xe45d486d.
//
// Solidity: function getAgentByDID(string did) view returns((string,string,string,string,bytes32[],string,address,uint256,uint256,bool))
func (_SageRegistryV4 *SageRegistryV4Session) GetAgentByDID(did string) (ISageRegistryV4AgentMetadata, error) {
	return _SageRegistryV4.Contract.GetAgentByDID(&_SageRegistryV4.CallOpts, did)
}

// GetAgentByDID is a free data retrieval call binding the contract method 0xe45d486d.
//
// Solidity: function getAgentByDID(string did) view returns((string,string,string,string,bytes32[],string,address,uint256,uint256,bool))
func (_SageRegistryV4 *SageRegistryV4CallerSession) GetAgentByDID(did string) (ISageRegistryV4AgentMetadata, error) {
	return _SageRegistryV4.Contract.GetAgentByDID(&_SageRegistryV4.CallOpts, did)
}

// GetAgentKeys is a free data retrieval call binding the contract method 0xd21a4764.
//
// Solidity: function getAgentKeys(bytes32 agentId) view returns(bytes32[])
func (_SageRegistryV4 *SageRegistryV4Caller) GetAgentKeys(opts *bind.CallOpts, agentId [32]byte) ([][32]byte, error) {
	var out []interface{}
	err := _SageRegistryV4.contract.Call(opts, &out, "getAgentKeys", agentId)

	if err != nil {
		return *new([][32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([][32]byte)).(*[][32]byte)

	return out0, err

}

// GetAgentKeys is a free data retrieval call binding the contract method 0xd21a4764.
//
// Solidity: function getAgentKeys(bytes32 agentId) view returns(bytes32[])
func (_SageRegistryV4 *SageRegistryV4Session) GetAgentKeys(agentId [32]byte) ([][32]byte, error) {
	return _SageRegistryV4.Contract.GetAgentKeys(&_SageRegistryV4.CallOpts, agentId)
}

// GetAgentKeys is a free data retrieval call binding the contract method 0xd21a4764.
//
// Solidity: function getAgentKeys(bytes32 agentId) view returns(bytes32[])
func (_SageRegistryV4 *SageRegistryV4CallerSession) GetAgentKeys(agentId [32]byte) ([][32]byte, error) {
	return _SageRegistryV4.Contract.GetAgentKeys(&_SageRegistryV4.CallOpts, agentId)
}

// GetAgentsByOwner is a free data retrieval call binding the contract method 0x1ab6f888.
//
// Solidity: function getAgentsByOwner(address ownerAddress) view returns(bytes32[])
func (_SageRegistryV4 *SageRegistryV4Caller) GetAgentsByOwner(opts *bind.CallOpts, ownerAddress common.Address) ([][32]byte, error) {
	var out []interface{}
	err := _SageRegistryV4.contract.Call(opts, &out, "getAgentsByOwner", ownerAddress)

	if err != nil {
		return *new([][32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([][32]byte)).(*[][32]byte)

	return out0, err

}

// GetAgentsByOwner is a free data retrieval call binding the contract method 0x1ab6f888.
//
// Solidity: function getAgentsByOwner(address ownerAddress) view returns(bytes32[])
func (_SageRegistryV4 *SageRegistryV4Session) GetAgentsByOwner(ownerAddress common.Address) ([][32]byte, error) {
	return _SageRegistryV4.Contract.GetAgentsByOwner(&_SageRegistryV4.CallOpts, ownerAddress)
}

// GetAgentsByOwner is a free data retrieval call binding the contract method 0x1ab6f888.
//
// Solidity: function getAgentsByOwner(address ownerAddress) view returns(bytes32[])
func (_SageRegistryV4 *SageRegistryV4CallerSession) GetAgentsByOwner(ownerAddress common.Address) ([][32]byte, error) {
	return _SageRegistryV4.Contract.GetAgentsByOwner(&_SageRegistryV4.CallOpts, ownerAddress)
}

// GetKey is a free data retrieval call binding the contract method 0x12aaac70.
//
// Solidity: function getKey(bytes32 keyHash) view returns((uint8,bytes,bytes,bool,uint256))
func (_SageRegistryV4 *SageRegistryV4Caller) GetKey(opts *bind.CallOpts, keyHash [32]byte) (ISageRegistryV4AgentKey, error) {
	var out []interface{}
	err := _SageRegistryV4.contract.Call(opts, &out, "getKey", keyHash)

	if err != nil {
		return *new(ISageRegistryV4AgentKey), err
	}

	out0 := *abi.ConvertType(out[0], new(ISageRegistryV4AgentKey)).(*ISageRegistryV4AgentKey)

	return out0, err

}

// GetKey is a free data retrieval call binding the contract method 0x12aaac70.
//
// Solidity: function getKey(bytes32 keyHash) view returns((uint8,bytes,bytes,bool,uint256))
func (_SageRegistryV4 *SageRegistryV4Session) GetKey(keyHash [32]byte) (ISageRegistryV4AgentKey, error) {
	return _SageRegistryV4.Contract.GetKey(&_SageRegistryV4.CallOpts, keyHash)
}

// GetKey is a free data retrieval call binding the contract method 0x12aaac70.
//
// Solidity: function getKey(bytes32 keyHash) view returns((uint8,bytes,bytes,bool,uint256))
func (_SageRegistryV4 *SageRegistryV4CallerSession) GetKey(keyHash [32]byte) (ISageRegistryV4AgentKey, error) {
	return _SageRegistryV4.Contract.GetKey(&_SageRegistryV4.CallOpts, keyHash)
}

// IsAgentActive is a free data retrieval call binding the contract method 0x8a92792b.
//
// Solidity: function isAgentActive(bytes32 agentId) view returns(bool)
func (_SageRegistryV4 *SageRegistryV4Caller) IsAgentActive(opts *bind.CallOpts, agentId [32]byte) (bool, error) {
	var out []interface{}
	err := _SageRegistryV4.contract.Call(opts, &out, "isAgentActive", agentId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsAgentActive is a free data retrieval call binding the contract method 0x8a92792b.
//
// Solidity: function isAgentActive(bytes32 agentId) view returns(bool)
func (_SageRegistryV4 *SageRegistryV4Session) IsAgentActive(agentId [32]byte) (bool, error) {
	return _SageRegistryV4.Contract.IsAgentActive(&_SageRegistryV4.CallOpts, agentId)
}

// IsAgentActive is a free data retrieval call binding the contract method 0x8a92792b.
//
// Solidity: function isAgentActive(bytes32 agentId) view returns(bool)
func (_SageRegistryV4 *SageRegistryV4CallerSession) IsAgentActive(agentId [32]byte) (bool, error) {
	return _SageRegistryV4.Contract.IsAgentActive(&_SageRegistryV4.CallOpts, agentId)
}

// VerifyAgentOwnership is a free data retrieval call binding the contract method 0x745e8f81.
//
// Solidity: function verifyAgentOwnership(bytes32 agentId, address claimedOwner) view returns(bool)
func (_SageRegistryV4 *SageRegistryV4Caller) VerifyAgentOwnership(opts *bind.CallOpts, agentId [32]byte, claimedOwner common.Address) (bool, error) {
	var out []interface{}
	err := _SageRegistryV4.contract.Call(opts, &out, "verifyAgentOwnership", agentId, claimedOwner)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// VerifyAgentOwnership is a free data retrieval call binding the contract method 0x745e8f81.
//
// Solidity: function verifyAgentOwnership(bytes32 agentId, address claimedOwner) view returns(bool)
func (_SageRegistryV4 *SageRegistryV4Session) VerifyAgentOwnership(agentId [32]byte, claimedOwner common.Address) (bool, error) {
	return _SageRegistryV4.Contract.VerifyAgentOwnership(&_SageRegistryV4.CallOpts, agentId, claimedOwner)
}

// VerifyAgentOwnership is a free data retrieval call binding the contract method 0x745e8f81.
//
// Solidity: function verifyAgentOwnership(bytes32 agentId, address claimedOwner) view returns(bool)
func (_SageRegistryV4 *SageRegistryV4CallerSession) VerifyAgentOwnership(agentId [32]byte, claimedOwner common.Address) (bool, error) {
	return _SageRegistryV4.Contract.VerifyAgentOwnership(&_SageRegistryV4.CallOpts, agentId, claimedOwner)
}

// AddKey is a paid mutator transaction binding the contract method 0x6d45ada5.
//
// Solidity: function addKey(bytes32 agentId, uint8 keyType, bytes keyData, bytes signature) returns(bytes32)
func (_SageRegistryV4 *SageRegistryV4Transactor) AddKey(opts *bind.TransactOpts, agentId [32]byte, keyType uint8, keyData []byte, signature []byte) (*types.Transaction, error) {
	return _SageRegistryV4.contract.Transact(opts, "addKey", agentId, keyType, keyData, signature)
}

// AddKey is a paid mutator transaction binding the contract method 0x6d45ada5.
//
// Solidity: function addKey(bytes32 agentId, uint8 keyType, bytes keyData, bytes signature) returns(bytes32)
func (_SageRegistryV4 *SageRegistryV4Session) AddKey(agentId [32]byte, keyType uint8, keyData []byte, signature []byte) (*types.Transaction, error) {
	return _SageRegistryV4.Contract.AddKey(&_SageRegistryV4.TransactOpts, agentId, keyType, keyData, signature)
}

// AddKey is a paid mutator transaction binding the contract method 0x6d45ada5.
//
// Solidity: function addKey(bytes32 agentId, uint8 keyType, bytes keyData, bytes signature) returns(bytes32)
func (_SageRegistryV4 *SageRegistryV4TransactorSession) AddKey(agentId [32]byte, keyType uint8, keyData []byte, signature []byte) (*types.Transaction, error) {
	return _SageRegistryV4.Contract.AddKey(&_SageRegistryV4.TransactOpts, agentId, keyType, keyData, signature)
}

// ApproveEd25519Key is a paid mutator transaction binding the contract method 0xb08f5664.
//
// Solidity: function approveEd25519Key(bytes32 keyHash) returns()
func (_SageRegistryV4 *SageRegistryV4Transactor) ApproveEd25519Key(opts *bind.TransactOpts, keyHash [32]byte) (*types.Transaction, error) {
	return _SageRegistryV4.contract.Transact(opts, "approveEd25519Key", keyHash)
}

// ApproveEd25519Key is a paid mutator transaction binding the contract method 0xb08f5664.
//
// Solidity: function approveEd25519Key(bytes32 keyHash) returns()
func (_SageRegistryV4 *SageRegistryV4Session) ApproveEd25519Key(keyHash [32]byte) (*types.Transaction, error) {
	return _SageRegistryV4.Contract.ApproveEd25519Key(&_SageRegistryV4.TransactOpts, keyHash)
}

// ApproveEd25519Key is a paid mutator transaction binding the contract method 0xb08f5664.
//
// Solidity: function approveEd25519Key(bytes32 keyHash) returns()
func (_SageRegistryV4 *SageRegistryV4TransactorSession) ApproveEd25519Key(keyHash [32]byte) (*types.Transaction, error) {
	return _SageRegistryV4.Contract.ApproveEd25519Key(&_SageRegistryV4.TransactOpts, keyHash)
}

// DeactivateAgent is a paid mutator transaction binding the contract method 0x59b5acf3.
//
// Solidity: function deactivateAgent(bytes32 agentId) returns()
func (_SageRegistryV4 *SageRegistryV4Transactor) DeactivateAgent(opts *bind.TransactOpts, agentId [32]byte) (*types.Transaction, error) {
	return _SageRegistryV4.contract.Transact(opts, "deactivateAgent", agentId)
}

// DeactivateAgent is a paid mutator transaction binding the contract method 0x59b5acf3.
//
// Solidity: function deactivateAgent(bytes32 agentId) returns()
func (_SageRegistryV4 *SageRegistryV4Session) DeactivateAgent(agentId [32]byte) (*types.Transaction, error) {
	return _SageRegistryV4.Contract.DeactivateAgent(&_SageRegistryV4.TransactOpts, agentId)
}

// DeactivateAgent is a paid mutator transaction binding the contract method 0x59b5acf3.
//
// Solidity: function deactivateAgent(bytes32 agentId) returns()
func (_SageRegistryV4 *SageRegistryV4TransactorSession) DeactivateAgent(agentId [32]byte) (*types.Transaction, error) {
	return _SageRegistryV4.Contract.DeactivateAgent(&_SageRegistryV4.TransactOpts, agentId)
}

// RegisterAgent is a paid mutator transaction binding the contract method 0x3a17bf0f.
//
// Solidity: function registerAgent((string,string,string,string,uint8[],bytes[],bytes[],string) params) returns(bytes32)
func (_SageRegistryV4 *SageRegistryV4Transactor) RegisterAgent(opts *bind.TransactOpts, params ISageRegistryV4RegistrationParams) (*types.Transaction, error) {
	return _SageRegistryV4.contract.Transact(opts, "registerAgent", params)
}

// RegisterAgent is a paid mutator transaction binding the contract method 0x3a17bf0f.
//
// Solidity: function registerAgent((string,string,string,string,uint8[],bytes[],bytes[],string) params) returns(bytes32)
func (_SageRegistryV4 *SageRegistryV4Session) RegisterAgent(params ISageRegistryV4RegistrationParams) (*types.Transaction, error) {
	return _SageRegistryV4.Contract.RegisterAgent(&_SageRegistryV4.TransactOpts, params)
}

// RegisterAgent is a paid mutator transaction binding the contract method 0x3a17bf0f.
//
// Solidity: function registerAgent((string,string,string,string,uint8[],bytes[],bytes[],string) params) returns(bytes32)
func (_SageRegistryV4 *SageRegistryV4TransactorSession) RegisterAgent(params ISageRegistryV4RegistrationParams) (*types.Transaction, error) {
	return _SageRegistryV4.Contract.RegisterAgent(&_SageRegistryV4.TransactOpts, params)
}

// RevokeKey is a paid mutator transaction binding the contract method 0x1a9cb151.
//
// Solidity: function revokeKey(bytes32 agentId, bytes32 keyHash) returns()
func (_SageRegistryV4 *SageRegistryV4Transactor) RevokeKey(opts *bind.TransactOpts, agentId [32]byte, keyHash [32]byte) (*types.Transaction, error) {
	return _SageRegistryV4.contract.Transact(opts, "revokeKey", agentId, keyHash)
}

// RevokeKey is a paid mutator transaction binding the contract method 0x1a9cb151.
//
// Solidity: function revokeKey(bytes32 agentId, bytes32 keyHash) returns()
func (_SageRegistryV4 *SageRegistryV4Session) RevokeKey(agentId [32]byte, keyHash [32]byte) (*types.Transaction, error) {
	return _SageRegistryV4.Contract.RevokeKey(&_SageRegistryV4.TransactOpts, agentId, keyHash)
}

// RevokeKey is a paid mutator transaction binding the contract method 0x1a9cb151.
//
// Solidity: function revokeKey(bytes32 agentId, bytes32 keyHash) returns()
func (_SageRegistryV4 *SageRegistryV4TransactorSession) RevokeKey(agentId [32]byte, keyHash [32]byte) (*types.Transaction, error) {
	return _SageRegistryV4.Contract.RevokeKey(&_SageRegistryV4.TransactOpts, agentId, keyHash)
}

// SetAfterRegisterHook is a paid mutator transaction binding the contract method 0xda7d9d8f.
//
// Solidity: function setAfterRegisterHook(address hook) returns()
func (_SageRegistryV4 *SageRegistryV4Transactor) SetAfterRegisterHook(opts *bind.TransactOpts, hook common.Address) (*types.Transaction, error) {
	return _SageRegistryV4.contract.Transact(opts, "setAfterRegisterHook", hook)
}

// SetAfterRegisterHook is a paid mutator transaction binding the contract method 0xda7d9d8f.
//
// Solidity: function setAfterRegisterHook(address hook) returns()
func (_SageRegistryV4 *SageRegistryV4Session) SetAfterRegisterHook(hook common.Address) (*types.Transaction, error) {
	return _SageRegistryV4.Contract.SetAfterRegisterHook(&_SageRegistryV4.TransactOpts, hook)
}

// SetAfterRegisterHook is a paid mutator transaction binding the contract method 0xda7d9d8f.
//
// Solidity: function setAfterRegisterHook(address hook) returns()
func (_SageRegistryV4 *SageRegistryV4TransactorSession) SetAfterRegisterHook(hook common.Address) (*types.Transaction, error) {
	return _SageRegistryV4.Contract.SetAfterRegisterHook(&_SageRegistryV4.TransactOpts, hook)
}

// SetBeforeRegisterHook is a paid mutator transaction binding the contract method 0x783f054c.
//
// Solidity: function setBeforeRegisterHook(address hook) returns()
func (_SageRegistryV4 *SageRegistryV4Transactor) SetBeforeRegisterHook(opts *bind.TransactOpts, hook common.Address) (*types.Transaction, error) {
	return _SageRegistryV4.contract.Transact(opts, "setBeforeRegisterHook", hook)
}

// SetBeforeRegisterHook is a paid mutator transaction binding the contract method 0x783f054c.
//
// Solidity: function setBeforeRegisterHook(address hook) returns()
func (_SageRegistryV4 *SageRegistryV4Session) SetBeforeRegisterHook(hook common.Address) (*types.Transaction, error) {
	return _SageRegistryV4.Contract.SetBeforeRegisterHook(&_SageRegistryV4.TransactOpts, hook)
}

// SetBeforeRegisterHook is a paid mutator transaction binding the contract method 0x783f054c.
//
// Solidity: function setBeforeRegisterHook(address hook) returns()
func (_SageRegistryV4 *SageRegistryV4TransactorSession) SetBeforeRegisterHook(hook common.Address) (*types.Transaction, error) {
	return _SageRegistryV4.Contract.SetBeforeRegisterHook(&_SageRegistryV4.TransactOpts, hook)
}

// UpdateAgent is a paid mutator transaction binding the contract method 0x3ae37799.
//
// Solidity: function updateAgent(bytes32 agentId, string name, string description, string endpoint, string capabilities, bytes signature) returns()
func (_SageRegistryV4 *SageRegistryV4Transactor) UpdateAgent(opts *bind.TransactOpts, agentId [32]byte, name string, description string, endpoint string, capabilities string, signature []byte) (*types.Transaction, error) {
	return _SageRegistryV4.contract.Transact(opts, "updateAgent", agentId, name, description, endpoint, capabilities, signature)
}

// UpdateAgent is a paid mutator transaction binding the contract method 0x3ae37799.
//
// Solidity: function updateAgent(bytes32 agentId, string name, string description, string endpoint, string capabilities, bytes signature) returns()
func (_SageRegistryV4 *SageRegistryV4Session) UpdateAgent(agentId [32]byte, name string, description string, endpoint string, capabilities string, signature []byte) (*types.Transaction, error) {
	return _SageRegistryV4.Contract.UpdateAgent(&_SageRegistryV4.TransactOpts, agentId, name, description, endpoint, capabilities, signature)
}

// UpdateAgent is a paid mutator transaction binding the contract method 0x3ae37799.
//
// Solidity: function updateAgent(bytes32 agentId, string name, string description, string endpoint, string capabilities, bytes signature) returns()
func (_SageRegistryV4 *SageRegistryV4TransactorSession) UpdateAgent(agentId [32]byte, name string, description string, endpoint string, capabilities string, signature []byte) (*types.Transaction, error) {
	return _SageRegistryV4.Contract.UpdateAgent(&_SageRegistryV4.TransactOpts, agentId, name, description, endpoint, capabilities, signature)
}

// SageRegistryV4AfterRegisterHookIterator is returned from FilterAfterRegisterHook and is used to iterate over the raw logs and unpacked data for AfterRegisterHook events raised by the SageRegistryV4 contract.
type SageRegistryV4AfterRegisterHookIterator struct {
	Event *SageRegistryV4AfterRegisterHook // Event containing the contract specifics and raw log

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
func (it *SageRegistryV4AfterRegisterHookIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SageRegistryV4AfterRegisterHook)
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
		it.Event = new(SageRegistryV4AfterRegisterHook)
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
func (it *SageRegistryV4AfterRegisterHookIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SageRegistryV4AfterRegisterHookIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SageRegistryV4AfterRegisterHook represents a AfterRegisterHook event raised by the SageRegistryV4 contract.
type SageRegistryV4AfterRegisterHook struct {
	AgentId  [32]byte
	Caller   common.Address
	HookData []byte
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterAfterRegisterHook is a free log retrieval operation binding the contract event 0x3cb7ebe7deaec2743a657dc0d45c0ae4aaae6befdb78e20ca9f93d6d0023893c.
//
// Solidity: event AfterRegisterHook(bytes32 indexed agentId, address indexed caller, bytes hookData)
func (_SageRegistryV4 *SageRegistryV4Filterer) FilterAfterRegisterHook(opts *bind.FilterOpts, agentId [][32]byte, caller []common.Address) (*SageRegistryV4AfterRegisterHookIterator, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}
	var callerRule []interface{}
	for _, callerItem := range caller {
		callerRule = append(callerRule, callerItem)
	}

	logs, sub, err := _SageRegistryV4.contract.FilterLogs(opts, "AfterRegisterHook", agentIdRule, callerRule)
	if err != nil {
		return nil, err
	}
	return &SageRegistryV4AfterRegisterHookIterator{contract: _SageRegistryV4.contract, event: "AfterRegisterHook", logs: logs, sub: sub}, nil
}

// WatchAfterRegisterHook is a free log subscription operation binding the contract event 0x3cb7ebe7deaec2743a657dc0d45c0ae4aaae6befdb78e20ca9f93d6d0023893c.
//
// Solidity: event AfterRegisterHook(bytes32 indexed agentId, address indexed caller, bytes hookData)
func (_SageRegistryV4 *SageRegistryV4Filterer) WatchAfterRegisterHook(opts *bind.WatchOpts, sink chan<- *SageRegistryV4AfterRegisterHook, agentId [][32]byte, caller []common.Address) (event.Subscription, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}
	var callerRule []interface{}
	for _, callerItem := range caller {
		callerRule = append(callerRule, callerItem)
	}

	logs, sub, err := _SageRegistryV4.contract.WatchLogs(opts, "AfterRegisterHook", agentIdRule, callerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SageRegistryV4AfterRegisterHook)
				if err := _SageRegistryV4.contract.UnpackLog(event, "AfterRegisterHook", log); err != nil {
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
// Solidity: event AfterRegisterHook(bytes32 indexed agentId, address indexed caller, bytes hookData)
func (_SageRegistryV4 *SageRegistryV4Filterer) ParseAfterRegisterHook(log types.Log) (*SageRegistryV4AfterRegisterHook, error) {
	event := new(SageRegistryV4AfterRegisterHook)
	if err := _SageRegistryV4.contract.UnpackLog(event, "AfterRegisterHook", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SageRegistryV4AgentDeactivatedIterator is returned from FilterAgentDeactivated and is used to iterate over the raw logs and unpacked data for AgentDeactivated events raised by the SageRegistryV4 contract.
type SageRegistryV4AgentDeactivatedIterator struct {
	Event *SageRegistryV4AgentDeactivated // Event containing the contract specifics and raw log

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
func (it *SageRegistryV4AgentDeactivatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SageRegistryV4AgentDeactivated)
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
		it.Event = new(SageRegistryV4AgentDeactivated)
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
func (it *SageRegistryV4AgentDeactivatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SageRegistryV4AgentDeactivatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SageRegistryV4AgentDeactivated represents a AgentDeactivated event raised by the SageRegistryV4 contract.
type SageRegistryV4AgentDeactivated struct {
	AgentId   [32]byte
	Owner     common.Address
	Timestamp *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterAgentDeactivated is a free log retrieval operation binding the contract event 0x529469922704beaa9a686518bed28d19385256536629b42365b7f4d9caca13f1.
//
// Solidity: event AgentDeactivated(bytes32 indexed agentId, address indexed owner, uint256 timestamp)
func (_SageRegistryV4 *SageRegistryV4Filterer) FilterAgentDeactivated(opts *bind.FilterOpts, agentId [][32]byte, owner []common.Address) (*SageRegistryV4AgentDeactivatedIterator, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}
	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _SageRegistryV4.contract.FilterLogs(opts, "AgentDeactivated", agentIdRule, ownerRule)
	if err != nil {
		return nil, err
	}
	return &SageRegistryV4AgentDeactivatedIterator{contract: _SageRegistryV4.contract, event: "AgentDeactivated", logs: logs, sub: sub}, nil
}

// WatchAgentDeactivated is a free log subscription operation binding the contract event 0x529469922704beaa9a686518bed28d19385256536629b42365b7f4d9caca13f1.
//
// Solidity: event AgentDeactivated(bytes32 indexed agentId, address indexed owner, uint256 timestamp)
func (_SageRegistryV4 *SageRegistryV4Filterer) WatchAgentDeactivated(opts *bind.WatchOpts, sink chan<- *SageRegistryV4AgentDeactivated, agentId [][32]byte, owner []common.Address) (event.Subscription, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}
	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _SageRegistryV4.contract.WatchLogs(opts, "AgentDeactivated", agentIdRule, ownerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SageRegistryV4AgentDeactivated)
				if err := _SageRegistryV4.contract.UnpackLog(event, "AgentDeactivated", log); err != nil {
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
func (_SageRegistryV4 *SageRegistryV4Filterer) ParseAgentDeactivated(log types.Log) (*SageRegistryV4AgentDeactivated, error) {
	event := new(SageRegistryV4AgentDeactivated)
	if err := _SageRegistryV4.contract.UnpackLog(event, "AgentDeactivated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SageRegistryV4AgentRegisteredIterator is returned from FilterAgentRegistered and is used to iterate over the raw logs and unpacked data for AgentRegistered events raised by the SageRegistryV4 contract.
type SageRegistryV4AgentRegisteredIterator struct {
	Event *SageRegistryV4AgentRegistered // Event containing the contract specifics and raw log

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
func (it *SageRegistryV4AgentRegisteredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SageRegistryV4AgentRegistered)
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
		it.Event = new(SageRegistryV4AgentRegistered)
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
func (it *SageRegistryV4AgentRegisteredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SageRegistryV4AgentRegisteredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SageRegistryV4AgentRegistered represents a AgentRegistered event raised by the SageRegistryV4 contract.
type SageRegistryV4AgentRegistered struct {
	AgentId   [32]byte
	Owner     common.Address
	Did       string
	Timestamp *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterAgentRegistered is a free log retrieval operation binding the contract event 0x848b086b4ab56ffb70fbcbb34fd5e8f35d1dd5347ee5344efbe6c0f5b97c70f4.
//
// Solidity: event AgentRegistered(bytes32 indexed agentId, address indexed owner, string did, uint256 timestamp)
func (_SageRegistryV4 *SageRegistryV4Filterer) FilterAgentRegistered(opts *bind.FilterOpts, agentId [][32]byte, owner []common.Address) (*SageRegistryV4AgentRegisteredIterator, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}
	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _SageRegistryV4.contract.FilterLogs(opts, "AgentRegistered", agentIdRule, ownerRule)
	if err != nil {
		return nil, err
	}
	return &SageRegistryV4AgentRegisteredIterator{contract: _SageRegistryV4.contract, event: "AgentRegistered", logs: logs, sub: sub}, nil
}

// WatchAgentRegistered is a free log subscription operation binding the contract event 0x848b086b4ab56ffb70fbcbb34fd5e8f35d1dd5347ee5344efbe6c0f5b97c70f4.
//
// Solidity: event AgentRegistered(bytes32 indexed agentId, address indexed owner, string did, uint256 timestamp)
func (_SageRegistryV4 *SageRegistryV4Filterer) WatchAgentRegistered(opts *bind.WatchOpts, sink chan<- *SageRegistryV4AgentRegistered, agentId [][32]byte, owner []common.Address) (event.Subscription, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}
	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _SageRegistryV4.contract.WatchLogs(opts, "AgentRegistered", agentIdRule, ownerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SageRegistryV4AgentRegistered)
				if err := _SageRegistryV4.contract.UnpackLog(event, "AgentRegistered", log); err != nil {
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
func (_SageRegistryV4 *SageRegistryV4Filterer) ParseAgentRegistered(log types.Log) (*SageRegistryV4AgentRegistered, error) {
	event := new(SageRegistryV4AgentRegistered)
	if err := _SageRegistryV4.contract.UnpackLog(event, "AgentRegistered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SageRegistryV4AgentUpdatedIterator is returned from FilterAgentUpdated and is used to iterate over the raw logs and unpacked data for AgentUpdated events raised by the SageRegistryV4 contract.
type SageRegistryV4AgentUpdatedIterator struct {
	Event *SageRegistryV4AgentUpdated // Event containing the contract specifics and raw log

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
func (it *SageRegistryV4AgentUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SageRegistryV4AgentUpdated)
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
		it.Event = new(SageRegistryV4AgentUpdated)
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
func (it *SageRegistryV4AgentUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SageRegistryV4AgentUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SageRegistryV4AgentUpdated represents a AgentUpdated event raised by the SageRegistryV4 contract.
type SageRegistryV4AgentUpdated struct {
	AgentId   [32]byte
	Owner     common.Address
	Timestamp *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterAgentUpdated is a free log retrieval operation binding the contract event 0xb28fb12b8366d2fb9a1adf15f6b59fcccc9e3b377eb5db8dcdc758c055dde5e5.
//
// Solidity: event AgentUpdated(bytes32 indexed agentId, address indexed owner, uint256 timestamp)
func (_SageRegistryV4 *SageRegistryV4Filterer) FilterAgentUpdated(opts *bind.FilterOpts, agentId [][32]byte, owner []common.Address) (*SageRegistryV4AgentUpdatedIterator, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}
	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _SageRegistryV4.contract.FilterLogs(opts, "AgentUpdated", agentIdRule, ownerRule)
	if err != nil {
		return nil, err
	}
	return &SageRegistryV4AgentUpdatedIterator{contract: _SageRegistryV4.contract, event: "AgentUpdated", logs: logs, sub: sub}, nil
}

// WatchAgentUpdated is a free log subscription operation binding the contract event 0xb28fb12b8366d2fb9a1adf15f6b59fcccc9e3b377eb5db8dcdc758c055dde5e5.
//
// Solidity: event AgentUpdated(bytes32 indexed agentId, address indexed owner, uint256 timestamp)
func (_SageRegistryV4 *SageRegistryV4Filterer) WatchAgentUpdated(opts *bind.WatchOpts, sink chan<- *SageRegistryV4AgentUpdated, agentId [][32]byte, owner []common.Address) (event.Subscription, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}
	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _SageRegistryV4.contract.WatchLogs(opts, "AgentUpdated", agentIdRule, ownerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SageRegistryV4AgentUpdated)
				if err := _SageRegistryV4.contract.UnpackLog(event, "AgentUpdated", log); err != nil {
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
func (_SageRegistryV4 *SageRegistryV4Filterer) ParseAgentUpdated(log types.Log) (*SageRegistryV4AgentUpdated, error) {
	event := new(SageRegistryV4AgentUpdated)
	if err := _SageRegistryV4.contract.UnpackLog(event, "AgentUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SageRegistryV4BeforeRegisterHookIterator is returned from FilterBeforeRegisterHook and is used to iterate over the raw logs and unpacked data for BeforeRegisterHook events raised by the SageRegistryV4 contract.
type SageRegistryV4BeforeRegisterHookIterator struct {
	Event *SageRegistryV4BeforeRegisterHook // Event containing the contract specifics and raw log

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
func (it *SageRegistryV4BeforeRegisterHookIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SageRegistryV4BeforeRegisterHook)
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
		it.Event = new(SageRegistryV4BeforeRegisterHook)
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
func (it *SageRegistryV4BeforeRegisterHookIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SageRegistryV4BeforeRegisterHookIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SageRegistryV4BeforeRegisterHook represents a BeforeRegisterHook event raised by the SageRegistryV4 contract.
type SageRegistryV4BeforeRegisterHook struct {
	AgentId  [32]byte
	Caller   common.Address
	HookData []byte
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterBeforeRegisterHook is a free log retrieval operation binding the contract event 0xe9e7066ed0bb4551380e108afced4a59ed1503dccf6c69f572e8f0b2686b7e6d.
//
// Solidity: event BeforeRegisterHook(bytes32 indexed agentId, address indexed caller, bytes hookData)
func (_SageRegistryV4 *SageRegistryV4Filterer) FilterBeforeRegisterHook(opts *bind.FilterOpts, agentId [][32]byte, caller []common.Address) (*SageRegistryV4BeforeRegisterHookIterator, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}
	var callerRule []interface{}
	for _, callerItem := range caller {
		callerRule = append(callerRule, callerItem)
	}

	logs, sub, err := _SageRegistryV4.contract.FilterLogs(opts, "BeforeRegisterHook", agentIdRule, callerRule)
	if err != nil {
		return nil, err
	}
	return &SageRegistryV4BeforeRegisterHookIterator{contract: _SageRegistryV4.contract, event: "BeforeRegisterHook", logs: logs, sub: sub}, nil
}

// WatchBeforeRegisterHook is a free log subscription operation binding the contract event 0xe9e7066ed0bb4551380e108afced4a59ed1503dccf6c69f572e8f0b2686b7e6d.
//
// Solidity: event BeforeRegisterHook(bytes32 indexed agentId, address indexed caller, bytes hookData)
func (_SageRegistryV4 *SageRegistryV4Filterer) WatchBeforeRegisterHook(opts *bind.WatchOpts, sink chan<- *SageRegistryV4BeforeRegisterHook, agentId [][32]byte, caller []common.Address) (event.Subscription, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}
	var callerRule []interface{}
	for _, callerItem := range caller {
		callerRule = append(callerRule, callerItem)
	}

	logs, sub, err := _SageRegistryV4.contract.WatchLogs(opts, "BeforeRegisterHook", agentIdRule, callerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SageRegistryV4BeforeRegisterHook)
				if err := _SageRegistryV4.contract.UnpackLog(event, "BeforeRegisterHook", log); err != nil {
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
// Solidity: event BeforeRegisterHook(bytes32 indexed agentId, address indexed caller, bytes hookData)
func (_SageRegistryV4 *SageRegistryV4Filterer) ParseBeforeRegisterHook(log types.Log) (*SageRegistryV4BeforeRegisterHook, error) {
	event := new(SageRegistryV4BeforeRegisterHook)
	if err := _SageRegistryV4.contract.UnpackLog(event, "BeforeRegisterHook", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SageRegistryV4Ed25519KeyApprovedIterator is returned from FilterEd25519KeyApproved and is used to iterate over the raw logs and unpacked data for Ed25519KeyApproved events raised by the SageRegistryV4 contract.
type SageRegistryV4Ed25519KeyApprovedIterator struct {
	Event *SageRegistryV4Ed25519KeyApproved // Event containing the contract specifics and raw log

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
func (it *SageRegistryV4Ed25519KeyApprovedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SageRegistryV4Ed25519KeyApproved)
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
		it.Event = new(SageRegistryV4Ed25519KeyApproved)
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
func (it *SageRegistryV4Ed25519KeyApprovedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SageRegistryV4Ed25519KeyApprovedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SageRegistryV4Ed25519KeyApproved represents a Ed25519KeyApproved event raised by the SageRegistryV4 contract.
type SageRegistryV4Ed25519KeyApproved struct {
	KeyHash   [32]byte
	Timestamp *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterEd25519KeyApproved is a free log retrieval operation binding the contract event 0xd21ccfb64f959401f8286dc090d479c2014eb3ac0bd4b8d7bcecfd16e24bcdad.
//
// Solidity: event Ed25519KeyApproved(bytes32 indexed keyHash, uint256 timestamp)
func (_SageRegistryV4 *SageRegistryV4Filterer) FilterEd25519KeyApproved(opts *bind.FilterOpts, keyHash [][32]byte) (*SageRegistryV4Ed25519KeyApprovedIterator, error) {

	var keyHashRule []interface{}
	for _, keyHashItem := range keyHash {
		keyHashRule = append(keyHashRule, keyHashItem)
	}

	logs, sub, err := _SageRegistryV4.contract.FilterLogs(opts, "Ed25519KeyApproved", keyHashRule)
	if err != nil {
		return nil, err
	}
	return &SageRegistryV4Ed25519KeyApprovedIterator{contract: _SageRegistryV4.contract, event: "Ed25519KeyApproved", logs: logs, sub: sub}, nil
}

// WatchEd25519KeyApproved is a free log subscription operation binding the contract event 0xd21ccfb64f959401f8286dc090d479c2014eb3ac0bd4b8d7bcecfd16e24bcdad.
//
// Solidity: event Ed25519KeyApproved(bytes32 indexed keyHash, uint256 timestamp)
func (_SageRegistryV4 *SageRegistryV4Filterer) WatchEd25519KeyApproved(opts *bind.WatchOpts, sink chan<- *SageRegistryV4Ed25519KeyApproved, keyHash [][32]byte) (event.Subscription, error) {

	var keyHashRule []interface{}
	for _, keyHashItem := range keyHash {
		keyHashRule = append(keyHashRule, keyHashItem)
	}

	logs, sub, err := _SageRegistryV4.contract.WatchLogs(opts, "Ed25519KeyApproved", keyHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SageRegistryV4Ed25519KeyApproved)
				if err := _SageRegistryV4.contract.UnpackLog(event, "Ed25519KeyApproved", log); err != nil {
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

// ParseEd25519KeyApproved is a log parse operation binding the contract event 0xd21ccfb64f959401f8286dc090d479c2014eb3ac0bd4b8d7bcecfd16e24bcdad.
//
// Solidity: event Ed25519KeyApproved(bytes32 indexed keyHash, uint256 timestamp)
func (_SageRegistryV4 *SageRegistryV4Filterer) ParseEd25519KeyApproved(log types.Log) (*SageRegistryV4Ed25519KeyApproved, error) {
	event := new(SageRegistryV4Ed25519KeyApproved)
	if err := _SageRegistryV4.contract.UnpackLog(event, "Ed25519KeyApproved", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SageRegistryV4KeyAddedIterator is returned from FilterKeyAdded and is used to iterate over the raw logs and unpacked data for KeyAdded events raised by the SageRegistryV4 contract.
type SageRegistryV4KeyAddedIterator struct {
	Event *SageRegistryV4KeyAdded // Event containing the contract specifics and raw log

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
func (it *SageRegistryV4KeyAddedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SageRegistryV4KeyAdded)
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
		it.Event = new(SageRegistryV4KeyAdded)
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
func (it *SageRegistryV4KeyAddedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SageRegistryV4KeyAddedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SageRegistryV4KeyAdded represents a KeyAdded event raised by the SageRegistryV4 contract.
type SageRegistryV4KeyAdded struct {
	AgentId   [32]byte
	KeyHash   [32]byte
	KeyType   uint8
	Timestamp *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterKeyAdded is a free log retrieval operation binding the contract event 0x11f138c8931fc92ab4fbeb5dd32df17d56c9411a543739c3526ed0265d8fad13.
//
// Solidity: event KeyAdded(bytes32 indexed agentId, bytes32 indexed keyHash, uint8 keyType, uint256 timestamp)
func (_SageRegistryV4 *SageRegistryV4Filterer) FilterKeyAdded(opts *bind.FilterOpts, agentId [][32]byte, keyHash [][32]byte) (*SageRegistryV4KeyAddedIterator, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}
	var keyHashRule []interface{}
	for _, keyHashItem := range keyHash {
		keyHashRule = append(keyHashRule, keyHashItem)
	}

	logs, sub, err := _SageRegistryV4.contract.FilterLogs(opts, "KeyAdded", agentIdRule, keyHashRule)
	if err != nil {
		return nil, err
	}
	return &SageRegistryV4KeyAddedIterator{contract: _SageRegistryV4.contract, event: "KeyAdded", logs: logs, sub: sub}, nil
}

// WatchKeyAdded is a free log subscription operation binding the contract event 0x11f138c8931fc92ab4fbeb5dd32df17d56c9411a543739c3526ed0265d8fad13.
//
// Solidity: event KeyAdded(bytes32 indexed agentId, bytes32 indexed keyHash, uint8 keyType, uint256 timestamp)
func (_SageRegistryV4 *SageRegistryV4Filterer) WatchKeyAdded(opts *bind.WatchOpts, sink chan<- *SageRegistryV4KeyAdded, agentId [][32]byte, keyHash [][32]byte) (event.Subscription, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}
	var keyHashRule []interface{}
	for _, keyHashItem := range keyHash {
		keyHashRule = append(keyHashRule, keyHashItem)
	}

	logs, sub, err := _SageRegistryV4.contract.WatchLogs(opts, "KeyAdded", agentIdRule, keyHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SageRegistryV4KeyAdded)
				if err := _SageRegistryV4.contract.UnpackLog(event, "KeyAdded", log); err != nil {
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
func (_SageRegistryV4 *SageRegistryV4Filterer) ParseKeyAdded(log types.Log) (*SageRegistryV4KeyAdded, error) {
	event := new(SageRegistryV4KeyAdded)
	if err := _SageRegistryV4.contract.UnpackLog(event, "KeyAdded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SageRegistryV4KeyRevokedIterator is returned from FilterKeyRevoked and is used to iterate over the raw logs and unpacked data for KeyRevoked events raised by the SageRegistryV4 contract.
type SageRegistryV4KeyRevokedIterator struct {
	Event *SageRegistryV4KeyRevoked // Event containing the contract specifics and raw log

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
func (it *SageRegistryV4KeyRevokedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SageRegistryV4KeyRevoked)
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
		it.Event = new(SageRegistryV4KeyRevoked)
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
func (it *SageRegistryV4KeyRevokedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SageRegistryV4KeyRevokedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SageRegistryV4KeyRevoked represents a KeyRevoked event raised by the SageRegistryV4 contract.
type SageRegistryV4KeyRevoked struct {
	AgentId   [32]byte
	KeyHash   [32]byte
	Timestamp *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterKeyRevoked is a free log retrieval operation binding the contract event 0x209fb85e2522622566ffdf13e48258218f4c155aefc75703539e1a971380cd3f.
//
// Solidity: event KeyRevoked(bytes32 indexed agentId, bytes32 indexed keyHash, uint256 timestamp)
func (_SageRegistryV4 *SageRegistryV4Filterer) FilterKeyRevoked(opts *bind.FilterOpts, agentId [][32]byte, keyHash [][32]byte) (*SageRegistryV4KeyRevokedIterator, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}
	var keyHashRule []interface{}
	for _, keyHashItem := range keyHash {
		keyHashRule = append(keyHashRule, keyHashItem)
	}

	logs, sub, err := _SageRegistryV4.contract.FilterLogs(opts, "KeyRevoked", agentIdRule, keyHashRule)
	if err != nil {
		return nil, err
	}
	return &SageRegistryV4KeyRevokedIterator{contract: _SageRegistryV4.contract, event: "KeyRevoked", logs: logs, sub: sub}, nil
}

// WatchKeyRevoked is a free log subscription operation binding the contract event 0x209fb85e2522622566ffdf13e48258218f4c155aefc75703539e1a971380cd3f.
//
// Solidity: event KeyRevoked(bytes32 indexed agentId, bytes32 indexed keyHash, uint256 timestamp)
func (_SageRegistryV4 *SageRegistryV4Filterer) WatchKeyRevoked(opts *bind.WatchOpts, sink chan<- *SageRegistryV4KeyRevoked, agentId [][32]byte, keyHash [][32]byte) (event.Subscription, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}
	var keyHashRule []interface{}
	for _, keyHashItem := range keyHash {
		keyHashRule = append(keyHashRule, keyHashItem)
	}

	logs, sub, err := _SageRegistryV4.contract.WatchLogs(opts, "KeyRevoked", agentIdRule, keyHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SageRegistryV4KeyRevoked)
				if err := _SageRegistryV4.contract.UnpackLog(event, "KeyRevoked", log); err != nil {
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
func (_SageRegistryV4 *SageRegistryV4Filterer) ParseKeyRevoked(log types.Log) (*SageRegistryV4KeyRevoked, error) {
	event := new(SageRegistryV4KeyRevoked)
	if err := _SageRegistryV4.contract.UnpackLog(event, "KeyRevoked", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

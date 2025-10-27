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

// IRegistryHookMetaData contains all meta data concerning the IRegistryHook contract.
var IRegistryHookMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"afterRegister\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"beforeRegister\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"},{\"internalType\":\"string\",\"name\":\"reason\",\"type\":\"string\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// IRegistryHookABI is the input ABI used to generate the binding from.
// Deprecated: Use IRegistryHookMetaData.ABI instead.
var IRegistryHookABI = IRegistryHookMetaData.ABI

// IRegistryHook is an auto generated Go binding around an Ethereum contract.
type IRegistryHook struct {
	IRegistryHookCaller     // Read-only binding to the contract
	IRegistryHookTransactor // Write-only binding to the contract
	IRegistryHookFilterer   // Log filterer for contract events
}

// IRegistryHookCaller is an auto generated read-only Go binding around an Ethereum contract.
type IRegistryHookCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IRegistryHookTransactor is an auto generated write-only Go binding around an Ethereum contract.
type IRegistryHookTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IRegistryHookFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type IRegistryHookFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IRegistryHookSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type IRegistryHookSession struct {
	Contract     *IRegistryHook    // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// IRegistryHookCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type IRegistryHookCallerSession struct {
	Contract *IRegistryHookCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts        // Call options to use throughout this session
}

// IRegistryHookTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type IRegistryHookTransactorSession struct {
	Contract     *IRegistryHookTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// IRegistryHookRaw is an auto generated low-level Go binding around an Ethereum contract.
type IRegistryHookRaw struct {
	Contract *IRegistryHook // Generic contract binding to access the raw methods on
}

// IRegistryHookCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type IRegistryHookCallerRaw struct {
	Contract *IRegistryHookCaller // Generic read-only contract binding to access the raw methods on
}

// IRegistryHookTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type IRegistryHookTransactorRaw struct {
	Contract *IRegistryHookTransactor // Generic write-only contract binding to access the raw methods on
}

// NewIRegistryHook creates a new instance of IRegistryHook, bound to a specific deployed contract.
func NewIRegistryHook(address common.Address, backend bind.ContractBackend) (*IRegistryHook, error) {
	contract, err := bindIRegistryHook(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &IRegistryHook{IRegistryHookCaller: IRegistryHookCaller{contract: contract}, IRegistryHookTransactor: IRegistryHookTransactor{contract: contract}, IRegistryHookFilterer: IRegistryHookFilterer{contract: contract}}, nil
}

// NewIRegistryHookCaller creates a new read-only instance of IRegistryHook, bound to a specific deployed contract.
func NewIRegistryHookCaller(address common.Address, caller bind.ContractCaller) (*IRegistryHookCaller, error) {
	contract, err := bindIRegistryHook(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &IRegistryHookCaller{contract: contract}, nil
}

// NewIRegistryHookTransactor creates a new write-only instance of IRegistryHook, bound to a specific deployed contract.
func NewIRegistryHookTransactor(address common.Address, transactor bind.ContractTransactor) (*IRegistryHookTransactor, error) {
	contract, err := bindIRegistryHook(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &IRegistryHookTransactor{contract: contract}, nil
}

// NewIRegistryHookFilterer creates a new log filterer instance of IRegistryHook, bound to a specific deployed contract.
func NewIRegistryHookFilterer(address common.Address, filterer bind.ContractFilterer) (*IRegistryHookFilterer, error) {
	contract, err := bindIRegistryHook(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &IRegistryHookFilterer{contract: contract}, nil
}

// bindIRegistryHook binds a generic wrapper to an already deployed contract.
func bindIRegistryHook(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := IRegistryHookMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IRegistryHook *IRegistryHookRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IRegistryHook.Contract.IRegistryHookCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IRegistryHook *IRegistryHookRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IRegistryHook.Contract.IRegistryHookTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IRegistryHook *IRegistryHookRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IRegistryHook.Contract.IRegistryHookTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IRegistryHook *IRegistryHookCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IRegistryHook.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IRegistryHook *IRegistryHookTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IRegistryHook.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IRegistryHook *IRegistryHookTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IRegistryHook.Contract.contract.Transact(opts, method, params...)
}

// AfterRegister is a paid mutator transaction binding the contract method 0xc847ad35.
//
// Solidity: function afterRegister(bytes32 agentId, address owner, bytes data) returns()
func (_IRegistryHook *IRegistryHookTransactor) AfterRegister(opts *bind.TransactOpts, agentId [32]byte, owner common.Address, data []byte) (*types.Transaction, error) {
	return _IRegistryHook.contract.Transact(opts, "afterRegister", agentId, owner, data)
}

// AfterRegister is a paid mutator transaction binding the contract method 0xc847ad35.
//
// Solidity: function afterRegister(bytes32 agentId, address owner, bytes data) returns()
func (_IRegistryHook *IRegistryHookSession) AfterRegister(agentId [32]byte, owner common.Address, data []byte) (*types.Transaction, error) {
	return _IRegistryHook.Contract.AfterRegister(&_IRegistryHook.TransactOpts, agentId, owner, data)
}

// AfterRegister is a paid mutator transaction binding the contract method 0xc847ad35.
//
// Solidity: function afterRegister(bytes32 agentId, address owner, bytes data) returns()
func (_IRegistryHook *IRegistryHookTransactorSession) AfterRegister(agentId [32]byte, owner common.Address, data []byte) (*types.Transaction, error) {
	return _IRegistryHook.Contract.AfterRegister(&_IRegistryHook.TransactOpts, agentId, owner, data)
}

// BeforeRegister is a paid mutator transaction binding the contract method 0x7b319ba1.
//
// Solidity: function beforeRegister(bytes32 agentId, address owner, bytes data) returns(bool success, string reason)
func (_IRegistryHook *IRegistryHookTransactor) BeforeRegister(opts *bind.TransactOpts, agentId [32]byte, owner common.Address, data []byte) (*types.Transaction, error) {
	return _IRegistryHook.contract.Transact(opts, "beforeRegister", agentId, owner, data)
}

// BeforeRegister is a paid mutator transaction binding the contract method 0x7b319ba1.
//
// Solidity: function beforeRegister(bytes32 agentId, address owner, bytes data) returns(bool success, string reason)
func (_IRegistryHook *IRegistryHookSession) BeforeRegister(agentId [32]byte, owner common.Address, data []byte) (*types.Transaction, error) {
	return _IRegistryHook.Contract.BeforeRegister(&_IRegistryHook.TransactOpts, agentId, owner, data)
}

// BeforeRegister is a paid mutator transaction binding the contract method 0x7b319ba1.
//
// Solidity: function beforeRegister(bytes32 agentId, address owner, bytes data) returns(bool success, string reason)
func (_IRegistryHook *IRegistryHookTransactorSession) BeforeRegister(agentId [32]byte, owner common.Address, data []byte) (*types.Transaction, error) {
	return _IRegistryHook.Contract.BeforeRegister(&_IRegistryHook.TransactOpts, agentId, owner, data)
}

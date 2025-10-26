// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package hook

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

// SageVerificationHookMetaData contains all meta data concerning the SageVerificationHook contract.
var SageVerificationHookMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"MAX_REGISTRATIONS_PER_DAY\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"OWNER\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"REGISTRATION_COOLDOWN\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"addToBlacklist\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"agentOwner\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"name\":\"afterRegister\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"agentOwner\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"beforeRegister\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"},{\"internalType\":\"string\",\"name\":\"reason\",\"type\":\"string\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"blacklisted\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"lastRegistrationTime\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"registrationAttempts\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"removeFromBlacklist\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x60a0604052346100395761001233608052565b604051610b2761003f823960805181818160ed0152818161087001526108fd0152610b2790f35b600080fdfe6080604052600436101561001257600080fd5b60003560e01c8063117803e3146100b25780633db059b6146100ad57806344337ea1146100a8578063537df3b6146100a35780635a2a26bd1461009e5780637b319ba1146100995780637f6e8cbf14610094578063b0ff63831461008f578063c847ad351461008a5763dbac26e9036100b75761042c565b6103df565b6103c4565b610397565b610353565b61023a565b6101ba565b61019d565b61013f565b6100d8565b600080fd5b60009103126100b757565b6001600160a01b031690565b90565b565b346100b7576100e83660046100bc565b6040517f00000000000000000000000000000000000000000000000000000000000000006001600160a01b03168152602090f35b0390f35b6100d36100d36100d39290565b6100d36005610120565b6100d361012d565b346100b75761014f3660046100bc565b61011c61015a610137565b6040519182918290815260200190565b6001600160a01b0381165b036100b757565b905035906100d68261016a565b906020828203126100b7576100d39161017c565b346100b7576101b56101b0366004610189565b6108df565b604051005b346100b7576101b56101cd366004610189565b610933565b6100d3906100c7906001600160a01b031682565b6100d3906101d2565b6100d3906101e6565b90610202906101ef565b600052602052604060002090565b6100d3916008021c81565b906100d39154610210565b60006102356100d392826101f8565b61021b565b346100b75761011c61015a610250366004610189565b610226565b80610175565b905035906100d682610255565b909182601f830112156100b75781359167ffffffffffffffff83116100b75760200192600183028401116100b757565b916060838303126100b7576102ad828461025b565b926102bb836020830161017c565b92604082013567ffffffffffffffff81116100b7576102da9201610268565b9091565b60005b8381106102f15750506000910152565b81810151838201526020016102e1565b61032261032b60209361033593610316815190565b80835293849260200190565b958691016102de565b601f01601f191690565b0190565b90151581526040602082018190526100d392910190610301565b346100b75761036f610366366004610298565b929190916106d8565b9061011c61037c60405190565b92839283610339565b6100d3906102356001916000926101f8565b346100b75761011c61015a6103ad366004610189565b610385565b6100d3603c610120565b6100d36103b2565b346100b7576103d43660046100bc565b61011c61015a6103bc565b346100b7576101b56103f2366004610298565b929190916107e9565b6100d3916008021c5b60ff1690565b906100d391546103fb565b6100d3906104276002916000926101f8565b61040a565b346100b75761011c610447610442366004610189565b610415565b60405191829182901515815260200190565b6100d390610404565b6100d39054610459565b634e487b7160e01b600052604160045260246000fd5b90601f01601f1916810190811067ffffffffffffffff8211176104a457604052565b61046c565b906100d66104b660405190565b9283610482565b67ffffffffffffffff81116104a457602090601f01601f19160190565b906104ec6104e7836104bd565b6104a9565b918252565b6104fb60136104da565b721059191c995cdcc8189b1858dadb1a5cdd1959606a1b602082015290565b6100d36104f1565b6100d39081565b6100d39054610522565b634e487b7160e01b600052601160045260246000fd5b9190820180921161055657565b610533565b610565601c6104da565b7f526567697374726174696f6e20636f6f6c646f776e2061637469766500000000602082015290565b6100d361055b565b90600019905b9181191691161790565b906105b66100d36105bd92610120565b8254610596565b9055565b6105cb60206104da565b7f4461696c7920726567697374726174696f6e206c696d69742072656163686564602082015290565b6100d36105c1565b90826000939282370152565b909291926106186104e7826104bd565b938185526020850190828401116100b7576100d6926105fc565b9080601f830112156100b7578160206100d393359101610608565b9190916040818403126100b757803567ffffffffffffffff81116100b75783610677918301610632565b92602082013567ffffffffffffffff81116100b7576100d39201610632565b6106a060126104da565b71125b9d985b1a590811125108199bdc9b585d60721b602082015290565b6100d3610696565b6100d360006104da565b6100d36106c6565b50906106ed6106e88360026101f8565b610462565b6107cc576001924261072261071e6100d361071061070b888a6101f8565b610529565b6107186103b2565b90610549565b9190565b106107bd5761073083610961565b61079b575b61074461070b600094856101f8565b61075261071e6100d361012d565b101561078e5761076b826107719261077594019061064d565b506109ec565b1590565b6107835750906100d36106d0565b9050906100d36106be565b50509050906100d36105f4565b6107b860006107b3856107ad83610120565b926101f8565b6105a6565b610735565b505050506000906100d361058e565b5050506000906100d361051a565b60001981146105565760010190565b5090506100d691506108166107ff8260006101f8565b61081061080b82610529565b6107da565b906105a6565b6107b3429160016101f8565b1561082957565b60405162461bcd60e51b815260206004820152600a60248201526927b7363c9037bbb732b960b11b6044820152606490fd5b6100d6906108a43361089e6001600160a01b037f0000000000000000000000000000000000000000000000000000000000000000165b916001600160a01b031690565b14610822565b6108c9565b9060ff9061059c565b906108c26100d36105bd92151590565b82546108a9565b6100d6906108da60019160026101f8565b6108b2565b6100d69061085b565b6100d6906109223361089e6001600160a01b037f000000000000000000000000000000000000000000000000000000000000000016610891565b6100d6906108da60009160026101f8565b6100d6906108e8565b634e487b7160e01b600052601260045260246000fd5b811561095c570490565b61093c565b61099661071e6100d36201518061098261099061070b610988428486610120565b90610952565b9760016101f8565b91610120565b1190565b634e487b7160e01b600052603260045260246000fd5b906109b9825190565b8110156109c7570160200190565b61099a565b6109df6109d96100d39290565b60f81b90565b6001600160f81b03191690565b80516109fb61071e600a610120565b10610aeb57600090610a26610a18610a1284610120565b836109b0565b516001600160f81b03191690565b90606491610a44610a36846109cc565b916001600160f81b03191690565b1415918215610ac4575b8215610a96575b508115610a68575b506100d35750600190565b610a819150610a1890610a7b6003610120565b906109b0565b610a8e610a36603a6109cc565b141538610a5d565b909150610abb610a36610ab5610a18610aaf6002610120565b866109b0565b926109cc565b14159038610a55565b9150610ad6610a18610a126001610120565b610ae3610a3660696109cc565b141591610a4e565b5060009056fea264697066735822122087bb5e1b9c901338d4e26eb82c950cbe0bfd618c72d5882113c40ea1b047e45664736f6c63430008130033",
}

// SageVerificationHookABI is the input ABI used to generate the binding from.
// Deprecated: Use SageVerificationHookMetaData.ABI instead.
var SageVerificationHookABI = SageVerificationHookMetaData.ABI

// SageVerificationHookBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use SageVerificationHookMetaData.Bin instead.
var SageVerificationHookBin = SageVerificationHookMetaData.Bin

// DeploySageVerificationHook deploys a new Ethereum contract, binding an instance of SageVerificationHook to it.
func DeploySageVerificationHook(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *SageVerificationHook, error) {
	parsed, err := SageVerificationHookMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(SageVerificationHookBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &SageVerificationHook{SageVerificationHookCaller: SageVerificationHookCaller{contract: contract}, SageVerificationHookTransactor: SageVerificationHookTransactor{contract: contract}, SageVerificationHookFilterer: SageVerificationHookFilterer{contract: contract}}, nil
}

// SageVerificationHook is an auto generated Go binding around an Ethereum contract.
type SageVerificationHook struct {
	SageVerificationHookCaller     // Read-only binding to the contract
	SageVerificationHookTransactor // Write-only binding to the contract
	SageVerificationHookFilterer   // Log filterer for contract events
}

// SageVerificationHookCaller is an auto generated read-only Go binding around an Ethereum contract.
type SageVerificationHookCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SageVerificationHookTransactor is an auto generated write-only Go binding around an Ethereum contract.
type SageVerificationHookTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SageVerificationHookFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type SageVerificationHookFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SageVerificationHookSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type SageVerificationHookSession struct {
	Contract     *SageVerificationHook // Generic contract binding to set the session for
	CallOpts     bind.CallOpts         // Call options to use throughout this session
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// SageVerificationHookCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type SageVerificationHookCallerSession struct {
	Contract *SageVerificationHookCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts               // Call options to use throughout this session
}

// SageVerificationHookTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type SageVerificationHookTransactorSession struct {
	Contract     *SageVerificationHookTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts               // Transaction auth options to use throughout this session
}

// SageVerificationHookRaw is an auto generated low-level Go binding around an Ethereum contract.
type SageVerificationHookRaw struct {
	Contract *SageVerificationHook // Generic contract binding to access the raw methods on
}

// SageVerificationHookCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type SageVerificationHookCallerRaw struct {
	Contract *SageVerificationHookCaller // Generic read-only contract binding to access the raw methods on
}

// SageVerificationHookTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type SageVerificationHookTransactorRaw struct {
	Contract *SageVerificationHookTransactor // Generic write-only contract binding to access the raw methods on
}

// NewSageVerificationHook creates a new instance of SageVerificationHook, bound to a specific deployed contract.
func NewSageVerificationHook(address common.Address, backend bind.ContractBackend) (*SageVerificationHook, error) {
	contract, err := bindSageVerificationHook(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &SageVerificationHook{SageVerificationHookCaller: SageVerificationHookCaller{contract: contract}, SageVerificationHookTransactor: SageVerificationHookTransactor{contract: contract}, SageVerificationHookFilterer: SageVerificationHookFilterer{contract: contract}}, nil
}

// NewSageVerificationHookCaller creates a new read-only instance of SageVerificationHook, bound to a specific deployed contract.
func NewSageVerificationHookCaller(address common.Address, caller bind.ContractCaller) (*SageVerificationHookCaller, error) {
	contract, err := bindSageVerificationHook(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &SageVerificationHookCaller{contract: contract}, nil
}

// NewSageVerificationHookTransactor creates a new write-only instance of SageVerificationHook, bound to a specific deployed contract.
func NewSageVerificationHookTransactor(address common.Address, transactor bind.ContractTransactor) (*SageVerificationHookTransactor, error) {
	contract, err := bindSageVerificationHook(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &SageVerificationHookTransactor{contract: contract}, nil
}

// NewSageVerificationHookFilterer creates a new log filterer instance of SageVerificationHook, bound to a specific deployed contract.
func NewSageVerificationHookFilterer(address common.Address, filterer bind.ContractFilterer) (*SageVerificationHookFilterer, error) {
	contract, err := bindSageVerificationHook(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &SageVerificationHookFilterer{contract: contract}, nil
}

// bindSageVerificationHook binds a generic wrapper to an already deployed contract.
func bindSageVerificationHook(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := SageVerificationHookMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SageVerificationHook *SageVerificationHookRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SageVerificationHook.Contract.SageVerificationHookCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SageVerificationHook *SageVerificationHookRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SageVerificationHook.Contract.SageVerificationHookTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SageVerificationHook *SageVerificationHookRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SageVerificationHook.Contract.SageVerificationHookTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SageVerificationHook *SageVerificationHookCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SageVerificationHook.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SageVerificationHook *SageVerificationHookTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SageVerificationHook.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SageVerificationHook *SageVerificationHookTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SageVerificationHook.Contract.contract.Transact(opts, method, params...)
}

// MAXREGISTRATIONSPERDAY is a free data retrieval call binding the contract method 0x3db059b6.
//
// Solidity: function MAX_REGISTRATIONS_PER_DAY() view returns(uint256)
func (_SageVerificationHook *SageVerificationHookCaller) MAXREGISTRATIONSPERDAY(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _SageVerificationHook.contract.Call(opts, &out, "MAX_REGISTRATIONS_PER_DAY")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MAXREGISTRATIONSPERDAY is a free data retrieval call binding the contract method 0x3db059b6.
//
// Solidity: function MAX_REGISTRATIONS_PER_DAY() view returns(uint256)
func (_SageVerificationHook *SageVerificationHookSession) MAXREGISTRATIONSPERDAY() (*big.Int, error) {
	return _SageVerificationHook.Contract.MAXREGISTRATIONSPERDAY(&_SageVerificationHook.CallOpts)
}

// MAXREGISTRATIONSPERDAY is a free data retrieval call binding the contract method 0x3db059b6.
//
// Solidity: function MAX_REGISTRATIONS_PER_DAY() view returns(uint256)
func (_SageVerificationHook *SageVerificationHookCallerSession) MAXREGISTRATIONSPERDAY() (*big.Int, error) {
	return _SageVerificationHook.Contract.MAXREGISTRATIONSPERDAY(&_SageVerificationHook.CallOpts)
}

// OWNER is a free data retrieval call binding the contract method 0x117803e3.
//
// Solidity: function OWNER() view returns(address)
func (_SageVerificationHook *SageVerificationHookCaller) OWNER(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SageVerificationHook.contract.Call(opts, &out, "OWNER")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OWNER is a free data retrieval call binding the contract method 0x117803e3.
//
// Solidity: function OWNER() view returns(address)
func (_SageVerificationHook *SageVerificationHookSession) OWNER() (common.Address, error) {
	return _SageVerificationHook.Contract.OWNER(&_SageVerificationHook.CallOpts)
}

// OWNER is a free data retrieval call binding the contract method 0x117803e3.
//
// Solidity: function OWNER() view returns(address)
func (_SageVerificationHook *SageVerificationHookCallerSession) OWNER() (common.Address, error) {
	return _SageVerificationHook.Contract.OWNER(&_SageVerificationHook.CallOpts)
}

// REGISTRATIONCOOLDOWN is a free data retrieval call binding the contract method 0xb0ff6383.
//
// Solidity: function REGISTRATION_COOLDOWN() view returns(uint256)
func (_SageVerificationHook *SageVerificationHookCaller) REGISTRATIONCOOLDOWN(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _SageVerificationHook.contract.Call(opts, &out, "REGISTRATION_COOLDOWN")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// REGISTRATIONCOOLDOWN is a free data retrieval call binding the contract method 0xb0ff6383.
//
// Solidity: function REGISTRATION_COOLDOWN() view returns(uint256)
func (_SageVerificationHook *SageVerificationHookSession) REGISTRATIONCOOLDOWN() (*big.Int, error) {
	return _SageVerificationHook.Contract.REGISTRATIONCOOLDOWN(&_SageVerificationHook.CallOpts)
}

// REGISTRATIONCOOLDOWN is a free data retrieval call binding the contract method 0xb0ff6383.
//
// Solidity: function REGISTRATION_COOLDOWN() view returns(uint256)
func (_SageVerificationHook *SageVerificationHookCallerSession) REGISTRATIONCOOLDOWN() (*big.Int, error) {
	return _SageVerificationHook.Contract.REGISTRATIONCOOLDOWN(&_SageVerificationHook.CallOpts)
}

// Blacklisted is a free data retrieval call binding the contract method 0xdbac26e9.
//
// Solidity: function blacklisted(address ) view returns(bool)
func (_SageVerificationHook *SageVerificationHookCaller) Blacklisted(opts *bind.CallOpts, arg0 common.Address) (bool, error) {
	var out []interface{}
	err := _SageVerificationHook.contract.Call(opts, &out, "blacklisted", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Blacklisted is a free data retrieval call binding the contract method 0xdbac26e9.
//
// Solidity: function blacklisted(address ) view returns(bool)
func (_SageVerificationHook *SageVerificationHookSession) Blacklisted(arg0 common.Address) (bool, error) {
	return _SageVerificationHook.Contract.Blacklisted(&_SageVerificationHook.CallOpts, arg0)
}

// Blacklisted is a free data retrieval call binding the contract method 0xdbac26e9.
//
// Solidity: function blacklisted(address ) view returns(bool)
func (_SageVerificationHook *SageVerificationHookCallerSession) Blacklisted(arg0 common.Address) (bool, error) {
	return _SageVerificationHook.Contract.Blacklisted(&_SageVerificationHook.CallOpts, arg0)
}

// LastRegistrationTime is a free data retrieval call binding the contract method 0x7f6e8cbf.
//
// Solidity: function lastRegistrationTime(address ) view returns(uint256)
func (_SageVerificationHook *SageVerificationHookCaller) LastRegistrationTime(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _SageVerificationHook.contract.Call(opts, &out, "lastRegistrationTime", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// LastRegistrationTime is a free data retrieval call binding the contract method 0x7f6e8cbf.
//
// Solidity: function lastRegistrationTime(address ) view returns(uint256)
func (_SageVerificationHook *SageVerificationHookSession) LastRegistrationTime(arg0 common.Address) (*big.Int, error) {
	return _SageVerificationHook.Contract.LastRegistrationTime(&_SageVerificationHook.CallOpts, arg0)
}

// LastRegistrationTime is a free data retrieval call binding the contract method 0x7f6e8cbf.
//
// Solidity: function lastRegistrationTime(address ) view returns(uint256)
func (_SageVerificationHook *SageVerificationHookCallerSession) LastRegistrationTime(arg0 common.Address) (*big.Int, error) {
	return _SageVerificationHook.Contract.LastRegistrationTime(&_SageVerificationHook.CallOpts, arg0)
}

// RegistrationAttempts is a free data retrieval call binding the contract method 0x5a2a26bd.
//
// Solidity: function registrationAttempts(address ) view returns(uint256)
func (_SageVerificationHook *SageVerificationHookCaller) RegistrationAttempts(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _SageVerificationHook.contract.Call(opts, &out, "registrationAttempts", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// RegistrationAttempts is a free data retrieval call binding the contract method 0x5a2a26bd.
//
// Solidity: function registrationAttempts(address ) view returns(uint256)
func (_SageVerificationHook *SageVerificationHookSession) RegistrationAttempts(arg0 common.Address) (*big.Int, error) {
	return _SageVerificationHook.Contract.RegistrationAttempts(&_SageVerificationHook.CallOpts, arg0)
}

// RegistrationAttempts is a free data retrieval call binding the contract method 0x5a2a26bd.
//
// Solidity: function registrationAttempts(address ) view returns(uint256)
func (_SageVerificationHook *SageVerificationHookCallerSession) RegistrationAttempts(arg0 common.Address) (*big.Int, error) {
	return _SageVerificationHook.Contract.RegistrationAttempts(&_SageVerificationHook.CallOpts, arg0)
}

// AddToBlacklist is a paid mutator transaction binding the contract method 0x44337ea1.
//
// Solidity: function addToBlacklist(address account) returns()
func (_SageVerificationHook *SageVerificationHookTransactor) AddToBlacklist(opts *bind.TransactOpts, account common.Address) (*types.Transaction, error) {
	return _SageVerificationHook.contract.Transact(opts, "addToBlacklist", account)
}

// AddToBlacklist is a paid mutator transaction binding the contract method 0x44337ea1.
//
// Solidity: function addToBlacklist(address account) returns()
func (_SageVerificationHook *SageVerificationHookSession) AddToBlacklist(account common.Address) (*types.Transaction, error) {
	return _SageVerificationHook.Contract.AddToBlacklist(&_SageVerificationHook.TransactOpts, account)
}

// AddToBlacklist is a paid mutator transaction binding the contract method 0x44337ea1.
//
// Solidity: function addToBlacklist(address account) returns()
func (_SageVerificationHook *SageVerificationHookTransactorSession) AddToBlacklist(account common.Address) (*types.Transaction, error) {
	return _SageVerificationHook.Contract.AddToBlacklist(&_SageVerificationHook.TransactOpts, account)
}

// AfterRegister is a paid mutator transaction binding the contract method 0xc847ad35.
//
// Solidity: function afterRegister(bytes32 , address agentOwner, bytes ) returns()
func (_SageVerificationHook *SageVerificationHookTransactor) AfterRegister(opts *bind.TransactOpts, arg0 [32]byte, agentOwner common.Address, arg2 []byte) (*types.Transaction, error) {
	return _SageVerificationHook.contract.Transact(opts, "afterRegister", arg0, agentOwner, arg2)
}

// AfterRegister is a paid mutator transaction binding the contract method 0xc847ad35.
//
// Solidity: function afterRegister(bytes32 , address agentOwner, bytes ) returns()
func (_SageVerificationHook *SageVerificationHookSession) AfterRegister(arg0 [32]byte, agentOwner common.Address, arg2 []byte) (*types.Transaction, error) {
	return _SageVerificationHook.Contract.AfterRegister(&_SageVerificationHook.TransactOpts, arg0, agentOwner, arg2)
}

// AfterRegister is a paid mutator transaction binding the contract method 0xc847ad35.
//
// Solidity: function afterRegister(bytes32 , address agentOwner, bytes ) returns()
func (_SageVerificationHook *SageVerificationHookTransactorSession) AfterRegister(arg0 [32]byte, agentOwner common.Address, arg2 []byte) (*types.Transaction, error) {
	return _SageVerificationHook.Contract.AfterRegister(&_SageVerificationHook.TransactOpts, arg0, agentOwner, arg2)
}

// BeforeRegister is a paid mutator transaction binding the contract method 0x7b319ba1.
//
// Solidity: function beforeRegister(bytes32 , address agentOwner, bytes data) returns(bool success, string reason)
func (_SageVerificationHook *SageVerificationHookTransactor) BeforeRegister(opts *bind.TransactOpts, arg0 [32]byte, agentOwner common.Address, data []byte) (*types.Transaction, error) {
	return _SageVerificationHook.contract.Transact(opts, "beforeRegister", arg0, agentOwner, data)
}

// BeforeRegister is a paid mutator transaction binding the contract method 0x7b319ba1.
//
// Solidity: function beforeRegister(bytes32 , address agentOwner, bytes data) returns(bool success, string reason)
func (_SageVerificationHook *SageVerificationHookSession) BeforeRegister(arg0 [32]byte, agentOwner common.Address, data []byte) (*types.Transaction, error) {
	return _SageVerificationHook.Contract.BeforeRegister(&_SageVerificationHook.TransactOpts, arg0, agentOwner, data)
}

// BeforeRegister is a paid mutator transaction binding the contract method 0x7b319ba1.
//
// Solidity: function beforeRegister(bytes32 , address agentOwner, bytes data) returns(bool success, string reason)
func (_SageVerificationHook *SageVerificationHookTransactorSession) BeforeRegister(arg0 [32]byte, agentOwner common.Address, data []byte) (*types.Transaction, error) {
	return _SageVerificationHook.Contract.BeforeRegister(&_SageVerificationHook.TransactOpts, arg0, agentOwner, data)
}

// RemoveFromBlacklist is a paid mutator transaction binding the contract method 0x537df3b6.
//
// Solidity: function removeFromBlacklist(address account) returns()
func (_SageVerificationHook *SageVerificationHookTransactor) RemoveFromBlacklist(opts *bind.TransactOpts, account common.Address) (*types.Transaction, error) {
	return _SageVerificationHook.contract.Transact(opts, "removeFromBlacklist", account)
}

// RemoveFromBlacklist is a paid mutator transaction binding the contract method 0x537df3b6.
//
// Solidity: function removeFromBlacklist(address account) returns()
func (_SageVerificationHook *SageVerificationHookSession) RemoveFromBlacklist(account common.Address) (*types.Transaction, error) {
	return _SageVerificationHook.Contract.RemoveFromBlacklist(&_SageVerificationHook.TransactOpts, account)
}

// RemoveFromBlacklist is a paid mutator transaction binding the contract method 0x537df3b6.
//
// Solidity: function removeFromBlacklist(address account) returns()
func (_SageVerificationHook *SageVerificationHookTransactorSession) RemoveFromBlacklist(account common.Address) (*types.Transaction, error) {
	return _SageVerificationHook.Contract.RemoveFromBlacklist(&_SageVerificationHook.TransactOpts, account)
}

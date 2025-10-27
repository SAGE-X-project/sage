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

// AgentCardStorageMetaData contains all meta data concerning the AgentCardStorage contract.
var AgentCardStorageMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"}],\"name\":\"AgentDeactivatedByHash\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"string\",\"name\":\"did\",\"type\":\"string\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"}],\"name\":\"AgentRegistered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"}],\"name\":\"AgentUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"operator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"approved\",\"type\":\"bool\"}],\"name\":\"ApprovalForAgent\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"committer\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"commitHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"}],\"name\":\"CommitmentRecorded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"keyHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"enumAgentCardStorage.KeyType\",\"name\":\"keyType\",\"type\":\"uint8\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"}],\"name\":\"KeyAdded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"agentId\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"keyHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"}],\"name\":\"KeyRevoked\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"agentNonce\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"agentOperators\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"name\":\"didToAgentId\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"registrationCommitments\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"commitHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"revealed\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// AgentCardStorageABI is the input ABI used to generate the binding from.
// Deprecated: Use AgentCardStorageMetaData.ABI instead.
var AgentCardStorageABI = AgentCardStorageMetaData.ABI

// AgentCardStorage is an auto generated Go binding around an Ethereum contract.
type AgentCardStorage struct {
	AgentCardStorageCaller     // Read-only binding to the contract
	AgentCardStorageTransactor // Write-only binding to the contract
	AgentCardStorageFilterer   // Log filterer for contract events
}

// AgentCardStorageCaller is an auto generated read-only Go binding around an Ethereum contract.
type AgentCardStorageCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AgentCardStorageTransactor is an auto generated write-only Go binding around an Ethereum contract.
type AgentCardStorageTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AgentCardStorageFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type AgentCardStorageFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AgentCardStorageSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type AgentCardStorageSession struct {
	Contract     *AgentCardStorage // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// AgentCardStorageCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type AgentCardStorageCallerSession struct {
	Contract *AgentCardStorageCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts           // Call options to use throughout this session
}

// AgentCardStorageTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type AgentCardStorageTransactorSession struct {
	Contract     *AgentCardStorageTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts           // Transaction auth options to use throughout this session
}

// AgentCardStorageRaw is an auto generated low-level Go binding around an Ethereum contract.
type AgentCardStorageRaw struct {
	Contract *AgentCardStorage // Generic contract binding to access the raw methods on
}

// AgentCardStorageCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type AgentCardStorageCallerRaw struct {
	Contract *AgentCardStorageCaller // Generic read-only contract binding to access the raw methods on
}

// AgentCardStorageTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type AgentCardStorageTransactorRaw struct {
	Contract *AgentCardStorageTransactor // Generic write-only contract binding to access the raw methods on
}

// NewAgentCardStorage creates a new instance of AgentCardStorage, bound to a specific deployed contract.
func NewAgentCardStorage(address common.Address, backend bind.ContractBackend) (*AgentCardStorage, error) {
	contract, err := bindAgentCardStorage(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &AgentCardStorage{AgentCardStorageCaller: AgentCardStorageCaller{contract: contract}, AgentCardStorageTransactor: AgentCardStorageTransactor{contract: contract}, AgentCardStorageFilterer: AgentCardStorageFilterer{contract: contract}}, nil
}

// NewAgentCardStorageCaller creates a new read-only instance of AgentCardStorage, bound to a specific deployed contract.
func NewAgentCardStorageCaller(address common.Address, caller bind.ContractCaller) (*AgentCardStorageCaller, error) {
	contract, err := bindAgentCardStorage(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &AgentCardStorageCaller{contract: contract}, nil
}

// NewAgentCardStorageTransactor creates a new write-only instance of AgentCardStorage, bound to a specific deployed contract.
func NewAgentCardStorageTransactor(address common.Address, transactor bind.ContractTransactor) (*AgentCardStorageTransactor, error) {
	contract, err := bindAgentCardStorage(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &AgentCardStorageTransactor{contract: contract}, nil
}

// NewAgentCardStorageFilterer creates a new log filterer instance of AgentCardStorage, bound to a specific deployed contract.
func NewAgentCardStorageFilterer(address common.Address, filterer bind.ContractFilterer) (*AgentCardStorageFilterer, error) {
	contract, err := bindAgentCardStorage(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &AgentCardStorageFilterer{contract: contract}, nil
}

// bindAgentCardStorage binds a generic wrapper to an already deployed contract.
func bindAgentCardStorage(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := AgentCardStorageMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_AgentCardStorage *AgentCardStorageRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _AgentCardStorage.Contract.AgentCardStorageCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_AgentCardStorage *AgentCardStorageRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AgentCardStorage.Contract.AgentCardStorageTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_AgentCardStorage *AgentCardStorageRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _AgentCardStorage.Contract.AgentCardStorageTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_AgentCardStorage *AgentCardStorageCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _AgentCardStorage.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_AgentCardStorage *AgentCardStorageTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AgentCardStorage.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_AgentCardStorage *AgentCardStorageTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _AgentCardStorage.Contract.contract.Transact(opts, method, params...)
}

// AgentNonce is a free data retrieval call binding the contract method 0x6073c341.
//
// Solidity: function agentNonce(bytes32 ) view returns(uint256)
func (_AgentCardStorage *AgentCardStorageCaller) AgentNonce(opts *bind.CallOpts, arg0 [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _AgentCardStorage.contract.Call(opts, &out, "agentNonce", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// AgentNonce is a free data retrieval call binding the contract method 0x6073c341.
//
// Solidity: function agentNonce(bytes32 ) view returns(uint256)
func (_AgentCardStorage *AgentCardStorageSession) AgentNonce(arg0 [32]byte) (*big.Int, error) {
	return _AgentCardStorage.Contract.AgentNonce(&_AgentCardStorage.CallOpts, arg0)
}

// AgentNonce is a free data retrieval call binding the contract method 0x6073c341.
//
// Solidity: function agentNonce(bytes32 ) view returns(uint256)
func (_AgentCardStorage *AgentCardStorageCallerSession) AgentNonce(arg0 [32]byte) (*big.Int, error) {
	return _AgentCardStorage.Contract.AgentNonce(&_AgentCardStorage.CallOpts, arg0)
}

// AgentOperators is a free data retrieval call binding the contract method 0x0633d3d3.
//
// Solidity: function agentOperators(bytes32 , address ) view returns(bool)
func (_AgentCardStorage *AgentCardStorageCaller) AgentOperators(opts *bind.CallOpts, arg0 [32]byte, arg1 common.Address) (bool, error) {
	var out []interface{}
	err := _AgentCardStorage.contract.Call(opts, &out, "agentOperators", arg0, arg1)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// AgentOperators is a free data retrieval call binding the contract method 0x0633d3d3.
//
// Solidity: function agentOperators(bytes32 , address ) view returns(bool)
func (_AgentCardStorage *AgentCardStorageSession) AgentOperators(arg0 [32]byte, arg1 common.Address) (bool, error) {
	return _AgentCardStorage.Contract.AgentOperators(&_AgentCardStorage.CallOpts, arg0, arg1)
}

// AgentOperators is a free data retrieval call binding the contract method 0x0633d3d3.
//
// Solidity: function agentOperators(bytes32 , address ) view returns(bool)
func (_AgentCardStorage *AgentCardStorageCallerSession) AgentOperators(arg0 [32]byte, arg1 common.Address) (bool, error) {
	return _AgentCardStorage.Contract.AgentOperators(&_AgentCardStorage.CallOpts, arg0, arg1)
}

// DidToAgentId is a free data retrieval call binding the contract method 0xf0944df4.
//
// Solidity: function didToAgentId(string ) view returns(bytes32)
func (_AgentCardStorage *AgentCardStorageCaller) DidToAgentId(opts *bind.CallOpts, arg0 string) ([32]byte, error) {
	var out []interface{}
	err := _AgentCardStorage.contract.Call(opts, &out, "didToAgentId", arg0)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// DidToAgentId is a free data retrieval call binding the contract method 0xf0944df4.
//
// Solidity: function didToAgentId(string ) view returns(bytes32)
func (_AgentCardStorage *AgentCardStorageSession) DidToAgentId(arg0 string) ([32]byte, error) {
	return _AgentCardStorage.Contract.DidToAgentId(&_AgentCardStorage.CallOpts, arg0)
}

// DidToAgentId is a free data retrieval call binding the contract method 0xf0944df4.
//
// Solidity: function didToAgentId(string ) view returns(bytes32)
func (_AgentCardStorage *AgentCardStorageCallerSession) DidToAgentId(arg0 string) ([32]byte, error) {
	return _AgentCardStorage.Contract.DidToAgentId(&_AgentCardStorage.CallOpts, arg0)
}

// RegistrationCommitments is a free data retrieval call binding the contract method 0xadd0b94e.
//
// Solidity: function registrationCommitments(address ) view returns(bytes32 commitHash, uint256 timestamp, bool revealed)
func (_AgentCardStorage *AgentCardStorageCaller) RegistrationCommitments(opts *bind.CallOpts, arg0 common.Address) (struct {
	CommitHash [32]byte
	Timestamp  *big.Int
	Revealed   bool
}, error) {
	var out []interface{}
	err := _AgentCardStorage.contract.Call(opts, &out, "registrationCommitments", arg0)

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
func (_AgentCardStorage *AgentCardStorageSession) RegistrationCommitments(arg0 common.Address) (struct {
	CommitHash [32]byte
	Timestamp  *big.Int
	Revealed   bool
}, error) {
	return _AgentCardStorage.Contract.RegistrationCommitments(&_AgentCardStorage.CallOpts, arg0)
}

// RegistrationCommitments is a free data retrieval call binding the contract method 0xadd0b94e.
//
// Solidity: function registrationCommitments(address ) view returns(bytes32 commitHash, uint256 timestamp, bool revealed)
func (_AgentCardStorage *AgentCardStorageCallerSession) RegistrationCommitments(arg0 common.Address) (struct {
	CommitHash [32]byte
	Timestamp  *big.Int
	Revealed   bool
}, error) {
	return _AgentCardStorage.Contract.RegistrationCommitments(&_AgentCardStorage.CallOpts, arg0)
}

// AgentCardStorageAgentDeactivatedByHashIterator is returned from FilterAgentDeactivatedByHash and is used to iterate over the raw logs and unpacked data for AgentDeactivatedByHash events raised by the AgentCardStorage contract.
type AgentCardStorageAgentDeactivatedByHashIterator struct {
	Event *AgentCardStorageAgentDeactivatedByHash // Event containing the contract specifics and raw log

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
func (it *AgentCardStorageAgentDeactivatedByHashIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AgentCardStorageAgentDeactivatedByHash)
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
		it.Event = new(AgentCardStorageAgentDeactivatedByHash)
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
func (it *AgentCardStorageAgentDeactivatedByHashIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AgentCardStorageAgentDeactivatedByHashIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AgentCardStorageAgentDeactivatedByHash represents a AgentDeactivatedByHash event raised by the AgentCardStorage contract.
type AgentCardStorageAgentDeactivatedByHash struct {
	AgentId   [32]byte
	Timestamp *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterAgentDeactivatedByHash is a free log retrieval operation binding the contract event 0xb744c2ed9952c7041822a155d9cf761a7456f4876bdf9df34784d690c8647684.
//
// Solidity: event AgentDeactivatedByHash(bytes32 indexed agentId, uint256 timestamp)
func (_AgentCardStorage *AgentCardStorageFilterer) FilterAgentDeactivatedByHash(opts *bind.FilterOpts, agentId [][32]byte) (*AgentCardStorageAgentDeactivatedByHashIterator, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}

	logs, sub, err := _AgentCardStorage.contract.FilterLogs(opts, "AgentDeactivatedByHash", agentIdRule)
	if err != nil {
		return nil, err
	}
	return &AgentCardStorageAgentDeactivatedByHashIterator{contract: _AgentCardStorage.contract, event: "AgentDeactivatedByHash", logs: logs, sub: sub}, nil
}

// WatchAgentDeactivatedByHash is a free log subscription operation binding the contract event 0xb744c2ed9952c7041822a155d9cf761a7456f4876bdf9df34784d690c8647684.
//
// Solidity: event AgentDeactivatedByHash(bytes32 indexed agentId, uint256 timestamp)
func (_AgentCardStorage *AgentCardStorageFilterer) WatchAgentDeactivatedByHash(opts *bind.WatchOpts, sink chan<- *AgentCardStorageAgentDeactivatedByHash, agentId [][32]byte) (event.Subscription, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}

	logs, sub, err := _AgentCardStorage.contract.WatchLogs(opts, "AgentDeactivatedByHash", agentIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AgentCardStorageAgentDeactivatedByHash)
				if err := _AgentCardStorage.contract.UnpackLog(event, "AgentDeactivatedByHash", log); err != nil {
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
func (_AgentCardStorage *AgentCardStorageFilterer) ParseAgentDeactivatedByHash(log types.Log) (*AgentCardStorageAgentDeactivatedByHash, error) {
	event := new(AgentCardStorageAgentDeactivatedByHash)
	if err := _AgentCardStorage.contract.UnpackLog(event, "AgentDeactivatedByHash", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AgentCardStorageAgentRegisteredIterator is returned from FilterAgentRegistered and is used to iterate over the raw logs and unpacked data for AgentRegistered events raised by the AgentCardStorage contract.
type AgentCardStorageAgentRegisteredIterator struct {
	Event *AgentCardStorageAgentRegistered // Event containing the contract specifics and raw log

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
func (it *AgentCardStorageAgentRegisteredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AgentCardStorageAgentRegistered)
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
		it.Event = new(AgentCardStorageAgentRegistered)
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
func (it *AgentCardStorageAgentRegisteredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AgentCardStorageAgentRegisteredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AgentCardStorageAgentRegistered represents a AgentRegistered event raised by the AgentCardStorage contract.
type AgentCardStorageAgentRegistered struct {
	AgentId   [32]byte
	Did       common.Hash
	Owner     common.Address
	Timestamp *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterAgentRegistered is a free log retrieval operation binding the contract event 0x1d1edfbe93d381ab532cea862c0ee3fde2e6e88803b77f675fe3946635608ade.
//
// Solidity: event AgentRegistered(bytes32 indexed agentId, string indexed did, address indexed owner, uint256 timestamp)
func (_AgentCardStorage *AgentCardStorageFilterer) FilterAgentRegistered(opts *bind.FilterOpts, agentId [][32]byte, did []string, owner []common.Address) (*AgentCardStorageAgentRegisteredIterator, error) {

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

	logs, sub, err := _AgentCardStorage.contract.FilterLogs(opts, "AgentRegistered", agentIdRule, didRule, ownerRule)
	if err != nil {
		return nil, err
	}
	return &AgentCardStorageAgentRegisteredIterator{contract: _AgentCardStorage.contract, event: "AgentRegistered", logs: logs, sub: sub}, nil
}

// WatchAgentRegistered is a free log subscription operation binding the contract event 0x1d1edfbe93d381ab532cea862c0ee3fde2e6e88803b77f675fe3946635608ade.
//
// Solidity: event AgentRegistered(bytes32 indexed agentId, string indexed did, address indexed owner, uint256 timestamp)
func (_AgentCardStorage *AgentCardStorageFilterer) WatchAgentRegistered(opts *bind.WatchOpts, sink chan<- *AgentCardStorageAgentRegistered, agentId [][32]byte, did []string, owner []common.Address) (event.Subscription, error) {

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

	logs, sub, err := _AgentCardStorage.contract.WatchLogs(opts, "AgentRegistered", agentIdRule, didRule, ownerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AgentCardStorageAgentRegistered)
				if err := _AgentCardStorage.contract.UnpackLog(event, "AgentRegistered", log); err != nil {
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
func (_AgentCardStorage *AgentCardStorageFilterer) ParseAgentRegistered(log types.Log) (*AgentCardStorageAgentRegistered, error) {
	event := new(AgentCardStorageAgentRegistered)
	if err := _AgentCardStorage.contract.UnpackLog(event, "AgentRegistered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AgentCardStorageAgentUpdatedIterator is returned from FilterAgentUpdated and is used to iterate over the raw logs and unpacked data for AgentUpdated events raised by the AgentCardStorage contract.
type AgentCardStorageAgentUpdatedIterator struct {
	Event *AgentCardStorageAgentUpdated // Event containing the contract specifics and raw log

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
func (it *AgentCardStorageAgentUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AgentCardStorageAgentUpdated)
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
		it.Event = new(AgentCardStorageAgentUpdated)
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
func (it *AgentCardStorageAgentUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AgentCardStorageAgentUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AgentCardStorageAgentUpdated represents a AgentUpdated event raised by the AgentCardStorage contract.
type AgentCardStorageAgentUpdated struct {
	AgentId   [32]byte
	Timestamp *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterAgentUpdated is a free log retrieval operation binding the contract event 0x7a8c7d2cea9391cb6922e32d2c81a85e5b2307519a0f23f37665800328e42253.
//
// Solidity: event AgentUpdated(bytes32 indexed agentId, uint256 timestamp)
func (_AgentCardStorage *AgentCardStorageFilterer) FilterAgentUpdated(opts *bind.FilterOpts, agentId [][32]byte) (*AgentCardStorageAgentUpdatedIterator, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}

	logs, sub, err := _AgentCardStorage.contract.FilterLogs(opts, "AgentUpdated", agentIdRule)
	if err != nil {
		return nil, err
	}
	return &AgentCardStorageAgentUpdatedIterator{contract: _AgentCardStorage.contract, event: "AgentUpdated", logs: logs, sub: sub}, nil
}

// WatchAgentUpdated is a free log subscription operation binding the contract event 0x7a8c7d2cea9391cb6922e32d2c81a85e5b2307519a0f23f37665800328e42253.
//
// Solidity: event AgentUpdated(bytes32 indexed agentId, uint256 timestamp)
func (_AgentCardStorage *AgentCardStorageFilterer) WatchAgentUpdated(opts *bind.WatchOpts, sink chan<- *AgentCardStorageAgentUpdated, agentId [][32]byte) (event.Subscription, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}

	logs, sub, err := _AgentCardStorage.contract.WatchLogs(opts, "AgentUpdated", agentIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AgentCardStorageAgentUpdated)
				if err := _AgentCardStorage.contract.UnpackLog(event, "AgentUpdated", log); err != nil {
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
func (_AgentCardStorage *AgentCardStorageFilterer) ParseAgentUpdated(log types.Log) (*AgentCardStorageAgentUpdated, error) {
	event := new(AgentCardStorageAgentUpdated)
	if err := _AgentCardStorage.contract.UnpackLog(event, "AgentUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AgentCardStorageApprovalForAgentIterator is returned from FilterApprovalForAgent and is used to iterate over the raw logs and unpacked data for ApprovalForAgent events raised by the AgentCardStorage contract.
type AgentCardStorageApprovalForAgentIterator struct {
	Event *AgentCardStorageApprovalForAgent // Event containing the contract specifics and raw log

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
func (it *AgentCardStorageApprovalForAgentIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AgentCardStorageApprovalForAgent)
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
		it.Event = new(AgentCardStorageApprovalForAgent)
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
func (it *AgentCardStorageApprovalForAgentIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AgentCardStorageApprovalForAgentIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AgentCardStorageApprovalForAgent represents a ApprovalForAgent event raised by the AgentCardStorage contract.
type AgentCardStorageApprovalForAgent struct {
	AgentId  [32]byte
	Owner    common.Address
	Operator common.Address
	Approved bool
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterApprovalForAgent is a free log retrieval operation binding the contract event 0x5f0fcbccc201a658d8ff8a6739186b23cce22dd2c40436fbd54d325bf28b1983.
//
// Solidity: event ApprovalForAgent(bytes32 indexed agentId, address indexed owner, address indexed operator, bool approved)
func (_AgentCardStorage *AgentCardStorageFilterer) FilterApprovalForAgent(opts *bind.FilterOpts, agentId [][32]byte, owner []common.Address, operator []common.Address) (*AgentCardStorageApprovalForAgentIterator, error) {

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

	logs, sub, err := _AgentCardStorage.contract.FilterLogs(opts, "ApprovalForAgent", agentIdRule, ownerRule, operatorRule)
	if err != nil {
		return nil, err
	}
	return &AgentCardStorageApprovalForAgentIterator{contract: _AgentCardStorage.contract, event: "ApprovalForAgent", logs: logs, sub: sub}, nil
}

// WatchApprovalForAgent is a free log subscription operation binding the contract event 0x5f0fcbccc201a658d8ff8a6739186b23cce22dd2c40436fbd54d325bf28b1983.
//
// Solidity: event ApprovalForAgent(bytes32 indexed agentId, address indexed owner, address indexed operator, bool approved)
func (_AgentCardStorage *AgentCardStorageFilterer) WatchApprovalForAgent(opts *bind.WatchOpts, sink chan<- *AgentCardStorageApprovalForAgent, agentId [][32]byte, owner []common.Address, operator []common.Address) (event.Subscription, error) {

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

	logs, sub, err := _AgentCardStorage.contract.WatchLogs(opts, "ApprovalForAgent", agentIdRule, ownerRule, operatorRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AgentCardStorageApprovalForAgent)
				if err := _AgentCardStorage.contract.UnpackLog(event, "ApprovalForAgent", log); err != nil {
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
func (_AgentCardStorage *AgentCardStorageFilterer) ParseApprovalForAgent(log types.Log) (*AgentCardStorageApprovalForAgent, error) {
	event := new(AgentCardStorageApprovalForAgent)
	if err := _AgentCardStorage.contract.UnpackLog(event, "ApprovalForAgent", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AgentCardStorageCommitmentRecordedIterator is returned from FilterCommitmentRecorded and is used to iterate over the raw logs and unpacked data for CommitmentRecorded events raised by the AgentCardStorage contract.
type AgentCardStorageCommitmentRecordedIterator struct {
	Event *AgentCardStorageCommitmentRecorded // Event containing the contract specifics and raw log

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
func (it *AgentCardStorageCommitmentRecordedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AgentCardStorageCommitmentRecorded)
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
		it.Event = new(AgentCardStorageCommitmentRecorded)
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
func (it *AgentCardStorageCommitmentRecordedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AgentCardStorageCommitmentRecordedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AgentCardStorageCommitmentRecorded represents a CommitmentRecorded event raised by the AgentCardStorage contract.
type AgentCardStorageCommitmentRecorded struct {
	Committer  common.Address
	CommitHash [32]byte
	Timestamp  *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterCommitmentRecorded is a free log retrieval operation binding the contract event 0xf76059fd91b15b2d465b41fe5d794955f8ac948e38e126713fbfb120585ff6bc.
//
// Solidity: event CommitmentRecorded(address indexed committer, bytes32 commitHash, uint256 timestamp)
func (_AgentCardStorage *AgentCardStorageFilterer) FilterCommitmentRecorded(opts *bind.FilterOpts, committer []common.Address) (*AgentCardStorageCommitmentRecordedIterator, error) {

	var committerRule []interface{}
	for _, committerItem := range committer {
		committerRule = append(committerRule, committerItem)
	}

	logs, sub, err := _AgentCardStorage.contract.FilterLogs(opts, "CommitmentRecorded", committerRule)
	if err != nil {
		return nil, err
	}
	return &AgentCardStorageCommitmentRecordedIterator{contract: _AgentCardStorage.contract, event: "CommitmentRecorded", logs: logs, sub: sub}, nil
}

// WatchCommitmentRecorded is a free log subscription operation binding the contract event 0xf76059fd91b15b2d465b41fe5d794955f8ac948e38e126713fbfb120585ff6bc.
//
// Solidity: event CommitmentRecorded(address indexed committer, bytes32 commitHash, uint256 timestamp)
func (_AgentCardStorage *AgentCardStorageFilterer) WatchCommitmentRecorded(opts *bind.WatchOpts, sink chan<- *AgentCardStorageCommitmentRecorded, committer []common.Address) (event.Subscription, error) {

	var committerRule []interface{}
	for _, committerItem := range committer {
		committerRule = append(committerRule, committerItem)
	}

	logs, sub, err := _AgentCardStorage.contract.WatchLogs(opts, "CommitmentRecorded", committerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AgentCardStorageCommitmentRecorded)
				if err := _AgentCardStorage.contract.UnpackLog(event, "CommitmentRecorded", log); err != nil {
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
func (_AgentCardStorage *AgentCardStorageFilterer) ParseCommitmentRecorded(log types.Log) (*AgentCardStorageCommitmentRecorded, error) {
	event := new(AgentCardStorageCommitmentRecorded)
	if err := _AgentCardStorage.contract.UnpackLog(event, "CommitmentRecorded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AgentCardStorageKeyAddedIterator is returned from FilterKeyAdded and is used to iterate over the raw logs and unpacked data for KeyAdded events raised by the AgentCardStorage contract.
type AgentCardStorageKeyAddedIterator struct {
	Event *AgentCardStorageKeyAdded // Event containing the contract specifics and raw log

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
func (it *AgentCardStorageKeyAddedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AgentCardStorageKeyAdded)
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
		it.Event = new(AgentCardStorageKeyAdded)
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
func (it *AgentCardStorageKeyAddedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AgentCardStorageKeyAddedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AgentCardStorageKeyAdded represents a KeyAdded event raised by the AgentCardStorage contract.
type AgentCardStorageKeyAdded struct {
	AgentId   [32]byte
	KeyHash   [32]byte
	KeyType   uint8
	Timestamp *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterKeyAdded is a free log retrieval operation binding the contract event 0x11f138c8931fc92ab4fbeb5dd32df17d56c9411a543739c3526ed0265d8fad13.
//
// Solidity: event KeyAdded(bytes32 indexed agentId, bytes32 indexed keyHash, uint8 keyType, uint256 timestamp)
func (_AgentCardStorage *AgentCardStorageFilterer) FilterKeyAdded(opts *bind.FilterOpts, agentId [][32]byte, keyHash [][32]byte) (*AgentCardStorageKeyAddedIterator, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}
	var keyHashRule []interface{}
	for _, keyHashItem := range keyHash {
		keyHashRule = append(keyHashRule, keyHashItem)
	}

	logs, sub, err := _AgentCardStorage.contract.FilterLogs(opts, "KeyAdded", agentIdRule, keyHashRule)
	if err != nil {
		return nil, err
	}
	return &AgentCardStorageKeyAddedIterator{contract: _AgentCardStorage.contract, event: "KeyAdded", logs: logs, sub: sub}, nil
}

// WatchKeyAdded is a free log subscription operation binding the contract event 0x11f138c8931fc92ab4fbeb5dd32df17d56c9411a543739c3526ed0265d8fad13.
//
// Solidity: event KeyAdded(bytes32 indexed agentId, bytes32 indexed keyHash, uint8 keyType, uint256 timestamp)
func (_AgentCardStorage *AgentCardStorageFilterer) WatchKeyAdded(opts *bind.WatchOpts, sink chan<- *AgentCardStorageKeyAdded, agentId [][32]byte, keyHash [][32]byte) (event.Subscription, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}
	var keyHashRule []interface{}
	for _, keyHashItem := range keyHash {
		keyHashRule = append(keyHashRule, keyHashItem)
	}

	logs, sub, err := _AgentCardStorage.contract.WatchLogs(opts, "KeyAdded", agentIdRule, keyHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AgentCardStorageKeyAdded)
				if err := _AgentCardStorage.contract.UnpackLog(event, "KeyAdded", log); err != nil {
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
func (_AgentCardStorage *AgentCardStorageFilterer) ParseKeyAdded(log types.Log) (*AgentCardStorageKeyAdded, error) {
	event := new(AgentCardStorageKeyAdded)
	if err := _AgentCardStorage.contract.UnpackLog(event, "KeyAdded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AgentCardStorageKeyRevokedIterator is returned from FilterKeyRevoked and is used to iterate over the raw logs and unpacked data for KeyRevoked events raised by the AgentCardStorage contract.
type AgentCardStorageKeyRevokedIterator struct {
	Event *AgentCardStorageKeyRevoked // Event containing the contract specifics and raw log

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
func (it *AgentCardStorageKeyRevokedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AgentCardStorageKeyRevoked)
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
		it.Event = new(AgentCardStorageKeyRevoked)
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
func (it *AgentCardStorageKeyRevokedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AgentCardStorageKeyRevokedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AgentCardStorageKeyRevoked represents a KeyRevoked event raised by the AgentCardStorage contract.
type AgentCardStorageKeyRevoked struct {
	AgentId   [32]byte
	KeyHash   [32]byte
	Timestamp *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterKeyRevoked is a free log retrieval operation binding the contract event 0x209fb85e2522622566ffdf13e48258218f4c155aefc75703539e1a971380cd3f.
//
// Solidity: event KeyRevoked(bytes32 indexed agentId, bytes32 indexed keyHash, uint256 timestamp)
func (_AgentCardStorage *AgentCardStorageFilterer) FilterKeyRevoked(opts *bind.FilterOpts, agentId [][32]byte, keyHash [][32]byte) (*AgentCardStorageKeyRevokedIterator, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}
	var keyHashRule []interface{}
	for _, keyHashItem := range keyHash {
		keyHashRule = append(keyHashRule, keyHashItem)
	}

	logs, sub, err := _AgentCardStorage.contract.FilterLogs(opts, "KeyRevoked", agentIdRule, keyHashRule)
	if err != nil {
		return nil, err
	}
	return &AgentCardStorageKeyRevokedIterator{contract: _AgentCardStorage.contract, event: "KeyRevoked", logs: logs, sub: sub}, nil
}

// WatchKeyRevoked is a free log subscription operation binding the contract event 0x209fb85e2522622566ffdf13e48258218f4c155aefc75703539e1a971380cd3f.
//
// Solidity: event KeyRevoked(bytes32 indexed agentId, bytes32 indexed keyHash, uint256 timestamp)
func (_AgentCardStorage *AgentCardStorageFilterer) WatchKeyRevoked(opts *bind.WatchOpts, sink chan<- *AgentCardStorageKeyRevoked, agentId [][32]byte, keyHash [][32]byte) (event.Subscription, error) {

	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}
	var keyHashRule []interface{}
	for _, keyHashItem := range keyHash {
		keyHashRule = append(keyHashRule, keyHashItem)
	}

	logs, sub, err := _AgentCardStorage.contract.WatchLogs(opts, "KeyRevoked", agentIdRule, keyHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AgentCardStorageKeyRevoked)
				if err := _AgentCardStorage.contract.UnpackLog(event, "KeyRevoked", log); err != nil {
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
func (_AgentCardStorage *AgentCardStorageFilterer) ParseKeyRevoked(log types.Log) (*AgentCardStorageKeyRevoked, error) {
	event := new(AgentCardStorageKeyRevoked)
	if err := _AgentCardStorage.contract.UnpackLog(event, "KeyRevoked", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

package registry011

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

func replayGeneratedOperationObserved(method string, meter *providerWorkMeter) (string, int, bool) {
	if meter == nil {
		return "", 0, false
	}
	for _, item := range [...]struct {
		method, operation string
		arity             int
	}{
		{"revealClaim", "create", 5}, {"activate", "activate", 2}, {"deactivate", "deactivate", 2},
		{"addKey", "add-key", 3}, {"revokeKey", "revoke-key", 3}, {"updateServices", "update-services", 3},
		{"authorizeOperator", "authorize-operator", 4}, {"revokeOperator", "revoke-operator", 4},
	} {
		if providerReplayTextEqual(method, item.method, meter) {
			return item.operation, item.arity, true
		}
	}
	return "", 0, false
}

// Values are the unchanged result of the adopted ABI's framed canonical
// decode. This path owns the tuple conversion/generation work before handing
// its actual bytes to the same original strict consumer. It grants no Source
// authority and does not qualify ABI construction/claims/state/native costs.
func replayDecodedCommandObserved(registry, actor, method string, values []any, original []byte, contract abi.ABI, meter *providerWorkMeter) (replayCommand, bool) {
	var empty replayCommand
	op, arity, ok := replayGeneratedOperationObserved(method, meter)
	if !ok || len(values) != arity || len(original) < 4 || len(original) > 148804 || !providerReplayTextEqual(registry, "eip155:11155111:"+SepoliaRegistryAddress, meter) || !replayContentAddressObserved(actor, meter) {
		return empty, false
	}
	agent, ok := values[0].(string)
	if !ok || len(agent) > 64 || !meter.event(providerCompareEvent, uint64(len(agent))) || !rpcAgent(agent) {
		return empty, false
	}
	expected := ""
	if op != "create" {
		version, ok := values[1].(uint64)
		if !ok {
			return empty, false
		}
		expected, ok = replayGeneratedUintObserved(version, meter)
		if !ok {
			return empty, false
		}
	}
	args, ok := replayGeneratedArgumentsObserved(op, values, meter)
	if !ok {
		return empty, false
	}
	did := providerReplayJoin([]string{"did:sage:", registry, ":", agent}, meter)
	value, ok := replayGeneratedObjectObserved([]string{"binding", "registry", "did", "actor", "operation", "expectedVersion", "arguments"}, []any{"sage-registry-chain-0.11.0", registry, did, actor, op, expected, args}, meter)
	if !ok {
		return empty, false
	}
	raw, ok := replayGeneratedMarshalObserved(value, meter)
	if !ok {
		return empty, false
	}
	c, ok := replayParseCommandObserved(registry, agent, raw, contract, meter)
	if !ok || !providerReplayEqual(c.calldata, original, meter) {
		return empty, false
	}
	return c, true
}

func replayGeneratedArgumentsObserved(operation string, values []any, meter *providerWorkMeter) (map[string]any, bool) {
	count := 0
	switch operation {
	case "create":
		count = 5
	case "activate", "deactivate":
		count = 2
	case "add-key", "revoke-key", "update-services":
		count = 3
	case "authorize-operator", "revoke-operator":
		count = 4
	}
	if meter == nil || count == 0 || len(values) != count {
		return nil, false
	}
	switch operation {
	case "create":
		controller, ok := values[1].(common.Address)
		if !ok {
			return nil, false
		}
		address, ok := nodeAddressTextObserved(controller, meter)
		if !ok {
			return nil, false
		}
		keys, ok := replayCommandTupleKeysObserved(values[2], meter)
		if !ok {
			return nil, false
		}
		keyValues, ok := replayGeneratedKeysObserved(keys, meter)
		if !ok {
			return nil, false
		}
		services, ok := replayCommandTupleServicesObserved(values[3], meter)
		if !ok {
			return nil, false
		}
		serviceValues, ok := replayGeneratedServicesObserved(services, meter)
		if !ok {
			return nil, false
		}
		salt, ok := values[4].([32]byte)
		if !ok || !meter.event(providerCopyEvent, 32) {
			return nil, false
		}
		saltValue, ok := replayGeneratedBase64Observed(salt[:], meter)
		if !ok {
			return nil, false
		}
		return replayGeneratedObjectObserved([]string{"controller", "keys", "services", "salt"}, []any{address, keyValues, serviceValues, saltValue}, meter)
	case "activate", "deactivate":
		return replayGeneratedObjectObserved(nil, nil, meter)
	case "add-key":
		key, ok := replayCommandTupleKeyObserved(values[2], meter)
		if !ok {
			return nil, false
		}
		value, ok := replayGeneratedKeyObserved(key, meter)
		if !ok {
			return nil, false
		}
		return replayGeneratedObjectObserved([]string{"key"}, []any{value}, meter)
	case "revoke-key":
		name, ok := values[2].(string)
		if !ok {
			return nil, false
		}
		return replayGeneratedObjectObserved([]string{"name"}, []any{name}, meter)
	case "update-services":
		services, ok := replayCommandTupleServicesObserved(values[2], meter)
		if !ok {
			return nil, false
		}
		value, ok := replayGeneratedServicesObserved(services, meter)
		if !ok {
			return nil, false
		}
		return replayGeneratedObjectObserved([]string{"services"}, []any{value}, meter)
	case "authorize-operator", "revoke-operator":
		operator, ok := values[2].(common.Address)
		if !ok {
			return nil, false
		}
		scope, ok := values[3].(string)
		if !ok {
			return nil, false
		}
		address, ok := nodeAddressTextObserved(operator, meter)
		if !ok {
			return nil, false
		}
		return replayGeneratedObjectObserved([]string{"operator", "scope"}, []any{address, scope}, meter)
	}
	return nil, false
}

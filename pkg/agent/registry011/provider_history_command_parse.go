package registry011

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// This is the private command consumer, not an entry point that accepts a
// producer's ABI or grants collection/Source authority. Legacy nil DATA keeps
// its original parser. All nonnil work uses the actual original observer.
func replayParseCommandObserved(registry, agent string, raw []byte, contract abi.ABI, meter *providerWorkMeter) (replayCommand, bool) {
	if meter == nil {
		return replayParseCommand(registry, agent, raw, contract)
	}
	var empty replayCommand
	if len(raw) == 0 || len(raw) > 262144 || !meter.event(providerCompareEvent, uint64(len(registry))) || registry != "eip155:11155111:"+SepoliaRegistryAddress || len(agent) > 64 || !meter.event(providerCompareEvent, uint64(len(agent))) || !rpcAgent(agent) || !meter.event(providerCopyEvent, uint64(len(raw))) {
		return empty, false
	}
	c := replayCommand{agent: agent, raw: bytes.Clone(raw)}
	m := replayCommandObjectObserved(replayCommandJSONObserved(c.raw, meter), []string{"binding", "registry", "did", "operation", "expectedVersion", "arguments", "actor"}, meter)
	did := providerReplayJoin([]string{"did:sage:", registry, ":", agent}, meter)
	if m == nil || !providerReplayTextEqual(contentString(m["binding"]), "sage-registry-chain-0.11.0", meter) || !providerReplayTextEqual(contentString(m["registry"]), registry, meter) || !providerReplayTextEqual(contentString(m["did"]), did, meter) {
		return empty, false
	}
	c.op, c.actor = contentString(m["operation"]), contentString(m["actor"])
	args, ok := replayCommandArgumentsObserved(c.op, m["arguments"], meter)
	if !ok {
		return empty, false
	}
	actor, ok := replayCommandAddressObserved(c.actor, meter)
	if !ok {
		return empty, false
	}
	c.args = args
	expected, ok := m["expectedVersion"].(string)
	if !ok {
		return empty, false
	}
	count := 2
	switch c.op {
	case "create":
		count = 5
	case "add-key", "revoke-key", "update-services":
		count = 3
	case "authorize-operator", "revoke-operator":
		count = 4
	}
	params, ok := replayCommandParamsObserved(count, meter)
	if !ok || !replayCommandParamObserved(params, 0, agent, meter) {
		return empty, false
	}
	if c.op == "create" {
		if !providerReplayTextEqual(expected, "", meter) || !providerReplayTextEqual(contentString(args["controller"]), c.actor, meter) {
			return empty, false
		}
		c.keys, ok = replayCommandKeysObserved(args["keys"], did, meter)
		if !ok {
			return empty, false
		}
		c.services, ok = replayCommandServicesObserved(args["services"], c.keys, meter)
		if !ok {
			return empty, false
		}
		salt, ok := replayCommandBinaryObserved(args["salt"], 32, meter)
		if !ok || len(salt) != 32 {
			return empty, false
		}
		keys, ok := replayCommandABIKeysObserved(c.keys, meter)
		if !ok {
			return empty, false
		}
		services, ok := replayCommandABIServicesObserved(c.services, meter)
		if !ok || !meter.event(providerCopyEvent, 32) {
			return empty, false
		}
		var saltWord [32]byte
		copy(saltWord[:], salt)
		if !replayCommandParamObserved(params, 1, actor, meter) || !replayCommandParamObserved(params, 2, keys, meter) || !replayCommandParamObserved(params, 3, services, meter) || !replayCommandParamObserved(params, 4, saltWord, meter) {
			return empty, false
		}
	} else {
		c.expected, ok = replayCommandUintObserved(expected, meter)
		if !ok || c.expected == 0 || !replayCommandParamObserved(params, 1, c.expected, meter) {
			return empty, false
		}
		switch c.op {
		case "add-key":
			c.keys, ok = replayCommandKeysObserved([]any{args["key"]}, did, meter)
			if !ok {
				return empty, false
			}
			keys, ok := replayCommandABIKeysObserved(c.keys, meter)
			if !ok || len(keys) != 1 {
				return empty, false
			}
			for _, key := range keys {
				if !replayCommandParamObserved(params, 2, key, meter) {
					return empty, false
				}
			}
		case "revoke-key":
			if !replayCommandNameObserved(contentString(args["name"]), meter) || !replayCommandParamObserved(params, 2, args["name"], meter) {
				return empty, false
			}
		case "update-services":
			c.services, ok = replayCommandServicesObserved(args["services"], nil, meter)
			if !ok {
				return empty, false
			}
			services, ok := replayCommandABIServicesObserved(c.services, meter)
			if !ok || !replayCommandParamObserved(params, 2, services, meter) {
				return empty, false
			}
		case "authorize-operator", "revoke-operator":
			if !replayCommandGrantObserved(args, meter) {
				return empty, false
			}
			operator, ok := replayCommandAddressObserved(contentString(args["operator"]), meter)
			if !ok || !replayCommandParamObserved(params, 2, operator, meter) || !replayCommandParamObserved(params, 3, args["scope"], meter) {
				return empty, false
			}
		}
	}
	c.calldata, ok = replayCommandPackObserved(contract, replayMethods[c.op], params, meter)
	if !ok || !meter.preflightWork(candidateWorkVector{}) {
		return empty, false
	}
	return c, true
}

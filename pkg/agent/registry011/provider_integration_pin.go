package registry011

import (
	"context"
	"encoding/json"
)

func providerPinInput(start, agent string) []byte {
	if !lowerHash(start) || !rpcAgent(agent) {
		return nil
	}
	return providerIntegrationJSON(map[string]string{"format": "sage-provider-owned-pin-input-v1", "bootstrap_start": start, "registry": SepoliaRegistryAddress, "agent": agent})
}

// This initial pin reuses the first retained finalized exchange. In particular
// it cannot call the old chain-first PinAgentReads to obtain a free new start.
// The gated client remains private to the subsequent concrete collector.
func (c *providerIntegration) pinAgent(ctx context.Context, agent string) (*PinnedAgentReads, error) {
	if c == nil || ctx == nil || ctx.Err() != nil {
		return nil, errProviderIntegration
	}
	c.command.Lock()
	defer c.command.Unlock()
	if !c.valid() || c.bootstrap == "" || c.pin != "" || !rpcAgent(agent) {
		return nil, errProviderIntegration
	}
	raw := providerPinInput(c.bootstrap, agent)
	input, err := newProviderIntegrationObject("pin-input", raw, c.store.config.bounds.maximumObjectBytes)
	if err != nil {
		return nil, err
	}
	permit, err := c.beginLocalObjects(input.digest, []providerIntegrationObjectInput{{"pin-input", raw}})
	if err != nil {
		return nil, err
	}
	defer c.restoreMeter(permit.meter)
	client, err := c.readClient()
	if err != nil {
		c.failLocal(permit)
		return nil, err
	}
	accepted := false
	defer func() {
		if !accepted {
			client.Close()
			c.failLocal(permit)
		}
	}()
	reads := &PinnedAgentReads{client, SepoliaRegistryAddress, agent, c.selected}
	var digests [4]string
	methods := [...]string{"eth_chainId", "eth_getBlockByNumber", "web3_clientVersion", "eth_call"}
	params := providerPinParams(c.selected.Hash)
	for i, method := range methods {
		result, err := client.call(ctx, method, params[i])
		if err != nil {
			return nil, err
		}
		if !c.store.checkPinPart(i, result) {
			return nil, errProviderIntegration
		}
		digests[i] = c.lastExchange
	}
	pin, err := c.finishPinResult(permit, input.digest, digests)
	if err != nil {
		return nil, err
	}
	c.pin = pin
	accepted = true
	return reads, nil
}

func providerCheckPinResults(results [4]json.RawMessage) bool {
	for i, result := range results {
		if !providerCheckPinPart(i, result) {
			return false
		}
	}
	return true
}

// Each validation invocation decodes its actual input again, even when it is
// the same header material. A prior decode or a generous reservation cannot
// stand in for observing this invocation's bytes and logical work events.
func (s *providerCustodyStore) checkPinPart(part int, result json.RawMessage) bool {
	return s.event(providerCodecEvent, uint64(len(result))) && providerCheckPinPart(part, result)
}

func (s *providerCustodyStore) checkPinResults(results [4]json.RawMessage) bool {
	for i, result := range results {
		if !s.checkPinPart(i, result) {
			return false
		}
	}
	return true
}

func providerCheckPinPart(part int, result json.RawMessage) bool {
	var chain, version string
	switch part {
	case 0:
		return json.Unmarshal(result, &chain) == nil && chain == "0xaa36a7"
	case 1:
		genesis, err := providerDecodeHeader(result)
		return err == nil && genesis.Hash == sepoliaGenesis && genesis.Number == 0
	case 2:
		return json.Unmarshal(result, &version) == nil && len(version) != 0 && len(version) <= 1024
	case 3:
		values, _, err := rpcABI(result, false, 128)
		return err == nil && string(values[0]) == "eip155:11155111:"+SepoliaRegistryAddress
	}
	return false
}

func providerDecodeHeader(raw []byte) (FinalizedHeader, error) {
	var value struct{ Hash, ParentHash, Number, Timestamp string }
	if json.Unmarshal(raw, &value) != nil {
		return FinalizedHeader{}, errProviderIntegration
	}
	number, a := rpcQuantity(value.Number)
	timestamp, b := rpcQuantity(value.Timestamp)
	header := FinalizedHeader{value.Hash, value.ParentHash, number, timestamp}
	if !a || !b || !validHeader(header) {
		return FinalizedHeader{}, errProviderIntegration
	}
	return header, nil
}

func providerPinRequests(selected string) ([4]*providerIntegrationRequest, error) {
	var requests [4]*providerIntegrationRequest
	if !lowerHash(selected) {
		return requests, errProviderIntegration
	}
	methods := [...]string{"eth_chainId", "eth_getBlockByNumber", "web3_clientVersion", "eth_call"}
	params := providerPinParams(selected)
	for i := range requests {
		var err error
		requests[i], err = newProviderIntegrationRequest(methods[i], params[i])
		if err != nil {
			return requests, err
		}
	}
	return requests, nil
}

func providerPinParams(selected string) [4]any {
	return [4]any{[]any{}, []any{"0x0", false}, []any{}, []any{map[string]string{"to": SepoliaRegistryAddress, "data": "0x04ef898d"}, map[string]any{"blockHash": selected, "requireCanonical": true}}}
}

// A closed local result still requires the exact retained child exchanges and
// the original live parent permit. This helper cannot acquire or dispatch RPC.
func (c *providerIntegration) finishPinResult(permit *providerIntegrationPermit, input string, digests [4]string) (string, error) {
	// The live path uses the same independently retained typed response closure
	// that ACK readback uses; no success is inferred merely from HTTP status.
	results, err := c.store.replay.pinResults(c.store, digests, c.selected.Hash)
	var observed candidateWorkVector
	observed[3] = 1 // The owned pin parser decodes the one genesis header.
	if err != nil || !permit.meter.add(observed) || !c.store.checkPinResults(results) {
		return "", errProviderIntegration
	}
	raw := providerIntegrationJSON(map[string]any{"format": "sage-provider-owned-pin-v1", "input": input, "selected_block_hash": c.selected.Hash, "exchanges": digests[:]})
	return c.finishLocalResult(permit, "pin", raw)
}

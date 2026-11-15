package registry011

import (
	"encoding/hex"

	"golang.org/x/crypto/sha3"
)

// Closed calldata DATA, constructed under the original caller's observer.
// No ReadRPC/PinnedAgentReads or callback is synthesized to obtain its bytes.
func providerHistoryRegistryParamsObserved(registry, agent, method string, version uint64, block string, meter *providerWorkMeter) ([]any, error) {
	if registry != SepoliaRegistryAddress || !rpcAgent(agent) || !lowerHash(block) || meter == nil {
		return nil, ErrConfiguration
	}
	var signature string
	switch method {
	case "record":
		signature = "record(string)"
	case "activeGrants":
		signature = "activeGrants(string)"
	case "historyItem":
		signature = "historyItem(string,uint64)"
	default:
		return nil, ErrConfiguration
	}
	if (method == "historyItem") != (version != 0) || !meter.event(providerHashEvent, uint64(len(signature))) {
		return nil, ErrConfiguration
	}
	hash := sha3.NewLegacyKeccak256()
	_, _ = hash.Write([]byte(signature))
	selector := hash.Sum(nil)
	head := uint64(32)
	if method == "historyItem" {
		head = 64
	}
	padding := (32 - len(agent)%32) % 32
	size := 4 + int(head) + 32 + len(agent) + padding
	if !meter.event(providerCopyEvent, uint64(size)) {
		return nil, ErrConfiguration
	}
	data := make([]byte, 0, size)
	data = append(data, selector[:4]...)
	data = append(data, rpcWord(head)...)
	if method == "historyItem" {
		data = append(data, rpcWord(version)...)
	}
	data = append(data, rpcWord(uint64(len(agent)))...)
	data = append(data, agent...)
	data = append(data, make([]byte, padding)...)
	if !meter.event(providerEncodeEvent, uint64(len(data))) || !meter.event(providerCopyEvent, uint64(2+2*len(data))) {
		return nil, ErrConfiguration
	}
	return []any{map[string]string{"to": registry, "data": "0x" + hex.EncodeToString(data)}, map[string]any{"blockHash": block, "requireCanonical": true}}, nil
}

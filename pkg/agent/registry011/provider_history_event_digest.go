package registry011

import (
	"crypto/sha256"

	"github.com/ethereum/go-ethereum/common"
)

// Payload bounds are the existing command/grants limits plus at most64fixed
// domain bytes. Logical hash/copy observations do not qualify native CPU/RAM.
func replaySHA256Observed(raw []byte, meter *providerWorkMeter) ([32]byte, bool) {
	if meter != nil && (len(raw) > 262144+64 || !meter.event(providerHashEvent, uint64(len(raw)))) {
		return [32]byte{}, false
	}
	return sha256.Sum256(raw), true
}

func replayTextSHA256Observed(text string, meter *providerWorkMeter) ([32]byte, bool) {
	if meter != nil && (len(text) > 262144 || !meter.event(providerCopyEvent, uint64(len(text)))) {
		return [32]byte{}, false
	}
	return replaySHA256Observed([]byte(text), meter)
}

func replayAddressTopicObserved(text string, meter *providerWorkMeter) (common.Hash, bool) {
	if meter == nil {
		return common.BytesToHash(common.HexToAddress(text).Bytes()), true
	}
	address, ok := replayCommandAddressObserved(text, meter)
	if !ok || !meter.event(providerCopyEvent, 32) {
		return common.Hash{}, false
	}
	var topic common.Hash
	if !meter.event(providerCopyEvent, 20) {
		return common.Hash{}, false
	}
	copy(topic[12:], address[:])
	return topic, true
}

func replayDigestObserved(raw []byte, meter *providerWorkMeter) (string, bool) {
	if meter == nil {
		return replayDigest(raw), true
	}
	hash, ok := replaySHA256Observed(raw, meter)
	if !ok {
		return "", false
	}
	return replayGeneratedBase64Observed(hash[:], meter)
}

// The actual decoded create command supplies a closed builtin JSON graph.
// Its exact adopted domain and raw bytes are hashed without an append-growth
// buffer. No producer digest, claim admission or successful partial result is
// accepted here; the event reader must still check caller and prior indexes.
func replayClaimObserved(registry string, c replayCommand, meter *providerWorkMeter) ([32]byte, bool) {
	if meter == nil {
		return replayClaim(registry, c), true
	}
	if c.op != "create" || len(c.agent) > 64 || !meter.event(providerCompareEvent, uint64(len(c.agent))) || !rpcAgent(c.agent) || !providerReplayTextEqual(registry, "eip155:11155111:"+SepoliaRegistryAddress, meter) {
		return [32]byte{}, false
	}
	args := replayCommandObjectObserved(c.args, []string{"controller", "keys", "services", "salt"}, meter)
	if args == nil || !providerReplayTextEqual(contentString(args["controller"]), c.actor, meter) {
		return [32]byte{}, false
	}
	value, ok := replayGeneratedObjectObserved([]string{"registryId", "agentId", "controller", "keys", "services", "salt"}, []any{registry, c.agent, args["controller"], args["keys"], args["services"], args["salt"]}, meter)
	if !ok {
		return [32]byte{}, false
	}
	raw, ok := replayGeneratedMarshalObserved(value, meter)
	if !ok || len(raw) > 262144 {
		return [32]byte{}, false
	}
	const domain = "sage-claim-0.11.0\x00"
	size := len(domain) + len(raw)
	if !meter.event(providerCopyEvent, uint64(size)) {
		return [32]byte{}, false
	}
	input := make([]byte, size)
	if !meter.event(providerCopyEvent, uint64(len(domain))) {
		return [32]byte{}, false
	}
	copy(input, domain)
	if !meter.event(providerCopyEvent, uint64(len(raw))) {
		return [32]byte{}, false
	}
	copy(input[len(domain):], raw)
	return replaySHA256Observed(input, meter)
}

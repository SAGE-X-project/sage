package registry011

import (
	"bytes"
	"encoding/json"
	"math/big"
	"strconv"
	"strings"
)

// These are concrete bounded JSON/scalar primitives selected by the original
// reader. A nil observer retains the existing DATA API. They do not qualify
// typed transaction/signature/hash internals, an RPC, a Source or a complete
// semantic execution. Those remain separate parts of the closed collector.
func nodeObjectObserved(raw json.RawMessage, allowed string, meter *providerWorkMeter) (map[string]json.RawMessage, bool) {
	if meter == nil {
		return nodeObject(raw, allowed)
	}
	if len(raw) > 64<<20 || !meter.event(providerCompareEvent, uint64(len(raw))) {
		return nil, false
	}
	raw = bytes.TrimSpace(raw)
	var fields map[string]json.RawMessage
	if len(raw) == 0 || raw[0] != '{' || providerReplayUnmarshal(raw, &fields, meter) != nil || fields == nil {
		return nil, false
	}
	for key := range fields {
		known := false
		for _, name := range strings.Fields(allowed) {
			if !meter.event(providerCompareEvent, uint64(len(key)+len(name))) {
				return nil, false
			}
			if key == name {
				known = true
				break
			}
		}
		if !known {
			return nil, false
		}
	}
	return fields, true
}

func nodeStringObserved(raw json.RawMessage, meter *providerWorkMeter) (string, bool) {
	if meter == nil {
		return nodeString(raw)
	}
	if len(raw) > 64<<20 || !meter.event(providerCompareEvent, uint64(len(raw))) {
		return "", false
	}
	raw = bytes.TrimSpace(raw)
	var text string
	if len(raw) == 0 || raw[0] != '"' || providerReplayUnmarshal(raw, &text, meter) != nil {
		return "", false
	}
	return text, true
}

func nodeDataObserved(raw json.RawMessage, meter *providerWorkMeter) ([]byte, bool) {
	if meter == nil {
		return nodeData(raw)
	}
	if len(raw) > 64<<20 {
		return nil, false
	}
	// The existing closed hex primitive already decodes the JSON string and
	// rejects non-hex/upper-case bytes. Do not decode the same string twice.
	value, err := rpcDataObserved(raw, meter)
	return value, err == nil
}

func nodeIndexObserved(raw json.RawMessage, meter *providerWorkMeter) (uint64, bool) {
	if meter == nil {
		return nodeIndex(raw)
	}
	text, ok := nodeStringObserved(raw, meter)
	if ok {
		return rpcQuantityObserved(text, meter)
	}
	// A failed original observer must never be retried as the decimal form.
	if len(raw) > 20 || !meter.event(providerCopyEvent, uint64(len(raw))) || !meter.event(providerCodecEvent, uint64(len(raw))) {
		return 0, false
	}
	text = string(raw)
	n, err := strconv.ParseUint(text, 10, 64)
	if err != nil || !meter.preflight(providerEncodeEvent, 20) {
		return 0, false
	}
	canonical := strconv.FormatUint(n, 10)
	if !meter.event(providerEncodeEvent, uint64(len(canonical))) {
		return 0, false
	}
	return n, providerReplayTextEqual(canonical, text, meter)
}

func nodeValueObserved(text string, meter *providerWorkMeter) (*big.Int, bool) {
	if meter == nil {
		return replayValue(text)
	}
	if len(text) < 3 || len(text) > 66 || !meter.event(providerCompareEvent, uint64(len(text))) || !meter.preflight(providerCodecEvent, uint64(len(text)-2)) {
		return nil, false
	}
	value, ok := replayValue(text)
	if !ok || !meter.event(providerCodecEvent, uint64(len(text)-2)) {
		return nil, false
	}
	return value, true
}

func nodeLiteralObserved(raw json.RawMessage, literal string, meter *providerWorkMeter) bool {
	if meter != nil && !meter.event(providerCompareEvent, uint64(len(raw))) {
		return false
	}
	return providerReplayEqual(bytes.TrimSpace(raw), []byte(literal), meter)
}

func nodeSignedGasObserved(raw json.RawMessage, meter *providerWorkMeter) bool {
	if meter == nil {
		return nodeSignedGas(raw)
	}
	text, ok := nodeStringObserved(raw, meter)
	if !ok || !meter.event(providerCompareEvent, uint64(len(text))) {
		return false
	}
	negative := strings.HasPrefix(text, "-")
	if negative {
		text = text[1:]
	}
	v, valid := rpcQuantityObserved(text, meter)
	return valid && ((!negative && v < 1<<63) || (negative && v > 0 && v <= 1<<63))
}

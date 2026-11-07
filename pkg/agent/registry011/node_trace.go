package registry011

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/ethereum/go-ethereum/common"
)

// DecodeCallTracer translates the supported Geth-style callTracer result, with
// onlyTopCall=false and withLog=true, into replay frames. It does not authenticate
// the caller's acquisition options or the node. A Source must retain the raw
// result and bind the complete tree to signed transactions and receipt logs.
// Unknown frame fields and unsupported opcodes fail closed. Supported dimensional
// diagnostics are validated but do not establish execution or gas authority.
func DecodeCallTracer(ctx context.Context, raw json.RawMessage, limits ReplayLimits) (ReplayFrame, error) {
	return decodeCallTracerObserved(ctx, raw, limits, nil)
}

// This adds only JSON/frame-normalization observation. It does not turn a
// parsed tree into a transaction-bound execution or an operating credential.
func decodeCallTracerObserved(ctx context.Context, raw json.RawMessage, limits ReplayLimits, observer *providerWorkMeter) (ReplayFrame, error) {
	b := replayBudget{limits: limits, observer: observer}
	return decodeCallTracerBudgetObserved(ctx, raw, &b)
}

// A collector uses a separate normalization budget seeded from its ORIGINAL
// semantic window, then carries these frame/log counts across all of one ready
// block's roots. It never passes the semantic execution budget itself: walking
// the normalized trees still consumes that original budget independently.
// This private DATA decoder neither authenticates the raw source nor supplies
// a collection capability. Standalone callers retain a fresh per-trace budget.
func decodeCallTracerBudgetObserved(ctx context.Context, raw json.RawMessage, b *replayBudget) (ReplayFrame, error) {
	if ctx == nil || b == nil || !replayLimits(b.limits) ||
		b.frames < 0 || b.frames > b.limits.MaxFrames || b.traceLogs < 0 || b.traceLogs > b.limits.MaxLogs {
		return ReplayFrame{}, ErrConfiguration
	}
	limits, observer := b.limits, b.observer
	if err := ctx.Err(); err != nil {
		return ReplayFrame{}, err
	}
	if len(raw) == 0 || len(raw) > limits.MaxEvidenceBytes ||
		observer != nil && !observer.event(providerCompareEvent, uint64(len(raw))) || !utf8.Valid(raw) ||
		observer != nil && !observer.event(providerCodecEvent, uint64(len(raw))) || !json.Valid(raw) ||
		observer != nil && !observer.event(providerCompareEvent, uint64(len(raw))) || !validRPCUnicode(raw) ||
		observer != nil && !observer.event(providerCodecEvent, uint64(len(raw))) {
		return ReplayFrame{}, ErrReplayUnavailable
	}
	// Check duplicates before unmarshalling can discard them. Each call adds an
	// object and an array level; the actual frame-depth limit remains 32 below.
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if !readRPCValue(d, 0, 128) {
		return ReplayFrame{}, ErrReplayUnavailable
	}
	if _, err := d.Token(); err != io.EOF {
		return ReplayFrame{}, ErrReplayUnavailable
	}
	f, err := nodeFrame(ctx, raw, 0, b, false)
	if err != nil {
		return ReplayFrame{}, err
	}
	return f, nil
}

func nodeFrame(ctx context.Context, raw json.RawMessage, depth int, b *replayBudget, erigon bool) (ReplayFrame, error) {
	var f ReplayFrame
	if err := ctx.Err(); err != nil {
		return f, err
	}
	if depth > 32 || b.frames >= b.limits.MaxFrames || !b.frameObserved() {
		return f, ErrReplayUnavailable
	}
	b.frames++
	observer := b.observer
	m, ok := nodeJSONFieldsObserved(raw, "type from to gas gasUsed input output error revertReason calls logs value executionGasUsed regularGasUsed stateGasUsed gasRefund stateGasReservoir", "type from to gas gasUsed input", observer)
	if !ok {
		return f, ErrReplayUnavailable
	}
	f.Type, ok = nodeStringObserved(m["type"], observer)
	if !ok || (f.Type != "CALL" && f.Type != "STATICCALL" && f.Type != "DELEGATECALL" && f.Type != "CALLCODE" && f.Type != "CREATE" && f.Type != "CREATE2") {
		return f, ErrReplayUnavailable
	}
	f.From, ok = nodeStringObserved(m["from"], observer)
	if !ok || !contentAddress(f.From) {
		return f, ErrReplayUnavailable
	}
	f.To, ok = nodeStringObserved(m["to"], observer)
	if !ok || !contentAddress(f.To) {
		// In particular, a failed CREATE without a destination cannot be
		// represented by inventing a zero-address execution context.
		return f, ErrReplayUnavailable
	}
	_, execution := m["executionGasUsed"]
	_, regular := m["regularGasUsed"]
	_, state := m["stateGasUsed"]
	_, refund := m["gasRefund"]
	_, reservoir := m["stateGasReservoir"]
	if depth == 0 {
		// This narrow root shape selects the pinned Erigon child diagnostics.
		// A client-version string never selects a decoder or grants trust.
		erigon = reservoir && regular && state && refund && !execution
	}
	// Known dimensional diagnostics are optional PER FRAME, but are one whole
	// trio when present. The pinned Erigon root instead supplies the trio once,
	// while its children supply only signed net state usage and an optional
	// unsigned reservoir. Never mix these two frame contracts.
	if reservoir && !erigon || erigon && depth > 0 && (!state || execution || regular || refund) ||
		(!erigon || depth == 0) && (execution && regular || (execution || regular) != state || state != refund) {
		return f, ErrReplayUnavailable
	}
	for _, k := range []string{"gas", "gasUsed", "executionGasUsed", "regularGasUsed", "stateGasUsed", "gasRefund", "stateGasReservoir"} {
		v, present := m[k]
		if !present && k != "gas" && k != "gasUsed" {
			continue
		}
		if erigon && depth > 0 && k == "stateGasUsed" {
			if !nodeSignedGasObserved(v, observer) {
				return f, ErrReplayUnavailable
			}
			continue
		}
		s, valid := nodeStringObserved(v, observer)
		if _, validQuantity := rpcQuantityObserved(s, observer); !valid || !validQuantity {
			return f, ErrReplayUnavailable
		}
	}
	f.Input, ok = nodeDataObserved(m["input"], observer)
	if !ok {
		return f, ErrReplayUnavailable
	}
	if output, present := m["output"]; present {
		if _, ok := nodeDataObserved(output, observer); !ok {
			return f, ErrReplayUnavailable
		}
	}
	f.Value = "0x0"
	if v, present := m["value"]; present {
		f.Value, ok = nodeStringObserved(v, observer)
		if _, valid := nodeValueObserved(f.Value, observer); !ok || !valid {
			return f, ErrReplayUnavailable
		}
	} else if f.Type != "STATICCALL" {
		return f, ErrReplayUnavailable
	}
	for _, key := range []string{"error", "revertReason"} {
		if v, present := m[key]; present {
			s, valid := nodeStringObserved(v, observer)
			if !valid || len(s) > 4096 {
				return f, ErrReplayUnavailable
			}
			if key == "error" {
				f.Error = s
			}
		}
	}
	var calls, logs []json.RawMessage
	if !nodeArrayObserved(m, "calls", &calls, b.limits.MaxFrames-b.frames, observer) || !nodeArrayObserved(m, "logs", &logs, b.limits.MaxLogs-b.traceLogs, observer) {
		return f, ErrReplayUnavailable
	}
	b.traceLogs += len(logs)
	if observer != nil && !nodeFrameStorageObserved(&f, len(calls), len(logs), observer) {
		return ReplayFrame{}, ErrReplayUnavailable
	}
	for index, call := range calls {
		child, err := nodeFrame(ctx, call, depth+1, b, erigon)
		if err != nil {
			return ReplayFrame{}, err
		}
		if observer == nil {
			f.Calls = append(f.Calls, child)
		} else if !nodeFrameCopyObserved(&f, index, child, observer) {
			return ReplayFrame{}, ErrReplayUnavailable
		}
	}
	for index, rawLog := range logs {
		if !b.logsObserved(1) {
			return ReplayFrame{}, ErrReplayUnavailable
		}
		l, valid := nodeLogObserved(rawLog, len(calls), observer)
		if !valid {
			return ReplayFrame{}, ErrReplayUnavailable
		}
		if observer == nil {
			f.Logs = append(f.Logs, l)
		} else if !nodeFrameLogCopyObserved(&f, index, l, observer) {
			return ReplayFrame{}, ErrReplayUnavailable
		}
	}
	return f, nil
}

// The pinned Erigon diagnostic is signed int64 hexadecimal, including -2^63.
// Negative zero, aliases and values outside that exact range refuse.
func nodeSignedGas(raw json.RawMessage) bool {
	s, ok := nodeString(raw)
	if !ok {
		return false
	}
	negative := strings.HasPrefix(s, "-")
	if negative {
		s = s[1:]
	}
	v, valid := rpcQuantity(s)
	return valid && ((!negative && v < 1<<63) || (negative && v > 0 && v <= 1<<63))
}

func nodeLog(raw json.RawMessage, children int) (ReplayFrameLog, bool) {
	return nodeLogObserved(raw, children, nil)
}

func nodeLogObserved(raw json.RawMessage, children int, observer *providerWorkMeter) (ReplayFrameLog, bool) {
	var l ReplayFrameLog
	m, ok := nodeObjectObserved(raw, "address topics data position index", observer)
	if !ok {
		return l, false
	}
	l.Address, ok = nodeStringObserved(m["address"], observer)
	if !ok || !contentAddress(l.Address) {
		return l, false
	}
	position, ok := nodeIndexObserved(m["position"], observer)
	// The acquisition permits at most 65,536 frames. Check this absolute
	// bound before converting even on a 32-bit consumer.
	if !ok || position > 65536 || position > uint64(children) {
		return l, false
	}
	l.AfterChildren = int(position)
	if v, present := m["index"]; present {
		if _, ok := nodeIndexObserved(v, observer); !ok {
			return l, false
		}
	}
	l.Data, ok = nodeDataObserved(m["data"], observer)
	if !ok {
		return l, false
	}
	var topics []json.RawMessage
	if !nodeArrayObserved(m, "topics", &topics, 4, observer) || m["topics"] == nil {
		return l, false
	}
	if observer != nil && len(topics) != 0 {
		if !observer.event(providerCopyEvent, uint64(len(topics))*32) {
			return l, false
		}
		l.Topics = make([]common.Hash, len(topics))
	}
	for index, v := range topics {
		s, valid := nodeStringObserved(v, observer)
		if !valid || !lowerHash(s) {
			return l, false
		}
		if observer == nil {
			l.Topics = append(l.Topics, common.HexToHash(s))
		} else {
			if !observer.event(providerCodecEvent, 64) || !observer.event(providerCopyEvent, 32) {
				return l, false
			}
			l.Topics[index] = common.HexToHash(s)
		}
	}
	return l, true
}

func nodeObject(raw json.RawMessage, allowed string) (map[string]json.RawMessage, bool) {
	raw = bytes.TrimSpace(raw)
	var m map[string]json.RawMessage
	if len(raw) == 0 || raw[0] != '{' || json.Unmarshal(raw, &m) != nil || m == nil {
		return nil, false
	}
	for key := range m {
		known := false
		for _, name := range strings.Fields(allowed) {
			known = known || key == name
		}
		if !known {
			return nil, false
		}
	}
	return m, true
}
func nodeArray(m map[string]json.RawMessage, key string, out *[]json.RawMessage, maximum int) bool {
	v, present := m[key]
	if !present {
		return true
	}
	var ok bool
	*out, ok = nodeRawArray(v, maximum)
	return ok
}
func nodeRawArray(raw json.RawMessage, maximum int) ([]json.RawMessage, bool) {
	d := json.NewDecoder(bytes.NewReader(raw))
	t, err := d.Token()
	if err != nil || t != json.Delim('[') {
		return nil, false
	}
	out := []json.RawMessage{}
	for d.More() {
		if len(out) >= maximum {
			return nil, false
		}
		var value json.RawMessage
		if d.Decode(&value) != nil {
			return nil, false
		}
		out = append(out, value)
	}
	t, err = d.Token()
	if err != nil || t != json.Delim(']') {
		return nil, false
	}
	_, err = d.Token()
	return out, err == io.EOF
}
func nodeString(raw json.RawMessage) (string, bool) {
	raw = bytes.TrimSpace(raw)
	var s string
	if len(raw) == 0 || raw[0] != '"' || json.Unmarshal(raw, &s) != nil {
		return "", false
	}
	return s, true
}
func nodeData(raw json.RawMessage) ([]byte, bool) {
	s, ok := nodeString(raw)
	if !ok || strings.ToLower(s) != s {
		return nil, false
	}
	b, err := rpcData(raw)
	return b, err == nil
}
func nodeIndex(raw json.RawMessage) (uint64, bool) {
	if s, ok := nodeString(raw); ok {
		return rpcQuantity(s)
	}
	s := string(raw)
	n, err := strconv.ParseUint(s, 10, 64)
	return n, err == nil && strconv.FormatUint(n, 10) == s
}

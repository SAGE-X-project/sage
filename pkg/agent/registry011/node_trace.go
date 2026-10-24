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
// Unknown frame fields and unsupported opcodes fail closed. Known Reth 2.8.1
// gas accounting fields are validated but do not establish execution authority.
func DecodeCallTracer(ctx context.Context, raw json.RawMessage, limits ReplayLimits) (ReplayFrame, error) {
	if ctx == nil || !replayLimits(limits) {
		return ReplayFrame{}, ErrConfiguration
	}
	if err := ctx.Err(); err != nil {
		return ReplayFrame{}, err
	}
	if len(raw) == 0 || len(raw) > limits.MaxEvidenceBytes || !utf8.Valid(raw) || !json.Valid(raw) || !validRPCUnicode(raw) {
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
	b := replayBudget{limits: limits}
	f, err := nodeFrame(ctx, raw, 0, &b)
	if err != nil {
		return ReplayFrame{}, err
	}
	return f, nil
}

func nodeFrame(ctx context.Context, raw json.RawMessage, depth int, b *replayBudget) (ReplayFrame, error) {
	var f ReplayFrame
	if err := ctx.Err(); err != nil {
		return f, err
	}
	if depth > 32 || b.frames >= b.limits.MaxFrames {
		return f, ErrReplayUnavailable
	}
	b.frames++
	m, ok := nodeObject(raw, "type from to gas gasUsed input output error revertReason calls logs value executionGasUsed stateGasUsed gasRefund")
	if !ok {
		return f, ErrReplayUnavailable
	}
	f.Type, ok = nodeString(m["type"])
	if !ok || (f.Type != "CALL" && f.Type != "STATICCALL" && f.Type != "DELEGATECALL" && f.Type != "CALLCODE" && f.Type != "CREATE" && f.Type != "CREATE2") {
		return f, ErrReplayUnavailable
	}
	f.From, ok = nodeString(m["from"])
	if !ok || !contentAddress(f.From) {
		return f, ErrReplayUnavailable
	}
	f.To, ok = nodeString(m["to"])
	if !ok || !contentAddress(f.To) {
		// In particular, a failed CREATE without a destination cannot be
		// represented by inventing a zero-address execution context.
		return f, ErrReplayUnavailable
	}
	for _, k := range []string{"gas", "gasUsed", "executionGasUsed", "stateGasUsed", "gasRefund"} {
		v, present := m[k]
		if !present && k != "gas" && k != "gasUsed" {
			continue
		}
		s, valid := nodeString(v)
		if _, validQuantity := rpcQuantity(s); !valid || !validQuantity {
			return f, ErrReplayUnavailable
		}
	}
	f.Input, ok = nodeData(m["input"])
	if !ok {
		return f, ErrReplayUnavailable
	}
	if output, present := m["output"]; present {
		if _, ok := nodeData(output); !ok {
			return f, ErrReplayUnavailable
		}
	}
	f.Value = "0x0"
	if v, present := m["value"]; present {
		f.Value, ok = nodeString(v)
		if _, valid := replayValue(f.Value); !ok || !valid {
			return f, ErrReplayUnavailable
		}
	} else if f.Type != "STATICCALL" {
		return f, ErrReplayUnavailable
	}
	for _, key := range []string{"error", "revertReason"} {
		if v, present := m[key]; present {
			s, valid := nodeString(v)
			if !valid || len(s) > 4096 {
				return f, ErrReplayUnavailable
			}
			if key == "error" {
				f.Error = s
			}
		}
	}
	var calls, logs []json.RawMessage
	if !nodeArray(m, "calls", &calls, b.limits.MaxFrames-b.frames) || !nodeArray(m, "logs", &logs, b.limits.MaxLogs-b.traceLogs) {
		return f, ErrReplayUnavailable
	}
	b.traceLogs += len(logs)
	for _, call := range calls {
		child, err := nodeFrame(ctx, call, depth+1, b)
		if err != nil {
			return ReplayFrame{}, err
		}
		f.Calls = append(f.Calls, child)
	}
	for _, rawLog := range logs {
		l, valid := nodeLog(rawLog, len(calls))
		if !valid {
			return ReplayFrame{}, ErrReplayUnavailable
		}
		f.Logs = append(f.Logs, l)
	}
	return f, nil
}

func nodeLog(raw json.RawMessage, children int) (ReplayFrameLog, bool) {
	var l ReplayFrameLog
	m, ok := nodeObject(raw, "address topics data position index")
	if !ok {
		return l, false
	}
	l.Address, ok = nodeString(m["address"])
	if !ok || !contentAddress(l.Address) {
		return l, false
	}
	position, ok := nodeIndex(m["position"])
	if !ok || position > uint64(children) {
		return l, false
	}
	l.AfterChildren = int(position)
	if v, present := m["index"]; present {
		if _, ok := nodeIndex(v); !ok {
			return l, false
		}
	}
	l.Data, ok = nodeData(m["data"])
	if !ok {
		return l, false
	}
	var topics []json.RawMessage
	if !nodeArray(m, "topics", &topics, 4) || m["topics"] == nil {
		return l, false
	}
	for _, v := range topics {
		s, valid := nodeString(v)
		if !valid || !lowerHash(s) {
			return l, false
		}
		l.Topics = append(l.Topics, common.HexToHash(s))
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

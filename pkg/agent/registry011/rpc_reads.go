package registry011

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"

	"golang.org/x/crypto/sha3"
)

// PinnedAgentReads fixes the Registry, agent and provider-reported finalized
// block. Byte collection is not enrollment, command replay, consensus proof,
// authenticated absence, durable publication or permission for protected use.
type PinnedAgentReads struct {
	rpc             *ReadRPC
	registry, agent string
	header          FinalizedHeader
}

func (c *ReadRPC) PinAgentReads(ctx context.Context, registry, agent string) (*PinnedAgentReads, error) {
	if c == nil || c.client == nil || ctx == nil || registry != SepoliaRegistryAddress || !rpcAgent(agent) {
		return nil, ErrConfiguration
	}
	chain, err := c.call(ctx, "eth_chainId", []any{})
	if err != nil {
		return nil, err
	}
	var chainID string
	if json.Unmarshal(chain, &chainID) != nil || chainID != "0xaa36a7" {
		return nil, ErrRPCState
	}
	genesis, err := c.header(ctx, "0x0")
	if err != nil {
		return nil, err
	}
	if genesis.Number != 0 || genesis.Hash != sepoliaGenesis {
		return nil, ErrRPCState
	}
	header, err := c.header(ctx, "finalized")
	if err != nil {
		return nil, err
	}
	reads := &PinnedAgentReads{c, registry, agent, header}
	result, err := reads.read(ctx, "registryId()", "", 0)
	if err != nil {
		return nil, err
	}
	values, _, err := rpcABI(result, false, 128)
	if err != nil || string(values[0]) != "eip155:11155111:"+registry {
		return nil, ErrRPCState
	}
	return reads, nil
}

func (c *ReadRPC) header(ctx context.Context, tag string) (FinalizedHeader, error) {
	result, err := c.call(ctx, "eth_getBlockByNumber", []any{tag, false})
	if err != nil {
		return FinalizedHeader{}, err
	}
	var value struct {
		Hash       string
		ParentHash string
		Number     string
		Timestamp  string
	}
	if json.Unmarshal(result, &value) != nil {
		return FinalizedHeader{}, ErrRPCState
	}
	number, ok := rpcQuantity(value.Number)
	stamp, valid := rpcQuantity(value.Timestamp)
	h := FinalizedHeader{value.Hash, value.ParentHash, number, stamp}
	if !ok || !valid || !validHeader(h) {
		return FinalizedHeader{}, ErrRPCState
	}
	return h, nil
}

func (r *PinnedAgentReads) Header() (FinalizedHeader, error) {
	if r == nil || r.rpc == nil {
		return FinalizedHeader{}, ErrConfiguration
	}
	return r.header, nil
}

// CodeBytes collects exact runtime bytes for a required dependency. The caller
// must compare every required code to the separately qualified manifest.
func (r *PinnedAgentReads) CodeBytes(ctx context.Context, address string) ([]byte, error) {
	if r == nil || r.rpc == nil || !rpcAddress(address) {
		return nil, ErrConfiguration
	}
	result, err := r.rpc.call(ctx, "eth_getCode", []any{address, r.block()})
	if err != nil {
		return nil, err
	}
	return rpcData(result)
}

// RecordBytes returns only a present bit and ABI-checked bytes. A false bit is
// not authenticated absence; durable positive/terminal history must be checked.
func (r *PinnedAgentReads) RecordBytes(ctx context.Context) (bool, []byte, error) {
	result, err := r.read(ctx, "record(string)", r.agentID(), 0)
	if err != nil {
		return false, nil, err
	}
	values, present, err := rpcABI(result, true, 65536)
	if err != nil || present != (len(values[0]) > 0) {
		return false, nil, ErrRPCState
	}
	return present, values[0], nil
}

func (r *PinnedAgentReads) GrantsBytes(ctx context.Context) ([]byte, error) {
	result, err := r.read(ctx, "activeGrants(string)", r.agentID(), 0)
	if err != nil {
		return nil, err
	}
	values, _, err := rpcABI(result, false, 262144)
	if err != nil || len(values[0]) == 0 {
		return nil, ErrRPCState
	}
	return values[0], nil
}

// HistoryBytes are copied retained entry/command bytes at one requested version.
// Their JCS, digests, transitions, actual caller and execution are NOT verified.
type HistoryBytes struct {
	Version        uint64
	Entry, Command []byte
}

// HistoryReadLimits requires explicit budgets, up to 4096 items and 8 MiB of
// combined entry/command bytes per range. These are ceilings, not defaults.
type HistoryReadLimits struct {
	MaxItems uint64
	MaxBytes int
}

// HistoryRange collects (after,current] with no gaps or partial-success return.
// Its endpoints must come from separately validated record/durable state. A
// collected list is not proof that current is the true final record version.
func (r *PinnedAgentReads) HistoryRange(ctx context.Context, after, current uint64, limits HistoryReadLimits) ([]HistoryBytes, error) {
	if r == nil || r.rpc == nil || ctx == nil || limits.MaxItems == 0 || limits.MaxItems > 4096 || limits.MaxBytes <= 0 || limits.MaxBytes > 8<<20 {
		return nil, ErrConfiguration
	}
	if current < after || current-after > limits.MaxItems {
		return nil, ErrRPCState
	}
	rows := []HistoryBytes{}
	remaining := limits.MaxBytes
	for version := after; version < current; {
		version++ // Cannot wrap: increment only while strictly below current.
		result, err := r.read(ctx, "historyItem(string,uint64)", r.agent, version)
		if err != nil {
			return nil, err
		}
		values, _, err := rpcABI(result, false, 262144, 262144)
		if err != nil || len(values[0]) == 0 || len(values[1]) == 0 || len(values[0]) > remaining {
			return nil, ErrRPCState
		}
		remaining -= len(values[0])
		if len(values[1]) > remaining {
			return nil, ErrRPCState
		}
		remaining -= len(values[1])
		rows = append(rows, HistoryBytes{version, values[0], values[1]})
	}
	return rows, nil
}

func (r *PinnedAgentReads) agentID() string {
	if r == nil {
		return ""
	}
	return r.agent
}

func (r *PinnedAgentReads) block() map[string]any {
	return map[string]any{"blockHash": r.header.Hash, "requireCanonical": true}
}

func (r *PinnedAgentReads) read(ctx context.Context, signature, agent string, version uint64) (json.RawMessage, error) {
	if r == nil || r.rpc == nil {
		return nil, ErrConfiguration
	}
	hash := sha3.NewLegacyKeccak256()
	_, _ = hash.Write([]byte(signature))
	data := append([]byte(nil), hash.Sum(nil)[:4]...)
	if agent != "" {
		head := 32
		if version != 0 {
			head = 64
		}
		data = append(data, rpcWord(uint64(head))...)
		if version != 0 {
			data = append(data, rpcWord(version)...)
		}
		data = append(data, rpcWord(uint64(len(agent)))...)
		data = append(data, agent...)
		data = append(data, make([]byte, (32-len(agent)%32)%32)...)
	}
	return r.rpc.call(ctx, "eth_call", []any{map[string]string{"to": r.registry, "data": "0x" + hex.EncodeToString(data)}, r.block()})
}

func rpcWord(value uint64) []byte {
	word := make([]byte, 32)
	binary.BigEndian.PutUint64(word[24:], value)
	return word
}

// rpcABI accepts only the exact canonical bool/bytes and bytes tuple forms
// returned by the four pinned Registry reads. All ranges are checked before
// slicing/copying; offsets must follow ABI declaration order with no gaps.
func rpcABI(result json.RawMessage, hasBool bool, maxima ...int) ([][]byte, bool, error) {
	data, err := rpcData(result)
	fields := len(maxima)
	if hasBool {
		fields++
	}
	if err != nil || len(data) < fields*32 || len(data)%32 != 0 {
		return nil, false, ErrRPCState
	}
	present := false
	index := 0
	if hasBool {
		if !bytes.Equal(data[:31], make([]byte, 31)) || data[31] > 1 {
			return nil, false, ErrRPCState
		}
		present = data[31] == 1
		index = 32
	}
	cursor := fields * 32
	values := make([][]byte, 0, len(maxima))
	for _, maximum := range maxima {
		offset, ok := rpcUint(data[index : index+32])
		index += 32
		if !ok || offset != uint64(cursor) || len(data)-cursor < 32 {
			return nil, false, ErrRPCState
		}
		length, ok := rpcUint(data[cursor : cursor+32])
		cursor += 32
		if !ok || length > uint64(maximum) || length > uint64(len(data)-cursor) {
			return nil, false, ErrRPCState
		}
		size := int(length)
		padded := size + (32-size%32)%32
		if padded > len(data)-cursor || !bytes.Equal(data[cursor+size:cursor+padded], make([]byte, padded-size)) {
			return nil, false, ErrRPCState
		}
		values = append(values, append([]byte(nil), data[cursor:cursor+size]...))
		cursor += padded
	}
	if cursor != len(data) {
		return nil, false, ErrRPCState
	}
	return values, present, nil
}

func rpcUint(word []byte) (uint64, bool) {
	return binary.BigEndian.Uint64(word[24:]), bytes.Equal(word[:24], make([]byte, 24))
}

func rpcData(raw json.RawMessage) ([]byte, error) {
	var text string
	if json.Unmarshal(raw, &text) != nil || len(text) < 2 || text[:2] != "0x" || len(text)%2 != 0 || strings.ToLower(text) != text {
		return nil, ErrRPCState
	}
	data, err := hex.DecodeString(text[2:])
	if err != nil {
		return nil, ErrRPCState
	}
	return data, nil
}

func rpcQuantity(text string) (uint64, bool) {
	if len(text) < 3 || text[:2] != "0x" || (len(text) > 3 && text[2] == '0') || strings.ToLower(text) != text {
		return 0, false
	}
	value, err := strconv.ParseUint(text[2:], 16, 64)
	return value, err == nil && strconv.FormatUint(value, 16) == text[2:]
}
func rpcAddress(text string) bool {
	return len(text) == 42 && lowerHash("0x"+strings.Repeat("0", 24)+strings.TrimPrefix(text, "0x")) && text != "0x"+strings.Repeat("0", 40)
}
func rpcAgent(text string) bool {
	if len(text) < 1 || len(text) > 64 || text == "." || text == ".." {
		return false
	}
	for _, b := range []byte(text) {
		if (b < 'a' || b > 'z') && (b < 'A' || b > 'Z') && (b < '0' || b > '9') && b != '.' && b != '_' && b != '-' {
			return false
		}
	}
	return true
}

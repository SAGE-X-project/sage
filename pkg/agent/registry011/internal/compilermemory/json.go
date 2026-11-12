package compilermemory

import (
	"encoding/binary"
	"errors"
	"math"
)

var (
	ErrJSON     = errors.New("compiler memory: invalid JSON document")
	ErrJSONType = errors.New("compiler memory: unexpected JSON value type")
)

// JSONKind identifies a value without materializing a Go object tree.
type JSONKind uint64

const (
	JSONObject JSONKind = iota + 1
	JSONArray
	JSONString
	JSONNumber
	JSONBoolean
	JSONNull
	jsonObjectKey
)

const jsonNodeBytes = 40
const jsonFrameBytes = 16

// JSONLimits bound the actual native node table and parser stack. They are
// caller-supplied DATA, not resource admission for an entire compiler.
type JSONLimits struct {
	MaximumDepth int
	MaximumNodes int
}

// JSONDocument retains only original-owner spans and a bounded record count.
// Variable-sized records and parsing frames occupy the actual Region. The Go
// wrappers/runtime and the rest of the preparation compiler remain separate.
type JSONDocument struct {
	owner *Region
	input Span
	nodes Span
	count int
}

// JSONValue is a checked index into one original document. A copied handle
// cannot extend the Region's lifetime or reconstruct an execution permission.
type JSONValue struct {
	document JSONDocument
	index    int
}

func (*JSONDocument) UnmarshalJSON([]byte) error { return ErrConfiguration }
func (*JSONValue) UnmarshalJSON([]byte) error    { return ErrConfiguration }

type jsonRecord struct {
	kind   JSONKind
	start  uint64
	length uint64
	parent uint64
	end    uint64
}

func readJSONRecord(raw []byte) jsonRecord {
	return jsonRecord{
		kind: JSONKind(binary.LittleEndian.Uint64(raw)), start: binary.LittleEndian.Uint64(raw[8:]),
		length: binary.LittleEndian.Uint64(raw[16:]), parent: binary.LittleEndian.Uint64(raw[24:]),
		end: binary.LittleEndian.Uint64(raw[32:]),
	}
}

func writeJSONRecord(raw []byte, node jsonRecord) {
	binary.LittleEndian.PutUint64(raw, uint64(node.kind))
	binary.LittleEndian.PutUint64(raw[8:], node.start)
	binary.LittleEndian.PutUint64(raw[16:], node.length)
	binary.LittleEndian.PutUint64(raw[24:], node.parent)
	binary.LittleEndian.PutUint64(raw[32:], node.end)
}

func jsonOffset(value uint64, maximum int) (int, error) {
	if maximum < 0 || value > math.MaxInt || value > uint64(maximum) {
		return 0, ErrJSON
	}
	return int(value), nil
}

func (d JSONDocument) tokenSpan(node jsonRecord) (Span, error) {
	start, err := jsonOffset(node.start, d.input.length)
	if err != nil {
		return Span{}, err
	}
	length, err := jsonOffset(node.length, d.input.length-start)
	if err != nil {
		return Span{}, err
	}
	return Span{owner: d.owner, offset: d.input.offset + start, length: length}, nil
}

// ParseJSON builds a native byte/offset tree and seals this Region. It rejects
// duplicate decoded keys, invalid UTF-8, trailing DATA and exhausted bounds.
// Once parsing enters its reserved table, failure makes this Region unusable;
// the caller must still Close it under the original cleanup lifetime.
func (r *Region) ParseJSON(input Span, limits JSONLimits) (JSONDocument, error) {
	if r == nil || r.self != r {
		return JSONDocument{}, ErrConfiguration
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.accessError(); err != nil {
		return JSONDocument{}, err
	}
	if !r.contains(input) || r.sealed {
		return JSONDocument{}, ErrRange
	}
	if limits.MaximumDepth <= 0 || limits.MaximumDepth > 64 || limits.MaximumNodes <= 0 {
		return JSONDocument{}, ErrConfiguration
	}
	frames := limits.MaximumDepth * jsonFrameBytes
	remaining := r.capacity - r.used
	if frames > remaining || limits.MaximumNodes > (remaining-frames)/jsonNodeBytes {
		return JSONDocument{}, ErrRange
	}
	tableBytes := limits.MaximumNodes * jsonNodeBytes
	table := Span{owner: r, offset: r.used, length: tableBytes}
	frameStart := r.used + tableBytes
	r.used, r.sealed = frameStart+frames, true
	p := jsonBuilder{
		region: r, input: r.data[input.offset : input.offset+input.length],
		table: r.data[table.offset:frameStart], frames: r.data[frameStart:r.used],
		nodeCap: limits.MaximumNodes, depthCap: limits.MaximumDepth,
	}
	err := p.parse()
	if contextErr := r.actionError(); contextErr != nil {
		err = contextErr
	}
	if err != nil {
		r.usable = false
		return JSONDocument{}, err
	}
	table.length = p.count * jsonNodeBytes
	return JSONDocument{owner: r, input: input, nodes: table, count: p.count}, nil
}

func (d JSONDocument) Root() JSONValue { return JSONValue{document: d} }

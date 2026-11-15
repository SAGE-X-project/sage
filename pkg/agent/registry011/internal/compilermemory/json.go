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

// JSONValueDepthLimits uses the preparation validator's value-depth convention:
// the root has depth zero, and an empty container at the last value depth is
// valid. This separate contract does not change JSONLimits' container bound.
type JSONValueDepthLimits struct {
	MaximumValueDepth int
	MaximumNodes      int
}

// JSONCapacity calculates storage for the raw input, records and bounded stack.
// The checked byte count describes a representation; it reserves no resources.
func JSONCapacity(inputBytes int, limits JSONLimits) (int, error) {
	return jsonCapacity(inputBytes, limits.MaximumNodes, limits.MaximumDepth, false)
}

// JSONValueCapacity uses the same root-zero depth contract as
// ParseJSONAtValueDepth, including its possible last empty container frame.
func JSONValueCapacity(inputBytes int, limits JSONValueDepthLimits) (int, error) {
	return jsonCapacity(inputBytes, limits.MaximumNodes, limits.MaximumValueDepth, true)
}

func jsonCapacity(inputBytes, nodes, depth int, valueDepth bool) (int, error) {
	if inputBytes <= 0 || nodes <= 0 || depth < 0 || depth > 64 || (!valueDepth && depth == 0) {
		return 0, ErrConfiguration
	}
	if valueDepth {
		depth++
	}
	frames := depth * jsonFrameBytes
	if inputBytes > math.MaxInt-frames || nodes > (math.MaxInt-frames-inputBytes)/jsonNodeBytes {
		return 0, ErrRange
	}
	return inputBytes + nodes*jsonNodeBytes + frames, nil
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
	return r.parseJSON(input, limits.MaximumDepth, limits.MaximumNodes, false)
}

// ParseJSONAtValueDepth preserves the existing preparation policy's root-zero
// value-depth limit. It uses the same grammar, record table and lifetime rules.
func (r *Region) ParseJSONAtValueDepth(input Span, limits JSONValueDepthLimits) (JSONDocument, error) {
	return r.parseJSON(input, limits.MaximumValueDepth, limits.MaximumNodes, true)
}

func (r *Region) parseJSON(input Span, depth, maximumNodes int, valueDepth bool) (JSONDocument, error) {
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
	if depth < 0 || depth > 64 || (!valueDepth && depth == 0) || maximumNodes <= 0 {
		return JSONDocument{}, ErrConfiguration
	}
	frameCount := depth
	if valueDepth {
		frameCount++
	}
	frames := frameCount * jsonFrameBytes
	remaining := r.capacity - r.used
	if frames > remaining || maximumNodes > (remaining-frames)/jsonNodeBytes {
		return JSONDocument{}, ErrRange
	}
	tableBytes := maximumNodes * jsonNodeBytes
	table := Span{owner: r, offset: r.used, length: tableBytes}
	frameStart := r.used + tableBytes
	r.used, r.sealed = frameStart+frames, true
	p := jsonBuilder{
		region: r, input: r.data[input.offset : input.offset+input.length],
		table: r.data[table.offset:frameStart], frames: r.data[frameStart:r.used],
		nodeCap: maximumNodes, depthCap: frameCount,
		valueDepthLimit: depth, enforceValueDepth: valueDepth,
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

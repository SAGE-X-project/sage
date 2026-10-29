package registry011

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"strconv"
)

const providerHistoryOriginalPackDomain = "sage-provider-history-original-pack-v1"
const providerHistoryOriginalPackMagic = providerHistoryOriginalPackDomain + "\n"
const providerHistoryOriginalPackHeaderBytes = 64 << 10
const providerHistoryOriginalPackBytes = 32 << 20

// Packing is a new private history representation, not a change to the old
// object inventory. Only the small descriptor is canonical JSON. The original
// request/response/TLS bytes follow it verbatim; RPC JSON is never rewritten.
// A valid pack proves framing and content bindings only, not RPC semantics,
// TLS provenance, custody, an ACK or permission to dispatch a member.
type providerHistoryOriginalPack struct {
	stage                 uint64
	table, purpose        string
	first, count, entries int
	records               [2 * providerHistoryGroupMembers]providerHistoryOriginalReference
	raw, digest           string
}

type providerHistoryOriginalReference struct {
	kind, digest string
	offset, size int
}

type providerHistoryOriginalInput struct{ kind, raw string }

func (*providerHistoryOriginalPack) UnmarshalJSON([]byte) error { return errProviderIntegration }

func providerHistoryOriginalPackShape(stage uint64, table, purpose string, first, count int, maximum uint64) (int, bool) {
	if stage == 0 || stage > 1<<53 || !lowerHash(table) || first < 0 || count <= 0 || count > providerHistoryGroupMembers ||
		first > providerHistoryGroupMembers-count || maximum <= uint64(len(providerHistoryOriginalPackMagic)+4) || maximum > providerHistoryOriginalPackBytes {
		return 0, false
	}
	switch purpose {
	case "requests":
		return count, true
	case "results":
		return 2 * count, true
	}
	return 0, false
}

func providerHistoryOriginalKind(purpose string, index int) string {
	if purpose == "requests" {
		return "request"
	}
	if index%2 == 0 {
		return "response"
	}
	return "tls"
}

func providerHistoryOriginalSize(kind string, size uint64) bool {
	return size > 0 && size <= providerHistoryOriginalPackBytes &&
		(kind == "request" && size <= 16<<10 || kind == "response" || kind == "tls" && size <= 64<<10)
}

func newProviderHistoryOriginalPackObserved(stage uint64, table, purpose string, first, count int, inputs []providerHistoryOriginalInput, maximum uint64, meter *providerWorkMeter) (*providerHistoryOriginalPack, error) {
	entries, ok := providerHistoryOriginalPackShape(stage, table, purpose, first, count, maximum)
	if !ok || len(inputs) != entries {
		return nil, errProviderIntegration
	}
	// Bound every payload before hashing or descriptor allocation. In particular,
	// an oversized later entry cannot make an earlier entry fund a larger pack.
	payload := uint64(0)
	for i, input := range inputs {
		if input.kind != providerHistoryOriginalKind(purpose, i) || !providerHistoryOriginalSize(input.kind, uint64(len(input.raw))) ||
			!candidateCapacity(0, payload, uint64(len(input.raw)), maximum-uint64(len(providerHistoryOriginalPackMagic)+4)) {
			return nil, errProviderIntegration
		}
		payload += uint64(len(input.raw))
	}
	p := &providerHistoryOriginalPack{stage: stage, table: table, purpose: purpose, first: first, count: count, entries: entries}
	rows := make([]map[string]string, entries)
	stride := entries / count
	for i, input := range inputs {
		digest := providerReplayDigest("", providerReplayBytes(input.raw, meter), meter)
		if digest == "" {
			return nil, errProviderIntegration
		}
		p.records[i] = providerHistoryOriginalReference{kind: input.kind, digest: digest, size: len(input.raw)}
		rows[i] = map[string]string{"ordinal": strconv.Itoa(first + i/stride), "kind": input.kind, "digest": digest, "bytes": strconv.Itoa(len(input.raw))}
	}
	header := providerReplayCanonical(map[string]any{
		"format": providerHistoryOriginalPackDomain, "stage": strconv.FormatUint(stage, 10), "table": table,
		"purpose": purpose, "first": strconv.Itoa(first), "count": strconv.Itoa(count), "entries": rows,
	}, providerHistoryOriginalPackHeaderBytes, meter)
	total, sumOK := providerUpperSum(uint64(len(providerHistoryOriginalPackMagic)+4), uint64(len(header)), payload)
	if len(header) == 0 || len(header) > providerHistoryOriginalPackHeaderBytes || !sumOK || total > maximum {
		return nil, errProviderIntegration
	}
	// Each actual copy and the fixed four-byte length encoding is observed.
	// The bounded output owns all payload bytes; no input slice is retained.
	raw := make([]byte, int(total))
	if !providerHistoryOriginalCopy(raw[:len(providerHistoryOriginalPackMagic)], providerHistoryOriginalPackMagic, meter) ||
		meter != nil && !meter.event(providerEncodeEvent, 4) {
		return nil, errProviderIntegration
	}
	binary.BigEndian.PutUint32(raw[len(providerHistoryOriginalPackMagic):], uint32(len(header)))
	start := len(providerHistoryOriginalPackMagic) + 4
	if meter != nil && !meter.event(providerCopyEvent, uint64(len(header))) {
		return nil, errProviderIntegration
	}
	copy(raw[start:], header)
	start += len(header)
	for i, input := range inputs {
		p.records[i].offset = start
		if !providerHistoryOriginalCopy(raw[start:start+len(input.raw)], input.raw, meter) {
			return nil, errProviderIntegration
		}
		start += len(input.raw)
	}
	p.raw, p.digest = providerReplayString(raw, meter), providerReplayDigest(providerHistoryOriginalPackDomain, raw, meter)
	if p.raw == "" || p.digest == "" {
		return nil, errProviderIntegration
	}
	return p, nil
}

func providerHistoryOriginalCopy(dst []byte, src string, meter *providerWorkMeter) bool {
	if len(dst) != len(src) || meter != nil && !meter.event(providerCopyEvent, uint64(len(src))) {
		return false
	}
	return copy(dst, src) == len(src)
}

// The expected digest and range come from the independently selected history
// inventory/table. A descriptor does not select its own table or member range.
// Offsets are derived from consecutive lengths, never supplied by a producer.
func parseProviderHistoryOriginalPackObserved(raw []byte, expected string, stage uint64, table, purpose string, first, count int, maximum uint64, meter *providerWorkMeter) (*providerHistoryOriginalPack, error) {
	entries, ok := providerHistoryOriginalPackShape(stage, table, purpose, first, count, maximum)
	start := len(providerHistoryOriginalPackMagic) + 4
	if !ok || !lowerHash(expected) || len(raw) <= start || uint64(len(raw)) > maximum {
		return nil, errProviderIntegration
	}
	if !providerReplayTextEqual(providerReplayString(raw[:len(providerHistoryOriginalPackMagic)], meter), providerHistoryOriginalPackMagic, meter) ||
		meter != nil && !meter.event(providerCodecEvent, 4) {
		return nil, errProviderIntegration
	}
	n := uint64(binary.BigEndian.Uint32(raw[len(providerHistoryOriginalPackMagic):start]))
	if n == 0 || n > providerHistoryOriginalPackHeaderBytes || n > uint64(len(raw)-start) {
		return nil, errProviderIntegration
	}
	header := raw[start : start+int(n)]
	m := contentObject(providerReplayJSON(header, meter), "format", "stage", "table", "purpose", "first", "count", "entries")
	if m == nil || m["format"] != providerHistoryOriginalPackDomain || m["stage"] != strconv.FormatUint(stage, 10) || m["table"] != table ||
		m["purpose"] != purpose || m["first"] != strconv.Itoa(first) || m["count"] != strconv.Itoa(count) {
		return nil, errProviderIntegration
	}
	items, valid := m["entries"].([]any)
	if !valid || len(items) != entries || !providerReplayEqual(providerReplayCanonical(m, n, meter), header, meter) {
		return nil, errProviderIntegration
	}
	p := &providerHistoryOriginalPack{stage: stage, table: table, purpose: purpose, first: first, count: count, entries: entries}
	start += int(n)
	stride := entries / count
	for i, item := range items {
		row := contentObject(item, "ordinal", "kind", "digest", "bytes")
		size, sizeOK := providerCandidateUint(row["bytes"], true)
		kind, digest := contentString(row["kind"]), contentString(row["digest"])
		if row == nil || row["ordinal"] != strconv.Itoa(first+i/stride) || kind != providerHistoryOriginalKind(purpose, i) ||
			!sizeOK || !providerHistoryOriginalSize(kind, size) || !lowerHash(digest) || size > uint64(len(raw)-start) {
			return nil, errProviderIntegration
		}
		end := start + int(size)
		if !providerReplayTextEqual(providerReplayDigest("", raw[start:end], meter), digest, meter) {
			return nil, errProviderIntegration
		}
		p.records[i] = providerHistoryOriginalReference{kind, digest, start, int(size)}
		start = end
	}
	if start != len(raw) {
		return nil, errProviderIntegration
	}
	p.digest = providerReplayDigest(providerHistoryOriginalPackDomain, raw, meter)
	if !providerReplayTextEqual(p.digest, expected, meter) {
		return nil, errProviderIntegration
	}
	p.raw = providerReplayString(raw, meter)
	if p.raw == "" {
		return nil, errProviderIntegration
	}
	return p, nil
}

// This immutable string view owns no separate historical cache. Semantic
// readers still have to reconstruct the exact request/HTTP response/TLS data.
func (p *providerHistoryOriginalPack) original(index int) (string, string, bool) {
	if p == nil || index < 0 || index >= p.entries || p.entries <= 0 || p.entries > len(p.records) {
		return "", "", false
	}
	r := p.records[index]
	if r.offset < 0 || r.offset > len(p.raw) || r.size <= 0 || r.size > len(p.raw)-r.offset {
		return "", "", false
	}
	return r.kind, p.raw[r.offset : r.offset+r.size], true
}

// Reconstruct the complete T request table from independently read HTTP
// originals. Neither a supplied decoded request nor the pack descriptor alone
// supplies the table. Program grammar/retention/ACK remain outside this DATA
// boundary, and request table purpose is independently selected by the caller.
func parseProviderHistoryRequestPackObserved(raw []byte, expected string, stage uint64, program, purpose, table string, count int, bounds providerMethodBounds, capture ProviderCaptureLimits, maximum uint64, meter *providerWorkMeter) (*providerHistoryOriginalPack, *providerHistoryGroupTable, error) {
	p, err := parseProviderHistoryOriginalPackObserved(raw, expected, stage, table, "requests", 0, count, maximum, meter)
	if err != nil {
		return nil, nil, errProviderIntegration
	}
	requests := make([]*providerIntegrationRequest, count)
	for i := range count {
		kind, wire, ok := p.original(i)
		if !ok || kind != "request" {
			return nil, nil, errProviderIntegration
		}
		request, err := providerHistoryRequestOriginalObserved(providerReplayBytes(wire, meter), meter)
		if err != nil {
			return nil, nil, errProviderIntegration
		}
		requests[i] = request
	}
	t, err := newProviderHistoryGroupTableObserved(program, purpose, stage, requests, bounds, capture, meter)
	if err != nil || !providerReplayTextEqual(t.digest, table, meter) {
		return nil, nil, errProviderIntegration
	}
	return p, t, nil
}

func providerHistoryRequestOriginalObserved(wire []byte, meter *providerWorkMeter) (*providerIntegrationRequest, error) {
	if len(wire) == 0 || len(wire) > 16<<10 || meter != nil && !meter.event(providerCompareEvent, uint64(min(len(wire), 512)+4)) {
		return nil, errProviderIntegration
	}
	end := bytes.Index(wire[:min(len(wire), 512)], []byte("\r\n\r\n"))
	if end < 0 || end+4 == len(wire) {
		return nil, errProviderIntegration
	}
	var body struct {
		Version string          `json:"jsonrpc"`
		ID      uint64          `json:"id"`
		Method  string          `json:"method"`
		Params  json.RawMessage `json:"params"`
	}
	if providerReplayUnmarshal(wire[end+4:], &body, meter) != nil || body.Version != "2.0" || body.ID != 1 || len(body.Method) > 64 || len(body.Params) == 0 || len(body.Params) > 8<<10 {
		return nil, errProviderIntegration
	}
	var params any
	if providerReplayUnmarshal(body.Params, &params, meter) != nil {
		return nil, errProviderIntegration
	}
	// Only bounded JSON containers reach recursive numeric normalization.
	// In particular deeply nested JSON cannot walk an unbounded request tree.
	nodes := 0
	normalized, ok := providerHistoryRequestNumbersObserved(params, 0, &nodes, meter)
	if !ok {
		return nil, errProviderIntegration
	}
	r, err := newProviderIntegrationRequestObserved(body.Method, normalized, meter)
	if err != nil || !providerReplayEqual(providerReplayBytes(r.wire, meter), wire, meter) {
		return nil, errProviderIntegration
	}
	return r, nil
}

func providerHistoryRequestNumbersObserved(value any, depth int, nodes *int, meter *providerWorkMeter) (any, bool) {
	*nodes += 1
	if *nodes > 512 || depth > 8 || meter != nil && !meter.event(providerCompareEvent, 1) {
		return nil, false
	}
	switch v := value.(type) {
	case float64:
		return int(0), v == 0
	case bool, string:
		return value, true
	case []any:
		if len(v) > 16 {
			return nil, false
		}
		for i, child := range v {
			converted, ok := providerHistoryRequestNumbersObserved(child, depth+1, nodes, meter)
			if !ok {
				return nil, false
			}
			v[i] = converted
		}
		return v, true
	case map[string]any:
		if len(v) > 16 {
			return nil, false
		}
		for name, child := range v {
			converted, ok := providerHistoryRequestNumbersObserved(child, depth+1, nodes, meter)
			if !ok {
				return nil, false
			}
			v[name] = converted
		}
		return v, true
	}
	return nil, false
}

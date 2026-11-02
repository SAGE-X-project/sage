package registry011

import (
	"bytes"
	"encoding/json"
	"io"
	"strconv"
)

func nodeArrayObserved(fields map[string]json.RawMessage, key string, out *[]json.RawMessage, maximum int, meter *providerWorkMeter) bool {
	if meter == nil {
		return nodeArray(fields, key, out, maximum)
	}
	if out == nil {
		return false
	}
	raw, present := fields[key]
	if !present {
		return true // Optional field absence never produces a decoded array.
	}
	var ok bool
	*out, ok = nodeRawArrayObserved(raw, maximum, meter)
	return ok
}

// The bounded streaming array parse is one JSON codec invocation, like the
// existing strict token parser. Its input bytes, owned slice initialization
// and explicit element-header copies are observed separately. Decoder internals
// remain part of that pinned JSON primitive, not additional reported actuals.
func nodeRawArrayObserved(raw json.RawMessage, maximum int, meter *providerWorkMeter) ([]json.RawMessage, bool) {
	if meter == nil {
		return nodeRawArray(raw, maximum)
	}
	if maximum < 0 || maximum > 65536 || len(raw) > 64<<20 || !meter.event(providerCodecEvent, uint64(len(raw))) {
		return nil, false
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	token, err := d.Token()
	if err != nil || token != json.Delim('[') {
		return nil, false
	}
	// Each nonempty JSON element needs at least one byte and a separator;
	// this raw-length ceiling is independently bounded by the selected count.
	// Preallocate once so append growth cannot hide copies of an old prefix.
	slots := min(maximum, max(0, (len(raw)-1)/2))
	const sliceBytes = uint64(3 * (strconv.IntSize / 8))
	if !meter.event(providerCopyEvent, uint64(slots)*sliceBytes) {
		return nil, false
	}
	out := make([]json.RawMessage, slots)
	count := 0
	for d.More() {
		if count >= len(out) {
			return nil, false
		}
		var value json.RawMessage
		if d.Decode(&value) != nil || !meter.event(providerCopyEvent, sliceBytes) {
			return nil, false
		}
		out[count] = value
		count++
	}
	token, err = d.Token()
	if err != nil || token != json.Delim(']') {
		return nil, false
	}
	if _, err = d.Token(); err != io.EOF {
		return nil, false
	}
	return out[:count], true
}

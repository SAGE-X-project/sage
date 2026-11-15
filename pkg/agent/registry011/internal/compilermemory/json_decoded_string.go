package compilermemory

import "unicode/utf8"

// DecodedStringLength measures the detached UTF-8 destination needed by
// CopyString. The result neither exposes native bytes nor extends their life.
func (v JSONValue) DecodedStringLength() (int, error) {
	r := v.document.owner
	if r == nil || r.self != r {
		return 0, ErrConfiguration
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if err := r.accessError(); err != nil {
		return 0, err
	}
	raw, err := v.decodedStringLocked()
	if err != nil {
		return 0, err
	}
	length, err := r.decodedStringSize(raw)
	if err != nil {
		return 0, err
	}
	if err := r.actionError(); err != nil {
		return 0, err
	}
	return length, nil
}

// CopyString writes to an exact-sized Go destination without exposing an alias
// into the native mapping. Every failure clears the supplied destination. The
// Go destination and any later Go string remain separate resource obligations.
func (v JSONValue) CopyString(destination []byte) (err error) {
	defer func() {
		if err != nil {
			clear(destination)
		}
	}()
	r := v.document.owner
	if r == nil || r.self != r {
		return ErrConfiguration
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if err := r.accessError(); err != nil {
		return err
	}
	raw, err := v.decodedStringLocked()
	if err != nil {
		return err
	}
	length, err := r.decodedStringSize(raw)
	if err != nil {
		return err
	}
	if len(destination) != length {
		return ErrRange
	}
	written := 0
	for position := 1; position < len(raw)-1; {
		if err := r.actionError(); err != nil {
			return err
		}
		value, next, valid := jsonStringRune(raw, position, len(raw)-1)
		if !valid || next <= position || next > len(raw)-1 {
			return ErrJSON
		}
		size := utf8.RuneLen(value)
		if size <= 0 || size > len(destination)-written {
			return ErrJSON
		}
		written += utf8.EncodeRune(destination[written:], value)
		position = next
	}
	if written != len(destination) {
		return ErrJSON
	}
	return r.actionError()
}

// The caller holds the original owner's read lock throughout use of this slice.
func (v JSONValue) decodedStringLocked() ([]byte, error) {
	node, err := v.inspectLocked()
	if err != nil {
		return nil, err
	}
	if node.kind != JSONString {
		return nil, ErrJSONType
	}
	span, err := v.document.tokenSpan(node)
	if err != nil {
		return nil, err
	}
	raw := v.document.owner.data[span.offset : span.offset+span.length]
	if len(raw) < 2 || raw[0] != '"' || raw[len(raw)-1] != '"' {
		return nil, ErrJSON
	}
	return raw, nil
}

func (r *Region) decodedStringSize(raw []byte) (int, error) {
	length := 0
	for position := 1; position < len(raw)-1; {
		if err := r.actionError(); err != nil {
			return 0, err
		}
		value, next, valid := jsonStringRune(raw, position, len(raw)-1)
		if !valid || next <= position || next > len(raw)-1 {
			return 0, ErrJSON
		}
		size := utf8.RuneLen(value)
		if size <= 0 || size > len(raw)-length {
			return 0, ErrJSON
		}
		length += size
		position = next
	}
	return length, nil
}

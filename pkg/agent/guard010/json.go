// Package guard010 implements bounded Execution Guard commitments and signed
// message verification. Verification alone does not authorize tool dispatch.
package guard010

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/sage-x-project/sage/pkg/agent/crypto/jcs"
	"io"
	"math"
	"strconv"
	"unicode/utf8"
)

const MaxBytes = 1 << 20

var ErrInvalid = errors.New("guard authentication failed")

// Canonicalize validates Guard JSON before canonicalizing it. Limits include
// the entire input and all nested object members, before signature verification.
func Canonicalize(raw []byte) ([]byte, error) { return canonicalize(raw, 4096) }
func canonicalize(raw []byte, limit int) ([]byte, error) {
	if len(raw) > MaxBytes || !utf8.Valid(raw) || !validEscapes(raw) {
		return nil, ErrInvalid
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	members := 0
	if _, err := value(d, 0, &members, limit); err != nil {
		return nil, ErrInvalid
	}
	if _, err := d.Token(); err != io.EOF {
		return nil, ErrInvalid
	}
	b, err := jcs.Canonicalize(raw)
	if err != nil || len(b) > MaxBytes {
		return nil, ErrInvalid
	}
	return b, nil
}
func value(d *json.Decoder, depth int, members *int, limit int) (any, error) {
	t, e := d.Token()
	if e != nil {
		return nil, e
	}
	switch x := t.(type) {
	case json.Delim:
		if depth >= 32 {
			return nil, ErrInvalid
		}
		switch x {
		case '{':
			m := map[string]any{}
			for d.More() {
				k, e := d.Token()
				if e != nil {
					return nil, e
				}
				s, ok := k.(string)
				if !ok {
					return nil, ErrInvalid
				}
				if _, ok = m[s]; ok {
					return nil, ErrInvalid
				}
				*members++
				if *members > limit {
					return nil, ErrInvalid
				}
				v, e := value(d, depth+1, members, limit)
				if e != nil {
					return nil, e
				}
				m[s] = v
			}
			end, e := d.Token()
			if e != nil || end != json.Delim('}') {
				return nil, ErrInvalid
			}
			return m, nil
		case '[':
			a := []any{}
			for d.More() {
				v, e := value(d, depth+1, members, limit)
				if e != nil {
					return nil, e
				}
				a = append(a, v)
			}
			end, e := d.Token()
			if e != nil || end != json.Delim(']') {
				return nil, ErrInvalid
			}
			return a, nil
		default:
			return nil, ErrInvalid
		}
	case json.Number:
		n, e := strconv.ParseFloat(string(x), 64)
		if e != nil || math.IsInf(n, 0) || math.IsNaN(n) || (n == 0 && math.Signbit(n)) {
			return nil, ErrInvalid
		}
	}
	return t, nil
}

// encoding/json replaces lone UTF-16 surrogates, so reject them before decoding.
func validEscapes(b []byte) bool {
	inside := false
	for i := 0; i < len(b); i++ {
		if b[i] == '"' {
			inside = !inside
			continue
		}
		if !inside || b[i] != '\\' {
			continue
		}
		i++
		if i >= len(b) {
			return false
		}
		if b[i] != 'u' {
			continue
		}
		if i+4 >= len(b) {
			return false
		}
		n, e := strconv.ParseUint(string(b[i+1:i+5]), 16, 16)
		if e != nil {
			return false
		}
		i += 4
		if n >= 0xDC00 && n <= 0xDFFF {
			return false
		}
		if n < 0xD800 || n > 0xDBFF {
			continue
		}
		if i+6 >= len(b) || b[i+1] != '\\' || b[i+2] != 'u' {
			return false
		}
		v, e := strconv.ParseUint(string(b[i+3:i+7]), 16, 16)
		if e != nil || v < 0xDC00 || v > 0xDFFF {
			return false
		}
		i += 6
	}
	return true
}
func object(raw []byte) (map[string]any, []byte, error) { return objectLimit(raw, 4096) }
func objectLimit(raw []byte, limit int) (map[string]any, []byte, error) {
	b, e := canonicalize(raw, limit)
	if e != nil {
		return nil, nil, e
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var m map[string]any
	if d.Decode(&m) != nil || m == nil {
		return nil, nil, ErrInvalid
	}
	return m, b, nil
}
func encode(v any) []byte { b, _ := json.Marshal(v); return b }
func closed(m map[string]any, fields string) bool {
	names := split(fields)
	if len(m) != len(names) {
		return false
	}
	for _, n := range names {
		if _, ok := m[n]; !ok {
			return false
		}
	}
	return true
}

package registry010

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"
)

// ErrSizeExceeded010 is the REG-08 size.exceeded result for a response body
// larger than the web Registry envelope limit.
var ErrSizeExceeded010 = errors.New("size.exceeded")

const maxWebBody010 = 69632
const maxExactInteger010 = 9007199254740991

// CheckWebRegistryEnvelope010 checks the bounded JSON response wrapper and
// its short lifetime. It does not validate the nested Registry record, its
// proofs, HTTPS origin, or authority. Success must never authorize a record.
// The trusted HTTP adapter must stop reading at the first byte past 69632.
func CheckWebRegistryEnvelope010(raw []byte, now int64) error {
	if len(raw) > maxWebBody010 {
		return ErrSizeExceeded010
	}
	if len(raw) == 0 || !utf8.Valid(raw) || bytes.HasPrefix(raw, []byte{0xef, 0xbb, 0xbf}) || !validWebEscapes010(raw) {
		return ErrInvalidRecord010
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	value, err := readWebValue010(decoder, 0)
	if err != nil {
		return ErrInvalidRecord010
	}
	if _, err = decoder.Token(); err != io.EOF {
		return ErrInvalidRecord010
	}
	root, ok := value.(map[string]any)
	if !ok || len(root) != 3 {
		return ErrInvalidRecord010
	}
	if _, ok = root["record"].(map[string]any); !ok {
		return ErrInvalidRecord010
	}
	issued, ok := webInteger010(root["issued"])
	if !ok {
		return ErrInvalidRecord010
	}
	expires, ok := webInteger010(root["expires"])
	if !ok || now < -maxExactInteger010 || now > maxExactInteger010 || issued > now || now >= expires || expires <= issued || expires-issued > 5 {
		return ErrInvalidRecord010
	}
	return nil
}

func readWebValue010(decoder *json.Decoder, depth int) (any, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	switch v := token.(type) {
	case json.Delim:
		if depth >= 64 {
			return nil, ErrInvalidRecord010
		}
		switch v {
		case '{':
			members := make(map[string]any)
			for decoder.More() {
				key, err := decoder.Token()
				name, ok := key.(string)
				if err != nil || !ok {
					return nil, ErrInvalidRecord010
				}
				if _, found := members[name]; found {
					return nil, ErrInvalidRecord010
				}
				value, err := readWebValue010(decoder, depth+1)
				if err != nil {
					return nil, err
				}
				members[name] = value
			}
			end, err := decoder.Token()
			if err != nil || end != json.Delim('}') {
				return nil, ErrInvalidRecord010
			}
			return members, nil
		case '[':
			items := make([]any, 0)
			for decoder.More() {
				value, err := readWebValue010(decoder, depth+1)
				if err != nil {
					return nil, err
				}
				items = append(items, value)
			}
			end, err := decoder.Token()
			if err != nil || end != json.Delim(']') {
				return nil, ErrInvalidRecord010
			}
			return items, nil
		default:
			return nil, ErrInvalidRecord010
		}
	case json.Number:
		floating, err := strconv.ParseFloat(string(v), 64)
		if err != nil || math.IsInf(floating, 0) || math.IsNaN(floating) || (floating == 0 && math.Signbit(floating)) {
			return nil, ErrInvalidRecord010
		}
	}
	return token, nil
}

func webInteger010(value any) (int64, bool) {
	number, ok := value.(json.Number)
	if !ok {
		return 0, false
	}
	literal := string(number)
	negative := strings.HasPrefix(literal, "-")
	if negative {
		literal = literal[1:]
	}
	exponent := int64(0)
	if index := strings.IndexAny(literal, "eE"); index >= 0 {
		var err error
		exponent, err = strconv.ParseInt(literal[index+1:], 10, 64)
		if err != nil || exponent < -100000 || exponent > 100000 {
			return 0, false
		}
		literal = literal[:index]
	}
	fraction := 0
	if index := strings.IndexByte(literal, '.'); index >= 0 {
		fraction = len(literal) - index - 1
		literal = literal[:index] + literal[index+1:]
	}
	digits := strings.TrimLeft(literal, "0")
	if digits == "" {
		return 0, !negative
	}
	scale := exponent - int64(fraction)
	if scale < 0 {
		trailing := int64(len(digits) - len(strings.TrimRight(digits, "0")))
		if trailing < -scale {
			return 0, false
		}
		digits = digits[:len(digits)+int(scale)]
		scale = 0
	}
	if scale > 16 || int64(len(digits))+scale > 16 {
		return 0, false
	}
	result, err := strconv.ParseInt(digits+strings.Repeat("0", int(scale)), 10, 64)
	if err != nil || result > maxExactInteger010 {
		return 0, false
	}
	if negative {
		result = -result
	}
	return result, true
}

// encoding/json replaces lone UTF-16 surrogates, so reject those escapes
// before decoding. JSON syntax is checked separately by the decoder.
func validWebEscapes010(raw []byte) bool {
	inside := false
	for i := 0; i < len(raw); i++ {
		if raw[i] == '"' {
			inside = !inside
			continue
		}
		if !inside || raw[i] != '\\' {
			continue
		}
		i++
		if i >= len(raw) {
			return false
		}
		if raw[i] != 'u' {
			continue
		}
		if i+4 >= len(raw) {
			return false
		}
		first, err := strconv.ParseUint(string(raw[i+1:i+5]), 16, 16)
		if err != nil {
			return false
		}
		i += 4
		if first >= 0xdc00 && first <= 0xdfff {
			return false
		}
		if first < 0xd800 || first > 0xdbff {
			continue
		}
		if i+6 >= len(raw) || raw[i+1] != '\\' || raw[i+2] != 'u' {
			return false
		}
		second, err := strconv.ParseUint(string(raw[i+3:i+7]), 16, 16)
		if err != nil || second < 0xdc00 || second > 0xdfff {
			return false
		}
		i += 6
	}
	return true
}

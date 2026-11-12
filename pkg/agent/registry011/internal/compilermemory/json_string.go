package compilermemory

import (
	"unicode/utf16"
	"unicode/utf8"
)

func jsonHexRune(raw []byte, at, end int) (rune, bool) {
	if end < 0 || end > len(raw) || at < 0 || at > end || end-at < 4 {
		return 0, false
	}
	var value rune
	for _, b := range raw[at : at+4] {
		value <<= 4
		switch {
		case b >= '0' && b <= '9':
			value |= rune(b - '0')
		case b >= 'a' && b <= 'f':
			value |= rune(b - 'a' + 10)
		case b >= 'A' && b <= 'F':
			value |= rune(b - 'A' + 10)
		default:
			return 0, false
		}
	}
	return value, true
}

// Decode one string-body rune without a temporary string. Unpaired escaped
// surrogates use the same replacement behavior as the existing Go JSON decoder;
// malformed raw UTF-8 remains rejected by the preparation boundary.
func jsonStringRune(raw []byte, at, end int) (rune, int, bool) {
	if end < 0 || end > len(raw) || at < 0 || at >= end {
		return 0, at, false
	}
	b := raw[at]
	if b < 0x20 || b == '"' {
		return 0, at, false
	}
	if b != '\\' {
		if b < utf8.RuneSelf {
			return rune(b), at + 1, true
		}
		value, size := utf8.DecodeRune(raw[at:end])
		return value, at + size, value != utf8.RuneError || size != 1
	}
	if end-at < 2 {
		return 0, at, false
	}
	switch raw[at+1] {
	case '"', '\\', '/':
		return rune(raw[at+1]), at + 2, true
	case 'b':
		return '\b', at + 2, true
	case 'f':
		return '\f', at + 2, true
	case 'n':
		return '\n', at + 2, true
	case 'r':
		return '\r', at + 2, true
	case 't':
		return '\t', at + 2, true
	case 'u':
		value, ok := jsonHexRune(raw, at+2, end)
		if !ok {
			return 0, at, false
		}
		next := at + 6
		if value >= 0xd800 && value <= 0xdbff && end-next >= 6 && raw[next] == '\\' && raw[next+1] == 'u' {
			low, valid := jsonHexRune(raw, next+2, end)
			if valid && low >= 0xdc00 && low <= 0xdfff {
				return utf16.DecodeRune(value, low), next + 6, true
			}
		}
		if utf16.IsSurrogate(value) {
			value = utf8.RuneError
		}
		return value, next, true
	default:
		return 0, at, false
	}
}

func (r *Region) equalJSONStrings(a, b []byte) (bool, error) {
	if len(a) < 2 || len(b) < 2 || a[0] != '"' || b[0] != '"' || a[len(a)-1] != '"' || b[len(b)-1] != '"' {
		return false, ErrJSON
	}
	i, j := 1, 1
	for i < len(a)-1 && j < len(b)-1 {
		if err := r.actionError(); err != nil {
			return false, err
		}
		x, nextI, validI := jsonStringRune(a, i, len(a)-1)
		y, nextJ, validJ := jsonStringRune(b, j, len(b)-1)
		if !validI || !validJ {
			return false, ErrJSON
		}
		if x != y {
			return false, nil
		}
		i, j = nextI, nextJ
	}
	return i == len(a)-1 && j == len(b)-1, nil
}

func (r *Region) equalJSONStringText(raw []byte, text string) (bool, error) {
	if len(raw) < 2 || raw[0] != '"' || raw[len(raw)-1] != '"' {
		return false, ErrJSON
	}
	if len(text) > len(raw)-2 || !utf8.ValidString(text) {
		return false, nil
	}
	i, j := 1, 0
	for i < len(raw)-1 && j < len(text) {
		if err := r.actionError(); err != nil {
			return false, err
		}
		x, next, valid := jsonStringRune(raw, i, len(raw)-1)
		if !valid {
			return false, ErrJSON
		}
		y, size := utf8.DecodeRuneInString(text[j:])
		if x != y {
			return false, nil
		}
		i, j = next, j+size
	}
	return i == len(raw)-1 && j == len(text), nil
}

package rfc8785

import (
	"bytes"
	"math"
	"sort"
	"unicode/utf8"
)

// Eligibility has no output or user callbacks. Unknown types and deep/cyclic
// values use the original encoding/json normalization, including its errors,
// custom marshalers, nil containers and invalid UTF-8 replacement semantics.
func nativeCanonicalValue(v any, depth int) bool {
	if depth > 64 {
		return false
	}
	switch x := v.(type) {
	case nil, bool:
		return true
	case string:
		return utf8.ValidString(x)
	case float64:
		return !math.IsNaN(x) && !math.IsInf(x, 0)
	case map[string]any:
		if x == nil {
			return false
		}
		for key, value := range x {
			if !utf8.ValidString(key) || !nativeCanonicalValue(value, depth+1) {
				return false
			}
		}
		return true
	case map[string]string:
		if x == nil {
			return false
		}
		for key, value := range x {
			if !utf8.ValidString(key) || !utf8.ValidString(value) {
				return false
			}
		}
		return true
	case []map[string]any:
		if x == nil {
			return false
		}
		for _, value := range x {
			if !nativeCanonicalValue(value, depth+1) {
				return false
			}
		}
		return true
	case []map[string]string:
		if x == nil {
			return false
		}
		for _, value := range x {
			if !nativeCanonicalValue(value, depth+1) {
				return false
			}
		}
		return true
	case []any:
		if x == nil {
			return false
		}
		for _, value := range x {
			if !nativeCanonicalValue(value, depth+1) {
				return false
			}
		}
		return true
	case []string:
		if x == nil {
			return false
		}
		for _, value := range x {
			if !utf8.ValidString(value) {
				return false
			}
		}
		return true
	}
	return false
}

// Called only after the ENTIRE value passed eligibility. It preserves the
// existing UTF-16 key order and canonical string/number writers byte for byte.
func writeNativeCanonicalValue(buf *bytes.Buffer, v any) error {
	switch x := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(x))
		for key := range x {
			keys = append(keys, key)
		}
		sort.Slice(keys, func(i, j int) bool { return lessUTF16(keys[i], keys[j]) })
		buf.WriteByte('{')
		for i, key := range keys {
			if i != 0 {
				buf.WriteByte(',')
			}
			if err := writeString(buf, key); err != nil {
				return err
			}
			buf.WriteByte(':')
			if err := writeNativeCanonicalValue(buf, x[key]); err != nil {
				return err
			}
		}
		buf.WriteByte('}')
	case map[string]string:
		keys := make([]string, 0, len(x))
		for key := range x {
			keys = append(keys, key)
		}
		sort.Slice(keys, func(i, j int) bool { return lessUTF16(keys[i], keys[j]) })
		buf.WriteByte('{')
		for i, key := range keys {
			if i != 0 {
				buf.WriteByte(',')
			}
			if err := writeString(buf, key); err != nil {
				return err
			}
			buf.WriteByte(':')
			if err := writeString(buf, x[key]); err != nil {
				return err
			}
		}
		buf.WriteByte('}')
	case []map[string]any:
		buf.WriteByte('[')
		for i, value := range x {
			if i != 0 {
				buf.WriteByte(',')
			}
			if err := writeNativeCanonicalValue(buf, value); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
	case []map[string]string:
		buf.WriteByte('[')
		for i, value := range x {
			if i != 0 {
				buf.WriteByte(',')
			}
			if err := writeNativeCanonicalValue(buf, value); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
	case []any:
		buf.WriteByte('[')
		for i, value := range x {
			if i != 0 {
				buf.WriteByte(',')
			}
			if err := writeNativeCanonicalValue(buf, value); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
	case []string:
		buf.WriteByte('[')
		for i, value := range x {
			if i != 0 {
				buf.WriteByte(',')
			}
			if err := writeString(buf, value); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
	default:
		return writeValue(buf, v)
	}
	return nil
}

package registry011

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"unicode/utf8"
)

var ErrRPCResponse = errors.New("registry011: unusable RPC response")

// RPCResponseLimits are explicit parsing limits, not operating policy defaults.
// MaxDepth counts the response object as depth one. The implementation supports
// at most 128 container levels; callers must qualify a smaller operating limit.
type RPCResponseLimits struct {
	MaxBytes int
	MaxDepth int
}

// DecodeRPCResult accepts one response to a numeric request ID in 1..2^53-1.
// IDs use their exact decimal spelling, without float conversion or coercion.
// Duplicate names at any depth, ambiguous Unicode, trailing values, errors,
// batches and unrecognized envelope fields refuse. Result bytes are copied.
//
// This validates only the response envelope. A result (including null) is not
// authenticated state or absence. Method-specific ABI, source, block, clock,
// complete history and durable publication checks are separate requirements.
// The transport must enforce the body limit while reading, before allocating
// an unbounded body; this function cannot undo an earlier unbounded allocation.
func DecodeRPCResult(body []byte, requestID uint64, limits RPCResponseLimits) (json.RawMessage, error) {
	if limits.MaxBytes <= 0 || limits.MaxDepth <= 0 || limits.MaxDepth > 128 ||
		requestID == 0 || requestID > 1<<53-1 {
		return nil, ErrConfiguration
	}
	if len(body) == 0 || len(body) > limits.MaxBytes || !utf8.Valid(body) || !validRPCUnicode(body) {
		return nil, ErrRPCResponse
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if !readRPCValue(decoder, 0, limits.MaxDepth) {
		return nil, ErrRPCResponse
	}
	if _, err := decoder.Token(); err != io.EOF {
		return nil, ErrRPCResponse
	}
	var envelope map[string]json.RawMessage
	if json.Unmarshal(body, &envelope) != nil || len(envelope) != 3 ||
		string(envelope["id"]) != strconv.FormatUint(requestID, 10) {
		return nil, ErrRPCResponse
	}
	var version string
	if json.Unmarshal(envelope["jsonrpc"], &version) != nil || version != "2.0" {
		return nil, ErrRPCResponse
	}
	result, present := envelope["result"]
	if !present {
		return nil, ErrRPCResponse
	}
	return append(json.RawMessage(nil), result...), nil
}

// readRPCValue checks names before any map unmarshalling can discard duplicates.
func readRPCValue(decoder *json.Decoder, depth, maximum int) bool {
	token, err := decoder.Token()
	if err != nil {
		return false
	}
	delimiter, container := token.(json.Delim)
	if !container {
		return true
	}
	if depth >= maximum {
		return false
	}
	var closing json.Delim
	switch delimiter {
	case '{':
		closing = '}'
		seen := make(map[string]bool)
		for decoder.More() {
			key, err := decoder.Token()
			name, isString := key.(string)
			if err != nil || !isString || seen[name] {
				return false
			}
			seen[name] = true
			if !readRPCValue(decoder, depth+1, maximum) {
				return false
			}
		}
	case '[':
		closing = ']'
		for decoder.More() {
			if !readRPCValue(decoder, depth+1, maximum) {
				return false
			}
		}
	default:
		return false
	}
	end, err := decoder.Token()
	return err == nil && end == closing
}

// encoding/json replaces unpaired surrogate escapes with U+FFFD. Refuse these
// inputs before decoding, so distinct raw names cannot be silently repaired.
// The decoder, not this scan, remains responsible for all other JSON syntax.
func validRPCUnicode(body []byte) bool {
	for i := 0; i < len(body); i++ {
		if body[i] != '"' {
			continue
		}
		i++
		for i < len(body) && body[i] != '"' {
			if body[i] != '\\' {
				i++
				continue
			}
			i++
			if i >= len(body) {
				return false
			}
			if body[i] != 'u' {
				i++
				continue
			}
			unit, ok := rpcUnicodeUnit(body, i+1)
			if !ok || unit >= 0xdc00 && unit <= 0xdfff {
				return false
			}
			i += 5
			if unit >= 0xd800 && unit <= 0xdbff {
				if i+1 >= len(body) || body[i] != '\\' || body[i+1] != 'u' {
					return false
				}
				low, ok := rpcUnicodeUnit(body, i+2)
				if !ok || low < 0xdc00 || low > 0xdfff {
					return false
				}
				i += 6
			}
		}
	}
	return true
}

func rpcUnicodeUnit(body []byte, start int) (uint64, bool) {
	if len(body)-start < 4 {
		return 0, false
	}
	value, err := strconv.ParseUint(string(body[start:start+4]), 16, 16)
	return value, err == nil
}

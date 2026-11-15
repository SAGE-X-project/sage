//go:build darwin

package registry011

import (
	"errors"
	"sort"
	"sync/atomic"

	"github.com/sage-x-project/sage/pkg/agent/registry011/internal/compilermemory"
)

// This one-use DATA reader borrows regions owned by the original preparation
// session. It never constructs a Go policy tree or returns policy authority.
// Its control objects and bounded Go scratch remain separate resource duties.
// The original session serializes all use and owns both regions' cleanup.
type nativeProviderPolicyReader struct {
	self            *nativeProviderPolicyReader
	source, numbers *compilermemory.Region
	input           compilermemory.Span
	inputBytes      int
	root            compilermemory.JSONValue
	used            atomic.Bool
	err             error
}

type nativeProviderPolicyValue struct {
	reader *nativeProviderPolicyReader
	value  compilermemory.JSONValue
	valid  bool
}

func newNativeProviderPolicyReader(source, numbers *compilermemory.Region, document compilermemory.JSONDocument, input compilermemory.Span, inputBytes int) (*nativeProviderPolicyReader, error) {
	if source == nil || numbers == nil || source == numbers || inputBytes <= 0 || inputBytes > 64<<10 {
		return nil, ErrConfiguration
	}
	root := document.Root()
	span, err := root.Raw()
	if err != nil {
		return nil, errors.Join(ErrProviderPolicyCandidate, err)
	}
	// Equality also rejects leading/trailing transport whitespace. Slice checks
	// that this exact parsed root belongs to the supplied original region.
	if span != input {
		return nil, ErrProviderPolicyCandidate
	}
	whole, err := source.Slice(span, 0, inputBytes)
	if err != nil {
		return nil, errors.Join(ErrProviderPolicyCandidate, err)
	}
	if whole != span {
		return nil, ErrProviderPolicyCandidate
	}
	r := &nativeProviderPolicyReader{source: source, numbers: numbers, input: input, inputBytes: inputBytes, root: root}
	r.self = r
	return r, nil
}

func (r *nativeProviderPolicyReader) validate() error {
	if r == nil || r.self != r || !r.used.CompareAndSwap(false, true) {
		return ErrConfiguration
	}
	valid := providerPolicyFieldsValid(nativeProviderPolicyValue{reader: r, value: r.root, valid: true})
	if r.err != nil {
		return r.err
	}
	if !valid {
		return ErrProviderPolicyCandidate
	}
	return r.canonicalText()
}

func (r *nativeProviderPolicyReader) fail(err error) {
	if err != nil && r.err == nil {
		r.err = errors.Join(ErrProviderPolicyCandidate, err)
	}
}

func (v nativeProviderPolicyValue) kind(want compilermemory.JSONKind) bool {
	if v.reader == nil || v.reader.self != v.reader || v.reader.err != nil || !v.valid {
		return false
	}
	kind, err := v.value.Kind()
	if err != nil {
		v.reader.fail(err)
		return false
	}
	return kind == want
}

func (v nativeProviderPolicyValue) object(names ...string) bool {
	if !v.kind(compilermemory.JSONObject) {
		return false
	}
	length, err := v.value.Length()
	if err != nil {
		v.reader.fail(err)
		return false
	}
	if length != len(names) {
		return false
	}
	// The unchanged closed policy has at most fourteen fields in an object.
	// These are schema names, never a materialized wire key/value map.
	var ordered [14]string
	if len(names) > len(ordered) {
		v.reader.fail(ErrConfiguration)
		return false
	}
	copy(ordered[:], names)
	sort.Strings(ordered[:len(names)])
	var previous compilermemory.Span
	for index, name := range ordered[:len(names)] {
		if index != 0 && name == ordered[index-1] {
			v.reader.fail(ErrConfiguration)
			return false
		}
		child, found, err := v.value.Member(name)
		if err != nil {
			v.reader.fail(err)
			return false
		}
		if !found {
			return false
		}
		span, err := child.Raw()
		if err != nil {
			v.reader.fail(err)
			return false
		}
		if index != 0 {
			before, err := v.reader.source.Precedes(previous, span)
			if err != nil {
				v.reader.fail(err)
				return false
			}
			if !before {
				return false
			}
		}
		previous = span
	}
	return true
}

func (v nativeProviderPolicyValue) member(name string) providerPolicyValue {
	missing := nativeProviderPolicyValue{reader: v.reader}
	if !v.kind(compilermemory.JSONObject) {
		return missing
	}
	child, found, err := v.value.Member(name)
	if err != nil {
		v.reader.fail(err)
		return missing
	}
	return nativeProviderPolicyValue{reader: v.reader, value: child, valid: found}
}

func (v nativeProviderPolicyValue) arrayLength() (int, bool) {
	if !v.kind(compilermemory.JSONArray) {
		return 0, false
	}
	length, err := v.value.Length()
	if err != nil {
		v.reader.fail(err)
		return 0, false
	}
	return length, true
}

func (v nativeProviderPolicyValue) element(index int) providerPolicyValue {
	missing := nativeProviderPolicyValue{reader: v.reader}
	length, ok := v.arrayLength()
	if !ok || index < 0 || index >= length {
		return missing
	}
	child, err := v.value.Index(index)
	if err != nil {
		v.reader.fail(err)
		return missing
	}
	return nativeProviderPolicyValue{reader: v.reader, value: child, valid: true}
}

func (v nativeProviderPolicyValue) equals(text string) bool {
	if !v.kind(compilermemory.JSONString) {
		return false
	}
	equal, err := v.value.StringEqual(text)
	if err != nil {
		v.reader.fail(err)
		return false
	}
	return equal
}

func (v nativeProviderPolicyValue) text(destination []byte) (int, bool) {
	if !v.kind(compilermemory.JSONString) {
		return 0, false
	}
	length, err := v.value.DecodedStringLength()
	if err != nil {
		v.reader.fail(err)
		return 0, false
	}
	if length > len(destination) {
		return 0, false
	}
	if err := v.value.CopyString(destination[:length]); err != nil {
		v.reader.fail(err)
		return 0, false
	}
	return length, true
}

func (v nativeProviderPolicyValue) decimal(positive bool) (uint64, bool) {
	if !v.kind(compilermemory.JSONString) {
		return 0, false
	}
	length, err := v.value.DecodedStringLength()
	if err != nil {
		v.reader.fail(err)
		return 0, false
	}
	if length == 0 || length > 20 {
		return 0, false
	}
	span, err := v.value.StoreDecodedString(v.reader.numbers)
	if err != nil {
		v.reader.fail(err)
		return 0, false
	}
	number, err := v.reader.numbers.JSONUint64(span)
	if errors.Is(err, compilermemory.ErrJSONInteger) {
		return 0, false
	}
	if err != nil {
		v.reader.fail(err)
		return 0, false
	}
	return number, !positive || number != 0
}

// This is deliberately policy-schema-specific, not a general JCS encoder. The
// shared semantic validator has already proved that every string is a fixed
// ASCII value, a lowercase digest, a canonical decimal, or a printable clock
// ID. Consequently only quote/backslash escapes can occur in a valid candidate's
// minimal JCS spelling. object has separately checked lexical member order.
func (r *nativeProviderPolicyReader) canonicalText() error {
	var scratch [4096]byte
	var inString, escaped bool
	for offset := 0; offset < r.inputBytes; {
		length := min(len(scratch), r.inputBytes-offset)
		part, err := r.source.Slice(r.input, offset, length)
		if err != nil {
			return errors.Join(ErrProviderPolicyCandidate, err)
		}
		if err := r.source.Read(part, scratch[:length]); err != nil {
			return errors.Join(ErrProviderPolicyCandidate, err)
		}
		for _, b := range scratch[:length] {
			if inString {
				if escaped {
					if b != '"' && b != '\\' {
						return ErrProviderPolicyCandidate
					}
					escaped = false
				} else if b == '\\' {
					escaped = true
				} else if b == '"' {
					inString = false
				} else if b < 32 || b > 127 {
					return ErrProviderPolicyCandidate
				}
			} else if b == '"' {
				inString = true
			} else if b <= 32 || b > 127 {
				return ErrProviderPolicyCandidate
			}
		}
		offset += length
	}
	if inString || escaped {
		return ErrProviderPolicyCandidate
	}
	if _, err := r.root.Kind(); err != nil {
		return errors.Join(ErrProviderPolicyCandidate, err)
	}
	return nil
}

//go:build darwin

package registry011

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/sage-x-project/sage/pkg/agent/registry011/internal/compilermemory"
)

func checkNativeSelectionFixtureValue(t *testing.T, value compilermemory.JSONValue, expected reflect.Value) {
	t.Helper()
	switch expected.Kind() {
	case reflect.Struct:
		for index := range expected.NumField() {
			name, included, err := nativeHistoryFieldName(expected.Type().Field(index))
			if err != nil {
				t.Fatal(err)
			}
			if !included {
				continue
			}
			child, found, err := value.Member(name)
			if err != nil || !found {
				t.Fatalf("native field %s: found=%v, error=%v", name, found, err)
			}
			checkNativeSelectionFixtureValue(t, child, expected.Field(index))
		}
	case reflect.Array:
		for index := range expected.Len() {
			child, err := value.Index(index)
			if err != nil {
				t.Fatal(err)
			}
			checkNativeSelectionFixtureValue(t, child, expected.Index(index))
		}
	case reflect.String:
		want := expected.String()
		length, err := value.DecodedStringLength()
		if err != nil || length != len(want) {
			t.Fatalf("native text length=%d, error=%v; want=%d", length, err, len(want))
		}
		actual := make([]byte, length)
		if err := value.CopyString(actual); err != nil || string(actual) != want {
			t.Fatalf("native text differs: error=%v", err)
		}
	case reflect.Bool:
		if actual, err := value.Boolean(); err != nil || actual != expected.Bool() {
			t.Fatalf("native boolean=%v, error=%v", actual, err)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if actual, err := value.JSONInt64(); err != nil || actual != expected.Int() {
			t.Fatalf("native integer=%d, error=%v", actual, err)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if actual, err := value.JSONUint64(); err != nil || actual != expected.Uint() {
			t.Fatalf("native unsigned integer=%d, error=%v", actual, err)
		}
	default:
		t.Fatalf("unsupported reference type %v", expected.Type())
	}
}

func TestProviderHistoryNativeOwnedSelectionPreservesAllInputData(t *testing.T) {
	raw, _, _ := nativeHistoryTestRaw(t)
	session, state, _ := nativeHistoryTestSession(t, len(raw))
	defer func() {
		if err := session.Close(); err != nil {
			t.Error(err)
		}
	}()
	expected, err := decodeHistoryPreparationInput(session.compiler.ctx, raw)
	if err != nil {
		t.Fatalf("original fixture precondition: %v", err)
	}
	digest := sha256.Sum256(raw)
	selection, err := state.decodeOwnedSelection(session, raw)
	if err != nil {
		t.Fatal(err)
	}
	if selection.compiler != session.compiler || selection.session != session || state.selection != selection || session.plan != nil {
		t.Fatal("selection replaced its original owner or created a plan")
	}
	if selection.inputHash != hex.EncodeToString(digest[:]) {
		t.Fatal("selection did not hash its original consumed bytes")
	}
	clear(raw)
	checkNativeSelectionFixtureValue(t, selection.document.Root(), reflect.ValueOf(expected))
	if _, err := selection.field("maximum_policy_bytes"); err != nil {
		t.Fatal(err)
	}
	if _, err := selection.field("unknown"); !errors.Is(err, ErrConfiguration) {
		t.Fatalf("unknown field=%v", err)
	}
	if _, err := state.decodeOwnedSelection(session, []byte(`{}`)); !errors.Is(err, ErrConfiguration) {
		t.Fatalf("selection attempt was reused: %v", err)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	if err := selection.check(); !errors.Is(err, ErrConfiguration) || !selection.closed || selection.compiler != nil || selection.policy != nil || selection.inputHash != "" {
		t.Fatalf("selection survived owner cleanup: %v", err)
	}
	if !selection.numbers.closed {
		t.Fatal("selection cleanup omitted its numeric region")
	}
}

func TestProviderHistoryNativeOwnedSelectionRefusesUnvalidatedAndLostOwners(t *testing.T) {
	raw, _, _ := nativeHistoryTestRaw(t)
	for _, name := range []string{"missing-field", "unknown-field", "wrong-shape", "policy-limit", "invalid-policy", "canceled"} {
		t.Run(name, func(t *testing.T) {
			session, state, cancel := nativeHistoryTestSession(t, len(raw)+4096)
			defer func() {
				if err := session.Close(); err != nil {
					t.Error(err)
				}
			}()
			var object map[string]json.RawMessage
			if err := json.Unmarshal(raw, &object); err != nil {
				t.Fatal(err)
			}
			switch name {
			case "missing-field":
				delete(object, "format")
			case "unknown-field":
				object["unknown"] = json.RawMessage(`"value"`)
			case "wrong-shape":
				object["store"] = json.RawMessage(`[]`)
			case "policy-limit":
				object["maximum_policy_bytes"] = json.RawMessage(`0`)
			case "invalid-policy":
				object["policy"] = json.RawMessage(`"{}"`)
			case "canceled":
				cancel()
			}
			changed, err := json.Marshal(object)
			if err != nil {
				t.Fatal(err)
			}
			selection, err := state.decodeOwnedSelection(session, changed)
			if err == nil || selection == nil || selection.validated {
				t.Fatalf("invalid selection=%v, error=%v", selection, err)
			}
			switch name {
			case "canceled":
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("original cancellation was lost: %v", err)
				}
			case "invalid-policy":
				if err != ErrProviderPolicyCandidate {
					t.Fatalf("semantic refusal was masked by another failure: %v", err)
				}
			case "unknown-field":
				// An extra member can exceed the exact fixed-schema node table
				// during parsing. Require stored native input and configuration
				// refusal; an acquisition/lifetime failure is not this scenario.
				if selection.input == (compilermemory.Span{}) || !errors.Is(err, ErrConfiguration) ||
					errors.Is(err, compilermemory.ErrUnavailable) || errors.Is(err, compilermemory.ErrClosed) ||
					errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					t.Fatalf("unknown-field refusal was not reached: %v", err)
				}
			default:
				if err != ErrConfiguration {
					t.Fatalf("typed-input refusal was masked by another failure: %v", err)
				}
			}
			if _, err := selection.field("format"); !errors.Is(err, ErrConfiguration) {
				t.Fatalf("partial selection supplied a field: %v", err)
			}
		})
	}
	var absent *nativeHistoryOwnedSelection
	if err := absent.check(); !errors.Is(err, ErrConfiguration) {
		t.Fatalf("nil selection=%v", err)
	}
	if err := json.Unmarshal([]byte(`{}`), &nativeHistoryOwnedSelection{}); !errors.Is(err, ErrConfiguration) {
		t.Fatalf("serialized selection=%v", err)
	}
}

func TestProviderHistoryNativeOwnedSelectionHashesOuterWhitespaceExactly(t *testing.T) {
	raw, _, _ := nativeHistoryTestRaw(t)
	input := append([]byte(" \n"), raw...)
	input = append(input, '\t', '\n')
	session, state, _ := nativeHistoryTestSession(t, len(input))
	defer func() {
		if err := session.Close(); err != nil {
			t.Error(err)
		}
	}()
	if _, err := decodeHistoryPreparationInput(session.compiler.ctx, input); err != nil {
		t.Fatalf("original whitespace precondition: %v", err)
	}
	selection, err := state.decodeOwnedSelection(session, input)
	if err != nil {
		t.Fatal(err)
	}
	want := sha256.Sum256(input)
	if selection.inputHash != hex.EncodeToString(want[:]) || bytes.Equal(input, raw) {
		t.Fatal("input identity discarded transport whitespace")
	}
}

func TestProviderHistoryNativeOwnedSelectionKeepsScalarErrorContract(t *testing.T) {
	raw, _, _ := nativeHistoryTestRaw(t)
	for _, item := range []struct {
		name, parent, field, token string
		cause                      error
	}{
		{"numeric-namespace", "", "namespace", "1", compilermemory.ErrJSONType},
		{"boolean-namespace", "", "namespace", "true", compilermemory.ErrJSONType},
		{"null-namespace", "", "namespace", "null", compilermemory.ErrJSONType},
		{"unsigned-overflow", "clock", "maximum", "18446744073709551616", compilermemory.ErrJSONInteger},
		{"negative-unsigned", "clock", "maximum", "-1", compilermemory.ErrJSONInteger},
		{"signed-overflow", "clock", "base_uncertainty_ms", "9223372036854775808", compilermemory.ErrJSONInteger},
		{"signed-underflow", "clock", "read_window_ms", "-9223372036854775809", compilermemory.ErrJSONInteger},
	} {
		t.Run(item.name, func(t *testing.T) {
			session, state, _ := nativeHistoryTestSession(t, len(raw)+4096)
			defer func() {
				if err := session.Close(); err != nil {
					t.Error(err)
				}
			}()
			if _, err := decodeHistoryPreparationInput(session.compiler.ctx, raw); err != nil {
				t.Fatalf("original positive fixture precondition: %v", err)
			}
			var object map[string]json.RawMessage
			if err := json.Unmarshal(raw, &object); err != nil {
				t.Fatal(err)
			}
			if item.parent == "" {
				object[item.field] = json.RawMessage(item.token)
			} else {
				var child map[string]json.RawMessage
				if err := json.Unmarshal(object[item.parent], &child); err != nil {
					t.Fatal(err)
				}
				child[item.field] = json.RawMessage(item.token)
				encoded, err := json.Marshal(child)
				if err != nil {
					t.Fatal(err)
				}
				object[item.parent] = encoded
			}
			changed, err := json.Marshal(object)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := decodeHistoryPreparationInput(session.compiler.ctx, changed); !errors.Is(err, ErrConfiguration) {
				t.Fatalf("original input error contract was not reached: %v", err)
			}
			selection, err := state.decodeOwnedSelection(session, changed)
			if selection == nil || selection.validated || !errors.Is(err, ErrConfiguration) || !errors.Is(err, item.cause) {
				t.Fatalf("typed input refusal=%v; expected configuration and %v", err, item.cause)
			}
			if errors.Is(err, compilermemory.ErrUnavailable) || errors.Is(err, compilermemory.ErrClosed) ||
				errors.Is(err, compilermemory.ErrRange) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("typed input scenario was masked by resource/lifetime refusal: %v", err)
			}
			if _, err := selection.field("format"); err != ErrConfiguration {
				t.Fatalf("invalid typed input supplied a usable field: %v", err)
			}
		})
	}
}

func TestProviderHistoryNativeOwnedSelectionTypedWidthBoundaries(t *testing.T) {
	for _, item := range []struct {
		name, token string
		schema      reflect.Type
		want        error
	}{
		{"signed-minimum", "-128", reflect.TypeFor[int8](), nil},
		{"signed-maximum", "127", reflect.TypeFor[int8](), nil},
		{"signed-low-overflow", "-129", reflect.TypeFor[int8](), ErrConfiguration},
		{"signed-high-overflow", "128", reflect.TypeFor[int8](), ErrConfiguration},
		{"unsigned-maximum", "255", reflect.TypeFor[uint8](), nil},
		{"unsigned-overflow", "256", reflect.TypeFor[uint8](), ErrConfiguration},
		{"int64-minimum", "-9223372036854775808", reflect.TypeFor[int64](), nil},
		{"int64-maximum", "9223372036854775807", reflect.TypeFor[int64](), nil},
		{"uint64-maximum", "18446744073709551615", reflect.TypeFor[uint64](), nil},
		{"boolean", "true", reflect.TypeFor[bool](), nil},
		{"wrong-boolean-kind", `"true"`, reflect.TypeFor[bool](), compilermemory.ErrJSONType},
	} {
		t.Run(item.name, func(t *testing.T) {
			// This helper acquires a bounded native JSON document. Policy
			// validation is deliberately not invoked for these typed leaves.
			reader, _, err := nativePolicyTestReader(t, []byte(item.token), len(item.token))
			if err != nil {
				t.Fatalf("native scalar setup: %v", err)
			}
			if err := validateNativeHistorySelectionValue(reader.root, item.schema, 0); err != item.want {
				t.Fatalf("typed leaf error=%v; expected exact %v", err, item.want)
			}
		})
	}
}

func TestProviderHistoryNativeOwnedSelectionTypedReadsKeepLifetimeCauses(t *testing.T) {
	for _, name := range []string{"closed", "canceled"} {
		t.Run(name, func(t *testing.T) {
			reader, cancel, err := nativePolicyTestReader(t, []byte(`"value"`), 16)
			if err != nil {
				t.Fatalf("native lifetime setup: %v", err)
			}
			want := compilermemory.ErrClosed
			if name == "closed" {
				if err := reader.source.Close(); err != nil {
					t.Fatal(err)
				}
			} else {
				cancel()
				want = context.Canceled
			}
			err = validateNativeHistorySelectionValue(reader.root, reflect.TypeFor[string](), 0)
			if !errors.Is(err, want) {
				t.Fatalf("typed read lost lifetime cause=%v; expected %v", err, want)
			}
		})
	}
}

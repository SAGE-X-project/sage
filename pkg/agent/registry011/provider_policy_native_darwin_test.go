//go:build darwin

package registry011

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/registry011/internal/compilermemory"
)

// These are bounded owned-buffer runtime cases, not full compiler admission.
// The fixture supplies DATA through its existing public API and never mints D.
func nativePolicyTestReader(t *testing.T, raw []byte, numberCapacity int) (*nativeProviderPolicyReader, context.CancelFunc, error) {
	t.Helper()
	action, cancelAction := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancelAction)
	cleanup, cancelCleanup := context.WithTimeout(context.Background(), 35*time.Second)
	t.Cleanup(cancelCleanup)
	limits := compilermemory.JSONValueDepthLimits{MaximumValueDepth: 64, MaximumNodes: len(raw)}
	capacity, err := compilermemory.JSONValueCapacity(len(raw), limits)
	if err != nil {
		t.Fatalf("native policy setup capacity: %v", err)
		return nil, cancelAction, err
	}
	acquire := func(capacity int) (*compilermemory.Region, error) {
		region, err := compilermemory.Acquire(action, cleanup, capacity)
		if region != nil {
			t.Cleanup(func() {
				if err := region.Close(); err != nil {
					t.Error(err)
				}
			})
		}
		return region, err
	}
	source, err := acquire(capacity)
	if err != nil {
		t.Fatalf("native policy source acquisition: %v", err)
		return nil, cancelAction, err
	}
	numbers, err := acquire(numberCapacity)
	if err != nil {
		t.Fatalf("native policy numeric acquisition: %v", err)
		return nil, cancelAction, err
	}
	input, err := source.Store(raw)
	if err != nil {
		t.Fatalf("native policy setup input: %v", err)
		return nil, cancelAction, err
	}
	document, err := source.ParseJSONAtValueDepth(input, limits)
	if err != nil {
		t.Fatalf("native policy setup JSON: %v", err)
		return nil, cancelAction, err
	}
	reader, err := newNativeProviderPolicyReader(source, numbers, document, input, len(raw))
	return reader, cancelAction, err
}

func TestProviderPolicyNativeSharesExactSemanticContract(t *testing.T) {
	for _, item := range []struct {
		name string
		edit func(map[string]any)
		want bool
	}{
		{"original", func(map[string]any) {}, true},
		{"quoted-clock", func(m map[string]any) { m["clock"].(map[string]any)["id"] = "quoted\"\\clock" }, true},
		{"maximum-clock-id", func(m map[string]any) { m["clock"].(map[string]any)["id"] = strings.Repeat("x", 256) }, true},
		{"maximum-unsigned-scalar", func(m map[string]any) { m["limits"].(map[string]any)["retained_bytes"] = "18446744073709551615" }, true},
		{"missing-clock", func(m map[string]any) { delete(m, "clock") }, false},
		{"unknown-root", func(m map[string]any) { m["unexpected"] = "value" }, false},
		{"non-ascii-clock", func(m map[string]any) { m["clock"].(map[string]any)["id"] = "시계" }, false},
		{"wrong-numeric-type", func(m map[string]any) { m["limits"].(map[string]any)["retained_bytes"] = json.Number("1") }, false},
		{"noncanonical-decimal", func(m map[string]any) { m["limits"].(map[string]any)["retained_bytes"] = "01" }, false},
		{"unsigned-overflow", func(m map[string]any) { m["limits"].(map[string]any)["retained_bytes"] = "18446744073709551616" }, false},
		{"zero-positive-limit", func(m map[string]any) { m["limits"].(map[string]any)["retained_bytes"] = "0" }, false},
		{"short-carriers", func(m map[string]any) { m["carrier_formats"] = []any{"protected-legacy"} }, false},
	} {
		t.Run(item.name, func(t *testing.T) {
			object := providerPolicyTestObject()
			item.edit(object)
			raw := providerPolicyTestBytes(t, object)
			_, originalErr := ParseProviderPolicyCandidate(raw, 64<<10)
			if (originalErr == nil) != item.want {
				t.Fatalf("original fixture expectation changed: %v", originalErr)
			}
			reader, _, err := nativePolicyTestReader(t, raw, len(raw))
			if err != nil {
				t.Fatalf("native semantic precondition: %v", err)
			}
			err = reader.validate()
			if item.want {
				if err != nil {
					t.Fatalf("native semantic acceptance: %v", err)
				}
			} else if err != ErrProviderPolicyCandidate {
				t.Fatalf("native semantic refusal=%v; expected exact policy rejection", err)
			}
		})
	}
}

func TestProviderPolicyNativeRejectsNoncanonicalOriginalBytes(t *testing.T) {
	object := providerPolicyTestObject()
	object["clock"].(map[string]any)["id"] = "a"
	raw := providerPolicyTestBytes(t, object)
	if _, err := ParseProviderPolicyCandidate(raw, 64<<10); err != nil {
		t.Fatalf("canonical precondition: %v", err)
	}
	var values map[string]json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil {
		t.Fatal(err)
	}
	keys := make([]string, 0, len(values))
	for name := range values {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	keys[0], keys[1] = keys[1], keys[0]
	var reordered bytes.Buffer
	reordered.WriteByte('{')
	for index, name := range keys {
		if index != 0 {
			reordered.WriteByte(',')
		}
		key, err := json.Marshal(name)
		if err != nil {
			t.Fatal(err)
		}
		reordered.Write(key)
		reordered.WriteByte(':')
		reordered.Write(values[name])
	}
	reordered.WriteByte('}')
	for name, changed := range map[string][]byte{
		"leading-space": append([]byte(" "), raw...),
		"trailing-newline": append(bytes.Clone(raw), '\n'),
		"interior-space": bytes.Replace(raw, []byte(":"), []byte(": "), 1),
		"escaped-key": bytes.Replace(raw, []byte(`"format"`), []byte(`"\u0066ormat"`), 1),
		"escaped-clock": bytes.Replace(raw, []byte(`"id":"a"`), []byte(`"id":"\u0061"`), 1),
		"escaped-slash": bytes.Replace(raw, []byte("https://"), []byte(`https:\/\/`), 1),
		"reordered-members": reordered.Bytes(),
	} {
		t.Run(name, func(t *testing.T) {
			if bytes.Equal(changed, raw) {
				t.Fatal("noncanonical fixture did not change the bytes")
			}
			if _, err := ParseProviderPolicyCandidate(changed, 64<<10); err == nil {
				t.Fatal("original parser accepted noncanonical input")
			}
			reader, _, err := nativePolicyTestReader(t, changed, len(changed))
			if name != "leading-space" && name != "trailing-newline" && err != nil {
				t.Fatalf("native canonical precondition: %v", err)
			}
			if err == nil {
				err = reader.validate()
			}
			if err != ErrProviderPolicyCandidate {
				t.Fatalf("native canonical refusal=%v", err)
			}
		})
	}
}

func TestProviderPolicyNativeUsesOwnedDecodedNumbersAndOneAttempt(t *testing.T) {
	object := providerPolicyTestObject()
	raw := providerPolicyTestBytes(t, object)
	reader, _, err := nativePolicyTestReader(t, raw, len(raw))
	if err != nil {
		t.Fatal(err)
	}
	if err := reader.validate(); err != nil {
		t.Fatal(err)
	}
	if err := reader.validate(); !errors.Is(err, ErrConfiguration) {
		t.Fatalf("reader reused its validation attempt: %v", err)
	}
	if err := reader.source.Close(); err != nil {
		t.Fatal(err)
	}
	snapshot, err := reader.numbers.SealSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	want := object["limits"].(map[string]any)["live_disk_bytes"].(string)
	field, err := reader.numbers.Slice(snapshot, 0, len(want))
	if err != nil {
		t.Fatal(err)
	}
	actual := make([]byte, len(want))
	if err := reader.numbers.Read(field, actual); err != nil || string(actual) != want {
		t.Fatalf("decoded numeric bytes=%q, error=%v; expected=%q", actual, err, want)
	}
}

func TestProviderPolicyNativeRefusesLostLifetimeAndCapacity(t *testing.T) {
	raw := providerPolicyTestBytes(t, providerPolicyTestObject())
	for _, name := range []string{"source-closed", "numbers-closed", "canceled", "full-number-region"} {
		t.Run(name, func(t *testing.T) {
			capacity := len(raw)
			if name == "full-number-region" {
				capacity = 1
			}
			reader, cancel, err := nativePolicyTestReader(t, raw, capacity)
			if err != nil {
				t.Fatal(err)
			}
			var expected error
			switch name {
			case "source-closed":
				expected = compilermemory.ErrClosed
				if err := reader.source.Close(); err != nil {
					t.Fatal(err)
				}
			case "numbers-closed":
				expected = compilermemory.ErrClosed
				if err := reader.numbers.Close(); err != nil {
					t.Fatal(err)
				}
			case "canceled":
				expected = context.Canceled
				cancel()
			case "full-number-region":
				expected = compilermemory.ErrRange
				if _, err := reader.numbers.Store([]byte{1}); err != nil {
					t.Fatal(err)
				}
			}
			if err := reader.validate(); !errors.Is(err, expected) {
				t.Fatalf("refusal=%v; expected %v", err, expected)
			}
		})
	}
}

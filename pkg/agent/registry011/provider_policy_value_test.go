package registry011

import (
	"strings"
	"testing"
)

func TestProviderPolicyValueKeepsLegacyReaderBoundaries(t *testing.T) {
	invalid := legacyProviderPolicyValue{}
	if invalid.object("key") || invalid.member("missing").equals("") {
		t.Fatal("missing values became an object or string")
	}
	if length, ok := invalid.arrayLength(); ok || length != 0 {
		t.Fatalf("missing array=%d, %v", length, ok)
	}
	array := legacyProviderPolicyValue{[]any{"first", "second"}}
	if length, ok := array.arrayLength(); !ok || length != 2 || !array.element(1).equals("second") {
		t.Fatal("array reader changed its selected element")
	}
	for _, index := range []int{-1, 2} {
		if array.element(index).equals("") || invalid.element(index).equals("") {
			t.Fatal("out-of-range element became a string")
		}
	}
	var destination [8]byte
	text := legacyProviderPolicyValue{"text"}
	if n, ok := text.text(destination[:]); !ok || n != 4 || string(destination[:n]) != "text" {
		t.Fatalf("text reader=%d, %v, %q", n, ok, destination)
	}
	if _, ok := text.text(destination[:3]); ok {
		t.Fatal("text reader accepted insufficient space")
	}
	if _, ok := invalid.text(destination[:]); ok {
		t.Fatal("non-string became text")
	}
	if n, ok := (legacyProviderPolicyValue{"0"}).decimal(false); !ok || n != 0 {
		t.Fatalf("non-positive counter contract changed: %d, %v", n, ok)
	}
	if _, ok := (legacyProviderPolicyValue{"0"}).decimal(true); ok {
		t.Fatal("positive counter accepted zero")
	}
}

func TestProviderPolicyValueDigestKeepsOriginalEncoding(t *testing.T) {
	for _, value := range []any{
		nil, 1, "", "0x", "0x" + strings.Repeat("a", 63), "0x" + strings.Repeat("a", 65),
		"0X" + strings.Repeat("a", 64), "0x" + strings.Repeat("A", 64),
		"0x" + strings.Repeat("g", 64), "0x" + strings.Repeat("0", 64), "0x" + strings.Repeat("f", 64),
	} {
		expected := providerCandidateDigest(value)
		if actual := providerPolicyDigestValue(legacyProviderPolicyValue{value}); actual != expected {
			t.Fatalf("digest encoding changed for %q: %v, expected %v", value, actual, expected)
		}
	}
}

func TestProviderPolicyValueSharedContractRetainsRefusals(t *testing.T) {
	if !providerPolicyFieldsValid(legacyProviderPolicyValue{providerPolicyTestObject()}) {
		t.Fatal("original positive fixture does not satisfy the shared contract")
	}
	for _, item := range []struct {
		name string
		edit func(map[string]any)
	}{
		{"missing-clock", func(m map[string]any) { delete(m, "clock") }},
		{"unknown-root", func(m map[string]any) { m["unknown"] = "value" }},
		{"wrong-root-field-type", func(m map[string]any) { m["instance"] = []any{} }},
		{"short-carrier-array", func(m map[string]any) { m["carrier_formats"] = []any{"protected-legacy"} }},
		{"clock-control-byte", func(m map[string]any) { m["clock"].(map[string]any)["id"] = "clock\n" }},
		{"clock-too-long", func(m map[string]any) { m["clock"].(map[string]any)["id"] = strings.Repeat("x", 257) }},
		{"zero-positive-limit", func(m map[string]any) { m["limits"].(map[string]any)["retained_bytes"] = "0" }},
		{"operation-request-ceiling", func(m map[string]any) { m["limits"].(map[string]any)["operation"].(map[string]any)["requests"] = "65537" }},
		{"decreasing-segment", func(m map[string]any) {
			limits := m["limits"].(map[string]any)
			limits["operation"].(map[string]any)["wire_bytes"] = "2"
			limits["segment"].(map[string]any)["wire_bytes"] = "1"
		}},
	} {
		t.Run(item.name, func(t *testing.T) {
			object := providerPolicyTestObject()
			item.edit(object)
			if providerPolicyFieldsValid(legacyProviderPolicyValue{object}) {
				t.Fatal("shared contract accepted the invalid policy")
			}
		})
	}
}

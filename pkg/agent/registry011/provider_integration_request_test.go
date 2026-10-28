package registry011

import (
	"strings"
	"testing"
)

// Finite JSON values only: these units exercise refusal before serialization,
// admission or dispatch. No peer, callback or runtime attack is constructed.
func TestProviderRequestDataKeepsDepthNodeAndByteBudgets(t *testing.T) {
	for _, value := range []any{true, false, 0, "normal", []any{"0x0", false}, map[string]string{"to": SepoliaRegistryAddress}, map[string]bool{"requireCanonical": true}, map[string]any{"tracer": "callTracer", "reexec": 0}} {
		if !providerRequestData(value, 0) {
			t.Fatal("closed normal parameter refused")
		}
	}
	for _, value := range []any{nil, 1, -1, uint64(0), struct{ Name string }{"inert"}, []string{"inert"}, string([]byte{0xff}), strings.Repeat("a", 8193), []any{strings.Repeat("a", 4097), strings.Repeat("b", 4096)}} {
		if providerRequestData(value, 0) {
			t.Fatal("unsupported or over-budget data accepted")
		}
	}
	largeSlice := make([]any, 17)
	largeStringMap, largeBoolMap, largeAnyMap := map[string]string{}, map[string]bool{}, map[string]any{}
	for i := 0; i < 17; i++ {
		key := strings.Repeat("a", i+1)
		largeSlice[i] = "normal"
		largeStringMap[key], largeBoolMap[key], largeAnyMap[key] = "normal", true, "normal"
	}
	for _, value := range []any{largeSlice, largeStringMap, largeBoolMap, largeAnyMap, map[string]string{string([]byte{0xff}): "normal"}, map[string]string{"normal": string([]byte{0xff})}, map[string]bool{string([]byte{0xff}): true}, map[string]any{string([]byte{0xff}): true}, map[string]any{"normal": nil}} {
		if providerRequestData(value, 0) {
			t.Fatal("collection bypassed finite data boundary")
		}
	}
	var deep any = "normal"
	for i := 0; i < 9; i++ {
		deep = []any{deep}
	}
	if providerRequestData(deep, 0) {
		t.Fatal("nested data exceeded depth ceiling")
	}
	leaf := make([]any, 16)
	for i := range leaf {
		leaf[i] = []any{true, false}
	}
	nodes := make([]any, 16)
	for i := range nodes {
		nodes[i] = leaf
	}
	if providerRequestData(nodes, 0) {
		t.Fatal("individually bounded lists exceeded common node ceiling")
	}
}

func TestProviderRequestFirstFinalizedBindsEverySerializedPart(t *testing.T) {
	original, err := newProviderIntegrationRequest("eth_getBlockByNumber", []any{"finalized", false})
	if err != nil || !original.firstFinalized() {
		t.Fatal("closed first request", err)
	}
	var absent *providerIntegrationRequest
	if absent.firstFinalized() {
		t.Fatal("absent request selected a first operation")
	}
	for _, part := range []string{"method", "params", "body", "head", "wire"} {
		t.Run(part, func(t *testing.T) {
			copy := *original
			switch part {
			case "method":
				copy.method = "eth_chainId"
			case "params":
				copy.params = `["latest",false]`
			case "body":
				copy.body += " "
			case "head":
				copy.head += " "
			case "wire":
				copy.wire += " "
			}
			if copy.firstFinalized() {
				t.Fatal("different serialized bytes inherited first-operation selection")
			}
		})
	}
	for _, value := range []any{true, "normal", map[string]any{"normal": true}, []any{strings.Repeat("a", 8190)}} {
		if _, err := newProviderIntegrationRequest("eth_getBlockByNumber", value); err == nil {
			t.Fatal("non-array or encoded-over-limit params accepted")
		}
	}
}

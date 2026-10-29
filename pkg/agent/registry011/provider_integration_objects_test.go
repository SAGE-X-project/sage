package registry011

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
)

func TestProviderIntegrationObjectsBindKindDomainAndBytes(t *testing.T) {
	for _, kind := range []string{"request", "params", "response", "tls", "setup", "setup-commit", "enrollment", "admission", "ticket", "consumption", "settlement", "ledger-node", "budget", "exchange", "wire-binding", "bootstrap", "pin-input", "pin", "segment", "batch", "checkpoint", "ledger-page", "ledger-seal", "index-empty", "index-mutation", "frontier", "verification-receipt"} {
		t.Run(kind, func(t *testing.T) {
			domain, known := providerIntegrationDomain(kind)
			raw := []byte("hello")
			if domain != "" {
				raw = providerIntegrationJSON(map[string]string{"format": domain})
			}
			object, err := newProviderIntegrationObject(kind, raw, 4096)
			if !known || err != nil || !lowerHash(object.digest) || object.kind != kind || object.raw != string(raw) {
				t.Fatal("typed content identity", err)
			}
			if domain == "" && object.digest != "0x2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824" {
				t.Fatal("ordinary SHA-256 changed")
			}
			if kind == "bootstrap" && object.digest != "0x2a41b89267d86cdbeac9446ba900ff39f37d122c861cdcf03fe0c7860b964b5d" {
				t.Fatal("independent domain-separated fixture changed")
			}
			raw[0] ^= 1
			if object.raw == string(raw) {
				t.Fatal("mutable input escaped into retained object")
			}
		})
	}
	if digest, known := providerIntegrationObjectDigest("unknown", []byte("hello")); known || digest != "" {
		t.Fatal("unknown object kind")
	}
	if providerIntegrationJSON(make(chan int)) != nil {
		t.Fatal("failed metadata encoding created bytes")
	}
}

func TestProviderFrontierInventoryKeepsV1AndV2DomainsAndCapsSeparate(t *testing.T) {
	objects := make([]providerIntegrationObject, 0, 256)
	for n := 0; n < 256; n++ {
		object, err := newProviderIntegrationObject("response", []byte(strconv.Itoa(n)), 4096)
		if err != nil {
			t.Fatal(err)
		}
		objects = append(objects, object)
	}
	first, err := newProviderIntegrationInventory("", nil, 64, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	second, err := newProviderFrontierInventory("", nil, 256, 64<<20)
	if err != nil || first.digest == second.digest || first.raw == second.raw {
		t.Fatal("empty V2 inventory aliased V1", err)
	}
	inv, err := newProviderFrontierInventory(second.digest, objects, 256, 64<<20)
	if err != nil || len(inv.entries) != 256 {
		t.Fatal("complete bounded V2 inventory", err)
	}
	parsed, err := parseProviderFrontierInventory([]byte(inv.raw), second.digest, 256, 4096, 64<<20)
	if err != nil || parsed.digest != inv.digest {
		t.Fatal("V2 exact readback", err)
	}
	if _, err := newProviderIntegrationInventory(first.digest, objects[:65], 65, 64<<20); err == nil {
		t.Fatal("V1 cap increased")
	}
	if _, err := parseProviderIntegrationInventory([]byte(inv.raw), second.digest, 64, 4096, 64<<20); err == nil {
		t.Fatal("V2 bytes accepted as V1")
	}
	if _, err := parseProviderFrontierInventory([]byte(first.raw), "", 256, 4096, 64<<20); err == nil {
		t.Fatal("V1 bytes relabeled V2")
	}
	if _, err := newProviderFrontierInventory(second.digest, objects, 257, 64<<20); err == nil {
		t.Fatal("unbounded V2 entries")
	}
	if _, err := newProviderInventory(3, "", nil, 64, 4096); err == nil {
		t.Fatal("unknown inventory version")
	}
	if _, err := parseProviderInventory(3, []byte(first.raw), "", 64, 4096, 4096); err == nil {
		t.Fatal("unknown parser version")
	}
}

func TestProviderIntegrationObjectRefusalsAreBounded(t *testing.T) {
	for _, fixture := range []struct {
		kind, raw string
		maximum   uint64
	}{
		{"response", "a", 0}, {"response", "a", 32<<20 + 1}, {"response", "", 10}, {"response", "four", 3},
		{"unknown", "a", 10}, {"bootstrap", `null`, 4096}, {"bootstrap", `{}`, 4096},
		{"bootstrap", `{"format":"wrong"}`, 4096}, {"bootstrap", `{ "format":"sage-provider-start-v2"}`, 4096},
		{"request", strings.Repeat("a", 16<<10+1), 32 << 20}, {"params", strings.Repeat("a", 8<<10+1), 32 << 20},
	} {
		if _, err := newProviderIntegrationObject(fixture.kind, []byte(fixture.raw), fixture.maximum); err == nil {
			t.Fatal("unsupported object")
		}
	}
}

func TestProviderIntegrationInventoryHasExactPreviousAndSortedClosure(t *testing.T) {
	first, _ := newProviderIntegrationObject("response", []byte("a"), 4096)
	second, _ := newProviderIntegrationObject("request", []byte("b"), 4096)
	empty, err := newProviderIntegrationInventory("", nil, 8, 4096)
	if err != nil || empty.payloadBytes != 0 || empty.previous != "" {
		t.Fatal("initial inventory")
	}
	objects := []providerIntegrationObject{first, second}
	inv, err := newProviderIntegrationInventory(empty.digest, objects, 8, 4096)
	if err != nil || inv.payloadBytes != 2 || inv.entries[0].kind != "request" {
		t.Fatal("exact sorted inventory", err)
	}
	parsed, err := parseProviderIntegrationInventory([]byte(inv.raw), empty.digest, 8, 4096, 4096)
	if err != nil || parsed.digest != inv.digest || !bytes.Equal([]byte(parsed.raw), []byte(inv.raw)) {
		t.Fatal("inventory round trip", err)
	}
	if _, err := parseProviderIntegrationInventory([]byte(empty.raw), "", 8, 4096, 4096); err != nil {
		t.Fatal("authenticated initial empty shape", err)
	}
	for _, fixture := range []struct {
		previous string
		objects  []providerIntegrationObject
		entries  int
		payload  uint64
	}{
		{"invalid", objects, 8, 4096}, {empty.digest, nil, 8, 4096}, {"", objects, 0, 4096},
		{"", objects, 65, 4096}, {"", objects, 1, 4096}, {"", objects, 8, 0}, {"", objects, 8, 64<<20 + 1},
		{"", objects, 8, 1}, {"", []providerIntegrationObject{first, first}, 8, 4096},
		{"", []providerIntegrationObject{{"response", first.digest, "changed"}}, 8, 4096},
		{"", []providerIntegrationObject{{"response", first.digest, ""}}, 8, 4096},
	} {
		if _, err := newProviderIntegrationInventory(fixture.previous, fixture.objects, fixture.entries, fixture.payload); err == nil {
			t.Fatal("unsupported inventory")
		}
	}
	var decoded providerIntegrationInventory
	if json.Unmarshal([]byte(`{}`), &decoded) == nil {
		t.Fatal("JSON reconstructed a private inventory")
	}
	for _, mutate := range []string{"extra", "previous", "missing-entries", "unknown-kind", "bad-digest", "zero-bytes", "entry-extra", "duplicate", "descending", "noncanonical"} {
		t.Run(mutate, func(t *testing.T) {
			m := contentJSON([]byte(inv.raw)).(map[string]any)
			entries := m["entries"].([]any)
			e := entries[0].(map[string]any)
			switch mutate {
			case "extra":
				m["extra"] = true
			case "previous":
				m["previous"] = nil
			case "missing-entries":
				delete(m, "entries")
			case "unknown-kind":
				e["kind"] = "unknown"
			case "bad-digest":
				e["digest"] = "invalid"
			case "zero-bytes":
				e["bytes"] = "0"
			case "entry-extra":
				e["extra"] = true
			case "duplicate":
				entries[1] = entries[0]
			case "descending":
				entries[0], entries[1] = entries[1], entries[0]
			}
			raw := providerIntegrationJSON(m)
			if mutate == "noncanonical" {
				raw = append([]byte(" "), raw...)
			}
			if _, err := parseProviderIntegrationInventory(raw, empty.digest, 8, 4096, 4096); err == nil {
				t.Fatal("ambiguous retained inventory")
			}
		})
	}
}

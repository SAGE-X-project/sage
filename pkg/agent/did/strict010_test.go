package did

import (
	"errors"
	"strings"
	"testing"
)

func TestParseDID010(t *testing.T) {
	address := "0x" + strings.Repeat("a", 40)
	for _, tc := range []struct {
		input                string
		kind, locator, agent string
	}{
		{"did:sage:web:agents.example.com:billing-bot", "web", "agents.example.com", "billing-bot"},
		{"did:sage:web:xn--bcher-kva.example:agent_1", "web", "xn--bcher-kva.example", "agent_1"},
		{"did:sage:eip155:11155111:" + address + ":0x1234abcd", "eip155", "11155111:" + address, "0x1234abcd"},
	} {
		got, err := ParseDID010(tc.input)
		if err != nil || got != (DID010{tc.kind, tc.locator, tc.agent}) {
			t.Fatalf("%q: %#v, %v", tc.input, got, err)
		}
	}
	for _, input := range []string{
		"DID:sage:web:agents.example.com:a",
		"did:SAGE:web:agents.example.com:a",
		"did:sage:web:Agents.example.com:a",
		"did:sage:web:127.0.0.1:a",
		"did:sage:web:-bad.example:a",
		"did:sage:web:bad-.example:a",
		"did:sage:web:example.com.:a",
		"did:sage:web:example.com:..",
		"did:sage:web:example.com:a?x=1",
		"did:sage:web:example.com:a#key",
		"did:sage:web:example.com:a:extra",
		"did:sage:eip155:0:" + address + ":a",
		"did:sage:eip155:01:" + address + ":a",
		"did:sage:eip155:1:0x" + strings.Repeat("A", 40) + ":a",
		"did:sage:ethereum:0xabc",
		"did:sage:web:example.com:" + strings.Repeat("a", 65),
	} {
		if _, err := ParseDID010(input); !errors.Is(err, ErrMalformedDID010) &&
			!errors.Is(err, ErrUnknownDIDKind010) {
			t.Fatalf("accepted %q: %v", input, err)
		}
	}
	for _, input := range []string{
		"did:sage:solana:abc:agent", "did:sage:other:loc:agent",
	} {
		if _, err := ParseDID010(input); !errors.Is(err, ErrUnknownDIDKind010) {
			t.Fatalf("unknown kind %q: %v", input, err)
		}
	}
	// The explicit 0.10.0 entry point does not alter legacy consumer behavior.
	if _, _, err := ParseDID("did:sage:ETH:0xabc"); err != nil {
		t.Fatalf("legacy parser changed: %v", err)
	}
}

func TestParseDIDURL010(t *testing.T) {
	url := "did:sage:web:agents.example.com:billing-bot#key-1"
	got, err := ParseDIDURL010(url)
	if err != nil || got.KeyID != "key-1" || got.DID.AgentID != "billing-bot" {
		t.Fatalf("valid key URL: %#v, %v", got, err)
	}
	for _, input := range []string{
		"did:sage:web:agents.example.com:billing-bot",
		url + "#other",
		"did:sage:web:agents.example.com:billing-bot#",
		"did:sage:web:agents.example.com:billing-bot#key.1",
		"did:sage:web:agents.example.com:billing-bot#" + strings.Repeat("x", 33),
		url + "/path",
	} {
		if _, err := ParseDIDURL010(input); !errors.Is(err, ErrMalformedDID010) {
			t.Fatalf("accepted key URL %q: %v", input, err)
		}
	}
}

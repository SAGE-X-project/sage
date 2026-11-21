// SAGE - Secure Agent Guarantee Engine
// Copyright (C) 2025 SAGE-X-project
//
// This file is part of SAGE.
//
// SAGE is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// SAGE is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with SAGE. If not, see <https://www.gnu.org/licenses/>.

package suites

import (
	"errors"
	"strings"

	"github.com/sage-x-project/sage/core/did"
	"github.com/sage-x-project/sage/core/obs"
	"github.com/sage-x-project/sage/core/vectors"
)

// DIDDescription is the description of the did suite file (unchanged from
// draft.1; the card cases join with W-040).
const DIDDescription = "did:sage method: did:sage:<chain>:<identifier> with chain in {ethereum, solana} (aliases eth, sol, case-insensitive), key proof-of-possession over SHA-256(\"SAGE-PoP:\" || DID || \":\" || hex(key)), and the A2A agent-card proof (JCS + Ed25519/secp256k1, base58 proofValue)."

// DID returns the did suite cases of 06: the published parse,
// chain-aliases and pop-* vectors, canonical-compare, the errata §11 parse
// additions as reject vectors (E06-02, E06-03), and the errata §11 proof and
// key-selection cases (E06-10, E06-11, §3).
func DID() (vectors.Suite, error) {
	valid := []any{
		"did:sage:ethereum:0x1111111111111111111111111111111111111111",
		"did:sage:ethereum:0x1111111111111111111111111111111111111111:42",
		"did:sage:solana:7Zn3QwXk9vQ2yP1sJ4cR8tL6mN5bV3xA2dF9gH1kJ8Lm",
		"did:sage:ETHEREUM:agent-one",
		"did:sage:eth:0xabc",
	}
	invalid := []any{"", "did:sage", "did:sage:ethereum", "did:web:example.com", "did:sage:polkadot:abc", "sage:ethereum:0xabc", "did:sage::0xabc"}
	cases := []vectors.Case{
		{
			Name: "parse", Mode: vectors.ModeDeterministic,
			Description: "ParseDID splits chain and identifier; the identifier keeps any further colons. Invalid inputs fail.",
			Input:       map[string]any{"valid": valid, "invalid": invalid},
			Produce:     produceParse,
		},
		{
			Name: "chain-aliases", Mode: vectors.ModeDeterministic,
			Description: "ParseChain normalises case and whitespace and accepts the eth/sol aliases.",
			Input:       map[string]any{"names": []any{"ethereum", "ETH", " Solana ", "sol"}},
			Produce:     produceChainAliases,
		},
		{
			Name: "canonical-compare", Mode: vectors.ModeDeterministic,
			Description: "E06-04: DIDs are compared in canonical form; did:sage:eth:0xabc equals did:sage:ethereum:0xabc (for ExpectedDID and every other comparison), while the as-sent forms differ.",
			Input:       map[string]any{"a": "did:sage:eth:0xabc", "b": "did:sage:ethereum:0xabc"},
			Produce:     produceCanonicalCompare,
		},
	}
	long := "did:sage:ethereum:" + strings.Repeat("a", did.MaxDIDBytes-len("did:sage:ethereum:")+1)
	for _, r := range []struct{ name, desc, in string }{
		{"parse-reject-percent", "a percent-encoded byte", "did:sage:ethereum:0x11%20"},
		{"parse-reject-fragment", "a DID URL with a fragment", "did:sage:ethereum:a#b"},
		{"parse-reject-query", "a query", "did:sage:ethereum:a?b"},
		{"parse-reject-slash", "a path", "did:sage:ethereum:a/b"},
		{"parse-reject-pipe", "a pipe", "did:sage:ethereum:a|b"},
		{"parse-reject-space", "a space", "did:sage:ethereum:a b"},
		{"parse-reject-non-ascii", "a non-ASCII letter", "did:sage:ethereum:ä"},
		{"parse-reject-empty-identifier", "an identifier of two empty segments", "did:sage:ethereum::"},
		{"parse-reject-empty-segment", "an empty segment between colons", "did:sage:ethereum:a::b"},
		{"parse-reject-trailing-colon", "an empty last segment", "did:sage:ethereum:a:"},
		{"parse-reject-257-bytes", "a DID of 257 bytes", long},
		{"parse-reject-method-upper", "an upper-case \"DID\" scheme", "DID:sage:ethereum:a"},
		{"parse-reject-method-name-upper", "an upper-case method name", "did:SAGE:ethereum:a"},
	} {
		cases = append(cases, vectors.Case{
			Name: r.name, Mode: vectors.ModeReject,
			Description: rule(r.name) + ": the DID parser rejects " + r.desc + ".",
			Input:       map[string]any{"did": r.in}, Reject: rejectDID, Reason: obs.ReasonIdentifier,
		})
	}
	pop, err := didPoP()
	if err != nil {
		return vectors.Suite{}, err
	}
	return vectors.Suite{Name: "did", Description: DIDDescription, Cases: append(cases, pop...)}, nil
}

// rule names the errata item a parse reject vector tests.
func rule(name string) string {
	if name == "parse-reject-257-bytes" {
		return "E06-03"
	}
	return "E06-02"
}

func produceParse(in map[string]any) (map[string]any, error) {
	valid, err := stringList(in, "valid")
	if err != nil {
		return nil, err
	}
	invalid, err := stringList(in, "invalid")
	if err != nil {
		return nil, err
	}
	parsed := make([]any, 0, len(valid))
	for _, s := range valid {
		d, err := did.Parse(s)
		if err != nil {
			return nil, errors.New("a valid DID was rejected")
		}
		parsed = append(parsed, map[string]any{"chain": d.Chain().String(), "did": d.AsSent(), "identifier": d.Identifier()})
	}
	rejected := make([]any, 0, len(invalid))
	for _, s := range invalid {
		if _, err := did.Parse(s); err == nil {
			return nil, errors.New("an invalid DID was accepted")
		}
		rejected = append(rejected, s)
	}
	return map[string]any{"parsed": parsed, "rejected": rejected}, nil
}

func produceChainAliases(in map[string]any) (map[string]any, error) {
	names, err := stringList(in, "names")
	if err != nil {
		return nil, err
	}
	chains := map[string]any{}
	for _, n := range names {
		c, err := did.ParseChain(n)
		if err != nil {
			return nil, err
		}
		chains[n] = c.String()
	}
	g, err := did.New(did.ChainEthereum, "0xabc")
	if err != nil {
		return nil, err
	}
	return map[string]any{"chains": chains, "generated": g.String()}, nil
}

func produceCanonicalCompare(in map[string]any) (map[string]any, error) {
	a, err := parseDIDField(in, "a")
	if err != nil {
		return nil, err
	}
	b, err := parseDIDField(in, "b")
	if err != nil {
		return nil, err
	}
	return map[string]any{"equal": a.Equal(b), "canonical_a": a.String(), "canonical_b": b.String()}, nil
}

func rejectDID(in map[string]any) error {
	if len(in) != 1 {
		return errors.New("input must hold exactly did")
	}
	_, err := parseDIDField(in, "did")
	return err
}

func parseDIDField(in map[string]any, name string) (did.DID, error) {
	s, ok := in[name].(string)
	if !ok {
		return did.DID{}, errors.New(name + " must be a string")
	}
	return did.Parse(s)
}

func stringList(in map[string]any, name string) ([]string, error) {
	list, ok := in[name].([]any)
	if !ok {
		return nil, errors.New(name + " must be a list")
	}
	out := make([]string, len(list))
	for i, v := range list {
		s, ok := v.(string)
		if !ok {
			return nil, errors.New(name + " must hold strings")
		}
		out[i] = s
	}
	return out, nil
}

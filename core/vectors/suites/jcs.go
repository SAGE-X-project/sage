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

// Package suites defines the test vector suites of the SAGE core for the
// runner in package vectors. Each suite reproduces one file of sage-spec
// v1/vectors.
package suites

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/sage-x-project/sage/core/jcs"
	"github.com/sage-x-project/sage/core/obs"
	"github.com/sage-x-project/sage/core/vectors"
)

// JCSDescription is the description of the jcs suite file.
const JCSDescription = "RFC 8785 JSON Canonicalization Scheme and the strict reading of signed JSON (02-jcs.md §2). " +
	"Deterministic vectors canonicalise input.json; reject vectors are read as a signed object " +
	"(64 KiB, depth 16, 256 members or elements, no numbers) from input.json or, for bytes that a JSON string cannot hold, input.json_hex."

// The agent-card-shaped text of the published vector agent-card-like; the
// reject vectors mutate it (errata §11: one minimal mutation of a valid
// vector).
const cardJSON = `{"type":["Agent","AIAgent"],"name":"vector-agent","id":"did:sage:ethereum:0x000000000000000000000000000000000000abcd",` +
	`"publicKey":[{"type":"Ed25519VerificationKey2020","publicKeyHex":"00","publicKeyBase58":"1","id":"did:sage:ethereum:0x000000000000000000000000000000000000abcd#key-1",` +
	`"controller":"did:sage:ethereum:0x000000000000000000000000000000000000abcd"}],"@context":["https://www.w3.org/ns/did/v1"],"service":[],` +
	`"created":"2026-01-01T00:00:00Z","updated":"2026-01-01T00:00:00Z","description":""}`

// An HPKE response envelope (04-hpke.md) with its 13 members; the binary
// members hold zero bytes.
const hpkeResponseJSON = `{"v":"v1","task":"hpke/complete@v1","ctx":"ctx-vector","kid":"kid-00000000-0000-4000-8000-000000000000",` +
	`"ephS":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA","ackTagB64":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",` +
	`"ts":"2026-01-01T00:00:00Z","did":"did:sage:ethereum:0x000000000000000000000000000000000000abcd",` +
	`"infoHash":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA","exportCtxHash":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",` +
	`"enc":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA","ephC":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",` +
	`"sigB64":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"}`

// JCS returns the jcs suite: the four published canonicalisation vectors and
// the reject vectors of errata §11 (E02-02 to E02-06).
func JCS() vectors.Suite {
	return vectors.Suite{
		Name:        "jcs",
		Description: JCSDescription,
		Cases:       append(jcsCanonical(), jcsRejects()...),
	}
}

func jcsCanonical() []vectors.Case {
	c := func(name, desc, in string) vectors.Case {
		return vectors.Case{
			Name: name, Mode: vectors.ModeDeterministic, Description: desc,
			Input: map[string]any{"json": in}, Produce: canonicalise,
		}
	}
	return []vectors.Case{
		c("rfc8785-appendix-a", "RFC 8785 Appendix A sample: numbers, escaped string and literals.",
			`{"numbers": [333333333.33333329, 1E30, 4.50, 2e-3, 0.000000000000000000000000001], "string": "\u20ac$\u000F\u000aA'\u0042\u0022\u005c\\\"\/", "literals": [null, true, false]}`),
		c("rfc8785-unicode-sorting", "RFC 8785 section 3.2.3: keys sorted by UTF-16 code units.",
			"{\"\u20ac\": \"Euro Sign\", \"\\r\": \"Carriage Return\", \"\ufb33\": \"Hebrew Letter Dalet With Dagesh\", \"1\": \"One\", \"\U0001f602\": \"Smiley\", \"\\u0080\": \"Control\", \"\u00f6\": \"Latin Small Letter O With Diaeresis\", \"\\n\": \"Newline\"}"),
		c("agent-card-like", "Nested object with the shape of an A2A agent card (proof removed before signing).", cardJSON),
		c("numbers", "Integers, negative zero, exponents and fractions per ECMAScript Number::toString.",
			`{"i":10,"z":-0,"e":1e21,"f":1.0,"g":0.1,"h":100000000000000000000,"j":123456789012345680000,"k":-1.5e-7}`),
	}
}

func jcsRejects() []vectors.Case {
	text := func(s string) map[string]any { return map[string]any{"json": s} }
	raw := func(s string) map[string]any { return map[string]any{"json_hex": hex.EncodeToString([]byte(s))} }
	card := func(old, repl string) string { return strings.Replace(cardJSON, old, repl, 1) }
	r := func(name, desc string, in map[string]any, reason obs.Reason) vectors.Case {
		return vectors.Case{Name: name, Mode: vectors.ModeReject, Description: desc, Input: in, Reject: rejectSigned, Reason: reason}
	}
	desc := `"description":""`
	deep := `"service":` + strings.Repeat("[", jcs.MaxDepth) + strings.Repeat("]", jcs.MaxDepth)
	large := card(desc, `"description":"`+strings.Repeat("a", jcs.MaxSignedBytes+1-len(cardJSON))+`"`)
	return []vectors.Case{
		r("duplicate-member-top", "E02-02: agent-card-like with the top-level member name repeated.",
			text(card(`"name":"vector-agent",`, `"name":"vector-agent","name":"vector-agent",`)), obs.ReasonDuplicate),
		r("duplicate-member-nested", "E02-02: agent-card-like with publicKey[0].publicKeyHex repeated.",
			text(card(`"publicKeyHex":"00",`, `"publicKeyHex":"00","publicKeyHex":"00",`)), obs.ReasonDuplicate),
		r("case-variant-member", `E02-03: an HPKE response envelope with "V" next to "v".`,
			text(strings.Replace(hpkeResponseJSON, `"v":"v1",`, `"v":"v1","V":"v1",`, 1)), obs.ReasonDuplicate),
		r("lone-high-surrogate", `E02-04: agent-card-like whose description is the escape \ud800 alone.`,
			text(card(desc, `"description":"\ud800"`)), obs.ReasonMalformed),
		r("lone-low-surrogate", `E02-04: agent-card-like whose description is the escape \udc00 alone.`,
			text(card(desc, `"description":"\udc00"`)), obs.ReasonMalformed),
		r("invalid-utf8", "E02-04: agent-card-like whose description holds the byte 0xff.",
			raw(card(desc, "\"description\":\"\xff\"")), obs.ReasonMalformed),
		r("bom", "E02-04: agent-card-like preceded by the UTF-8 byte order mark EF BB BF.",
			raw("\xef\xbb\xbf"+cardJSON), obs.ReasonMalformed),
		r("trailing-data", "E02-04: agent-card-like followed by a second value.",
			text(cardJSON+" {}"), obs.ReasonMalformed),
		r("number-in-signed-object", `E02-05: agent-card-like with "version": 1.`,
			text(card(`"name":"vector-agent",`, `"name":"vector-agent","version":1,`)), obs.ReasonMalformed),
		r("too-deep", "E02-06: agent-card-like whose service holds 16 nested arrays (depth 17).",
			text(card(`"service":[]`, deep)), obs.ReasonTooLarge),
		r("too-large", "E02-06: agent-card-like padded in description to 64 KiB + 1 bytes.",
			text(large), obs.ReasonTooLarge),
	}
}

func canonicalise(in map[string]any) (map[string]any, error) {
	text, err := inputText(in)
	if err != nil {
		return nil, err
	}
	lim := jcs.SignedLimits()
	lim.Numbers = true // the RFC 8785 samples have numbers; signed SAGE objects do not
	c, err := jcs.Canonicalize(text, lim)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(c)
	return map[string]any{
		"canonical":     string(c),
		"canonical_hex": hex.EncodeToString(c),
		"sha256":        hex.EncodeToString(sum[:]),
	}, nil
}

func rejectSigned(in map[string]any) error {
	text, err := inputText(in)
	if err != nil {
		return err
	}
	_, err = jcs.Parse(text, jcs.SignedLimits())
	return err
}

// inputText returns input.json or the decoded input.json_hex; exactly one
// must be present.
func inputText(in map[string]any) ([]byte, error) {
	if len(in) != 1 {
		return nil, errors.New("input must hold exactly one of json and json_hex")
	}
	if s, ok := in["json"].(string); ok {
		return []byte(s), nil
	}
	if h, ok := in["json_hex"].(string); ok {
		b, err := hex.DecodeString(h)
		if err != nil || hex.EncodeToString(b) != h {
			return nil, errors.New("json_hex must be lower-case hex")
		}
		return b, nil
	}
	return nil, errors.New("input must hold a string json or json_hex")
}

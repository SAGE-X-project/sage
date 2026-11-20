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
	"net/http"
	"strings"

	"github.com/sage-x-project/sage/core"
	"github.com/sage-x-project/sage/core/httpsig"
	"github.com/sage-x-project/sage/core/obs"
	"github.com/sage-x-project/sage/core/vectors"
)

// RFC9421Description is the description of the rfc9421 suite file
// (unchanged from draft.1; the published request and response vectors join
// with the verifier, W-012).
const RFC9421Description = "HTTP Message Signatures profile: covered components, signature base, Signature-Input/Signature headers, Content-Digest (sha-256), keyid = DID#fragment, and response signatures bound to the request with ;req."

// The received request of the published vector rfc9421/request-ed25519.
const (
	reqMethod         = "POST"
	reqURL            = "https://agent-b.example/mcp/tools/call"
	reqBody           = `{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"echo","arguments":{"text":"hello"}}}`
	reqKeyLabel       = "sage-spec/vectors/ed25519/agent-a"
	reqCreated        = 1788609600
	reqSignatureInput = `sig1=("@method" "@target-uri" "@authority" "content-type" "content-digest" "x-sage-did" "date");keyid="did:sage:ethereum:0x1111111111111111111111111111111111111111#key-1";alg="ed25519";created=1788609600;nonce="8f0c5b1c-0d1a-4c8e-9a51-6d2e4b7f1a10"`
	reqSignature      = `sig1=:VO0yputjPcCYt/44t6UUG+tjhuZRPTG18gcSj9D+x42OFhGyt7pTc3hhMjqQKEvYsxl9KpD9nKEzoar80rGJAA==:`
	reqContentDigest  = `sha-256=:8LVpz99pqw3Qk15ZR9KrTw7BNYt9yrfJQf92GvVs0X0=:`
)

// receivedRequest returns request-ed25519 as received: its field lines in
// order, with the two signature fields given, and extra lines appended.
func receivedRequest(input, signature string, extra ...[2]string) map[string]any {
	fields := []any{
		[]any{"content-type", "application/json"},
		[]any{"date", "Tue, 01 Sep 2026 12:00:00 GMT"},
		[]any{"x-sage-did", "did:sage:ethereum:0x1111111111111111111111111111111111111111"},
		[]any{"content-digest", reqContentDigest},
		[]any{"signature-input", input},
		[]any{"signature", signature},
	}
	for _, f := range extra {
		fields = append(fields, []any{f[0], f[1]})
	}
	return map[string]any{
		"method": reqMethod, "url": reqURL, "body": reqBody, "fields": fields,
		"key_label": reqKeyLabel, "verify_at": reqCreated,
	}
}

// RFC9421 returns the rfc9421 suite cases that need no verifier yet: the
// errata §11 vectors that the parser rejects (E03-03, E03-05 to E03-07,
// E03-21).
func RFC9421() vectors.Suite {
	r := func(name, item, desc string, in map[string]any, reason obs.Reason) vectors.Case {
		return vectors.Case{
			Name: name, Mode: vectors.ModeReject,
			Description: item + ": request-ed25519 as received, with " + desc + "; rejected when Signature-Input and Signature are parsed (03 §5 step 1).",
			Input:       in, Reject: rejectHTTPRequest, Reason: reason,
		}
	}
	padded := strings.Replace(reqSignatureInput, `nonce="`, `nonce="`+strings.Repeat("0", core.SpecMaxSignatureFieldBytes+1-len(reqSignatureInput)), 1)
	return vectors.Suite{Name: "rfc9421", Description: RFC9421Description, Cases: []vectors.Case{
		r("tag-present", "E03-03", `;tag="x" added to Signature-Input`,
			receivedRequest(reqSignatureInput+`;tag="x"`, reqSignature), obs.ReasonParameters),
		r("unknown-param", "E03-03", "the undefined parameter ;foo=1 added to Signature-Input",
			receivedRequest(reqSignatureInput+`;foo=1`, reqSignature), obs.ReasonUnknownMember),
		r("noncanonical-params", "E03-05", "a second space between the first two covered components",
			receivedRequest(strings.Replace(reqSignatureInput, `"@method" "@target-uri"`, `"@method"  "@target-uri"`, 1), reqSignature), obs.ReasonNonCanonical),
		r("two-labels", "E03-06", "a second label sig2 (a copy of sig1) in both dictionaries",
			receivedRequest(reqSignatureInput+", "+strings.Replace(reqSignatureInput, "sig1=", "sig2=", 1), reqSignature+", "+strings.Replace(reqSignature, "sig1=", "sig2=", 1)), obs.ReasonDuplicate),
		r("label-sets-differ", "E03-06", "the Signature label renamed to sig2",
			receivedRequest(reqSignatureInput, strings.Replace(reqSignature, "sig1=", "sig2=", 1)), obs.ReasonMalformed),
		r("repeated-label", "E03-06", "the sig1 member repeated in both dictionaries",
			receivedRequest(reqSignatureInput+", "+reqSignatureInput, reqSignature+", "+reqSignature), obs.ReasonDuplicate),
		r("two-signature-input-lines", "E03-07", "a second Signature-Input field line with the same value",
			receivedRequest(reqSignatureInput, reqSignature, [2]string{"signature-input", reqSignatureInput}), obs.ReasonDuplicate),
		r("signature-input-too-large", "E03-21", "the nonce padded so that Signature-Input is 8 KiB + 1 bytes",
			receivedRequest(padded, reqSignature), obs.ReasonTooLarge),
	}}
}

// rejectHTTPRequest reads the received request of a vector and runs the
// signature parser on it (step 1 of 03 §5).
func rejectHTTPRequest(in map[string]any) error {
	list, ok := in["fields"].([]any)
	if !ok {
		return errors.New("fields must be a list")
	}
	h := http.Header{}
	for _, f := range list {
		pair, ok := f.([]any)
		if !ok || len(pair) != 2 {
			return errors.New("each field must be [name, value]")
		}
		name, ok1 := pair[0].(string)
		value, ok2 := pair[1].(string)
		if !ok1 || !ok2 || name == "" || strings.ToLower(name) != name {
			return errors.New("field names must be lower-case strings")
		}
		h.Add(name, value)
	}
	_, err := httpsig.Parse(h, "", core.DefaultLimits())
	return err
}

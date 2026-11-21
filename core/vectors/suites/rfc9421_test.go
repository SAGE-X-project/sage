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
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/sage-x-project/sage/core/vectors"
)

func publishedRFC9421(t *testing.T) (string, map[string]map[string]any) {
	t.Helper()
	data, err := os.ReadFile(published + "/rfc9421.json")
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Description string           `json:"description"`
		Vectors     []map[string]any `json:"vectors"`
	}
	if err := json.Unmarshal(data, &f); err != nil {
		t.Fatal(err)
	}
	byName := map[string]map[string]any{}
	for _, v := range f.Vectors {
		byName[v["name"].(string)] = v
	}
	return f.Description, byName
}

// TestRFC9421ReproducesPublished regenerates the four published rfc9421
// vectors with httpsig.Signer and requires values identical to the
// published file; the randomised P-256 signature is taken from it and
// verified.
func TestRFC9421ReproducesPublished(t *testing.T) {
	s := RFC9421()
	s.Cases = rfc9421Published()
	files, err := runner(t, s, publishedGenerator).Generate(os.DirFS(published))
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Description string           `json:"description"`
		Vectors     []map[string]any `json:"vectors"`
	}
	if err := json.Unmarshal(files["rfc9421.json"], &got); err != nil {
		t.Fatal(err)
	}
	desc, want := publishedRFC9421(t)
	if got.Description != desc || len(got.Vectors) != len(want) {
		t.Fatalf("description or vector count differs from the published file")
	}
	for _, v := range got.Vectors {
		if !reflect.DeepEqual(v, want[v["name"].(string)]) {
			t.Errorf("%s differs from the published vector:\n got %v\nwant %v", v["name"], v, want[v["name"].(string)])
		}
	}
}

// TestRFC9421SuiteBuilder: the suite's own signature base builder, which
// signs the reject vectors, reproduces the published request and response
// bases and signatures, independently of httpsig.
func TestRFC9421SuiteBuilder(t *testing.T) {
	_, pub := publishedRFC9421(t)
	req, resp := baseRequest(), baseResponse()
	for _, c := range []struct{ name, got, want string }{
		{"request input", req.member(), reqSignatureInput},
		{"request signature", req.sign(), reqSignature},
		{"request base", req.base(nil), pub["request-ed25519"]["output"].(map[string]any)["signature_base"].(string)},
		{"response input", resp.member(), pub["response-ed25519"]["output"].(map[string]any)["signature_input"].(string)},
		{"response signature", resp.sign(), respSignature},
		{"response base", resp.signFor.base(&resp), pub["response-ed25519"]["output"].(map[string]any)["signature_base"].(string)},
	} {
		if c.got != c.want {
			t.Errorf("%s:\n got %q\nwant %q", c.name, c.got, c.want)
		}
	}
}

// TestRFC9421BaseMessagesPass: request-ed25519 and response-ed25519 as
// received pass every step the reject vectors run, so each reject vector
// fails only through its mutation.
func TestRFC9421BaseMessagesPass(t *testing.T) {
	if err := rejectHTTPRequest(baseRequest().received()); err != nil {
		t.Fatalf("base request: %v", err)
	}
	if err := rejectHTTPResponse(baseResponse().received()); err != nil {
		t.Fatalf("base response: %v", err)
	}
}

// TestRFC9421RejectsAreResigned: every reject vector that changes the
// signed content carries a signature over its mutated base, so a verifier
// without the rule under test would accept it.
func TestRFC9421RejectsAreResigned(t *testing.T) {
	for _, c := range rfc9421Rejects() {
		if c.Mode != vectors.ModeReject || !strings.Contains(c.Description, "re-signed") {
			continue
		}
		fields := c.Input["fields"].([]any)
		var input, sig string
		for _, f := range fields {
			pair := f.([]any)
			switch pair[0] {
			case "signature-input":
				input = pair[1].(string)
			case "signature":
				sig = pair[1].(string)
			}
		}
		if input == "" || sig == "" || sig == reqSignature {
			t.Errorf("%s: not re-signed", c.Name)
		}
	}
}

func TestRFC9421InputErrors(t *testing.T) {
	for _, in := range []map[string]any{
		{"fields": "x"},
		{"fields": []any{"x"}},
		{"fields": []any{[]any{"Content-Type", "a"}}},
		{"fields": []any{[]any{"a", 1}}},
		{"fields": []any{}, "method": 1},
	} {
		if err := rejectHTTPRequest(in); err == nil || isRejection(err) {
			t.Errorf("%v: %v", in, err)
		}
	}
	if err := rejectHTTPResponse(map[string]any{}); err == nil || isRejection(err) {
		t.Errorf("response without request: %v", err)
	}
}

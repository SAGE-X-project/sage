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
	"testing"

	"github.com/sage-x-project/sage/core/vectors"
)

// TestDIDReproducesPublished regenerates the published did vectors this
// suite covers (parse, chain-aliases) and requires values identical to the
// published file. The other published vectors (pop-*, a2a-card-proof-*)
// join the suite with W-030 and W-040; until then testdata/did.json holds
// only the cases defined here.
func TestDIDReproducesPublished(t *testing.T) {
	s := DID()
	var draft1 []vectors.Case
	for _, c := range s.Cases {
		if c.Name == "parse" || c.Name == "chain-aliases" {
			draft1 = append(draft1, c)
		}
	}
	if len(draft1) != 2 {
		t.Fatalf("found %d published cases", len(draft1))
	}
	s.Cases = draft1
	files, err := runner(t, s, publishedGenerator).Generate(nil)
	if err != nil {
		t.Fatal(err)
	}
	var got, want struct {
		Description string           `json:"description"`
		Vectors     []map[string]any `json:"vectors"`
	}
	if err := json.Unmarshal(files["did.json"], &got); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(published + "/did.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &want); err != nil {
		t.Fatal(err)
	}
	if got.Description != want.Description {
		t.Fatal("suite description differs from the published file")
	}
	byName := map[string]map[string]any{}
	for _, v := range want.Vectors {
		byName[v["name"].(string)] = v
	}
	for _, v := range got.Vectors {
		if !reflect.DeepEqual(v, byName[v["name"].(string)]) {
			t.Errorf("%s differs from the published vector:\n got %v\nwant %v", v["name"], v, byName[v["name"].(string)])
		}
	}
}

// TestRFC9421BaseRequestParses: the received request the reject vectors
// mutate parses, so each vector fails only through its mutation.
func TestRFC9421BaseRequestParses(t *testing.T) {
	if err := rejectHTTPRequest(receivedRequest(reqSignatureInput, reqSignature)); err != nil {
		t.Fatalf("base request: %v", err)
	}
	for _, in := range []map[string]any{
		{"fields": "x"},
		{"fields": []any{"x"}},
		{"fields": []any{[]any{"Content-Type", "a"}}},
		{"fields": []any{[]any{"a", 1}}},
	} {
		if err := rejectHTTPRequest(in); err == nil || isRejection(err) {
			t.Errorf("%v: %v", in, err)
		}
	}
}

// TestRFC9421BaseMatchesPublished pins the constants of the base request to
// the published vector rfc9421/request-ed25519.
func TestRFC9421BaseMatchesPublished(t *testing.T) {
	data, err := os.ReadFile(published + "/rfc9421.json")
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Vectors []struct {
			Name   string         `json:"name"`
			Input  map[string]any `json:"input"`
			Output map[string]any `json:"output"`
		} `json:"vectors"`
	}
	if err := json.Unmarshal(data, &f); err != nil {
		t.Fatal(err)
	}
	for _, v := range f.Vectors {
		if v.Name != "request-ed25519" {
			continue
		}
		got := map[string]any{"signature_input": reqSignatureInput, "signature": reqSignature, "content_digest": reqContentDigest,
			"method": reqMethod, "url": reqURL, "body": reqBody, "key_label": reqKeyLabel}
		want := map[string]any{"signature_input": v.Output["signature_input"], "signature": v.Output["signature"], "content_digest": v.Output["content_digest"],
			"method": v.Input["method"], "url": v.Input["url"], "body": v.Input["body"], "key_label": v.Input["key_label"]}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("base request differs from the published vector:\n got %v\nwant %v", got, want)
		}
		if c, _ := v.Input["created"].(float64); int64(c) != reqCreated {
			t.Fatalf("created %v", v.Input["created"])
		}
		return
	}
	t.Fatal("request-ed25519 not found")
}

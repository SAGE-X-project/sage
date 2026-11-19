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

package vectors

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateThenCheck(t *testing.T) {
	dir := t.TempDir()
	if err := Generate(dir); err != nil {
		t.Fatalf("generate: %v", err)
	}
	if err := Check(dir); err != nil {
		t.Fatalf("check: %v", err)
	}
	// Regenerating must be a no-op for deterministic vectors and must keep
	// verify-mode outputs that still verify.
	before := readAll(t, dir)
	if err := Generate(dir); err != nil {
		t.Fatalf("regenerate: %v", err)
	}
	after := readAll(t, dir)
	for name := range before {
		if before[name] != after[name] {
			t.Errorf("%s changed on regeneration", name)
		}
	}
}

func TestCheckDetectsTampering(t *testing.T) {
	dir := t.TempDir()
	if err := Generate(dir); err != nil {
		t.Fatal(err)
	}
	for _, suite := range []string{"crypto", "rfc9421", "session", "hpke", "did", "jcs"} {
		path := filepath.Join(dir, suite+".json")
		f, err := readFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var v *Vector
		tampered := false
		for i := range f.Vectors {
			for k, val := range f.Vectors[i].Output {
				if s, ok := val.(string); ok && len(s) > 2 {
					f.Vectors[i].Output[k] = flipLast(s)
					v = &f.Vectors[i]
					tampered = true
					break
				}
			}
			if tampered {
				break
			}
		}
		if !tampered {
			t.Fatalf("%s: no string output to tamper with", suite)
		}
		if err := writeFile(path, f); err != nil {
			t.Fatal(err)
		}
		err = Check(dir)
		if err == nil || !strings.Contains(err.Error(), suite+"/"+v.Name) {
			t.Errorf("%s: tampering not detected: %v", suite, err)
		}
		if err := Generate(dir); err != nil { // restore
			t.Fatal(err)
		}
	}
}

func TestEveryVectorHasStableInput(t *testing.T) {
	for _, s := range Suites() {
		for _, c := range s.Cases {
			if c.Mode == ModeVerify && c.Verify == nil {
				t.Errorf("%s/%s: verify-mode case without Verify", s.Name, c.Name)
			}
			if c.Name == "" || c.Description == "" {
				t.Errorf("%s: case without name or description", s.Name)
			}
			if _, err := json.Marshal(c.Input); err != nil {
				t.Errorf("%s/%s: input not JSON: %v", s.Name, c.Name, err)
			}
		}
	}
}

func readAll(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		out[e.Name()] = string(data)
	}
	return out
}

func flipLast(s string) string {
	b := []byte(s)
	last := b[len(b)-1]
	if last == '0' {
		b[len(b)-1] = '1'
	} else {
		b[len(b)-1] = '0'
	}
	return string(b)
}

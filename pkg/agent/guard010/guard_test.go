package guard010_test

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/sage-x-project/sage/pkg/agent/guard010"
	"os"
	"reflect"
	"strings"
	"testing"
)

func cases(t *testing.T) []struct {
	ID        string          `json:"id"`
	Operation string          `json:"operation"`
	Input     json.RawMessage `json:"input"`
	Expected  map[string]any  `json:"expected"`
} {
	t.Helper()
	b, e := os.ReadFile("testdata/guard-records.json")
	if e != nil {
		t.Fatal(e)
	}
	var s struct {
		Cases []struct {
			ID        string          `json:"id"`
			Operation string          `json:"operation"`
			Input     json.RawMessage `json:"input"`
			Expected  map[string]any  `json:"expected"`
		}
	}
	if e = json.Unmarshal(b, &s); e != nil {
		t.Fatal(e)
	}
	return s.Cases
}
func TestFrozenIndependentVectors(t *testing.T) {
	n := 0
	for _, c := range cases(t) {
		if !strings.HasPrefix(c.Operation, "sage.guard.") {
			continue
		}
		n++
		t.Run(c.ID, func(t *testing.T) {
			v, o, e := guardObserve(c.Operation, c.Input)
			if e != nil {
				t.Fatal(e)
			}
			if !reflect.DeepEqual(map[string]any{"verdict": v, "output": o}, c.Expected) {
				t.Fatalf("got %s %v, expected %v", v, o, c.Expected)
			}
		})
	}
	if n != 94 {
		t.Fatal(n)
	}
}
func TestStrictJSON(t *testing.T) {
	for _, s := range []string{`{"a":1,"\u0061":2}`, `"\ud800"`, `"\udc00"`, `{"x":-0}`, `{"x":-1e-999}`, `{"x":1e999}`, "{}{}", "\ufeff{}", "\xff"} {
		if _, e := guard010.Canonicalize([]byte(s)); e == nil {
			t.Errorf("accepted %s", s)
		}
	}
	for _, s := range []string{`{"x":"\ud83d\ude00"}`, `{"x":"\\ud800"}`} {
		if _, e := guard010.Canonicalize([]byte(s)); e != nil {
			t.Errorf("rejected %s", s)
		}
	}
	for _, n := range []int{32, 33} {
		_, e := guard010.Canonicalize([]byte(strings.Repeat("[", n) + "0" + strings.Repeat("]", n)))
		if (e == nil) != (n == 32) {
			t.Fatal(n, e)
		}
	}
}

type noAuthority struct{ t *testing.T }

func (a noAuthority) Now(context.Context) (int64, error) {
	a.t.Fatal("malformed input reached clock")
	return 0, guard010.ErrInvalid
}
func (a noAuthority) ActiveKey(context.Context, string, string) (ed25519.PublicKey, error) {
	a.t.Fatal("malformed input reached key service")
	return nil, guard010.ErrInvalid
}
func TestProtocolNumbersBeforeAuthority(t *testing.T) {
	for _, c := range cases(t) {
		if c.ID != "intent-valid" {
			continue
		}
		f := guardFixture{}
		_ = json.Unmarshal(c.Input, &f)
		raw, _ := hex.DecodeString(f.s("envelope_hex"))
		for _, n := range []string{"1700000000.0000000001", "-0", "9007199254740992"} {
			s := strings.Replace(string(raw), `"created":1700000000`, `"created":`+n, 1)
			if _, e := guard010.VerifyIntent(context.Background(), []byte(s), f.s("expected_recipient"), noAuthority{t}, f); e == nil {
				t.Fatal(n)
			}
		}
	}
}
func TestOwnedVerificationAndCancellation(t *testing.T) {
	for _, c := range cases(t) {
		if c.ID != "intent-valid" {
			continue
		}
		f := guardFixture{}
		_ = json.Unmarshal(c.Input, &f)
		raw, _ := hex.DecodeString(f.s("envelope_hex"))
		v, e := guard010.VerifyIntent(context.Background(), raw, f.s("expected_recipient"), f, f)
		if e != nil {
			t.Fatal(e)
		}
		original := v.Digest()
		raw[0] = '!'
		copy := v.Canonical()
		copy[0] = '!'
		if v.Digest() != original || v.Canonical()[0] != '{' {
			t.Fatal("mutable verification result")
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, e = guard010.VerifyIntent(ctx, v.Canonical(), f.s("expected_recipient"), f, f); e == nil {
			t.Fatal("cancelled request accepted")
		}
		if _, e = guard010.VerifyIntent(context.Background(), v.Canonical(), f.s("expected_recipient"), nil, f); e == nil {
			t.Fatal("nil authority accepted")
		}
	}
}

func TestManifestFileLimit(t *testing.T) {
	files := []any{}
	for n := 0; n < 4097; n++ {
		files = append(files, map[string]any{"path": fmt.Sprintf("f%04d", n), "sha256": strings.Repeat("0", 64)})
	}
	for _, n := range []int{4096, 4097} {
		raw, _ := json.Marshal(map[string]any{"version": "0.10.0", "files": files[:n]})
		_, e := guard010.ManifestCommitment(raw)
		if (e == nil) != (n == 4096) {
			t.Fatal(n, e)
		}
	}
}

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

package did

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/sage-x-project/sage/core"
	"github.com/sage-x-project/sage/core/obs"
)

func wantIdentifierRejection(t *testing.T, in string, err error) {
	t.Helper()
	var r *core.Rejection
	if !errors.As(err, &r) {
		t.Fatalf("%q: error %v is not a core.Rejection", in, err)
	}
	if r.Reason() != obs.ReasonIdentifier || err.Error() != core.AuthenticationFailedText {
		t.Fatalf("%q: reason %s, text %q", in, r.Reason(), err)
	}
}

// The published did/parse vector (sage-spec v1/vectors/did.json).
func TestPublishedParse(t *testing.T) {
	valid := map[string][2]string{
		"did:sage:ethereum:0x1111111111111111111111111111111111111111":    {"ethereum", "0x1111111111111111111111111111111111111111"},
		"did:sage:ethereum:0x1111111111111111111111111111111111111111:42": {"ethereum", "0x1111111111111111111111111111111111111111:42"},
		"did:sage:solana:7Zn3QwXk9vQ2yP1sJ4cR8tL6mN5bV3xA2dF9gH1kJ8Lm":    {"solana", "7Zn3QwXk9vQ2yP1sJ4cR8tL6mN5bV3xA2dF9gH1kJ8Lm"},
		"did:sage:ETHEREUM:agent-one":                                     {"ethereum", "agent-one"},
		"did:sage:eth:0xabc":                                              {"ethereum", "0xabc"},
	}
	for in, want := range valid {
		d, err := Parse(in)
		if err != nil {
			t.Fatalf("%q: %v", in, err)
		}
		if d.Chain().String() != want[0] || d.Identifier() != want[1] || d.AsSent() != in {
			t.Errorf("%q: chain %s identifier %s as-sent %s", in, d.Chain(), d.Identifier(), d.AsSent())
		}
		if d.String() != "did:sage:"+want[0]+":"+want[1] {
			t.Errorf("%q: canonical %s", in, d)
		}
	}
	for _, in := range []string{"", "did:sage", "did:sage:ethereum", "did:web:example.com", "did:sage:polkadot:abc", "sage:ethereum:0xabc", "did:sage::0xabc"} {
		_, err := Parse(in)
		wantIdentifierRejection(t, in, err)
	}
}

// TestGrammar covers the errata §11 parse additions (E06-02, E06-03).
func TestGrammar(t *testing.T) {
	long := "did:sage:ethereum:" + strings.Repeat("a", MaxDIDBytes-len("did:sage:ethereum:")+1)
	for _, in := range []string{
		"did:sage:ethereum:0x11%20", "did:sage:ethereum:a#b", "did:sage:ethereum:a?b", "did:sage:ethereum:a/b",
		"did:sage:ethereum:a|b", "did:sage:ethereum:a b", "did:sage:ethereum:ä", "did:sage:ethereum::",
		"did:sage:ethereum:a::b", "did:sage:ethereum:a:", long, "DID:sage:ethereum:a", "did:SAGE:ethereum:a",
		"did:sage: ethereum:a", "did:sage:ethereum :a", "did:sage:ethereum:a\x00", "did:sage:ethereum:a\n",
		"did:sage:ethereum:key-1#key-1",
	} {
		_, err := Parse(in)
		wantIdentifierRejection(t, in, err)
	}
	okLong := long[:len(long)-1]
	if _, err := Parse(okLong); err != nil || len(okLong) != MaxDIDBytes {
		t.Fatalf("a %d-byte DID must parse: %v", len(okLong), err)
	}
	for _, in := range []string{"did:sage:sol:a.b_c~d-e", "did:sage:Solana:X", "did:sage:ethereum:a:b:c"} {
		if _, err := Parse(in); err != nil {
			t.Errorf("%q: %v", in, err)
		}
	}
}

func TestCanonicalEquality(t *testing.T) {
	a, err := Parse("did:sage:eth:0xabc")
	if err != nil {
		t.Fatal(err)
	}
	b, err := Parse("did:sage:ethereum:0xabc")
	if err != nil {
		t.Fatal(err)
	}
	if !a.Equal(b) || a.AsSent() == b.AsSent() || a.String() != b.String() {
		t.Fatal("canonical forms must compare equal while the as-sent forms differ")
	}
	c, _ := Parse("did:sage:ethereum:0xABC")
	if a.Equal(c) {
		t.Fatal("the identifier is compared exactly")
	}
	var zero DID
	if zero.Equal(zero) || a.Equal(zero) || !zero.IsZero() || a.IsZero() {
		t.Fatal("the zero DID equals nothing")
	}
}

func TestParseChain(t *testing.T) {
	for in, want := range map[string]Chain{"ethereum": ChainEthereum, "ETH": ChainEthereum, " Solana ": ChainSolana, "sol": ChainSolana} {
		got, err := ParseChain(in)
		if err != nil || got != want {
			t.Errorf("%q: %v %v", in, got, err)
		}
	}
	for _, in := range []string{"", "polkadot", "ethereum-mainnet", "eth2"} {
		if _, err := ParseChain(in); err == nil {
			t.Errorf("%q accepted", in)
		}
	}
	if ChainUnknown.String() != "" {
		t.Error("unknown chain name")
	}
}

func TestNew(t *testing.T) {
	d, err := New(ChainEthereum, "0xabc")
	if err != nil || d.String() != "did:sage:ethereum:0xabc" || d.AsSent() != d.String() {
		t.Fatalf("%v %v", d, err)
	}
	for _, c := range []struct {
		chain Chain
		id    string
	}{{ChainUnknown, "a"}, {ChainEthereum, ""}, {ChainEthereum, "a#b"}, {Chain(9), "a"}} {
		if _, err := New(c.chain, c.id); err == nil {
			t.Errorf("New(%d, %q) accepted", c.chain, c.id)
		}
	}
}

func TestKeyRef(t *testing.T) {
	r, err := ParseKeyRef("did:sage:eth:0xabc#key-12")
	if err != nil {
		t.Fatal(err)
	}
	if r.DID().String() != "did:sage:ethereum:0xabc" || r.Fragment() != "key-12" || r.AsSent() != "did:sage:eth:0xabc#key-12" {
		t.Fatalf("%s %s %s", r.DID(), r.Fragment(), r.AsSent())
	}
	plain, err := ParseKeyRef("did:sage:ethereum:0xabc")
	if err != nil || plain.Fragment() != "" || !plain.DID().Equal(r.DID()) {
		t.Fatalf("plain keyid: %v", err)
	}
	if _, err := ParseKeyRef("did:sage:ethereum:0xabc#key-0"); err != nil {
		t.Fatalf("key-0 has no leading zero: %v", err)
	}
	base := "did:sage:ethereum:" + strings.Repeat("a", MaxDIDBytes-len("did:sage:ethereum:"))
	fragment := "#key-" + strings.Repeat("9", MaxKeyRefBytes-MaxDIDBytes-len("#key-"))
	if _, err := ParseKeyRef(base + fragment); err != nil {
		t.Fatalf("a %d-byte keyid must parse: %v", len(base+fragment), err)
	}
	for _, in := range []string{
		base + fragment + "9", // 290 bytes
		base + "a#key-1",      // the DID part is 257 bytes
		"did:sage:ethereum:0xabc#key-01", "did:sage:ethereum:0xabc#key-", "did:sage:ethereum:0xabc#key-1a",
		"did:sage:ethereum:0xabc#Key-1", "did:sage:ethereum:0xabc#k1", "did:sage:ethereum:0xabc#", "did:sage:ethereum:0xabc#key-1#key-2",
		"did:sage:ethereum:0xabc#key--1", "did:sage:ethereum:0xabc#key-+1", "did:sage:ethereum:0x%41#key-1", "",
	} {
		_, err := ParseKeyRef(in)
		wantIdentifierRejection(t, in, err)
	}
}

type fakeResolver struct{}

func (fakeResolver) Resolve(context.Context, DID) (Record, Observation, error) {
	return Record{}, Observation{}, errors.New("not found")
}

func TestResolverShape(t *testing.T) {
	var r KeyResolver = fakeResolver{}
	d, _ := New(ChainEthereum, "0xabc")
	if _, _, err := r.Resolve(context.Background(), d); err == nil {
		t.Fatal("fake resolver")
	}
	rec := Record{Keys: []Key{{Type: KeyTypeECDSA}, {Type: KeyTypeEd25519}, {Type: KeyTypeX25519}}}
	if rec.Keys[0].Type != 0 || rec.Keys[1].Type != 1 || rec.Keys[2].Type != 2 {
		t.Fatal("registry key type codes are 0, 1, 2 (01-crypto.md §1)")
	}
}

func FuzzParse(f *testing.F) {
	for _, s := range []string{"did:sage:eth:0xabc", "did:sage:ethereum:a:b", "did:sage:x:y", "did:sage:ethereum:a#key-1",
		"did:sage:eth:" + strings.Repeat("0", MaxDIDBytes-len("did:sage:eth:"))} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		d, err := Parse(s)
		if err == nil {
			if len(s) > MaxDIDBytes {
				t.Fatalf("accepted %d bytes", len(s))
			}
			// The 256-byte limit applies to the DID as sent (E06-03); an
			// alias can make the canonical form longer, and only a
			// canonical form within the limit is itself a valid DID.
			if c := d.String(); len(c) <= MaxDIDBytes {
				again, err := Parse(c)
				if err != nil || !again.Equal(d) || again.String() != c {
					t.Fatalf("canonical form of %q does not parse back: %v", s, err)
				}
			}
		}
		if r, err := ParseKeyRef(s); err == nil && len(r.AsSent()) > MaxKeyRefBytes {
			t.Fatal("keyid over the limit accepted")
		}
	})
}

// TestAliasAtTheLimit pins the case the fuzzer found: a 256-byte DID that
// uses the eth alias is valid (E06-03 limits the DID as sent), compares
// equal to its canonical form, and that 261-byte canonical form is not
// itself a DID, so New refuses to emit it.
func TestAliasAtTheLimit(t *testing.T) {
	id := strings.Repeat("0", MaxDIDBytes-len("did:sage:eth:"))
	d, err := Parse("did:sage:eth:" + id)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.String()) != MaxDIDBytes+len("ereum") {
		t.Fatalf("canonical length %d", len(d.String()))
	}
	if _, err := New(ChainEthereum, id); err == nil {
		t.Fatal("New emitted a DID over 256 bytes")
	}
	other, err := Parse("did:sage:ETH:" + id)
	if err != nil || !other.Equal(d) {
		t.Fatalf("alias forms must compare equal: %v", err)
	}
}

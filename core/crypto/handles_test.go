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

package crypto

import (
	"bytes"
	"crypto/ecdh"
	"crypto/hpke"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/sage-x-project/sage/core/secret"
)

func labelScalar(label string) []byte {
	h := sha256.Sum256([]byte(label))
	return h[:]
}

func TestX25519KEMKey(t *testing.T) {
	scalar := labelScalar("sage-spec/vectors/x25519/agent-b")
	k, err := NewX25519KEMKey(secret.From(scalar))
	if err != nil {
		t.Fatal(err)
	}
	want, err := ecdh.X25519().NewPrivateKey(scalar)
	if err != nil {
		t.Fatal(err)
	}
	if pub := k.Public(); !bytes.Equal(pub[:], want.PublicKey().Bytes()) {
		t.Fatal("public key differs")
	}
	peer, _ := ecdh.X25519().NewPrivateKey(labelScalar("sage-spec/vectors/x25519/agent-a"))
	got, err := k.ECDH(peer.PublicKey())
	if err != nil {
		t.Fatal(err)
	}
	exp, _ := peer.ECDH(want.PublicKey())
	if !bytes.Equal(got, exp) {
		t.Fatal("shared secret differs")
	}
	if _, err := k.ECDH(nil); err == nil {
		t.Fatal("nil remote accepted")
	}
	p256, _ := ecdh.P256().GenerateKey(fixed{})
	if _, err := k.ECDH(p256.PublicKey()); err == nil {
		t.Fatal("P-256 remote accepted")
	}
	low, _ := ecdh.X25519().NewPublicKey(make([]byte, 32))
	if _, err := k.ECDH(low); err == nil {
		t.Fatal("low-order remote accepted")
	}
	if _, err := hpke.NewDHKEMPrivateKey(k); err != nil {
		t.Fatalf("crypto/hpke refuses the handle: %v", err)
	}
}

type fixed struct{}

func (fixed) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 1
	}
	return len(p), nil
}

func TestX25519KEMKeyRejectsBadInput(t *testing.T) {
	for _, sk := range []secret.Bytes{secret.From(make([]byte, 31)), secret.From(make([]byte, 33)), {}} {
		if _, err := NewX25519KEMKey(sk); err == nil {
			t.Errorf("accepted a %d-byte scalar", sk.Len())
		}
	}
	d := secret.From(labelScalar("x"))
	d.Destroy()
	if _, err := NewX25519KEMKey(d); !errors.Is(err, secret.ErrDestroyed) {
		t.Fatalf("destroyed secret: %v", err)
	}
}

func TestKEMKeyIsNotPrinted(t *testing.T) {
	scalar := labelScalar("sage-spec/vectors/x25519/agent-b")
	k, err := NewX25519KEMKey(secret.From(scalar))
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	type unexported struct{ k KEMKey }
	for _, verb := range []string{"%v", "%+v", "%#v", "%s", "%x", "%X", "%d", "%q", "%t", "%c", "%e"} {
		out = append(out, fmt.Sprintf(verb, k), fmt.Sprintf(verb, struct{ K KEMKey }{k}), fmt.Sprintf(verb, unexported{k}))
	}
	j, _ := json.Marshal(struct{ K KEMKey }{k})
	out = append(out, string(j))
	var buf bytes.Buffer
	slog.New(slog.NewJSONHandler(&buf, nil)).Info("x", "k", k)
	slog.New(slog.NewTextHandler(&buf, nil)).Info("x", "k", k, "u", unexported{k})
	out = append(out, buf.String())
	// %d of a byte slice prints space-separated decimals; look for the
	// first four bytes of the scalar in that form.
	decimal := fmt.Sprintf("%d %d %d %d", scalar[0], scalar[1], scalar[2], scalar[3])
	for _, o := range out {
		h := hex.EncodeToString(scalar)
		if strings.Contains(o, h) || strings.Contains(o, strings.ToUpper(h)) || strings.Contains(o, decimal) {
			t.Fatalf("output leaks the scalar: %q", o)
		}
	}
}

func TestPurposeNames(t *testing.T) {
	seen := map[string]bool{}
	for p := PurposeHTTPRequest; p <= PurposePoP; p++ {
		n := p.String()
		if n == "unknown" || seen[n] {
			t.Errorf("purpose %d name %q", p, n)
		}
		seen[n] = true
	}
	if PurposeUnknown.String() != "unknown" || Purpose(200).String() != "unknown" {
		t.Error("values outside the set must be unknown")
	}
}

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
	"crypto/sha256"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"

	"github.com/sage-x-project/sage/core"
	"github.com/sage-x-project/sage/core/clock"
	"github.com/sage-x-project/sage/core/crypto"
	"github.com/sage-x-project/sage/core/obs"
	"github.com/sage-x-project/sage/core/secret"
)

const evalDID = "did:sage:ethereum:0x1111111111111111111111111111111111111111"

var evalAt = time.Unix(1788609600, 0)

type evalKeys struct {
	ed, k1, k1b crypto.Signer
}

func newEvalKeys(t *testing.T) evalKeys {
	t.Helper()
	mk := func(f func(secret.Bytes) (crypto.Signer, error), label string) crypto.Signer {
		sum := sha256.Sum256([]byte(label))
		s, err := f(secret.From(sum[:]))
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	return evalKeys{
		ed:  mk(crypto.NewEd25519Signer, "sage-core/did-test/ed25519"),
		k1:  mk(crypto.NewSecp256k1Signer, "sage-core/did-test/secp256k1-a"),
		k1b: mk(crypto.NewSecp256k1Signer, "sage-core/did-test/secp256k1-b"),
	}
}

// entry returns a listed key with a valid proof for d over data, or over
// the canonical encoding when data is nil.
func entry(t *testing.T, s crypto.Signer, d string, data []byte) Key {
	t.Helper()
	if data == nil {
		data = s.Public().Bytes()
	}
	proof, err := crypto.SignPoP(context.Background(), s, d, data)
	if err != nil {
		t.Fatal(err)
	}
	typ := KeyTypeECDSA
	if s.Public().Alg() == crypto.AlgEd25519 {
		typ = KeyTypeEd25519
	}
	return Key{Type: typ, KeyData: data, Signature: proof.Bytes(), Verified: true}
}

func activeRecord(t *testing.T, keys ...Key) Record {
	t.Helper()
	d, err := Parse(evalDID)
	if err != nil {
		t.Fatal(err)
	}
	return Record{DID: d, IsActive: true, Keys: keys}
}

type events struct {
	mu  sync.Mutex
	got []obs.Event
}

func (e *events) Trace(_ context.Context, ev obs.Event) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.got = append(e.got, ev)
}

func policy(t *testing.T, opts ...core.Option) Policy {
	t.Helper()
	s, err := core.NewSettings(append([]core.Option{core.WithSpecVersion(string(core.V0_9_0_Draft1_1))}, opts...)...)
	if err != nil {
		t.Fatal(err)
	}
	return Policy{Clock: clock.NewFixed(evalAt), Settings: s}
}

func wantDID(t *testing.T) DID {
	t.Helper()
	d, err := Parse(evalDID)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func fresh() Observation { return Observation{At: evalAt, Block: 1, Source: "test"} }

func evaluate(t *testing.T, rec Record) (Agent, error) {
	t.Helper()
	return Evaluate(context.Background(), rec.DID, rec, fresh(), policy(t))
}

func wantRejection(t *testing.T, err error, want obs.Reason) {
	t.Helper()
	var r *core.Rejection
	if !errors.As(err, &r) {
		t.Fatalf("error %v is not a core.Rejection", err)
	}
	if r.Reason() != want || err.Error() != core.AuthenticationFailedText {
		t.Fatalf("reason %s, want %s (text %q)", r.Reason(), want, err)
	}
}

func wantKey(t *testing.T, a Agent, err error, s crypto.Signer) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
	if !a.SigningKey().PublicKey().Equal(s.Public()) {
		t.Fatal("another signing key was selected")
	}
}

func TestEvaluatePolicy(t *testing.T) {
	k := newEvalKeys(t)
	rec := activeRecord(t, entry(t, k.k1, evalDID, nil))
	good := policy(t)
	for name, p := range map[string]Policy{
		"zero":        {},
		"no clock":    {Settings: good.Settings},
		"no settings": {Clock: good.Clock},
	} {
		if _, err := Evaluate(context.Background(), rec.DID, rec, fresh(), p); !errors.Is(err, ErrPolicy) {
			t.Fatalf("%s: %v", name, err)
		}
	}
}

// TestEvaluateSelection covers 06 §3.1 with the D33 reading: each entry
// reached in selection order is decoded first, and an undecodable one fails
// resolution.
func TestEvaluateSelection(t *testing.T) {
	k := newEvalKeys(t)
	edKey, k1Key, k1bKey := entry(t, k.ed, evalDID, nil), entry(t, k.k1, evalDID, nil), entry(t, k.k1b, evalDID, nil)
	noProof := k1Key
	noProof.Signature = k1bKey.Signature
	malformed := Key{Type: KeyTypeECDSA, KeyData: make([]byte, 40), Signature: k1Key.Signature, Verified: true}
	edMalformed := Key{Type: KeyTypeEd25519, KeyData: make([]byte, 31), Signature: edKey.Signature, Verified: true}
	kem := Key{Type: KeyTypeX25519, KeyData: make([]byte, 32), Verified: true}

	t.Run("secp256k1 before ed25519", func(t *testing.T) {
		a, err := evaluate(t, activeRecord(t, edKey, k1Key))
		wantKey(t, a, err, k.k1)
		if a.SigningKey().Type() != KeyTypeECDSA || !a.DID().Equal(activeRecord(t).DID) || a.IsZero() {
			t.Fatal("agent")
		}
	})
	t.Run("ed25519 when no secp256k1 is eligible", func(t *testing.T) {
		a, err := evaluate(t, activeRecord(t, noProof, edKey))
		wantKey(t, a, err, k.ed)
	})
	t.Run("first eligible in registry order", func(t *testing.T) {
		a, err := evaluate(t, activeRecord(t, noProof, k1bKey, k1Key))
		wantKey(t, a, err, k.k1b)
	})
	t.Run("malformed entry reached first fails", func(t *testing.T) {
		_, err := evaluate(t, activeRecord(t, malformed, k1Key))
		wantRejection(t, err, obs.ReasonResolution)
	})
	t.Run("malformed ed25519 reached fails", func(t *testing.T) {
		_, err := evaluate(t, activeRecord(t, noProof, edMalformed, edKey))
		wantRejection(t, err, obs.ReasonResolution)
	})
	t.Run("malformed entry after the selected one is not reached", func(t *testing.T) {
		a, err := evaluate(t, activeRecord(t, k1Key, malformed, edMalformed))
		wantKey(t, a, err, k.k1)
	})
	t.Run("no eligible key", func(t *testing.T) {
		_, err := evaluate(t, activeRecord(t, noProof))
		wantRejection(t, err, obs.ReasonKeyNotEligible)
	})
	t.Run("no keys are synthesised", func(t *testing.T) {
		rec := activeRecord(t, kem)
		rec.PublicKEMKey = kem.KeyData
		rec.Endpoint = "https://agent.example"
		_, err := evaluate(t, rec)
		wantRejection(t, err, obs.ReasonKeyNotEligible)
		_, err = evaluate(t, activeRecord(t))
		wantRejection(t, err, obs.ReasonKeyNotEligible)
	})
	t.Run("unknown key type is not eligible", func(t *testing.T) {
		odd := k1Key
		odd.Type = 7
		_, err := evaluate(t, activeRecord(t, odd))
		wantRejection(t, err, obs.ReasonKeyNotEligible)
	})
	t.Run("a type-1 entry holding a secp256k1 key", func(t *testing.T) {
		swapped := k1Key
		swapped.Type = KeyTypeEd25519
		_, err := evaluate(t, activeRecord(t, swapped))
		wantRejection(t, err, obs.ReasonResolution)
	})
}

// TestEvaluateKeyDataAsStored: a 33-byte or 64-byte key_data with a valid
// proof over that form is eligible, and the key is the 65-byte form
// (E01-03, E06-10).
func TestEvaluateKeyDataAsStored(t *testing.T) {
	k := newEvalKeys(t)
	pk, err := secp256k1.ParsePubKey(k.k1.Public().Bytes())
	if err != nil {
		t.Fatal(err)
	}
	for _, data := range [][]byte{pk.SerializeCompressed(), k.k1.Public().Bytes()[1:]} {
		a, err := evaluate(t, activeRecord(t, entry(t, k.k1, evalDID, data)))
		wantKey(t, a, err, k.k1)
		if len(a.SigningKey().PublicKey().Bytes()) != 65 {
			t.Fatal("signing key is not the 65-byte form")
		}
		// The proof over the stored form does not cover another form.
		e := entry(t, k.k1, evalDID, data)
		e.KeyData = k.k1.Public().Bytes()
		_, err = evaluate(t, activeRecord(t, e))
		wantRejection(t, err, obs.ReasonKeyNotEligible)
	}
}

// TestEvaluateUsesTheStoredDID: the challenge uses the record's did as
// stored, not its canonical form (06 §4).
func TestEvaluateUsesTheStoredDID(t *testing.T) {
	k := newEvalKeys(t)
	const alias = "did:sage:eth:0x1111111111111111111111111111111111111111"
	d, err := Parse(alias)
	if err != nil {
		t.Fatal(err)
	}
	rec := Record{DID: d, IsActive: true, Keys: []Key{entry(t, k.k1, evalDID, nil)}}
	_, err = evaluate(t, rec)
	wantRejection(t, err, obs.ReasonKeyNotEligible)
	rec.Keys = []Key{entry(t, k.k1, alias, nil)}
	a, err := evaluate(t, rec)
	wantKey(t, a, err, k.k1)
}

// TestEvaluateRegistrationSignature: the deployed registry's EIP-191
// registration value is not a proof (E06-09), even signed by the listed key.
func TestEvaluateRegistrationSignature(t *testing.T) {
	k := newEvalKeys(t)
	ctx := context.Background()
	inner := sha256.Sum256([]byte("SAGE Agent Registration:"))
	msg := append([]byte("\x19Ethereum Signed Message:\n32"), inner[:]...)
	sig, err := k.k1.Sign(ctx, crypto.PurposeHTTPRequest, msg)
	if err != nil {
		t.Fatal(err)
	}
	b := sig.Bytes()
	b[64] += 27
	reg := entry(t, k.k1, evalDID, nil)
	for _, s := range [][]byte{sig.Bytes(), b} {
		reg.Signature = s
		_, err := evaluate(t, activeRecord(t, reg))
		wantRejection(t, err, obs.ReasonKeyNotEligible)
	}
}

// TestEvaluateBindsTheQueriedDID: a record of another DID is not accepted
// for want; an alias of want is the same DID (06 §2).
func TestEvaluateBindsTheQueriedDID(t *testing.T) {
	k := newEvalKeys(t)
	ctx := context.Background()
	rec := activeRecord(t, entry(t, k.k1, evalDID, nil))
	other, err := Parse("did:sage:ethereum:0x2222222222222222222222222222222222222222")
	if err != nil {
		t.Fatal(err)
	}
	_, err = Evaluate(ctx, other, rec, fresh(), policy(t))
	wantRejection(t, err, obs.ReasonResolution)
	_, err = Evaluate(ctx, DID{}, rec, fresh(), policy(t))
	wantRejection(t, err, obs.ReasonResolution)
	alias, err := Parse("did:sage:eth:0x1111111111111111111111111111111111111111")
	if err != nil {
		t.Fatal(err)
	}
	a, err := Evaluate(ctx, alias, rec, fresh(), policy(t))
	wantKey(t, a, err, k.k1)
}

func TestEvaluateInactive(t *testing.T) {
	k := newEvalKeys(t)
	rec := activeRecord(t, entry(t, k.k1, evalDID, nil))
	rec.IsActive = false
	_, err := evaluate(t, rec)
	wantRejection(t, err, obs.ReasonInactive)
	// A negative result may be cached without limit (E06-12): an old
	// observation of an inactive agent is still a rejection for inactivity.
	_, err = Evaluate(context.Background(), rec.DID, rec, Observation{At: evalAt.Add(-time.Hour)}, policy(t))
	wantRejection(t, err, obs.ReasonInactive)
	_, err = evaluate(t, Record{IsActive: true, Keys: rec.Keys})
	wantRejection(t, err, obs.ReasonResolution)
}

// TestEvaluateLimits: E06-14, each member at its limit and one over.
func TestEvaluateLimits(t *testing.T) {
	k := newEvalKeys(t)
	base := activeRecord(t, entry(t, k.k1, evalDID, nil))
	for name, set := range map[string]func(r *Record, n int){
		"name":         func(r *Record, n int) { r.Name = strings.Repeat("n", n) },
		"description":  func(r *Record, n int) { r.Description = strings.Repeat("d", n) },
		"endpoint":     func(r *Record, n int) { r.Endpoint = strings.Repeat("e", n) },
		"capabilities": func(r *Record, n int) { r.Capabilities = make([]byte, n) },
		"keys": func(r *Record, n int) {
			r.Keys = append([]Key(nil), base.Keys...)
			for len(r.Keys) < n {
				r.Keys = append(r.Keys, Key{Type: KeyTypeX25519, KeyData: make([]byte, 32)})
			}
		},
	} {
		limit := map[string]int{"name": MaxNameBytes, "description": MaxDescriptionBytes, "endpoint": MaxEndpointBytes, "capabilities": MaxCapabilitiesBytes, "keys": MaxKeys}[name]
		at := base
		set(&at, limit)
		if _, err := evaluate(t, at); err != nil {
			t.Fatalf("%s at the limit: %v", name, err)
		}
		over := base
		set(&over, limit+1)
		_, err := evaluate(t, over)
		wantRejection(t, err, obs.ReasonResolution)
	}
}

// TestEvaluateFreshness: E06-12 with MaxResolutionAge from the settings.
func TestEvaluateFreshness(t *testing.T) {
	k := newEvalKeys(t)
	rec := activeRecord(t, entry(t, k.k1, evalDID, nil))
	ctx := context.Background()
	p := policy(t)
	age := p.Settings.Limits().MaxResolutionAge
	if age != core.RecommendedResolutionAge {
		t.Fatalf("default MaxResolutionAge %s", age)
	}
	if _, err := Evaluate(ctx, rec.DID, rec, Observation{At: evalAt.Add(-age)}, p); err != nil {
		t.Fatalf("at MaxResolutionAge: %v", err)
	}
	for name, at := range map[string]time.Time{
		"older":     evalAt.Add(-age - time.Nanosecond),
		"no time":   {},
		"in future": evalAt.Add(time.Nanosecond),
	} {
		_, err := Evaluate(ctx, rec.DID, rec, Observation{At: at}, p)
		if err == nil {
			t.Fatalf("%s: accepted", name)
		}
		wantRejection(t, err, obs.ReasonResolutionStale)
	}
	lim := core.DefaultLimits()
	lim.MaxResolutionAge = core.SpecMaxResolutionAge
	long := policy(t, core.WithLimits(lim))
	if _, err := Evaluate(ctx, rec.DID, rec, Observation{At: evalAt.Add(-core.SpecMaxResolutionAge)}, long); err != nil {
		t.Fatalf("300 s with MaxResolutionAge 300 s: %v", err)
	}
	_, err := Evaluate(ctx, rec.DID, rec, Observation{At: evalAt.Add(-core.SpecMaxResolutionAge - time.Second)}, long)
	wantRejection(t, err, obs.ReasonResolutionStale)
	failing := p
	failing.Clock = clock.Failing(errors.New("no time"))
	_, err = Evaluate(ctx, rec.DID, rec, fresh(), failing)
	wantRejection(t, err, obs.ReasonClock)
	// A clock that returns the zero time does not date anything, and a zero
	// observation is not fresh even then.
	zero := p
	zero.Clock = clock.NewFixed(time.Time{})
	_, err = Evaluate(ctx, rec.DID, rec, Observation{}, zero)
	wantRejection(t, err, obs.ReasonClock)
}

// TestEvaluateIgnoresTheFlag: verified is not consulted (E06-08, E06-09);
// a listed key whose flag is false is reported as a diagnostic.
func TestEvaluateIgnoresTheFlag(t *testing.T) {
	k := newEvalKeys(t)
	unflagged := entry(t, k.k1, evalDID, nil)
	unflagged.Verified = false
	flaggedNoProof := entry(t, k.k1b, evalDID, nil)
	flaggedNoProof.Signature = unflagged.Signature
	tr := &events{}
	s, err := core.NewSettings(core.WithSpecVersion(string(core.V0_9_0_Draft1_1)), core.WithTracer(tr))
	if err != nil {
		t.Fatal(err)
	}
	p := Policy{Clock: clock.NewFixed(evalAt), Settings: s}
	a, err := Evaluate(context.Background(), wantDID(t), activeRecord(t, flaggedNoProof, unflagged), fresh(), p)
	wantKey(t, a, err, k.k1)
	if len(tr.got) != 1 {
		t.Fatalf("%d events, want one diagnostic", len(tr.got))
	}
	e := tr.got[0]
	if e.Outcome != obs.OutcomeDiagnostic || e.Stage != obs.StageResolve || !e.Subject.Unverified() || e.Spec != string(core.V0_9_0_Draft1_1) {
		t.Fatalf("event %+v", e)
	}
	tr.got = nil
	if _, err := Evaluate(context.Background(), wantDID(t), activeRecord(t, entry(t, k.k1, evalDID, nil)), fresh(), p); err != nil || len(tr.got) != 0 {
		t.Fatalf("flag true: err %v, %d events", err, len(tr.got))
	}
}

func TestAgentZero(t *testing.T) {
	var a Agent
	if !a.IsZero() || !a.SigningKey().IsZero() || a.SigningKey().PublicKey() != nil || !a.DID().IsZero() {
		t.Fatal("zero Agent")
	}
}

// TestVerifyPoPOnlyInEvaluate: crypto.VerifyPoP is called by no non-test
// code of the core other than this package and its own definition, so a
// proof of possession is accepted on no other path (E01-07).
func TestVerifyPoPOnlyInEvaluate(t *testing.T) {
	root := ".."
	var callers []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == "testdata" || strings.HasPrefix(d.Name(), ".")) && path != root {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.SelectorExpr:
				if x.Sel.Name == "VerifyPoP" {
					callers = append(callers, filepath.ToSlash(path))
				}
			case *ast.CallExpr:
				if id, ok := x.Fun.(*ast.Ident); ok && id.Name == "VerifyPoP" {
					callers = append(callers, filepath.ToSlash(path))
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range callers {
		if c != "../did/evaluate.go" && c != "../crypto/pop.go" {
			t.Errorf("crypto.VerifyPoP referenced in %s", c)
		}
	}
	if len(callers) == 0 {
		t.Fatal("found no reference at all; the scan is broken")
	}
}

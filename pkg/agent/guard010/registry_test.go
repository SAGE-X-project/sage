package guard010_test

import (
	"context"
	"encoding/json"
	"errors"
	g "github.com/sage-x-project/sage/pkg/agent/guard010"
	r "github.com/sage-x-project/sage/pkg/agent/registry010"
	"path/filepath"
	"strings"
	"testing"
)

type registryControl struct {
	snap            r.Snapshot
	mono, delay     int64
	reads           int
	fail, clockFail bool
}

func (c *registryControl) Now() (r.Stamp, error) {
	if c.clockFail {
		return r.Stamp{}, errors.New("clock")
	}
	return r.Stamp{MonoMS: c.mono, Unix: 1700000000}, nil
}
func (c *registryControl) Read(context.Context, string) (r.Snapshot, error) {
	c.reads++
	if c.fail {
		return r.Snapshot{}, errors.New("source")
	}
	s := c.snap
	s.AcquiredMS = c.mono
	return s, nil
}
func (c *registryControl) Advance(r.Scope, uint64, string, bool) error { c.mono += c.delay; return nil }
func registrySetup(t *testing.T) (*g.RegistryAuthority, *registryControl, guardFixture) {
	t.Helper()
	f := bridgeFixture(t)
	var env struct {
		Intent struct{ Issuer, Keyid string }
	}
	if json.Unmarshal(bridgeRaw(f), &env) != nil {
		t.Fatal("fixture")
	}
	issuer := env.Intent.Issuer
	parts := strings.Split(env.Intent.Keyid, "#")
	registry := strings.TrimSuffix(strings.TrimPrefix(issuer, "did:sage:"), ":"+strings.Split(issuer, ":")[len(strings.Split(issuer, ":"))-1])
	c := &registryControl{mono: 100, snap: r.Snapshot{Source: "fixture", Registry: registry, Network: "fixture", DID: issuer, Version: "1", State: "active", Digest: strings.Repeat("a", 64), Ready: true, Validated: true, Finalized: true, Keys: []r.Key{{Name: parts[1], Alg: "ed25519", Material: f.s("public_key_hex"), State: "accepted"}}}}
	gate, e := r.NewGate(r.Config{Source: "fixture", Registry: registry, Network: "fixture"}, c, c, c)
	if e != nil {
		t.Fatal(e)
	}
	a, e := g.NewRegistryAuthority(gate, issuer, env.Intent.Keyid)
	if e != nil {
		t.Fatal(e)
	}
	return a, c, f
}
func TestRegistryAuthorityFreshReadsAndPin(t *testing.T) {
	a, c, _ := registrySetup(t)
	ctx := context.Background()
	for range 2 {
		if _, e := a.Now(ctx); e != nil {
			t.Fatal(e)
		}
	}
	if c.reads != 2 {
		t.Fatal("positive cache")
	}
	if _, e := a.ActiveKey(ctx, c.snap.DID, c.snap.DID+"#"+c.snap.Keys[0].Name); e != nil {
		t.Fatal(e)
	}
	before := c.reads
	if _, e := a.ActiveKey(ctx, c.snap.DID, "wrong"); e == nil || c.reads != before {
		t.Fatal("wrong key")
	}
	c.snap.Keys[0].Material = strings.Repeat("00", 32)
	if _, e := a.Now(ctx); e == nil {
		t.Fatal("key substitution")
	}
}

type registrySink struct {
	*gateSink
	control *registryControl
	mode    string
}

func (s *registrySink) Check(ctx context.Context, m, tool string) error {
	if e := s.gateSink.Check(ctx, m, tool); e != nil {
		return e
	}
	if s.checks == 2 {
		switch s.mode {
		case "revoked":
			s.control.snap.Keys[0].State = "revoked"
		case "unready":
			s.control.snap.Ready = false
		case "source":
			s.control.fail = true
		case "clock":
			s.control.clockFail = true
		case "stale":
			s.control.delay = 5001
		case "boundary":
			s.control.delay = 5000
		case "rollback":
			s.control.mono = 0
		}
	}
	return nil
}
func TestRegistryAuthorityFinalDispatch(t *testing.T) {
	for _, mode := range []string{"valid", "boundary", "revoked", "unready", "source", "clock", "stale", "rollback"} {
		t.Run(mode, func(t *testing.T) {
			a, c, f := registrySetup(t)
			path := filepath.Join(t.TempDir(), "journal")
			sink := &registrySink{gateSink: &gateSink{f: f, path: path}, control: c, mode: mode}
			gate, e := g.OpenDispatchGate(path, true, f.s("expected_recipient"), a, f, sink)
			if e != nil {
				t.Fatal(e)
			}
			defer func() { _ = gate.Close() }()
			receipt, e := gate.Dispatch(context.Background(), bridgeRaw(f))
			want := mode == "valid" || mode == "boundary"
			if (e == nil) != want || sink.commits != map[bool]int{true: 1, false: 0}[want] {
				t.Fatalf("unexpected effect %v %d", e, sink.commits)
			}
			if want && (!receipt.Committed() || c.reads != 9) {
				t.Fatal("missing fresh callbacks")
			}
			if !want && stateAt(t, path) != "UNKNOWN" {
				t.Fatal("uncertain reservation reused")
			}
		})
	}
}
func TestRegistryAuthorityCancelledAndEmpty(t *testing.T) {
	a, c, _ := registrySetup(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := a.Now(ctx); e == nil || c.reads != 0 {
		t.Fatal("cancel")
	}
	var empty g.RegistryAuthority
	if _, e := empty.Now(context.Background()); e == nil {
		t.Fatal("empty")
	}
}

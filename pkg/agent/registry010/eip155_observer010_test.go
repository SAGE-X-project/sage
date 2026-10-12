package registry010

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

const eipRegistry = "eip155:11155111:0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const eipDID = "did:sage:" + eipRegistry + ":billing-bot"
const eipController = "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

type completeFixture struct {
	snapshot RecordSnapshot
	err      error
	calls    int
	stamps   []Stamp
	clockErr error
	tick     int
}

func (f *completeFixture) ReadRecord(context.Context, string) (RecordSnapshot, error) {
	f.calls++
	return f.snapshot, f.err
}
func (f *completeFixture) Now() (Stamp, error) {
	if f.clockErr != nil {
		return Stamp{}, f.clockErr
	}
	if len(f.stamps) > 0 {
		if f.tick >= len(f.stamps) {
			return Stamp{}, errors.New("clock lost")
		}
		s := f.stamps[f.tick]
		f.tick++
		return s, nil
	}
	return Stamp{MonoMS: 100, Unix: 100}, nil
}
func completeRecord(t *testing.T) map[string]any {
	t.Helper()
	r := proofFixtureFor010(t, eipDID, eipController)
	r["services"] = []any{map[string]any{"name": "e-a2a", "type": "A2AInterface", "uri": "https://agent.example/a2a"}}
	return r
}
func encodeComplete(t *testing.T, r map[string]any) []byte {
	t.Helper()
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
func newCompleteFixture(t *testing.T) (*completeFixture, *Journal, *EIP155Observer) {
	t.Helper()
	hash := strings.Repeat("a", 64)
	f := &completeFixture{snapshot: RecordSnapshot{Source: "trusted-local", Registry: eipRegistry, Network: "sepolia", Record: encodeComplete(t, completeRecord(t)), BlockHash: hash, RecordBlockHash: hash, KeysBlockHash: hash, ServicesBlockHash: hash, AcquiredMS: 100, Ready: true, Validated: true, Finalized: true}}
	j, err := OpenJournal(filepath.Join(t.TempDir(), "journal"), true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := j.Close(); err != nil {
			t.Error(err)
		}
	})
	o, err := NewEIP155Observer(Config{Source: "trusted-local", Registry: eipRegistry, Network: "sepolia", Blockchain: true}, f, f, j)
	if err != nil {
		t.Fatal(err)
	}
	return f, j, o
}
func TestCompleteEIP155Record(t *testing.T) {
	f, j, o := newCompleteFixture(t)
	s, now, err := o.Observe(t.Context(), eipDID)
	if err != nil {
		t.Fatal(err)
	}
	if now.Unix != 100 || f.calls != 1 || !bytes.Equal(s.Record, f.snapshot.Record) {
		t.Fatal("wrong observation")
	}
	s.Record[0] = 'x'
	if f.snapshot.Record[0] != '{' {
		t.Fatal("returned bytes alias reader")
	}
	w, ok := j.Get(Scope{eipRegistry, eipDID})
	if !ok || w.Version != 1 || w.Terminal || len(w.Digest) != 64 {
		t.Fatal("missing durable watermark")
	}
	if _, _, err = o.Observe(t.Context(), eipDID); err != nil || f.calls != 2 {
		t.Fatalf("read reused: %v", err)
	}
}
func TestCompleteObservationRefusals(t *testing.T) {
	cases := []struct {
		name string
		edit func(*completeFixture)
		want error
	}{
		{"reader unavailable", func(f *completeFixture) { f.err = errors.New("offline") }, ErrUnreachable},
		{"clock unavailable", func(f *completeFixture) { f.clockErr = errors.New("lost") }, ErrUnreachable},
		{"clock lost after read", func(f *completeFixture) { f.stamps = []Stamp{{100, 100}} }, ErrUnreachable},
		{"negative clock", func(f *completeFixture) { f.stamps = []Stamp{{-1, 100}} }, ErrUnreachable},
		{"unix overflow", func(f *completeFixture) { f.stamps = []Stamp{{100, 9007199254740992}} }, ErrUnreachable},
		{"clock rollback", func(f *completeFixture) { f.stamps = []Stamp{{100, 100}, {99, 100}} }, ErrStale},
		{"wall clock rollback", func(f *completeFixture) { f.stamps = []Stamp{{100, 100}, {100, 99}} }, ErrStale},
		{"not ready", func(f *completeFixture) { f.snapshot.Ready = false }, ErrUnreachable},
		{"source mismatch", func(f *completeFixture) { f.snapshot.Source = "peer" }, ErrUnreachable},
		{"registry mismatch", func(f *completeFixture) { f.snapshot.Registry = "eip155:1:x" }, ErrUnreachable},
		{"network mismatch", func(f *completeFixture) { f.snapshot.Network = "mainnet" }, ErrUnreachable},
		{"not validated", func(f *completeFixture) { f.snapshot.Validated = false }, ErrStale},
		{"not finalized", func(f *completeFixture) { f.snapshot.Finalized = false }, ErrStale},
		{"conflicting", func(f *completeFixture) { f.snapshot.Conflicting = true }, ErrStale},
		{"missing block", func(f *completeFixture) { f.snapshot.BlockHash = "" }, ErrStale},
		{"prefixed block", func(f *completeFixture) { f.snapshot.BlockHash = "0x" + f.snapshot.BlockHash }, ErrStale},
		{"mixed record block", func(f *completeFixture) { f.snapshot.RecordBlockHash = strings.Repeat("b", 64) }, ErrStale},
		{"mixed key block", func(f *completeFixture) { f.snapshot.KeysBlockHash = "" }, ErrStale},
		{"mixed services block", func(f *completeFixture) { f.snapshot.ServicesBlockHash = "" }, ErrStale},
		{"acquired before start", func(f *completeFixture) { f.snapshot.AcquiredMS = 99 }, ErrStale},
		{"acquired in future", func(f *completeFixture) { f.snapshot.AcquiredMS = 101 }, ErrStale},
		{"stale", func(f *completeFixture) { f.stamps = []Stamp{{100, 100}, {5101, 100}} }, ErrStale},
		{"oversized", func(f *completeFixture) { f.snapshot.Record = make([]byte, 65537) }, ErrSizeExceeded010},
		{"empty", func(f *completeFixture) { f.snapshot.Record = nil }, ErrInvalidRecord010},
		{"invalid JSON", func(f *completeFixture) { f.snapshot.Record = []byte("{") }, ErrInvalidRecord010},
		{"trailing value", func(f *completeFixture) { f.snapshot.Record = append(f.snapshot.Record, []byte(" {}")...) }, ErrInvalidRecord010},
		{"invalid UTF8", func(f *completeFixture) { f.snapshot.Record = []byte{255} }, ErrInvalidRecord010},
		{"BOM", func(f *completeFixture) { f.snapshot.Record = append([]byte{239, 187, 191}, f.snapshot.Record...) }, ErrInvalidRecord010},
		{"duplicate member", func(f *completeFixture) {
			f.snapshot.Record = append([]byte(`{"version":"1",`), f.snapshot.Record[1:]...)
		}, ErrInvalidRecord010},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f, j, o := newCompleteFixture(t)
			c.edit(f)
			s, _, err := o.Observe(t.Context(), eipDID)
			if !errors.Is(err, c.want) || len(s.Record) != 0 {
				t.Fatalf("got %v, want %v", err, c.want)
			}
			if _, ok := j.Get(Scope{eipRegistry, eipDID}); ok {
				t.Fatal("refusal advanced journal")
			}
		})
	}
}
func TestCompleteRecordRefusals(t *testing.T) {
	for name, edit := range map[string]func(map[string]any){
		"wrong id":               func(r map[string]any) { r["id"] = eipDID + "2" },
		"missing controller":     func(r map[string]any) { delete(r, "controller") },
		"non account controller": func(r map[string]any) { r["controller"] = "operator" },
		"uppercase controller":   func(r map[string]any) { r["controller"] = "0x" + strings.Repeat("B", 40) },
		"missing services":       func(r map[string]any) { delete(r, "services") },
		"missing version":        func(r map[string]any) { delete(r, "version") },
		"legacy active bool":     func(r map[string]any) { r["state"] = true },
		"unknown member":         func(r map[string]any) { r["nonce"] = 1 },
		"bad proof": func(r map[string]any) {
			r["keys"].([]any)[0].(map[string]any)["proof"].(map[string]any)["value"] = strings.Repeat("A", 86)
		},
		"key service collision": func(r map[string]any) { r["services"].([]any)[0].(map[string]any)["name"] = "a-ed" },
		"missing historical signer": func(r map[string]any) {
			r["keys"].([]any)[1].(map[string]any)["proof"].(map[string]any)["signer"] = eipDID + "#gone"
		},
		"unknown algorithm": func(r map[string]any) { r["keys"].([]any)[0].(map[string]any)["alg"] = "rsa" },
		"private material":  func(r map[string]any) { r["keys"].([]any)[0].(map[string]any)["d"] = "private" },
	} {
		t.Run(name, func(t *testing.T) {
			f, _, o := newCompleteFixture(t)
			r := completeRecord(t)
			edit(r)
			f.snapshot.Record = encodeComplete(t, r)
			if _, _, err := o.Observe(t.Context(), eipDID); !errors.Is(err, ErrInvalidRecord010) {
				t.Fatalf("got %v", err)
			}
		})
	}
}
func TestCompleteDurableContinuity(t *testing.T) {
	f, j, o := newCompleteFixture(t)
	if _, _, err := o.Observe(t.Context(), eipDID); err != nil {
		t.Fatal(err)
	}
	r := completeRecord(t)
	// Reuse the original keys: random optional key bytes would otherwise change.
	if err := json.Unmarshal(f.snapshot.Record, &r); err != nil {
		t.Fatal(err)
	}
	r["version"] = "2"
	f.snapshot.Record = encodeComplete(t, r)
	if _, _, err := o.Observe(t.Context(), eipDID); err != nil {
		t.Fatal(err)
	}
	r["version"] = "1"
	f.snapshot.Record = encodeComplete(t, r)
	if _, _, err := o.Observe(t.Context(), eipDID); !errors.Is(err, ErrStale) {
		t.Fatalf("rollback: %v", err)
	}
	r["version"] = "2"
	r["services"] = []any{}
	f.snapshot.Record = encodeComplete(t, r)
	if _, _, err := o.Observe(t.Context(), eipDID); !errors.Is(err, ErrStale) {
		t.Fatalf("equal version conflict: %v", err)
	}
	r["version"] = "3"
	r["state"] = "deactivated"
	f.snapshot.Record = encodeComplete(t, r)
	if _, _, err := o.Observe(t.Context(), eipDID); err != nil {
		t.Fatal(err)
	}
	r["version"] = "4"
	r["state"] = "active"
	f.snapshot.Record = encodeComplete(t, r)
	if _, _, err := o.Observe(t.Context(), eipDID); !errors.Is(err, ErrStale) {
		t.Fatalf("terminal resurrection: %v", err)
	}
	w, _ := j.Get(Scope{eipRegistry, eipDID})
	if !w.Terminal || w.Version != 3 {
		t.Fatal("terminal watermark lost")
	}
}

type completeStoreFunc func(Scope, uint64, string, bool) error

func (f completeStoreFunc) Advance(s Scope, v uint64, d string, t bool) error { return f(s, v, d, t) }
func TestCompleteFinalFreshness(t *testing.T) {
	for _, name := range []string{"late commit", "clock lost after commit", "cancel after commit", "store failure"} {
		t.Run(name, func(t *testing.T) {
			f, _, _ := newCompleteFixture(t)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			store := completeStoreFunc(func(Scope, uint64, string, bool) error {
				switch name {
				case "late commit":
					f.stamps = []Stamp{{6100, 100}}
					f.tick = 0
				case "clock lost after commit":
					f.clockErr = errors.New("clock lost")
				case "cancel after commit":
					cancel()
				case "store failure":
					return ErrUnreachable
				}
				return nil
			})
			o, err := NewEIP155Observer(Config{Source: "trusted-local", Registry: eipRegistry, Network: "sepolia", Blockchain: true}, f, f, store)
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err = o.Observe(ctx, eipDID); err == nil {
				t.Fatal("final gate accepted")
			}
		})
	}
}
func TestCompletePreflight(t *testing.T) {
	f, _, o := newCompleteFixture(t)
	if _, _, err := o.Observe(t.Context(), eipDID+":extra"); err == nil || f.calls != 0 {
		t.Fatal("invalid identifier read")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, _, err := o.Observe(ctx, eipDID); !errors.Is(err, ErrUnreachable) || f.calls != 0 {
		t.Fatal("cancelled request read")
	}
	good := Config{Source: "trusted-local", Registry: eipRegistry, Network: "sepolia", Blockchain: true}
	for name, edit := range map[string]func(*Config){"missing source": func(c *Config) { c.Source = "" }, "bad registry": func(c *Config) { c.Registry = "eip155:01:bad" }, "missing network": func(c *Config) { c.Network = "" }, "web": func(c *Config) { c.Registry = "web:agent.example" }, "not blockchain": func(c *Config) { c.Blockchain = false }} {
		t.Run(name, func(t *testing.T) {
			c := good
			edit(&c)
			if _, err := NewEIP155Observer(c, f, f, completeStoreFunc(func(Scope, uint64, string, bool) error { return nil })); err == nil {
				t.Fatal("bad config accepted")
			}
		})
	}
	if _, err := NewEIP155Observer(good, nil, f, completeStoreFunc(func(Scope, uint64, string, bool) error { return nil })); err == nil {
		t.Fatal("nil reader")
	}
	if _, err := NewEIP155Observer(good, f, nil, completeStoreFunc(func(Scope, uint64, string, bool) error { return nil })); err == nil {
		t.Fatal("nil clock")
	}
	if _, err := NewEIP155Observer(good, f, f, nil); err == nil {
		t.Fatal("nil store")
	}
}

func TestCompleteDigestAndRestart(t *testing.T) {
	f, _, o := newCompleteFixture(t)
	if _, _, err := o.Observe(t.Context(), eipDID); err != nil {
		t.Fatal(err)
	}
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, f.snapshot.Record, "", "  "); err != nil {
		t.Fatal(err)
	}
	f.snapshot.Record = pretty.Bytes()
	if _, _, err := o.Observe(t.Context(), eipDID); err != nil {
		t.Fatalf("same canonical record conflict: %v", err)
	}
	path := filepath.Join(t.TempDir(), "restart")
	j, err := OpenJournal(path, true)
	if err != nil {
		t.Fatal(err)
	}
	cfg := Config{Source: "trusted-local", Registry: eipRegistry, Network: "sepolia", Blockchain: true}
	current, err := NewEIP155Observer(cfg, f, f, j)
	if err != nil {
		t.Fatal(err)
	}
	var rec map[string]any
	if err := json.Unmarshal(f.snapshot.Record, &rec); err != nil {
		t.Fatal(err)
	}
	rec["version"] = "2"
	f.snapshot.Record = encodeComplete(t, rec)
	if _, _, err := current.Observe(t.Context(), eipDID); err != nil {
		t.Fatal(err)
	}
	if err := j.Close(); err != nil {
		t.Fatal(err)
	}
	j, err = OpenJournal(path, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := j.Close(); err != nil {
			t.Error(err)
		}
	})
	current, err = NewEIP155Observer(cfg, f, f, j)
	if err != nil {
		t.Fatal(err)
	}
	rec["version"] = "1"
	f.snapshot.Record = encodeComplete(t, rec)
	if _, _, err := current.Observe(t.Context(), eipDID); !errors.Is(err, ErrStale) {
		t.Fatalf("restart forgot finalized version: %v", err)
	}
}
func TestCompleteFiveSecondBoundary(t *testing.T) {
	f, _, o := newCompleteFixture(t)
	f.stamps = []Stamp{{100, 100}, {5100, 105}, {5100, 105}}
	if _, _, err := o.Observe(t.Context(), eipDID); err != nil {
		t.Fatalf("exact five seconds: %v", err)
	}
}

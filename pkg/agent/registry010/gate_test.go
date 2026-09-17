package registry010

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
)

type request struct {
	Action   string   `json:"action"`
	DID      string   `json:"did"`
	Snapshot Snapshot `json:"snapshot"`
	Times    []struct {
		MonoMS int64 `json:"mono_ms"`
		Unix   int64 `json:"unix"`
	} `json:"times"`
	ClockOK    bool   `json:"clock_ok"`
	SourceOK   bool   `json:"source_ok"`
	SigningURL string `json:"signing_url"`
	RequireKEM bool   `json:"require_kem"`
}
type controls struct {
	q            request
	index, reads int
}

func (c *controls) Now() (Stamp, error) {
	if !c.q.ClockOK || len(c.q.Times) == 0 {
		return Stamp{}, errors.New("clock unavailable")
	}
	i := c.index
	c.index++
	if i >= len(c.q.Times) {
		i = len(c.q.Times) - 1
	}
	v := c.q.Times[i]
	return Stamp{v.MonoMS, v.Unix}, nil
}
func (c *controls) Read(context.Context, string) (Snapshot, error) {
	c.reads++
	if !c.q.SourceOK {
		return Snapshot{}, errors.New("source unavailable")
	}
	return cloneSnapshot(c.q.Snapshot), nil
}

type fixture struct {
	Config struct {
		Source, Registry, Network string
		Blockchain                bool
	}
	Cases []struct {
		ID    string
		Steps []struct {
			Request  request
			Expected struct {
				Verdict string
				Output  map[string]any
			}
		}
	}
}

func loadFixture(t *testing.T) fixture {
	t.Helper()
	b, e := os.ReadFile("testdata/gate.json")
	if e != nil {
		t.Fatal(e)
	}
	var f fixture
	if e = json.Unmarshal(b, &f); e != nil {
		t.Fatal(e)
	}
	return f
}
func TestGateIndependentScenarios(t *testing.T) {
	f := loadFixture(t)
	if len(f.Cases) != 39 {
		t.Fatal("missing scenarios")
	}
	cfg := Config{f.Config.Source, f.Config.Registry, f.Config.Network, f.Config.Blockchain}
	for _, c := range f.Cases {
		t.Run(c.ID, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "state")
			j, e := OpenJournal(path, true)
			if e != nil {
				t.Fatal(e)
			}
			defer func() { _ = j.Close() }()
			control := &controls{}
			g, e := NewGate(cfg, control, control, j)
			if e != nil {
				t.Fatal(e)
			}
			var p *Pinned
			for _, step := range c.Steps {
				q := step.Request
				control.q = q
				control.index = 0
				out := map[string]any{}
				var err error
				switch q.Action {
				case "observe":
					var s Snapshot
					s, err = g.Observe(context.Background(), q.DID)
					if err == nil {
						out = map[string]any{"state": s.State, "version": s.Version}
					}
				case "select":
					var pin *Pinned
					pin, err = g.Select(context.Background(), q.DID, q.SigningURL, q.RequireKEM)
					if err == nil {
						p = pin
						kem := ""
						if k := p.KEM(); k != nil {
							kem = p.DID() + "#" + k.Name
						}
						out = map[string]any{"signing_keyid": p.DID() + "#" + p.Signing().Name, "kem_keyid": kem}
					}
				case "check":
					err = g.CheckPinned(context.Background(), p)
					if err == nil {
						out = map[string]any{"valid": true}
					}
				case "inspect":
					w, ok := j.Get(Scope{cfg.Registry, q.DID})
					version := "0"
					if ok {
						version = strconv.FormatUint(w.Version, 10)
					}
					out = map[string]any{"highest_finalized_version": version, "tombstone": w.Terminal}
				case "restart":
					if e = j.Close(); e != nil {
						t.Fatal(e)
					}
					j, e = OpenJournal(path, false)
					if e != nil {
						t.Fatal(e)
					}
					g, e = NewGate(cfg, control, control, j)
					if e != nil {
						t.Fatal(e)
					}
					p = nil
				default:
					t.Fatal("unknown action")
				}
				verdict := "ACCEPT"
				if err != nil {
					verdict = "REJECT"
					out = map[string]any{}
				}
				if verdict != step.Expected.Verdict || !reflect.DeepEqual(out, step.Expected.Output) {
					t.Fatalf("%s got %s %v (%v), expected %s %v", q.Action, verdict, out, err, step.Expected.Verdict, step.Expected.Output)
				}
			}
		})
	}
}
func TestJournalRejectsSecondWriterAndIncompleteState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state")
	j, e := OpenJournal(path, true)
	if e != nil {
		t.Fatal(e)
	}
	if second, e := OpenJournal(path, false); e == nil {
		_ = second.Close()
		t.Fatal("second writer opened")
	}
	s := Scope{"r", "d"}
	if e = j.Advance(s, 2, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", false); e != nil {
		t.Fatal(e)
	}
	if e = j.Close(); e != nil {
		t.Fatal(e)
	}
	j, e = OpenJournal(path, false)
	if e != nil {
		t.Fatal(e)
	}
	if e = j.Advance(s, 1, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", false); !errors.Is(e, ErrStale) {
		t.Fatal("rollback after reopen", e)
	}
	if e = j.Close(); e != nil {
		t.Fatal(e)
	}
	f, e := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = f.WriteString("{\"scope\":"); e != nil {
		t.Fatal(e)
	}
	if e = f.Close(); e != nil {
		t.Fatal(e)
	}
	if reopened, e := OpenJournal(path, false); e == nil {
		_ = reopened.Close()
		t.Fatal("incomplete state opened")
	}
	if missing, e := OpenJournal(filepath.Join(t.TempDir(), "missing"), false); e == nil {
		_ = missing.Close()
		t.Fatal("missing state recreated")
	}
}
func TestClosedStoreAndPinnedOwnership(t *testing.T) {
	f := loadFixture(t)
	q := f.Cases[0].Steps[0].Request
	cfg := Config{f.Config.Source, f.Config.Registry, f.Config.Network, true}
	// Use a valid active snapshot from the freshness scenario.
	for _, c := range f.Cases {
		if c.ID == "registry-freshness" {
			q = c.Steps[0].Request
		}
	}
	c := &controls{q: q}
	j, e := OpenJournal(filepath.Join(t.TempDir(), "state"), true)
	if e != nil {
		t.Fatal(e)
	}
	defer func() { _ = j.Close() }()
	g, e := NewGate(cfg, c, c, j)
	if e != nil {
		t.Fatal(e)
	}
	p, e := g.Select(context.Background(), q.DID, q.SigningURL, true)
	if e != nil {
		t.Fatal(e)
	}
	copy := p.Signing()
	copy.Material = "changed"
	if p.Signing().Material == copy.Material {
		t.Fatal("pinned mutation")
	}
	if e = j.Close(); e != nil {
		t.Fatal(e)
	}
	c.index = 0
	c.q.Times[0].MonoMS = c.q.Times[1].MonoMS
	if e = g.CheckPinned(context.Background(), p); e == nil {
		t.Fatal("closed store granted access")
	}
}

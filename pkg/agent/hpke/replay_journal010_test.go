package hpke

import (
	"context"
	"encoding/json"
	"github.com/sage-x-project/sage/pkg/agent/registry010"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

type journalClock010 struct{ utc, mono int64 }

func (c *journalClock010) Now() (registry010.Stamp, error) {
	return registry010.Stamp{Unix: c.utc, MonoMS: c.mono}, nil
}
func journalPlatform010(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("durable filesystem backend supports Linux/macOS")
	}
}
func TestReplayJournalVectors010(t *testing.T) {
	journalPlatform010(t)
	var f struct {
		Cases []struct {
			ID    string
			Steps []struct {
				Action                        string
				Unix                          int64
				Mono                          int64 `json:"mono_ms"`
				OK, Create, Gate              bool
				Calls                         int
				ID, Nonce, Context, Recipient string
				Expires                       int64
			}
		}
	}
	data, e := os.ReadFile("testdata/replay-journal010.json")
	if e != nil || json.Unmarshal(data, &f) != nil {
		t.Fatal("fixture")
	}
	for _, v := range f.Cases {
		t.Run(v.ID, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "replay")
			c := &journalClock010{}
			var j *ReplayJournal010
			defer func() {
				if j != nil {
					_ = j.Close()
				}
			}()
			for i, s := range v.Steps {
				c.utc, c.mono = s.Unix, s.Mono
				ok := false
				calls := 0
				switch s.Action {
				case "open":
					j, e = OpenReplayJournal010(path, s.Create, c)
					ok = e == nil
				case "close":
					ok = j.Close() == nil
				case "ready":
					ok = j.Ready()
				default:
					id, nonce, recipient, expires := s.ID, s.Nonce, s.Recipient, s.Expires
					if id == "" {
						id = "id"
					}
					if nonce == "" {
						nonce = "nonce"
					}
					if recipient == "" {
						recipient = "bob"
					}
					if expires == 0 {
						expires = 760
					}
					r := Replay010{"alice", recipient, id, nonce, s.Context, expires}
					if s.Action == "record" {
						e = j.ReserveRecord(r, func() error {
							calls++
							if !s.Gate {
								return errCompletion010
							}
							return nil
						})
					} else {
						e = j.Reserve(r)
					}
					ok = e == nil
				}
				if ok != s.OK || calls != s.Calls {
					t.Fatalf("step %d %s: ok=%v calls=%d err=%v", i, s.Action, ok, calls, e)
				}
			}
		})
	}
}
func TestReplayJournalFailures010(t *testing.T) {
	journalPlatform010(t)
	for _, kind := range []string{"lock", "partial", "blank", "capacity", "io", "concurrent"} {
		t.Run(kind, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "state")
			c := &journalClock010{100, 0}
			j, e := OpenReplayJournal010(path, true, c)
			if e != nil {
				t.Fatal(e)
			}
			defer j.Close()
			c.utc, c.mono = 460, 360000
			r := Replay010{"alice", "bob", "id", "nonce", "", 760}
			switch kind {
			case "lock":
				if _, e = OpenReplayJournal010(path, false, c); e == nil {
					t.Fatal("second writer")
				}
			case "partial", "blank":
				if e = j.Close(); e != nil {
					t.Fatal(e)
				}
				suffix := []byte("{")
				if kind == "blank" {
					suffix = []byte("\n")
				}
				f, e := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
				if e != nil {
					t.Fatal(e)
				}
				_, e = f.Write(suffix)
				if e != nil {
					t.Fatal(e)
				}
				_ = f.Close()
				if _, e = OpenReplayJournal010(path, false, c); e == nil {
					t.Fatal("corrupt accepted")
				}
			case "capacity":
				j.rows = 4096
				if j.Reserve(r) == nil {
					t.Fatal("capacity")
				}
			case "io":
				_ = j.file.Close()
				calls := 0
				if j.ReserveRecord(r, func() error { calls++; return nil }) == nil || calls != 0 || !j.failed {
					t.Fatal("storage failure")
				}
			case "concurrent":
				var wg sync.WaitGroup
				results := make(chan bool, 2)
				for range 2 {
					wg.Add(1)
					go func() { defer wg.Done(); results <- j.Reserve(r) == nil }()
				}
				wg.Wait()
				close(results)
				n := 0
				for ok := range results {
					if ok {
						n++
					}
				}
				if n != 1 {
					t.Fatal("acceptances", n)
				}
			}
		})
	}
}
func TestDurableCompletion010(t *testing.T) {
	journalPlatform010(t)
	a, b, c := completionPair(t)
	for _, e := range []*CompletionEndpoint010{a, b} {
		j, x := OpenReplayJournal010(filepath.Join(t.TempDir(), "replay"), true, c)
		if x != nil {
			t.Fatal(x)
		}
		defer j.Close()
		e.replay = j
	}
	c.utc, c.mono = 460, 360000
	ctx := context.Background()
	p, q, e := a.Start(ctx, completionBob, completionBob+"#signing-1", 300)
	if e != nil {
		t.Fatal(e)
	}
	s, r, e := b.Respond(ctx, q, 300)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	i, e := p.Complete(ctx, r)
	if e != nil {
		t.Fatal(e)
	}
	defer i.Close()
	wire, e := i.SealRequest(ctx, []byte("durable"), 300)
	if e != nil {
		t.Fatal(e)
	}
	plain, e := s.OpenRequest(ctx, wire)
	if e != nil || string(plain) != "durable" {
		t.Fatal(e)
	}
	if _, e = s.OpenRequest(ctx, wire); e == nil {
		t.Fatal("replay")
	}
}

package execution010

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

func supported(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("local durable storage requires Linux/macOS")
	}
}
func baseEntry() Entry {
	return Entry{"issuer", "executor", "call", "nonce", 1000, "7b7d", "RESERVED", ""}
}
func TestSharedVectors(t *testing.T) {
	supported(t)
	var fixture struct {
		Cases []struct {
			ID    string `json:"id"`
			Steps []struct {
				Request struct {
					Action string `json:"action"`
					Entry  Entry  `json:"entry"`
					Issuer string `json:"issuer"`
					CallID string `json:"call_id"`
				} `json:"request"`
				Expected struct {
					OK      bool   `json:"ok"`
					Changed bool   `json:"changed"`
					Entry   *Entry `json:"entry"`
				} `json:"expected"`
			} `json:"steps"`
		} `json:"cases"`
	}
	raw, e := os.ReadFile("testdata/ledger.json")
	if e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal(raw, &fixture); e != nil {
		t.Fatal(e)
	}
	for _, c := range fixture.Cases {
		t.Run(c.ID, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "ledger")
			l, e := Open(path, true)
			if e != nil {
				t.Fatal(e)
			}
			defer func() { _ = l.Close() }()
			for i, s := range c.Steps {
				changed := false
				var found *Entry
				var err error
				switch s.Request.Action {
				case "commit":
					changed, err = l.Commit(s.Request.Entry)
				case "lookup":
					var entry Entry
					var ok bool
					entry, ok, err = l.Lookup(s.Request.Issuer, s.Request.CallID)
					if ok {
						found = &entry
					}
				case "reopen":
					if err = l.Close(); err == nil {
						l, err = Open(path, false)
					}
				default:
					t.Fatal("unknown action")
				}
				if (err == nil) != s.Expected.OK || changed != s.Expected.Changed || (found == nil) != (s.Expected.Entry == nil) || (found != nil && *found != *s.Expected.Entry) {
					t.Fatalf("step %d: changed=%v found=%+v err=%v expected=%+v", i, changed, found, err, s.Expected)
				}
			}
		})
	}
}
func TestStorageFailures(t *testing.T) {
	supported(t)
	t.Run("missing-and-exclusive", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "ledger")
		if _, e := Open(p, false); e == nil {
			t.Fatal("missing reopened")
		}
		l, e := Open(p, true)
		if e != nil {
			t.Fatal(e)
		}
		defer func() { _ = l.Close() }()
		if _, e = Open(p, false); e == nil {
			t.Fatal("second writer")
		}
		if _, e = Open(p, true); e == nil {
			t.Fatal("overwritten")
		}
	})
	t.Run("partial-and-noncanonical", func(t *testing.T) {
		for _, suffix := range []string{"{", "\n", "{}\n"} {
			p := filepath.Join(t.TempDir(), "ledger")
			if e := os.WriteFile(p, []byte(header+suffix), 0600); e != nil {
				t.Fatal(e)
			}
			if _, e := Open(p, false); e == nil {
				t.Fatal("bad journal opened")
			}
		}
	})
	t.Run("poison-keeps-lock", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "ledger")
		l, e := Open(p, true)
		if e != nil {
			t.Fatal(e)
		}
		if e = l.file.Close(); e != nil {
			t.Fatal(e)
		}
		if ok, e := l.Commit(baseEntry()); e == nil || ok {
			t.Fatal("failed write accepted")
		}
		if _, _, e = l.Lookup("issuer", "call"); e == nil {
			t.Fatal("poison lookup")
		}
		_ = l.Close()
		if _, e = os.Stat(p + ".lock"); e != nil {
			t.Fatal("lost failed writer lock")
		}
	})
	t.Run("capacity", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "ledger")
		l, e := Open(p, true)
		if e != nil {
			t.Fatal(e)
		}
		defer func() { _ = l.Close() }()
		l.rows = maxRows
		if ok, e := l.Commit(baseEntry()); e == nil || ok {
			t.Fatal("row capacity")
		}
		l.rows = 0
		l.size = maxSize
		if ok, e := l.Commit(baseEntry()); e == nil || ok {
			t.Fatal("byte capacity")
		}
	})
	t.Run("recovery-conversion-keeps-lock", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "ledger")
		var journal strings.Builder
		journal.WriteString(header)
		for n := 0; n < maxRows-2; n++ {
			e := Entry{"issuer", "executor", fmt.Sprintf("terminal-%04d", n), fmt.Sprintf("nonce-%04d", n), 1000, "7b7d", "REJECTED", "7b7d"}
			row, err := json.Marshal(e)
			if err != nil {
				t.Fatal(err)
			}
			journal.Write(row)
			journal.WriteByte('\n')
		}
		pending := Entry{"issuer", "executor", "unresolved", "unresolved", 1000, "7b7d", "RESERVED", ""}
		for _, state := range []string{"RESERVED", "EXECUTING"} {
			pending.State = state
			row, err := json.Marshal(pending)
			if err != nil {
				t.Fatal(err)
			}
			journal.Write(row)
			journal.WriteByte('\n')
		}
		if err := os.WriteFile(p, []byte(journal.String()), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := Open(p, false); err == nil {
			t.Fatal("recovery conversion at capacity succeeded")
		}
		if _, err := os.Stat(p + ".lock"); err != nil {
			t.Fatal("failed recovery released exclusive ownership", err)
		}
		if _, err := Open(p, false); err == nil {
			t.Fatal("failed recovery allowed a new writer")
		}
		raw, err := os.ReadFile(p)
		if err != nil || strings.Contains(string(raw), `"state":"UNKNOWN"`) {
			t.Fatal("failed recovery fabricated an UNKNOWN row", err)
		}
	})
}
func TestConcurrentReservation(t *testing.T) {
	supported(t)
	p := filepath.Join(t.TempDir(), "ledger")
	l, e := Open(p, true)
	if e != nil {
		t.Fatal(e)
	}
	defer func() { _ = l.Close() }()
	var wg sync.WaitGroup
	wins := make(chan bool, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, e := l.Commit(baseEntry())
			if e != nil {
				t.Error(e)
			}
			wins <- ok
		}()
	}
	wg.Wait()
	close(wins)
	count := 0
	for ok := range wins {
		if ok {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("%d reservations", count)
	}
}

func TestTerminalWriteFailure(t *testing.T) {
	supported(t)
	p := filepath.Join(t.TempDir(), "ledger")
	l, err := Open(p, true)
	if err != nil {
		t.Fatal(err)
	}
	e := baseEntry()
	if _, err = l.Commit(e); err != nil {
		t.Fatal(err)
	}
	e.State = "EXECUTING"
	if _, err = l.Commit(e); err != nil {
		t.Fatal(err)
	}
	if err = l.file.Close(); err != nil {
		t.Fatal(err)
	}
	e.State = "COMPLETED"
	e.ResultHex = "7b7d"
	if changed, err := l.Commit(e); changed || err == nil {
		t.Fatal("terminal write falsely accepted")
	}
	_ = l.Close()
	// The test owns the only closed writer; explicit recovery never publishes completed.
	if err = os.Remove(p + ".lock"); err != nil {
		t.Fatal(err)
	}
	l, err = Open(p, false)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = l.Close() }()
	got, ok, err := l.Lookup(e.Issuer, e.CallID)
	if err != nil || !ok || got.State != "UNKNOWN" || got.ResultHex != "" {
		t.Fatalf("unexpected recovery: %+v %v", got, err)
	}
}
func TestRejectNoncanonicalHistory(t *testing.T) {
	supported(t)
	row, err := json.Marshal(baseEntry())
	if err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{string(row) + "\n" + string(row) + "\n", " " + string(row) + "\n", string(row[:len(row)-1]) + ",\"extra\":1}\n"} {
		p := filepath.Join(t.TempDir(), "ledger")
		if err = os.WriteFile(p, []byte(header+body), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err = Open(p, false); err == nil {
			t.Fatal("noncanonical history accepted")
		}
	}
}

func TestReserveReadsEveryExistingState(t *testing.T) {
	supported(t)
	for _, state := range []string{"RESERVED", "EXECUTING", "COMPLETED", "REJECTED", "UNKNOWN"} {
		t.Run(state, func(t *testing.T) {
			l, e := Open(filepath.Join(t.TempDir(), "journal"), true)
			if e != nil {
				t.Fatal(e)
			}
			defer func() { _ = l.Close() }()
			base := baseEntry()
			if _, _, e = l.Reserve(base); e != nil {
				t.Fatal(e)
			}
			want := base
			if state == "EXECUTING" || state == "COMPLETED" {
				want.State = "EXECUTING"
				if _, e = l.Commit(want); e != nil {
					t.Fatal(e)
				}
			}
			if state == "COMPLETED" || state == "REJECTED" {
				want.ResultHex = "7b7d"
			}
			want.State = state
			if _, e = l.Commit(want); e != nil {
				t.Fatal(e)
			}
			size, rows := l.size, l.rows
			got, changed, e := l.Reserve(base)
			if e != nil || changed || got != want || l.size != size || l.rows != rows {
				t.Fatal("read mutated terminal state", e)
			}
			bad := base
			bad.Nonce = "different"
			if _, _, e = l.Reserve(bad); e == nil {
				t.Fatal("changed identity accepted")
			}
		})
	}
}
func TestReserveFailedStorage(t *testing.T) {
	supported(t)
	l, e := Open(filepath.Join(t.TempDir(), "journal"), true)
	if e != nil {
		t.Fatal(e)
	}
	_ = l.file.Close()
	if _, changed, e := l.Reserve(baseEntry()); e == nil || changed {
		t.Fatal("failed write accepted")
	}
	if _, found, _ := l.Lookup("issuer", "call"); found {
		t.Fatal("failed identity published")
	}
	_ = l.Close()
}

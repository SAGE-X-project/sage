package registry011

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/sage-x-project/sage/pkg/agent/internal/rfc8785"
)

type historyContentFixture struct {
	Registry, Agent, Record, Grants string
	Steps                           []struct {
		Version                        uint64
		Entry, Command, Record, Grants string
	}
}

func loadHistoryContent(t *testing.T) (string, string, []byte, []byte, []HistoryContentStep) {
	t.Helper()
	raw, err := os.ReadFile("testdata/history-content.json")
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	if hex.EncodeToString(sum[:]) != "ece28aedf487937f8ff6bb6e228f0da3241fb1d80d39f01d060140c301c91cd1" {
		t.Fatal("independent content fixture changed")
	}
	var f historyContentFixture
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	rows := make([]HistoryContentStep, len(f.Steps))
	for i, s := range f.Steps {
		rows[i] = HistoryContentStep{HistoryBytes{s.Version, []byte(s.Entry), []byte(s.Command)}, []byte(s.Record), []byte(s.Grants)}
	}
	return f.Registry, f.Agent, []byte(f.Record), []byte(f.Grants), rows
}

func contentEdit(t *testing.T, raw []byte, change func(map[string]any)) []byte {
	t.Helper()
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var value map[string]any
	if err := d.Decode(&value); err != nil {
		t.Fatal(err)
	}
	change(value)
	out, err := rfc8785.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func contentHash(raw []byte) string {
	s := sha256.Sum256(raw)
	return base64.RawURLEncoding.EncodeToString(s[:])
}

func rehashContent(t *testing.T, s *HistoryContentStep) {
	t.Helper()
	s.History.Entry = contentEdit(t, s.History.Entry, func(h map[string]any) {
		h["commandDigest"], h["recordDigest"], h["grantsDigest"] = contentHash(s.History.Command), contentHash(s.Record), contentHash(s.Grants)
	})
}

func TestHistoryContentIndependentGolden(t *testing.T) {
	reg, agent, record, grants, rows := loadHistoryContent(t)
	before, _ := json.Marshal(rows)
	if err := CheckHistoryContent(reg, agent, record, grants, rows, HistoryReadLimits{4096, 8 << 20}); err != nil {
		t.Fatal(err)
	}
	after, _ := json.Marshal(rows)
	if !bytes.Equal(before, after) {
		t.Fatal("content input mutated")
	}
	// Every complete prefix agrees with its own final observed bytes.
	for i := range rows {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			if err := CheckHistoryContent(reg, agent, rows[i].Record, rows[i].Grants, rows[:i+1], HistoryReadLimits{4096, 8 << 20}); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestHistoryContentConfigurationAndBudgets(t *testing.T) {
	reg, agent, record, grants, rows := loadHistoryContent(t)
	for name, limits := range map[string]HistoryReadLimits{
		"zero": {}, "zero-items": {0, 100}, "zero-bytes": {1, 0},
		"negative-bytes": {1, -1}, "too-many-items": {4097, 100}, "too-many-bytes": {1, 8<<20 + 1},
	} {
		t.Run(name, func(t *testing.T) {
			if err := CheckHistoryContent(reg, agent, record, grants, rows, limits); !errors.Is(err, ErrConfiguration) {
				t.Fatal(err)
			}
		})
	}
	for _, scope := range [][2]string{{"", agent}, {reg + "0", agent}, {reg, ".."}, {reg, ""}} {
		if err := CheckHistoryContent(scope[0], scope[1], record, grants, rows, HistoryReadLimits{4096, 8 << 20}); !errors.Is(err, ErrConfiguration) {
			t.Fatal(err)
		}
	}
	total := len(record) + len(grants)
	for _, row := range rows {
		total += len(row.Record) + len(row.Grants) + len(row.History.Entry) + len(row.History.Command)
	}
	if err := CheckHistoryContent(reg, agent, record, grants, rows, HistoryReadLimits{uint64(len(rows)), total}); err != nil {
		t.Fatal(err)
	}
	for _, limits := range []HistoryReadLimits{{uint64(len(rows)), total - 1}, {uint64(len(rows) - 1), total}, {4096, 1}, {4096, len(record)}} {
		if err := CheckHistoryContent(reg, agent, record, grants, rows, limits); !errors.Is(err, ErrHistoryContent) {
			t.Fatal(err)
		}
	}
	if err := CheckHistoryContent(reg, agent, record, grants, nil, HistoryReadLimits{4096, 8 << 20}); !errors.Is(err, ErrHistoryContent) {
		t.Fatal(err)
	}
}

func TestHistoryContentByteAndIdentityRefusals(t *testing.T) {
	tests := map[string]func(*HistoryContentStep){
		"wrong-row-version":    func(s *HistoryContentStep) { s.History.Version = 2 },
		"empty-record":         func(s *HistoryContentStep) { s.Record = nil },
		"record-too-large":     func(s *HistoryContentStep) { s.Record = make([]byte, 65537) },
		"empty-grants":         func(s *HistoryContentStep) { s.Grants = nil },
		"grants-too-large":     func(s *HistoryContentStep) { s.Grants = make([]byte, 262145) },
		"entry-too-large":      func(s *HistoryContentStep) { s.History.Entry = make([]byte, 262145) },
		"command-too-large":    func(s *HistoryContentStep) { s.History.Command = make([]byte, 262145) },
		"empty-entry":          func(s *HistoryContentStep) { s.History.Entry = nil },
		"empty-command":        func(s *HistoryContentStep) { s.History.Command = nil },
		"noncanonical-entry":   func(s *HistoryContentStep) { s.History.Entry = append([]byte(" "), s.History.Entry...) },
		"noncanonical-command": func(s *HistoryContentStep) { s.History.Command = append(s.History.Command, '\n') },
		"noncanonical-record":  func(s *HistoryContentStep) { s.Record = append(s.Record, ' ') },
		"noncanonical-grants":  func(s *HistoryContentStep) { s.Grants = []byte("[ ]") },
		"null-grants":          func(s *HistoryContentStep) { s.Grants = []byte("null") },
		"object-grants":        func(s *HistoryContentStep) { s.Grants = []byte("{}") },
	}
	for name, change := range tests {
		t.Run(name, func(t *testing.T) {
			reg, agent, r, g, rows := loadHistoryContent(t)
			change(&rows[0])
			if err := CheckHistoryContent(reg, agent, r, g, rows, HistoryReadLimits{4096, 8 << 20}); !errors.Is(err, ErrHistoryContent) {
				t.Fatal(err)
			}
		})
	}
	for name, fields := range map[string]map[string]any{
		"history": {"version": "01", "previousVersion": nil, "actor": "bad", "operation": "bad", "admitted": "0", "target": map[string]any{}, "commandDigest": "wrong", "recordDigest": "wrong", "grantsDigest": "wrong", "automaticRemovals": nil},
		"command": {"binding": "sage-registry-chain-0.10.0", "registry": "bad", "did": "bad", "expectedVersion": nil, "operation": "activate", "actor": "bad", "arguments": nil},
		"record":  {"id": "bad", "controller": "bad", "version": json.Number("1"), "keys": nil, "services": nil, "state": "absent"},
	} {
		for field, value := range fields {
			t.Run(name+"/"+field, func(t *testing.T) {
				reg, agent, r, g, rows := loadHistoryContent(t)
				s := &rows[0]
				target := &s.History.Entry
				if name == "command" {
					target = &s.History.Command
				}
				if name == "record" {
					target = &s.Record
				}
				*target = contentEdit(t, *target, func(m map[string]any) { m[field] = value })
				if name != "history" {
					rehashContent(t, s)
				}
				if err := CheckHistoryContent(reg, agent, r, g, rows, HistoryReadLimits{4096, 8 << 20}); !errors.Is(err, ErrHistoryContent) {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestHistoryContentCoherentDigestDoesNotHideConflicts(t *testing.T) {
	for name, change := range map[string]func(*testing.T, []HistoryContentStep){
		"controller-change": func(t *testing.T, rows []HistoryContentStep) {
			rows[1].Record = contentEdit(t, rows[1].Record, func(m map[string]any) { m["controller"] = "0x0000000000000000000000000000000000002000" })
		},
		"time-rollback": func(t *testing.T, rows []HistoryContentStep) {
			rows[2].History.Entry = contentEdit(t, rows[2].History.Entry, func(m map[string]any) { m["admitted"] = 0 })
		},
		"terminal-revival": func(t *testing.T, rows []HistoryContentStep) {
			rows[1].Record = contentEdit(t, rows[1].Record, func(m map[string]any) { m["state"] = "deactivated" })
		},
		"later-create": func(t *testing.T, rows []HistoryContentStep) {
			rows[1].History.Command = contentEdit(t, rows[1].History.Command, func(m map[string]any) { m["operation"] = "create" })
		},
		"wrong-target": func(t *testing.T, rows []HistoryContentStep) {
			rows[1].History.Entry = contentEdit(t, rows[1].History.Entry, func(m map[string]any) { m["target"] = nil })
		},
		"unknown-argument": func(t *testing.T, rows []HistoryContentStep) {
			rows[0].History.Command = contentEdit(t, rows[0].History.Command, func(m map[string]any) { m["arguments"].(map[string]any)["extra"] = true })
		},
		"wrong-created-state": func(t *testing.T, rows []HistoryContentStep) {
			rows[0].Record = contentEdit(t, rows[0].Record, func(m map[string]any) { m["state"] = "active" })
		},
		"wrong-create-owner": func(t *testing.T, rows []HistoryContentStep) {
			rows[0].History.Command = contentEdit(t, rows[0].History.Command, func(m map[string]any) {
				m["arguments"].(map[string]any)["controller"] = "0x0000000000000000000000000000000000002000"
			})
		},
		"empty-keys": func(t *testing.T, rows []HistoryContentStep) {
			rows[1].Record = contentEdit(t, rows[1].Record, func(m map[string]any) { m["keys"] = []any{} })
		},
		"too-many-keys": func(t *testing.T, rows []HistoryContentStep) {
			rows[1].Record = contentEdit(t, rows[1].Record, func(m map[string]any) { m["keys"] = make([]any, 129) })
		},
		"too-many-services": func(t *testing.T, rows []HistoryContentStep) {
			rows[1].Record = contentEdit(t, rows[1].Record, func(m map[string]any) { m["services"] = make([]any, 17) })
		},
	} {
		t.Run(name, func(t *testing.T) {
			reg, agent, r, g, rows := loadHistoryContent(t)
			change(t, rows)
			for i := range rows {
				rehashContent(t, &rows[i])
			}
			if err := CheckHistoryContent(reg, agent, r, g, rows, HistoryReadLimits{4096, 8 << 20}); !errors.Is(err, ErrHistoryContent) {
				t.Fatal(err)
			}
		})
	}
	reg, agent, r, g, rows := loadHistoryContent(t)
	for _, finals := range [][2][]byte{{append(r, ' '), g}, {r, []byte("[ ]")}, {nil, g}, {make([]byte, 65537), g}, {r, nil}, {r, make([]byte, 262145)}} {
		if err := CheckHistoryContent(reg, agent, finals[0], finals[1], rows, HistoryReadLimits{4096, 8 << 20}); !errors.Is(err, ErrHistoryContent) {
			t.Fatal(err)
		}
	}
}

func TestHistoryContentCanonicalBoundaries(t *testing.T) {
	for name, raw := range map[string]string{
		"duplicate": `{"a":1,"a":2}`, "trailing": `{}null`, "invalid": `{`,
		"surrogate": `"\ud800"`, "invalid-utf8": string([]byte{0xff}),
		"space": ` {}`, "number-overflow": `1e999`, "escaped-name": `{"\u0061":0}`,
		"negative-zero": `-0`, "unicode": `"é"`, "unicode-key": `{"é":0}`, "unicode-array": `["é"]`,
		"deep": strings.Repeat("[", 33) + "0" + strings.Repeat("]", 33),
	} {
		t.Run(name, func(t *testing.T) {
			if contentJSON([]byte(raw)) != nil {
				t.Fatal("accepted invalid canonical content")
			}
		})
	}
	if contentJSON([]byte(`{"type":"\u0000\b\t\n\f\r\"\\"}`)) == nil {
		t.Fatal("ASCII escapes refused")
	}
	for _, v := range []any{nil, "0", json.Number("-1"), json.Number("1.0"), json.Number("9007199254740992"), json.Number("18446744073709551616")} {
		if _, ok := contentSeconds(v); ok {
			t.Fatal("invalid seconds accepted", v)
		}
	}
	for _, v := range []json.Number{"0", "9007199254740991"} {
		if _, ok := contentSeconds(v); !ok {
			t.Fatal(v)
		}
	}
	for _, v := range []any{nil, []any{}, map[string]any{"extra": 0}} {
		if contentObject(v, "required") != nil {
			t.Fatal("bad closed object")
		}
	}
	if contentObject(map[string]any{"wrong": 0}, "required") != nil {
		t.Fatal("missing required member")
	}
	for _, op := range []string{"create", "activate", "add-key", "revoke-key", "update-services", "authorize-operator", "revoke-operator", "deactivate"} {
		if _, ok := contentArguments(op, nil); ok {
			t.Fatal("nil argument object")
		}
	}
	if _, ok := contentArguments("unknown", map[string]any{}); ok {
		t.Fatal("unknown operation")
	}
}

func TestHistoryContentGrantOrdering(t *testing.T) {
	pair := func(operator, scope string) any { return map[string]any{"operator": operator, "scope": scope} }
	a, b := "0x0000000000000000000000000000000000001000", "0x0000000000000000000000000000000000002000"
	for _, scope := range []string{"activate", "add-key", "deactivate", "revoke-key", "update-services"} {
		if !contentGrants([]any{pair(a, scope)}) {
			t.Fatal(scope)
		}
	}
	if !contentGrants([]any{pair(a, "activate"), pair(a, "deactivate"), pair(b, "add-key")}) {
		t.Fatal("valid sorted pairs")
	}
	for _, v := range []any{nil, map[string]any{}, make([]any, 129), []any{nil}, []any{pair("bad", "activate")}, []any{pair(a, "unknown")}, []any{pair(a, "activate"), pair(a, "activate")}, []any{pair(b, "activate"), pair(a, "activate")}} {
		if contentGrants(v) {
			t.Fatal("invalid grants accepted")
		}
	}
	for _, op := range []string{"authorize-operator", "revoke-operator"} {
		args := map[string]any{"operator": a, "scope": "activate"}
		if !contentTarget(args, op, args) || contentTarget(nil, op, args) || contentTarget(pair(b, "activate"), op, args) || contentTarget(pair(a, "unknown"), op, args) {
			t.Fatal(op)
		}
	}
	if contentTarget(map[string]any{}, "activate", nil) {
		t.Fatal("unexpected target")
	}
}

func TestHistoryContentConcurrentReadOnly(t *testing.T) {
	reg, agent, r, g, rows := loadHistoryContent(t)
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			if err := CheckHistoryContent(reg, agent, r, g, rows, HistoryReadLimits{4096, 8 << 20}); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
}

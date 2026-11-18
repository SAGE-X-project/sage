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

package obs

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestEveryReasonHasOneStageAndName(t *testing.T) {
	names := map[string]Reason{}
	for _, r := range Reasons() {
		if StageOf(r) == StageUnknown {
			t.Errorf("reason %d has no stage", r)
		}
		n := r.String()
		if n == "unknown" {
			t.Errorf("reason %d has no name", r)
		}
		if prev, dup := names[n]; dup {
			t.Errorf("reasons %d and %d share name %q", prev, r, n)
		}
		names[n] = r
	}
	if StageOf(ReasonUnknown) != StageUnknown || StageOf(reasonEnd) != StageUnknown {
		t.Error("values outside the set must map to StageUnknown")
	}
}

func TestStageLabels(t *testing.T) {
	want := []string{"S0", "S1", "S2", "S3", "S4", "S5", "S6", "S7", "S8", "S9", "S10"}
	for i, w := range want {
		if got := (StageRead + Stage(i)).String(); got != w {
			t.Errorf("stage %d = %q, want %q", i, got, w)
		}
	}
}

func checkDetail(t *testing.T, in string, d Detail) {
	t.Helper()
	s := d.String()
	if len(s) > MaxDetailBytes {
		t.Fatalf("len %d > %d for input of %d bytes", len(s), MaxDetailBytes, len(in))
	}
	if !utf8.ValidString(s) {
		t.Fatalf("invalid UTF-8 output %q", s)
	}
	for i := 0; i < len(s); i++ {
		if s[i] < 0x20 || s[i] >= 0x7f {
			t.Fatalf("byte 0x%02x at %d is not printable ASCII: %q", s[i], i, s)
		}
	}
}

func TestDetailEscapesAndBounds(t *testing.T) {
	cases := map[string]string{
		"plain":       "did:sage:ethereum:0xabc",
		"newline":     "a\nlevel=ERROR msg=forged",
		"control":     "a\x00b\x1bc\x7f",
		"backslash":   `a\nb`,
		"invalid":     "a\xffb",
		"nonascii":    "ä\u202eevil",
		"huge":        strings.Repeat("A\n", 10*1024),
		"exact":       strings.Repeat("a", MaxDetailBytes),
		"exactplus":   strings.Repeat("a", MaxDetailBytes+1),
		"escapedhuge": strings.Repeat("\x01", 1000),
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			d := Unverified(in)
			checkDetail(t, in, d)
			if !d.Unverified() {
				t.Fatal("not marked unverified")
			}
		})
	}
	if got := Known("a\nb").String(); got != `a\nb` {
		t.Errorf("newline escape = %q", got)
	}
	if got := Known(`a\b`).String(); got != `a\\b` {
		t.Errorf("backslash escape = %q", got)
	}
	if got := Known(strings.Repeat("a", MaxDetailBytes)).String(); len(got) != MaxDetailBytes || strings.HasSuffix(got, "...") {
		t.Errorf("a value of exactly the limit must not be cut, got %d bytes", len(got))
	}
	if got := Known(strings.Repeat("a", MaxDetailBytes+1)).String(); len(got) != MaxDetailBytes || !strings.HasSuffix(got, "...") {
		t.Errorf("a value over the limit must end with ..., got %q", got)
	}
	if Known("x").Unverified() {
		t.Error("Known must not be marked unverified")
	}
}

func FuzzUnverified(f *testing.F) {
	f.Add("did:sage:ethereum:0xabc")
	f.Add("a\nb\x00\xff\u202e")
	f.Add(strings.Repeat("\x1b[31m", 3000))
	f.Fuzz(func(t *testing.T, in string) {
		checkDetail(t, in, Unverified(in))
	})
}

func TestSlogTracerWritesSpecAndMarksUnverified(t *testing.T) {
	var buf bytes.Buffer
	tr := SlogTracer{Logger: slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))}
	tr.Trace(context.Background(), Event{
		Spec:    "0.9.0-draft.1.1",
		Outcome: OutcomeRejected,
		Stage:   StageReplay,
		Reason:  ReasonReplay,
		Subject: Unverified(strings.Repeat("did\n", 1000)),
	})
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("log is not one JSON object: %v: %s", err, buf.String())
	}
	if got["spec"] != "0.9.0-draft.1.1" || got["stage"] != "S8" || got["reason"] != "replay" {
		t.Fatalf("log = %v", got)
	}
	subject, ok := got["subject"].(map[string]any)
	if !ok || subject["unverified"] != true {
		t.Fatalf("subject = %v", got["subject"])
	}
	if v, _ := subject["value"].(string); len(v) > MaxDetailBytes {
		t.Fatalf("subject value %d bytes", len(v))
	}
	if strings.Count(buf.String(), "\n") != 1 {
		t.Fatalf("one event produced more than one log line: %q", buf.String())
	}
}

func TestEmitNilAndPanic(t *testing.T) {
	Emit(context.Background(), nil, Event{})
	Emit(context.Background(), SlogTracer{}, Event{})
	Emit(context.Background(), panicTracer{}, Event{})
}

type panicTracer struct{}

func (panicTracer) Trace(context.Context, Event) { panic("boom") }

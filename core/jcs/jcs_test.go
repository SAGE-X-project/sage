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

package jcs

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/sage-x-project/sage/core"
	"github.com/sage-x-project/sage/core/obs"
)

func reasonOf(t *testing.T, err error) obs.Reason {
	t.Helper()
	var r *core.Rejection
	if !errors.As(err, &r) {
		t.Fatalf("error %v is not a core.Rejection", err)
	}
	if !errors.Is(err, core.ErrAuthentication) {
		t.Fatalf("rejection does not unwrap to ErrAuthentication")
	}
	return r.Reason()
}

func numbersAllowed() Limits {
	l := SignedLimits()
	l.Numbers = true
	return l
}

func TestCanonicalizeRFC8785(t *testing.T) {
	cases := []struct{ in, want string }{
		{`{"numbers": [333333333.33333329, 1E30, 4.50, 2e-3, 0.000000000000000000000000001], "string": "\u20ac$\u000F\u000aA'\u0042\u0022\u005c\\\"\/", "literals": [null, true, false]}`,
			`{"literals":[null,true,false],"numbers":[333333333.3333333,1e+30,4.5,0.002,1e-27],"string":"€$\u000f\nA'B\"\\\\\"/"}`},
		{` [ ] `, `[]`},
		{`{}`, `{}`},
		{`"\u2028\u007f\b\f\t\r"`, "\"\u2028\u007f\\b\\f\\t\\r\""},
		{`{"b":1,"a":{"d":[true],"c":null}}`, `{"a":{"c":null,"d":[true]},"b":1}`},
		{`"\ud83d\ude02"`, `"😂"`},
		// Arrays push items but no names; objects in arrays in objects.
		{`[0,0,{"":[]}]`, `[0,0,{"":[]}]`},
		{`["a",{"y":[{"q":"1","p":["x",{}]}],"x":"b"},"c",{"k":{"j":"2","i":[]}}]`,
			`["a",{"x":"b","y":[{"p":["x",{}],"q":"1"}]},"c",{"k":{"i":[],"j":"2"}}]`},
	}
	for _, c := range cases {
		got, err := Canonicalize([]byte(c.in), numbersAllowed())
		if err != nil {
			t.Fatalf("%s: %v", c.in, err)
		}
		if string(got) != c.want {
			t.Errorf("%s:\n got  %s\n want %s", c.in, got, c.want)
		}
	}
}

func TestUTF16Sorting(t *testing.T) {
	in := `{"€":1,"\r":2,"\ufb33":3,"1":4,"😂":5,"\u0080":6,"ö":7,"\n":8}`
	got, err := Canonicalize([]byte(in), numbersAllowed())
	if err != nil {
		t.Fatal(err)
	}
	want := "{\"\\n\":8,\"\\r\":2,\"1\":4,\"\u0080\":6,\"ö\":7,\"€\":1,\"😂\":5,\"\ufb33\":3}"
	if string(got) != want {
		t.Fatalf("got %s want %s", got, want)
	}
}

// TestNumberFormat uses the samples of RFC 8785 Appendix B.
func TestNumberFormat(t *testing.T) {
	cases := map[uint64]string{
		0x0000000000000000: "0",
		0x8000000000000000: "0",
		0x0000000000000001: "5e-324",
		0x8000000000000001: "-5e-324",
		0x7fefffffffffffff: "1.7976931348623157e+308",
		0xffefffffffffffff: "-1.7976931348623157e+308",
		0x4340000000000000: "9007199254740992",
		0xc340000000000000: "-9007199254740992",
		0x4430000000000000: "295147905179352830000",
		0x44b52d02c7e14af5: "9.999999999999997e+22",
		0x44b52d02c7e14af6: "1e+23",
		0x44b52d02c7e14af7: "1.0000000000000001e+23",
		0x444b1ae4d6e2ef4e: "999999999999999700000",
		0x444b1ae4d6e2ef4f: "999999999999999900000",
		0x444b1ae4d6e2ef50: "1e+21",
		0x3eb0c6f7a0b5ed8c: "9.999999999999997e-7",
		0x3eb0c6f7a0b5ed8d: "0.000001",
		0x41b3de4355555553: "333333333.3333332",
		0x41b3de4355555554: "333333333.33333325",
		0x41b3de4355555555: "333333333.3333333",
		0x41b3de4355555556: "333333333.3333334",
		0x41b3de4355555557: "333333333.33333343",
		0xbecbf647612f3696: "-0.0000033333333333333333",
		0x43143ff3c1cb0959: "1424953923781206.2",
	}
	for bits, want := range cases {
		if got := formatNumber(math.Float64frombits(bits)); got != want {
			t.Errorf("%#016x: got %s want %s", bits, got, want)
		}
	}
}

func TestStrictRejects(t *testing.T) {
	deep := strings.Repeat(`{"a":`, 17) + `""` + strings.Repeat(`}`, 17)
	okDeep := strings.Repeat(`[`, 16) + strings.Repeat(`]`, 16)
	many := func(n int, obj bool) string {
		var b strings.Builder
		if obj {
			b.WriteString("{")
		} else {
			b.WriteString("[")
		}
		for i := 0; i < n; i++ {
			if i > 0 {
				b.WriteString(",")
			}
			if obj {
				b.WriteString(`"k` + strings.Repeat("x", i) + `":""`)
			} else {
				b.WriteString(`""`)
			}
		}
		if obj {
			b.WriteString("}")
		} else {
			b.WriteString("]")
		}
		return b.String()
	}
	big := `{"d":"` + strings.Repeat("a", MaxSignedBytes-8) + `"}`
	cases := map[string]struct {
		in     string
		reason obs.Reason
	}{
		"duplicate top":        {`{"a":"x","a":"y"}`, obs.ReasonDuplicate},
		"duplicate nested":     {`{"a":{"b":"x","b":"x"}}`, obs.ReasonDuplicate},
		"duplicate by escape":  {`{"a":"x","\u0061":"y"}`, obs.ReasonDuplicate},
		"case variant":         {`{"v":"v1","V":"v1"}`, obs.ReasonDuplicate},
		"lone high surrogate":  {`{"a":"\ud800"}`, obs.ReasonMalformed},
		"high then non-low":    {`{"a":"\ud800\u0041"}`, obs.ReasonMalformed},
		"lone low surrogate":   {`{"a":"\udc00"}`, obs.ReasonMalformed},
		"high then high":       {`{"a":"\ud800\ud800"}`, obs.ReasonMalformed},
		"high then raw":        {`{"a":"\ud800x"}`, obs.ReasonMalformed},
		"high at end":          {`{"a":"\ud800"`, obs.ReasonMalformed},
		"cesu surrogate":       {"{\"a\":\"\xed\xa0\x80\"}", obs.ReasonMalformed},
		"form feed ws":         {"{}\f", obs.ReasonMalformed},
		"literal suffix":       {`[truex]`, obs.ReasonMalformed},
		"257 then comma":       {many(MaxEntries, false)[:len(many(MaxEntries, false))-1] + `,]`, obs.ReasonMalformed},
		"257 members, comma":   {many(MaxEntries, true)[:len(many(MaxEntries, true))-1] + `,}`, obs.ReasonMalformed},
		"surrogate in name":    {`{"\ud800":"a"}`, obs.ReasonMalformed},
		"invalid utf8":         {"{\"a\":\"\xff\"}", obs.ReasonMalformed},
		"overlong utf8":        {"{\"a\":\"\xc0\xaf\"}", obs.ReasonMalformed},
		"bom":                  {"\ufeff{}", obs.ReasonMalformed},
		"trailing data":        {`{} {}`, obs.ReasonMalformed},
		"trailing comma":       {`{"a":"b",}`, obs.ReasonMalformed},
		"raw control":          {"{\"a\":\"\x01\"}", obs.ReasonMalformed},
		"bad escape":           {`{"a":"\x"}`, obs.ReasonMalformed},
		"short unicode escape": {`{"a":"\u12"}`, obs.ReasonMalformed},
		"single quotes":        {`{'a':"b"}`, obs.ReasonMalformed},
		"empty":                {``, obs.ReasonMalformed},
		"only space":           {`  `, obs.ReasonMalformed},
		"unterminated":         {`{"a":"b"`, obs.ReasonMalformed},
		"bad literal":          {`[nul]`, obs.ReasonMalformed},
		"number signed":        {`{"version":1}`, obs.ReasonMalformed},
		"nul byte":             {"{}\x00", obs.ReasonMalformed},
		"vertical tab ws":      {"{}\v", obs.ReasonMalformed},
		"too deep":             {deep, obs.ReasonTooLarge},
		"too many members":     {many(MaxEntries+1, true), obs.ReasonTooLarge},
		"too many elements":    {many(MaxEntries+1, false), obs.ReasonTooLarge},
		"too large":            {big + strings.Repeat(" ", MaxSignedBytes+1-len(big)), obs.ReasonTooLarge},
	}
	for name, c := range cases {
		_, err := Parse([]byte(c.in), SignedLimits())
		if err == nil {
			t.Errorf("%s: accepted", name)
			continue
		}
		if got := reasonOf(t, err); got != c.reason {
			t.Errorf("%s: reason %s, want %s", name, got, c.reason)
		}
		if err.Error() != core.AuthenticationFailedText {
			t.Errorf("%s: error text %q", name, err)
		}
	}
	for name, in := range map[string]string{
		"depth 16":    okDeep,
		"256 members": many(MaxEntries, true),
		"256 items":   many(MaxEntries, false),
		"64 KiB":      big + strings.Repeat(" ", MaxSignedBytes-len(big)),
		"ws around":   " \t\r\n{} \t\r\n",
	} {
		if _, err := Parse([]byte(in), SignedLimits()); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestBadNumbers(t *testing.T) {
	for _, in := range []string{"01", "-", "1.", ".1", "1e", "1e+", "+1", "0x10", "1_0", "Infinity", "NaN", "1e400", "-1e400", "--1", "1.e5"} {
		if _, err := Parse([]byte(in), numbersAllowed()); err == nil {
			t.Errorf("%q accepted", in)
		} else if r := reasonOf(t, err); r != obs.ReasonMalformed {
			t.Errorf("%q: %s", in, r)
		}
	}
	for in, want := range map[string]string{"-0": "0", "1e-400": "0", "-0.0e0": "0", "1E+2": "100", "0.5": "0.5"} {
		got, err := Canonicalize([]byte(in), numbersAllowed())
		if err != nil || string(got) != want {
			t.Errorf("%q: %s %v", in, got, err)
		}
	}
}

func TestLimitsValidated(t *testing.T) {
	for _, l := range []Limits{{}, {MaxBytes: 1, MaxDepth: 0, MaxEntries: 1}, {MaxBytes: 1, MaxDepth: 1, MaxEntries: 0}, {MaxBytes: 1, MaxDepth: HardMaxDepth + 1, MaxEntries: 1}} {
		_, err := Parse([]byte(`{}`), l)
		if !errors.Is(err, ErrLimits) {
			t.Errorf("%+v: %v", l, err)
		}
		var r *core.Rejection
		if errors.As(err, &r) {
			t.Errorf("%+v: invalid limits must not look like a peer rejection", l)
		}
	}
	if l := TransportLimits(); l.MaxBytes != 1<<20 || l.MaxDepth != 16 || l.MaxEntries != 256 || l.Numbers {
		t.Errorf("transport limits %+v", l)
	}
	if l := SignedLimits(); l.MaxBytes != 64<<10 || l.MaxDepth != 16 || l.MaxEntries != 256 || l.Numbers {
		t.Errorf("signed limits %+v", l)
	}
}

func TestValueAccess(t *testing.T) {
	v, err := Parse([]byte(`{"proof":{"sig":"x"},"name":"n","list":["a",true,null],"ok":false}`), SignedLimits())
	if err != nil {
		t.Fatal(err)
	}
	if v.Kind() != KindObject || v.Len() != 4 {
		t.Fatalf("kind %v len %d", v.Kind(), v.Len())
	}
	if names := v.Names(); strings.Join(names, ",") != "proof,name,list,ok" {
		t.Fatalf("names %v", names)
	}
	names := v.Names()
	names[0] = "changed"
	if v.Names()[0] != "proof" {
		t.Fatal("Names exposes internal state")
	}
	if s, ok := mustGet(t, v, "name").Str(); !ok || s != "n" {
		t.Fatal("Str")
	}
	if _, ok := v.Get("Name"); ok {
		t.Fatal("Get must be case-sensitive")
	}
	list := mustGet(t, v, "list")
	if b, ok := mustIndex(t, list, 1).Bool(); !ok || !b {
		t.Fatal("Bool")
	}
	if mustIndex(t, list, 2).Kind() != KindNull {
		t.Fatal("Null")
	}
	if _, ok := list.Index(3); ok {
		t.Fatal("Index out of range")
	}
	if _, ok := list.Index(-1); ok {
		t.Fatal("negative Index")
	}
	if _, ok := mustGet(t, v, "name").Bool(); ok {
		t.Fatal("Bool on a string")
	}
	rest, removed, ok := v.Without("proof")
	if !ok || removed.Kind() != KindObject || rest.Len() != 3 || v.Len() != 4 {
		t.Fatal("Without")
	}
	c, err := Canonical(rest)
	if err != nil || string(c) != `{"list":["a",true,null],"name":"n","ok":false}` {
		t.Fatalf("%s %v", c, err)
	}
	if _, _, ok := v.Without("missing"); ok {
		t.Fatal("Without of a missing member")
	}
	if _, _, ok := list.Without("x"); ok {
		t.Fatal("Without on an array")
	}
	if _, err := Canonical(Value{}); !errors.Is(err, ErrInvalidValue) {
		t.Fatalf("zero Value: %v", err)
	}
	var r *core.Rejection
	if _, err := Canonical(Value{}); errors.As(err, &r) {
		t.Fatal("a caller bug must not look like a peer rejection")
	}
	n, err := Parse([]byte(`[2.5]`), numbersAllowed())
	if err != nil {
		t.Fatal(err)
	}
	if f, ok := mustIndex(t, n, 0).Number(); !ok || f != 2.5 {
		t.Fatal("Number")
	}
}

func mustGet(t *testing.T, v Value, name string) Value {
	t.Helper()
	m, ok := v.Get(name)
	if !ok {
		t.Fatalf("no member %q", name)
	}
	return m
}

func mustIndex(t *testing.T, v Value, i int) Value {
	t.Helper()
	m, ok := v.Index(i)
	if !ok {
		t.Fatalf("no element %d", i)
	}
	return m
}

// FuzzParse checks that the parser never panics, accepts only text that
// encoding/json also accepts, and that canonical output is a fixed point.
func FuzzParse(f *testing.F) {
	for _, s := range []string{`{"a":[1,2.5e3,"\u00e9"]}`, `[]`, `"\ud83d\ude02"`, `{"a":"x","a":"y"}`, "\ufeff{}", `{"v":"1","V":"2"}`, `-0.0e-0`} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		v, err := Parse(data, numbersAllowed())
		if err != nil {
			var r *core.Rejection
			if !errors.As(err, &r) {
				t.Fatalf("non-rejection error %v", err)
			}
			return
		}
		if !json.Valid(data) {
			t.Fatalf("accepted text encoding/json rejects: %q", data)
		}
		c, err := Canonical(v)
		if err != nil {
			t.Fatal(err)
		}
		// Canonical numbers can be longer than their input ("1e20").
		wide := numbersAllowed()
		wide.MaxBytes = 64 * MaxSignedBytes
		again, err := Canonicalize(c, wide)
		if err != nil {
			t.Fatalf("canonical output rejected: %q: %v", c, err)
		}
		if !bytes.Equal(c, again) {
			t.Fatalf("not a fixed point: %q -> %q", c, again)
		}
		var a, b any
		if json.Unmarshal(data, &a) != nil || json.Unmarshal(c, &b) != nil {
			t.Fatal("unmarshal")
		}
		// reflect.DeepEqual compares numbers with ==, so -0 equals 0.
		if !reflect.DeepEqual(a, b) {
			t.Fatalf("canonical output changes the value: %q vs %q", data, c)
		}
	})
}

func TestTransportAndDepthBoundaries(t *testing.T) {
	pad := func(n int) []byte {
		b := []byte(`{"d":""}`)
		return append(b, bytes.Repeat([]byte(" "), n-len(b))...)
	}
	if _, err := Parse(pad(MaxTransportBytes), TransportLimits()); err != nil {
		t.Fatalf("1 MiB: %v", err)
	}
	if _, err := Parse(pad(MaxTransportBytes+1), TransportLimits()); reasonOf(t, err) != obs.ReasonTooLarge {
		t.Fatal("1 MiB + 1 accepted")
	}
	lim := SignedLimits()
	lim.MaxDepth = HardMaxDepth
	nested := strings.Repeat("[", HardMaxDepth) + strings.Repeat("]", HardMaxDepth)
	if _, err := Parse([]byte(nested), lim); err != nil {
		t.Fatalf("depth %d: %v", HardMaxDepth, err)
	}
	if _, err := Parse([]byte("["+nested+"]"), lim); reasonOf(t, err) != obs.ReasonTooLarge {
		t.Fatal("depth above HardMaxDepth accepted")
	}
}

func TestCompareUTF16(t *testing.T) {
	ref := func(a, b string) int {
		return slices.Compare(utf16.Encode([]rune(a)), utf16.Encode([]rune(b)))
	}
	words := []string{"", "a", "ab", "b", "\u00f6", "\u20ac", "\ue000", "\uffff", "\U00010000", "\U0001f602", "\U0010ffff", "\ufb33", "a\U0001f602", "a\uffff", "\U0001f602a", "\U0001f603"}
	for _, a := range words {
		for _, b := range words {
			if got, want := compareUTF16(a, b), ref(a, b); got != want {
				t.Errorf("compareUTF16(%q, %q) = %d, want %d", a, b, got, want)
			}
		}
	}
}

// TestMemoryBound pins the cost of the worst texts the security review found:
// many empty containers and many one-member objects.
func TestMemoryBound(t *testing.T) {
	inner := "[" + strings.TrimSuffix(strings.Repeat("[],", MaxEntries), ",") + "]"
	outer := "[" + strings.TrimSuffix(strings.Repeat(inner+",", MaxEntries), ",") + "]"
	members := "[" + strings.TrimSuffix(strings.Repeat(`{"":""},`, MaxEntries), ",") + "]"
	for name, text := range map[string]string{"empty arrays": outer, "small objects": "[" + strings.TrimSuffix(strings.Repeat(members+",", 128), ",") + "]"} {
		data := []byte(text)
		var v Value
		allocs := testing.AllocsPerRun(1, func() { v, _ = Parse(data, TransportLimits()) })
		var before, after runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&before)
		v, err := Parse(data, TransportLimits())
		runtime.ReadMemStats(&after)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		perByte := float64(after.TotalAlloc-before.TotalAlloc) / float64(len(data))
		t.Logf("%s: %d bytes, %.1f bytes allocated per input byte, %.0f allocs", name, len(data), perByte, allocs)
		if perByte > 40 {
			t.Errorf("%s: %.1f bytes allocated per input byte", name, perByte)
		}
		runtime.KeepAlive(v)
	}
}

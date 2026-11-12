package compilermemory

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"testing"
	"time"
)

func jsonTestRegion(t *testing.T, action context.Context) *Region {
	t.Helper()
	cleanup, cancel := context.WithTimeout(context.Background(), time.Minute)
	r, err := acquireWithOperations(action, cleanup, 32<<10, &simulatedRegionOperations{})
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := r.Close(); err != nil {
			t.Error(err)
		}
		cancel()
	})
	return r
}

func jsonTestDocument(t *testing.T, raw string) (*Region, JSONDocument) {
	t.Helper()
	action, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	r := jsonTestRegion(t, action)
	s, err := r.Store([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	d, err := r.ParseJSON(s, JSONLimits{MaximumDepth: 16, MaximumNodes: 256})
	if err != nil {
		t.Fatal(err)
	}
	return r, d
}

func jsonTestMember(t *testing.T, value JSONValue, name string) JSONValue {
	t.Helper()
	child, found, err := value.Member(name)
	if err != nil || !found {
		t.Fatalf("member %q: found=%v, error=%v", name, found, err)
	}
	return child
}

func TestRegionJSONDocumentGrammarAndOriginalBytes(t *testing.T) {
	for _, raw := range []string{
		`0`, `-0`, `1.25`, `1e2`, `-1E-3`, `true`, `false`, `null`,
		`""`, `"\"\\\/\b\f\n\r\t"`, `"한글"`, `"\u0061\u00Af\uABCD"`,
		`"\ud83d\ude00"`, `"\ud800"`, `"\udfff"`, `"\ud800\u0061"`,
		`{}`, `[]`, `{"x":[],"y":{},"z":[{},[]]}`,
		`{"same":{"same":0},"array":[true,false,null]}`,
		`{"":0,"a":1,"ab":2,"b":3}`, " \n\t[0, 1, -2, 3.5e+2]\r ",
	} {
		t.Run(raw, func(t *testing.T) {
			if !json.Valid([]byte(raw)) {
				t.Fatal("invalid standard-library reference fixture")
			}
			r, d := jsonTestDocument(t, raw)
			s, err := d.Root().Raw()
			if err != nil {
				t.Fatal(err)
			}
			out := make([]byte, len(strings.TrimSpace(raw)))
			if err := r.Read(s, out); err != nil || string(out) != strings.TrimSpace(raw) {
				t.Fatalf("original token bytes: %q, %v", out, err)
			}
			if _, err := r.Store([]byte("x")); !errors.Is(err, ErrRange) {
				t.Fatalf("successful parse did not seal storage: %v", err)
			}
			if _, err := r.ParseJSON(s, JSONLimits{MaximumDepth: 1, MaximumNodes: 1}); !errors.Is(err, ErrRange) {
				t.Fatalf("sealed storage reparsed: %v", err)
			}
		})
	}
}

func TestRegionJSONDocumentRejectsMalformedAndDuplicateData(t *testing.T) {
	for _, raw := range []string{
		``, ` `, `01`, `-01`, `+1`, `-`, `.1`, `1.`, `1e`, `1e+`, `1e-`, `1x`,
		`[`, `{`, `[1`, `{"a"`, `{"a":`, `{"a":0`, `{"a" 0}`, `{"a":}`, `{a:0}`,
		`[0,]`, `{"a":0,}`, `[,]`, `{,}`, `[true false]`, `{"a":0 "b":1}`,
		`[}`, `{]`, `truefalse`, `tru`, `fals`, `nul`, `True`, `NULL`,
		`"`, `"\"`, `"\q"`, `"\u123"`, `"\u12x4"`, `"\ud800\uZZZZ"`,
		`{"a":1,"a":2}`, `{"a":1,"\u0061":2}`, `{"":1,"":2}`,
		`{"\ud800":1,"\udfff":2}`, `{"nested":{"a":1,"a":2}}`,
		"\xef\xbb\xbf0", "\"\xff\"", "\"\x00\"", "0\x00", "0 1",
	} {
		t.Run(raw, func(t *testing.T) {
			action, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			r := jsonTestRegion(t, action)
			s, err := r.Store([]byte(raw))
			if err != nil {
				t.Fatal(err)
			}
			d, err := r.ParseJSON(s, JSONLimits{MaximumDepth: 16, MaximumNodes: 256})
			if err == nil || d.owner != nil {
				t.Fatalf("malformed document accepted: %v", err)
			}
			if _, err := r.JSONUint64(s); !errors.Is(err, ErrClosed) {
				t.Fatalf("entered parse failure remained usable: %v", err)
			}
		})
	}
}

func TestRegionJSONValuesUseOriginalTreeAndIntegerConsumers(t *testing.T) {
	r, d := jsonTestDocument(t, `{"nested":{"v":7},"clock":{"maximum":18446744073709551615,"signed":-9223372036854775808,"exact":9007199254740993},"items":[{"v":9},false,null],"label":"\ud83d\ude00"}`)
	root := d.Root()
	if kind, err := root.Kind(); kind != JSONObject || err != nil {
		t.Fatalf("root kind: %v, %v", kind, err)
	}
	if n, err := root.Length(); n != 4 || err != nil {
		t.Fatalf("root members: %d, %v", n, err)
	}
	clock := jsonTestMember(t, root, "clock")
	maximum := jsonTestMember(t, clock, "maximum")
	if value, err := maximum.JSONUint64(); value != math.MaxUint64 || err != nil {
		t.Fatalf("unsigned: %d, %v", value, err)
	}
	if value, err := jsonTestMember(t, clock, "signed").JSONInt64(); value != math.MinInt64 || err != nil {
		t.Fatalf("signed: %d, %v", value, err)
	}
	if value, err := jsonTestMember(t, clock, "exact").JSONUint64(); value != 9007199254740993 || err != nil {
		t.Fatalf("exact: %d, %v", value, err)
	}
	items := jsonTestMember(t, root, "items")
	if n, err := items.Length(); n != 3 || err != nil {
		t.Fatalf("array length: %d, %v", n, err)
	}
	first, err := items.Index(0)
	if err != nil {
		t.Fatal(err)
	}
	if n, err := jsonTestMember(t, first, "v").JSONInt64(); n != 9 || err != nil {
		t.Fatalf("nested array member: %d, %v", n, err)
	}
	second, err := items.Index(1)
	if err != nil {
		t.Fatal(err)
	}
	if b, err := second.Boolean(); b || err != nil {
		t.Fatalf("false value: %v, %v", b, err)
	}
	third, err := items.Index(2)
	if err != nil {
		t.Fatal(err)
	}
	if kind, err := third.Kind(); kind != JSONNull || err != nil {
		t.Fatalf("null value: %v, %v", kind, err)
	}
	if equal, err := jsonTestMember(t, root, "label").StringEqual("😀"); !equal || err != nil {
		t.Fatalf("decoded label: %v, %v", equal, err)
	}
	if _, found, err := root.Member("missing"); found || err != nil {
		t.Fatalf("missing member: %v, %v", found, err)
	}
	for _, index := range []int{-1, 3, 99} {
		if _, err := items.Index(index); !errors.Is(err, ErrRange) {
			t.Fatalf("array index %d: %v", index, err)
		}
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := maximum.JSONUint64(); !errors.Is(err, ErrClosed) {
		t.Fatalf("closed integer handle: %v", err)
	}
}

func TestRegionJSONStringsMatchDecodedKeysWithoutGoTree(t *testing.T) {
	for _, raw := range []string{
		`""`, `"a"`, `"ab"`, `"\"\\\/\b\f\n\r\t"`,
		`"한글"`, `"\u0061\u00Af\uABCD"`, `"\ud83d\ude00"`,
		`"\ud800"`, `"\udfff"`, `"\ud800\u0061"`,
	} {
		t.Run(raw, func(t *testing.T) {
			var expected string
			if err := json.Unmarshal([]byte(raw), &expected); err != nil {
				t.Fatal(err)
			}
			_, d := jsonTestDocument(t, raw)
			if equal, err := d.Root().StringEqual(expected); !equal || err != nil {
				t.Fatalf("decoded string %q: %v, %v", expected, equal, err)
			}
			for _, other := range []string{expected + "x", "different", "\xff"} {
				if equal, err := d.Root().StringEqual(other); equal || err != nil {
					t.Fatalf("different string %q: %v, %v", other, equal, err)
				}
			}
		})
	}
	_, d := jsonTestDocument(t, `{"\u0061":true,"":false,"ab":null}`)
	if b, err := jsonTestMember(t, d.Root(), "a").Boolean(); !b || err != nil {
		t.Fatalf("escaped key: %v, %v", b, err)
	}
	if b, err := jsonTestMember(t, d.Root(), "").Boolean(); b || err != nil {
		t.Fatalf("empty key: %v, %v", b, err)
	}
}

func TestRegionJSONLimitsAndOwnerRefusals(t *testing.T) {
	action, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	r := jsonTestRegion(t, action)
	s, err := r.Store([]byte(`{"a":0}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := (*Region)(nil).ParseJSON(s, JSONLimits{1, 1}); !errors.Is(err, ErrConfiguration) {
		t.Fatalf("nil region: %v", err)
	}
	if _, err := (&Region{}).ParseJSON(s, JSONLimits{1, 1}); !errors.Is(err, ErrConfiguration) {
		t.Fatalf("uninitialized region: %v", err)
	}
	other := jsonTestRegion(t, action)
	if _, err := other.ParseJSON(s, JSONLimits{1, 1}); !errors.Is(err, ErrRange) {
		t.Fatalf("foreign span: %v", err)
	}
	for _, limits := range []JSONLimits{{0, 1}, {65, 1}, {1, 0}, {1, math.MaxInt}} {
		if _, err := r.ParseJSON(s, limits); err == nil {
			t.Fatalf("limits accepted: %+v", limits)
		}
	}
	for _, tc := range []struct {
		raw    string
		limits JSONLimits
	}{
		{`{"a":0}`, JSONLimits{1, 1}},
		{`[[0]]`, JSONLimits{1, 16}},
	} {
		region := jsonTestRegion(t, action)
		span, err := region.Store([]byte(tc.raw))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := region.ParseJSON(span, tc.limits); !errors.Is(err, ErrRange) {
			t.Fatalf("exhausted limits: %v", err)
		}
	}
	if _, err := jsonOffset(math.MaxUint64, 1); !errors.Is(err, ErrJSON) {
		t.Fatalf("oversized offset: %v", err)
	}
	if _, err := jsonOffset(0, -1); !errors.Is(err, ErrJSON) {
		t.Fatalf("negative extent: %v", err)
	}
}

func TestRegionJSONValueTypesAndEmptyContainers(t *testing.T) {
	_, d := jsonTestDocument(t, `{"number":1.5,"text":"x","boolean":true,"object":{},"array":[]}`)
	root := d.Root()
	text := jsonTestMember(t, root, "text")
	if n, err := text.JSONUint64(); n != 0 || !errors.Is(err, ErrJSONType) {
		t.Fatalf("text as integer: %d, %v", n, err)
	}
	if _, err := root.JSONInt64(); !errors.Is(err, ErrJSONType) {
		t.Fatalf("object as integer: %v", err)
	}
	if n, err := jsonTestMember(t, root, "number").JSONUint64(); n != 0 || !errors.Is(err, ErrJSONInteger) {
		t.Fatalf("fraction as integer: %d, %v", n, err)
	}
	if _, err := text.Length(); !errors.Is(err, ErrJSONType) {
		t.Fatalf("text length: %v", err)
	}
	if _, _, err := text.Member("x"); !errors.Is(err, ErrJSONType) {
		t.Fatalf("text member: %v", err)
	}
	if _, err := text.Index(0); !errors.Is(err, ErrJSONType) {
		t.Fatalf("text index: %v", err)
	}
	if _, err := root.StringEqual("x"); !errors.Is(err, ErrJSONType) {
		t.Fatalf("object string: %v", err)
	}
	if b, err := text.Boolean(); b || !errors.Is(err, ErrJSONType) {
		t.Fatalf("text boolean: %v, %v", b, err)
	}
	for _, name := range []string{"object", "array"} {
		if n, err := jsonTestMember(t, root, name).Length(); n != 0 || err != nil {
			t.Fatalf("empty %s: %d, %v", name, n, err)
		}
	}
	if err := json.Unmarshal([]byte(`{}`), &d); !errors.Is(err, ErrConfiguration) {
		t.Fatalf("decoded document handle: %v", err)
	}
	if err := json.Unmarshal([]byte(`{}`), &root); !errors.Is(err, ErrConfiguration) {
		t.Fatalf("decoded value handle: %v", err)
	}
}

func TestRegionJSONParseObservesOriginalContext(t *testing.T) {
	refusals := 0
	for step := 1; step <= 100; step++ {
		action, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		ctx := &integerReadContext{Context: action}
		r := jsonTestRegion(t, ctx)
		s, err := r.Store([]byte(` {"a":"same","ab":[12345,{"a":false}],"last":null} `))
		if err != nil {
			cancel()
			t.Fatal(err)
		}
		ctx.mu.Lock()
		ctx.failAt = ctx.calls + step
		ctx.mu.Unlock()
		d, err := r.ParseJSON(s, JSONLimits{16, 128})
		if err != nil && (!errors.Is(err, context.Canceled) || d.owner != nil) {
			t.Fatalf("context observation %d: %v", step, err)
		}
		if errors.Is(err, context.Canceled) {
			refusals++
		}
		if step == 1 && !errors.Is(err, context.Canceled) {
			t.Fatal("initial original-context refusal was ignored")
		}
		cancel()
	}
	if refusals == 0 {
		t.Fatal("no original-context refusal was exercised")
	}
}

func TestRegionJSONValuesRefuseMalformedHandles(t *testing.T) {
	for _, mutate := range []func(*JSONValue){
		func(v *JSONValue) { v.document.owner = nil },
		func(v *JSONValue) { v.document.owner = &Region{self: v.document.owner} },
		func(v *JSONValue) { v.document.count = 0 },
		func(v *JSONValue) { v.document.nodes.length-- },
		func(v *JSONValue) { v.document.input.length = math.MaxInt },
		func(v *JSONValue) { v.index = -1 },
		func(v *JSONValue) { v.index = v.document.count },
	} {
		_, d := jsonTestDocument(t, `{"a":0}`)
		v := d.Root()
		mutate(&v)
		if _, err := v.Kind(); err == nil {
			t.Fatal("malformed kind handle accepted")
		}
		if _, err := v.Raw(); err == nil {
			t.Fatal("malformed raw handle accepted")
		}
		if _, _, err := v.Member("a"); err == nil {
			t.Fatal("malformed member handle accepted")
		}
		if _, err := v.Index(0); err == nil {
			t.Fatal("malformed array handle accepted")
		}
		if _, err := v.StringEqual("x"); err == nil {
			t.Fatal("malformed string handle accepted")
		}
		if _, err := v.Length(); err == nil {
			t.Fatal("malformed length handle accepted")
		}
	}
	for field := 0; field < 5; field++ {
		r, d := jsonTestDocument(t, `{"a":0}`)
		// Bounded private mock corruption of a record, without a native pointer,
		// operating change or public injection path.
		binary.LittleEndian.PutUint64(r.data[d.nodes.offset+field*8:], math.MaxUint64)
		if _, err := d.Root().Raw(); err == nil {
			t.Fatalf("malformed record field %d accepted", field)
		}
	}
}

func TestRegionJSONValueReadsDiscardResultsOnContextRefusal(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		read func(*testing.T, JSONValue) error
	}{
		{`1`, func(t *testing.T, v JSONValue) error {
			n, err := v.JSONUint64()
			if err != nil && n != 0 {
				t.Fatal("unsigned value escaped refusal")
			}
			return err
		}},
		{`-1`, func(t *testing.T, v JSONValue) error {
			n, err := v.JSONInt64()
			if err != nil && n != 0 {
				t.Fatal("signed value escaped refusal")
			}
			return err
		}},
		{`true`, func(t *testing.T, v JSONValue) error {
			b, err := v.Boolean()
			if err != nil && b {
				t.Fatal("boolean escaped refusal")
			}
			return err
		}},
		{`"label"`, func(t *testing.T, v JSONValue) error {
			b, err := v.StringEqual("label")
			if err != nil && b {
				t.Fatal("string equality escaped refusal")
			}
			return err
		}},
		{`{"a":1,"ab":2}`, func(t *testing.T, v JSONValue) error {
			x, found, err := v.Member("ab")
			if err != nil && (found || x.document.owner != nil) {
				t.Fatal("member handle escaped refusal")
			}
			return err
		}},
		{`[1,2]`, func(t *testing.T, v JSONValue) error {
			x, err := v.Index(1)
			if err != nil && x.document.owner != nil {
				t.Fatal("array handle escaped refusal")
			}
			return err
		}},
		{`[1,2]`, func(t *testing.T, v JSONValue) error {
			n, err := v.Length()
			if err != nil && n != 0 {
				t.Fatal("container length escaped refusal")
			}
			return err
		}},
		{`null`, func(t *testing.T, v JSONValue) error {
			n, err := v.Kind()
			if err != nil && n != 0 {
				t.Fatal("value kind escaped refusal")
			}
			return err
		}},
		{`null`, func(t *testing.T, v JSONValue) error {
			s, err := v.Raw()
			if err != nil && s.owner != nil {
				t.Fatal("token span escaped refusal")
			}
			return err
		}},
	} {
		for step := 1; step <= 12; step++ {
			action, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			ctx := &integerReadContext{Context: action}
			r := jsonTestRegion(t, ctx)
			s, err := r.Store([]byte(tc.raw))
			if err != nil {
				cancel()
				t.Fatal(err)
			}
			d, err := r.ParseJSON(s, JSONLimits{16, 128})
			if err != nil {
				cancel()
				t.Fatal(err)
			}
			ctx.mu.Lock()
			ctx.failAt = ctx.calls + step
			ctx.mu.Unlock()
			err = tc.read(t, d.Root())
			if err != nil && !errors.Is(err, context.Canceled) {
				t.Fatalf("value context refusal: %v", err)
			}
			if step == 1 && !errors.Is(err, context.Canceled) {
				t.Fatal("original context was ignored")
			}
			cancel()
		}
	}
}

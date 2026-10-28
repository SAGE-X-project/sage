package rfc8785

import (
	"bytes"
	"encoding/json"
	"math"
	"testing"
)

func TestCanonicalPrimitiveBytesRemainStable(t *testing.T) {
	for input, want := range map[string]string{
		`false`: `false`, `null`: `null`, `true`: `true`,
		`"\b\t\n\f\r\"\\\u0000\u001f/é"`: `"\b\t\n\f\r\"\\\u0000\u001f/é"`,
		`[]`:                             `[]`, `{}`: `{}`,
		`-12.50`: `-12.5`, `1e20`: `100000000000000000000`,
		`1e21`: `1e+21`, `1.25e21`: `1.25e+21`,
		`0.000001`: `0.000001`, `0.0000001`: `1e-7`,
		`1.25e-7`: `1.25e-7`, `4.5`: `4.5`, `1e3`: `1000`,
	} {
		got, err := Canonicalize([]byte(input))
		if err != nil || string(got) != want {
			t.Fatalf("primitive %s: %q %v; want %s", input, got, err, want)
		}
	}
	for _, input := range []string{`1e400`, `[1e400]`, `{"x":1e400}`} {
		if _, err := Canonicalize([]byte(input)); err == nil {
			t.Fatalf("nonfinite number accepted: %s", input)
		}
	}
	if _, err := Marshal(math.Inf(1)); err == nil {
		t.Fatal("nonfinite Marshal input accepted")
	}
}

func TestCanonicalPrimitiveRefusalsDoNotProduceOutput(t *testing.T) {
	for _, value := range []any{uint64(1), json.Number("not-a-number"), json.Number("1e400")} {
		var out bytes.Buffer
		if err := writeValue(&out, value); err == nil {
			t.Fatalf("unsupported primitive accepted: %T", value)
		}
	}
	var out bytes.Buffer
	if err := writeString(&out, string([]byte{0xff})); err == nil || out.Len() != 0 {
		t.Fatal("invalid UTF-8 produced a string")
	}
	for _, value := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		out.Reset()
		if err := writeNumber(&out, value); err == nil || out.Len() != 0 {
			t.Fatal("nonfinite number produced bytes")
		}
	}
	out.Reset()
	if err := writeValue(&out, float64(1.5)); err != nil || out.String() != "1.5" {
		t.Fatal("finite native number changed", err)
	}
}

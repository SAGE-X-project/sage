package registry011

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestProviderHistoryPreparationDecodePreservesIntegerAndPolicyBytes(t *testing.T) {
	// This tests the DATA decoder, not a qualified collection configuration.
	in := historyPreparationInput{
		Format: historyPreparationInputFormat,
		Policy: `{"html":"<>&","original":"policy","value":"18446744073709551615"}`,
		Clock:  historyPreparationClock{Maximum: math.MaxUint64, BaseUncertaintyMS: math.MaxInt64},
	}
	in.Bounds.SetupWork[10] = math.MaxUint64
	in.Collection.Codes[12].Name = "last-required-code"
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	out, err := decodeHistoryPreparationInput(context.Background(), raw)
	if err != nil || out.Clock != in.Clock || out.Bounds.SetupWork != in.Bounds.SetupWork ||
		out.Collection.Codes != in.Collection.Codes || out.Policy != in.Policy {
		t.Fatalf("lossless DATA decode: err=%v", err)
	}
}

func TestProviderHistoryPreparationDecodeRejectsAmbiguousInput(t *testing.T) {
	raw, err := json.Marshal(historyPreparationInput{Format: historyPreparationInputFormat, Policy: `{}`})
	if err != nil {
		t.Fatal(err)
	}
	base := string(raw)
	replace := func(old, next string) string {
		if !strings.Contains(base, old) {
			t.Fatalf("test replacement missing: %q", old)
		}
		return strings.Replace(base, old, next, 1)
	}
	cases := map[string]string{
		"unknown":        replace(`"format":`, `"unknown":1,"format":`),
		"duplicate":      replace(`"format":`, `"format":"earlier","format":`),
		"escaped-key":    replace(`"format":`, `"\u0066ormat":"earlier","format":`),
		"wrong-case":     replace(`"format":`, `"Format":`),
		"missing":        replace(`"maximum_policy_bytes":0,`, ""),
		"null-required":  replace(`"maximum_policy_bytes":0`, `"maximum_policy_bytes":null`),
		"float-integer":  replace(`"maximum_policy_bytes":0`, `"maximum_policy_bytes":0.0`),
		"exponent":       replace(`"maximum_policy_bytes":0`, `"maximum_policy_bytes":1e0`),
		"negative-zero":  replace(`"maximum_policy_bytes":0`, `"maximum_policy_bytes":-0`),
		"int-overflow":   replace(`"maximum_policy_bytes":0`, `"maximum_policy_bytes":9223372036854775808`),
		"uint-overflow":  replace(`"maximum":0`, `"maximum":18446744073709551616`),
		"negative-uint":  replace(`"maximum":0`, `"maximum":-1`),
		"short-work":     replace(`"setup_work":[0,0,0,0,0,0,0,0,0,0,0]`, `"setup_work":[0]`),
		"long-work":      replace(`"setup_work":[0,0,0,0,0,0,0,0,0,0,0]`, `"setup_work":[0,0,0,0,0,0,0,0,0,0,0,0]`),
		"null-array":     replace(`"setup_work":[0,0,0,0,0,0,0,0,0,0,0]`, `"setup_work":null`),
		"short-codes":    replace(`"codes":[`, `"codes":[],"unused":[`),
		"nested-missing": replace(`"custody_directory":"",`, ""),
		"policy-dup":     replace(`"policy":"{}"`, `"policy":"{\"x\":1,\"x\":2}"`),
		"policy-object":  replace(`"policy":"{}"`, `"policy":{}`),
		"policy-null":    replace(`"policy":"{}"`, `"policy":null`),
		"trailing":       base + ` {}`,
		"truncated":      base[:len(base)-1],
		"root-array":     `[]`,
		"root-null":      `null`,
		"invalid-utf8":   string([]byte{0xff}),
	}
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			if out, err := decodeHistoryPreparationInput(context.Background(), []byte(data)); err == nil || out.Format != "" {
				t.Fatalf("malformed input yielded DATA: err=%v", err)
			}
		})
	}
	deep := strings.Repeat(`[`, 65) + `0` + strings.Repeat(`]`, 65)
	encodedDepth, err := json.Marshal(deep)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeHistoryPreparationInput(context.Background(), []byte(replace(`"policy":"{}"`, `"policy":`+string(encodedDepth)))); err == nil {
		t.Fatal("excessive policy depth accepted")
	}
}

func TestProviderHistoryPreparationTokenSchemaChecks(t *testing.T) {
	type sample struct {
		Text     string          `json:"text"`
		Flag     bool            `json:"flag"`
		Signed   int8            `json:"signed"`
		Unsigned uint8           `json:"unsigned"`
		Pair     [2]uint64       `json:"pair"`
		Raw      json.RawMessage `json:"raw"`
		Ignored  string          `json:"-"`
		private  string
	}
	schema := reflect.TypeFor[sample]()
	valid := `{"text":"x","flag":true,"signed":-128,"unsigned":255,"pair":[0,18446744073709551615],"raw":{"values":[null,false,1.5,"x"]}}`
	if err := validateHistoryPreparationJSON(context.Background(), []byte(valid), schema); err != nil {
		t.Fatal(err)
	}
	for _, data := range []string{
		strings.Replace(valid, `"text":"x"`, `"text":false`, 1),
		strings.Replace(valid, `"flag":true`, `"flag":"true"`, 1),
		strings.Replace(valid, `"signed":-128`, `"signed":-129`, 1),
		strings.Replace(valid, `"unsigned":255`, `"unsigned":256`, 1),
		strings.Replace(valid, `"pair":[0,18446744073709551615]`, `"pair":{}`, 1),
		strings.Replace(valid, `"raw":`, `"raw":{"duplicate":1,"duplicate":2},"unknown":`, 1),
	} {
		if err := validateHistoryPreparationJSON(context.Background(), []byte(data), schema); err == nil {
			t.Fatal("invalid typed DATA accepted")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := validateHistoryPreparationJSON(ctx, []byte(valid), schema); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation lost: %v", err)
	}
	for _, c := range []struct {
		ctx context.Context
		raw []byte
		typ reflect.Type
	}{
		{nil, []byte(valid), schema},
		{context.Background(), nil, schema},
		{context.Background(), []byte(valid), nil},
		{context.Background(), []byte(`{}`), reflect.TypeFor[int]()},
		{context.Background(), []byte(`[]`), reflect.TypeFor[int]()},
		{context.Background(), []byte(`0`), reflect.TypeFor[any]()},
	} {
		if err := validateHistoryPreparationJSON(c.ctx, c.raw, c.typ); err == nil {
			t.Fatal("invalid schema/context accepted")
		}
	}
}

package registry011

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/sage-x-project/sage/pkg/agent/internal/rfc8785"
)

func TestProviderHistoryObservedGenerationMatchesAllExistingCommandBytes(t *testing.T) {
	contract, _, err := replayABIs()
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, step := range replayTestSteps(t) {
		command, accepted := replayParseCommand(replayTestRegistry, replayTestAgent, step.History.Command, contract)
		want, legacy := replayDecodeCommand(replayTestRegistry, command.actor, command.calldata, contract)
		if !accepted || !legacy || !reflect.DeepEqual(command, want) {
			t.Fatal("legacy generator/consumer precursor")
		}
		m := providerHistoryExecutionUnitMeter()
		got, ok := replayDecodeCommandObserved(replayTestRegistry, command.actor, command.calldata, contract, m)
		if !ok || !reflect.DeepEqual(got, want) || m.failed {
			t.Fatal("same-meter generator bytes or consumer changed", command.op, m.actual)
		}
		for i, count := range m.actual {
			if i != 2 && i != 10 && count != 0 {
				t.Fatal("DATA generation invented network/proof/authority", m.actual)
			}
		}
		seen[command.op] = true
	}
	if len(seen) != 8 {
		t.Fatal("all eight actual generator methods required")
	}
	zero := uint64(0)
	key := replayTestKey(t, "b", true)
	key.Expires = &zero
	c := replayTestCommand(t, "add-key", replayTestWallet, 1, map[string]any{"key": key})
	got, ok := replayDecodeCommandObserved(replayTestRegistry, c.actor, c.calldata, contract, providerHistoryExecutionUnitMeter())
	if !ok || !reflect.DeepEqual(got, c) || got.keys[0].Expires == nil || *got.keys[0].Expires != 0 {
		t.Fatal("generated expiry zero changed to absence")
	}
}

func TestProviderHistoryGeneratedNativeWriterDerivesItsOwnFiniteEnvelope(t *testing.T) {
	value := map[string]any{"a": []any{json.Number("7"), "quoted\"\\\n", true, nil}}
	want, err := rfc8785.Marshal(value)
	if err != nil {
		t.Fatal("existing native/fallback canonical precursor", err)
	}
	got, ok := replayGeneratedMarshalObserved(value, providerHistoryExecutionUnitMeter())
	if !ok || !bytes.Equal(got, want) {
		t.Fatal("actual generated JCS bytes", string(got), string(want))
	}
	// One entered node1 + compare('a')2 + owned copy2 = prefix5.
	// Actual payload1/node1 gives encoder ceiling76, required5+76^2+1=5782.
	m := providerHistoryContentUnitMeter(candidateWorkVector{10: 5782})
	if got, ok := replayGeneratedMarshalObserved("a", m); !ok || string(got) != `"a"` || m.actual != (candidateWorkVector{10: 15}) {
		t.Fatal("generated envelope recorded as actual", got, ok, m.actual)
	}
	for _, upper := range []candidateWorkVector{{}, {10: 5781}} {
		short := providerHistoryContentUnitMeter(upper)
		if got, ok := replayGeneratedMarshalObserved("a", short); ok || got != nil || !short.failed {
			t.Fatal("short generator envelope")
		}
		before := short.actual
		if got, ok := replayGeneratedMarshalObserved("a", short); ok || got != nil || short.actual != before {
			t.Fatal("failed original generator resumed")
		}
	}
	for _, value := range []any{providerHistoryTupleCallback{}, make(chan int), strings.Repeat("a", 262145)} {
		if got, ok := replayGeneratedMarshalObserved(value, providerHistoryExecutionUnitMeter()); ok || got != nil {
			t.Fatal("foreign graph/callback or payload bound")
		}
	}
}

func TestProviderHistoryGeneratedWriterKeepsCanonicalBoundaryNumbersAndEscapes(t *testing.T) {
	for _, tc := range []struct {
		value any
		want  string
	}{
		{json.Number("1e20"), "100000000000000000000"},
		{json.Number("1e-6"), "0.000001"},
		{json.Number("1e-7"), "1e-7"},
		{json.Number("-0"), "0"},
		{json.Number("9007199254740991"), "9007199254740991"},
		{"\x00", `"\u0000"`},
		{map[string]any{"": false}, `{"":false}`},
		{[]any{false, nil, true}, `[false,null,true]`},
	} {
		got, ok := replayGeneratedMarshalObserved(tc.value, providerHistoryExecutionUnitMeter())
		if !ok || string(got) != tc.want {
			t.Fatal("finite tree envelope/canonical boundary", string(got), tc.want)
		}
	}
	for _, value := range []json.Number{"NaN", "Inf", "1e999", "invalid"} {
		if got, ok := replayGeneratedMarshalObserved(value, providerHistoryExecutionUnitMeter()); ok || got != nil {
			t.Fatal("nonfinite/invalid generator number")
		}
	}
}

func TestProviderHistoryGeneratedCommandPreservesFullVersionAndSafeExpiry(t *testing.T) {
	contract, _, err := replayABIs()
	if err != nil {
		t.Fatal(err)
	}
	maxVersion := ^uint64(0)
	c := replayTestCommand(t, "activate", replayTestWallet, maxVersion, map[string]any{})
	got, ok := replayDecodeCommandObserved(replayTestRegistry, c.actor, c.calldata, contract, providerHistoryExecutionUnitMeter())
	if !ok || got.expected != maxVersion || !bytes.Equal(got.raw, c.raw) || !bytes.Equal(got.calldata, c.calldata) {
		t.Fatal("uint64 version must not pass through float64")
	}
	expires := uint64(1<<53 - 1)
	key := replayTestKey(t, "b", true)
	key.Expires = &expires
	c = replayTestCommand(t, "add-key", replayTestWallet, 1, map[string]any{"key": key})
	got, ok = replayDecodeCommandObserved(replayTestRegistry, c.actor, c.calldata, contract, providerHistoryExecutionUnitMeter())
	if !ok || len(got.keys) != 1 || got.keys[0].Expires == nil || *got.keys[0].Expires != expires || !bytes.Equal(got.raw, c.raw) {
		t.Fatal("largest safe expiry must remain exact")
	}
}

func TestProviderHistoryGeneratedScalarsAndKeyExpiryPreserveExactShape(t *testing.T) {
	raw := bytes.Repeat([]byte{3}, 32)
	m := providerHistoryContentUnitMeter(candidateWorkVector{10: 1850})
	if got, ok := replayGeneratedBase64Observed(raw, m); !ok || got != base64.RawURLEncoding.EncodeToString(raw) || m.actual[10] != 1850 || m.actual[2] != 0 {
		t.Fatal("actual generated base64 length")
	}
	if got, ok := replayGeneratedBase64Observed(raw, providerHistoryContentUnitMeter(candidateWorkVector{10: 1849})); ok || got != "" {
		t.Fatal("base64 encoder one-short")
	}
	m = providerHistoryContentUnitMeter(candidateWorkVector{10: 401})
	if got, ok := replayGeneratedUintObserved(1, m); !ok || got != "1" || m.actual[10] != 2 {
		t.Fatal("decimal bound is not actual output", m.actual)
	}
	base := replayTestCreate(t)
	keys := replayABIKeys(base.keys)
	for _, expiry := range []bool{false, true} {
		keys[0].HasExpiry, keys[0].Expires = expiry, 0
		value, ok := replayGeneratedKeyObserved(keys[0], providerHistoryExecutionUnitMeter())
		if !ok {
			t.Fatal("normal key generation")
		}
		v, present := value["expires"]
		if present != expiry || present && v != json.Number("0") {
			t.Fatal("expiry presence changed", value)
		}
	}
	for _, key := range []replayABIKey{{Expires: 1}, {HasExpiry: true, Expires: 1 << 53}} {
		if got, ok := replayGeneratedKeyObserved(key, providerHistoryExecutionUnitMeter()); ok || got != nil {
			t.Fatal("inconsistent or unsafe expiry")
		}
	}
	if got, ok := replayGeneratedServicesObserved(nil, providerHistoryExecutionUnitMeter()); !ok || got == nil || len(got) != 0 {
		t.Fatal("generated services preserve old empty list normalization")
	}
	if _, ok := replayGeneratedBase64Observed(make([]byte, 66), providerHistoryExecutionUnitMeter()); ok {
		t.Fatal("generated binary cap")
	}
}

func TestProviderHistoryGeneratedObjectsArraysAndMissingInputsRefusePartialOutput(t *testing.T) {
	for _, names := range [][]string{{"a", "a"}, {""}, {"é"}} {
		if got, ok := replayGeneratedObjectObserved(names, make([]any, len(names)), providerHistoryExecutionUnitMeter()); ok || got != nil {
			t.Fatal("invalid generated field list")
		}
	}
	if got, ok := replayGeneratedObjectObserved([]string{"a"}, nil, providerHistoryExecutionUnitMeter()); ok || got != nil {
		t.Fatal("generated field/value mismatch")
	}
	for _, count := range []int{-1, 129} {
		if got, ok := replayGeneratedArrayObserved(count, providerHistoryExecutionUnitMeter()); ok || got != nil {
			t.Fatal("generated count cap")
		}
	}
	if replayGeneratedArrayPutObserved(nil, 0, nil, providerHistoryExecutionUnitMeter()) || replayGeneratedArrayPutObserved(make([]any, 1), -1, nil, providerHistoryExecutionUnitMeter()) {
		t.Fatal("generated array index")
	}
	if got, ok := replayGeneratedObjectObserved(nil, nil, nil); ok || got != nil {
		t.Fatal("missing generated object observer")
	}
	if got, ok := replayGeneratedArrayObserved(1, nil); ok || got != nil {
		t.Fatal("missing array observer")
	}
	if _, ok := replayGeneratedUintObserved(1, nil); ok {
		t.Fatal("missing number observer")
	}
	if got, ok := replayGeneratedServicesObserved(make([]replayABIService, 17), providerHistoryExecutionUnitMeter()); ok || got != nil {
		t.Fatal("generated service count cap")
	}
}

func TestProviderHistoryDecodedGenerationRejectsChangedCallerTypesAndExpiry(t *testing.T) {
	contract, _, _ := replayABIs()
	c := replayTestCreate(t)
	values := providerHistoryTupleUnitValues(t)
	if _, ok := replayDecodedCommandObserved(replayTestRegistry, c.actor, "revealClaim", values, c.calldata, contract, providerHistoryExecutionUnitMeter()); !ok {
		t.Fatal("otherwise valid decoded generator precursor")
	}
	for _, tc := range []struct {
		method, actor string
		values        []any
		raw           []byte
	}{
		{"unknown", c.actor, values, c.calldata}, {"revealClaim", "invalid", values, c.calldata},
		{"revealClaim", c.actor, values[:4], c.calldata}, {"revealClaim", c.actor, values, nil},
	} {
		if got, ok := replayDecodedCommandObserved(replayTestRegistry, tc.actor, tc.method, tc.values, tc.raw, contract, providerHistoryExecutionUnitMeter()); ok || !reflect.DeepEqual(got, replayCommand{}) {
			t.Fatal("changed decoded generator input")
		}
	}
	if _, ok := replayDecodedCommandObserved(replayTestRegistry, c.actor, "revealClaim", values, c.calldata, contract, nil); ok {
		t.Fatal("missing generator original observer")
	}
	for _, field := range []int{0, 1, 2, 3, 4} {
		bad := append([]any{}, values...)
		bad[field] = 1
		if got, ok := replayDecodedCommandObserved(replayTestRegistry, c.actor, "revealClaim", bad, c.calldata, contract, providerHistoryExecutionUnitMeter()); ok || !reflect.DeepEqual(got, replayCommand{}) {
			t.Fatal("wrong actual decoded type")
		}
	}
	// A well-framed private UNIT tuple with a contradictory presence flag.
	bad := append([]any{}, values...)
	v := reflect.MakeSlice(reflect.TypeOf(values[2]), 1, 1)
	v.Index(0).Set(reflect.ValueOf(values[2]).Index(0))
	v.Index(0).FieldByName("HasExpiry").SetBool(false)
	v.Index(0).FieldByName("Expires").SetUint(1)
	bad[2] = v.Interface()
	data, err := contract.Methods["revealClaim"].Inputs.Pack(bad...)
	if err != nil {
		t.Fatal(err)
	}
	method := contract.Methods["revealClaim"]
	original := append(bytes.Clone(method.ID), data...)
	if _, ok := replayDecodeCommand(replayTestRegistry, c.actor, original, contract); ok {
		t.Fatal("legacy must reject contradictory expiry before new path")
	}
	if got, ok := replayDecodeCommandObserved(replayTestRegistry, c.actor, original, contract, providerHistoryExecutionUnitMeter()); ok || !reflect.DeepEqual(got, replayCommand{}) {
		t.Fatal("contradictory generated expiry returned partial command")
	}
}

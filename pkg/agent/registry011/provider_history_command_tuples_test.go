package registry011

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

type providerHistoryTupleCallback struct{}

func (providerHistoryTupleCallback) MarshalJSON() ([]byte, error) {
	panic("closed tuple/generator must never invoke a caller marshaler")
}

func providerHistoryTupleUnitValues(t *testing.T) []any {
	t.Helper()
	contract, _, err := replayABIs()
	if err != nil {
		t.Fatal(err)
	}
	c := replayTestCreate(t)
	values, err := contract.Methods["revealClaim"].Inputs.Unpack(c.calldata[4:])
	if err != nil {
		t.Fatal("existing ABI tuple precursor", err)
	}
	return values
}

func TestProviderHistoryCommandTuplesMatchExistingFixedJSONBridgeAndOwnPayloads(t *testing.T) {
	values := providerHistoryTupleUnitValues(t)
	var want []replayABIKey
	if !replayConvert(values[2], &want) || len(want) != 2 {
		t.Fatal("existing fixed tuple bridge precursor")
	}
	m := providerHistoryExecutionUnitMeter()
	got, ok := replayCommandTupleKeysObserved(values[2], m)
	if !ok || !reflect.DeepEqual(got, want) || m.failed {
		t.Fatal("fixed key tuple conversion", got, want)
	}
	original := reflect.ValueOf(values[2]).Index(0).FieldByName("Key").Bytes()
	before := bytes.Clone(original)
	originalProof := reflect.ValueOf(values[2]).Index(0).FieldByName("Proof").Bytes()
	beforeProof := bytes.Clone(originalProof)
	got[0].Key[0] ^= 1
	got[0].Proof[0] ^= 1
	if !bytes.Equal(original, before) || !bytes.Equal(originalProof, beforeProof) || bytes.Equal(got[0].Proof, want[0].Proof) {
		t.Fatal("tuple byte payload ownership")
	}
	command := replayTestCommand(t, "update-services", replayTestWallet, 1, map[string]any{"services": []replayService{{Name: "api", Type: "A2A", URI: "https://example.invalid/api"}}})
	contract, _, _ := replayABIs()
	serviceValues, err := contract.Methods["updateServices"].Inputs.Unpack(command.calldata[4:])
	if err != nil {
		t.Fatal(err)
	}
	var expected []replayABIService
	if !replayConvert(serviceValues[2], &expected) {
		t.Fatal("existing service bridge precursor")
	}
	services, ok := replayCommandTupleServicesObserved(serviceValues[2], m)
	if !ok || !reflect.DeepEqual(services, expected) || len(services) != 1 {
		t.Fatal("raw uri tag did not map to own URI field")
	}
	entry, ok := replayCommandTupleServiceObserved(reflect.ValueOf(serviceValues[2]).Index(0).Interface(), m)
	if !ok || !reflect.DeepEqual(entry, expected[0]) {
		t.Fatal("single service tuple")
	}
}

func TestProviderHistoryCommandTupleNilEmptyAndExpiryValuesRemainDistinct(t *testing.T) {
	values := providerHistoryTupleUnitValues(t)
	typ := reflect.TypeOf(values[2]).Elem()
	for _, emptyBytes := range []bool{false, true} {
		v := reflect.New(typ).Elem()
		if emptyBytes {
			for _, name := range []string{"Key", "Proof"} {
				field := v.FieldByName(name)
				field.Set(reflect.MakeSlice(field.Type(), 0, 0))
			}
		}
		v.FieldByName("HasExpiry").SetBool(true)
		v.FieldByName("Expires").SetUint(0)
		var expected replayABIKey
		if !replayConvert(v.Interface(), &expected) {
			t.Fatal("existing nil/empty tuple precursor")
		}
		got, ok := replayCommandTupleKeyObserved(v.Interface(), providerHistoryExecutionUnitMeter())
		if !ok || !reflect.DeepEqual(got, expected) || !got.HasExpiry || got.Expires != 0 {
			t.Fatal("nil/empty or explicit zero changed", got, expected)
		}
	}
	for _, value := range []any{reflect.Zero(reflect.TypeOf(values[2])).Interface(), reflect.MakeSlice(reflect.TypeOf(values[2]), 0, 0).Interface()} {
		var expected []replayABIKey
		if !replayConvert(value, &expected) {
			t.Fatal("existing array shape precursor")
		}
		got, ok := replayCommandTupleKeysObserved(value, providerHistoryExecutionUnitMeter())
		if !ok || !reflect.DeepEqual(got, expected) {
			t.Fatal("nil versus empty tuple array", got, expected)
		}
	}
}

func TestProviderHistoryCommandServiceTuplesKeepEmptyShapesAndFixedFieldTags(t *testing.T) {
	values := providerHistoryTupleUnitValues(t)
	typ := reflect.TypeOf(values[3])
	for _, value := range []any{reflect.Zero(typ).Interface(), reflect.MakeSlice(typ, 0, 0).Interface()} {
		var want []replayABIService
		if !replayConvert(value, &want) {
			t.Fatal("existing empty service precursor")
		}
		got, ok := replayCommandTupleServicesObserved(value, providerHistoryExecutionUnitMeter())
		if !ok || !reflect.DeepEqual(got, want) {
			t.Fatal("service null/empty conversion changed", got, want)
		}
	}
	for _, value := range []any{nil, providerHistoryTupleCallback{}, struct{}{}, struct {
		Name string `json:"name"`
		Type string `json:"serviceType"`
		URI  uint64 `json:"uri"`
	}{}} {
		if got, ok := replayCommandTupleServiceObserved(value, providerHistoryExecutionUnitMeter()); ok || !reflect.DeepEqual(got, replayABIService{}) {
			t.Fatal("non-string or unsupported service tuple")
		}
	}
	entry := reflect.New(typ.Elem()).Elem()
	entry.FieldByName("Name").SetString("api")
	entry.FieldByName("ServiceType").SetString("A2A")
	entry.FieldByName("Uri").SetString("https://example.invalid/api")
	if got, ok := replayCommandTupleServiceObserved(entry.Interface(), providerHistoryExecutionUnitMeter()); !ok || got.URI != "https://example.invalid/api" {
		t.Fatal("otherwise valid service tuple precursor", got)
	}
	entry.FieldByName("Uri").SetString(strings.Repeat("a", 2049))
	if got, ok := replayCommandTupleServiceObserved(entry.Interface(), providerHistoryExecutionUnitMeter()); ok || !reflect.DeepEqual(got, replayABIService{}) {
		t.Fatal("oversized service URI")
	}
	if _, ok := replayCommandTupleServicesObserved(reflect.MakeSlice(typ, 1, 1).Interface(), nil); ok {
		t.Fatal("missing service observer")
	}
}

func TestProviderHistoryCommandTuplePrimitiveQuotasAreConcreteAndSticky(t *testing.T) {
	text := reflect.ValueOf("ab")
	m := providerHistoryContentUnitMeter(candidateWorkVector{10: 6})
	if got, ok := replayCommandTupleStringObserved(text, 2, m); !ok || got != "ab" || m.actual[10] != 6 {
		t.Fatal("string compare3/copy3", got, ok, m.actual)
	}
	short := providerHistoryContentUnitMeter(candidateWorkVector{10: 5})
	if got, ok := replayCommandTupleStringObserved(text, 2, short); ok || got != "" || !short.failed || short.actual[10] != 3 {
		t.Fatal("string copy one-short", got, ok, short.actual)
	}
	before := short.actual
	if _, ok := replayCommandTupleStringObserved(text, 2, short); ok || short.actual != before {
		t.Fatal("failed tuple string resumed")
	}
	raw := []byte{1, 2, 3}
	m = providerHistoryContentUnitMeter(candidateWorkVector{10: 4})
	got, ok := replayCommandTupleBytesObserved(reflect.ValueOf(raw), m)
	if !ok || !bytes.Equal(got, raw) || m.actual[10] != 4 {
		t.Fatal("byte clone actual")
	}
	got[0] = 9
	if raw[0] != 1 {
		t.Fatal("byte clone aliases input")
	}
	if got, ok := replayCommandTupleBytesObserved(reflect.ValueOf(raw), providerHistoryContentUnitMeter(candidateWorkVector{10: 3})); ok || got != nil {
		t.Fatal("unfunded byte clone")
	}
}

func TestProviderHistoryCommandTuplesRejectUnsupportedTypesAndNeverCallMarshalers(t *testing.T) {
	for _, value := range []any{nil, 1, struct{}{}, providerHistoryTupleCallback{}, struct {
		Name string `json:"other"`
	}{}} {
		if got, ok := replayCommandTupleKeyObserved(value, providerHistoryExecutionUnitMeter()); ok || !reflect.DeepEqual(got, replayABIKey{}) {
			t.Fatal("unsupported tuple type")
		}
	}
	for _, value := range []any{nil, 1, []string{"a"}, make([]int, 129)} {
		if got, ok := replayCommandTupleKeysObserved(value, providerHistoryExecutionUnitMeter()); ok || got != nil {
			t.Fatal("unsupported key tuple array")
		}
	}
	for _, value := range []any{nil, 1, []string{"a"}, make([]int, 17)} {
		if got, ok := replayCommandTupleServicesObserved(value, providerHistoryExecutionUnitMeter()); ok || got != nil {
			t.Fatal("unsupported service tuple array")
		}
	}
	for _, value := range []reflect.Value{reflect.Value{}, reflect.ValueOf(1), reflect.ValueOf("é")} {
		if _, ok := replayCommandTupleStringObserved(value, 32, providerHistoryExecutionUnitMeter()); ok {
			t.Fatal("unsupported tuple string")
		}
	}
	if _, ok := replayCommandTupleStringObserved(reflect.ValueOf("ab"), 1, providerHistoryExecutionUnitMeter()); ok {
		t.Fatal("tuple string cap")
	}
	for _, value := range []reflect.Value{reflect.Value{}, reflect.ValueOf("a"), reflect.ValueOf([]uint64{1}), reflect.ValueOf(make([]byte, 66))} {
		if _, ok := replayCommandTupleBytesObserved(value, providerHistoryExecutionUnitMeter()); ok {
			t.Fatal("unsupported tuple bytes")
		}
	}
	values := providerHistoryTupleUnitValues(t)
	typ := reflect.TypeOf(values[2]).Elem()
	for _, change := range []string{"tag", "boolean", "number", "bytes", "name-cap"} {
		fields := make([]reflect.StructField, typ.NumField())
		for i := range fields {
			fields[i] = typ.Field(i)
		}
		switch change {
		case "tag":
			fields[0].Tag = `json:"other"`
		case "boolean":
			fields[5].Type = reflect.TypeFor[string]()
		case "number":
			fields[6].Type = reflect.TypeFor[uint32]()
		case "bytes":
			fields[2].Type = reflect.TypeFor[[]uint64]()
		}
		v := reflect.New(reflect.StructOf(fields)).Elem()
		if change == "name-cap" {
			v.FieldByName("Name").SetString(strings.Repeat("a", 33))
		}
		if got, ok := replayCommandTupleKeyObserved(v.Interface(), providerHistoryExecutionUnitMeter()); ok || !reflect.DeepEqual(got, replayABIKey{}) {
			t.Fatal("changed fixed tuple shape", change)
		}
	}
	if _, ok := replayCommandTupleKeysObserved(values[2], nil); ok {
		t.Fatal("missing tuple observer")
	}
	m := providerHistoryContentUnitMeter(candidateWorkVector{})
	if got, ok := replayCommandTupleKeysObserved(values[2], m); ok || got != nil || !m.failed {
		t.Fatal("unfunded tuple array")
	}
}

package registry011

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

func providerHistoryABIUnitWord(n uint64) []byte {
	raw := make([]byte, 32)
	binary.BigEndian.PutUint64(raw[24:], n)
	return raw
}

func TestProviderHistoryABIWordAndStringUseConcreteExactQuotas(t *testing.T) {
	m := providerHistoryContentUnitMeter(candidateWorkVector{2: 8, 10: 114})
	n, ok := replayABIWordObserved(providerHistoryABIUnitWord(148804), m)
	// Zero-prefix comparison48+1 and actual uint64 decode8*8+1.
	if !ok || n != 148804 || m.actual != (candidateWorkVector{2: 8, 10: 114}) {
		t.Fatal("fixed word arithmetic", n, ok, m.actual)
	}
	str, err := abi.NewType("string", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	fields := abi.Arguments{{Name: "name", Type: str}}
	raw, err := fields.Pack("ab")
	if err != nil || len(raw) != 96 {
		t.Fatal("finite canonical string precursor", err)
	}
	// Two words228 + padding60+1 + two actual ASCII comparisons2+2.
	m = providerHistoryContentUnitMeter(candidateWorkVector{2: 16, 10: 293})
	if n, ok := replayABIFieldsObserved(raw, fields, m); !ok || n != 96 || m.actual != (candidateWorkVector{2: 16, 10: 293}) {
		t.Fatal("exact string framing arithmetic", n, ok, m.actual)
	}
	for _, upper := range []candidateWorkVector{{}, {2: 15, 10: 293}, {2: 16, 10: 292}} {
		short := providerHistoryContentUnitMeter(upper)
		if n, ok := replayABIFieldsObserved(raw, fields, short); ok || n != 0 || !short.failed {
			t.Fatal("short framing quota returned a parsed prefix", n, ok, short.actual)
		}
		before := short.actual
		if _, ok := replayABIFieldsObserved(raw, fields, short); ok || short.actual != before {
			t.Fatal("failed original framing observer resumed")
		}
	}
	if n, ok := replayABIWordObserved(providerHistoryABIUnitWord(2), nil); !ok || n != 2 {
		t.Fatal("legacy nil word")
	}
	if n, ok := replayABIFieldsObserved(raw, fields, nil); !ok || n != 96 {
		t.Fatal("legacy nil fields")
	}
}

func TestProviderHistoryABIStaticFieldsPreserveClosedScalarDomains(t *testing.T) {
	for _, tc := range []struct {
		kind     string
		accepted bool
	}{
		{"address", true}, {"uint64", true}, {"bool", true}, {"bytes32", true},
		{"uint256", false}, {"bytes4", false}, {"int64", false},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			typ, err := abi.NewType(tc.kind, "", nil)
			if err != nil {
				t.Fatal(err)
			}
			fields := abi.Arguments{{Name: "x", Type: typ}}
			raw := providerHistoryABIUnitWord(0)
			m := providerHistoryExecutionUnitMeter()
			n, ok := replayABIFieldsObserved(raw, fields, m)
			if ok != tc.accepted || ok && n != 32 {
				t.Fatal("scalar domain", tc.kind, n, ok)
			}
			if tc.accepted && tc.kind != "bytes32" {
				raw[0] = 1
				if _, ok := replayABIFieldsObserved(raw, fields, providerHistoryExecutionUnitMeter()); ok {
					t.Fatal("nonzero high padding")
				}
			}
			if tc.kind == "bool" {
				if _, ok := replayABIFieldsObserved(providerHistoryABIUnitWord(2), fields, providerHistoryExecutionUnitMeter()); ok {
					t.Fatal("bool outside zero/one")
				}
			}
		})
	}
	for _, raw := range [][]byte{nil, bytes.Repeat([]byte{255}, 32), providerHistoryABIUnitWord(148805)} {
		if _, ok := replayABIWordObserved(raw, providerHistoryExecutionUnitMeter()); ok {
			t.Fatal("word bound")
		}
	}
}

func TestProviderHistoryABIFramingRejectsOnlyChangedCanonicalRegions(t *testing.T) {
	str, _ := abi.NewType("string", "", nil)
	fields := abi.Arguments{{Name: "name", Type: str}}
	normal, err := fields.Pack("ab")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		edit func([]byte) []byte
	}{
		{"short-head", func(raw []byte) []byte { return raw[:31] }},
		{"overlapping-tail", func(raw []byte) []byte { raw[31] = 0; return raw }},
		{"misaligned-tail", func(raw []byte) []byte { raw[31] = 33; return raw }},
		{"length-cap", func(raw []byte) []byte { raw[63] = 33; return raw }},
		{"truncated-tail", func(raw []byte) []byte { return raw[:95] }},
		{"nonzero-padding", func(raw []byte) []byte { raw[95] = 1; return raw }},
		{"non-ascii", func(raw []byte) []byte { raw[64] = 128; return raw }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad := tc.edit(bytes.Clone(normal))
			if _, ok := replayABIFields(bad, fields); ok {
				t.Fatal("existing framing must reject the single changed region")
			}
			if n, ok := replayABIFieldsObserved(bad, fields, providerHistoryExecutionUnitMeter()); ok || n != 0 {
				t.Fatal("changed framing accepted")
			}
		})
	}
	if _, ok := replayABIFieldsDepthObserved(normal, fields, 33, providerHistoryExecutionUnitMeter()); ok {
		t.Fatal("unbounded recursive ABI")
	}
	if _, ok := replayABIFieldsDepthObserved(normal, fields, -1, providerHistoryExecutionUnitMeter()); ok {
		t.Fatal("negative depth")
	}
	if _, ok := replayABIFieldsDepthObserved(normal, fields, 0, nil); ok {
		t.Fatal("internal recursive path requires original observer")
	}
	if _, ok := replayABIFieldsObserved(nil, make(abi.Arguments, 129), providerHistoryExecutionUnitMeter()); ok {
		t.Fatal("unbounded argument count")
	}
}

func TestProviderHistoryABIFramingMatchesAllAdoptedMutationMethods(t *testing.T) {
	contract, _, err := replayABIs()
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, step := range replayTestSteps(t) {
		command, ok := replayParseCommand(replayTestRegistry, replayTestAgent, step.History.Command, contract)
		if !ok {
			t.Fatal("existing finite command precursor")
		}
		method, err := contract.MethodById(command.calldata[:4])
		if err != nil || !replayCallFraming(*method, command.calldata) {
			t.Fatal("existing framing precursor", err)
		}
		m := providerHistoryExecutionUnitMeter()
		if !replayCallFramingObserved(*method, command.calldata, m) || m.failed || m.actual[10] == 0 {
			t.Fatal("adopted framing", command.op, m.actual)
		}
		if !replayCallFramingObserved(*method, command.calldata, nil) {
			t.Fatal("legacy nil call")
		}
		seen[command.op] = true
		bad := append(bytes.Clone(command.calldata), make([]byte, 32)...)
		if replayCallFramingObserved(*method, bad, providerHistoryExecutionUnitMeter()) {
			t.Fatal("trailing calldata")
		}
	}
	if len(seen) != 8 {
		t.Fatal("all eight mutation methods required", seen)
	}
	for _, name := range []string{"record", "registryId", "commitClaim", "absent"} {
		if replayCallFramingObserved(contract.Methods[name], make([]byte, 36), providerHistoryExecutionUnitMeter()) || replayABICallMaximum(name) != 0 {
			t.Fatal("non-mutation method framed", name)
		}
	}
}

func TestProviderHistoryABIDynamicArraysAndArgumentStorageAreBounded(t *testing.T) {
	contract, _, _ := replayABIs()
	keys := contract.Methods["revealClaim"].Inputs[2]
	services := contract.Methods["revealClaim"].Inputs[3]
	for _, field := range []abi.Argument{keys, services} {
		m := providerHistoryExecutionUnitMeter()
		n, ok := replayABIDynamicObserved(providerHistoryABIUnitWord(0), field, 0, m)
		if !ok || n != 32 || m.actual[2] != 8 {
			t.Fatal("empty canonical array", n, ok, m.actual)
		}
		maximum := uint64(129)
		if field.Name == "services" {
			maximum = 17
		}
		if _, ok := replayABIDynamicObserved(providerHistoryABIUnitWord(maximum), field, 0, providerHistoryExecutionUnitMeter()); ok {
			t.Fatal("array count cap")
		}
		if _, ok := replayABIDynamicObserved(providerHistoryABIUnitWord(1), field, 0, providerHistoryExecutionUnitMeter()); ok {
			t.Fatal("truncated array head")
		}
	}
	for _, field := range []abi.Argument{
		{Name: "unknown", Type: keys.Type},
		{Type: abi.Type{T: abi.TupleTy, TupleElems: []*abi.Type{nil}, TupleRawNames: []string{"name"}}},
		{Type: abi.Type{T: abi.TupleTy, TupleElems: []*abi.Type{nil}}},
		{Type: abi.Type{T: abi.FixedBytesTy, Size: 32}},
		{Name: "keys", Type: abi.Type{T: abi.SliceTy}},
	} {
		if _, ok := replayABIDynamicObserved(providerHistoryABIUnitWord(0), field, 0, providerHistoryExecutionUnitMeter()); ok {
			t.Fatal("invalid internal pinned type shape")
		}
	}
	for _, count := range []int{-1, 129} {
		if fields, ok := replayABIArgumentStorageObserved(count, providerHistoryExecutionUnitMeter()); ok || fields != nil {
			t.Fatal("argument storage cap")
		}
	}
	m := providerHistoryExecutionUnitMeter()
	fields, ok := replayABIArgumentStorageObserved(3, m)
	if !ok || len(fields) != 3 || cap(fields) != 3 || m.actual[10] != 3*uint64(reflect.TypeFor[abi.Argument]().Size())+1 {
		t.Fatal("actual argument initialization", m.actual)
	}
	if got, ok := replayABIArgumentStorageObserved(1, nil); ok || got != nil || replayABIZeroObserved(make([]byte, 33), providerHistoryExecutionUnitMeter()) {
		t.Fatal("missing observer or oversized zero comparison")
	}
	str, _ := abi.NewType("string", "", nil)
	if _, ok := replayABIDynamicObserved(providerHistoryABIUnitWord(0), abi.Argument{Name: "unknown", Type: str}, 0, providerHistoryExecutionUnitMeter()); ok {
		t.Fatal("unknown variable field")
	}
	for _, depth := range []int{-1, 33} {
		if _, ok := replayABIDynamicObserved(nil, keys, depth, providerHistoryExecutionUnitMeter()); ok {
			t.Fatal("invalid dynamic depth")
		}
	}
	if _, ok := replayABIDynamicObserved(nil, keys, 0, nil); ok || replayABIZeroObserved(nil, nil) {
		t.Fatal("missing private dynamic/zero observer")
	}
}

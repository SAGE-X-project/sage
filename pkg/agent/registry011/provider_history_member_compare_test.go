package registry011

import (
	"reflect"
	"strings"
	"testing"
)

func TestProviderHistoryMemberTypedComparisonChargesEqualBytesWithNewBacking(t *testing.T) {
	type child struct {
		text  string
		count uint64
	}
	type subject struct {
		text     string
		children [2]child
		pointer  *int
	}
	n := 7
	original := subject{"original", [2]child{{"first", 1}, {"second", 2}}, &n}
	m := providerReplayUnitMeter()
	weight, ok := providerHistoryMemberComparisonWeightObserved(original, m)
	want := uint64(reflect.TypeFor[subject]().Size() + uintptr(len(original.text)+len(original.children[0].text)+len(original.children[1].text)))
	if !ok || weight != want || m.actual[10] == 0 {
		t.Fatal("metadata walk or conservative typed comparison weight", weight, want)
	}
	actual := original
	actual.text, actual.children[0].text = strings.Clone(original.text), strings.Clone(original.children[0].text)
	before := m.actual[10]
	if !providerHistoryMemberEqualObserved(actual, original, weight, m) || m.actual[10] != before+2*weight+1 {
		t.Fatal("equal byte content bypassed the comparison observer")
	}
	actual.children[1].text = "SECOND"
	before = m.actual[10]
	if providerHistoryMemberEqualObserved(actual, original, weight, m) || m.actual[10] != before+2*weight+1 {
		t.Fatal("unequal same-length bytes bypassed observed comparison")
	}
	if _, ok := providerHistoryMemberComparisonWeightObserved(original, nil); ok || providerHistoryMemberEqualObserved(original, original, weight, nil) {
		t.Fatal("missing observer")
	}
	short := providerHistoryContentUnitMeter(candidateWorkVector{10: 1})
	if _, ok := providerHistoryMemberComparisonWeightObserved(original, short); ok || !short.failed || short.actual[10] != 1 {
		t.Fatal("metadata walk exhaustion refunded an entered prefix")
	}
	shortArray := providerHistoryContentUnitMeter(candidateWorkVector{10: 1})
	if _, ok := providerHistoryMemberComparisonWeightObserved([2]string{"one", "two"}, shortArray); ok || !shortArray.failed || shortArray.actual[10] != 1 {
		t.Fatal("array metadata walk exhaustion refunded an entered prefix")
	}
	overflow := providerReplayUnitMeter()
	if providerHistoryMemberEqualObserved(original, original, ^uint64(0), overflow) || !overflow.failed {
		t.Fatal("comparison byte weight overflow did not refuse")
	}
	remaining := uint64(1)
	if _, ok := providerHistoryMemberStringWeightObserved(reflect.ValueOf("x"), m, 33, &remaining); ok || remaining != 1 {
		t.Fatal("depth bound must precede traversal")
	}
	unsupported := providerReplayUnitMeter()
	if _, ok := providerHistoryMemberComparisonWeightObserved(1.5, unsupported); ok {
		t.Fatal("unsupported reflected kind must refuse")
	}
}

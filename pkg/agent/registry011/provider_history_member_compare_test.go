package registry011

import (
	"reflect"
	"strings"
	"testing"
)

func TestProviderHistoryMemberTypedWalkKeepsOriginalGroupCounterAndLimits(t *testing.T) {
	o := historyNativeInputFixture(t, false, 0)
	m := o.meter
	room := candidateWorkVector{10: 65536}
	m.mu.Lock()
	fits := m.hasOwner() && o.fitsWork(m, room)
	for coordinate, units := range room {
		fits = fits && candidateCapacity(0, m.actual[coordinate], units, m.upper[coordinate])
	}
	m.mu.Unlock()
	if !fits {
		t.Fatal("original active group has no complete typed-walk headroom")
	}
	type subject struct {
		label  string
		values [2]uint64
	}
	// Five visited nodes: struct, string, array and its two numeric leaves.
	before, upper, window, reads, held := m.actual, m.upper, o.windowUpper, o.store.clock.read, o.store.stored
	weight, ok := providerHistoryMemberComparisonWeightObserved(subject{"abc", [2]uint64{1, 2}}, m)
	want := before
	want[10] += 5
	if !ok || weight != uint64(reflect.TypeFor[subject]().Size())+3 || m.actual != want || m.failed || m.closed ||
		m.upper != upper || o.windowUpper != window || o.store.clock.read != reads || o.store.stored != held || o.store.activeMeter.Load() != m {
		t.Fatal("bounded typed primitive changed its original owner, counter or limits")
	}
	// Unsupported finite DATA still spends exactly its entered node, without
	// consuming the prospective maximum or manufacturing a successful value.
	before = m.actual
	weight, ok = providerHistoryMemberComparisonWeightObserved(1.5, m)
	want = before
	want[10]++
	if ok || weight != 0 || m.actual != want || m.failed || m.closed || m.upper != upper || o.store.clock.read != reads {
		t.Fatal("failed typed primitive changed the original partial-prefix accounting")
	}
}

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
	if _, ok := providerHistoryMemberStringWeightWithVisit(reflect.ValueOf("x"), 0, &remaining, nil); ok || remaining != 1 {
		t.Fatal("missing internal debit must refuse before traversal")
	}
	unsupported := providerReplayUnitMeter()
	if _, ok := providerHistoryMemberComparisonWeightObserved(1.5, unsupported); ok {
		t.Fatal("unsupported reflected kind must refuse")
	}
}

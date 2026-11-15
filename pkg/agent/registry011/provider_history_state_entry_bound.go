package registry011

import "reflect"

// One original history-entry operation after the observed command parser and
// applyObserved caller have established op<=18, command<=262144, record<=65536
// and grants<=262144. The private command type alone does not prove that route.
// This scalar DATA quote authenticates no caller/state/Source; its eventual
// factory must bind the actual inspected parse/decode/apply path. Version and
// removed-grant text use the original global walker bound, not clone ceilings.
func providerHistoryParsedStateEntryUpper(removed uint64) (candidateWorkVector, bool) {
	var empty candidateWorkVector
	if removed > 128 {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	object := uint64(reflect.TypeFor[map[string]any]().Size())
	value := uint64(reflect.TypeFor[any]().Size())
	slot := a.sum(uint64(reflect.TypeFor[string]().Size()), value)
	removals := a.sum(1, a.product(removed, a.sum(a.product(2, value), object, a.product(2, slot), 19)))
	// Three hash/base64 operations, now encode20, both optional operation
	// comparisons and target-field checks, then the ten-field history map.
	before := a.sum(595377, removals, 401, 71, 15, object, a.product(10, slot), 125)
	nodes := a.sum(25, a.product(5, removed))
	strings := a.sum(21, a.product(4, removed))
	maps := a.sum(2, removed)
	fields := a.sum(12, a.product(2, removed))
	walk := a.sum(nodes, 2*262144, a.product(2, strings), 257,
		a.product(maps, a.sum(object, 1)), a.product(fields, a.sum(slot, 1)),
		a.product(removed, a.sum(a.product(2, value), 1)), 1)
	// The admitted Number has<=16 digits; every other generated value is
	// a string/null/array/map. Preserve both decoded and logical codec work.
	maximum := a.sum(a.product(6, a.sum(nodes, 262144)), 64)
	units := a.sum(before, walk, a.product(maximum, maximum), 1)
	if a.failed {
		return empty, false
	}
	var work candidateWorkVector
	work[2], work[10] = 16, units
	return work, true
}

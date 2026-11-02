package registry011

import "reflect"

// A finite typed metadata-length walk derives a conservative comparison debit.
// Pointer fields are compared by identity; their selected children must be
// bound separately. It never follows pointers, reads payload bytes or calls a
// method on the value. Every visited node spends the original observer before
// reflection. This is logical work, not a native CPU or resident-RAM witness.
func providerHistoryMemberComparisonWeightObserved[T comparable](value T, m *providerWorkMeter) (uint64, bool) {
	if m == nil {
		return 0, false
	}
	remaining := uint64(65536)
	visit := func() bool { return m.event(providerCompareEvent, 0) }
	// The caller already serializes the original group with store.mu. This
	// primitive traverses only a copied comparable value: no pointer contents,
	// methods, I/O, clock samples or caller hooks can change its owner/window.
	// If ALL possible visits fit every original limit, keep meter.mu for the
	// bounded walk instead of rebuilding the same whole owner at every node.
	// Debit each actual visited node, not the prospective maximum. Without
	// that complete headroom, preserve the original per-node refusal path.
	m.mu.Lock()
	room := candidateWorkVector{10: remaining}
	fast := m.owner == m && !m.failed && !m.closed && m.historyGroup != nil && m.historyRecovery == nil
	if fast {
		for coordinate, units := range room {
			if !candidateCapacity(0, m.actual[coordinate], units, m.upper[coordinate]) {
				fast = false
				break
			}
		}
	}
	if fast && m.hasOwner() && m.historyGroup.fitsWorkAfterOwnerCheck(m, room) {
		defer m.mu.Unlock()
		visit = func() bool { m.actual[10]++; return true }
	} else {
		m.mu.Unlock()
	}
	strings, ok := providerHistoryMemberStringWeightWithVisit(reflect.ValueOf(value), 0, &remaining, visit)
	if !ok {
		return 0, false
	}
	weight, ok := providerUpperSum(uint64(reflect.TypeFor[T]().Size()), strings)
	return weight, ok && weight > 0
}

func providerHistoryMemberStringWeightObserved(v reflect.Value, m *providerWorkMeter, depth int, remaining *uint64) (uint64, bool) {
	if m == nil {
		return 0, false
	}
	return providerHistoryMemberStringWeightWithVisit(v, depth, remaining, func() bool { return m.event(providerCompareEvent, 0) })
}

// Only the two fixed internal debit closures above call this walker. Neither
// a visitation function nor a held observer is accepted by any public API.
func providerHistoryMemberStringWeightWithVisit(v reflect.Value, depth int, remaining *uint64, visit func() bool) (uint64, bool) {
	if visit == nil || remaining == nil || *remaining == 0 || depth > 32 || !visit() {
		return 0, false
	}
	*remaining -= 1
	var total uint64
	switch v.Kind() {
	case reflect.String:
		return uint64(v.Len()), true
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			part, ok := providerHistoryMemberStringWeightWithVisit(v.Field(i), depth+1, remaining, visit)
			if !ok {
				return 0, false
			}
			var added bool
			total, added = providerUpperSum(total, part)
			if !added {
				return 0, false
			}
		}
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			part, ok := providerHistoryMemberStringWeightWithVisit(v.Index(i), depth+1, remaining, visit)
			if !ok {
				return 0, false
			}
			var added bool
			total, added = providerUpperSum(total, part)
			if !added {
				return 0, false
			}
		}
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr, reflect.Pointer:
		// Fixed-size comparable leaves; pointer contents are not traversed.
	default:
		return 0, false
	}
	return total, true
}

// With Go comparable values, unequal string lengths fail before byte
// comparison. Equal lengths are bounded by the immutable original weight.
// The debit precedes == even if equal bytes now have different backing.
func providerHistoryMemberEqualObserved[T comparable](actual, original T, weight uint64, m *providerWorkMeter) bool {
	if m == nil || weight == 0 {
		return false
	}
	bytes, ok := providerUpperProduct(weight, 2)
	if !ok {
		return m.refuse()
	}
	return m.event(providerCompareEvent, bytes) && actual == original
}

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
	strings, ok := providerHistoryMemberStringWeightObserved(reflect.ValueOf(value), m, 0, &remaining)
	if !ok {
		return 0, false
	}
	weight, ok := providerUpperSum(uint64(reflect.TypeFor[T]().Size()), strings)
	return weight, ok && weight > 0
}

func providerHistoryMemberStringWeightObserved(v reflect.Value, m *providerWorkMeter, depth int, remaining *uint64) (uint64, bool) {
	if m == nil || remaining == nil || *remaining == 0 || depth > 32 || !m.event(providerCompareEvent, 0) {
		return 0, false
	}
	*remaining -= 1
	var total uint64
	switch v.Kind() {
	case reflect.String:
		return uint64(v.Len()), true
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			part, ok := providerHistoryMemberStringWeightObserved(v.Field(i), m, depth+1, remaining)
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
			part, ok := providerHistoryMemberStringWeightObserved(v.Index(i), m, depth+1, remaining)
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

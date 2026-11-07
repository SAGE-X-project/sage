package registry011

// These are values produced by the bounded standard JSON decoder, not
// arbitrary reflection graphs. Keep the old float64/null semantics while
// observing each entered structural/string/scalar comparison explicitly.
func nodeJSONValueEqualObserved(a, b any, depth int, meter *providerWorkMeter) bool {
	if depth > 32 || !meter.event(providerCompareEvent, 0) {
		return false
	}
	switch x := a.(type) {
	case nil:
		return b == nil
	case string:
		y, ok := b.(string)
		return ok && providerReplayTextEqual(x, y, meter)
	case bool:
		y, ok := b.(bool)
		return ok && x == y
	case float64:
		y, ok := b.(float64)
		return ok && x == y
	case []any:
		y, ok := b.([]any)
		if !ok || len(x) != len(y) {
			return false
		}
		for i := range x {
			if !nodeJSONValueEqualObserved(x[i], y[i], depth+1, meter) {
				return false
			}
		}
		return true
	case map[string]any:
		y, ok := b.(map[string]any)
		if !ok || len(x) != len(y) {
			return false
		}
		for key, v := range x {
			if !meter.event(providerCompareEvent, uint64(len(key))) {
				return false
			}
			w, present := y[key]
			if !present || !nodeJSONValueEqualObserved(v, w, depth+1, meter) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

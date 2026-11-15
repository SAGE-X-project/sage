package registry011

import "reflect"

type providerHistoryStateSerializationLimits struct {
	Keys, Services, Grants uint64
	Record                 bool
}

// One original record or grant serialization's explicit events, including
// entered refusal paths within the prospective cardinalities.
// Builders, the native-tree walk and canonical preflight are distinct work.
// The serializer does not check every clone field's text ceiling: this quote
// uses the original global walker budget, including records rejected by the
// later output cap. A successful-output-only byte bound cannot fund those.
// These are logical work units, not native CPU/RAM or a factory certificate.
func providerHistoryStateSerializationUpper(l providerHistoryStateSerializationLimits) (candidateWorkVector, bool) {
	var empty candidateWorkVector
	if l.Keys > 128 || l.Services > 16 || l.Grants > 128 || (l.Record && l.Grants != 0) || (!l.Record && (l.Keys != 0 || l.Services != 0)) {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	object := uint64(reflect.TypeFor[map[string]any]().Size())
	value := uint64(reflect.TypeFor[any]().Size())
	slot := a.sum(uint64(reflect.TypeFor[string]().Size()), value)
	var nodes, sourceBytes, build, walk, decoded uint64
	if l.Record {
		k, t := l.Keys, l.Services
		decoded = a.product(16, k) // Expiry Number text records codec bytes too.
		nodes = a.sum(13, a.product(17, k), a.product(7, t))
		// The65536 output guard follows canonicalization. Before its
		// decision, the walker may consume its full262144 source-byte cap.
		sourceBytes = 262144
		build = a.sum(object, a.product(6, slot), 51,
			a.product(k, a.sum(a.product(2, value), a.product(2, object), a.product(8, slot), 458)),
			a.product(t, a.sum(a.product(2, value), object, a.product(3, slot), 19)))
		strings := a.sum(10, a.product(14, k), a.product(6, t))
		maps := a.sum(1, a.product(2, k), t)
		fields := a.sum(6, a.product(8, k), a.product(3, t))
		walk = a.sum(nodes, a.product(2, sourceBytes), a.product(2, strings), a.product(257, k),
			a.product(maps, a.sum(object, 1)), a.product(fields, a.sum(slot, 1)),
			a.product(a.sum(k, t), a.sum(a.product(2, value), 1)), 2)
	} else {
		g := l.Grants
		nodes = a.sum(1, a.product(5, g))
		if g != 0 {
			sourceBytes = 262144
		}
		build = a.sum(1, a.product(g, a.sum(a.product(2, value), object, a.product(2, slot), 19)))
		walk = a.sum(nodes, a.product(2, sourceBytes), a.product(8, g), a.product(g, a.sum(object, 1)),
			a.product(a.product(2, g), a.sum(slot, 1)), a.product(g, a.sum(a.product(2, value), 1)), 1)
	}
	// Preserve the actual encode(6*(visited nodes+source bytes)+64)
	// preflight. Final encoded-byte usage alone cannot fund that check.
	maximum := a.sum(a.product(6, a.sum(nodes, sourceBytes)), 64)
	units := a.sum(build, walk, a.product(maximum, maximum), 1)
	if a.failed {
		return empty, false
	}
	var work candidateWorkVector
	work[2], work[10] = decoded, units
	return work, true
}

// Numeric parts for the original parse/clone/mutate/apply stream, not arbitrary
// serializer DATA. Normal applies only to successful original serialization:
// record output<=65536, and grants contain validated42-byte addresses plus one
// of the five fixed scopes (at most15 ASCII bytes, without JSON escapes).
// Envelope additionally covers typed-clone text caps, six-byte ASCII escapes,
// late output refusal and every original encoder preflight in this ONE call.
// Neither part authenticates that route or supplies a production allowance.
type providerHistoryAppliedStateSerializationCost struct {
	normal, envelope     candidateWorkVector
	maximum, normalBytes uint64
}

func providerHistoryAppliedStateSerializationUpper(l providerHistoryStateSerializationLimits) (providerHistoryAppliedStateSerializationCost, bool) {
	var empty providerHistoryAppliedStateSerializationCost
	if l.Keys > 128 || l.Services > 16 || l.Grants > 128 || l.Record && l.Grants != 0 || !l.Record && (l.Keys != 0 || l.Services != 0) {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	object := uint64(reflect.TypeFor[map[string]any]().Size())
	value := uint64(reflect.TypeFor[any]().Size())
	slot := a.sum(uint64(reflect.TypeFor[string]().Size()), value)
	var nodes, sourceBytes, build, walk, decoded, normalBytes uint64
	if l.Record {
		k, t := l.Keys, l.Services
		decoded = a.product(16, k)
		nodes = a.sum(13, a.product(17, k), a.product(7, t))
		// Typed clone field maxima, fixed property names and Number text.
		// contentASCII includes control bytes; retain SIX-byte escaping.
		sourceBytes = a.sum(303, a.product(444, k), a.product(2155, t))
		// Uint formatting consumes at most encode16=257. Its encode20=401
		// preflight is not charged actual; the final larger envelope funds it.
		build = a.sum(object, a.product(6, slot), 51,
			a.product(k, a.sum(a.product(2, value), a.product(2, object), a.product(8, slot), 314)),
			a.product(t, a.sum(a.product(2, value), object, a.product(3, slot), 19)))
		strings := a.sum(10, a.product(14, k), a.product(6, t))
		maps := a.sum(1, a.product(2, k), t)
		fields := a.sum(6, a.product(8, k), a.product(3, t))
		walk = a.sum(nodes, a.product(2, sourceBytes), a.product(2, strings), a.product(257, k),
			a.product(maps, a.sum(object, 1)), a.product(fields, a.sum(slot, 1)),
			a.product(a.sum(k, t), a.sum(a.product(2, value), 1)), 2)
		normalBytes = 65536
	} else {
		g := l.Grants
		nodes, sourceBytes = a.sum(1, a.product(5, g)), a.product(71, g)
		build = a.sum(1, a.product(g, a.sum(a.product(2, value), object, a.product(2, slot), 19)))
		walk = a.sum(nodes, a.product(2, sourceBytes), a.product(8, g), a.product(g, a.sum(object, 1)),
			a.product(a.product(2, g), a.sum(slot, 1)), a.product(g, a.sum(a.product(2, value), 1)), 1)
		normalBytes = 2 // []
		if g != 0 {
			// Each original validated item has at most83 bytes; commas and
			// array brackets give84*g+1. This is NOT a generic ASCII bound.
			normalBytes = a.sum(a.product(84, g), 1)
		}
	}
	maximum := a.sum(a.product(6, a.sum(nodes, sourceBytes)), 64)
	normalBytes = min(normalBytes, maximum)
	var normal, envelope candidateWorkVector
	normal[2], envelope[2] = decoded, decoded
	normal[10] = a.sum(build, walk, a.product(normalBytes, normalBytes), 1)
	// This covers complete entered work AND every outstanding preflight,
	// including numeric formatting. It is never reported as entered actual.
	envelope[10] = a.sum(build, walk, a.product(maximum, maximum), 1)
	if a.failed {
		return empty, false
	}
	return providerHistoryAppliedStateSerializationCost{normal, envelope, maximum, normalBytes}, true
}

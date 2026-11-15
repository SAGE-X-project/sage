package registry011

import (
	"strconv"
	"testing"
)

func TestProviderHistoryCollectionPrefixReadBoundsMatchHandwrittenCounts(t *testing.T) {
	for _, tc := range []struct {
		name              string
		groups, rpc, objs uint64
		want              providerHistoryCollectionReadBound
	}{
		{"one-member", 1, 1, 1, providerHistoryCollectionReadBound{28, 30, 180, 84, 0, 4, 0}},
		{"one-full-group", 1, 32, 64, providerHistoryCollectionReadBound{154, 156, 936, 6720, 1984, 2112, 0}},
		{"two-members", 2, 2, 1, providerHistoryCollectionReadBound{28, 110, 660, 218, 4, 12, 80}},
		{"absent-unit-envelope", 4, 28, 64, providerHistoryCollectionReadBound{154, 1212, 7272, 10644, 1512, 1624, 1056}},
		{"three-full-groups", 3, 96, 64, providerHistoryCollectionReadBound{154, 990, 5940, 36864, 5952, 6336, 834}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := providerHistoryCollectionPrefixReadBound(tc.groups, tc.rpc, tc.objs)
			if !ok || got != tc.want || got.healthyPrefixes != 6*got.planning {
				t.Fatal("read-call rows differ from handwritten counts or double-count planning", got, ok)
			}
		})
	}
}

// Independent chronological enumeration of the actual read-call topology.
// Small bounded numbers only; no authority, clock, source I/O or network.
func historyCollectionEnumeratedReadCounts(members []uint64, objects uint64) providerHistoryCollectionReadBound {
	out := providerHistoryCollectionReadBound{initialModels: 2 * (13 + objects)}
	healthy := uint64(14) + objects
	for group, count := range members {
		out.planning += 2 * healthy
		out.healthyPrefixes += 12 * healthy
		for member := uint64(0); member < count; member++ {
			out.memberReplay += 2*(healthy+19) + 16
			out.memberPreparation += 4 * member
			out.memberRetention += 4 * (member + 1)
		}
		if group != 0 {
			out.advance += 2 * healthy
		}
		healthy += 23 + 2*count
	}
	return out
}

func TestProviderHistoryCollectionReadBoundsCoverSmallerRealizedPrefixes(t *testing.T) {
	for maximumGroups := uint64(1); maximumGroups <= 4; maximumGroups++ {
		for maximumRPC := maximumGroups; maximumRPC <= maximumGroups+5; maximumRPC++ {
			t.Run(strconv.FormatUint(maximumGroups, 10)+"-"+strconv.FormatUint(maximumRPC, 10), func(t *testing.T) {
				bound, ok := providerHistoryCollectionPrefixReadBound(maximumGroups, maximumRPC, 2)
				if !ok {
					t.Fatal("finite prospective envelope")
				}
				var visit func([]uint64, uint64, uint64)
				visit = func(prefix []uint64, groups, remaining uint64) {
					if groups == 0 {
						if remaining != 0 {
							return
						}
						actual := historyCollectionEnumeratedReadCounts(prefix, 2)
						for _, pair := range [7][2]uint64{{actual.initialModels, bound.initialModels}, {actual.planning, bound.planning},
							{actual.healthyPrefixes, bound.healthyPrefixes}, {actual.memberReplay, bound.memberReplay},
							{actual.memberPreparation, bound.memberPreparation}, {actual.memberRetention, bound.memberRetention}, {actual.advance, bound.advance}} {
							if pair[0] > pair[1] {
								t.Fatal("prospective read row underfunds a realized chronological prefix", prefix, actual, bound)
							}
						}
						return
					}
					for count := uint64(1); count <= 32 && count <= remaining; count++ {
						visit(append(prefix, count), groups-1, remaining-count)
					}
				}
				for groups := uint64(1); groups <= maximumGroups; groups++ {
					for rpc := groups; rpc <= maximumRPC && rpc <= 32*groups; rpc++ {
						visit(nil, groups, rpc)
					}
				}
			})
		}
	}
	// The saturated quotient and a partial 33rd member cross different groups.
	bound, ok := providerHistoryCollectionPrefixReadBound(2, 33, 2)
	if !ok {
		t.Fatal("two-group partial saturation")
	}
	for first := uint64(1); first <= 32; first++ {
		actual := historyCollectionEnumeratedReadCounts([]uint64{first, 33 - first}, 2)
		if actual.planning > bound.planning || actual.memberReplay > bound.memberReplay || actual.memberPreparation > bound.memberPreparation || actual.memberRetention > bound.memberRetention || actual.advance > bound.advance {
			t.Fatal("partial saturated group underfunded", first, actual, bound)
		}
	}
}

func TestProviderHistoryCollectionReadBoundsRejectDetachedOrOversizedCounts(t *testing.T) {
	for _, tc := range []struct {
		name              string
		groups, rpc, objs uint64
	}{
		{"zero-groups", 0, 1, 1}, {"zero-rpc", 1, 0, 1}, {"too-few-rpc", 2, 1, 1},
		{"too-many-rpc", 1, 33, 1}, {"zero-objects", 1, 1, 0}, {"too-many-objects", 1, 1, 1<<20 + 1},
		{"too-many-groups", (1<<20)/3 + 1, (1<<20)/3 + 1, 1}, {"group-overflow", ^uint64(0), 1, 1},
		{"rpc-overflow", 1, ^uint64(0), 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got, ok := providerHistoryCollectionPrefixReadBound(tc.groups, tc.rpc, tc.objs); ok || got != (providerHistoryCollectionReadBound{}) {
				t.Fatal("invalid count DATA became a read row", got, ok)
			}
		})
	}
	groups := uint64((1 << 20) / 3)
	if got, ok := providerHistoryCollectionPrefixReadBound(groups, 32*groups, 1<<20); !ok || got.planning == 0 || got.memberReplay == 0 || got.advance == 0 {
		t.Fatal("largest original finite domain overflowed or became empty", got, ok)
	}
}

// Invalid source-cost arithmetic is sticky even when subsequent operands are
// individually valid. A partial result must not silently become a funded row.
func TestProviderHistoryCollectionReadArithmeticNeverRevivesPartialQuotes(t *testing.T) {
	for _, mode := range []string{"sum-overflow", "product-overflow", "difference-underflow"} {
		t.Run(mode, func(t *testing.T) {
			var a providerHistoryReadArithmetic
			if a.sum(2, 3) != 5 || a.product(2, 3) != 6 || a.difference(7, 3) != 4 || a.failed {
				t.Fatal("normal checked arithmetic prerequisite")
			}
			var value uint64
			switch mode {
			case "sum-overflow":
				value = a.sum(^uint64(0), 1)
			case "product-overflow":
				value = a.product(^uint64(0), 2)
			case "difference-underflow":
				value = a.difference(0, 1)
			}
			if value != 0 || !a.failed || a.sum(1) != 0 || a.product(1, 1) != 0 || a.difference(1, 0) != 0 || !a.failed {
				t.Fatal("failed arithmetic revived an incomplete cost quote")
			}
		})
	}
}

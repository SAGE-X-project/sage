package registry011

import "reflect"

type providerHistoryCursorInitializationCost struct {
	configuration, semantic, cursor candidateWorkVector
}

// Only the distinct first-cursor constructor. Earlier interpretation funds
// its own two selected checks; this constructor has a third check, four hash
// text copies, the cursor/header backing value and semantic initialization.
// The semantic row INCLUDES configuration: never add both rows again.
// reflect.Size is the fixed logical copy argument, not physical allocation,
// complete native CPU, heap/backing/peak RAM or a sufficient program quote.
func providerHistoryCursorInitializationUpper() providerHistoryCursorInitializationCost {
	var out providerHistoryCursorInitializationCost
	checks := uint64(reflect.TypeFor[ObservationChecks]().Size())
	fold := uint64(reflect.TypeFor[providerHistorySemanticFold]().Size())
	cursor := uint64(reflect.TypeFor[providerHistoryCollectionCursor]().Size() + reflect.TypeFor[FinalizedHeader]().Size())
	// The real encoder preflight remains64KiB even though the finite JSON
	// schema is below9929B. It is a preflight, not actual decoded bytes/work.
	out.configuration[10] = checks + 1 + providerHistoryJournalEncodeLimit*providerHistoryJournalEncodeLimit + 1
	out.semantic[10] = fold + 1 + out.configuration[10]
	out.cursor[10] = 265 + 265 + cursor + 1 + out.semantic[10]
	return out
}

// Initial models are rebuilt14G times by the original independent readers.
// Cursor copies/commitments and window ABI initialization are separate rows.
func providerHistoryCollectionRepeatedCursorInitializationUpper(groups, requests uint64) (candidateWorkVector, bool) {
	var empty candidateWorkVector
	calls, ok := providerHistoryCollectionRootCallsBound(groups, requests)
	if !ok {
		return empty, false
	}
	one := providerHistoryCursorInitializationUpper()
	var a providerHistoryReadArithmetic
	var work candidateWorkVector
	for c := range work {
		work[c] = a.product(calls.initialModels, one.cursor[c])
	}
	if a.failed {
		return empty, false
	}
	return work, true
}

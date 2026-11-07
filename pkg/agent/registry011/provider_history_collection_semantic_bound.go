package registry011

// Prospective maxima must come from the same original selected manifest and
// cover every successful state, not reported actual lengths. Numeric DATA
// alone supplies no sealed input, caller, reservation or execution authority.
type providerHistoryCollectionSemanticLimits struct {
	Groups, Requests, Blocks, Windows, Versions, PresentStops uint64
}

// These are selected semantic calls only. Observed ABI pairs count window
// opening and the two same-C member copies; native configuration/first-input
// setup, acquisition, other-subject events and final/recovery remain separate.
type providerHistoryCollectionSemanticCallBound struct {
	models, windowOpens, blockApplies, selectedCommands              uint64
	stopSerializations, historyItems, windowCloses, observedABIPairs uint64
}

func providerHistoryCollectionSemanticCallsBound(l providerHistoryCollectionSemanticLimits) (providerHistoryCollectionSemanticCallBound, bool) {
	var empty providerHistoryCollectionSemanticCallBound
	roots, ok := providerHistoryCollectionRootCallsBound(l.Groups, l.Requests)
	if !ok || l.Blocks == 0 || l.Blocks > 1<<20 || l.Windows == 0 || l.Windows > l.Blocks || l.Versions > 1<<20 || l.PresentStops > l.Windows {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	// Every full original prefix contains at most one complete semantic
	// journey. The three current S routes together process at most three
	// further whole journeys: each copies the prior prefix then applies
	// only its next boundary. Member equivalents never apply commands.
	factor := a.sum(roots.prefixes, 3)
	windows := a.product(factor, l.Windows)
	blocks := a.product(factor, l.Blocks)
	commands := a.product(factor, l.Versions)
	// A present state response serializes BOTH record and grants; the
	// later grants response independently serializes both again. Absence
	// retains version zero and enters neither serializer.
	stops := a.product(a.product(2, factor), l.PresentStops)
	abi := a.sum(windows, a.product(2, l.Requests))
	if a.failed {
		return empty, false
	}
	return providerHistoryCollectionSemanticCallBound{roots.initialModels, windows, blocks, commands, stops, commands, windows, abi}, true
}

// Complete selected state.applyObserved and present-stop serialization calls
// only. Command/event parsing, queue/history comparisons, ABI/configuration,
// block/native processing, window clones, roots and physical RAM are separate.
// The returned 2/7/10 subset is not an all-eleven-coordinate cost certificate.
func providerHistoryCollectionRepeatedStateUpper(l providerHistoryCollectionSemanticLimits) (candidateWorkVector, bool) {
	var empty candidateWorkVector
	calls, callsOK := providerHistoryCollectionSemanticCallsBound(l)
	state, stateOK := providerHistoryStateApplyUpper()
	record, recordOK := providerHistoryStateSerializationUpper(providerHistoryStateSerializationLimits{Keys: 128, Services: 16, Record: true})
	grants, grantsOK := providerHistoryStateSerializationUpper(providerHistoryStateSerializationLimits{Grants: 128})
	if !callsOK || !stateOK || !recordOK || !grantsOK {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	var work candidateWorkVector
	for coordinate := range work {
		pair := a.sum(record[coordinate], grants[coordinate])
		work[coordinate] = a.sum(a.product(calls.selectedCommands, state[coordinate]), a.product(calls.stopSerializations, pair))
	}
	if a.failed {
		return empty, false
	}
	return work, true
}

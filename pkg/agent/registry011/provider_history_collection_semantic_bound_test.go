package registry011

import "testing"

func TestProviderHistoryCollectionSemanticCallsPreserveWholePrefixMultiplicity(t *testing.T) {
	for _, tc := range []struct {
		limits providerHistoryCollectionSemanticLimits
		want   providerHistoryCollectionSemanticCallBound
	}{
		{providerHistoryCollectionSemanticLimits{1, 1, 1, 1, 0, 0}, providerHistoryCollectionSemanticCallBound{14, 15, 15, 0, 0, 0, 15, 17}},
		{providerHistoryCollectionSemanticLimits{1, 32, 8, 2, 3, 1}, providerHistoryCollectionSemanticCallBound{14, 30, 120, 45, 30, 45, 30, 94}},
		{providerHistoryCollectionSemanticLimits{2, 33, 12, 3, 11, 3}, providerHistoryCollectionSemanticCallBound{28, 87, 348, 319, 174, 319, 87, 153}},
		{providerHistoryCollectionSemanticLimits{4, 28, 20, 5, 18, 3}, providerHistoryCollectionSemanticCallBound{56, 285, 1140, 1026, 342, 1026, 285, 341}},
	} {
		got, ok := providerHistoryCollectionSemanticCallsBound(tc.limits)
		if !ok || got != tc.want || got.selectedCommands != got.historyItems || got.windowOpens != got.windowCloses || got.observedABIPairs != got.windowOpens+2*tc.limits.Requests {
			t.Fatal("selected semantic calls differ from original prefix/current-S rows", tc, got, ok)
		}
	}
}

func TestProviderHistoryCollectionRepeatedStateCostKeepsProofsAndDoubleStopSerialization(t *testing.T) {
	state, a := providerHistoryStateApplyUpper()
	record, b := providerHistoryStateSerializationUpper(providerHistoryStateSerializationLimits{Keys: 128, Services: 16, Record: true})
	grants, c := providerHistoryStateSerializationUpper(providerHistoryStateSerializationLimits{Grants: 128})
	if !a || !b || !c || state[7] == 0 || record[2] == 0 {
		t.Fatal("independently closed nonzero state/proof/expiry child precursors")
	}
	for _, tc := range []struct {
		limits              providerHistoryCollectionSemanticLimits
		commands, stopPairs uint64
	}{
		{providerHistoryCollectionSemanticLimits{1, 32, 8, 2, 3, 1}, 45, 30},
		{providerHistoryCollectionSemanticLimits{2, 33, 12, 3, 11, 3}, 319, 174},
		{providerHistoryCollectionSemanticLimits{4, 28, 20, 5, 18, 3}, 1026, 342},
	} {
		var want candidateWorkVector
		for coordinate := range want {
			want[coordinate] = tc.commands*state[coordinate] + tc.stopPairs*(record[coordinate]+grants[coordinate])
		}
		got, funded := providerHistoryCollectionRepeatedStateUpper(tc.limits)
		if !funded || got != want || got[2] == 0 || got[7] != tc.commands*128 || got[10] == 0 {
			t.Fatal("selected command/proof/stop work was omitted or reset", tc, got, want, funded)
		}
		for _, coordinate := range []int{0, 1, 3, 4, 5, 6, 8, 9} {
			if got[coordinate] != 0 {
				t.Fatal("state subset invented other execution authority", coordinate, got)
			}
		}
	}
	// No selected subject yet: zero mutation/stop work is legitimate even
	// though block/ABI/window and all other collection work are still required.
	empty := providerHistoryCollectionSemanticLimits{1, 1, 1, 1, 0, 0}
	if got, funded := providerHistoryCollectionRepeatedStateUpper(empty); !funded || got != (candidateWorkVector{}) {
		t.Fatal("zero-version history was rejected or invented state work", got, funded)
	}
	if calls, ok := providerHistoryCollectionSemanticCallsBound(empty); !ok || calls.blockApplies == 0 || calls.observedABIPairs == 0 {
		t.Fatal("zero state work hid required other semantic paths", calls, ok)
	}
}

func TestProviderHistoryCollectionSemanticBoundsRefuseInvalidOrOverflowedQuotes(t *testing.T) {
	valid := providerHistoryCollectionSemanticLimits{2, 33, 12, 3, 11, 3}
	for _, field := range []string{"groups-zero", "groups-large", "requests-small", "requests-large", "blocks-zero", "blocks-large", "windows-zero", "windows-large", "versions-large", "stops-large"} {
		l := valid
		switch field {
		case "groups-zero":
			l.Groups = 0
		case "groups-large":
			l.Groups, l.Requests = 349526, 349526
		case "requests-small":
			l.Requests = 1
		case "requests-large":
			l.Requests = 65
		case "blocks-zero":
			l.Blocks = 0
		case "blocks-large":
			l.Blocks = 1<<20 + 1
		case "windows-zero":
			l.Windows = 0
		case "windows-large":
			l.Windows = 13
		case "versions-large":
			l.Versions = 1<<20 + 1
		case "stops-large":
			l.PresentStops = 4
		}
		if got, ok := providerHistoryCollectionSemanticCallsBound(l); ok || got != (providerHistoryCollectionSemanticCallBound{}) {
			t.Fatal("invalid semantic count became a partial row", field, got, ok)
		}
		if got, ok := providerHistoryCollectionRepeatedStateUpper(l); ok || got != (candidateWorkVector{}) {
			t.Fatal("invalid semantic count became state funding", field, got, ok)
		}
	}
	l := providerHistoryCollectionSemanticLimits{349525, 11184800, 1 << 20, 1 << 20, 1 << 20, 1 << 20}
	if calls, ok := providerHistoryCollectionSemanticCallsBound(l); !ok || calls.selectedCommands == 0 {
		t.Fatal("finite count domain precursor refused before product overflow", calls, ok)
	}
	if got, ok := providerHistoryCollectionRepeatedStateUpper(l); ok || got != (candidateWorkVector{}) {
		t.Fatal("overflow exposed a partial selected-state budget", got, ok)
	}
}

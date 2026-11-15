package registry011

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestProviderHistoryAdmissionClusterControlMatchesIndependentRows(t *testing.T) {
	for _, tc := range []struct{ members, normalDecoded, normalLogical, genericDecoded, genericLogical uint64 }{
		{1, 5286, 12596813, 147460, 23766554546},
		{2, 10056, 20122457, 229380, 25916129512},
		{3, 14826, 28853957, 311300, 28065704478},
		{32, 153156, 806614817, 2686980, 90403378492},
	} {
		got, ok := providerHistoryAdmissionClusterControlUpper(tc.members, 1<<20)
		want := providerHistoryAdmissionClusterControlCost{
			candidateWorkVector{2: tc.normalDecoded, 10: tc.normalLogical},
			candidateWorkVector{2: tc.genericDecoded, 10: tc.genericLogical},
		}
		if !ok || got != want {
			t.Fatal("whole assigned leaf-control subgraph differs from independent rows", tc, got)
		}
	}
	for members := uint64(1); members <= 32; members++ {
		for _, maximum := range []uint64{uint64(len(providerHistoryOriginalPackMagic) + 5), 1024, 1 << 20, 32 << 20} {
			got, ok := providerHistoryAdmissionClusterControlUpper(members, maximum)
			if !ok {
				t.Fatal("finite child domain", members, maximum)
			}
			for c := range got.normal {
				if got.normal[c] > got.envelope[c] {
					t.Fatal("normal escaped complete unchanged generic children", members, maximum, c)
				}
			}
		}
	}
	for _, tc := range []struct{ atoms, decoded, logical uint64 }{
		{1, 2686980, 90403378492}, {2, 2840136, 91209993309}, {10, 4065384, 97662911845},
		{22869334402, 3502575782206536, 18446744073177798109},
	} {
		got, ok := providerHistoryCollectionPointwiseAdmissionClusterControlUpper(tc.atoms, 1<<20)
		if !ok || got != (candidateWorkVector{2: tc.decoded, 10: tc.logical}) {
			t.Fatal("independent completed-normal plus whole-pending leaf row", tc, got)
		}
	}
	// The next logical sum is18446744073984412926, beyond uint64, while
	// decoded=3502575782359692 is still finite. Discard all earlier fields.
	for _, atoms := range []uint64{0, 22869334403, ^uint64(0)} {
		if got, ok := providerHistoryCollectionPointwiseAdmissionClusterControlUpper(atoms, 1<<20); ok || got != (candidateWorkVector{}) {
			t.Fatal("invalid/late-overflow atom count leaked partial work", atoms, got)
		}
	}
}

func historyAdmissionClusterControlFixture(t *testing.T, count int) (*providerHistoryGroupTicket, *providerHistoryOriginalPack, []providerHistoryOriginalInput) {
	t.Helper()
	ticket := historyRecordTicket(t, count, 1, admissionTestDigest("e"))
	inputs := make([]providerHistoryOriginalInput, count)
	for i := range inputs {
		inputs[i] = providerHistoryOriginalInput{"request", ticket.table.members[i].request.wire}
	}
	pack, err := newProviderHistoryOriginalPackObserved(ticket.table.stage, ticket.table.digest, "requests", 0, count, inputs, 1<<20, nil)
	if err != nil {
		t.Fatal("ordinary retained request-pack precursor", err)
	}
	return ticket, pack, inputs
}

// Original CONTROL calls on finite typed DATA only. The second T validation
// is the distinct call made by admit, not a fabricated whole ledger/fence.
// This fixture has no original reader, real nextTable provenance or live role.
func historyAdmissionClusterControlReplay(ticket *providerHistoryGroupTicket, pack *providerHistoryOriginalPack, m *providerWorkMeter) bool {
	bounds, capture := historyGroupBounds()
	parsed, table, err := parseProviderHistoryRequestPackObserved(providerReplayBytes(pack.raw, m), pack.digest,
		ticket.table.stage, ticket.program, ticket.table.purpose, ticket.table.digest, ticket.table.count, bounds, capture, 1<<20, m)
	if err != nil || parsed == nil || *parsed != *pack || table == nil {
		return false
	}
	got, err := parseProviderHistoryGroupTicketObserved(providerReplayBytes(ticket.raw, m), ticket.program, ticket.profile,
		ticket.costs, ticket.previous, ticket.inventory, table, ticket.upper, m)
	return err == nil && got != nil && *got == *ticket && providerHistoryGroupTicketValidObserved(got, m) &&
		providerHistoryCollectionTableEqualObserved(&ticket.table, &got.table, m)
}

func TestProviderHistoryAdmissionClusterControlFundsBothOriginalSubgraphs(t *testing.T) {
	for _, count := range []int{1, 32} {
		ticket, pack, inputs := historyAdmissionClusterControlFixture(t, count)
		beforeT, beforeP := *ticket, *pack
		parts, ok := providerHistoryAdmissionClusterControlUpper(uint64(count), 1<<20)
		if !ok {
			t.Fatal("finite original control quote")
		}
		// Both subjects choose the entire old generic envelope BEFORE entry.
		replay := providerHistoryContentUnitMeter(parts.envelope)
		if !historyAdmissionClusterControlReplay(ticket, pack, replay) || replay.failed {
			t.Fatal("complete parser/original/table/two-T/equality subgraph", count)
		}
		coordinator := providerHistoryContentUnitMeter(parts.envelope)
		built, err := newProviderHistoryOriginalPackObserved(ticket.table.stage, ticket.table.digest, "requests", 0, count, inputs, 1<<20, coordinator)
		if err != nil || built == nil || *built != *pack {
			t.Fatal("original coordinator pack", count, err)
		}
		got, err := newProviderHistoryGroupTicketObserved(ticket.program, ticket.profile, ticket.costs, ticket.previous,
			ticket.inventory, &ticket.table, ticket.upper, coordinator)
		if err != nil || got == nil || *got != *ticket || !providerHistoryGroupTicketValidObserved(got, coordinator) ||
			!providerHistoryCollectionTableEqualObserved(&ticket.table, &got.table, coordinator) || coordinator.failed {
			t.Fatal("original coordinator distinct T calls/full equality", count, err)
		}
		for _, m := range []*providerWorkMeter{replay, coordinator} {
			for c, actual := range m.actual {
				if actual > parts.normal[c] {
					t.Fatal("original subgraph actual escaped completed-normal row", count, c, actual)
				}
			}
		}
		if *ticket != beforeT || *pack != beforeP {
			t.Fatal("control processing changed retained typed DATA")
		}
	}
}

func TestProviderHistoryAdmissionClusterControlKeepsRefusalAndGenericData(t *testing.T) {
	ticket, pack, _ := historyAdmissionClusterControlFixture(t, 1)
	beforeT, beforeP := *ticket, *pack
	parts, ok := providerHistoryAdmissionClusterControlUpper(1, 1<<20)
	probe := providerHistoryContentUnitMeter(parts.envelope)
	if !ok || !historyAdmissionClusterControlReplay(ticket, pack, probe) || probe.actual[2] < 2 {
		t.Fatal("ordinary completed decode precursor")
	}
	short := parts.envelope
	short[2] = probe.actual[2] - 1
	m := providerHistoryContentUnitMeter(short)
	if historyAdmissionClusterControlReplay(ticket, pack, m) || !m.failed || m.actual[2] == 0 || *ticket != beforeT || *pack != beforeP {
		t.Fatal("genuine late decode shortage lost prior work or changed originals")
	}
	prefix := m.actual
	if historyAdmissionClusterControlReplay(ticket, pack, m) || m.actual != prefix {
		t.Fatal("failed original observer resumed or changed its charged prefix")
	}
	// An independently selected full envelope succeeds; the failed observer
	// is never modified. A generic larger request still uses the old domain.
	full := providerHistoryContentUnitMeter(parts.envelope)
	if !historyAdmissionClusterControlReplay(ticket, pack, full) || full.failed {
		t.Fatal("separate complete subject failed")
	}
	bounds, capture := historyGroupBounds()
	r := historyGroupRequest(t, "eth_getCode", []any{strings.Repeat("a", 1024), "0x1"})
	table, err := newProviderHistoryGroupTable(ticket.program, "content", 1, []*providerIntegrationRequest{r}, bounds, capture)
	if err != nil {
		t.Fatal("finite generic request table", err)
	}
	generic, err := newProviderHistoryGroupTicket(ticket.program, ticket.profile, ticket.costs, ticket.previous, ticket.inventory, table, ticket.upper)
	if err != nil {
		t.Fatal("finite generic T", err)
	}
	widePack, err := newProviderHistoryOriginalPackObserved(1, table.digest, "requests", 0, 1,
		[]providerHistoryOriginalInput{{"request", r.wire}}, 1<<20, nil)
	if err != nil {
		t.Fatal("finite generic pack", err)
	}
	beforeGeneric, beforeWide := *generic, *widePack
	wide := providerHistoryContentUnitMeter(parts.envelope)
	if !historyAdmissionClusterControlReplay(generic, widePack, wide) || wide.failed || *generic != beforeGeneric || *widePack != beforeWide {
		t.Fatal("generic original domain was narrowed")
	}
	if providerHistoryCollectionTableEqualObserved(&ticket.table, &generic.table, wide) || wide.failed {
		t.Fatal("generic DATA acquired the different expected table shape")
	}
}

func TestProviderHistoryAdmissionClusterDescriptorExtentAndInvalidDomains(t *testing.T) {
	for count := 1; count <= 32; count++ {
		rows := make([]map[string]string, count)
		for i := range rows {
			rows[i] = map[string]string{"ordinal": "31", "kind": "request", "digest": admissionTestDigest("a"), "bytes": "18446744073709551615"}
		}
		// Conservative ASCII object-size oracle, not a valid20-digit stage or
		// payload claim. It exceeds every allowed decimal field width.
		raw, err := json.Marshal(map[string]any{"format": providerHistoryOriginalPackDomain, "stage": "18446744073709551615",
			"table": admissionTestDigest("b"), "purpose": "requests", "first": "0", "count": "32", "entries": rows})
		if err != nil || len(raw) > 512+192*count {
			t.Fatal("independent complete descriptor extent", count, len(raw), err)
		}
	}
	for _, members := range []uint64{0, 33, ^uint64(0)} {
		if got, ok := providerHistoryAdmissionClusterControlUpper(members, 1<<20); ok || got != (providerHistoryAdmissionClusterControlCost{}) {
			t.Fatal("invalid cardinality leaked partial component", members, got)
		}
	}
	for _, maximum := range []uint64{0, uint64(len(providerHistoryOriginalPackMagic) + 4), 32<<20 + 1, ^uint64(0)} {
		if got, ok := providerHistoryAdmissionClusterControlUpper(1, maximum); ok || got != (providerHistoryAdmissionClusterControlCost{}) {
			t.Fatal("invalid raw cap leaked partial component", maximum, got)
		}
		if got, ok := providerHistoryCollectionPointwiseAdmissionClusterControlUpper(1, maximum); ok || got != (candidateWorkVector{}) {
			t.Fatal("invalid child cap leaked pointwise component", maximum, got)
		}
	}
}

func TestProviderHistoryAdmissionAtomCountsKeepTheOriginalWindowAndEveryStageEnvelope(t *testing.T) {
	for _, row := range []struct{ groups, whole, uniform uint64 }{
		{1, 11, 11}, {2, 36, 23}, {3, 75, 35}, {6, 276, 71}, {7, 371, 83}, {8, 480, 97},
		{32, 7296, 433}, {9226, 595870436, 129149}, {349525, 855175477475, 4893335},
	} {
		for _, requests := range []uint64{row.groups, row.groups * 32} {
			whole, a := providerHistoryCollectionWholeAdmissionAtomsBound(row.groups, requests)
			uniform, c := providerHistoryCollectionUniformAdmissionAtomsBound(row.groups, requests)
			if !a || !c || whole != row.whole || uniform != row.uniform {
				t.Fatal("independent whole/window literal", row, requests, whole, uniform)
			}
		}
	}
	// Independent phase inventory: T planning2j; newT/readbacks2j+3;
	// C3j+3; dispatch2j+2; S3j+3; ownadvance2(j+1) when nonfinal.
	// This is arithmetic over pinned source roles, not runtime call counting.
	for groups := uint64(1); groups <= 32; groups++ {
		var total uint64
		for prior := uint64(0); prior < groups; prior++ {
			want := 2*prior + (2*prior + 3) + (3*prior + 3) + (2*prior + 2) + (3*prior + 3)
			advance := prior+1 < groups
			if advance {
				want += 2 * (prior + 1)
			}
			got, ok := providerHistoryCollectionStageAdmissionAtomsBound(prior, advance)
			if !ok || got != want {
				t.Fatal("original phase/window cut", groups, prior, got, want)
			}
			total += got
		}
		got, ok := providerHistoryCollectionWholeAdmissionAtomsBound(groups, groups)
		if !ok || got != total {
			t.Fatal("all real stage counts do not sum to whole topology", groups)
		}
	}
	const maximum = uint64(1 << 20)
	atoms, ok := providerHistoryCollectionUniformAdmissionAtomsBound(9226, 9226)
	part, valid := providerHistoryCollectionPointwiseAdmissionClusterControlUpper(atoms, maximum)
	if !ok || !valid || part != (candidateWorkVector{2: 19782478068, 10: 104263093764408}) {
		t.Fatal("current R2 component and source-count row disagree")
	}
	// Each original window retains E. A single whole-program E would lose
	// (G-1) copies of the envelope headroom and cannot fund original SUM uppers.
	wholeAtoms, _ := providerHistoryCollectionWholeAdmissionAtomsBound(9226, 9226)
	whole, valid := providerHistoryCollectionPointwiseAdmissionClusterControlUpper(wholeAtoms, maximum)
	if !valid || whole != (candidateWorkVector{2: 91261135029840, 10: 480638012286613887}) ||
		part[2]*9226 != 182513142655368 || part[10]*9226 != 961931303070428208 ||
		part[2]*9226 <= whole[2] || part[10]*9226 <= whole[10] {
		t.Fatal("uniform full-stage SUM was replaced by global pointwise row")
	}
	// These finite numbers cover only this component, not a chosen real
	// profile, native/RAM cost, guard suffix or production compiler.
}

func TestProviderHistoryAdmissionAtomCountsRejectInvalidDomainsWithCompleteZero(t *testing.T) {
	for _, row := range []struct{ groups, requests uint64 }{
		{0, 0}, {0, 1}, {1, 0}, {2, 1}, {1, 33}, {349526, 349526}, {^uint64(0), ^uint64(0)},
	} {
		if got, ok := providerHistoryCollectionWholeAdmissionAtomsBound(row.groups, row.requests); ok || got != 0 {
			t.Fatal("invalid whole atom domain", row)
		}
		if got, ok := providerHistoryCollectionUniformAdmissionAtomsBound(row.groups, row.requests); ok || got != 0 {
			t.Fatal("invalid uniform atom domain", row)
		}
	}
	for _, row := range []struct {
		prior   uint64
		advance bool
	}{
		{349525, false}, {349524, true}, {^uint64(0), false}, {^uint64(0), true},
	} {
		if got, ok := providerHistoryCollectionStageAdmissionAtomsBound(row.prior, row.advance); ok || got != 0 {
			t.Fatal("invalid original stage atom domain", row)
		}
	}
	if got, ok := providerHistoryCollectionStageAdmissionAtomsBound(349524, false); !ok || got != 4194299 {
		t.Fatal("last permitted original stage")
	}
	if got, ok := providerHistoryCollectionStageAdmissionAtomsBound(349523, true); !ok || got != 4893335 {
		t.Fatal("last permitted advancing original stage")
	}
	atoms, ok := providerHistoryCollectionWholeAdmissionAtomsBound(349525, 349525)
	if !ok {
		t.Fatal("maximum topology count must itself remain finite")
	}
	if work, valid := providerHistoryCollectionPointwiseAdmissionClusterControlUpper(atoms, 1<<20); valid || work != (candidateWorkVector{}) {
		t.Fatal("finite topology was accepted as a sufficient overflowing work allowance")
	}
}

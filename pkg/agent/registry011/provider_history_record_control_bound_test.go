package registry011

import (
	"crypto/tls"
	"testing"
)

func TestProviderHistoryRecordControlCostsMatchIndependentRows(t *testing.T) {
	for _, row := range []struct {
		members, decoded, ticket, consumption, settlement, context uint64
	}{
		{1, 16384, 4832722487, 4899864432, 9799679441, 9732521377},
		{2, 32768, 5303040786, 5370182731, 10740316039, 10673157975},
		{32, 524288, 19412589756, 19479731701, 38959413979, 38892255915},
	} {
		got, ok := providerHistoryRecordControlUpper(row.members)
		want := providerHistoryRecordControlCost{
			ticket:      candidateWorkVector{2: row.decoded, 10: row.ticket},
			consumption: candidateWorkVector{2: row.decoded, 10: row.consumption},
			settlement:  candidateWorkVector{2: 2 * row.decoded, 10: row.settlement},
			context:     candidateWorkVector{2: 2 * row.decoded, 10: row.context},
		}
		if !ok || got != want {
			t.Fatal("independently derived record/context row", row.members, got, want)
		}
	}
}

func TestProviderHistoryTLSControlCostsMatchIndependentRows(t *testing.T) {
	for _, row := range []struct {
		members, decoded, original, parsed uint64
	}{
		{1, 32768, 14027604970, 18322652581},
		{2, 65536, 14968241568, 19263289179},
		{32, 1048576, 43187339508, 47482387119},
	} {
		got, ok := providerHistoryTLSControlUpper(row.members, providerTLSBounds{1, 1, 8192})
		want := providerHistoryTLSControlCost{
			original: candidateWorkVector{2: row.decoded, 10: row.original},
			parsed:   candidateWorkVector{2: row.decoded + 65536, 10: row.parsed},
		}
		if !ok || got != want {
			t.Fatal("independent diagnostic TLS row", row.members, got, want)
		}
		maximum, ok := providerHistoryTLSControlUpper(row.members, providerTLSBounds{8, 16, 65536})
		want.original[10] += 8509 // 67*(128-1) digest copies; no new codec.
		if !ok || maximum != want {
			t.Fatal("maximum certificate DATA copy row", row.members, maximum, want)
		}
	}
}

func TestProviderHistoryRecordControlCostsFundOriginalRecordOperations(t *testing.T) {
	for _, count := range []int{1, providerHistoryGroupMembers} {
		ticket := historyRecordTicket(t, count, 1, admissionTestDigest("e"))
		consumption, err := newProviderHistoryGroupConsumption(ticket, admissionTestDigest("f"), admissionTestDigest("1"))
		if err != nil {
			t.Fatal("original C precursor", err)
		}
		digests := make([]string, count)
		for i := range digests {
			digests[i] = admissionTestDigest("2")
		}
		settlement, err := newProviderHistoryGroupSettlement(ticket, consumption, admissionTestDigest("3"), admissionTestDigest("4"), "complete", "", digests)
		if err != nil {
			t.Fatal("original S precursor", err)
		}
		beforeT, beforeC, beforeS := *ticket, *consumption, *settlement
		upper, ok := providerHistoryRecordControlUpper(uint64(count))
		if !ok {
			t.Fatal("finite record quote")
		}
		for _, row := range []struct {
			name  string
			upper candidateWorkVector
			run   func(*providerWorkMeter) bool
		}{
			{"T-constructor", upper.ticket, func(m *providerWorkMeter) bool {
				v, e := newProviderHistoryGroupTicketObserved(ticket.program, ticket.profile, ticket.costs, ticket.previous, ticket.inventory, &ticket.table, ticket.upper, m)
				return e == nil && v != nil && *v == *ticket
			}},
			{"T-parser-with-input-copy", upper.ticket, func(m *providerWorkMeter) bool {
				v, e := parseProviderHistoryGroupTicketObserved(providerReplayBytes(ticket.raw, m), ticket.program, ticket.profile, ticket.costs, ticket.previous, ticket.inventory, &ticket.table, ticket.upper, m)
				return e == nil && v != nil && *v == *ticket
			}},
			{"C-constructor", upper.consumption, func(m *providerWorkMeter) bool {
				v, e := newProviderHistoryGroupConsumptionObserved(ticket, consumption.previous, consumption.inventory, m)
				return e == nil && v != nil && *v == *consumption
			}},
			{"C-parser-with-input-copy", upper.consumption, func(m *providerWorkMeter) bool {
				v, e := parseProviderHistoryGroupConsumptionObserved(providerReplayBytes(consumption.raw, m), ticket, consumption.previous, consumption.inventory, m)
				return e == nil && v != nil && *v == *consumption
			}},
			{"S-constructor-and-fixed-array-equality", upper.settlement, func(m *providerWorkMeter) bool {
				v, e := newProviderHistoryGroupSettlementObserved(ticket, consumption, settlement.previous, settlement.inventory, settlement.outcome, settlement.failure, settlement.results[:settlement.count], m)
				return e == nil && v != nil && providerHistoryGroupSettlementEqualObserved(v, settlement, m)
			}},
			{"S-parser-with-input-copy", upper.settlement, func(m *providerWorkMeter) bool {
				v, e := parseProviderHistoryGroupSettlementObserved(providerReplayBytes(settlement.raw, m), ticket, consumption, settlement.previous, settlement.inventory, settlement.outcome, settlement.failure, settlement.results[:settlement.count], m)
				return e == nil && v != nil && *v == *settlement
			}},
			{"context-first-ordinal", upper.context, func(m *providerWorkMeter) bool {
				return providerHistoryResultContextObserved(ticket, consumption, 0, m)
			}},
			{"context-last-ordinal", upper.context, func(m *providerWorkMeter) bool {
				return providerHistoryResultContextObserved(ticket, consumption, count-1, m)
			}},
		} {
			m := providerHistoryContentUnitMeter(row.upper)
			if !row.run(m) || m.failed || *ticket != beforeT || *consumption != beforeC || *settlement != beforeS {
				t.Fatal("original record was unfunded or changed", count, row.name)
			}
			for i, actual := range m.actual {
				if actual > row.upper[i] {
					t.Fatal("explicit record event exceeds quote", count, row.name, i)
				}
			}
		}
	}
}

func TestProviderHistoryTLSControlCostsFundOriginalDiagnosticData(t *testing.T) {
	for _, count := range []int{1, providerHistoryGroupMembers} {
		ticket, consumption, inputs, roots, certificates := historyResultDataFixture(t, count)
		ordinal := count - 1
		request := ticket.table.members[ordinal].request
		sample := &ProviderTransportSample{Endpoint: SepoliaRPCURL, Method: request.method, HTTPVersion: "HTTP/1.1", RequestHead: []byte(request.head), RequestBody: []byte(request.body), Complete: true, ServerName: "rpc.sepolia.ethpandaops.io", TLSVersion: tls.VersionTLS13, CipherSuite: tls.TLS_AES_128_GCM_SHA256, VerifiedChains: [][]string{{admissionTestDigest("3")}}}
		beforeT, beforeC, original := *ticket, *consumption, inputs[2*ordinal+1].raw
		upper, ok := providerHistoryTLSControlUpper(uint64(count), certificates)
		if !ok {
			t.Fatal("finite diagnostic TLS quote")
		}
		construction := providerHistoryContentUnitMeter(upper.original)
		got, err := newProviderHistoryTLSOriginalObserved(ticket, consumption, ordinal, sample, roots, certificates, construction)
		if err != nil || got != original || construction.failed {
			t.Fatal("diagnostic DATA constructor differs", count, err)
		}
		parsed := providerHistoryContentUnitMeter(upper.parsed)
		if parseProviderHistoryTLSOriginalObserved(providerReplayBytes(original, parsed), ticket, consumption, ordinal, roots, certificates, parsed) != nil || parsed.failed || *ticket != beforeT || *consumption != beforeC || inputs[2*ordinal+1].raw != original {
			t.Fatal("diagnostic DATA parser was unfunded or changed originals", count)
		}
		for _, row := range []struct {
			meter *providerWorkMeter
			upper candidateWorkVector
		}{{construction, upper.original}, {parsed, upper.parsed}} {
			for i, actual := range row.meter.actual {
				if actual > row.upper[i] {
					t.Fatal("explicit diagnostic event exceeds quote", count, i)
				}
			}
		}
	}
}

func TestProviderHistoryResultContextCountsCoverOriginalRepeatedPrefixes(t *testing.T) {
	for _, row := range []struct {
		groups, requests, packs, tls, contexts, decoded, logical uint64
	}{
		{1, 1, 5, 5, 11, 11862016, 475060554678},
		{1, 32, 129, 2144, 2305, 2557476864, 108201333860427},
		{2, 2, 24, 28, 54, 58458112, 2349295660308},
		{2, 33, 148, 2597, 2778, 3083141120, 130492995427227},
		{3, 96, 429, 7776, 8301, 9213837312, 390053804617647},
		{32, 32, 7104, 9088, 16224, 17607688192, 709192515021888},
	} {
		got, ok := providerHistoryCollectionResultContextCallsBound(row.groups, row.requests)
		want := providerHistoryResultContextCalls{row.requests, row.packs, row.tls, row.contexts}
		work, workOK := providerHistoryCollectionRepeatedResultControlUpper(row.groups, row.requests, providerTLSBounds{1, 1, 8192})
		if !ok || got != want || !workOK || work != (candidateWorkVector{2: row.decoded, 10: row.logical}) {
			t.Fatal("independent repeated-context row", row, got, work)
		}
	}
	// Enumerate actual source call topology independently of the O(1) formula.
	// Also realize fewer groups than G; packing increases member prefix work.
	for _, members := range [][]uint64{{1}, {32}, {1, 1}, {32, 1}, {1, 32}, {16, 17}, {32, 32, 32}, {1, 2, 3}, {32, 1, 1}} {
		var requests, packs, tlsParses uint64
		for group, count := range members {
			requests += count
			for ordinal := uint64(0); ordinal < count; ordinal++ {
				if ordinal > 0 {
					packs += 2
					tlsParses += 2 * ordinal
				}
				packs += 2
				tlsParses += 2 * (ordinal + 1)
			}
			packs += 3
			tlsParses += 3 * count
			for range 12 {
				for _, previous := range members[:group] {
					packs++
					tlsParses += previous
				}
			}
			if group+1 < len(members) {
				for range 2 {
					for _, previous := range members[:group+1] {
						packs++
						tlsParses += previous
					}
				}
			}
		}
		for groups := uint64(len(members)); groups <= min(requests, uint64(len(members))+2); groups++ {
			upper, ok := providerHistoryCollectionResultContextCallsBound(groups, requests)
			if !ok || upper.tlsOriginals < requests || upper.packParses < packs || upper.tlsParses < tlsParses || upper.contexts < requests+packs+tlsParses {
				t.Fatal("actual source topology exceeded prospective count", members, groups, upper)
			}
		}
	}
}

func TestProviderHistoryRecordControlRefusesUnfundedAndInvalidInputs(t *testing.T) {
	ticket, consumption, inputs, roots, certificates := historyResultDataFixture(t, 2)
	beforeT, beforeC := *ticket, *consumption
	record, ok := providerHistoryRecordControlUpper(2)
	tlsUpper, tlsOK := providerHistoryTLSControlUpper(2, certificates)
	if !ok || !tlsOK {
		t.Fatal("valid cost precursor")
	}
	contextProbe := providerHistoryContentUnitMeter(record.context)
	if !providerHistoryResultContextObserved(ticket, consumption, 0, contextProbe) || contextProbe.failed || contextProbe.actual[2] == 0 {
		t.Fatal("valid full context precursor")
	}
	short := record.settlement
	short[2] = contextProbe.actual[2] - 1
	m := providerHistoryContentUnitMeter(short)
	if s, err := newProviderHistoryGroupSettlementObserved(ticket, consumption, admissionTestDigest("3"), admissionTestDigest("4"), "complete", "", []string{admissionTestDigest("5"), admissionTestDigest("6")}, m); err == nil || s != nil || !m.failed || m.event(providerCopyEvent, 0) {
		t.Fatal("nested record validation exceeded original decoded allowance")
	}
	validTLS := providerHistoryContentUnitMeter(tlsUpper.parsed)
	raw := inputs[1].raw
	if parseProviderHistoryTLSOriginalObserved(providerReplayBytes(raw, validTLS), ticket, consumption, 0, roots, certificates, validTLS) != nil || validTLS.failed {
		t.Fatal("valid original TLS parser precursor")
	}
	short = tlsUpper.parsed
	short[2] = contextProbe.actual[2] + uint64(len(raw)) - 1
	m = providerHistoryContentUnitMeter(short)
	if parseProviderHistoryTLSOriginalObserved(providerReplayBytes(raw, m), ticket, consumption, 0, roots, certificates, m) == nil || !m.failed || m.actual[2] != contextProbe.actual[2] || m.event(providerCopyEvent, 0) || *ticket != beforeT || *consumption != beforeC || inputs[1].raw != raw {
		t.Fatal("one-short TLS decode did not preserve the successful context prefix")
	}
	ordinalProbe := providerHistoryContentUnitMeter(record.context)
	if providerHistoryResultContextObserved(ticket, consumption, ticket.table.count, ordinalProbe) || ordinalProbe.failed || ordinalProbe.actual[2] == 0 {
		t.Fatal("ordinal refusal must remain distinct from an unfunded context")
	}
	for _, members := range []uint64{0, 33, ^uint64(0)} {
		if got, valid := providerHistoryRecordControlUpper(members); valid || got != (providerHistoryRecordControlCost{}) {
			t.Fatal("invalid cardinality left a record quote")
		}
		if got, valid := providerHistoryTLSControlUpper(members, certificates); valid || got != (providerHistoryTLSControlCost{}) {
			t.Fatal("invalid cardinality left a TLS quote")
		}
	}
	for _, bounds := range []providerTLSBounds{{}, {9, 1, 1}, {1, 17, 1}, {1, 1, 65537}, {-1, 1, 1}, {1, 1, 0}} {
		if got, valid := providerHistoryTLSControlUpper(1, bounds); valid || got != (providerHistoryTLSControlCost{}) {
			t.Fatal("invalid certificate limits left a partial quote")
		}
		if work, valid := providerHistoryCollectionRepeatedResultControlUpper(1, 1, bounds); valid || work != (candidateWorkVector{}) {
			t.Fatal("invalid certificate limits left repeated work")
		}
	}
	for _, row := range [][2]uint64{{0, 1}, {1, 0}, {1, 33}, {3, 2}, {349526, 349526}, {^uint64(0), ^uint64(0)}} {
		if calls, valid := providerHistoryCollectionResultContextCallsBound(row[0], row[1]); valid || calls != (providerHistoryResultContextCalls{}) {
			t.Fatal("invalid G/R left a call count")
		}
		if work, valid := providerHistoryCollectionRepeatedResultControlUpper(row[0], row[1], certificates); valid || work != (candidateWorkVector{}) {
			t.Fatal("invalid G/R left partial work")
		}
	}
	// A finite valid count row reaches composition before logical overflow.
	if calls, valid := providerHistoryCollectionResultContextCallsBound(349525, 11184800); !valid || calls.contexts == 0 {
		t.Fatal("maximum finite count precursor")
	}
	if work, valid := providerHistoryCollectionRepeatedResultControlUpper(349525, 11184800, certificates); valid || work != (candidateWorkVector{}) {
		t.Fatal("finite logical overflow leaked an earlier decoded coordinate")
	}
}

func TestProviderHistoryPlannedTLSControlMatchesIndependentRows(t *testing.T) {
	for _, row := range []struct {
		chains, certificates                        int
		bytes, original, parsedDecoded, parsed, raw uint64
	}{
		{1, 1, 2122, 453928631, 66122, 458433422, 43187339508},
		{2, 4, 2616, 456270166, 66616, 463116048, 43187339977},
		{8, 16, 11040, 571324774, 75040, 693217608, 43187348017},
	} {
		got, ok := providerHistoryPlannedTLSControlUpper(32, providerTLSBounds{row.chains, row.certificates, 8192})
		want := providerHistoryPlannedTLSControlCost{
			original:    providerHistoryControlPair{candidateWorkVector{2: 64000, 10: row.original}, candidateWorkVector{2: 1048576, 10: row.raw}},
			parsed:      providerHistoryControlPair{candidateWorkVector{2: row.parsedDecoded, 10: row.parsed}, candidateWorkVector{2: 1114112, 10: 47482387119}},
			context:     providerHistoryControlPair{candidateWorkVector{2: 64000, 10: 449421284}, candidateWorkVector{2: 1048576, 10: 38892255915}},
			normalBytes: row.bytes,
		}
		if !ok || got != want {
			t.Fatal("independent closed diagnostic TLS/context pairs", row, got)
		}
	}
	for members := uint64(1); members <= 32; members++ {
		for _, bounds := range []providerTLSBounds{{1, 1, 1}, {8, 16, 65536}} {
			got, ok := providerHistoryPlannedTLSControlUpper(members, bounds)
			if !ok || got.normalBytes > 11040 {
				t.Fatal("finite original selected TLS bounds", members, bounds)
			}
			for _, pair := range []providerHistoryControlPair{got.original, got.parsed, got.context} {
				for coordinate, normal := range pair.normal {
					if normal > pair.envelope[coordinate] {
						t.Fatal("qualified normal exceeded unchanged unfinished envelope", members, coordinate)
					}
				}
			}
		}
	}
}

func TestProviderHistoryPlannedTLSControlFundsOriginalDiagnosticsUnderFullGuards(t *testing.T) {
	for _, count := range []int{1, 32} {
		ticket := historyRecordTicket(t, count, 1, admissionTestDigest("e"))
		consumption, err := newProviderHistoryGroupConsumption(ticket, admissionTestDigest("f"), admissionTestDigest("1"))
		if err != nil {
			t.Fatal("original typed C precursor", err)
		}
		beforeT, beforeC := *ticket, *consumption
		for _, bounds := range []providerTLSBounds{{1, 1, 8192}, {2, 4, 8192}, {8, 16, 65536}} {
			parts, ok := providerHistoryPlannedTLSControlUpper(uint64(count), bounds)
			if !ok {
				t.Fatal("complete diagnostic pair precursor")
			}
			ordinal, roots := count-1, admissionTestDigest("2")
			request := ticket.table.members[ordinal].request
			chains := make([][]string, bounds.chains)
			for chain := range chains {
				chains[chain] = make([]string, bounds.certificates)
				for certificate := range chains[chain] {
					chains[chain][certificate] = admissionTestDigest("3")
				}
			}
			// Finite diagnostic DATA only; no certificate bytes, connection,
			// native verification or real origin claim are created by this test.
			sample := &ProviderTransportSample{Endpoint: SepoliaRPCURL, Method: request.method, HTTPVersion: "HTTP/1.1", RequestHead: []byte(request.head), RequestBody: []byte(request.body), Complete: true, ServerName: "rpc.sepolia.ethpandaops.io", TLSVersion: tls.VersionTLS13, CipherSuite: tls.TLS_AES_128_GCM_SHA256, VerifiedChains: chains}
			construction := providerHistoryContentUnitMeter(parts.original.envelope)
			raw, err := newProviderHistoryTLSOriginalObserved(ticket, consumption, ordinal, sample, roots, bounds, construction)
			if err != nil || raw == "" || uint64(len(raw)) > parts.normalBytes || construction.failed {
				t.Fatal("unchanged original diagnostic constructor or complete ASCII shape", count, bounds, err)
			}
			parsed := providerHistoryContentUnitMeter(parts.parsed.envelope)
			if parseProviderHistoryTLSOriginalObserved(providerReplayBytes(raw, parsed), ticket, consumption, ordinal, roots, bounds, parsed) != nil || parsed.failed || *ticket != beforeT || *consumption != beforeC {
				t.Fatal("original diagnostic parser refused or changed T/C", count, bounds)
			}
			for _, row := range []struct {
				meter  *providerWorkMeter
				normal candidateWorkVector
			}{{construction, parts.original.normal}, {parsed, parts.parsed.normal}} {
				for coordinate, actual := range row.meter.actual {
					if actual > row.normal[coordinate] {
						t.Fatal("completed original diagnostic escaped qualified normal", count, bounds, coordinate, actual)
					}
				}
			}
			// Successful normal DATA must not become a smaller incoming budget.
			short := providerHistoryContentUnitMeter(parts.original.normal)
			if v, err := newProviderHistoryTLSOriginalObserved(ticket, consumption, ordinal, sample, roots, bounds, short); err == nil || v != "" || !short.failed || short.actual[10] == 0 || *ticket != beforeT || *consumption != beforeC {
				t.Fatal("original wide guards were bypassed by a normal diagnostic row", count, bounds)
			}
		}
	}
}

func TestProviderHistoryRoleResultControlCountsMatchIndependentCuts(t *testing.T) {
	for _, row := range []struct {
		prior, past, members, ordinal uint64
		roles                         [6][3]uint64
	}{
		{0, 0, 1, 0, [6][3]uint64{{0, 0, 0}, {0, 0, 0}, {1, 2, 2}, {0, 3, 3}, {0, 0, 5}, {0, 2, 2}}},
		{0, 0, 2, 1, [6][3]uint64{{0, 0, 0}, {0, 2, 2}, {1, 6, 4}, {0, 6, 3}, {0, 3, 8}, {0, 4, 2}}},
		{1, 32, 32, 31, [6][3]uint64{{0, 288, 9}, {0, 62, 2}, {1, 126, 4}, {0, 192, 6}, {0, 189, 11}, {0, 128, 4}}},
		{7, 224, 32, 31, [6][3]uint64{{0, 2016, 63}, {0, 62, 2}, {1, 126, 4}, {0, 768, 24}, {0, 765, 29}, {0, 512, 16}}},
	} {
		var roles [6]providerHistoryRoleResultControlCalls
		for i, r := range row.roles {
			roles[i] = providerHistoryRoleResultControlCalls{r[0], r[1], r[2]}
		}
		want := providerHistoryRoleResultControlCounts{roles[0], roles[1], roles[2], roles[3], roles[4], roles[5]}
		got, ok := providerHistoryCollectionRoleResultControlCallsBound(row.prior, row.past, row.members, row.ordinal, true)
		if !ok || got != want {
			t.Fatal("independent prefix/member/known-failure context source cuts", row, got)
		}
		want.advance = providerHistoryRoleResultControlCalls{}
		got, ok = providerHistoryCollectionRoleResultControlCallsBound(row.prior, row.past, row.members, row.ordinal, false)
		if !ok || got != want {
			t.Fatal("final closure acquired an extra advance", row, got)
		}
	}
}

func TestProviderHistoryRoleResultControlConservesVariableMemberGrammar(t *testing.T) {
	for groups := uint64(1); groups <= 64; groups++ {
		var requests, squares, weighted uint64
		var total providerHistoryRoleResultControlCalls
		add := func(r providerHistoryRoleResultControlCalls) {
			total.originals += r.originals
			total.parsed += r.parsed
			total.contexts += r.contexts
		}
		for prior := uint64(0); prior < groups; prior++ {
			members := 1 + (19*prior+7*groups)%32
			calls, ok := providerHistoryCollectionRoleResultControlCallsBound(prior, requests, members, 0, prior+1 < groups)
			if !ok {
				t.Fatal("valid finite variable-cardinality precursor")
			}
			add(calls.before)
			add(calls.successClosure)
			add(calls.advance)
			for ordinal := uint64(0); ordinal < members; ordinal++ {
				step, ok := providerHistoryCollectionRoleResultControlCallsBound(prior, requests, members, ordinal, false)
				if !ok {
					t.Fatal("valid original member ordinal")
				}
				add(step.member) // P is already inside actual M, not added twice.
				failurePacks := uint64(0)
				if ordinal != 0 {
					failurePacks = 3
				}
				if step.failedClosure.contexts != 3*prior+failurePacks+5 {
					t.Fatal("failed constructor plus two rebuilding parsers require five extra contexts")
				}
			}
			requests += members
			squares += members * members
			weighted += (groups - 1 - prior) * members
		}
		tri := groups * (groups - 1) / 2
		if total.originals != requests || total.contexts != 4*requests+groups+14*tri || total.parsed != 2*squares+3*requests+14*weighted {
			t.Fatal("independent whole grammar or TLS-internal context was duplicated", groups, total)
		}
		old, ok := providerHistoryCollectionResultContextCallsBound(groups, requests)
		if !ok || old.tlsOriginals != requests || old.packParses != total.contexts || old.tlsParses < total.parsed || old.contexts < total.originals+total.parsed+total.contexts {
			t.Fatal("actual role grammar exceeded existing prospective aggregate", groups, total)
		}
	}
}

func TestProviderHistoryRoleResultControlKeepsFailedClosureAndPreparationCosts(t *testing.T) {
	for _, row := range []struct {
		prior, members                 uint64
		before, prep, member, close, a [2]uint64
	}{
		{0, 1, [2]uint64{}, [2]uint64{}, [2]uint64{3341880, 130474520343}, [2]uint64{2423920, 88199717698}, [2]uint64{2293304, 87287180366}},
		{0, 2, [2]uint64{}, [2]uint64{2293304, 87287180366}, [2]uint64{3736344, 133225827103}, [2]uint64{2743920, 90446824118}, [2]uint64{2426536, 88213412462}},
		{1, 32, [2]uint64{21793480, 222884319082}, [2]uint64{6290264, 115074143246}, [2]uint64{11730264, 188799752863}, [2]uint64{15326496, 177934672898}, [2]uint64{10814920, 146538644982}},
		{7, 32, [2]uint64{140361928, 1047417599362}, [2]uint64{6290264, 115074143246}, [2]uint64{11730264, 188799752863}, [2]uint64{54849312, 452779099658}, [2]uint64{37163464, 329768262822}},
	} {
		vector := func(v [2]uint64) candidateWorkVector { return candidateWorkVector{2: v[0], 10: v[1]} }
		got, ok := providerHistoryCollectionRoleResultControlPartsUpper(row.prior, row.members, providerTLSBounds{2, 4, 8192}, true)
		want := providerHistoryRoleResultControlParts{before: vector(row.before), preparation: vector(row.prep), member: vector(row.member), closure: vector(row.close), advance: vector(row.a)}
		if !ok || got != want {
			t.Fatal("independent role prices with full first-refusal envelopes", row, got)
		}
		for coordinate, preparation := range got.preparation {
			if preparation > got.member[coordinate] {
				t.Fatal("literal remaining guard cannot cover its already-entered preparation", coordinate)
			}
		}
		want.advance = candidateWorkVector{}
		got, ok = providerHistoryCollectionRoleResultControlPartsUpper(row.prior, row.members, providerTLSBounds{2, 4, 8192}, false)
		if !ok || got != want {
			t.Fatal("final stage changed anything except its absent A", row, got)
		}
	}
	tlsCost, a := providerHistoryPlannedTLSControlUpper(32, providerTLSBounds{2, 4, 8192})
	calls, b := providerHistoryCollectionRoleResultControlCallsBound(0, 0, 2, 1, false)
	good, c := calls.successClosure.upper(tlsCost)
	failed, d := calls.failedClosure.upper(tlsCost)
	if !a || !b || !c || !d || good != (candidateWorkVector{2: 2623768, 10: 89589065842}) || failed != (candidateWorkVector{2: 2743920, 10: 90446824118}) ||
		failed[2]-good[2] != 120152 || failed[10]-good[10] != 857758276 {
		t.Fatal("five original failure contexts can exceed successful closure despite fewer TLS records", good, failed)
	}
}

func TestProviderHistoryRoleResultControlRejectsInvalidAndLatePartialQuotes(t *testing.T) {
	for _, members := range []uint64{0, 33, ^uint64(0)} {
		if got, ok := providerHistoryPlannedTLSControlUpper(members, providerTLSBounds{1, 1, 8192}); ok || got != (providerHistoryPlannedTLSControlCost{}) {
			t.Fatal("invalid original member domain returned partial TLS parts")
		}
	}
	for _, bounds := range []providerTLSBounds{{}, {9, 1, 1}, {1, 17, 1}, {1, 1, 65537}, {-1, 1, 1}, {1, 1, 0}} {
		if got, ok := providerHistoryCollectionRoleResultControlPartsUpper(0, 1, bounds, false); ok || got != (providerHistoryRoleResultControlParts{}) {
			t.Fatal("invalid selected TLS profile leaked earlier role work")
		}
	}
	for _, row := range [][4]uint64{{349525, 349525, 1, 0}, {^uint64(0), ^uint64(0), 1, 0}, {0, 1, 1, 0}, {1, 0, 1, 0}, {1, 33, 1, 0}, {0, 0, 0, 0}, {0, 0, 33, 0}, {0, 0, 32, 32}} {
		if got, ok := providerHistoryCollectionRoleResultControlCallsBound(row[0], row[1], row[2], row[3], false); ok || got != (providerHistoryRoleResultControlCounts{}) {
			t.Fatal("invalid original stage/past/member/ordinal returned partial counts", row)
		}
	}
	for _, row := range [][2]uint64{{^uint64(0), 32}, {0, 0}, {0, 33}} {
		if got, ok := providerHistoryCollectionRoleResultControlPartsUpper(row[0], row[1], providerTLSBounds{1, 1, 8192}, false); ok || got != (providerHistoryRoleResultControlParts{}) {
			t.Fatal("invalid selected role domain leaked positive TLS/context parts", row)
		}
	}
	if got, ok := providerHistoryCollectionRoleResultControlPartsUpper(349524, 32, providerTLSBounds{1, 1, 8192}, true); ok || got != (providerHistoryRoleResultControlParts{}) {
		t.Fatal("out-of-domain own advance returned a partial full role")
	}
	if _, ok := providerHistoryCollectionRoleResultControlPartsUpper(349524, 32, providerTLSBounds{1, 1, 8192}, false); !ok {
		t.Fatal("last legal original stage without own advancement")
	}
	tlsCost, ok := providerHistoryPlannedTLSControlUpper(32, providerTLSBounds{1, 1, 8192})
	if !ok {
		t.Fatal("finite late-refusal precursor")
	}
	late := tlsCost
	late.context.normal[10], late.context.envelope[10] = ^uint64(0), ^uint64(0)
	if got, ok := (providerHistoryRoleResultControlCalls{1, 1, 2}).upper(late); ok || got != (candidateWorkVector{}) {
		t.Fatal("late context multiplication overflow leaked completed TLS work", got)
	}
	late = tlsCost
	for _, pair := range []*providerHistoryControlPair{&late.original, &late.parsed} {
		pair.normal[10], pair.envelope[10] = ^uint64(0)/2+1, ^uint64(0)/2+1
	}
	if got, ok := (providerHistoryRoleResultControlCalls{1, 1, 0}).upper(late); ok || got != (candidateWorkVector{}) {
		t.Fatal("two individually finite TLS parts overflowed without whole-zero refusal", got)
	}
}

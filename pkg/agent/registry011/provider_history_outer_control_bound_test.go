package registry011

import "testing"

func TestProviderHistoryOuterControlMatchesIndependentRows(t *testing.T) {
	for _, row := range []struct {
		metadata, objectBytes, objects                     uint64
		profile, waterBytes, waterParsed, predecessor      uint64
		frameBuilt, frameParsed, object, inventory, parsed uint64
	}{
		{8192, 8192, 1, 67117089, 134225954, 201351204, 268468360, 872546777, 939729376, 16429, 67141848, 67166559},
		{65536, 1048576, 2, 4295032865, 4362141730, 4429266980, 12952215688, 30467957209, 30535139808, 2097197, 71319946, 71344657},
		{1048576, 33554432, 8, 1099512676385, 1099579785250, 1099646910500, 3298605146248, 7696991461849, 7697058644448, 67108909, 201344438, 201369149},
		{4096, 4096, 2, 16781345, 83890210, 151015460, 117461128, 520196569, 587379168, 8237, 16802186, 16814609},
	} {
		got, ok := providerHistoryOuterControlUpper(row.metadata, row.objectBytes, row.objects)
		want := providerHistoryOuterControlCost{
			candidateWorkVector{10: row.profile}, candidateWorkVector{10: row.waterBytes}, candidateWorkVector{2: 8192, 10: row.waterParsed},
			candidateWorkVector{10: row.predecessor}, candidateWorkVector{10: row.object}, candidateWorkVector{10: row.inventory},
			candidateWorkVector{10: row.parsed}, candidateWorkVector{10: row.frameBuilt}, candidateWorkVector{2: 8192, 10: row.frameParsed},
		}
		if !ok || got != want {
			t.Fatal("independent outer profile/pin/frame/object/inventory row", row, got)
		}
	}
	for _, row := range []struct{ groups, requests, decoded, logical uint64 }{
		{1, 1, 229376, 834985939814}, {1, 32, 1753088, 3277393713430},
		{2, 3, 1196032, 3220349912960}, {2, 33, 2670592, 5583970339040},
		{3, 96, 7323648, 14246953197678},
	} {
		got, ok := providerHistoryCollectionRepeatedOuterControlUpper(row.groups, row.requests, 65536, 1<<20)
		if !ok || got != (candidateWorkVector{2: row.decoded, 10: row.logical}) {
			t.Fatal("independent disjoint group outer-control composition", row, got)
		}
	}
}

func TestProviderHistoryOuterControlFundsOriginalObjectsAndInventories(t *testing.T) {
	// Normal typed record/pack DATA only. Hashing an outer object does not
	// qualify its inner semantics, custody, original ACK or execution role.
	ticket, consumption, inputs, roots, certificates := historyResultDataFixture(t, 1)
	requestPack, err := newProviderHistoryOriginalPackObserved(ticket.table.stage, ticket.table.digest, "requests", 0, 1,
		[]providerHistoryOriginalInput{{"request", ticket.table.members[0].request.wire}}, 1<<20, nil)
	if err != nil {
		t.Fatal("normal original request-pack precursor", err)
	}
	resultPack, err := newProviderHistoryOriginalPackObserved(ticket.table.stage, ticket.table.digest, "results", 0, 1, inputs, 1<<20, nil)
	if err != nil {
		t.Fatal("normal original result-pack precursor", err)
	}
	result, err := parseProviderHistoryMemberOriginalsObserved(inputs[0].raw, inputs[1].raw, ticket, consumption, 0, roots, certificates, providerReplayUnitMeter())
	if err != nil || !lowerHash(result.digest) {
		t.Fatal("normal original member-result DATA precursor", err)
	}
	settlement, err := newProviderHistoryGroupSettlement(ticket, consumption, admissionTestDigest("c"), admissionTestDigest("d"), "complete", "", []string{result.digest})
	if err != nil {
		t.Fatal("normal typed S precursor", err)
	}
	for _, phase := range [][]providerHistoryOriginalInput{
		{{"requests", requestPack.raw}, {"ticket", ticket.raw}},
		{{"consumption", consumption.raw}},
		{{"results", resultPack.raw}, {"settlement", settlement.raw}},
	} {
		upper, ok := providerHistoryOuterControlUpper(65536, 1<<20, uint64(len(phase)))
		if !ok {
			t.Fatal("finite outer primitive quote")
		}
		objects := make([]providerHistoryStoredObject, len(phase))
		for i, raw := range phase {
			m := providerHistoryContentUnitMeter(upper.storedObject)
			objects[i], err = newProviderHistoryStoredObjectObserved(raw.kind, raw.raw, 1<<20, m)
			if err != nil || objects[i].raw != raw.raw || m.failed || m.actual[2] != 0 || m.actual[10] > upper.storedObject[10] {
				t.Fatal("opaque stored original changed bytes or exceeded quote", raw.kind, err)
			}
		}
		before := append([]providerHistoryStoredObject(nil), objects...)
		m := providerHistoryContentUnitMeter(upper.inventoryBuilt)
		inventory, err := newProviderHistoryInventoryObserved(admissionTestDigest("b"), objects, 8192, m)
		if err != nil || inventory == nil || inventory.count != len(objects) || m.failed || m.actual[2] != 0 || m.actual[10] > upper.inventoryBuilt[10] {
			t.Fatal("normal inventory reconstruction exceeded quote", err)
		}
		original := *inventory
		parsed := providerHistoryContentUnitMeter(upper.inventoryParsed)
		checked, err := parseProviderHistoryInventoryObserved([]byte(inventory.raw), inventory.previous, inventory.digest, objects, 8192, parsed)
		if err != nil || checked == nil || *checked != original || *inventory != original || parsed.failed || parsed.actual[2] != 0 || parsed.actual[10] > upper.inventoryParsed[10] {
			t.Fatal("exact original inventory parser exceeded quote or changed bytes", err)
		}
		for i := range objects {
			if objects[i] != before[i] {
				t.Fatal("inventory reconstruction changed caller's immutable original", i)
			}
		}
	}
}

func TestProviderHistoryOuterControlFundsRetainedProfilePinsAndFrames(t *testing.T) {
	// Keep the actual fixture's retained config/scope; these extra observers
	// inspect DATA and never replace its live meter or create another Source.
	config, scope := providerHistoryFrameFixture(t)
	upper, ok := providerHistoryOuterControlUpper(config.bounds.maximumMetadataPerCommit, 1<<20, 2)
	if !ok {
		t.Fatal("finite actual-config outer quote")
	}
	profile := providerHistoryContentUnitMeter(upper.profile)
	if providerIntegrationProfileObserved(config.bounds, config.capture, profile) != config.profile || profile.failed || profile.actual[2] != 0 || profile.actual[10] > upper.profile[10] {
		t.Fatal("original profile reconstruction exceeded its own cap", profile.actual)
	}
	before := scope.initial
	for index, phase := range []string{"admit", "consume", "settle"} {
		frame := providerHistoryFrameData(t, config, scope, before, phase, int64(index+1))
		original := *frame
		built := providerHistoryContentUnitMeter(upper.frameBuilt)
		got, err := newProviderHistoryCustodyFrameObserved(config, scope, before, 1, phase, frame.previousInventory,
			frame.inventory, frame.previousModel, frame.model, frame.operation, frame.clock, built)
		if err != nil || got == nil || *got != original || built.failed {
			t.Fatal("original-shaped outer frame constructor exceeded quote", phase, err)
		}
		parsed := providerHistoryContentUnitMeter(upper.frameParsed)
		checked, err := parseProviderHistoryCustodyFrameObserved(config, scope, before, frame.after, []byte(frame.intent), []byte(frame.commit), []byte(frame.frontier), parsed)
		if err != nil || checked == nil || *checked != original || *frame != original || parsed.failed {
			t.Fatal("original frame parser exceeded quote or changed retained data", phase, err)
		}
		water := providerHistoryContentUnitMeter(upper.highwaterBytes)
		raw, err := before.bytesObserved(config, water)
		if err != nil || len(raw) == 0 || water.failed {
			t.Fatal("original before-pin encoding exceeded quote", phase, err)
		}
		pin := providerHistoryContentUnitMeter(upper.highwaterParsed)
		h, err := parseProviderFrontierHighwaterObserved(raw, config, pin)
		if err != nil || h != before || pin.failed {
			t.Fatal("original before-pin parser exceeded quote", phase, err)
		}
		predecessor := providerHistoryContentUnitMeter(upper.predecessor)
		if !checkProviderHighwaterPredecessorObserved(before, frame.after, config, predecessor) || predecessor.failed {
			t.Fatal("predecessor's three independent profile rebuilds exceeded quote", phase)
		}
		for _, row := range []struct {
			meter *providerWorkMeter
			upper candidateWorkVector
		}{{built, upper.frameBuilt}, {parsed, upper.frameParsed}, {water, upper.highwaterBytes}, {pin, upper.highwaterParsed}, {predecessor, upper.predecessor}} {
			for c, actual := range row.meter.actual {
				if actual > row.upper[c] {
					t.Fatal("outer primitive escaped its coordinate", phase, c)
				}
			}
		}
		before = frame.after
	}
}

func TestProviderHistoryOuterControlTopologyKeepsMemberAnchors(t *testing.T) {
	for _, row := range []struct {
		groups, requests uint64
		want             providerHistoryOuterControlCalls
	}{
		{1, 1, providerHistoryOuterControlCalls{3, 22, 3, 19, 14, 22, 12, 19, 10, 6}},
		{1, 32, providerHistoryOuterControlCalls{3, 84, 3, 19, 14, 22, 12, 19, 10, 130}},
		{2, 3, providerHistoryOuterControlCalls{6, 88, 6, 80, 56, 86, 52, 80, 20, 58}},
	} {
		got, ok := providerHistoryCollectionOuterControlCallsBound(row.groups, row.requests)
		if !ok || got != row.want {
			t.Fatal("independent outer anchor counts", row, got)
		}
	}
	for _, sizes := range [][]uint64{{1}, {32}, {2, 1}, {1, 32}, {32, 32, 32}} {
		var expected providerHistoryOuterControlCalls
		var requests uint64
		for g, members := range sizes {
			requests += members
			expected.frameBuilds += 3
			expected.inventoryBuilds += 3
			for _, parses := range []uint64{10, 7, 2} {
				expected.frameParses += parses
				expected.inventoryParses += parses
			}
			// Direct outer objects:2 packs+3 records written; then original
			// T/C/S reads supply12 packs+19 records, before inventory rebuilds.
			expected.packObjects += 2 + 12
			expected.recordObjects += 3 + 19
			expected.packReads += 12
			expected.recordReads += 19
			expected.highwaterBytes += 3 + 7
			// Current C anchor: seal once, then once in each member role.
			for range 1 + 2*members {
				expected.frameParses++
				expected.highwaterParses += 2
			}
			for range g {
				for range 14 {
					expected.frameParses += 3
					expected.inventoryParses += 3
					expected.packObjects += 2
					expected.recordObjects += 3
					expected.packReads += 2
					expected.recordReads += 3
					expected.highwaterParses += 3
				}
			}
		}
		got, ok := providerHistoryCollectionOuterControlCallsBound(uint64(len(sizes)), requests)
		if !ok || got != expected {
			t.Fatal("source-enumerated original outer calls", sizes, got, expected)
		}
	}
}

func TestProviderHistoryOuterControlKeepsGenuinePreflightAndCodecPrefixes(t *testing.T) {
	config, scope := providerHistoryFrameFixture(t)
	upper, ok := providerHistoryOuterControlUpper(config.bounds.maximumMetadataPerCommit, 1<<20, 2)
	if !ok {
		t.Fatal("finite original-config refusal quote")
	}
	profile := providerHistoryContentUnitMeter(upper.profile)
	if providerIntegrationProfileObserved(config.bounds, config.capture, profile) != config.profile || profile.failed || profile.actual[10] == 0 {
		t.Fatal("normal first profile precursor")
	}
	frame := providerHistoryFrameData(t, config, scope, scope.initial, "admit", 1)
	original := *frame
	short := upper.frameBuilt
	// After the first profile's ACTUAL work, the second original profile
	// preflight is one unit short. Preflight itself consumes no actual units.
	short[10] = profile.actual[10] + config.bounds.maximumMetadataPerCommit*config.bounds.maximumMetadataPerCommit
	m := providerHistoryContentUnitMeter(short)
	got, err := newProviderHistoryCustodyFrameObserved(config, scope, frame.before, 1, frame.phase, frame.previousInventory,
		frame.inventory, frame.previousModel, frame.model, frame.operation, frame.clock, m)
	if err == nil || got != nil || !m.failed || m.actual[10] != profile.actual[10] || m.actual[2] != 0 || *frame != original || m.event(providerCopyEvent, 0) {
		t.Fatal("second profile preflight refunded first profile or returned frame", err, m.actual)
	}
	full := providerHistoryContentUnitMeter(upper.frameParsed)
	if f, err := parseProviderHistoryCustodyFrameObserved(config, scope, frame.before, frame.after, []byte(frame.intent), []byte(frame.commit), []byte(frame.frontier), full); err != nil || f == nil || full.failed || full.actual[2] == 0 {
		t.Fatal("normal frame intent codec precursor", err)
	}
	short = upper.frameParsed
	short[2] = uint64(len(frame.intent) - 1)
	m = providerHistoryContentUnitMeter(short)
	if !m.event(providerCopyEvent, 1) {
		t.Fatal("successful explicit caller prefix")
	}
	if f, err := parseProviderHistoryCustodyFrameObserved(config, scope, frame.before, frame.after, []byte(frame.intent), []byte(frame.commit), []byte(frame.frontier), m); err == nil || f != nil || !m.failed || m.actual != (candidateWorkVector{10: 2}) || *frame != original || m.event(providerCodecEvent, 0) {
		t.Fatal("intent codec shortage erased caller prefix or returned partial frame", err, m.actual)
	}
	objects := historyStorageObjects(t)
	invUpper, bounded := providerHistoryOuterControlUpper(65536, 1<<20, 2)
	if !bounded {
		t.Fatal("finite normal inventory refusal quote")
	}
	valid := providerHistoryContentUnitMeter(invUpper.inventoryBuilt)
	if inv, err := newProviderHistoryInventoryObserved(admissionTestDigest("b"), objects, 8192, valid); err != nil || inv == nil || valid.failed {
		t.Fatal("normal inventory preflight precursor", err)
	}
	before := append([]providerHistoryStoredObject(nil), objects...)
	// Independently count both actual raw copies/hashes and expected digest
	// comparisons; then make the ORIGINAL8192 canonical preflight one short.
	prefix := uint64(2*len(objects[0].raw) + 38 + 3 + 133 + 2*len(objects[1].raw) + 37 + 3 + 133)
	m = providerHistoryContentUnitMeter(candidateWorkVector{10: prefix + 8192*8192})
	if inv, err := newProviderHistoryInventoryObserved(admissionTestDigest("b"), objects, 8192, m); err == nil || inv != nil || !m.failed || m.actual != (candidateWorkVector{10: prefix}) || m.event(providerEncodeEvent, 0) {
		t.Fatal("inventory preflight erased complete object prefix or leaked inventory", err, m.actual)
	}
	for i := range objects {
		if objects[i] != before[i] {
			t.Fatal("inventory refusal changed original object", i)
		}
	}
}

func TestProviderHistoryOuterControlRefusesInvalidAndOverflowingQuotes(t *testing.T) {
	for _, row := range [][3]uint64{{0, 1, 1}, {(1 << 20) + 1, 1, 1}, {1, 0, 1}, {1, (32 << 20) + 1, 1}, {1, 1, 0}, {1, 1, 9}, {^uint64(0), ^uint64(0), ^uint64(0)}} {
		if got, ok := providerHistoryOuterControlUpper(row[0], row[1], row[2]); ok || got != (providerHistoryOuterControlCost{}) {
			t.Fatal("invalid outer primitive input returned partial data", row, got)
		}
	}
	for _, row := range [][2]uint64{{0, 0}, {1, 0}, {1, 33}, {2, 1}, {349526, 349526}, {^uint64(0), ^uint64(0)}} {
		if got, ok := providerHistoryCollectionOuterControlCallsBound(row[0], row[1]); ok || got != (providerHistoryOuterControlCalls{}) {
			t.Fatal("invalid outer call relation returned partial counts", row, got)
		}
		if got, ok := providerHistoryCollectionRepeatedOuterControlUpper(row[0], row[1], 65536, 1<<20); ok || got != (candidateWorkVector{}) {
			t.Fatal("invalid outer call relation returned aggregate cost", row, got)
		}
	}
	if got, ok := providerHistoryCollectionRepeatedOuterControlUpper(1, 1, 0, 1<<20); ok || got != (candidateWorkVector{}) {
		t.Fatal("invalid profile cap returned aggregate cost")
	}
	// Valid constituents, finite decoded42034008838553600, but logical
	// 89891331919486116558550 exceeds uint64: refuse the complete vector.
	if got, ok := providerHistoryCollectionRepeatedOuterControlUpper(349525, 11184800, 65536, 1<<20); ok || got != (candidateWorkVector{}) {
		t.Fatal("late outer logical overflow leaked earlier decoded row", got)
	}
}

package registry011

import "testing"

func TestProviderHistoryTransportRolesMatchIndependentClassRows(t *testing.T) {
	manifest, methods, certificates, _, _ := historyRoleSourceInputs(t)
	original, oldMethods, oldTLS := *manifest, methods, certificates
	got, ok := providerHistoryCollectionTransportRoleUpper(manifest, methods, certificates, 512<<10, 8192, 4096)
	if !ok || got.grammar.classes != ([8]uint64{28, 54, 91, 26, 13, 13, 0, 17}) {
		t.Fatal("original active grammar precursor", ok)
	}
	// Exact standalone integer oracles, not calls to another cost function.
	rows := []struct {
		class                                int
		request, wire, raw, capture, failure uint64
	}{
		{0, 16383, 16384, 32813, 268488996, 1353780472},
		{1, 16384, 32768, 49197, 1073778980, 1354124565},
		{2, 16384, 131072, 147501, 17179808036, 1356188949},
		{3, 16384, 131072, 147501, 17179808036, 1356188949},
		{4, 16384, 131072, 147501, 17179808036, 1356188949},
		{5, 16384, 131072, 147501, 17179808036, 1356188949},
		{7, 16384, 262144, 278573, 68719284516, 1358941461},
	}
	for _, row := range rows {
		if got.requestUpper[row.class] != row.request || got.capture[row.class] != (candidateWorkVector{1: row.wire + 1, 2: row.wire - 1, 10: row.capture}) ||
			got.failure[row.class].body.minimumObjectBytes != row.raw || got.failure[row.class].extra.work != (candidateWorkVector{2: 32776, 9: row.raw, 10: row.failure}) {
			t.Fatal("selected class escaped its independent local BYTE envelope", row.class)
		}
	}
	if got.member != (candidateWorkVector{0: 1, 1: 262145, 2: 262143, 10: 68719284516}) ||
		got.failed != (providerHistoryStageWorkCost{work: candidateWorkVector{2: 32776, 9: 278573, 10: 1358941461}}) ||
		got.failedStorage != (providerHistoryStorageCharge{282669, 278573, 262145, 16428, 1, 1}) {
		t.Fatal("one member or terminal failure was multiplied by the collection/group count")
	}
	if got.capture[6] != (candidateWorkVector{}) || got.failure[6] != (providerHistoryFailedSettlementExtraCost{}) || got.requestUpper[6] != 0 ||
		*manifest != original || methods != oldMethods || certificates != oldTLS {
		t.Fatal("unused class gained an allowance or original inputs changed")
	}
	// Exact original failure-layout boundary: full fixed header, framing,
	// and the already permitted extra wire byte. This selects numeric DATA;
	// no payload or owned observer is constructed or enlarged.
	if exact, valid := providerHistoryCollectionTransportRoleUpper(manifest, methods, certificates, 278573, 8192, 4096); !valid || exact.member != got.member {
		t.Fatal("exact funded failure-layout boundary was refused")
	}
	if short, valid := providerHistoryCollectionTransportRoleUpper(manifest, methods, certificates, 278572, 8192, 4096); valid || short != (providerHistoryTransportRoleCost{}) {
		t.Fatal("one missing failure-layout byte was silently funded")
	}
}

func TestProviderHistoryTransportRolesFollowOptionalTracerSelection(t *testing.T) {
	// An explicitly larger UNUSED gas class cannot prevent this root-call
	// quote by requiring a 16MiB failed object. It remains a valid original
	// method/global selection; no owned source or allowance is modified.
	in := historyCollectionInput(t)
	in.capture.WireBytes = 16 << 20
	manifest := historyCollectionManifest(t, in)
	b, _ := providerFrontierBoundsFixture()
	methods := *b.methods
	methods.gas.wire = 16 << 20
	got, ok := providerHistoryCollectionTransportRoleUpper(manifest, methods, *b.tls, 512<<10, 8192, 4096)
	if !ok || got.grammar.classes[6] != 0 || got.failure[6] != (providerHistoryFailedSettlementExtraCost{}) || got.member[1] != 262145 {
		t.Fatal("absent tracer changed the selected local envelope", ok)
	}
	absentInput, absentMethods := in, methods
	absentInput.native, absentInput.effects = 0, 0
	absentMethods.execution.wire = 16 << 20
	absent := historyCollectionManifest(t, absentInput)
	absentCost, absentOK := providerHistoryCollectionTransportRoleUpper(absent, absentMethods, *b.tls, 512<<10, 8192, 4096)
	if !absentOK || absentCost.grammar.classes[6] != 0 || absentCost.grammar.classes[7] != 0 ||
		absentCost.capture[7] != (candidateWorkVector{}) || absentCost.failure[7] != (providerHistoryFailedSettlementExtraCost{}) ||
		absentCost.member != (candidateWorkVector{0: 1, 1: 131073, 2: 131071, 10: 17179808036}) {
		t.Fatal("absent native/effect calls retained an unrelated execution allowance", absentOK)
	}
	// Select stateGasTracer in a NEW prospective manifest. The same object
	// selection is now genuinely too small; do not fall back to execution.
	in.gas = SepoliaAmsterdamGasProfile
	state := historyCollectionManifest(t, in)
	if cost, valid := providerHistoryCollectionTransportRoleUpper(state, methods, *b.tls, 512<<10, 8192, 4096); valid || cost != (providerHistoryTransportRoleCost{}) {
		t.Fatal("active unfunded tracer silently substituted another class")
	}
	methods.gas.wire = 64 << 10
	stateCost, stateOK := providerHistoryCollectionTransportRoleUpper(state, methods, *b.tls, 512<<10, 8192, 4096)
	if !stateOK || stateCost.grammar.classes[6] != 13 || stateCost.grammar.classes[7] != 4 ||
		stateCost.capture[6] != (candidateWorkVector{1: 65537, 2: 65535, 10: 4294971684}) ||
		stateCost.failure[6].extra.work != (candidateWorkVector{2: 32776, 9: 81965, 10: 1354812693}) {
		t.Fatal("explicit state tracer did not keep its own original class", stateOK)
	}
}

func TestProviderHistoryTransportSourcePreservesOriginalRoleAndStorageOwnership(t *testing.T) {
	manifest, methods, certificates, _, limits := historyRoleSourceInputs(t)
	// This is a NEW offline scalar selection before any source/H0. The
	// existing 256KiB object cap cannot hold a 256KiB wire failure plus header.
	const maximum = 512 << 10
	limits.legacy.objectBytes = maximum
	original, originalLimits := *manifest, limits
	member := candidateWorkVector{0: 1, 1: 262145, 2: 262143, 10: 68719284516}
	failure := candidateWorkVector{2: 32776, 9: 278573, 10: 1358941461}
	for _, cut := range []struct {
		prior   uint64
		advance bool
	}{{0, false}, {0, true}, {3, true}, {39, false}} {
		got, ok := providerHistoryCollectionTransportSourceUpper(manifest, methods, certificates, maximum, limits, cut.prior, 32, cut.advance)
		if !ok {
			t.Fatal("one same-input local source composition", cut)
		}
		base, joined := got.source.joined, got.joined
		if joined.initial != base.initial || joined.before != base.before || joined.preparation != base.preparation || joined.advance != base.advance ||
			joined.member.clocks != base.member.clocks || joined.closure.clocks != base.closure.clocks {
			t.Fatal("transport reassigned initializer/P/advance or owned clock events", cut)
		}
		for c := range member {
			if joined.member.work[c]-base.member.work[c] != member[c] || joined.closure.work[c]-base.closure.work[c] != failure[c] {
				t.Fatal("local role contribution repeated or dropped", cut, c)
			}
		}
		if got.storage[4] != (providerHistoryStorageCharge{909357, 868397, 786433, 81964, 3, 10}) {
			t.Fatal("failed closure refunded successful S or repeated the failure allocation", cut)
		}
		for role := range got.storage {
			if role != 4 && got.storage[role] != got.source.files.storage[role] {
				t.Fatal("a terminal failure became a per-member or initializer allocation", role)
			}
		}
		if *manifest != original || limits != originalLimits || got.source.files.firstInitialization[10] != 2*got.source.files.first.work[10] {
			t.Fatal("original selections or existing firstReading initializer bridge changed")
		}
	}
}

func TestProviderHistoryTransportSourceRefusesUnfundedOrInvalidOriginalSelections(t *testing.T) {
	manifest, methods, certificates, maximum, limits := historyRoleSourceInputs(t)
	// Both the six-component quote and its limits are otherwise genuine
	// finite DATA. A positive prior quote cannot hide the missing failure room.
	if _, ok := providerHistoryCollectionRoleSourceUpper(manifest, methods, certificates, maximum, limits, 0, 1, false); !ok {
		t.Fatal("original source-component precursor")
	}
	if got, ok := providerHistoryCollectionTransportSourceUpper(manifest, methods, certificates, maximum, limits, 0, 1, false); ok || got != (providerHistoryTransportSourceCost{}) {
		t.Fatal("original insufficient failure-object cap was enlarged or ignored")
	}
	maximum, limits.legacy.objectBytes = 512<<10, 512<<10
	for _, mode := range []string{"nil", "metadata", "overhead", "physical-overflow", "maximum", "tls", "method", "first-read", "prior", "members", "final-advance"} {
		t.Run(mode, func(t *testing.T) {
			m, limits, methods, tls, cap := manifest, limits, methods, certificates, maximum
			prior, members, advance := uint64(0), uint64(32), false
			switch mode {
			case "nil":
				m = nil
			case "metadata":
				limits.legacy.metadata = 0
			case "overhead":
				limits.fileOverhead = 0
			case "physical-overflow":
				limits.fileOverhead = ^uint64(0)
			case "maximum":
				cap++
			case "tls":
				tls.chains = 0
			case "method":
				methods.scalar.body = 0
			case "first-read":
				limits.legacy.firstRaw++
			case "prior":
				prior = manifest.counts.groups
			case "members":
				members = 33
			case "final-advance":
				prior, advance = manifest.counts.groups-1, true
			}
			if got, ok := providerHistoryCollectionTransportSourceUpper(m, methods, tls, cap, limits, prior, members, advance); ok || got != (providerHistoryTransportSourceCost{}) {
				t.Fatal("child refusal leaked a partial source/capture/storage result", mode)
			}
		})
	}
	if got, ok := providerHistoryCollectionTransportRoleUpper(manifest, methods, certificates, maximum, 8192, ^uint64(0)); ok || got != (providerHistoryTransportRoleCost{}) {
		t.Fatal("overflowing failure storage leaked a partial active-class result")
	}
}

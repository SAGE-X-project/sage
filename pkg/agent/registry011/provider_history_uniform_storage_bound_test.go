package registry011

import "testing"

func TestProviderHistoryUniformStorageUsesEveryOriginalAdmissionMaximum(t *testing.T) {
	manifest, methods, certificates, _, _ := historyRoleSourceInputs(t)
	original, originalMethods, originalTLS := *manifest, methods, certificates
	got, ok := providerHistoryCollectionUniformStorageUpper(manifest, methods, certificates, 512<<10, 8192, 4096)
	if !ok || manifest.counts.groups != 40 || manifest.counts.requests != 242 || got.groups != 40 {
		t.Fatal("original prospective source selection", ok)
	}
	// Independent writer literals: T/C/S use26 files and5 objects;32 members
	// add64 spools/objects. The terminal addition is one distinct failed file.
	if got.successful.storage != (providerHistoryStorageCharge{12099584, 11730944, 9437184, 2293760, 69, 90}) ||
		got.terminal != (providerHistoryStorageCharge{282669, 278573, 262145, 16428, 1, 1}) ||
		got.stage != (providerHistoryStorageCharge{12382253, 12009517, 9699329, 2310188, 70, 91}) ||
		got.future != (providerHistoryStorageCharge{495290120, 480380680, 387973160, 92407520, 2800, 3640}) {
		t.Fatal("admitted-stage storage was replaced by actual members, one global failure or a maximum")
	}
	if *manifest != original || methods != originalMethods || certificates != originalTLS {
		t.Fatal("offline compilation changed original source selections")
	}
	// The source allocations do not depend on prior prefix size or advance.
	// Compare with the existing full role graph, not a second local formula.
	_, _, _, _, limits := historyRoleSourceInputs(t)
	limits.legacy.objectBytes = 512 << 10
	for _, cut := range []struct {
		prior   uint64
		advance bool
	}{{0, false}, {3, true}, {39, false}} {
		role, valid := providerHistoryCollectionTransportSourceUpper(manifest, methods, certificates, 512<<10, limits, cut.prior, 32, cut.advance)
		if !valid {
			t.Fatal("existing complete assigned storage precursor", cut)
		}
		maximum := providerHistoryGroupTotals{physical: got.stage.physical, retainedRaw: got.stage.raw,
			payload: got.stage.payload, metadata: got.stage.metadata, objects: got.stage.objects, files: got.stage.files}
		var sum providerHistoryStorageCharge
		for _, index := range []int{0, 1, 2, 4, 5} {
			var added bool
			sum, added = sum.add(role.storage[index], maximum)
			if !added {
				t.Fatal("source nonmember allocation exceeds the exact stage", cut, index)
			}
		}
		for range 32 {
			var added bool
			sum, added = sum.add(role.storage[3], maximum)
			if !added {
				t.Fatal("source full member storage exceeds the exact stage", cut)
			}
		}
		if sum != got.stage {
			t.Fatal("compiled storage differs from its actual I/B/P/M/C/A source components", cut)
		}
	}
}

func TestProviderHistoryUniformStoragePreservesInactiveFailureClasses(t *testing.T) {
	in := historyCollectionInput(t)
	in.native, in.effects = 0, 0
	manifest := historyCollectionManifest(t, in)
	bounds, _ := providerFrontierBoundsFixture()
	got, ok := providerHistoryCollectionUniformStorageUpper(manifest, *bounds.methods, *bounds.tls, 512<<10, 8192, 4096)
	if !ok || got.terminal != (providerHistoryStorageCharge{151597, 147501, 131073, 16428, 1, 1}) {
		t.Fatal("an inactive gas/execution class supplied a failure allowance", ok)
	}
	if got.successful.storage != (providerHistoryStorageCharge{12099584, 11730944, 9437184, 2293760, 69, 90}) {
		t.Fatal("absence of a tracer narrowed the unchanged global storage selection")
	}
}

func historyUniformStorageMaximum(value providerHistoryStorageCharge) providerHistoryGroupTotals {
	return providerHistoryGroupTotals{work: candidateWorkVector{0: 1, 1: 1, 10: 1}, physical: value.physical,
		retainedRaw: value.raw, payload: value.payload, metadata: value.metadata, objects: value.objects, files: value.files, clocks: 1}
}

func TestProviderHistoryUniformStorageChecksAllSeparateFutureCapacities(t *testing.T) {
	manifest, methods, certificates, _, _ := historyRoleSourceInputs(t)
	got, ok := providerHistoryCollectionUniformStorageUpper(manifest, methods, certificates, 512<<10, 8192, 4096)
	maximum := historyUniformStorageMaximum(got.future)
	if !ok || !maximum.valid() || !got.within(maximum) || (providerHistoryUniformStorageCost{}).within(maximum) {
		t.Fatal("numeric capacity precursor; no store or witness is constructed", ok)
	}
	for _, name := range []string{"physical", "raw", "payload", "metadata", "objects", "files", "invalid"} {
		t.Run(name, func(t *testing.T) {
			selected := maximum
			switch name {
			case "physical":
				selected.physical--
			case "raw":
				selected.retainedRaw--
			case "payload":
				selected.payload--
			case "metadata":
				selected.metadata--
			case "objects":
				selected.objects--
			case "files":
				selected.files--
			case "invalid":
				selected.clocks = 0
			}
			if got.within(selected) || maximum != historyUniformStorageMaximum(got.future) {
				t.Fatal("a different resource domain funded a shortage or the original changed")
			}
		})
	}
	// Pure integer DATA, not a larger live manifest or quota. A finite quote
	// cannot authorize a profile exceeding the unchanged absolute payload cap.
	large, valid := providerHistoryStorageRepeatedUpper(got.stage, 2000)
	if !valid || large.payload != 19398658000 || large.physical != 24764506000 || large.payload <= 16<<30 {
		t.Fatal("independent finite but infeasible quote", valid)
	}
	selected := historyUniformStorageMaximum(large)
	if selected.valid() || (providerHistoryUniformStorageCost{groups: 2000, future: large}).within(selected) {
		t.Fatal("arithmetic finiteness replaced whole-profile feasibility")
	}
}

func TestProviderHistoryUniformStorageRefusesPartialOrOverflowedQuotes(t *testing.T) {
	manifest, methods, certificates, _, _ := historyRoleSourceInputs(t)
	for _, name := range []string{"nil", "copied", "maximum", "failure-object", "metadata", "tls", "method", "overhead", "late-overflow"} {
		t.Run(name, func(t *testing.T) {
			m, selectedMethods, selectedTLS := manifest, methods, certificates
			maximum, metadata, overhead := uint64(512<<10), uint64(8192), uint64(4096)
			switch name {
			case "nil":
				m = nil
			case "copied":
				copied := *manifest
				m = &copied
			case "maximum":
				maximum = 0
			case "failure-object":
				maximum = 278572
			case "metadata":
				metadata = 0
			case "tls":
				selectedTLS.chains = 0
			case "method":
				selectedMethods.scalar.body = 0
			case "overhead":
				overhead = 0
			case "late-overflow":
				// One original stage still fits; repetition over40 cannot.
				overhead = ^uint64(0)/(91*40) + 1
			}
			if got, valid := providerHistoryCollectionUniformStorageUpper(m, selectedMethods, selectedTLS, maximum, metadata, overhead); valid || got != (providerHistoryUniformStorageCost{}) {
				t.Fatal("a child refusal or late scalar overflow returned a partial quote")
			}
		})
	}
	for _, groups := range []uint64{0, (1<<20)/3 + 1} {
		if got, ok := providerHistoryStorageRepeatedUpper(providerHistoryStorageCharge{1, 1, 1, 1, 1, 1}, groups); ok || got != (providerHistoryStorageCharge{}) {
			t.Fatal("invalid original group count returned storage")
		}
	}
}

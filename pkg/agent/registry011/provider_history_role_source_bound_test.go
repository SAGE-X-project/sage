package registry011

import "testing"

func historyRoleSourceInputs(t *testing.T) (*providerHistoryCollectionManifest, providerMethodBounds, providerTLSBounds, uint64, providerHistoryRoleSourceLimits) {
	t.Helper()
	manifest, methods, certificates, maximum := historyRoleComponentInputs(t)
	limits := providerHistoryRoleSourceLimits{
		legacy: providerHistoryLegacyControlLimits{metadata: 8192, objectBytes: maximum,
			firstObjects: manifest.input.firstRead.objects, firstRaw: manifest.input.firstRead.raw, entries: 32, attempts: 1, chunks: 1},
		catalogEntries: 4096, catalogNameBytes: 1044480, directoryBytes: 1024, fileOverhead: 4096,
	}
	return manifest, methods, certificates, maximum, limits
}

func TestProviderHistoryRoleSourcePreservesOneOriginalSelection(t *testing.T) {
	manifest, methods, certificates, maximum, limits := historyRoleSourceInputs(t)
	original, oldMethods, oldCertificates, oldLimits := *manifest, methods, certificates, limits
	for _, cut := range []struct {
		prior   uint64
		advance bool
	}{{0, false}, {0, true}, {3, true}, {39, false}} {
		got, ok := providerHistoryCollectionRoleSourceUpper(manifest, methods, certificates, maximum, limits, cut.prior, 32, cut.advance)
		if !ok || got.files.first.objects != 64 || got.files.first.raw != 1048576 || got.retained.capacity != 4288 ||
			got.names.initialPerCatalog != 958 || got.raw.writers != ([2]uint64{524315, 393230}) || got.outer.joined.member != got.outer.joined.preparation {
			t.Fatal("one genuine manifest and its explicit original selections did not reach every child", cut, ok)
		}
		if *manifest != original || methods != oldMethods || certificates != oldCertificates || limits != oldLimits {
			t.Fatal("numeric composition changed original inputs")
		}
	}
}

func TestProviderHistoryRoleSourceKeepsInitializerAndAllocationOwnership(t *testing.T) {
	manifest, methods, certificates, maximum, limits := historyRoleSourceInputs(t)
	got, ok := providerHistoryCollectionRoleSourceUpper(manifest, methods, certificates, maximum, limits, 3, 32, true)
	if !ok {
		t.Fatal("ordinary selected source-component DATA")
	}
	// Independent source initial extras: two legacy frontier controls, eight
	// original reads/two identities, three indexed names and checkpoint copies.
	// The constructor's existing two complete firstRead rows are NOT in this
	// difference. No native type-size value appears in these initial extras.
	want := candidateWorkVector{2: 16384, 10: 8724406694}
	for coordinate := range want {
		if got.joined.initial.work[coordinate]-got.assigned.joined.initial.work[coordinate] != want[coordinate] ||
			got.files.firstInitialization[coordinate] != 2*got.files.first.work[coordinate] {
			t.Fatal("existing firstRead initializer bridge repeated or another original initial row omitted", coordinate)
		}
	}
	if got.joined.before.work[9] != 393216 || got.joined.member.work[9] != 327680 || got.joined.closure.work[9] != 327680 ||
		got.files.storage[1].physical != 462848 || got.files.storage[3].physical != 335872 || got.files.storage[4].physical != 364544 {
		t.Fatal("successful writer work/resources duplicated or changed during source composition")
	}
	joined := [6]providerHistoryStageWorkCost{got.joined.initial, got.joined.before, got.joined.preparation, got.joined.member, got.joined.closure, got.joined.advance}
	assigned := [6]providerHistoryStageWorkCost{got.assigned.joined.initial, got.assigned.joined.before, got.assigned.joined.preparation, got.assigned.joined.member, got.assigned.joined.closure, got.assigned.joined.advance}
	for role := range joined {
		if joined[role].clocks != assigned[role].clocks {
			t.Fatal("logical copy/codec source rows invented another native sample", role)
		}
	}
	if got.joined.member.work[10] < got.joined.preparation.work[10] {
		t.Fatal("same-role sum dropped entered P from full M")
	}
}

func TestProviderHistoryRoleSourceRefusesCrossSourceAndPartialResults(t *testing.T) {
	manifest, methods, certificates, maximum, limits := historyRoleSourceInputs(t)
	badNames, badDirectory, badCapacity, badOverhead, badChunks, badRaw, badSetup, overflow := limits, limits, limits, limits, limits, limits, limits, limits
	badNames.catalogNameBytes--
	badDirectory.directoryBytes = 1
	badCapacity.catalogEntries, badCapacity.catalogNameBytes = 1024, 261120
	badOverhead.fileOverhead = 0
	badChunks.legacy.chunks = ^uint64(0)
	badRaw.legacy.firstRaw++
	badSetup.setupRecoveries = 65
	overflow.catalogNameBytes = ^uint64(0)
	for _, row := range []struct {
		manifest       *providerHistoryCollectionManifest
		certificates   providerTLSBounds
		maximum        uint64
		limits         providerHistoryRoleSourceLimits
		prior, members uint64
		advance        bool
	}{
		{nil, certificates, maximum, limits, 0, 32, false},
		{manifest, certificates, maximum + 1, limits, 0, 32, false},
		{manifest, certificates, maximum, badNames, 0, 32, false},
		{manifest, certificates, maximum, badDirectory, 0, 32, false},
		{manifest, certificates, maximum, badCapacity, 0, 32, false},
		{manifest, certificates, maximum, badOverhead, 0, 32, false},
		{manifest, certificates, maximum, badChunks, 0, 32, false},
		{manifest, certificates, maximum, badRaw, 0, 32, false},
		{manifest, certificates, maximum, badSetup, 0, 32, false},
		{manifest, certificates, maximum, overflow, 0, 32, false},
		{manifest, providerTLSBounds{}, maximum, limits, 0, 32, false},
		{manifest, certificates, maximum, limits, 0, 0, false},
		{manifest, certificates, maximum, limits, 0, 33, false},
		{manifest, certificates, maximum, limits, manifest.counts.groups, 32, false},
		{manifest, certificates, maximum, limits, ^uint64(0), 32, true},
		{manifest, certificates, maximum, limits, manifest.counts.groups - 1, 32, true},
	} {
		got, ok := providerHistoryCollectionRoleSourceUpper(row.manifest, methods, row.certificates, row.maximum, row.limits, row.prior, row.members, row.advance)
		if ok || got != (providerHistoryRoleSourceCost{}) {
			t.Fatal("cross-source field, local child failure or late arithmetic leaked another partial component", row.prior, row.members, ok)
		}
	}
}

func TestProviderHistoryRoleSourceRetainsOriginalChildRefusals(t *testing.T) {
	_, methods, certificates, maximum, limits := historyRoleSourceInputs(t)
	in := historyCollectionInput(t)
	in.firstRead.work = candidateWorkVector{2: 1, 10: 1}
	underpriced := historyCollectionManifest(t, in)
	if got, ok := providerHistoryCollectionRoleSourceUpper(underpriced, methods, certificates, maximum, limits, 0, 1, false); ok || got != (providerHistoryRoleSourceCost{}) {
		t.Fatal("joined source components replaced a deficient original firstReading sublimit")
	}
	// Choose a syntactically valid tiny offline capture before construction.
	// Its original mechanical child cannot represent a complete four-byte
	// value. No source/owner/meter is patched or retried after failure.
	tinyInput := historyCollectionInput(t)
	tinyInput.capture.BodyBytes = 1
	tiny := historyCollectionManifest(t, tinyInput)
	limit := providerMethodLimit{1, 16 << 10}
	tinyMethods := providerMethodBounds{limit, limit, limit, limit, limit, limit, limit, limit}
	if !tinyMethods.fits(tiny.input.capture) {
		t.Fatal("valid tiny method/capture precursor")
	}
	if got, ok := providerHistoryCollectionRoleSourceUpper(tiny, tinyMethods, certificates, maximum, limits, 0, 1, false); ok || got != (providerHistoryRoleSourceCost{}) {
		t.Fatal("unpriceable original child was replaced by an earlier positive source component")
	}
}

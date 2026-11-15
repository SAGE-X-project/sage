package registry011

import (
	"bytes"
	"reflect"
	"testing"
)

func historyLegacyControlLimits() providerHistoryLegacyControlLimits {
	return providerHistoryLegacyControlLimits{65536, 1 << 20, 64, 64 << 20, 256, 4, 32}
}

func TestProviderHistoryLegacyControlMatchesIndependentRows(t *testing.T) {
	for _, row := range []struct {
		limits providerHistoryLegacyControlLimits
		want   [9][2]uint64
	}{
		{providerHistoryLegacyControlLimits{4096, 4096, 4, 65536, 1, 1, 1}, [9][2]uint64{
			{4096, 16785455}, {65536, 4295098404}, {0, 4295098404}, {540672, 38722506039},
			{0, 9111}, {0, 212}, {4096, 4311810123}, {20480, 2199107179213}, {606208, 39813463517},
		}},
		{historyLegacyControlLimits(), [9][2]uint64{
			{1048576, 1099513724975}, {65536, 4295194029}, {0, 4295098404}, {67633152, 70407534531009},
			{0, 2109171}, {0, 3392}, {65536, 8590245963}, {327680, 2220498682573}, {1925120, 1050864366231},
		}},
		{providerHistoryLegacyControlLimits{1 << 20, 32 << 20, 1 << 20, 64 << 20, 256, 64, 1024}, [9][2]uint64{
			{33554432, 1125899973951535}, {65536, 4295194029}, {0, 4295098404}, {67633152, 2251838700833953},
			{0, 67298483}, {0, 54272}, {65536, 8591229003}, {4259840, 6601373188813}, {43540480, 7196694811115687},
		}},
	} {
		got, ok := providerHistoryLegacyControlUpper(row.limits)
		values := [...]candidateWorkVector{got.object, got.inventoryParsed, got.emptyInventory, got.firstLoad,
			got.firstLookup, got.firstUnique, got.modelControl, got.firstCustody, got.catalog}
		if !ok {
			t.Fatal("finite independent legacy-control row", row.limits)
		}
		for i, value := range values {
			if value != (candidateWorkVector{2: row.want[i][0], 10: row.want[i][1]}) {
				t.Fatal("independent legacy primitive row", row.limits, i, value)
			}
		}
	}
	for _, row := range []struct{ groups, requests, decoded, logical uint64 }{
		{1, 1, 1010122752, 1048439013903752}, {1, 32, 1964982272, 1569667739554328},
		{2, 3, 2054897664, 2115793586399662}, {2, 33, 2978955264, 2620208482190542},
		{3, 96, 5902647296, 4713206676127908},
	} {
		got, ok := providerHistoryCollectionRepeatedLegacyControlUpper(row.groups, row.requests, historyLegacyControlLimits())
		if !ok || got != (candidateWorkVector{2: row.decoded, 10: row.logical}) {
			t.Fatal("independent full repeated legacy-control row", row, got)
		}
	}
}

func TestProviderHistoryLegacyControlFundsClosedObjectsAndPlainBytes(t *testing.T) {
	limits := historyLegacyControlLimits()
	upper, ok := providerHistoryLegacyControlUpper(limits)
	if !ok {
		t.Fatal("finite original-object quote")
	}
	kinds := []string{"request", "params", "response", "tls", "setup", "setup-commit", "guardian-origin", "guardian-admission-binding",
		"guardian-seal-binding", "scan-admission-binding-v5", "enrollment", "admission", "scan-admission-binding", "ticket", "consumption",
		"settlement", "ledger-node", "budget", "exchange", "wire-binding", "bootstrap", "pin-input", "pin", "ledger-page", "ledger-seal",
		"index-empty", "index-mutation", "verification-receipt", "frontier", "segment", "batch", "checkpoint"}
	var longestKind, longestDomain int
	for _, kind := range kinds {
		domain, known := providerIntegrationDomain(kind)
		longestKind, longestDomain = max(longestKind, len(kind)), max(longestDomain, len(domain))
		raw := []byte("hello")
		if domain != "" {
			raw = providerIntegrationJSON(map[string]string{"format": domain})
		}
		before := bytes.Clone(raw)
		m := providerHistoryContentUnitMeter(upper.object)
		got, err := newProviderIntegrationObjectObserved(kind, raw, limits.objectBytes, m)
		decoded := uint64(len(raw))
		if domain == "" {
			decoded = 0
		}
		if !known || err != nil || got.raw != string(before) || !bytes.Equal(raw, before) || !lowerHash(got.digest) ||
			m.failed || m.actual[2] != decoded || m.actual[10] > upper.object[10] {
			t.Fatal("closed legacy object escaped quote or changed raw bytes", kind, err)
		}
		if domain == "" && got.digest != "0x2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824" {
			t.Fatal("plain original content acquired an unrelated domain", kind)
		}
	}
	if len(kinds) != 32 || longestKind != 26 || longestDomain != 43 {
		t.Fatal("closed legacy kind/domain maximum changed", longestKind, longestDomain)
	}
}

func TestProviderHistoryLegacyControlParsesInventoryOnceAndRebuildsEmptyAnchor(t *testing.T) {
	limits := historyLegacyControlLimits()
	upper, ok := providerHistoryLegacyControlUpper(limits)
	if !ok {
		t.Fatal("finite inventory quote")
	}
	objects := make([]providerIntegrationObject, 0, 256)
	for i := range 256 {
		raw := providerIntegrationJSON(map[string]any{"format": "sage-provider-authority-setup-v1", "index": i})
		object, err := newProviderIntegrationObject("setup", raw, 4096)
		if err != nil {
			t.Fatal("normal finite object precursor", err)
		}
		objects = append(objects, object)
	}
	before := append([]providerIntegrationObject(nil), objects...)
	for _, count := range []int{1, 2, 256} {
		original, err := newProviderFrontierInventory(admissionTestDigest("a"), objects[:count], 256, 64<<20)
		if err != nil {
			t.Fatal("normal retained inventory precursor", err)
		}
		m := providerHistoryContentUnitMeter(upper.inventoryParsed)
		got, err := parseProviderFrontierInventoryObserved([]byte(original.raw), original.previous, 256, 1<<20, 64<<20, m)
		if err != nil || !reflect.DeepEqual(got, original) || m.failed || m.actual[2] != uint64(len(original.raw)) || m.actual[10] > upper.inventoryParsed[10] {
			t.Fatal("legacy parser acquired an extra codec or rebuilt object payloads", count, err)
		}
	}
	m := providerHistoryContentUnitMeter(upper.emptyInventory)
	empty, err := newProviderFrontierInventoryObserved("", nil, 256, 64<<20, m)
	if err != nil || empty == nil || empty.previous != "" || len(empty.entries) != 0 || m.failed || m.actual[2] != 0 || m.actual[10] > upper.emptyInventory[10] || !reflect.DeepEqual(before, objects) {
		t.Fatal("empty anchor needed a sort/codec or changed caller objects", err)
	}
}

func TestProviderHistoryLegacyControlFundsRetainedScanDataWithoutRestoringOwner(t *testing.T) {
	s := providerHistoryGroupStartFixture(t, true)
	first := s.history.firstAdmission
	if first == nil || first.scanPin.ordinal != 1 || first.attempt.ordinal != 1 {
		t.Fatal("normal selected first-inspection DATA precursor")
	}
	config, pin, clocks, held := s.config, s.frontierFiles.pin, s.clock.read, s.stored
	limits := historyLegacyControlLimits()
	limits.metadata = config.bounds.maximumMetadataPerCommit
	upper, ok := providerHistoryLegacyControlUpper(limits)
	if !ok {
		t.Fatal("retained-config finite scan quote")
	}
	m := providerHistoryContentUnitMeter(upper.catalog)
	x, err := newProviderScanCodec(config, m)
	if err != nil {
		t.Fatal("independent retained profile/guardian rebuild", err)
	}
	water, a := x.highwater(first.scanPin)
	head, b := x.head(first.scanPin)
	intent, c := x.attempt(first.attempt)
	attempt, d := x.parseAttempt(intent)
	after, commit, e := x.successor(providerScanHighwater{}, first.attempt)
	deadline, _, deadlineOK := x.deadline(first.attempt.clock)
	if a != nil || b != nil || c != nil || d != nil || e != nil || !deadlineOK || deadline != first.attempt.deadline ||
		attempt != first.attempt || after != first.scanPin || len(water) == 0 || len(head) == 0 || len(commit) == 0 || m.failed {
		t.Fatal("original scan encode/decode/successor exceeded quote", a, b, c, d, e)
	}
	for coordinate, actual := range m.actual {
		if actual > upper.catalog[coordinate] {
			t.Fatal("retained scan DATA exceeded coordinate", coordinate)
		}
	}
	if s.config.profile != config.profile || s.frontierFiles.pin != pin || s.clock.read != clocks || s.stored != held || s.activeMeter.Load() != nil {
		t.Fatal("read-only scan DATA restored an owner or changed live state")
	}
}

func TestProviderHistoryLegacyControlFundsOriginalFrontierAndCommit(t *testing.T) {
	link, root, inventory, clock, bounds := providerFrontierIntentFixture(t)
	intent, err := providerFrontierIntent(link, root, inventory, "enroll", admissionTestDigest("a"), clock, bounds)
	if err != nil {
		t.Fatal("normal original intent precursor", err)
	}
	commit := providerFrontierCommit(intent)
	d := providerClockDiagnostic(clock)
	frontier := providerFrontierRecord{identity: link.identity, fence: link.fence, sequence: 1,
		commit: providerCandidateHash("sage-provider-custody-commit-v2", commit), setup: link.setup, objects: inventory.digest,
		enrollment: root.digest, indexes: [3]string{admissionTestDigest("b"), admissionTestDigest("c"), admissionTestDigest("d")},
		initial: d, clock: d, deadline: uint64(d.WallMS) + 300000}
	raw, err := frontier.bytes()
	if err != nil || len(commit) == 0 {
		t.Fatal("normal frontier/commit precursor", err)
	}
	upper, ok := providerHistoryLegacyControlUpper(historyLegacyControlLimits())
	if !ok {
		t.Fatal("finite main control quote")
	}
	m := providerHistoryContentUnitMeter(upper.modelControl)
	parsed, err := parseProviderFrontierRecordObserved(raw, link.identity, m)
	canonical, encodeErr := parsed.bytesObserved(m)
	if err != nil || encodeErr != nil || parsed != frontier || !providerReplayEqual(canonical, raw, m) || m.failed || m.actual[2] != uint64(len(raw)) {
		t.Fatal("main frontier parse/canonical comparison exceeded quote", err, encodeErr)
	}
	first := providerHistoryContentUnitMeter(upper.firstCustody)
	for range 2 {
		if providerFrontierIntentObjectObserved(intent, first) == nil || !providerReplayEqual(providerFrontierCommitObserved(intent, first), commit, first) {
			t.Fatal("original C/S direct-and-reconstructed intent parsing", first.actual)
		}
	}
	if first.failed || first.actual[2] != uint64(4*len(intent)) || first.actual[10] > upper.firstCustody[10] {
		t.Fatal("four original intent codecs/1MiB preflights exceeded quote", first.actual)
	}
}

func TestProviderHistoryLegacyControlRefusalPreservesActualPrefix(t *testing.T) {
	limits := historyLegacyControlLimits()
	upper, ok := providerHistoryLegacyControlUpper(limits)
	if !ok {
		t.Fatal("normal refusal quote")
	}
	object, err := newProviderIntegrationObject("setup", []byte(`{"format":"sage-provider-authority-setup-v1"}`), 4096)
	if err != nil {
		t.Fatal("normal original object precursor", err)
	}
	inv, err := newProviderFrontierInventory(admissionTestDigest("a"), []providerIntegrationObject{object}, 256, 64<<20)
	if err != nil {
		t.Fatal("normal inventory precursor", err)
	}
	short := upper.inventoryParsed
	short[2] = uint64(len(inv.raw) - 1)
	m := providerHistoryContentUnitMeter(short)
	if !m.event(providerCopyEvent, 1) {
		t.Fatal("successful caller prefix")
	}
	if got, err := parseProviderFrontierInventoryObserved([]byte(inv.raw), inv.previous, 256, 1<<20, 64<<20, m); err == nil || got != nil || !m.failed || m.actual != (candidateWorkVector{10: 2}) || m.event(providerCodecEvent, 0) {
		t.Fatal("atomic inventory codec shortage refunded prefix or returned data", err, m.actual)
	}
	link, root, inventory, clock, bounds := providerFrontierIntentFixture(t)
	intent, err := providerFrontierIntent(link, root, inventory, "enroll", admissionTestDigest("b"), clock, bounds)
	if err != nil {
		t.Fatal("normal commit precursor", err)
	}
	full := providerHistoryContentUnitMeter(upper.firstCustody)
	if raw := providerFrontierCommitObserved(intent, full); len(raw) == 0 || full.failed {
		t.Fatal("normal commit must reach its original encoder")
	}
	// The atomic intent codec and hash succeed. The original1MiB canonical
	// preflight is then one short; the failed check records no actual event.
	n := uint64(len(intent))
	prefix := 2 + n*n + 1 + n + uint64(len("sage-provider-custody-intent-v2")) + 2
	m = providerHistoryContentUnitMeter(candidateWorkVector{2: n, 10: prefix + (1<<20)*(1<<20)})
	if !m.event(providerCopyEvent, 1) {
		t.Fatal("successful explicit commit caller prefix")
	}
	if raw := providerFrontierCommitObserved(intent, m); raw != nil || !m.failed || m.actual != (candidateWorkVector{2: n, 10: prefix}) || m.event(providerEncodeEvent, 0) {
		t.Fatal("commit preflight refunded intent codec/hash or leaked partial output", m.actual)
	}
}

func TestProviderHistoryLegacyControlPreservesRepeatedReadersAndPageCaps(t *testing.T) {
	limits := historyLegacyControlLimits()
	for _, sizes := range [][]uint64{{1}, {32}, {2, 1}, {1, 32}, {32, 32, 32}} {
		initial, catalogs, requests := uint64(2), uint64(0), uint64(0)
		for g, members := range sizes {
			requests += members
			initial += 12 // planning2 + original T2/C3/dispatch2/S3
			catalogs += 1 + 1 + 2 + 16*members - 2 + 12
			if g+1 < len(sizes) {
				initial += 2
				catalogs += 2
			}
		}
		roots, rootOK := providerHistoryCollectionRootCallsBound(uint64(len(sizes)), requests)
		pages, pageOK := providerHistoryCollectionCatalogReadBound(uint64(len(sizes)), requests, limits.attempts, limits.chunks)
		if !rootOK || !pageOK || roots.initialModels != initial || pages.calls != catalogs {
			t.Fatal("source-enumerated initial-model/catalog calls collapsed", sizes, roots, pages)
		}
	}
	// The four actual inventory pages cap selected object references at1024;
	// increasing only the numeric first-read allowance cannot invent more pages.
	limits.firstObjects = 1024
	one, a := providerHistoryLegacyControlUpper(limits)
	limits.firstObjects = 1 << 20
	two, b := providerHistoryLegacyControlUpper(limits)
	if !a || !b || one != two {
		t.Fatal("four-page closure ignored its original256-entry ceiling")
	}
}

func TestProviderHistoryLegacyControlRejectsInvalidLimitsAndLateOverflow(t *testing.T) {
	base := historyLegacyControlLimits()
	for _, row := range []providerHistoryLegacyControlLimits{
		{0, 1, 1, 1, 1, 1, 1}, {(1 << 20) + 1, 1, 1, 1, 1, 1, 1}, {1, 0, 1, 1, 1, 1, 1},
		{1, (32 << 20) + 1, 1, 1, 1, 1, 1}, {1, 1, 0, 1, 1, 1, 1}, {1, 1, (1 << 20) + 1, 1, 1, 1, 1},
		{1, 1, 1, 0, 1, 1, 1}, {1, 1, 1, (64 << 20) + 1, 1, 1, 1}, {1, 1, 1, 1, 0, 1, 1},
		{1, 1, 1, 1, 257, 1, 1}, {1, 1, 1, 1, 1, 0, 1}, {1, 1, 1, 1, 1, 65, 1}, {1, 1, 1, 1, 1, 1, 0},
		{65536, 1 << 20, 64, 64 << 20, 256, 4, ^uint64(0)},
	} {
		if got, ok := providerHistoryLegacyControlUpper(row); ok || got != (providerHistoryLegacyControlCost{}) {
			t.Fatal("invalid or overflowing legacy quote returned partial vector", row, got)
		}
	}
	for _, row := range [][2]uint64{{0, 1}, {1, 0}, {1, 33}, {2, 1}, {(1<<20)/3 + 1, (1<<20)/3 + 1}, {349525, 11184800}} {
		// The final row has finite decoded688156179906560 but logical
		// 549372721301211542288, which cannot fit uint64. Entire result is zero.
		if got, ok := providerHistoryCollectionRepeatedLegacyControlUpper(row[0], row[1], base); ok || got != (candidateWorkVector{}) {
			t.Fatal("invalid relation or later logical overflow leaked partial decoded quote", row, got)
		}
	}
}

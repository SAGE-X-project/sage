package registry011

import (
	"bytes"
	"testing"
)

func TestProviderReplayCanonicalAndComparisonObserveActualInvocations(t *testing.T) {
	value := map[string]string{"a": "ordinary", "z": "1"}
	want := providerIntegrationJSON(value)
	m := providerReplayUnitMeter()
	if raw := providerReplayCanonical(value, 64<<10, m); !bytes.Equal(raw, want) || m.actual[2] != 0 || m.actual[10] != uint64(len(want)*len(want)+1) {
		t.Fatal("canonical encode changed bytes or counted its upper as actual")
	}
	before := m.actual[10]
	if !providerReplayEqual(want, want, m) || m.actual[10]-before != uint64(2*len(want)+1) {
		t.Fatal("comparison invocation omitted")
	}
	if providerReplayEqual(want, []byte("different ordinary data"), m) {
		t.Fatal("comparison ignored the actual bytes")
	}
	for _, maximum := range []uint64{0, (1 << 20) + 1, 1} {
		if providerReplayCanonical(value, maximum, nil) != nil {
			t.Fatal("unsupported or insufficient encoding bound")
		}
	}
	if providerReplayCanonical(make(chan int), 4096, nil) != nil {
		t.Fatal("unencodable canonical data")
	}
	zero := providerReplayUnitMeter()
	zero.upper = candidateWorkVector{}
	if providerReplayCanonical(value, 4096, zero) != nil || !zero.failed {
		t.Fatal("unfunded canonical encoder ran")
	}
	zero = providerReplayUnitMeter()
	zero.upper = candidateWorkVector{}
	if providerReplayEqual(want, want, zero) || !zero.failed {
		t.Fatal("unfunded comparison returned success")
	}
}

func TestProviderReplayFrontierRecordsPreserveIndependentDataAndMeter(t *testing.T) {
	u := providerFrontierUnitFixture(t)
	u.firstTicket(t, 0)
	before, err := parseProviderFrontierRecord(u.records[0].frontier, u.s.config.identity)
	if err != nil {
		t.Fatal(err)
	}
	after, err := parseProviderFrontierRecord(u.records[1].frontier, u.s.config.identity)
	if err != nil {
		t.Fatal(err)
	}
	m := providerReplayUnitMeter()
	parsed, err := parseProviderFrontierRecordObserved(u.records[0].frontier, u.s.config.identity, m)
	if err != nil || parsed != before || m.actual[2] != uint64(len(u.records[0].frontier)) {
		t.Fatal("observed frontier parse changed its data", err)
	}
	raw, err := parsed.bytesObserved(m)
	if err != nil || !bytes.Equal(raw, u.records[0].frontier) || !checkProviderFrontierAdvanceObserved(before, after, m) {
		t.Fatal("observed closed frontier relation changed", err)
	}
	zero := providerReplayUnitMeter()
	zero.upper = candidateWorkVector{}
	if _, err := parseProviderFrontierRecordObserved(raw, u.s.config.identity, zero); err == nil {
		t.Fatal("unfunded frontier decoder")
	}
	zero = providerReplayUnitMeter()
	zero.upper = candidateWorkVector{}
	if _, err := parsed.bytesObserved(zero); err == nil || checkProviderFrontierAdvanceObserved(before, after, zero) {
		t.Fatal("unfunded frontier encoding/relation")
	}
}

func TestProviderReplayIntentAndInventoryReproduceOriginalBytes(t *testing.T) {
	u := providerFrontierUnitFixture(t)
	record := u.records[0]
	id, bounds := u.s.config.identity, u.s.config.bounds
	link := providerFrontierLink{identity: id, fence: 1, commit: u.s.config.anchor, setup: u.s.setups[0].commit, objects: record.inventory.previous}
	intent := providerFrontierIntentObject(record.intent)
	clock, valid := providerStoredClock(intent["clock"])
	if !valid {
		t.Fatal("normal retained clock fixture")
	}
	m := providerReplayUnitMeter()
	raw, err := providerFrontierIntentDiagnosticObserved(link, u.root, record.inventory, "enroll", contentString(intent["operation"]), clock, bounds, m)
	if err != nil || !bytes.Equal(raw, record.intent) || !checkProviderFrontierIntentObserved(raw, link, u.root, record.inventory, "enroll", contentString(intent["operation"]), bounds, m) || !bytes.Equal(providerFrontierCommitObserved(raw, m), record.commit) {
		t.Fatal("observed intent/commit changed the retained DATA", err)
	}
	m = providerReplayUnitMeter()
	inventory, err := newProviderFrontierInventoryObserved(record.inventory.previous, record.fresh, bounds.maximumEntries, 64<<20, m)
	if err != nil || inventory.raw != record.inventory.raw || inventory.digest != record.inventory.digest || m.actual[2] != 0 {
		t.Fatal("inventory hash/encoding became a decode or changed bytes", err)
	}
	parsed, err := parseProviderFrontierInventoryObserved([]byte(inventory.raw), inventory.previous, bounds.maximumEntries, bounds.maximumObjectBytes, 64<<20, m)
	if err != nil || parsed.raw != inventory.raw || parsed.digest != inventory.digest || m.actual[2] != uint64(len(inventory.raw)) {
		t.Fatal("inventory decoder omitted an actual invocation", err)
	}
	zero := providerReplayUnitMeter()
	zero.upper = candidateWorkVector{}
	if checkProviderFrontierIntentObserved(raw, link, u.root, record.inventory, "enroll", contentString(intent["operation"]), bounds, zero) || providerFrontierCommitObserved(raw, zero) != nil {
		t.Fatal("unfunded intent/commit validation")
	}
	zero = providerReplayUnitMeter()
	zero.upper = candidateWorkVector{}
	if _, err := newProviderFrontierInventoryObserved(inventory.previous, record.fresh, bounds.maximumEntries, 64<<20, zero); err == nil {
		t.Fatal("unfunded original object hashes accepted an inventory")
	}
	zero = providerReplayUnitMeter()
	zero.upper[2] = uint64(len(inventory.raw) - 1)
	if _, err := parseProviderFrontierInventoryObserved([]byte(inventory.raw), inventory.previous, bounds.maximumEntries, bounds.maximumObjectBytes, 64<<20, zero); err == nil || !zero.failed {
		t.Fatal("one decoded byte short accepted an inventory")
	}
	// Exact finite hash funding is insufficient for the subsequent sort. The
	// comparator must refuse under the same meter before encoding a result.
	if len(record.fresh) < 2 {
		t.Fatal("normal fixture must contain multiple immutable objects")
	}
	var hashUnits uint64
	for _, object := range record.fresh {
		domain, _ := providerIntegrationDomain(object.kind)
		n := uint64(len(object.raw))
		if domain != "" {
			n += uint64(len(domain) + 1)
		}
		hashUnits += n + 1
	}
	shortSort := providerReplayUnitMeter()
	shortSort.upper[10] = hashUnits
	if _, err := newProviderFrontierInventoryObserved(inventory.previous, record.fresh, bounds.maximumEntries, 64<<20, shortSort); err == nil || !shortSort.failed || shortSort.actual[10] != hashUnits {
		t.Fatal("unfunded sort produced an inventory or hid prior hash calls")
	}
}

func TestProviderReplayObjectConstructionCountsOnlyItsActualDecode(t *testing.T) {
	u := providerFrontierUnitFixture(t)
	raw := u.root.CanonicalBytes()
	m := providerReplayUnitMeter()
	object, err := newProviderIntegrationObjectObserved("enrollment", raw, 64<<10, m)
	if err != nil || object.raw != string(raw) || object.digest != u.root.digest || m.actual[2] != uint64(len(raw)) {
		t.Fatal("object builder decoded zero or two times", err)
	}
	m = providerReplayUnitMeter()
	plain := []byte("ordinary inert request bytes")
	object, err = newProviderIntegrationObjectObserved("request", plain, 4096, m)
	want, _ := providerIntegrationObjectDigest("request", plain)
	if err != nil || object.digest != want || m.actual[2] != 0 {
		t.Fatal("plain content was decoded or received a domain", err)
	}
	zero := providerReplayUnitMeter()
	zero.upper[10] = uint64(len(raw)*len(raw) + 1)
	if _, err := newProviderIntegrationObjectObserved("enrollment", raw, 64<<10, zero); err == nil || !zero.failed {
		t.Fatal("unfunded hash/copy yielded a retained object")
	}
}

func TestProviderReplaySpentBodyAndLateDecodeCannotConstructData(t *testing.T) {
	s := providerScanStoreFixture(t)
	a := providerScanAttemptFixture(t, s, providerScanHighwater{})
	if err := s.scanControl.claim(a); err != nil {
		t.Fatal(err)
	}
	m := &providerWorkMeter{scan: s.scanControl, upper: a.body.work, maximumClocks: a.body.clocks}
	m.owner = m
	m.actual[10] = m.upper[10] // Finite, already spent arithmetic subject.
	s.activeMeter.Store(m)
	defer s.activeMeter.Store(nil)
	if _, err := newProviderFrontierScanReplay(s.scanControl, m); err == nil || !m.failed || s.last != nil {
		t.Fatal("spent body created independent replay state")
	}
	u := providerFrontierUnitFixture(t)
	for _, mode := range []string{"policy", "commit"} {
		r := *u.r
		r.meter = providerReplayUnitMeter()
		if mode == "policy" {
			r.meter.upper[2] = 0
		} else {
			r.meter.upper[2] = uint64(len(r.body.policy.raw))
		}
		if _, err := r.deriveFrontier(u.s, nil, nil, u.records[0].commit, providerFrontierRecord{}, CandidateClockDiagnostic{}, ""); err == nil || !r.meter.failed {
			t.Fatal("late decode failure returned a partial frontier", mode)
		}
	}
}

func TestProviderReplayInventoryKeyCopiesPrecedeComparison(t *testing.T) {
	left := providerIntegrationObjectRef{kind: "pin", digest: "ordinary"}
	right := providerIntegrationObjectRef{kind: "pin-input", digest: "ordinary"}
	m := providerReplayUnitMeter()
	a, aOK := providerReplayRefKey(left, m)
	b, bOK := providerReplayRefKey(right, m)
	if !aOK || !bOK || a != providerIntegrationRefKey(left) || b != providerIntegrationRefKey(right) || a < b || m.actual[10] != uint64(len(a)+len(b)+2) {
		t.Fatal("key copies changed prefix ordering or omitted work")
	}
	short := providerReplayUnitMeter()
	short.upper[10] = uint64(len(a))
	if key, ok := providerReplayRefKey(left, short); ok || key != "" || !short.failed || short.actual[10] != 0 {
		t.Fatal("one unit short created a temporary comparison key")
	}
	if key, ok := providerReplayRefKey(left, nil); !ok || key != a {
		t.Fatal("unobserved DATA key contract changed")
	}
}

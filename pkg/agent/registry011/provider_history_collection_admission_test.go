package registry011

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
)

// UNIT-only numeric inputs, fixed before profile/anchor/H0/clock. This is NOT
// a production source/native/RAM certificate. Existing fixtures default false.
func providerHistoryCollectionAdmissionTestBounds(t *testing.T, b *providerIntegrationBounds, capture ProviderCaptureLimits, nativeInputCounts, fullPack bool) {
	t.Helper()
	h := b.history
	if h == nil || h.collection == nil || h.collection.input.blocks != 1 || h.collection.counts.groups != 4 || h.collection.counts.requests != 28 {
		t.Fatal("explicit one-block absent UNIT grammar required before H0")
	}
	// This helper receives the explicit collection first-P UNIT selection
	// after its startup plan was derived and before profile/anchor/H0.
	// Header wire262, selected Wire32768 and n128KiB give the COMPLETE
	// declared startup ceiling1123*131072+(32768-262)=147226362 bytes.
	// The later history first-P class plan is independently recomputed and
	// must fit this unchanged pre-H0 ceiling, not refund its spare capacity.
	objectBytes := uint64(128 << 10)
	if fullPack {
		objectBytes = 4 << 20
	}
	if b.maximumObjectBytes != objectBytes || h.startup.work[2] != 147226362 || h.startup.work[2] > 256<<20 {
		t.Fatal("collection UNIT startup exceeds unchanged operation/segment decoded envelope")
	}
	u := historyMultistageTestUpper(capture)
	// Collection has four UNIT stages. Keep their raw/file allocations
	// inside the existing store AFTER the pre-I and first-P holds, before
	// selecting the profile/anchor/H0. Legacy two/three-stage inputs stay
	// separate. No live hold or operating/store cap is changed.
	u.physical = 6 << 20
	fileBytes, fileOK := providerUpperProduct(u.files, b.fileOverheadBytes)
	minimumPhysical, minimumOK := providerUpperSum(u.retainedRaw, fileBytes)
	if !fileOK || !minimumOK || minimumPhysical > u.physical {
		t.Fatal("collection UNIT physical row omitted raw/file allocation")
	}
	if nativeInputCounts {
		// Separate finite DATA-decoder UNIT inputs, fixed before profile/H0.
		// The shared checked sum below binds these counts in both the stage
		// and whole original future. Existing profiles keep their zero rows.
		u.execution[3], u.execution[4], u.execution[6] = 4, 16, 64
	}
	u.execution[0] = providerHistoryGroupMembers
	wire, ok := providerUpperProduct(providerHistoryGroupMembers, uint64(capture.WireBytes+1))
	if !ok {
		t.Fatal("checked member wire maximum")
	}
	u.execution[1] = wire
	work, a := u.work()
	initial, c := providerHistoryGroupInitializationForProfileUpper(*b)
	if !a || !c {
		t.Fatal("explicit UNIT initialization/stage rows")
	}
	for i, n := range work {
		value, p := providerUpperProduct(n, h.collection.counts.groups)
		value, q := providerUpperSum(initial[i], value)
		if !p || !q {
			t.Fatal("checked future UNIT work sum")
		}
		h.future.work[i] = value
	}
	h.future.work[0] = h.collection.counts.requests
	h.future.work[1], ok = providerUpperProduct(h.future.work[0], uint64(capture.WireBytes+1))
	if !ok {
		t.Fatal("checked whole UNIT wire")
	}
	for _, pair := range []struct {
		value uint64
		dst   *uint64
	}{{u.physical, &h.future.physical}, {u.retainedRaw, &h.future.retainedRaw}, {u.payload, &h.future.payload},
		{u.metadata, &h.future.metadata}, {u.objects, &h.future.objects}, {u.files, &h.future.files}} {
		*pair.dst, ok = providerUpperProduct(pair.value, h.collection.counts.groups)
		if !ok {
			t.Fatal("checked retained UNIT domain")
		}
	}
	physical, physicalOK := providerUpperSum(b.scan.maximumPreIRetainedBytes, h.startup.retained, h.future.physical)
	if !physicalOK || physical > b.maximumStoredBytes {
		t.Fatal("collection UNIT pre-I/startup/stages exceed the unchanged original store")
	}
	clocks, a := providerUpperProduct(u.clocks, h.collection.counts.groups)
	clocks, c = providerUpperSum(clocks, 2)
	if !a || !c {
		t.Fatal("checked UNIT native samples")
	}
	h.future.clocks, h.stages = clocks, h.collection.counts.groups
	h.records, h.commits = 3*h.stages, 3*h.stages
	h.derivation.dispatch.advance = candidateWorkVector{2: 128 << 20, 10: 1 << 56}
	h.derivation.dispatch.advanceClocks, h.derivation.dispatch.advanceTailMS = 2, 299999
	h.derivation.admission = providerHistoryCollectionAdmissionBounds{h.derivation, h.collection, u}
	h.derivation.snapshot = *h
	if !b.valid() {
		t.Fatal("separate UNIT profile exceeds original enclosing limits")
	}
}

func historyCollectionAdmissionFixture(t *testing.T, admission bool) *providerHistoryGroupOwner {
	t.Helper()
	s := historyCollectionAdmissionStore(t, admission, nil)
	o, err := newProviderHistoryGroupOwnerCandidate(s)
	if err != nil || o == nil {
		t.Fatal("genuine independent first-P models", err)
	}
	return o
}

// Optional numeric first-reader inputs are selected before the manifest,
// profile, anchor and H0. A failing group cannot rewrite its live allowance.
func historyCollectionAdmissionStore(t *testing.T, admission bool, firstRead *providerHistoryFirstPReadBounds) *providerCustodyStore {
	t.Helper()
	return historyCollectionAdmissionStoreWithPack(t, admission, firstRead, false)
}

// The larger aggregate result pack and original-reader maximum are NEW UNIT
// inputs fixed before manifest/profile/anchor/H0. The default stays unchanged.
func historyCollectionAdmissionStoreWithPack(t *testing.T, admission bool, firstRead *providerHistoryFirstPReadBounds, fullPack bool) *providerCustodyStore {
	t.Helper()
	in := historyCollectionInput(t)
	in.maximumSelected, in.blocks, in.native, in.effects, in.versions, in.presentStops = in.journal.Deployment.Number, 1, 0, 0, 0, 0
	in.windowBlocks, in.contentBlocks = 1, 1
	if firstRead != nil {
		in.firstRead = *firstRead
	}
	if fullPack {
		if firstRead != nil || !admission {
			t.Fatal("dedicated aggregate-pack reader selection requires the full original admission")
		}
		// A declared4MiB read must fit BEFORE that read, in addition to
		// already consumed original bytes. Do not substitute actual lengths.
		in.firstRead.raw = 8 << 20
	}
	m := historyCollectionManifest(t, in)
	s := providerHistoryProvisionedInspectionFixture(t, providerScanFixtureSelection{capture: &in.capture, pin: true, pinLargeObject: true, historyProgram: true,
		historyGroupWrite: true, historyGroupConsume: true, historyGroupDispatch: true, historyRecoveryNamespace: true, historyRecoveryLegacy: true,
		historyCollection: m, historyCollectionAdmission: admission, historyCollectionFirstP: true, historyCollectionFullPack: fullPack})
	first, err := newProviderFirstOperationCandidate(s)
	if err != nil {
		t.Fatal("genuine first operation", err)
	}
	h := in.journal.Deployment
	response := string(replayTestJSON(t, map[string]any{"jsonrpc": "2.0", "id": 1, "result": map[string]string{
		"hash": h.Hash, "parentHash": h.ParentHash, "number": "0x" + strconv.FormatUint(h.Number, 16), "timestamp": "0x" + strconv.FormatUint(h.Timestamp, 16)}}))
	conn := providerIntegrationOwnedTLS(t, providerFirstOperationRequestBody, response)
	if out, err := first.executeCandidateWithConnection(context.Background(), conn); err != nil || out == nil {
		t.Fatal("genuine original first P", err)
	}
	return s
}

func TestProviderHistoryCollectionAdmissionDerivesOriginalTableAndConsumption(t *testing.T) {
	o := historyCollectionAdmissionFixture(t, true)
	s, m, p := o.store, o.meter, o.parent
	held, upper, start, origin, clocks, actual := s.stored, m.upper, o.start, p.original, s.clock.read, m.actual
	// A finite cached count is a proposal only. Different original-role DATA
	// refuses before a window or sample; restoring this untouched UNIT input
	// does not rewind actual work or revive an entered failed operation.
	o.custodian.collection.phase = "close"
	if bad, err := o.retainCollectionAdmissionCandidate(context.Background()); err == nil || bad != nil ||
		m.actual != actual || s.clock.read != clocks || o.phase != "prepared" || o.window || o.failed || m.failed || s.activeMeter.Load() != nil {
		t.Fatal("different count proposals reached original observation", err)
	}
	o.custodian.collection.phase = "open"
	ack, err := o.retainCollectionAdmissionCandidate(context.Background())
	if err != nil || ack == nil || o.phase != "admitted" || ack.claim.purpose != "identity" || ack.claim.members != 18 ||
		ack.claim.upper.execution[0] != 18 || ack.claim.upper.execution[1] != 18*uint64(s.config.capture.WireBytes+1) ||
		s.clock.read != clocks+2 || s.stored != held || m.upper != upper || o.start != start || p.original != origin ||
		m.actual[0] != actual[0] || m.actual[1] != actual[1] || m.actual[2] <= actual[2] || o.coordinator == o.custodian ||
		o.coordinator.collection == o.custodian.collection || o.coordinator.collection.semantic == o.custodian.collection.semantic || s.activeMeter.Load() != nil {
		t.Fatal("collection T borrowed caller input, renewed its hold, or shared original roles", err)
	}
	table := &o.coordinator.ticket.table
	want := []string{"eth_chainId", "eth_getBlockByNumber", "eth_call", "web3_clientVersion", "eth_getBlockByNumber"}
	for i := range table.count {
		method := "eth_getCode"
		if i < len(want) {
			method = want[i]
		}
		if table.members[i].request.method != method {
			t.Fatal("independent closed request order", i)
		}
	}
	consumed, err := o.retainConsumptionCandidate(context.Background(), ack)
	if err != nil || consumed == nil || o.phase != "consumed" || o.order == nil || o.order.count != 18 || o.order.sent != 0 ||
		o.coordinator.collection.phase != "open" || o.custodian.collection.phase != "open" || o.coordinator.head != o.custodian.head ||
		s.stored != held || m.upper != upper || o.start != start || p.original != origin || m.actual[0] != actual[0] || m.actual[1] != actual[1] {
		t.Fatal("original C failed to preserve exact collection T and lifetime", err)
	}
}

func TestProviderHistoryCollectionAdmissionRefusesCallerTablesAndMissingPlanBeforeEffects(t *testing.T) {
	for _, planned := range []bool{false, true} {
		t.Run(strconv.FormatBool(planned), func(t *testing.T) {
			o := historyCollectionAdmissionFixture(t, planned)
			before, actual, clocks, held := providerHistoryProvisionedOriginalBytes(t, o.store.config), o.meter.actual, o.store.clock.read, o.store.stored
			upper := historyMultistageTestUpper(o.store.config.capture)
			if ack, err := o.retainAdmissionCandidate(context.Background(), "selection", []*providerIntegrationRequest{historyGroupSimple(t)}, upper); err == nil || ack != nil {
				t.Fatal("legacy caller table entered collection profile")
			}
			if !planned {
				if ack, err := o.retainCollectionAdmissionCandidate(context.Background()); err == nil || ack != nil {
					t.Fatal("manifest DATA minted its missing derivation")
				}
			} else {
				if ack, err := o.retainSelectedAdmissionCandidate(context.Background(), "identity", []*providerIntegrationRequest{historyGroupSimple(t)}, upper, true); err == nil || ack != nil {
					t.Fatal("caller table slipped through the internal collection selector")
				}
			}
			if o.meter.actual != actual || o.store.clock.read != clocks || o.store.stored != held || o.window || o.phase != "prepared" ||
				o.last != nil || o.failed || o.meter.failed || o.meter.closed || o.store.activeMeter.Load() != nil ||
				!reflect.DeepEqual(before, providerHistoryProvisionedOriginalBytes(t, o.store.config)) {
				t.Fatal("early refusal produced effects")
			}
		})
	}
}

func TestProviderHistoryCollectionAdmissionBindsThePreH0PlanInBothRoles(t *testing.T) {
	for _, mode := range []string{"owner", "manifest", "wire", "work", "parent-snapshot"} {
		t.Run(mode, func(t *testing.T) {
			o := historyCollectionAdmissionFixture(t, true)
			p, r := o.parent, o.parent.recovery
			before, clocks := o.meter.actual, o.store.clock.read
			switch mode {
			case "owner":
				p.bounds.derivation.admission.owner = nil
			case "manifest":
				copy := *p.bounds.collection
				copy.owner = &copy
				p.bounds.derivation.admission.manifest = &copy
			case "wire":
				p.bounds.derivation.admission.maximum.execution[1]--
			case "work":
				p.bounds.derivation.admission.maximum.custody[10]++
			case "parent-snapshot":
				p.collectionAdmission.maximum.execution[1]--
			}
			if p.matchesLocked() || r.observerShapeLocked(r.meter) {
				t.Fatal("post-H0 plan edit preserved an original role", mode)
			}
			if ack, err := o.retainCollectionAdmissionCandidate(context.Background()); err == nil || ack != nil || o.meter.actual != before || o.store.clock.read != clocks || o.failed || o.window {
				t.Fatal("changed selected plan reached observation", mode, err)
			}
		})
	}
}

func TestProviderHistoryCollectionAdmissionOriginalMismatchQuarantinesBeforeWritingT(t *testing.T) {
	o := historyCollectionAdmissionFixture(t, true)
	s, m := o.store, o.meter
	path := filepath.Join(s.config.custodyDirectory, "HEAD")
	raw, err := os.ReadFile(path)
	if err != nil || len(raw) == 0 {
		t.Fatal("owned original prerequisite", err)
	}
	raw[len(raw)-1] ^= 1
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	before, actual, clocks, held, pin := providerHistoryProvisionedOriginalBytes(t, s.config), m.actual, s.clock.read, s.stored, o.pin
	ack, err := o.retainCollectionAdmissionCandidate(context.Background())
	if err == nil || ack != nil || !o.failed || !m.failed || !m.closed || s.clock.read != clocks+1 || o.pin != pin || s.stored != held ||
		o.last != nil || o.admission != nil || o.order != nil || m.actual[0] != actual[0] || m.actual[1] != actual[1] || m.actual[10] <= actual[10] ||
		s.activeMeter.Load() != nil || !reflect.DeepEqual(before, providerHistoryProvisionedOriginalBytes(t, s.config)) {
		t.Fatal("original mismatch wrote/admitted a new T, dispatched, or refunded work", err)
	}
	for i, n := range actual {
		if m.actual[i] < n {
			t.Fatal("refunded original work", i)
		}
	}
	var decoded providerHistoryCollectionAdmissionBounds
	if json.Unmarshal([]byte(`{}`), &decoded) == nil {
		t.Fatal("JSON reconstructed the source plan")
	}
}

// The candidate upper is numeric DATA for this accounting predicate, not a new
// original table/permit. Use a genuine original owner and its unchanged meter;
// no post-H0 profile, actual counter or retained ledger is modified.
func TestProviderHistoryCollectionAdmissionAccountingIncludesOriginalInitialization(t *testing.T) {
	o := historyCollectionAdmissionFixture(t, true)
	s, m, p := o.store, o.meter, o.parent
	s.mu.Lock()
	defer s.mu.Unlock()
	m.mu.Lock()
	defer m.mu.Unlock()
	upper, ok := p.collectionAdmission.forMembers(p.bounds.derivation, p.bounds.collection, s.config.capture, 18)
	initial, initialOK := providerHistoryGroupInitializationForProfileUpper(s.config.bounds)
	legacy, legacyOK := providerHistoryGroupInitializationUpper(s.config.bounds)
	if !ok || !initialOK || !legacyOK || initial[10] <= legacy[10] || m.actual[10] > initial[10] || p.bounds.future.work[10] <= initial[10]+upper.execution[10] {
		t.Fatal("genuine original and finite complete initialization precursors")
	}
	before, clocks, held, maximum := m.actual, m.clocks, s.stored, m.upper
	upper.custody[10] = p.bounds.future.work[10] - initial[10] - upper.execution[10]
	work, valid := upper.work()
	if !valid || !o.admissionBudgetFitsLocked(upper, work) {
		t.Fatal("exact complete-initialization accounting refused")
	}
	upper.custody[10]++
	work, valid = upper.work()
	if !valid || legacy[10]+work[10] > p.bounds.future.work[10] || !candidateCapacity(0, m.actual[10], work[10], m.upper[10]) {
		t.Fatal("one-short must isolate the missing complete initialization, not actual/global exhaustion")
	}
	if o.admissionBudgetFitsLocked(upper, work) || m.actual != before || m.clocks != clocks || s.stored != held || m.upper != maximum || m.failed || o.failed {
		t.Fatal("original-reader initialization was borrowed/refunded or pure accounting changed the original lifetime")
	}
}

func TestProviderHistoryCollectionAdmissionFixturePreservesCompletePhysicalHold(t *testing.T) {
	o := historyCollectionAdmissionFixture(t, true)
	s, p, m := o.store, o.parent, o.meter
	b := s.config.bounds
	first := p.firstOwner
	if first == nil || first != s.firstOperation || first.state != providerFirstOperationDone ||
		b.maximumObjectBytes != 128<<10 || first.plan.reservation.work != (candidateWorkVector{0: 1, 1: 32768, 2: 103465298, 8: 1, 9: 817372, 10: 11727832672075}) ||
		p.bounds.startup.work[2] != 147226362 ||
		first.limits.operation[2] != 256<<20 || first.limits.segment[2] != 256<<20 {
		t.Fatal("genuine completed first-P precursor or unchanged decoded limits missing")
	}
	for coordinate, amount := range first.plan.reservation.work {
		// Startup was conservatively fixed before the history graph existed.
		// The actual typed-class plan must fit it, without refunding the hold.
		if amount > first.limits.operation[coordinate] || amount > first.limits.segment[coordinate] || amount > p.bounds.startup.work[coordinate] {
			t.Fatal("complete original startup reservation omitted a coordinate or changed its pre-H0 hold", coordinate)
		}
	}
	// Independent literal from the selected header's original C/S storage:
	// 311296+506076 raw plus32*4096 files =948444 startup bytes.
	// 32MiB pre-I +948444 startup +4*6MiB stages =59668700.
	physical, ok := providerUpperSum(b.scan.maximumPreIRetainedBytes, p.bounds.startup.retained, p.bounds.future.physical)
	if !ok || b.maximumStoredBytes != 64<<20 || b.scan.maximumPreIRetainedBytes != 32<<20 ||
		p.bounds.startup.retained != 948444 || p.bounds.future.physical != 24<<20 || physical != 59668700 ||
		physical > b.maximumStoredBytes || p.held.retained != physical || s.stored != physical ||
		p.collectionAdmission.maximum.physical != 6<<20 || p.collectionAdmission.maximum.retainedRaw != 4<<20 ||
		p.collectionAdmission.maximum.files != 256 || b.fileOverheadBytes != 4096 ||
		p.collectionAdmission.maximum.retainedRaw+p.collectionAdmission.maximum.files*b.fileOverheadBytes > p.collectionAdmission.maximum.physical ||
		m.upper != p.bounds.future.work || m.clocks != 2 || o.phase != "prepared" || s.activeMeter.Load() != nil || m.failed || o.failed {
		t.Fatal("original collection precursor overbooked/renewed its complete physical hold", physical, ok)
	}
}

func TestProviderHistoryCollectionFullPackFixturePreservesStartupAndWholeStorage(t *testing.T) {
	s := historyCollectionAdmissionStoreWithPack(t, true, nil, true)
	p, b := s.history, s.config.bounds
	first := p.firstOwner
	if first == nil || first.state != providerFirstOperationDone || b.maximumObjectBytes != 4<<20 || b.maximumStoredBytes != 64<<20 ||
		first.plan.reservation.work[2] != 103465298 || p.bounds.startup.work[2] != 147226362 || p.bounds.startup.retained != 948444 ||
		first.limits.operation[2] != 256<<20 || first.limits.segment[2] != 256<<20 ||
		p.collectionInput.firstRead.raw != 8<<20 || p.collectionInput.firstRead.objects != 64 ||
		p.collectionInput.firstRead.work != (candidateWorkVector{2: 64 << 20, 10: 1 << 52}) {
		t.Fatal("dedicated pre-H0 whole-pack/first-read/typed-startup precursor is incomplete")
	}
	for coordinate, amount := range first.plan.reservation.work {
		if amount > p.bounds.startup.work[coordinate] || amount > first.limits.operation[coordinate] || amount > first.limits.segment[coordinate] {
			t.Fatal("first-P enlarged its original declared/policy ceiling", coordinate)
		}
	}
	physical, ok := providerUpperSum(b.scan.maximumPreIRetainedBytes, p.bounds.startup.retained, p.bounds.future.physical)
	if !ok || physical != 59668700 || s.stored != physical || p.held.retained != physical || p.bounds.future.physical != 24<<20 ||
		p.collectionAdmission.maximum.physical != 6<<20 || p.collectionAdmission.maximum.retainedRaw != 4<<20 ||
		p.collectionAdmission.maximum.retainedRaw+p.collectionAdmission.maximum.files*b.fileOverheadBytes > p.collectionAdmission.maximum.physical {
		t.Fatal("full result-pack selection overbooked or changed original complete hold")
	}
	// Loose independent18-member guard maximum, retaining full selected wire
	// for2scalar/2header/14state-code and TLS2332/member without subtracting
	// request bytes:65579+1933312+41976=2040867. Actual code bodies alone are
	//304000, so the previous128KiB selection could not be this positive.
	const pack = uint64(65579 + 2*16384 + 2*32768 + 14*131072 + 18*2332)
	if pack != 2040867 || pack > b.maximumObjectBytes || pack <= 128<<10 {
		t.Fatal("independent prospective whole-pack bound")
	}
}

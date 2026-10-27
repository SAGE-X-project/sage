package registry011

import (
	"bytes"
	"testing"
)

func providerFrontierWriterFixture(t *testing.T) (*providerFrontierFiles, *providerFrontierUnit, *providerFrontierReplay) {
	t.Helper()
	_, policy := providerPrivateLedgerFixture(t, 8, 32)
	clock, err := newProviderOwnedClock(policy, 512, 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	f, u := providerFrontierFileFixtureWithClock(t, clock)
	if _, err := f.initialize(); err != nil {
		t.Fatal(err)
	}
	u.firstTicket(t, 0)
	r, err := newProviderFrontierReplay(f.store.config)
	if err != nil {
		t.Fatal(err)
	}
	f.store.frontierReplay = r
	return f, u, r
}

func providerFrontierNextFileHead(t *testing.T, f *providerFrontierFiles, record providerFrontierUnitRecord) providerFrontierHighwater {
	t.Helper()
	frontier, err := parseProviderFrontierRecord(record.frontier, f.store.config.identity)
	if err != nil {
		t.Fatal(err)
	}
	h := f.pin
	h.ordinal, h.sequence, h.previous = h.ordinal+1, frontier.sequence, providerCandidateHash("sage-provider-custody-highwater-v1", []byte(f.pinRaw))
	h.commit, h.frontier = frontier.commit, providerCandidateHash("sage-provider-custody-frontier-v1", record.frontier)
	return h
}

func TestProviderFrontierCustodyRetainsIndependentOriginalsBeforeMetadataSuccess(t *testing.T) {
	f, u, r := providerFrontierWriterFixture(t)
	s := f.store
	for _, record := range u.records {
		before := s.stored
		h := providerFrontierNextFileHead(t, f, record)
		cost, err := r.retain(s, record.inventory, record.fresh, record.intent, record.commit, record.frontier, h)
		if err != nil || s.failed || f.failed || r.failed || f.pending != nil || s.stored != before+cost.stored || f.pin != h || s.tip != h.commit || s.sequence != h.sequence || r.current.commit != h.commit {
			t.Fatal("complete normal candidate retention", err)
		}
		for _, object := range record.fresh {
			original, err := s.object(object.kind, object.digest)
			if err != nil || !bytes.Equal(original, []byte(object.raw)) {
				t.Fatal("metadata success lacked an exact original", err)
			}
		}
		if !s.exactFrontierMain(s.sequence, record.inventory, record.intent, record.commit) || !s.exactFrontierFile(providerCustodyName("frontier", s.sequence), record.frontier) || !f.readExact("HEAD", []byte(f.pinRaw)) {
			t.Fatal("metadata success preceded durable complete readback")
		}
	}
	if r.body.ledger == nil || r.body.ledger == u.ledger || r.indexes == u.indexes || r.body.ledger.current.privateMetadata <= u.ledger.current.privateMetadata || s.last != nil {
		t.Fatal("candidate writer trusted coordinator maps or minted an execution ACK")
	}
	last := u.records[len(u.records)-1]
	if _, err := r.retain(s, last.inventory, last.fresh, last.intent, last.commit, last.frontier, f.pin); err == nil {
		t.Fatal("same retained transition was applied twice")
	}
}

func TestProviderFrontierCustodyRejectsInertObjectMismatchWithoutWrites(t *testing.T) {
	f, u, r := providerFrontierWriterFixture(t)
	s := f.store
	record := u.records[0]
	h := providerFrontierNextFileHead(t, f, record)
	fresh := append([]providerIntegrationObject(nil), record.fresh...)
	fresh[0].digest = admissionTestDigest("f") // Inert record, no attacker runtime.
	stored, raw, head := s.stored, s.retainedRaw, f.pinRaw
	if _, err := r.retain(s, record.inventory, fresh, record.intent, record.commit, record.frontier, h); err == nil || !s.failed || !f.failed || !r.failed || s.stored != stored || s.retainedRaw != raw || f.pinRaw != head || len(s.objects) != 0 {
		t.Fatal("invalid data wrote a prefix or kept an owner reusable", err)
	}
	if !f.readExact("HEAD", []byte(head)) {
		t.Fatal("inert data refusal changed the original head")
	}
	if _, err := r.retain(s, record.inventory, record.fresh, record.intent, record.commit, record.frontier, h); err == nil {
		t.Fatal("quarantined writer was automatically repaired")
	}
}

func TestProviderFrontierCustodyNilOwnerIsNotAnExecutionFactory(t *testing.T) {
	var r *providerFrontierReplay
	if _, err := r.retain(nil, nil, nil, nil, nil, nil, providerFrontierHighwater{}); err == nil {
		t.Fatal("nil owner manufactured candidate success")
	}
}

func TestProviderFrontierSealPreflightMatchesBothActualDebitBoundaries(t *testing.T) {
	m, _ := providerPrivateLedgerFixture(t, 2, 4)
	failed := ledgerTestSettlement(t, m.lastTicket, "failed", 20)
	if _, err := m.AppendCandidateSettlement(failed); err != nil {
		t.Fatal(err)
	}
	b, err := m.prepareProviderBoundary(true)
	if err != nil {
		t.Fatal(err)
	}
	heads := [3]string{admissionTestDigest("a"), admissionTestDigest("b"), admissionTestDigest("c")}
	seal, err := b.sealObject(heads)
	if err != nil {
		t.Fatal(err)
	}
	closing := uint64(len(b.pageObject.raw) + len(seal.raw))
	pageUpper, ok := providerLedgerPageUpper(uint64(len(m.continuation.records)))
	future, futureOK := providerUpperSum(pageUpper, providerLedgerSealLimit)
	if !ok || !futureOK || future <= closing {
		t.Fatal("finite normal fixture must distinguish upper from exact closing")
	}
	const nonclosing = uint64(113)
	current := m.current.metadata + m.current.privateMetadata
	c := m.continuation
	setLimit := func(limit uint64) {
		c.bounds.maximumMetadataBytes, c.bounds.pageMetadataBytes = limit, limit
	}
	// Exact closing fits, but the first actual nonclosing debit still owns its
	// future page upper. It must be refused before any physical write occurs.
	setLimit(current + nonclosing + closing)
	if providerFrontierSealQuotaFits(m, nonclosing+closing, closing) {
		t.Fatal("exact-only preflight released the future upper early")
	}
	setLimit(current + nonclosing + future - 1)
	if providerFrontierSealQuotaFits(m, nonclosing+closing, closing) {
		t.Fatal("one byte below the first actual debit bound")
	}
	setLimit(current + nonclosing + future)
	if !providerFrontierSealQuotaFits(m, nonclosing+closing, closing) {
		t.Fatal("exact full reservation limit was refused")
	}
	before := m.current
	if err := m.chargeProviderStorage(before, 0, nonclosing, b, seal.digest); err != nil {
		t.Fatal("preflight and actual nonclosing debit disagree", err)
	}
	if err := m.applyProviderBoundary(b, []byte(b.pageObject.raw), []byte(seal.raw), heads); err != nil {
		t.Fatal("preflight and actual closing debit disagree", err)
	}
	if !c.final || m.current.privateMetadata != before.privateMetadata+nonclosing+closing {
		t.Fatal("upper was recorded as actual or the closing was doubled")
	}
}

func TestProviderFrontierCustodyRetainsTheWholeNormalFailedTicketAndFinalSeal(t *testing.T) {
	_, policy := providerPrivateLedgerFixture(t, 8, 32)
	clock, err := newProviderOwnedClock(policy, 512, 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	u := providerFrontierUnitFixtureMode(t, clock, true)
	ticket := u.firstTicket(t, 0)
	failed := ledgerTestSettlement(t, ticket, "failed", 0)
	if _, err := u.ledger.AppendCandidateSettlement(failed); err != nil {
		t.Fatal(err)
	}
	u.transitionData(t, "settle", "settlement", failed.CanonicalBytes(), 0)
	heads := u.indexes.heads
	boundary, err := u.ledger.prepareProviderBoundary(true)
	if err != nil {
		t.Fatal(err)
	}
	seal, err := boundary.sealObject(heads)
	if err != nil {
		t.Fatal(err)
	}
	next := providerIndexRoot(t, u.root, u.ledger)
	u.applyData(t, "seal", admissionTestDigest("e"), next, "ledger-seal", []byte(seal.raw), []providerIntegrationObjectInput{{"enrollment", next.CanonicalBytes()}, {"budget", u.ledger.current.BudgetBytes()}, {"ledger-page", []byte(boundary.pageObject.raw)}, {"ledger-seal", []byte(seal.raw)}}, 0)
	if err := u.ledger.applyProviderBoundary(boundary, []byte(boundary.pageObject.raw), []byte(seal.raw), heads); err != nil {
		t.Fatal(err)
	}
	f, _ := providerFrontierFileFixtureFromUnit(t, clock, u)
	if _, err := f.initialize(); err != nil {
		t.Fatal(err)
	}
	r, err := newProviderFrontierReplay(f.store.config)
	if err != nil {
		t.Fatal(err)
	}
	f.store.frontierReplay = r
	for _, record := range u.records {
		h := providerFrontierNextFileHead(t, f, record)
		if _, err := r.retain(f.store, record.inventory, record.fresh, record.intent, record.commit, record.frontier, h); err != nil {
			t.Fatal("normal owned full retention through seal", err)
		}
	}
	if !r.body.ledger.continuation.final || r.body.ledger.current.privateMetadata != u.ledger.current.privateMetadata || r.body.ledger.current.privatePayload != u.ledger.current.privatePayload || r.current.seal != seal.digest || r.current.checkpoint != "" || f.store.last != nil {
		t.Fatal("final seal lost common metadata or invented publication/ACK")
	}
}

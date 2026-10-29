package registry011

import "testing"

// Finite inert quota scenarios only: these costs are explicit test amounts,
// not actual measurements, retained writes, a frontier CAS or custody ACK.
func TestProviderLedgerIndexesBindPreAdmissionRetentionExactlyOnce(t *testing.T) {
	u := providerFrontierUnitFixture(t)
	s := u.indexes
	m, _ := providerPrivateLedgerFixture(t, 8, 32)
	if err := s.bindRetainedLedger(m, providerFrontierRetention{payload: 7, metadata: 11}); err != nil {
		t.Fatal(err)
	}
	if s.bindRetainedLedger(m, providerFrontierRetention{payload: 7, metadata: 11}) == nil {
		t.Fatal("pre-admission retention bound twice")
	}
	if s.ledger != m || m.current.privatePayload != 7 || m.current.privateMetadata != s.schemaBytes+11 {
		t.Fatal("duplicate binding changed the ledger")
	}
}

func TestProviderLedgerIndexAndCustodyDebitsAreOneAtomicQuotaChange(t *testing.T) {
	s, m, root := providerIndexFixture(t)
	entry, err := newProviderIntegrationObject("ticket", m.lastTicket.CanonicalBytes(), 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	next := providerIndexRoot(t, root, m)
	d, err := s.prepare("admit", entry.digest, 2, entry, next, m)
	if err != nil {
		t.Fatal(err)
	}
	before := m.current
	heads := s.heads
	if s.applyRetained(d, providerFrontierRetention{payload: ^uint64(0), metadata: 17}) == nil || s.heads != heads || m.current != before || d.used {
		t.Fatal("failed combined quota partially changed the ledger/index")
	}
	if err := s.applyRetained(d, providerFrontierRetention{payload: 13, metadata: 17}); err != nil {
		t.Fatal(err)
	}
	if m.current.privatePayload != 13 || m.current.privateMetadata != before.privateMetadata+d.metadata+17 || m.current.digest != before.digest || m.current.totals != before.totals || s.tickets[entry.digest].ticket != entry.digest {
		t.Fatal("combined debit omitted custody cost or changed the original budget")
	}
}

func TestProviderLedgerSealOwnsOneCombinedIndexAndCustodyDebit(t *testing.T) {
	s, m, root := providerIndexFixture(t)
	first := m.lastTicket
	root = providerIndexApply(t, s, m, root, "admit", "ticket", first.CanonicalBytes())
	failed := ledgerTestSettlement(t, first, "failed", 20)
	if _, err := m.AppendCandidateSettlement(failed); err != nil {
		t.Fatal(err)
	}
	root = providerIndexApply(t, s, m, root, "settle", "settlement", failed.CanonicalBytes())
	boundary, err := m.prepareProviderBoundary(true)
	if err != nil {
		t.Fatal(err)
	}
	heads := s.heads
	seal, err := boundary.sealObject(heads)
	if err != nil {
		t.Fatal(err)
	}
	next := providerIndexRoot(t, root, m)
	d, err := s.prepare("seal", admissionTestDigest("b"), uint64(len(s.operations))+1, seal, next, m)
	if err != nil {
		t.Fatal(err)
	}
	before := m.current
	if s.applyRetained(d, providerFrontierRetention{payload: 1, metadata: 23}) == nil || m.current != before || s.heads != heads {
		t.Fatal("payload was added after seal preparation")
	}
	if err := s.applyRetained(d, providerFrontierRetention{metadata: 23}); err != nil {
		t.Fatal(err)
	}
	if boundary.sealMetadata != d.metadata+23 || m.addProviderRetention(m.current, 0, 1) == nil {
		t.Fatal("seal admitted a second metadata debit")
	}
	if err := m.applyProviderBoundary(boundary, []byte(boundary.pageObject.raw), []byte(seal.raw), heads); err != nil {
		t.Fatal(err)
	}
	if !m.continuation.final || m.current.privateMetadata != before.privateMetadata+d.metadata+23+uint64(len(boundary.pageObject.raw)+len(seal.raw)) {
		t.Fatal("final sealing omitted custody/index/page bytes")
	}
}

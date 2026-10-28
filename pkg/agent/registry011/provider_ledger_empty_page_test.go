package registry011

import "testing"

// Scalar DATA only: no selected ledger, G, clock, disks or dispatch authority.
func TestProviderNextPageMetadataReservesOneEntryWithoutAnEmptyPage(t *testing.T) {
	t.Parallel()
	const metadata uint64 = 500
	one, ok := providerLedgerPageUpper(1)
	if !ok {
		t.Fatal("one-entry bound")
	}
	if _, ok := providerLedgerPageUpper(0); ok {
		t.Fatal("empty page became serializable")
	}
	current := &CandidateAdmissionLedgerChange{metadata: 100}
	m := &CandidateAdmissionLedger{current: current}
	c := &providerLedgerContinuation{page: 1, sealedMetadata: 100, extraMetadata: 200, sealedExtra: 200, auxiliaryMetadata: 300, sealedAuxiliary: 300}
	needed := one + providerLedgerSealLimit + metadata
	c.bounds.maximumMetadataBytes = 100 + 200 + 300 + needed
	c.bounds.pageMetadataBytes = needed
	before := current.metadata
	if !c.metadataRoom(m, 0, metadata) || current.metadata != before || current.privateMetadata != 0 || current.payload != 0 || len(current.nodes) != 0 || m.current != current || c.auxiliaryMetadata != 300 || len(c.records) != 0 {
		t.Fatal("next metadata reservation changed records, counters or missed the one-entry reserve")
	}
	c.bounds.pageMetadataBytes--
	if c.metadataRoom(m, 0, metadata) {
		t.Fatal("one-short active-page capacity accepted metadata")
	}
	c.bounds.pageMetadataBytes++
	c.bounds.maximumMetadataBytes--
	if c.metadataRoom(m, 0, metadata) {
		t.Fatal("one-short complete-lifetime capacity accepted metadata")
	}
	c.bounds.maximumMetadataBytes++
	if c.metadataRoom(m, 4097, metadata) || c.metadataRoom(m, 0, ^uint64(0)) {
		t.Fatal("page cap or scalar overflow was ignored")
	}
	if current.metadata != before || current.privateMetadata != 0 || current.payload != 0 || len(current.nodes) != 0 || m.current != current || len(c.records) != 0 {
		t.Fatal("a scalar quote produced a ledger record")
	}
}

// The next G can account for its metadata before T, while payload still needs
// a typed page record. Neither scalar accounting creates an execution role.
func TestProviderNextEmptyPageAcceptsOnlyMetadata(t *testing.T) {
	t.Parallel()
	m, _ := providerPrivateLedgerFixture(t, 1, 2)
	boundary, err := m.prepareProviderBoundary(false)
	if err != nil {
		t.Fatal(err)
	}
	heads := [3]string{admissionTestDigest("a"), admissionTestDigest("b"), admissionTestDigest("c")}
	seal, err := boundary.sealObject(heads)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.applyProviderBoundary(boundary, []byte(boundary.pageObject.raw), []byte(seal.raw), heads); err != nil {
		t.Fatal(err)
	}
	before := m.current
	for _, delta := range [][2]uint64{{1, 0}, {1, 1}} {
		if m.addProviderRetention(before, delta[0], delta[1]) == nil || m.current != before {
			t.Fatal("untyped payload entered an empty page")
		}
	}
	if err := m.addProviderRetention(before, 0, 31); err != nil || m.current.privatePayload != before.privatePayload || m.current.privateMetadata != before.privateMetadata+31 || m.current.counts != before.counts || m.current.totals != before.totals || len(m.continuation.records) != 0 {
		t.Fatal("metadata-only next entry changed records or replenished a lifetime", err)
	}
	if _, err := m.prepareProviderBoundary(false); err == nil {
		t.Fatal("metadata serialized an empty page")
	}
}

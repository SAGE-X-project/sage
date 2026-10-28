package registry011

import (
	"bytes"
	"encoding/json"
	"strconv"
	"testing"
)

// Every value here is inert arithmetic fixture data. No store/frontier,
// operating policy, network transport or dispatch permission is constructed.
func providerPrivateLedgerFixture(t *testing.T, pageRecords, records uint64) (*CandidateAdmissionLedger, *ProviderPolicyCandidate) {
	return providerPrivateLedgerFixtureWithDeadline(t, pageRecords, records, 1000)
}

func providerPrivateLedgerFixtureWithDeadline(t *testing.T, pageRecords, records, deadlineMS uint64) (*CandidateAdmissionLedger, *ProviderPolicyCandidate) {
	t.Helper()
	object := providerPolicyTestObject()
	limits := object["limits"].(map[string]any)
	for _, level := range []string{"operation", "segment", "bootstrap"} {
		values := limits[level].(map[string]any)
		for key := range values {
			if key != "deadline_ms" {
				values[key] = "65536"
			} else {
				values[key] = strconv.FormatUint(deadlineMS, 10)
			}
		}
	}
	limits["live_disk_bytes"], limits["retained_bytes"] = "1073741824", "1073741824"
	p, intent, first := ledgerTestFirst(t, object)
	b := providerLedgerBoundsFixture()
	b.pageRecords, b.maximumRecords = pageRecords, records
	b.maximumPages = (records-1)/pageRecords + 1
	b.maximumCommits = records + b.maximumPages + 2
	m, err := newProviderContinuingLedger(p, intent, first, b)
	if err != nil {
		t.Fatal(err)
	}
	return m, p
}

func TestProviderPrivateLedgerContinuationKeepsGlobalClaimsPast4096(t *testing.T) {
	m, policy := providerPrivateLedgerFixture(t, 4096, 8192)
	public, err := NewCandidateAdmissionLedger(policy, m.intent, m.lastTicket, CandidateLedgerBounds{4096, 16 << 20, 64 << 20})
	if err != nil || public.continuation != nil {
		t.Fatal("exported candidate constructor changed", err)
	}
	for n := 1; n < 4096; n++ {
		next := ledgerTestNext(t, policy, m.lastTicket, 0)
		if _, err := m.AppendCandidateTicket(next); err != nil {
			t.Fatal(n, err)
		}
		if _, err := public.AppendCandidateTicket(next); err != nil {
			t.Fatal(n, err)
		}
	}
	before := ledgerTestCurrent(t, m)
	next := ledgerTestNext(t, policy, m.lastTicket, 0)
	if _, err := m.AppendCandidateTicket(next); err == nil {
		t.Fatal("private page crossed without a seal")
	}
	if _, err := public.AppendCandidateTicket(next); err == nil {
		t.Fatal("public candidate lifetime cap was expanded")
	}
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
	if !bytes.Equal(m.current.BudgetBytes(), before.BudgetBytes()) || before.privateMetadata != 0 || m.current.privateMetadata != uint64(len(boundary.pageObject.raw)+len(seal.raw)) || m.intent != public.intent || m.current.totals != before.totals || len(m.tickets) != 4096 || m.current.counts != [3]uint64{4096, 0, 0} {
		t.Fatal("page reset lifetime claims, original intent or outstanding totals")
	}
	if _, err := m.AppendCandidateTicket(next); err != nil {
		t.Fatal(err)
	}
	if m.current.counts[0] != 4097 || m.current.sequence != 4096 || m.lastTicket.previous != public.lastTicket.digest || m.continuation.sealedRecords != 4096 {
		t.Fatal("global sequence or predecessor restarted")
	}
	if _, err := m.AppendCandidateTicket(public.lastTicket); err == nil {
		t.Fatal("old ticket tombstone was dropped")
	}
	if err := m.applyProviderBoundary(boundary, []byte(boundary.pageObject.raw), []byte(seal.raw), heads); err == nil {
		t.Fatal("used/stale boundary accepted")
	}
	if _, err := public.AppendCandidateTicket(next); err == nil {
		t.Fatal("private continuation leaked into public model")
	}
}

func TestProviderPrivateLedgerPageRejectsMismatchesWithoutChangingState(t *testing.T) {
	m, policy := providerPrivateLedgerFixture(t, 2, 4)
	if _, err := m.prepareProviderBoundary(false); err == nil {
		t.Fatal("nonfinal short page")
	}
	if _, err := m.prepareProviderBoundary(true); err == nil {
		t.Fatal("final page discarded outstanding reservation")
	}
	if _, err := m.AppendCandidateTicket(ledgerTestNext(t, policy, m.lastTicket, 0)); err != nil {
		t.Fatal(err)
	}
	boundary, err := m.prepareProviderBoundary(false)
	if err != nil {
		t.Fatal(err)
	}
	heads := [3]string{admissionTestDigest("a"), admissionTestDigest("b"), admissionTestDigest("c")}
	seal, err := boundary.sealObject(heads)
	if err != nil {
		t.Fatal(err)
	}
	before := m.current
	for _, mode := range []string{"page", "seal", "owner", "page-index", "records", "payload", "metadata", "final", "index-head"} {
		t.Run(mode, func(t *testing.T) {
			copy := *boundary
			pageBytes, sealBytes := []byte(boundary.pageObject.raw), []byte(seal.raw)
			indexHeads := heads
			switch mode {
			case "page":
				pageBytes = []byte("wrong")
			case "seal":
				sealBytes = []byte("wrong")
			case "owner":
				copy.owner = &CandidateAdmissionLedger{}
			case "page-index":
				copy.page++
			case "records":
				copy.records++
			case "payload":
				copy.payload++
			case "metadata":
				copy.metadata++
			case "final":
				copy.final = true
			case "index-head":
				indexHeads[0] = "wrong"
			}
			if m.applyProviderBoundary(&copy, pageBytes, sealBytes, indexHeads) == nil {
				t.Fatal("mismatched arithmetic boundary")
			}
			if m.current != before || m.continuation.sealedRecords != 0 || boundary.used {
				t.Fatal("failed page changed lifetime state")
			}
		})
	}
	page := contentJSON([]byte(boundary.pageObject.raw)).(map[string]any)
	if page["first_record"] != "0" || page["last_record"] != "1" || page["previous_page"] != nil || page["previous_budget"] != nil {
		t.Fatal("wrong initial page closure")
	}
	entries := page["records"].([]any)
	if len(entries) != 2 || entries[1].(map[string]any)["ordinal"] != "1" {
		t.Fatal("page omitted a global record")
	}
	sealObject := contentJSON([]byte(seal.raw)).(map[string]any)
	if sealObject["ticket_index_root"] != heads[0] || sealObject["operation_index_root"] != heads[1] || sealObject["segment_totals_root"] != heads[2] {
		t.Fatal("seal used future index heads")
	}
	for _, value := range []any{&providerLedgerBoundary{}, &providerLedgerContinuation{}} {
		if json.Unmarshal([]byte(`{}`), value) == nil {
			t.Fatal("JSON recreated a private boundary")
		}
	}
}

func TestProviderPrivateLedgerFinalShortPageIsTerminalAndFullyCharged(t *testing.T) {
	m, policy := providerPrivateLedgerFixture(t, 4, 8)
	first := m.lastTicket
	settlement := ledgerTestSettlement(t, first, "failed", 10)
	if _, err := m.AppendCandidateSettlement(settlement); err != nil {
		t.Fatal(err)
	}
	boundary, err := m.prepareProviderBoundary(true)
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
	if !m.valid() || !m.continuation.final || m.current.totals.bootstrap.charged != first.work || m.current.totals.bootstrap.outstanding != (candidateWorkVector{}) {
		t.Fatal("final page refunded or lost work")
	}
	if _, err := m.AppendCandidateTicket(ledgerTestNext(t, policy, first, 0)); err == nil {
		t.Fatal("final short page resumed work")
	}
	if _, err := m.prepareProviderBoundary(true); err == nil {
		t.Fatal("final page repeated")
	}
	if !bytes.Equal(m.current.BudgetBytes(), boundary.change.BudgetBytes()) || strconv.FormatUint(m.intent.deadline, 10) != contentJSON([]byte(seal.raw)).(map[string]any)["deadline_wall_ms"] {
		t.Fatal("sealing changed the original budget/deadline")
	}
}

func TestProviderLedgerPageMetadataIsReservedAndDebited(t *testing.T) {
	m, _ := providerPrivateLedgerFixture(t, 1, 2)
	c, before := m.continuation, m.current
	pageUpper, ok := providerLedgerPageUpper(1)
	if !ok {
		t.Fatal("page upper")
	}
	exact := before.metadata + pageUpper + providerLedgerSealLimit
	c.bounds.pageMetadataBytes, c.bounds.maximumMetadataBytes = exact, exact
	if !c.metadataRoom(m, 1, 0) {
		t.Fatal("exact metadata reservation refused")
	}
	c.bounds.pageMetadataBytes--
	if c.metadataRoom(m, 1, 0) {
		t.Fatal("one byte over page metadata accepted")
	}
	if _, err := m.prepareProviderBoundary(false); err == nil {
		t.Fatal("page constructed without its complete reservation")
	}
	c.bounds.pageMetadataBytes++
	c.bounds.maximumMetadataBytes--
	c.bounds.pageMetadataBytes--
	if c.metadataRoom(m, 1, 0) {
		t.Fatal("one byte over global metadata accepted")
	}
	c.bounds.maximumMetadataBytes, c.bounds.pageMetadataBytes = exact, exact
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
	diagnostic, err := m.current.Diagnostic()
	if err != nil || diagnostic.MetadataBytes != before.metadata+uint64(len(boundary.pageObject.raw)+len(seal.raw)) || before.privateMetadata != 0 || !bytes.Equal(before.BudgetBytes(), m.current.BudgetBytes()) {
		t.Fatal("actual page/seal bytes were lost or old change was mutated", err)
	}
}

func TestProviderLedgerPageResidentPreflight(t *testing.T) {
	m, _ := providerPrivateLedgerFixture(t, 1, 2)
	c, before := m.continuation, m.current
	pageUpper, ok := providerLedgerPageUpper(1)
	if !ok {
		t.Fatal("page upper")
	}
	retained := before.payload + before.metadata + c.extraMetadata
	scratch := 3*pageUpper + 2*providerLedgerSealLimit
	exact, ok := c.bounds.residentUpper(uint64(len(m.tickets)), before.sequence+2, uint64(len(m.segments)), 0, retained, scratch)
	if !ok {
		t.Fatal("fixture resident upper")
	}
	c.bounds.maximumResidentBytes = exact
	if _, err := m.providerPage(); err != nil {
		t.Fatal("exact resident preflight refused", err)
	}
	c.bounds.maximumResidentBytes--
	if _, err := m.providerPage(); err == nil {
		t.Fatal("one byte over resident limit allocated a page")
	}
	if m.current != before || c.sealedRecords != 0 || len(c.records) != 1 {
		t.Fatal("preflight refusal changed the ledger")
	}
	for _, count := range []uint64{0, 4097, ^uint64(0)} {
		if _, ok := providerLedgerPageUpper(count); ok {
			t.Fatal("unbounded page", count)
		}
	}
	if _, err := (*CandidateAdmissionLedger)(nil).providerPage(); err == nil {
		t.Fatal("nil page owner")
	}
}

func TestProviderLedgerSecondPageEnforcesGlobalMetadataSeparately(t *testing.T) {
	m, policy := providerPrivateLedgerFixture(t, 1, 4)
	heads := [3]string{admissionTestDigest("a"), admissionTestDigest("b"), admissionTestDigest("c")}
	boundary, err := m.prepareProviderBoundary(false)
	if err != nil {
		t.Fatal(err)
	}
	seal, err := boundary.sealObject(heads)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.applyProviderBoundary(boundary, []byte(boundary.pageObject.raw), []byte(seal.raw), heads); err != nil {
		t.Fatal(err)
	}
	if _, err := m.AppendCandidateTicket(ledgerTestNext(t, policy, m.lastTicket, 0)); err != nil {
		t.Fatal(err)
	}
	c, before := m.continuation, m.current
	upper, ok := providerLedgerPageUpper(1)
	if !ok {
		t.Fatal("page upper")
	}
	global := before.metadata + c.extraMetadata + upper + providerLedgerSealLimit
	active := before.metadata - c.sealedMetadata + c.extraMetadata - c.sealedExtra + upper + providerLedgerSealLimit
	if active >= global || c.sealedExtra == 0 {
		t.Fatal("fixture lacks previously retained metadata")
	}
	c.bounds.maximumMetadataBytes, c.bounds.pageMetadataBytes = global, active
	if !c.metadataRoom(m, 1, 0) {
		t.Fatal("exact global bound refused")
	}
	c.bounds.maximumMetadataBytes--
	if !c.bounds.valid() || c.bounds.pageMetadataBytes != active {
		t.Fatal("page cap changed or fixture invalid")
	}
	if c.metadataRoom(m, 1, 0) {
		t.Fatal("previous page metadata escaped global cap")
	}
	if _, err := m.prepareProviderBoundary(false); err == nil {
		t.Fatal("global one-byte shortfall built a page")
	}
	if m.current != before || c.sealedRecords != 1 {
		t.Fatal("global refusal changed prefix")
	}
}

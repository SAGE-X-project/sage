package registry011

import (
	"encoding/json"
	"strconv"
	"testing"
)

// These fixtures reconstruct candidate arithmetic independently. They do not
// open custody, select a policy, use a transport or authorize any Agent work.
func providerIndexFixture(t *testing.T) (*providerLedgerIndexes, *CandidateAdmissionLedger, *CandidateEnrollmentRoot) {
	return providerIndexFixtureWithMetadata(t, 0)
}

func providerIndexFixtureWithMetadata(t *testing.T, maximum uint64) (*providerLedgerIndexes, *CandidateAdmissionLedger, *CandidateEnrollmentRoot) {
	t.Helper()
	m, policy := providerPrivateLedgerFixture(t, 8, 32)
	if maximum != 0 {
		bounds := m.continuation.bounds
		bounds.maximumMetadataBytes, bounds.pageMetadataBytes = maximum, maximum
		var err error
		m, err = newProviderContinuingLedger(policy, m.intent, m.lastTicket, bounds)
		if err != nil {
			t.Fatal("selected inert metadata profile", err)
		}
	}
	id := m.intent.identity
	context := &ProviderCandidateContext{identity: id, bound: true}
	object := admissionTestRootObject(t)
	object["policy"] = id.Policy
	root, err := ParseCandidateEnrollmentRoot(context, providerPolicyTestBytes(t, object), 64<<10)
	if err != nil || root.digest != m.intent.anchor {
		t.Fatal("initial typed root", err)
	}
	s, anchors, err := newProviderLedgerIndexes(id, m.intent.fence, m.continuation.bounds)
	if err != nil || len(anchors) != 3 {
		t.Fatal(err)
	}
	for i, anchor := range anchors {
		if anchor.digest != s.heads[i] || contentJSON([]byte(anchor.raw)).(map[string]any)["kind"] != []string{"tickets", "operations", "segments"}[i] {
			t.Fatal("empty index anchors")
		}
	}
	entry, err := newProviderIntegrationObject("enrollment", []byte(root.raw), 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	delta, err := s.prepare("enroll", admissionTestDigest("a"), 1, entry, root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.apply(delta); err != nil {
		t.Fatal(err)
	}
	if s.admission != "" || len(s.tickets) != 0 || len(s.segments) != 0 || s.counts != [3]uint64{0, 1, 0} {
		t.Fatal("initial enrollment referred to future admission")
	}
	if err := s.bindLedger(m); err != nil {
		t.Fatal(err)
	}
	return s, m, root
}

func providerIndexRoot(t *testing.T, before *CandidateEnrollmentRoot, m *CandidateAdmissionLedger) *CandidateEnrollmentRoot {
	t.Helper()
	object := contentJSON([]byte(before.raw)).(map[string]any)
	object["sequence"], object["previous_root"], object["budget_root"] = strconv.FormatUint(before.sequence+1, 10), before.digest, m.current.digest
	root, err := ParseCandidateEnrollmentRoot(&ProviderCandidateContext{identity: m.intent.identity, bound: true}, providerIntegrationJSON(object), 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func providerIndexApply(t *testing.T, s *providerLedgerIndexes, m *CandidateAdmissionLedger, root *CandidateEnrollmentRoot, kind, entryKind string, raw []byte) *CandidateEnrollmentRoot {
	t.Helper()
	next := providerIndexRoot(t, root, m)
	entry, err := newProviderIntegrationObject(entryKind, raw, 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	delta, err := s.prepare(kind, entry.digest, uint64(len(s.operations))+1, entry, next, m)
	if err != nil {
		t.Fatal(kind, err)
	}
	if err := s.apply(delta); err != nil {
		t.Fatal(kind, err)
	}
	if s.apply(delta) == nil {
		t.Fatal("index delta reused")
	}
	return next
}

func TestProviderLedgerIndexesRetainTicketTerminalAndSegmentTotals(t *testing.T) {
	s, m, root := providerIndexFixture(t)
	first := m.lastTicket
	root = providerIndexApply(t, s, m, root, "admit", "ticket", first.CanonicalBytes())
	key := candidateAdmissionSegment(m.intent.digest, 0)
	if len(s.tickets) != 1 || s.segments[key].totals.outstanding != first.work || s.admission != m.intent.digest {
		t.Fatal("ticket/segment mutation lost its reservation")
	}
	consumed := ledgerTestConsumption(t, first, nil, 10)
	if _, err := m.AppendCandidateConsumption(consumed); err != nil {
		t.Fatal(err)
	}
	segmentHead := s.heads[2]
	root = providerIndexApply(t, s, m, root, "consume", "consumption", []byte(consumed.raw))
	if s.heads[2] != segmentHead || s.tickets[first.digest].consumption != consumed.digest {
		t.Fatal("consumption changed segment totals or lost tombstone")
	}
	settled := ledgerTestSettlement(t, first, "failed", 20)
	if _, err := m.AppendCandidateSettlement(settled); err != nil {
		t.Fatal(err)
	}
	root = providerIndexApply(t, s, m, root, "settle", "settlement", settled.CanonicalBytes())
	if s.tickets[first.digest].settlement != settled.digest || s.segments[key].totals.charged != first.work || s.segments[key].totals.outstanding != (candidateWorkVector{}) {
		t.Fatal("terminal state was refunded or lost")
	}
	preseal := s.heads
	boundary, err := m.prepareProviderBoundary(true)
	if err != nil {
		t.Fatal(err)
	}
	seal, err := boundary.sealObject(preseal)
	if err != nil {
		t.Fatal(err)
	}
	next := providerIndexRoot(t, root, m)
	operation := admissionTestDigest("b")
	delta, err := s.prepare("seal", operation, uint64(len(s.operations))+1, seal, next, m)
	if err != nil || s.heads != preseal {
		t.Fatal("seal used a future operation head", err)
	}
	if err := s.apply(delta); err != nil {
		t.Fatal(err)
	}
	if s.heads[0] != preseal[0] || s.heads[2] != preseal[2] || s.heads[1] == preseal[1] || s.operations[operation].kind != "seal" {
		t.Fatal("seal changed the wrong index")
	}
	if err := m.applyProviderBoundary(boundary, []byte(boundary.pageObject.raw), []byte(seal.raw), preseal); err != nil {
		t.Fatal(err)
	}
	if !m.continuation.final || s.tickets[first.digest].settlement != settled.digest {
		t.Fatal("final seal dropped terminal history")
	}
}

func TestProviderLedgerIndexesIndependentReplayHasExactHeads(t *testing.T) {
	a, ma, ra := providerIndexFixture(t)
	b, mb, rb := providerIndexFixture(t)
	if a == b || a.heads != b.heads || ma == mb || ma.intent.digest != mb.intent.digest {
		t.Fatal("independent fixture")
	}
	ra = providerIndexApply(t, a, ma, ra, "admit", "ticket", ma.lastTicket.CanonicalBytes())
	rb = providerIndexApply(t, b, mb, rb, "admit", "ticket", mb.lastTicket.CanonicalBytes())
	ca := ledgerTestConsumption(t, ma.lastTicket, nil, 10)
	cb, err := ParseCandidateConsumption(mb.lastTicket, []byte(ca.raw), 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ma.AppendCandidateConsumption(ca); err != nil {
		t.Fatal(err)
	}
	if _, err := mb.AppendCandidateConsumption(cb); err != nil {
		t.Fatal(err)
	}
	ra = providerIndexApply(t, a, ma, ra, "consume", "consumption", []byte(ca.raw))
	rb = providerIndexApply(t, b, mb, rb, "consume", "consumption", []byte(cb.raw))
	if a.heads != b.heads || a.counts != b.counts || a.schemaBytes != b.schemaBytes || ra.digest != rb.digest || !bytesEqualProviderValue(a.tickets[ma.lastTicket.digest].object(), b.tickets[mb.lastTicket.digest].object()) {
		t.Fatal("independent typed replay disagrees")
	}
}

func TestProviderLedgerIndexDeltaRefusalIsAtomic(t *testing.T) {
	s, m, root := providerIndexFixture(t)
	entry, err := newProviderIntegrationObject("ticket", m.lastTicket.CanonicalBytes(), 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	next := providerIndexRoot(t, root, m)
	delta, err := s.prepare("admit", entry.digest, 2, entry, next, m)
	if err != nil {
		t.Fatal(err)
	}
	before := s.heads
	for _, name := range []string{"owner", "previous", "count", "used", "metadata", "duplicate-chain", "node", "expected", "value", "admission"} {
		t.Run(name, func(t *testing.T) {
			copy := *delta
			copy.mutations = append([]providerIndexMutation(nil), delta.mutations...)
			switch name {
			case "owner":
				copy.owner = &providerLedgerIndexes{}
			case "previous":
				copy.previous[0] = admissionTestDigest("f")
			case "count":
				copy.counts[0]++
			case "used":
				copy.used = true
			case "metadata":
				copy.metadata--
			case "duplicate-chain":
				copy.mutations[1] = copy.mutations[0]
			case "node":
				copy.mutations[0].object.digest = admissionTestDigest("e")
			case "expected":
				copy.mutations[0].expected = map[string]any{}
			case "value":
				copy.mutations[0].value = nil
			case "admission":
				copy.admission = admissionTestDigest("d")
			}
			if s.apply(&copy) == nil {
				t.Fatal("malformed delta accepted")
			}
			if s.heads != before || len(s.tickets) != 0 || len(s.operations) != 1 || delta.used {
				t.Fatal("refused delta changed a map")
			}
		})
	}
	if err := s.apply(delta); err != nil {
		t.Fatal(err)
	}
	if _, err := s.prepare("admit", entry.digest, 3, entry, next, m); err == nil {
		t.Fatal("operation tombstone discarded")
	}
	for _, value := range []any{&providerLedgerIndexes{}, &providerIndexDelta{}} {
		if json.Unmarshal([]byte(`{}`), value) == nil {
			t.Fatal("JSON recreated a private index owner")
		}
	}
}

func TestProviderLedgerIndexPreparationRejectsWrongBoundary(t *testing.T) {
	s, m, root := providerIndexFixture(t)
	entry, err := newProviderIntegrationObject("ticket", m.lastTicket.CanonicalBytes(), 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	next := providerIndexRoot(t, root, m)
	for _, name := range []string{"kind", "operation", "sequence", "root", "fence", "ledger", "digest", "entry-kind", "old-budget"} {
		t.Run(name, func(t *testing.T) {
			kind, operation, sequence := "admit", entry.digest, uint64(2)
			localEntry, localRoot, localLedger := entry, *next, m
			switch name {
			case "kind":
				kind = "unknown"
			case "operation":
				operation = admissionTestDigest("c")
			case "sequence":
				sequence++
			case "root":
				localRoot.identity.Instance = admissionTestDigest("c")
			case "fence":
				localRoot.fence++
			case "ledger":
				localLedger = nil
			case "digest":
				localEntry.digest = admissionTestDigest("d")
			case "entry-kind":
				localEntry.kind = "consumption"
			case "old-budget":
				localRoot.budget = admissionTestDigest("e")
			}
			if _, err := s.prepare(kind, operation, sequence, localEntry, &localRoot, localLedger); err == nil {
				t.Fatal("invalid index boundary")
			}
		})
	}
}

func TestProviderLedgerIndexApplyReconstructsTypedTerminalTransition(t *testing.T) {
	s, m, root := providerIndexFixture(t)
	ticket := m.lastTicket
	root = providerIndexApply(t, s, m, root, "admit", "ticket", ticket.CanonicalBytes())
	consumed := ledgerTestConsumption(t, ticket, nil, 10)
	if _, err := m.AppendCandidateConsumption(consumed); err != nil {
		t.Fatal(err)
	}
	root = providerIndexApply(t, s, m, root, "consume", "consumption", []byte(consumed.raw))
	settled := ledgerTestSettlement(t, ticket, "failed", 20)
	if _, err := m.AppendCandidateSettlement(settled); err != nil {
		t.Fatal(err)
	}
	entry, err := newProviderIntegrationObject("settlement", settled.CanonicalBytes(), 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	delta, err := s.prepare("settle", entry.digest, 4, entry, providerIndexRoot(t, root, m), m)
	if err != nil {
		t.Fatal(err)
	}
	before, ledgerBefore := s.heads, m.current
	for _, mode := range []string{"clear-consumption", "replace-consumption", "clear-settlement", "replace-entry", "replace-budget"} {
		t.Run(mode, func(t *testing.T) {
			copy := *delta
			copy.mutations = append([]providerIndexMutation(nil), delta.mutations...)
			node := &copy.mutations[0]
			value := contentJSON(providerIntegrationJSON(node.value)).(map[string]any)
			raw := contentJSON([]byte(node.object.raw)).(map[string]any)
			switch mode {
			case "clear-consumption":
				value["consumption_digest"] = nil
			case "replace-consumption":
				value["consumption_digest"] = admissionTestDigest("f")
			case "clear-settlement":
				value["settlement_digest"] = nil
			case "replace-entry":
				raw["entry_digest"] = admissionTestDigest("e")
			case "replace-budget":
				raw["budget_root"] = admissionTestDigest("e")
			}
			node.value, raw["value"] = value, value
			node.object, err = newProviderIntegrationObject("index-mutation", providerIntegrationJSON(raw), candidateLedgerMetadataLimit)
			if err != nil {
				t.Fatal(err)
			}
			copy.metadata = 0
			for _, item := range copy.mutations {
				copy.metadata += uint64(len(item.object.raw))
			}
			if s.apply(&copy) == nil {
				t.Fatal("canonical valid-shaped replacement bypassed typed reconstruction")
			}
			if s.heads != before || m.current != ledgerBefore || s.tickets[ticket.digest].consumption != consumed.digest || s.tickets[ticket.digest].settlement != "" || delta.used {
				t.Fatal("rejected delta changed a tombstone or quota")
			}
		})
	}
	if err := s.apply(delta); err != nil {
		t.Fatal(err)
	}
}

func TestProviderLedgerIndexesShareThePageAndLifetimeMetadataQuota(t *testing.T) {
	s, m, root := providerIndexFixture(t)
	if m.current.privateMetadata != s.schemaBytes || m.continuation.auxiliaryMetadata != s.schemaBytes {
		t.Fatal("initial anchors/enrollment were not charged to the common quota")
	}
	entry, err := newProviderIntegrationObject("ticket", m.lastTicket.CanonicalBytes(), 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	delta, err := s.prepare("admit", entry.digest, 2, entry, providerIndexRoot(t, root, m), m)
	if err != nil {
		t.Fatal(err)
	}
	c, before := m.continuation, m.current
	page, ok := providerLedgerPageUpper(uint64(len(c.records)))
	if !ok {
		t.Fatal("page upper")
	}
	exact := before.metadata + c.extraMetadata + c.auxiliaryMetadata + delta.metadata + page + providerLedgerSealLimit
	for _, short := range []bool{true, false} {
		limit := exact
		if short {
			limit--
		}
		// Recreate an inert candidate fixture with the final profile selected
		// before construction; never change a live profile after prepare.
		selected, model, initial := providerIndexFixtureWithMetadata(t, limit)
		entry, err := newProviderIntegrationObject("ticket", model.lastTicket.CanonicalBytes(), 64<<10)
		if err != nil {
			t.Fatal(err)
		}
		prepared, err := selected.prepare("admit", entry.digest, 2, entry, providerIndexRoot(t, initial, model), model)
		if err != nil {
			t.Fatal(err)
		}
		prior, heads := model.current, selected.heads
		err = selected.apply(prepared)
		if short {
			if err == nil || model.current != prior || selected.heads != heads || prepared.used {
				t.Fatal("combined shortfall accepted or changed an owner")
			}
		} else {
			if err != nil || model.current.privateMetadata != selected.schemaBytes || prior.privateMetadata+prepared.metadata != model.current.privateMetadata {
				t.Fatal("exact common quota/charge", err)
			}
			if err := selected.bindLedger(model); err == nil {
				t.Fatal("initial metadata bound twice")
			}
		}
	}
}

func TestProviderLedgerSealBoundaryRejectsInterleavedMetadata(t *testing.T) {
	s, m, root := providerIndexFixture(t)
	ticket := m.lastTicket
	root = providerIndexApply(t, s, m, root, "admit", "ticket", ticket.CanonicalBytes())
	settled := ledgerTestSettlement(t, ticket, "failed", 10)
	if _, err := m.AppendCandidateSettlement(settled); err != nil {
		t.Fatal(err)
	}
	root = providerIndexApply(t, s, m, root, "settle", "settlement", settled.CanonicalBytes())
	boundary, err := m.prepareProviderBoundary(true)
	if err != nil {
		t.Fatal(err)
	}
	seal, err := boundary.sealObject(s.heads)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.prepareProviderBoundary(true); err == nil {
		t.Fatal("two live boundaries")
	}
	before := m.current
	if m.addProviderMetadata(before, 1) == nil || m.current != before {
		t.Fatal("unrelated metadata crossed the seal boundary")
	}
	verification, err := newProviderIntegrationObject("verification-receipt", providerIntegrationJSON(map[string]any{"format": "sage-provider-verification-receipt-v1"}), 4096)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.prepare("verify", admissionTestDigest("c"), 4, verification, providerIndexRoot(t, root, m), m); err == nil {
		t.Fatal("verification interleaved with seal")
	}
	preseal := s.heads
	delta, err := s.prepare("seal", admissionTestDigest("d"), 4, seal, providerIndexRoot(t, root, m), m)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.apply(delta); err != nil {
		t.Fatal(err)
	}
	if boundary.sealDigest != seal.digest || boundary.sealMetadata != delta.metadata {
		t.Fatal("seal metadata was not tied to exact operation")
	}
	if _, err := s.prepare("seal", admissionTestDigest("e"), 5, seal, providerIndexRoot(t, root, m), m); err == nil {
		t.Fatal("second seal operation interleaved")
	}
	if _, err := boundary.sealObject(s.heads); err == nil {
		t.Fatal("future operation head replaced pre-seal head")
	}
	if err := m.applyProviderBoundary(boundary, []byte(boundary.pageObject.raw), []byte(seal.raw), preseal); err != nil {
		t.Fatal(err)
	}
	if m.continuation.pending != nil || !m.continuation.final {
		t.Fatal("exact seal did not close the boundary")
	}
}

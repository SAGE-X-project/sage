package registry011

import (
	"bytes"
	"strconv"
	"testing"
)

func TestProviderLedgerRetainedPlaintextAndMetadataUseOneLifetimeQuota(t *testing.T) {
	m, _ := providerPrivateLedgerFixture(t, 8, 16)
	before := m.current
	if err := m.addProviderRetention(before, 37, 53); err != nil {
		t.Fatal(err)
	}
	d, err := m.current.Diagnostic()
	if err != nil || d.PayloadBytes != before.payload+37 || d.MetadataBytes != before.metadata+53 || !bytes.Equal(before.BudgetBytes(), m.current.BudgetBytes()) || before.privatePayload != 0 || before.privateMetadata != 0 || m.current.totals != before.totals {
		t.Fatal("retention reset/relabelled the original ledger", err)
	}
	if m.addProviderRetention(before, 1, 1) == nil || m.addProviderRetention(m.current, 0, 0) == nil {
		t.Fatal("stale or empty retention delta accepted")
	}
	current := m.current
	if m.addProviderRetention(current, ^uint64(0), 0) == nil || m.addProviderRetention(current, 0, ^uint64(0)) == nil || m.current != current {
		t.Fatal("overflow changed retained accounting")
	}
	public := &CandidateAdmissionLedger{}
	if public.addProviderRetention(nil, 1, 1) == nil || (*CandidateAdmissionLedger)(nil).addProviderRetention(nil, 1, 1) == nil {
		t.Fatal("retention fabricated a private owner")
	}
}

func TestProviderLedgerRetainedPayloadCannotResetAtPageBoundary(t *testing.T) {
	m, policy := providerPrivateLedgerFixture(t, 1, 2)
	if err := m.addProviderRetention(m.current, 17, 29); err != nil {
		t.Fatal(err)
	}
	boundary, err := m.prepareProviderBoundary(false)
	if err != nil {
		t.Fatal(err)
	}
	if m.addProviderRetention(m.current, 1, 0) == nil || m.addProviderRetention(m.current, 0, 1) == nil {
		t.Fatal("unrelated retention entered a prepared seal")
	}
	heads := [3]string{admissionTestDigest("a"), admissionTestDigest("b"), admissionTestDigest("c")}
	seal, err := boundary.sealObject(heads)
	if err != nil {
		t.Fatal(err)
	}
	value := contentJSON([]byte(seal.raw)).(map[string]any)
	if value["payload_bytes"] != strconv.FormatUint(m.current.payload+17, 10) {
		t.Fatal("seal omitted retained plaintext")
	}
	if err := m.applyProviderBoundary(boundary, []byte(boundary.pageObject.raw), []byte(seal.raw), heads); err != nil {
		t.Fatal(err)
	}
	if err := m.addProviderRetention(m.current, 19, 31); err != nil {
		t.Fatal(err)
	}
	if m.continuation.auxiliaryPayload != 36 || m.continuation.sealedExternal != 17 || m.current.privatePayload != 36 || m.continuation.auxiliaryMetadata != 60 {
		t.Fatal("page replenished lifetime retention")
	}
	first := m.lastTicket
	next := ledgerTestNext(t, policy, first, 0)
	if _, err := m.AppendCandidateTicket(next); err != nil || m.lastTicket.previous != first.digest || m.current.privatePayload != 36 {
		t.Fatal("next record discarded retained plaintext", err)
	}
}

func TestProviderLedgerRetainedPayloadExactLimitAndOneByteShortfall(t *testing.T) {
	for _, short := range []bool{false, true} {
		m, policy := providerPrivateLedgerFixture(t, 4, 8)
		b := m.continuation.bounds
		// Select a separate inert profile before construction, never mutate the
		// limits of a live model to make a limit test pass.
		b.maximumPayloadBytes, b.pagePayloadBytes = m.current.payload+100, m.current.payload+100
		var err error
		m, err = newProviderContinuingLedger(policy, m.intent, m.lastTicket, b)
		if err != nil {
			t.Fatal(err)
		}
		delta := uint64(100)
		if short {
			delta++
		}
		before := m.current
		err = m.addProviderRetention(before, delta, 0)
		if short && (err == nil || m.current != before) || !short && (err != nil || !m.valid()) {
			t.Fatal("exact combined payload limit", short, err)
		}
		if !short && m.addProviderRetention(m.current, 1, 0) == nil {
			t.Fatal("combined quota spent twice")
		}
	}
}

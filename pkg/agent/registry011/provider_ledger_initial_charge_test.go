package registry011

import "testing"

func TestProviderInitialChargePrecedesFirstTicketWithoutChangingPublicData(t *testing.T) {
	p, intent, first := ledgerTestFirst(t, nil)
	seed := &providerLedgerInitialCharge{work: candidateWorkVector{0, 0, 5, 0, 0, 0, 0, 0, 0, 7, 9}, retained: 12}
	m, err := newCandidateAdmissionLedgerSeeded(p, intent, first, ledgerTestBounds(), nil, seed)
	if err != nil || m.current.totals.bootstrap.charged != seed.work || m.current.totals.bootstrap.outstanding != first.work || m.current.totals.liveUsed != 12 || m.current.totals.retainedUsed != 12 || m.current.totals.liveReserved != first.live || m.current.totals.retainedHeld != first.retained || m.segments[0].charged != (candidateWorkVector{}) {
		t.Fatal("pre-I hold omitted, refunded, or mixed into operation totals", err)
	}
	before := m.current.totals
	next := ledgerTestNext(t, p, first, 0)
	if _, err := m.AppendCandidateTicket(next); err != nil || m.current.totals.bootstrap.charged != before.bootstrap.charged || m.current.totals.liveUsed != before.liveUsed || m.current.totals.retainedUsed != before.retainedUsed {
		t.Fatal("next ticket reset or repeated the initial charge", err)
	}
	public := ledgerTestModel(t, p, intent, first, ledgerTestBounds())
	if public.initialTotals != (candidateLedgerTotals{}) || public.current.totals.bootstrap.charged != (candidateWorkVector{}) || public.current.totals.liveUsed != 0 || public.current.totals.retainedUsed != 0 || seed.UnmarshalJSON([]byte(`{}`)) == nil {
		t.Fatal("public candidate acquired a seed or JSON restored it")
	}
	plain, policy := providerPrivateLedgerFixture(t, 8, 32)
	private, err := newProviderContinuingLedgerSeeded(policy, plain.intent, plain.lastTicket, plain.continuation.bounds, nil, seed)
	if err != nil || private.current.totals.bootstrap.charged != seed.work || private.current.totals.liveUsed != seed.retained || private.continuation == nil {
		t.Fatal("private continuation dropped its pre-ticket charge", err)
	}
}

func TestProviderInitialChargeChecksEveryCapacityBeforeFirstTicket(t *testing.T) {
	p, intent, first := ledgerTestFirst(t, nil)
	plain := ledgerTestModel(t, p, intent, first, ledgerTestBounds())
	for _, mode := range []string{"zero", "live", "retained", "written", "requests", "wire", "protected", "bootstrap", "decoded-zero", "written-zero", "work-zero", "combined-work", "combined-disk"} {
		t.Run(mode, func(t *testing.T) {
			seed := providerLedgerInitialCharge{work: candidateWorkVector{0, 0, 1, 0, 0, 0, 0, 0, 0, 1, 1}, retained: 2}
			switch mode {
			case "zero":
				seed.retained = 0
			case "live":
				seed.retained = plain.live + 1
			case "retained":
				seed.retained = plain.retained + 1
			case "written":
				seed.work[9] = seed.retained + 1
			case "requests":
				seed.work[0] = 1
			case "wire":
				seed.work[1] = 1
			case "protected":
				seed.work[3] = 1
			case "bootstrap":
				seed.work[10] = plain.limits[2][10] + 1
			case "decoded-zero":
				seed.work[2] = 0
			case "written-zero":
				seed.work[9] = 0
			case "work-zero":
				seed.work[10] = 0
			case "combined-work":
				seed.work[10] = plain.limits[2][10] - first.work[10] + 1
			case "combined-disk":
				seed.retained = min(plain.live-first.live, plain.retained-first.retained) + 1
			}
			if got, err := newCandidateAdmissionLedgerSeeded(p, intent, first, ledgerTestBounds(), nil, &seed); err == nil || got != nil {
				t.Fatal("initial ticket bypassed a combined charged/reserved bound", mode)
			}
		})
	}
}

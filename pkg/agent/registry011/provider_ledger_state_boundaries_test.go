package registry011

import "testing"

func TestProviderLedgerSealByteTotalsIncludeAuxiliaryAndRejectWraparound(t *testing.T) {
	state := &CandidateAdmissionLedgerChange{metadata: 3, privateMetadata: 5, payload: 7, privatePayload: 11}
	metadata, payload, err := providerLedgerStateBytes(state)
	if err != nil || metadata != 8 || payload != 18 {
		t.Fatal("auxiliary byte totals omitted", err)
	}
	for _, mode := range []string{"metadata", "payload"} {
		copy := *state
		if mode == "metadata" {
			copy.metadata = ^uint64(0)
		} else {
			copy.payload = ^uint64(0)
		}
		if a, b, err := providerLedgerStateBytes(&copy); err == nil || a != 0 || b != 0 {
			t.Fatal("inert byte total wrapped", mode)
		}
	}
}

// Finite parser/arithmetic state only: these cases do not write a frontier,
// qualify a principal, renew a clock or supply an execution acknowledgment.
func TestProviderPrivateLedgerInvalidStateCannotAcquireWorkingLimits(t *testing.T) {
	for _, mode := range []string{"public-bounds", "pending", "final", "records", "private-bytes", "final-outstanding"} {
		t.Run(mode, func(t *testing.T) {
			m, _ := providerPrivateLedgerFixture(t, 8, 32)
			switch mode {
			case "public-bounds":
				m.bounds.MaximumRecords = 0
			case "pending":
				m.continuation.pending = &providerLedgerBoundary{}
			case "final":
				m.continuation.final = true
			case "records":
				m.current.counts[0] = m.continuation.bounds.maximumRecords + 1
			case "private-bytes":
				m.current.privateMetadata++
			case "final-outstanding":
				m.continuation.final, m.continuation.page = true, 1
			}
			if _, _, _, ok := m.workingLimits(); ok {
				t.Fatal("unusable ledger supplied working limits", mode)
			}
		})
	}
	var absent *CandidateAdmissionLedger
	if absent.addProviderRetention(nil, 1, 1) == nil {
		t.Fatal("absent ledger accepted storage")
	}
	if _, err := absent.prepareProviderBoundary(false); err == nil {
		t.Fatal("absent ledger prepared a boundary")
	}
	if absent.applyProviderBoundary(nil, nil, nil, [3]string{}) == nil {
		t.Fatal("absent ledger applied a boundary")
	}
}

func TestProviderPrivateLedgerConstructorDoesNotIgnoreInitialCapacity(t *testing.T) {
	m, policy := providerPrivateLedgerFixture(t, 8, 32)
	b := m.continuation.bounds
	if _, err := newProviderContinuingLedger(nil, m.intent, m.lastTicket, b); err == nil {
		t.Fatal("missing policy constructed a ledger")
	}
	b.maximumResidentBytes = 1
	if !b.valid() {
		t.Fatal("finite resident-cap case must remain structurally valid")
	}
	if _, err := newProviderContinuingLedger(policy, m.intent, m.lastTicket, b); err == nil {
		t.Fatal("initial resident capacity was ignored")
	}
}

func TestProviderPrivateLedgerPageRequiresExactRecordsAndFundedCodecSpace(t *testing.T) {
	for _, mode := range []string{"nil", "invalid-owner", "resident", "future-metadata", "operation-overflow", "ordinal", "kind", "entry", "node", "budget"} {
		t.Run(mode, func(t *testing.T) {
			m, _ := providerPrivateLedgerFixture(t, 8, 32)
			switch mode {
			case "nil":
				m = nil
			case "invalid-owner":
				m.owner = nil
			case "resident":
				m.continuation.bounds.maximumResidentBytes = 1
			case "future-metadata":
				m.continuation.bounds.pageMetadataBytes = m.current.metadata
			case "operation-overflow":
				m.current.sequence = ^uint64(0)
			case "ordinal":
				m.continuation.records[0].ordinal++
			case "kind":
				m.continuation.records[0].kind = "unknown"
			case "entry":
				m.continuation.records[0].entry = "invalid"
			case "node":
				m.continuation.records[0].node = "invalid"
			case "budget":
				m.continuation.records[0].budget = "invalid"
			}
			if _, err := m.providerPage(); err == nil {
				t.Fatal("invalid/unfunded page encoded", mode)
			}
		})
	}
}

func TestProviderPrivateLedgerSealRequiresExactPendingOwnership(t *testing.T) {
	for _, mode := range []string{"nil", "index", "used", "auxiliary", "unmatched-seal", "missing-seal", "page", "changed-current", "final-outstanding", "short-nonfinal"} {
		t.Run(mode, func(t *testing.T) {
			m, policy := providerPrivateLedgerFixture(t, 2, 4)
			if _, err := m.AppendCandidateTicket(ledgerTestNext(t, policy, m.lastTicket, 0)); err != nil {
				t.Fatal(err)
			}
			b, err := m.prepareProviderBoundary(false)
			if err != nil {
				t.Fatal(err)
			}
			heads := [3]string{admissionTestDigest("a"), admissionTestDigest("b"), admissionTestDigest("c")}
			switch mode {
			case "nil":
				b = nil
			case "index":
				heads[1] = "invalid"
			case "used":
				b.used = true
			case "auxiliary":
				b.auxiliaryAtBuild++
			case "unmatched-seal":
				b.sealDigest = admissionTestDigest("f")
			case "missing-seal":
				b.sealMetadata = 1
			case "page":
				b.page++
			case "changed-current":
				current := *m.current
				current.payload++
				m.current = &current
			case "final-outstanding":
				b.final = true
			case "short-nonfinal":
				m.continuation.bounds.pageRecords++
			}
			if _, err := b.sealObject(heads); err == nil {
				t.Fatal("unowned or inconsistent pending seal encoded", mode)
			}
		})
	}
}

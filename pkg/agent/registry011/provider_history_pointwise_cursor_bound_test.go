package registry011

import "testing"

func TestProviderHistoryPointwiseCursorKeepsLargerIncomingCopyDelta(t *testing.T) {
	l := providerHistoryCollectionCursorLimits{Spine: 1, PendingBlocks: 1, Transactions: 16384, EvidenceBytes: 65536, BodyBytes: 516}
	kinds := providerHistoryAppliedStateKinds{ConfirmedEmpty: true}
	one, ok := providerHistoryAppliedCursorUpper(l, 0, kinds)
	got, valid := (providerHistoryCollectionStateRootCalls{2, 1}).pointwisePartsUpper(l, 0, kinds)
	if !ok || !valid {
		t.Fatal("retained array versus full incoming copy precursor")
	}
	for coordinate := range got.headroom {
		root := one.envelopeCommitment[coordinate] - one.normalCommitment[coordinate]
		copied := one.envelopeCopy[coordinate] - one.normalCopy[coordinate]
		if got.headroom[coordinate] != max(root, copied) || got.normal[coordinate] != one.normalCommitment[coordinate]+one.normalCopy[coordinate] {
			t.Fatal("copy-specific incoming headroom was lost", coordinate)
		}
	}
	if one.envelopeCopy[10]-one.normalCopy[10] <= one.envelopeCommitment[10]-one.normalCommitment[10] {
		t.Fatal("fixture did not exercise the larger full-array copy delta")
	}
}

func TestProviderHistoryPointwiseCursorWorkKeepsChildAndScopeParts(t *testing.T) {
	for _, emptyState := range []bool{false, true} {
		kinds := providerHistoryAppliedStateKinds{ConfirmedEmpty: emptyState}
		limits := providerHistoryCollectionCursorLimits{Spine: 1, Transactions: 1, EvidenceBytes: 1, BodyBytes: 1}
		leftCalls, rightCalls := providerHistoryCollectionStateRootCalls{8, 2}, providerHistoryCollectionStateRootCalls{9, 0}
		left, ok := leftCalls.pointwisePartsUpper(limits, 0, kinds)
		one, child := providerHistoryAppliedCursorUpper(limits, 0, kinds)
		if !ok || !child {
			t.Fatal("finite conditional original-child parts")
		}
		for i := range left.normal {
			if one.envelopeCommitment[i] < one.normalCommitment[i] || one.envelopeCopy[i] < one.normalCopy[i] ||
				left.normal[i] != 6*one.normalCommitment[i]+2*one.normalCopy[i] ||
				left.headroom[i] != one.envelopeCommitment[i]-one.normalCommitment[i] ||
				left.headroom[i] != one.envelopeCopy[i]-one.normalCopy[i] {
				t.Fatal("copy root counted twice or original envelope delta lost", i)
			}
		}
		// Numeric prospective bands, not authenticated state/phase selection.
		// The second declared source bound deliberately has a larger spine.
		limits.Spine = 17
		right, valid := rightCalls.pointwisePartsUpper(limits, 0, kinds)
		joined, combined := left.joinWithinScope(right)
		upper, funded := joined.actualAndIncomingUpper()
		if !valid || !combined || !funded {
			t.Fatal("same-scope component composition")
		}
		for i := range upper {
			if joined.normal[i] != left.normal[i]+right.normal[i] || joined.headroom[i] != max(left.headroom[i], right.headroom[i]) ||
				upper[i] != left.normal[i]+right.normal[i]+max(left.headroom[i], right.headroom[i]) {
				t.Fatal("normal bands or first-false headroom changed accounting origin", i)
			}
		}
		if joined.headroom[10] == 0 || joined.headroom[10] >= left.headroom[10]+right.headroom[10] {
			t.Fatal("one-scope actual headroom was summed as repeated standalone envelopes")
		}
	}
}

func TestProviderHistoryPointwiseCursorWorkFitsEveryCounterPrefix(t *testing.T) {
	// Finite counter DATA: six normals10 and two full-copy normals15.
	// Complete incoming/closure envelopes110/115 share the difference100.
	// This uses the original meter primitive, not actual cursor/I/O readers,
	// native costs, a live owner or a production reservation.
	parts := providerHistoryCursorPointwiseParts{normal: candidateWorkVector{10: 90}, headroom: candidateWorkVector{10: 100}}
	upper, ok := parts.actualAndIncomingUpper()
	if !ok || upper != (candidateWorkVector{10: 190}) {
		t.Fatal("independent90+100 source-grammar illustration")
	}
	for stop := range 8 {
		m := providerHistoryContentUnitMeter(upper)
		var prefix uint64
		for i := 0; i <= stop; i++ {
			normal, envelope := uint64(10), uint64(110)
			if i >= 6 {
				normal, envelope = 15, 115
			}
			before := m.actual
			if !m.preflightWork(candidateWorkVector{10: envelope}) || m.actual != before {
				t.Fatal("incoming preflight charged actual or exceeded original prefix", stop, i)
			}
			charge := normal
			if i == stop {
				charge = envelope // Abstract COMPLETE first-false closure, then stop.
			}
			if !m.add(candidateWorkVector{10: charge}) {
				t.Fatal("same original counter could not fund its complete closure", stop, i)
			}
			prefix += charge
		}
		if m.failed || m.actual[10] != prefix || prefix > 190 || m.upper != upper {
			t.Fatal("counter scope reset, partial charge or understated closure", stop)
		}
	}
	// Select a separate insufficient subject BEFORE observation. Never edit an
	// entered upper. The last incoming envelope needs75+115=190, not189.
	m := providerHistoryContentUnitMeter(candidateWorkVector{10: 189})
	for i := range 7 {
		normal, envelope := uint64(10), uint64(110)
		if i >= 6 {
			normal, envelope = 15, 115
		}
		if !m.preflightWork(candidateWorkVector{10: envelope}) || !m.add(candidateWorkVector{10: normal}) {
			t.Fatal("late-refusal counter precursor")
		}
	}
	if m.actual[10] != 75 || m.preflightWork(candidateWorkVector{10: 115}) || !m.failed || m.actual[10] != 75 ||
		m.add(candidateWorkVector{}) || m.upper != (candidateWorkVector{10: 189}) {
		t.Fatal("insufficient incoming work was accepted, refunded or reopened")
	}
}

func TestProviderHistoryPointwiseActualDoesNotReplaceChargedStageUppers(t *testing.T) {
	// Two symbolic stages illustrate the DIFFERENT original ledger contract.
	// Their whole actual root component is180+100=280, but complete stage
	// allowances190+190 remain charged as380. Neither is an operating quote.
	parts := providerHistoryCursorPointwiseParts{normal: candidateWorkVector{10: 90}, headroom: candidateWorkVector{10: 100}}
	whole, ok := parts.joinWithinScope(parts)
	actual, valid := whole.actualAndIncomingUpper()
	if !ok || !valid || actual[10] != 280 {
		t.Fatal("two-stage actual-component precursor")
	}
	u := historyRecordUpper(1)
	u.execution[10], u.custody[10] = 90, 100
	if w, valid := u.work(); !valid || !u.valid(1) || w[10] != 190 {
		t.Fatal("complete symbolic admitted stage upper")
	}
	for _, ceiling := range []uint64{280, 380} {
		maximum := historyLedgerMaximum()
		maximum.work[10] = ceiling // Immutable selection before any addition.
		first, accepted := (providerHistoryGroupTotals{}).add(u, maximum)
		if !accepted || first.work[10] != 190 {
			t.Fatal("first whole upper was not charged")
		}
		second, accepted := first.add(u, maximum)
		if ceiling == 280 {
			if accepted || second != (providerHistoryGroupTotals{}) || first.work[10] != 190 {
				t.Fatal("actual-only bound replaced irreversible stage-upper sum")
			}
		} else if !accepted || second.work[10] != 380 || first.work[10] != 190 {
			t.Fatal("complete charged sums or original value receiver changed")
		}
	}
	// The complete initializer, nonroot rows, native resources and actual+new
	// work check remain additional obligations; this scalar test proves none.
}

func TestProviderHistoryPointwiseCursorWorkRejectsPartialOverflow(t *testing.T) {
	left := providerHistoryCursorPointwiseParts{normal: candidateWorkVector{2: 1, 10: ^uint64(0)}, headroom: candidateWorkVector{2: 2}}
	right := providerHistoryCursorPointwiseParts{normal: candidateWorkVector{2: 1, 10: 1}, headroom: candidateWorkVector{2: 3}}
	if p, ok := left.joinWithinScope(right); ok || p != (providerHistoryCursorPointwiseParts{}) {
		t.Fatal("late normal overflow exposed earlier nonzero parts")
	}
	left.headroom[10] = 1
	if p, ok := left.actualAndIncomingUpper(); ok || p != (candidateWorkVector{}) {
		t.Fatal("late incoming overflow exposed earlier decoded work")
	}
	l := providerHistoryCollectionCursorLimits{Spine: 1, Transactions: 1, EvidenceBytes: 1, BodyBytes: 1}
	for _, calls := range []providerHistoryCollectionStateRootCalls{{}, {1, 2}} {
		if p, ok := calls.pointwisePartsUpper(l, 0, providerHistoryAppliedStateKinds{ConfirmedEmpty: true}); ok || p != (providerHistoryCursorPointwiseParts{}) {
			t.Fatal("empty or invalid band exposed unused headroom")
		}
	}
	l.EvidenceBytes = 0
	if p, ok := (providerHistoryCollectionStateRootCalls{1, 0}).pointwisePartsUpper(l, 0, providerHistoryAppliedStateKinds{}); ok || p != (providerHistoryCursorPointwiseParts{}) {
		t.Fatal("invalid original child exposed partial parts")
	}
	l = providerHistoryCollectionCursorLimits{Spine: 1 << 20, Effects: 163840, PendingBlocks: 10, Transactions: 16384, EvidenceBytes: 64 << 20, BodyBytes: 8 << 20}
	l.Semantic = providerHistorySemanticCommitmentLimits{Window: true, Queue: 1, Pending: true}
	calls, ok := providerHistoryCollectionTerminalStateCallsBound(349525, 11184800, 0, 349525)
	if p, valid := calls.pointwisePartsUpper(l, 8<<20, providerHistoryAppliedStateKinds{}); !ok || valid || p != (providerHistoryCursorPointwiseParts{}) {
		t.Fatal("late composed overflow exposed earlier headroom or normal work")
	}
}

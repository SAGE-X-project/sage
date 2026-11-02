package registry011

import (
	"context"
	"encoding/json"
	"testing"
)

// This is typed transcript DATA with an inert codec meter. It is not a live
// original reader or a sufficient resource certificate. Separate existing
// genuine owned first-P/recovery tests cover the two actual initial readers.
func historyCollectionLedger(t *testing.T, c *providerHistoryCollectionCursor) *providerHistoryGroupLedger {
	t.Helper()
	maximum := historyLedgerMaximum()
	maximum.work[0] = 128
	l, err := newProviderHistoryCollectionLedgerObserved(c.selection.program, admissionTestDigest("b"), admissionTestDigest("c"), admissionTestDigest("e"), 64, maximum, c, c.semantic.observer)
	if err != nil || l == nil || !l.validObserved(providerReplayUnitMeter()) || l.collection == c || &l.collection.spine[0] == &c.spine[0] {
		t.Fatal("owned independent collection ledger DATA", err)
	}
	return l
}

func historyCollectionPendingSettlement(t *testing.T, l *providerHistoryGroupLedger, headers []FinalizedHeader, methods *providerMethodBounds) (*providerHistoryGroupSettlement, []providerHistoryMemberResult) {
	t.Helper()
	c := l.collection
	table := historyCollectionBoundaryPlan(t, c, methods)
	results := historyCollectionBoundaryResults(t, c, headers, table)
	ticket, err := newProviderHistoryGroupTicket(l.program, l.profile, l.costs, l.head, admissionTestDigest("d"), table, historyRecordUpper(table.count))
	if err != nil || l.admitOriginalsObserved(ticket, methods, providerReplayUnitMeter()) != nil {
		t.Fatal("exact original-shaped T", err)
	}
	consumption, err := newProviderHistoryGroupConsumption(ticket, l.head, admissionTestDigest("1"))
	if err != nil || l.consumeObserved(consumption, providerReplayUnitMeter()) != nil {
		t.Fatal("all-member C", err)
	}
	digests := make([]string, len(results))
	for i, result := range results {
		digests[i] = result.digest
	}
	s, err := newProviderHistoryGroupSettlement(ticket, consumption, l.head, admissionTestDigest("2"), "complete", "", digests)
	if err != nil {
		t.Fatal("complete ordered S", err)
	}
	return s, results
}

func TestProviderHistoryCollectionLedgerCommitsSeparateOriginalShapedTransitions(t *testing.T) {
	c, headers, methods := historyCollectionBoundaryFixture(t, 13, 4)
	other, _, _ := historyCollectionBoundaryFixture(t, 13, 4)
	a, b := historyCollectionLedger(t, c), historyCollectionLedger(t, other)
	initial := a.head
	if a.head != b.head || a.collection == b.collection || a.collection.semantic.observer == b.collection.semantic.observer {
		t.Fatal("separate initial reconstructed models differ or alias")
	}
	sa, ra := historyCollectionPendingSettlement(t, a, headers, methods)
	sb, rb := historyCollectionPendingSettlement(t, b, headers, methods)
	if sa.digest != sb.digest || a.head != b.head {
		t.Fatal("independent original-shaped T/C prefixes differ")
	}
	charged := a.charged
	if a.settleOriginalsObserved(context.Background(), sa, ra, methods, a.collection.semantic.observer) != nil || b.settleOriginalsObserved(context.Background(), sb, rb, methods, b.collection.semantic.observer) != nil ||
		!a.validObserved(providerReplayUnitMeter()) || !b.validObserved(providerReplayUnitMeter()) || a.head != b.head || a.head == initial || a.charged != charged || a.collection.completed != 1 || a.collection.phase != "ancestry" || a.stage != 1 {
		t.Fatal("actual result transition or full charged upper missing from root")
	}
	if c.completed != 0 || c.phase != "open" || c.client != "" || &a.collection.spine[0] == &b.collection.spine[0] {
		t.Fatal("settlement mutated an earlier or other independent model")
	}
	copy := providerHistoryLedgerCopyObserved(a, a.collection.semantic.observer)
	if copy == nil || copy.collection == a.collection || copy.head != a.head || &copy.collection.spine[0] == &a.collection.spine[0] {
		t.Fatal("ledger copy shared mutable collection state")
	}
	copy.collection.next--
	if copy.validObserved(providerReplayUnitMeter()) || !a.validObserved(providerReplayUnitMeter()) || a.head != b.head {
		t.Fatal("cursor changed without changing the committed ledger root")
	}
	if providerHistoryLedgerCopy(a) != nil {
		t.Fatal("collection ledger copied through an unobserved legacy path")
	}
}

func TestProviderHistoryCollectionLedgerRefusesHashOnlyOrChangedResultsWithoutRefund(t *testing.T) {
	for _, mode := range []string{"hash-only", "missing-result", "digest", "semantic-result", "nil-methods", "wrong-stage"} {
		t.Run(mode, func(t *testing.T) {
			c, headers, methods := historyCollectionBoundaryFixture(t, 13, 4)
			l := historyCollectionLedger(t, c)
			s, results := historyCollectionPendingSettlement(t, l, headers, methods)
			before, charged := l.head, l.charged
			cursor := l.collection.commitmentObserved(providerReplayUnitMeter())
			var err error
			switch mode {
			case "hash-only":
				err = l.settleObserved(s, providerReplayUnitMeter())
			case "missing-result":
				err = l.settleOriginalsObserved(context.Background(), s, results[:len(results)-1], methods, l.collection.semantic.observer)
			case "digest":
				results[0].digest = admissionTestDigest("f")
				err = l.settleOriginalsObserved(context.Background(), s, results, methods, l.collection.semantic.observer)
			case "semantic-result":
				results[0].result = `"0x1"`
				err = l.settleOriginalsObserved(context.Background(), s, results, methods, l.collection.semantic.observer)
			case "nil-methods":
				err = l.settleOriginalsObserved(context.Background(), s, results, nil, l.collection.semantic.observer)
			case "wrong-stage":
				l.collection.completed++
				err = l.settleOriginalsObserved(context.Background(), s, results, methods, l.collection.semantic.observer)
			}
			if err == nil || !l.failed || l.head != before || l.charged != charged || l.stage != 0 {
				t.Fatal("refused collection S changed a head or refunded its T upper")
			}
			if mode != "wrong-stage" && l.collection.commitmentObserved(providerReplayUnitMeter()) != cursor {
				t.Fatal("refused collection S mutated the earlier successful state")
			}
		})
	}
}

func TestProviderHistoryCollectionLedgerRequiresExactNextTableBeforeChargingT(t *testing.T) {
	for _, mode := range []string{"request", "purpose", "legacy-path", "nil-methods"} {
		t.Run(mode, func(t *testing.T) {
			c, _, methods := historyCollectionBoundaryFixture(t, 13, 4)
			l := historyCollectionLedger(t, c)
			table := historyCollectionBoundaryPlan(t, c, methods)
			requests := make([]*providerIntegrationRequest, table.count)
			for i := range requests {
				r := table.members[i].request
				requests[i] = &r
			}
			purpose := table.purpose
			if mode == "request" {
				requests[0] = historyGroupRequest(t, "web3_clientVersion", []any{})
			}
			if mode == "purpose" {
				purpose = "content"
			}
			changed, err := newProviderHistoryGroupTable(l.program, purpose, 1, requests, *methods, c.selection.input.capture)
			if err != nil {
				t.Fatal("valid alternate DATA table precursor", err)
			}
			ticket, err := newProviderHistoryGroupTicket(l.program, l.profile, l.costs, l.head, admissionTestDigest("d"), changed, historyRecordUpper(table.count))
			if err != nil {
				t.Fatal("valid typed ticket precursor", err)
			}
			before, charge := l.head, l.charged
			if mode == "legacy-path" {
				err = l.admitObserved(ticket, providerReplayUnitMeter())
			} else {
				if mode == "nil-methods" {
					methods = nil
				}
				err = l.admitOriginalsObserved(ticket, methods, providerReplayUnitMeter())
			}
			if err == nil || !l.failed || l.head != before || l.charged != charge || l.ticket != nil {
				t.Fatal("different next table reached T debit")
			}
		})
	}
}

func TestProviderHistoryCollectionFailedSettlementKeepsPriorSuccessfulCursor(t *testing.T) {
	c, headers, methods := historyCollectionBoundaryFixture(t, 13, 4)
	l := historyCollectionLedger(t, c)
	_, _ = historyCollectionPendingSettlement(t, l, headers, methods)
	before, charged := l.collection.commitmentObserved(providerReplayUnitMeter()), l.charged
	s, err := newProviderHistoryGroupSettlement(l.ticket, l.consumption, l.head, admissionTestDigest("2"), "failed", admissionTestDigest("f"), nil)
	if err != nil || l.settleOriginalsObserved(context.Background(), s, nil, nil, l.collection.semantic.observer) != nil || !l.terminalDataValidObserved(providerReplayUnitMeter()) || !l.failed || !l.terminal || l.validObserved(providerReplayUnitMeter()) ||
		l.charged != charged || l.collection.completed != 0 || l.collection.commitmentObserved(providerReplayUnitMeter()) != before {
		t.Fatal("failed S advanced successful grammar or renewed a charged row", err)
	}
}

func TestProviderHistoryCollectionLedgerRefusesNoninitialOrChangedInputAtConstruction(t *testing.T) {
	for _, mode := range []string{"completed", "phase", "next", "spine", "client", "program", "input"} {
		t.Run(mode, func(t *testing.T) {
			c, _, _ := historyCollectionBoundaryFixture(t, 13, 4)
			switch mode {
			case "completed":
				c.completed++
			case "phase":
				c.phase = "ancestry"
			case "next":
				c.next--
			case "spine":
				c.spine = append(c.spine, c.spine[0])
			case "client":
				c.client = "inert prepopulated identity"
			case "program":
				c.selection.program, c.sealed.program = admissionTestDigest("f"), admissionTestDigest("f")
			case "input":
				c.selection.input.traceTimeout++
				c.sealed = c.selection
			}
			l, err := newProviderHistoryCollectionLedgerObserved(admissionTestDigest("a"), admissionTestDigest("b"), admissionTestDigest("c"), admissionTestDigest("e"), 64, historyLedgerMaximum(), c, c.semantic.observer)
			if err == nil || l != nil {
				t.Fatal("inert later/changed state replaced a fresh original first-P seed")
			}
		})
	}
}

func TestProviderHistoryCollectionLedgerKeepsIndependentRootsAcrossAllAncestryCycles(t *testing.T) {
	c, headers, methods := historyCollectionBoundaryFixture(t, 71, 64)
	// Explicit inert DATA capacity chosen before either ledger is constructed.
	// No production bound, meter actual, clock, hold or authority is changed.
	maximum := historyLedgerMaximum()
	for i := range maximum.work {
		maximum.work[i] *= 32
	}
	maximum.work[0] = 2048
	maximum.physical *= 32
	maximum.retainedRaw *= 32
	maximum.payload *= 32
	maximum.metadata *= 32
	maximum.objects *= 32
	maximum.files *= 32
	maximum.clocks *= 32
	makeLedger := func(input *providerHistoryCollectionCursor) *providerHistoryGroupLedger {
		l, err := newProviderHistoryCollectionLedgerObserved(input.selection.program, admissionTestDigest("b"), admissionTestDigest("c"), admissionTestDigest("e"), 64, maximum, input, input.semantic.observer)
		if err != nil {
			t.Fatal("independent finite transcript DATA", err)
		}
		return l
	}
	other, _, _ := historyCollectionBoundaryFixture(t, 71, 64)
	a, b := makeLedger(c), makeLedger(other)
	// Two original windows: 64 parents in 32+32, then the remaining 6.
	counts := []int{18, 32, 32, 7, 18, 6, 7}
	phases := []string{"ancestry", "ancestry", "close", "open", "ancestry", "close", "open"}
	var requests uint64
	for i, count := range counts {
		prior := a.collection
		priorRoot := prior.commitmentObserved(providerReplayUnitMeter())
		sa, ra := historyCollectionPendingSettlement(t, a, headers, methods)
		sb, rb := historyCollectionPendingSettlement(t, b, headers, methods)
		if sa.count != count || sb.count != count || sa.digest != sb.digest || a.head != b.head {
			t.Fatal("handwritten ordered group differs", i)
		}
		if a.settleOriginalsObserved(context.Background(), sa, ra, methods, a.collection.semantic.observer) != nil || b.settleOriginalsObserved(context.Background(), sb, rb, methods, b.collection.semantic.observer) != nil {
			t.Fatal("valid independent DATA cycle refused", i)
		}
		requests += uint64(count)
		n := uint64(i + 1)
		want := providerHistoryGroupTotals{
			work:     candidateWorkVector{requests, 128 * n, 12 * n, n, n, n, n, n, n, 288 * n, 1152 * n},
			physical: 8192 * n, retainedRaw: 1024 * n, payload: 256 * n, metadata: 128 * n, objects: 4 * n, files: 18 * n, clocks: 17 * n,
		}
		if !a.validObserved(providerReplayUnitMeter()) || !b.validObserved(providerReplayUnitMeter()) || a.head != b.head || a.head == sa.previous ||
			a.stage != n || b.stage != n || a.sequence != 3*n || b.sequence != 3*n || a.charged != want || b.charged != want ||
			a.collection.completed != n || b.collection.completed != n || a.collection.phase != phases[i] || b.collection.phase != phases[i] ||
			a.collection == b.collection || &a.collection.spine[0] == &b.collection.spine[0] || prior == a.collection || prior.commitmentObserved(providerReplayUnitMeter()) != priorRoot {
			t.Fatal("later cycle lost exact commitment, independent state or a full cumulative charge", i)
		}
	}
	if requests != 120 || a.collection.mode != "content" || a.collection.phase != "open" || a.collection.next != sepoliaDeploymentBlock || len(a.collection.spine) != 71 ||
		c.completed != 0 || c.phase != "open" || len(c.spine) != 1 {
		t.Fatal("two closed ancestry windows changed original selection or skipped content identity")
	}
}

func TestProviderHistoryCollectionLedgerLateDecodeQuotaPreservesPendingPrefix(t *testing.T) {
	prepare := func() (*providerHistoryGroupLedger, *providerHistoryGroupSettlement, []providerHistoryMemberResult, *providerMethodBounds) {
		c, headers, methods := historyCollectionBoundaryFixture(t, 13, 12)
		l := historyCollectionLedger(t, c)
		first, firstResults := historyCollectionPendingSettlement(t, l, headers, methods)
		if l.settleOriginalsObserved(context.Background(), first, firstResults, methods, l.collection.semantic.observer) != nil || l.collection.phase != "ancestry" {
			t.Fatal("confirmed opening precursor")
		}
		s, results := historyCollectionPendingSettlement(t, l, headers, methods)
		return l, s, results, methods
	}
	l, s, results, methods := prepare()
	if len(results) != 12 || l.stage != 1 || l.sequence != 5 {
		t.Fatal("complete twelve-parent pending DATA precursor")
	}
	measured := l.collection.semantic.observer
	probe := providerHistoryLedgerCopyObserved(l, measured)
	start := measured.actual[2]
	if probe == nil || probe.settleOriginalsObserved(context.Background(), s, results, methods, measured) != nil || !probe.validObserved(providerReplayUnitMeter()) || probe.collection.phase != "close" {
		t.Fatal("sufficient positive probe")
	}
	decoded := measured.actual[2] - start
	// There are no decoded-byte events after the last header's timestamp hex
	// codec. Its 8 digits are independent of the earlier encode preflights.
	var last map[string]string
	if json.Unmarshal([]byte(results[len(results)-1].result), &last) != nil || len(last["timestamp"]) != 10 || decoded < 8 {
		t.Fatal("last timestamp precursor does not have eight hex digits")
	}
	for _, mode := range []string{"exact-decoded", "last-timestamp-one-short"} {
		t.Run(mode, func(t *testing.T) {
			subject, settlement, actualResults, actualMethods := prepare()
			if subject == nil || settlement.digest != s.digest {
				t.Fatal("separate pending prefix")
			}
			before, charged := subject.head, subject.charged
			cursor := subject.collection.commitmentObserved(providerReplayUnitMeter())
			meter := subject.collection.semantic.observer
			previousDecoded := meter.actual[2]
			// This inert DATA limit leaves all original consumed work intact.
			// A fresh semantic execution observer is never installed mid-prefix.
			meter.upper[2] = previousDecoded + decoded
			if mode == "last-timestamp-one-short" {
				meter.upper[2]--
			}
			err := subject.settleOriginalsObserved(context.Background(), settlement, actualResults, actualMethods, meter)
			if mode == "exact-decoded" {
				if err != nil || subject.head != probe.head || subject.charged != charged || meter.actual[2]-previousDecoded != decoded || meter.failed || !subject.validObserved(providerReplayUnitMeter()) {
					t.Fatal("exact observed decode boundary failed", err)
				}
			} else if err == nil || !meter.failed || !subject.failed || subject.head != before || subject.charged != charged || subject.stage != 1 || subject.sequence != 5 ||
				subject.collection.commitmentObserved(providerReplayUnitMeter()) != cursor || meter.actual[2]-previousDecoded != decoded-8 || subject.ticket == nil || subject.consumption == nil {
				t.Fatal("late timestamp refusal changed pending state, refunded the upper, or stopped at an earlier preflight")
			}
		})
	}
	if l.failed || l.stage != 1 || l.sequence != 5 || l.collection.completed != 1 {
		t.Fatal("one test copy poisoned another pending prefix")
	}
}

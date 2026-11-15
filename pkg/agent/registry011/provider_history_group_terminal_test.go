package registry011

import "testing"

func historyTerminalDataFixture(t *testing.T, completed int) *providerHistoryGroupLedger {
	t.Helper()
	l := historyLedgerFixture(t, historyLedgerMaximum())
	ticket := historyRecordTicket(t, 2, 1, l.head)
	if l.admit(ticket) != nil {
		t.Fatal("genuine typed T")
	}
	c, err := newProviderHistoryGroupConsumption(ticket, l.head, admissionTestDigest("1"))
	if err != nil || l.consume(c) != nil {
		t.Fatal("genuine typed C", err)
	}
	digests := make([]string, completed)
	for i := range digests {
		digests[i] = admissionTestDigest("2")
	}
	s, err := newProviderHistoryGroupSettlement(ticket, c, l.head, admissionTestDigest("3"), "failed", admissionTestDigest("4"), digests)
	if err != nil || l.settle(s) != nil {
		t.Fatal("genuine failed S DATA, not a filesystem ACK", err)
	}
	return l
}

func TestProviderHistoryTerminalDataNeverReopensLiveTransitions(t *testing.T) {
	for completed := range 3 {
		l := historyTerminalDataFixture(t, completed)
		before := *l
		m := historyGroupCodecMeter()
		if !l.terminalDataValidObserved(m) || l.validObserved(nil) || *l != before || m.failed || m.actual[10] == 0 || m.actual[0] != 0 || m.actual[1] != 0 || m.clocks != 0 {
			t.Fatal("failed replay DATA changed state or became a live ledger", completed)
		}
		if l.admit(nil) == nil || l.consume(nil) == nil || l.settle(nil) == nil || *l != before || !l.terminalDataValidObserved(nil) {
			t.Fatal("terminal validation renewed authority or refunded the original debit")
		}
	}
}

func TestProviderHistoryTerminalDataRejectsMalformedShapesWithAConsistentHash(t *testing.T) {
	original := historyTerminalDataFixture(t, 1)
	for _, mode := range []string{"copied-owner", "not-failed", "not-terminal", "wrong-phase", "program", "profile", "costs", "initial", "previous", "entry", "stage-zero", "stage-excess", "sequence", "maximum", "uncharged", "overcharged", "ticket", "consumption", "changed-root"} {
		t.Run(mode, func(t *testing.T) {
			data := *original
			data.owner = &data // Independent DATA instance; never a live role.
			switch mode {
			case "copied-owner":
				data.owner = original
			case "not-failed":
				data.failed = false
			case "not-terminal":
				data.terminal = false
			case "wrong-phase":
				data.phase = "settled"
			case "program":
				data.program = ""
			case "profile":
				data.profile = ""
			case "costs":
				data.costs = ""
			case "initial":
				data.initial = ""
			case "previous":
				data.previous = ""
			case "entry":
				data.entry = ""
			case "stage-zero":
				data.stage = 0
			case "stage-excess":
				data.stage = data.maximumStages + 1
			case "sequence":
				data.sequence++
			case "maximum":
				data.maximum.work[0] = 0
			case "uncharged":
				data.charged = providerHistoryGroupTotals{}
			case "overcharged":
				data.charged.work[0] = data.maximum.work[0] + 1
			case "ticket":
				data.ticket = &providerHistoryGroupTicket{}
			case "consumption":
				data.consumption = &providerHistoryGroupConsumption{}
			case "changed-root":
				data.head = admissionTestDigest("f")
			}
			if mode != "changed-root" {
				data.head = providerReplayDigest(providerHistoryGroupRootDomain, data.root(), nil)
			}
			before := data
			m := historyGroupCodecMeter()
			if data.terminalDataValidObserved(m) || data != before {
				t.Fatal("inconsistent terminal DATA was accepted or mutated", mode)
			}
			if mode != "changed-root" && m.actual != (candidateWorkVector{}) {
				t.Fatal("malformed terminal shape reached the first codec", mode)
			}
			if mode == "changed-root" && (m.actual[10] == 0 || m.failed || m.closed) {
				t.Fatal("changed root skipped observed comparison or poisoned a healthy DATA observer")
			}
		})
	}
}

func TestProviderHistoryTerminalDataObservationSeparatesAdmissionUpperFromActualWork(t *testing.T) {
	l := historyTerminalDataFixture(t, 0)
	before := *l
	probe := historyGroupCodecMeter()
	if !l.terminalDataValidObserved(probe) {
		t.Fatal("genuine failed DATA precursor")
	}
	needed := probe.actual
	// Canonical encoding preflights its admitted size, not the shorter actual
	// root. Actual work alone cannot be used as a sufficient invocation upper.
	square, productOK := providerUpperProduct(candidateLedgerMetadataLimit, candidateLedgerMetadataLimit)
	admission, sumOK := providerUpperSum(square, 1)
	if !productOK || !sumOK || admission == 0 {
		t.Fatal("finite original canonical preflight upper")
	}
	upper := needed
	upper[10] = max(upper[10], admission)
	sufficient := newProviderWorkMeter(&providerIntegration{}, providerOperationReservation{work: upper, execution: upper})
	if !l.terminalDataValidObserved(sufficient) || sufficient.actual != needed || sufficient.failed || *l != before {
		t.Fatal("sufficient admission changed exact observed DATA work")
	}
	// This is the first encode preflight boundary, not a later actual-work or
	// last-comparison proof. A separate original observer must refuse untouched.
	upper[10] = admission - 1
	short := newProviderWorkMeter(&providerIntegration{}, providerOperationReservation{work: upper, execution: upper})
	if l.terminalDataValidObserved(short) || !short.failed || short.actual != (candidateWorkVector{}) || *l != before || l.valid() {
		t.Fatal("short canonical admission was reset or consumed as actual work")
	}
}

package registry011

import (
	"encoding/json"
	"testing"
)

func historyLedgerMaximum() providerHistoryGroupTotals {
	u := historyRecordUpper(1)
	w, _ := u.work()
	for i := range w {
		w[i] *= 2
	}
	return providerHistoryGroupTotals{w, 2 * u.physical, 2 * u.retainedRaw, 2 * u.payload, 2 * u.metadata, 2 * u.objects, 2 * u.files, 2 * u.clocks}
}

func historyLedgerFixture(t *testing.T, maximum providerHistoryGroupTotals) *providerHistoryGroupLedger {
	t.Helper()
	l, err := newProviderHistoryGroupLedger(admissionTestDigest("a"), admissionTestDigest("b"), admissionTestDigest("c"), admissionTestDigest("e"), 2, maximum)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func historyLedgerComplete(t *testing.T, l *providerHistoryGroupLedger, result string) *providerHistoryGroupSettlement {
	t.Helper()
	ticket := historyRecordTicket(t, 1, l.stage+1, l.head)
	if err := l.admit(ticket); err != nil {
		t.Fatal("normal T refused", err)
	}
	c, err := newProviderHistoryGroupConsumption(ticket, l.head, admissionTestDigest("1"))
	if err != nil || l.consume(c) != nil {
		t.Fatal("normal C refused", err)
	}
	s, err := newProviderHistoryGroupSettlement(ticket, c, l.head, admissionTestDigest("2"), "complete", "", []string{result})
	if err != nil || l.settle(s) != nil {
		t.Fatal("normal S refused", err)
	}
	return s
}

func TestProviderHistoryGroupLedgerReplaysTwoIndependentOriginalTranscripts(t *testing.T) {
	a, b := historyLedgerFixture(t, historyLedgerMaximum()), historyLedgerFixture(t, historyLedgerMaximum())
	if a == b || a.head != b.head || !a.valid() || !b.valid() {
		t.Fatal("separate deterministic replay models were not constructed")
	}
	for stage := uint64(1); stage <= 2; stage++ {
		s := historyLedgerComplete(t, a, admissionTestDigest("3"))
		other := historyLedgerComplete(t, b, admissionTestDigest("3"))
		if a.head != b.head || s.digest != other.digest || a.entry != s.digest || a.phase != "settled" || !a.valid() || !b.valid() || a.stage != stage || a.sequence != 3*stage || a.ticket != nil || a.consumption != nil {
			t.Fatal("independent typed prefix roots disagree", stage)
		}
	}
	if a.charged != a.maximum || b.charged != b.maximum {
		t.Fatal("successful settlement discounted an admitted upper")
	}
	charged, head := a.charged, a.head
	extra := historyRecordTicket(t, 1, 3, head)
	if a.admit(extra) == nil || !a.failed || a.charged != charged || a.head != head {
		t.Fatal("maximum program stage was renewed")
	}
}

func TestProviderHistoryGroupLedgerBindsSuccessfulSettlementOriginal(t *testing.T) {
	a, b := historyLedgerFixture(t, historyLedgerMaximum()), historyLedgerFixture(t, historyLedgerMaximum())
	historyLedgerComplete(t, a, admissionTestDigest("3"))
	historyLedgerComplete(t, b, admissionTestDigest("4"))
	if !a.valid() || !b.valid() || a.head == b.head || a.entry == b.entry {
		t.Fatal("different final member originals produced the same replay root")
	}
}

func TestProviderHistoryGroupLedgerChargesEveryDimensionBeforeT(t *testing.T) {
	base := historyLedgerMaximum()
	for dimension := range base.work {
		maximum := base
		maximum.work[dimension]--
		l := historyLedgerFixture(t, maximum)
		historyLedgerComplete(t, l, admissionTestDigest("3"))
		charged, head := l.charged, l.head
		if l.admit(historyRecordTicket(t, 1, 2, head)) == nil || !l.failed || l.charged != charged || l.head != head {
			t.Fatal("exhausted work coordinate advanced a group", dimension)
		}
	}
	for _, resource := range []string{"physical", "raw", "payload", "metadata", "objects", "files", "clocks"} {
		maximum := base
		switch resource {
		case "physical":
			maximum.physical--
		case "raw":
			maximum.retainedRaw--
		case "payload":
			maximum.payload--
		case "metadata":
			maximum.metadata--
		case "objects":
			maximum.objects--
		case "files":
			maximum.files--
		case "clocks":
			maximum.clocks--
		}
		l := historyLedgerFixture(t, maximum)
		historyLedgerComplete(t, l, admissionTestDigest("3"))
		charged, head := l.charged, l.head
		if l.admit(historyRecordTicket(t, 1, 2, head)) == nil || l.charged != charged || l.head != head {
			t.Fatal("exhausted separate resource advanced a group", resource)
		}
	}
}

func TestProviderHistoryGroupLedgerFailureDoesNotRestoreUnusedPermissions(t *testing.T) {
	l := historyLedgerFixture(t, historyLedgerMaximum())
	ticket := historyRecordTicket(t, 1, 1, l.head)
	if l.admit(ticket) != nil {
		t.Fatal("normal T refused")
	}
	c, err := newProviderHistoryGroupConsumption(ticket, l.head, admissionTestDigest("1"))
	if err != nil || l.consume(c) != nil {
		t.Fatal("normal C refused", err)
	}
	charged := l.charged
	s, err := newProviderHistoryGroupSettlement(ticket, c, l.head, admissionTestDigest("2"), "failed", admissionTestDigest("3"), nil)
	if err != nil || l.settle(s) != nil || !l.failed || !l.terminal || l.phase != "failed" || l.entry != s.digest || l.sequence != 3 || l.charged != charged {
		t.Fatal("failed group did not retain its complete charge and terminal record", err)
	}
	head := l.head
	if l.admit(ticket) == nil || l.consume(c) == nil || l.settle(s) == nil || l.head != head || l.charged != charged || l.valid() {
		t.Fatal("failure was retried, refunded or treated as successful progress")
	}
}

func TestProviderHistoryGroupLedgerRefusesCrossProgramAndOutOfOrderRecords(t *testing.T) {
	for _, row := range []string{"program", "profile", "costs", "previous", "stage", "consume-before-T", "settle-before-C", "T-twice", "C-twice", "S-previous", "S-count", "S-raw"} {
		l := historyLedgerFixture(t, historyLedgerMaximum())
		ticket := historyRecordTicket(t, 1, 1, l.head)
		var refusal error
		switch row {
		case "program", "profile", "costs", "previous", "stage":
			copy := *ticket
			switch row {
			case "program":
				copy.program = admissionTestDigest("f")
			case "profile":
				copy.profile = admissionTestDigest("f")
			case "costs":
				copy.costs = admissionTestDigest("f")
			case "previous":
				copy.previous = admissionTestDigest("f")
			case "stage":
				copy.table.stage = 2
			}
			// Construct a valid record for the OTHER context. Merely damaging
			// a digest would test canonicality instead of these scope guards.
			bounds, global := historyGroupBounds()
			table, err := newProviderHistoryGroupTable(copy.program, "content", copy.table.stage, []*providerIntegrationRequest{historyGroupSimple(t)}, bounds, global)
			if err != nil {
				t.Fatal(err)
			}
			foreign, err := newProviderHistoryGroupTicket(copy.program, copy.profile, copy.costs, copy.previous, copy.inventory, table, copy.upper)
			if err != nil || !providerHistoryGroupTicketValid(foreign) {
				t.Fatal("independent other-context record could not be prepared", err)
			}
			refusal = l.admit(foreign)
		case "consume-before-T":
			c, _ := newProviderHistoryGroupConsumption(ticket, l.head, admissionTestDigest("1"))
			refusal = l.consume(c)
		default:
			if l.admit(ticket) != nil {
				t.Fatal("normal T refused")
			}
			c, _ := newProviderHistoryGroupConsumption(ticket, l.head, admissionTestDigest("1"))
			switch row {
			case "T-twice":
				refusal = l.admit(ticket)
			case "settle-before-C":
				s, _ := newProviderHistoryGroupSettlement(ticket, c, l.head, admissionTestDigest("2"), "complete", "", []string{admissionTestDigest("3")})
				refusal = l.settle(s)
			default:
				if l.consume(c) != nil {
					t.Fatal("normal C refused")
				}
				s, _ := newProviderHistoryGroupSettlement(ticket, c, l.head, admissionTestDigest("2"), "complete", "", []string{admissionTestDigest("3")})
				switch row {
				case "C-twice":
					refusal = l.consume(c)
				case "S-previous":
					s.previous = admissionTestDigest("f")
					refusal = l.settle(s)
				case "S-count":
					s.count = providerHistoryGroupMembers + 1
					refusal = l.settle(s)
				case "S-raw":
					s.raw += " "
					refusal = l.settle(s)
				}
			}
		}
		if refusal == nil || !l.failed || l.valid() {
			t.Fatal("unbound or out-of-order transition was accepted", row)
		}
	}
}

func TestProviderHistoryGroupLedgerKeepsUnfinishedMembersConsumedAfterMiddleFailure(t *testing.T) {
	maximum := historyLedgerMaximum()
	maximum.work[0] = 4
	l := historyLedgerFixture(t, maximum)
	ticket := historyRecordTicket(t, 2, 1, l.head)
	if l.admit(ticket) != nil {
		t.Fatal("normal two-member T refused")
	}
	c, err := newProviderHistoryGroupConsumption(ticket, l.head, admissionTestDigest("1"))
	if err != nil || l.consume(c) != nil || c.consumed != 3 {
		t.Fatal("both members were not consumed together", err)
	}
	charged := l.charged
	s, err := newProviderHistoryGroupSettlement(ticket, c, l.head, admissionTestDigest("2"), "failed", admissionTestDigest("3"), []string{admissionTestDigest("4")})
	if err != nil || l.settle(s) != nil || !l.terminal || l.charged != charged || l.charged.work[0] != 2 || c.consumed != 3 || s.count != 1 {
		t.Fatal("one completed member restored the unused member's charge", err)
	}
	if l.admit(historyRecordTicket(t, 1, 2, l.head)) == nil || l.consume(c) == nil {
		t.Fatal("middle failure permitted another group or consumption")
	}
}

func TestProviderHistoryGroupLedgerRefusesMissingAndInconsistentInitialBounds(t *testing.T) {
	for _, field := range []string{"program", "profile", "costs", "initial", "stages-zero", "stages-excessive", "work", "physical", "raw", "payload", "metadata", "objects", "files", "clocks"} {
		program, profile, costs, initial := admissionTestDigest("a"), admissionTestDigest("b"), admissionTestDigest("c"), admissionTestDigest("e")
		stages, maximum := uint64(2), historyLedgerMaximum()
		switch field {
		case "program":
			program = ""
		case "profile":
			profile = ""
		case "costs":
			costs = ""
		case "initial":
			initial = ""
		case "stages-zero":
			stages = 0
		case "stages-excessive":
			stages = 1 << 20
		case "work":
			maximum.work[0] = 0
		case "physical":
			maximum.physical = 0
		case "raw":
			maximum.retainedRaw = maximum.physical + 1
		case "payload":
			maximum.payload = maximum.retainedRaw + 1
		case "metadata":
			maximum.metadata = maximum.retainedRaw + 1
		case "objects":
			maximum.objects = 0
		case "files":
			maximum.files = maximum.objects - 1
		case "clocks":
			maximum.clocks = 0
		}
		if l, err := newProviderHistoryGroupLedger(program, profile, costs, initial, stages, maximum); err == nil || l != nil {
			t.Fatal("unbound or inconsistent initial ceiling created a replay model", field)
		}
	}
}

func TestProviderHistoryGroupLedgerOwnsCopiesAndRejectsReconstruction(t *testing.T) {
	l := historyLedgerFixture(t, historyLedgerMaximum())
	copy := *l
	if copy.valid() || copy.admit(historyRecordTicket(t, 1, 1, l.head)) == nil || l.failed {
		t.Fatal("copied model became the original live replay instance")
	}
	ticket := historyRecordTicket(t, 1, 1, l.head)
	if l.admit(ticket) != nil {
		t.Fatal("normal T refused")
	}
	ticket.table.members[0].request.wire = "changed caller DATA"
	ticket.upper.execution[0] = 2
	if !l.valid() || l.ticket.table.members[0].request.wire == ticket.table.members[0].request.wire || l.ticket.upper.execution[0] != 1 {
		t.Fatal("caller replaced the model's immutable current ticket")
	}
	c, _ := newProviderHistoryGroupConsumption(l.ticket, l.head, admissionTestDigest("1"))
	if l.consume(c) != nil {
		t.Fatal("normal C refused")
	}
	c.consumed = 0
	if !l.valid() || l.consumption.consumed != 1 || json.Unmarshal([]byte(`{}`), l) == nil {
		t.Fatal("caller or JSON reconstructed the original consumed state")
	}
	var missing *providerHistoryGroupLedger
	if missing.valid() || missing.admit(nil) == nil || missing.consume(nil) == nil || missing.settle(nil) == nil || missing.refuse() == nil {
		t.Fatal("missing model accepted a transition")
	}
}

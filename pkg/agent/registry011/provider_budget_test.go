package registry011

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func budgetTestPolicy(t *testing.T, edit func(map[string]any)) *ProviderPolicyCandidate {
	t.Helper()
	object := providerPolicyTestObject()
	if edit != nil {
		edit(object["limits"].(map[string]any))
	}
	p, err := ParseProviderPolicyCandidate(providerPolicyTestBytes(t, object), 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func budgetTestModel(t *testing.T, policy *ProviderPolicyCandidate, maximum int) *CandidateBudgetModel {
	t.Helper()
	m, err := NewCandidateBudgetModel(policy, maximum)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func budgetTestSegment(n int) string { return "0x" + strings.Repeat("0", 60) + fixedBudgetHex(n) }

func fixedBudgetHex(n int) string {
	s := strconv.FormatInt(int64(n), 16)
	return strings.Repeat("0", 4-len(s)) + s
}

func budgetTestView(t *testing.T, m *CandidateBudgetModel) CandidateBudgetDiagnostic {
	t.Helper()
	d, err := m.Diagnostic()
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestCandidateBudgetAllWorkDimensions(t *testing.T) {
	w := CandidateWork{Requests: 1, WireBytes: 2, DecodedBytes: 3, Blocks: 4, Transactions: 5, Frames: 6, Logs: 7, Proofs: 8, Segments: 9, DiskBytesWritten: 10, WorkUnits: 11}
	v := candidateWorkVector{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}
	if w.vector() != v || v.diagnostic() != w {
		t.Fatal("work dimension mapping")
	}
	fields := []struct {
		name string
		set  func(*CandidateWork, uint64)
	}{
		{"requests", func(w *CandidateWork, n uint64) { w.Requests = n }},
		{"wire_bytes", func(w *CandidateWork, n uint64) { w.WireBytes = n }},
		{"decoded_bytes", func(w *CandidateWork, n uint64) { w.DecodedBytes = n }},
		{"blocks", func(w *CandidateWork, n uint64) { w.Blocks = n }},
		{"transactions", func(w *CandidateWork, n uint64) { w.Transactions = n }},
		{"frames", func(w *CandidateWork, n uint64) { w.Frames = n }},
		{"logs", func(w *CandidateWork, n uint64) { w.Logs = n }},
		{"proofs", func(w *CandidateWork, n uint64) { w.Proofs = n }},
		{"disk_bytes_written", func(w *CandidateWork, n uint64) { w.DiskBytesWritten = n }},
		{"work_units", func(w *CandidateWork, n uint64) { w.WorkUnits = n }},
	}
	for _, field := range fields {
		t.Run(field.name, func(t *testing.T) {
			policy := budgetTestPolicy(t, func(l map[string]any) {
				l["operation"].(map[string]any)[field.name] = "5"
				l["segment"].(map[string]any)[field.name] = "8"
				l["bootstrap"].(map[string]any)[field.name] = "12"
			})
			m := budgetTestModel(t, policy, 16)
			before := budgetTestView(t, m)
			work := CandidateWork{Segments: 1}
			field.set(&work, 6)
			if ticket, err := m.ReserveCandidate(budgetTestSegment(1), work, 0, 0); ticket != nil || err != ErrCandidateBudgetLimit || budgetTestView(t, m) != before {
				t.Fatal("operation ceiling or mutation on refusal")
			}
			field.set(&work, 5)
			first, err := m.ReserveCandidate(budgetTestSegment(1), work, 0, 0)
			if err != nil {
				t.Fatal(err)
			}
			work.Segments = 0
			field.set(&work, 3)
			if _, err := m.ReserveCandidate(budgetTestSegment(1), work, 0, 0); err != nil {
				t.Fatal("inclusive segment ceiling", err)
			}
			field.set(&work, 1)
			if ticket, err := m.ReserveCandidate(budgetTestSegment(1), work, 0, 0); ticket != nil || err != ErrCandidateBudgetLimit {
				t.Fatal("segment outstanding ignored")
			}
			work.Segments = 1
			field.set(&work, 4)
			if _, err := m.ReserveCandidate(budgetTestSegment(2), work, 0, 0); err != nil {
				t.Fatal("inclusive bootstrap ceiling", err)
			}
			field.set(&work, 1)
			if ticket, err := m.ReserveCandidate(budgetTestSegment(3), work, 0, 0); ticket != nil || err != ErrCandidateBudgetLimit {
				t.Fatal("new segment reset bootstrap")
			}
			if err := m.ChargeCandidate(first, "failed"); err != nil {
				t.Fatal(err)
			}
			work.Segments = 0
			if ticket, err := m.ReserveCandidate(budgetTestSegment(1), work, 0, 0); ticket != nil || err != ErrCandidateBudgetLimit {
				t.Fatal("failed work refunded segment")
			}
			work.Segments = 1
			if ticket, err := m.ReserveCandidate(budgetTestSegment(3), work, 0, 0); ticket != nil || err != ErrCandidateBudgetLimit {
				t.Fatal("charged work refunded bootstrap")
			}
		})
	}
}

func TestCandidateBudgetSegmentIdentityAndBounds(t *testing.T) {
	policy := budgetTestPolicy(t, func(l map[string]any) {
		l["operation"].(map[string]any)["segments"] = "1"
		l["segment"].(map[string]any)["segments"] = "1"
		l["bootstrap"].(map[string]any)["segments"] = "2"
	})
	m := budgetTestModel(t, policy, 4)
	for _, amount := range []uint64{0, 2} {
		if ticket, err := m.ReserveCandidate(budgetTestSegment(1), CandidateWork{Segments: amount}, 0, 0); ticket != nil || err != ErrCandidateBudgetTicket {
			t.Fatal("new segment count")
		}
	}
	first, err := m.ReserveCandidate(budgetTestSegment(1), CandidateWork{Segments: 1}, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if ticket, err := m.ReserveCandidate(budgetTestSegment(1), CandidateWork{Segments: 1}, 0, 0); ticket != nil || err != ErrCandidateBudgetTicket {
		t.Fatal("duplicate segment start")
	}
	if _, err := m.ReserveCandidate(budgetTestSegment(1), CandidateWork{}, 0, 0); err != nil {
		t.Fatal("explicit zero vector", err)
	}
	if _, err := m.ReserveCandidate(budgetTestSegment(2), CandidateWork{Segments: 1}, 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := m.ChargeCandidate(first, "cancelled"); err != nil {
		t.Fatal(err)
	}
	if ticket, err := m.ReserveCandidate(budgetTestSegment(3), CandidateWork{Segments: 1}, 0, 0); ticket != nil || err != ErrCandidateBudgetLimit || budgetTestView(t, m).Segments != 2 {
		t.Fatal("failed segment forgotten")
	}
	for _, segment := range []string{"", "0x1", "0x" + strings.Repeat("A", 64)} {
		if ticket, err := m.ReserveCandidate(segment, CandidateWork{}, 0, 0); ticket != nil || err != ErrConfiguration {
			t.Fatal("invalid segment")
		}
	}
}

func TestCandidateBudgetFailureChargeAndCapacity(t *testing.T) {
	for _, outcome := range []string{"failed", "cancelled", "uncertain"} {
		t.Run(outcome, func(t *testing.T) {
			policy := budgetTestPolicy(t, func(l map[string]any) { l["live_disk_bytes"], l["retained_bytes"] = "10", "6" })
			m := budgetTestModel(t, policy, 8)
			work := CandidateWork{Requests: 1, WireBytes: 2, DecodedBytes: 3, Blocks: 4, Transactions: 5, Frames: 6, Logs: 7, Proofs: 8, Segments: 1, DiskBytesWritten: 9, WorkUnits: 10}
			ticket, err := m.ReserveCandidate(budgetTestSegment(1), work, 6, 4)
			if err != nil {
				t.Fatal(err)
			}
			view := budgetTestView(t, m)
			if view.Outstanding != work || view.Charged != (CandidateWork{}) || view.LiveReservedUpper != 6 || view.RetainedReservedUpper != 4 {
				t.Fatal("reservation not retained in model")
			}
			if err := m.ChargeCandidate(ticket, outcome); err != nil {
				t.Fatal(err)
			}
			charged := budgetTestView(t, m)
			if charged.Charged != work || charged.Outstanding != (CandidateWork{}) || charged.LiveUsedUpper != 6 || charged.LiveReservedUpper != 0 || charged.RetainedUsedUpper != 4 || charged.RetainedReservedUpper != 0 {
				t.Fatal("failure reduced work or capacity")
			}
			if err := m.ChargeCandidate(ticket, outcome); err != nil || budgetTestView(t, m) != charged {
				t.Fatal("idempotent charge")
			}
			for _, other := range []string{"failed", "cancelled", "uncertain", "complete", ""} {
				if other != outcome {
					if err := m.ChargeCandidate(ticket, other); err != ErrCandidateBudgetTicket || budgetTestView(t, m) != charged {
						t.Fatal("conflicting or unmeasured outcome")
					}
				}
			}
			if _, err := m.ReserveCandidate(budgetTestSegment(1), CandidateWork{Requests: 1}, 4, 2); err != nil {
				t.Fatal("inclusive capacity bounds", err)
			}
			full := budgetTestView(t, m)
			if ticket, err := m.ReserveCandidate(budgetTestSegment(1), CandidateWork{}, 1, 0); ticket != nil || err != ErrCandidateBudgetLimit || budgetTestView(t, m) != full {
				t.Fatal("live capacity reused after failure")
			}
			full.Policy, full.Charged.WorkUnits = "changed", 0
			if got := budgetTestView(t, m); got.Policy != policy.policy || got.Charged != work {
				t.Fatal("diagnostic mutation affected model")
			}
		})
	}
	policy := budgetTestPolicy(t, func(l map[string]any) { l["live_disk_bytes"], l["retained_bytes"] = "100", "2" })
	m := budgetTestModel(t, policy, 8)
	if _, err := m.ReserveCandidate(budgetTestSegment(1), CandidateWork{Segments: 1}, 2, 2); err != nil {
		t.Fatal(err)
	}
	if ticket, err := m.ReserveCandidate(budgetTestSegment(1), CandidateWork{}, 1, 1); ticket != nil || err != ErrCandidateBudgetLimit {
		t.Fatal("retained capacity ignored")
	}
	if ticket, err := m.ReserveCandidate(budgetTestSegment(1), CandidateWork{}, 1, 2); ticket != nil || err != ErrConfiguration {
		t.Fatal("retained exceeds peak")
	}
}

func TestCandidateBudgetOverflowAndLifetimeBound(t *testing.T) {
	const maximum = ^uint64(0)
	for _, parts := range [][4]uint64{{maximum, 0, 0, maximum}, {0, maximum, 0, maximum}, {0, 0, maximum, maximum}, {maximum - 2, 1, 1, maximum}} {
		if !candidateCapacity(parts[0], parts[1], parts[2], parts[3]) {
			t.Fatal("inclusive arithmetic boundary")
		}
	}
	for _, parts := range [][4]uint64{{maximum, 0, 1, maximum}, {0, maximum, 1, maximum}, {maximum, maximum, 0, maximum}, {2, 0, 0, 1}, {0, 2, 0, 1}} {
		if candidateCapacity(parts[0], parts[1], parts[2], parts[3]) {
			t.Fatal("wrapped arithmetic accepted")
		}
	}
	policy := budgetTestPolicy(t, func(l map[string]any) {
		for _, level := range []string{"operation", "segment", "bootstrap"} {
			l[level].(map[string]any)["work_units"] = strconv.FormatUint(maximum, 10)
		}
		l["live_disk_bytes"], l["retained_bytes"] = strconv.FormatUint(maximum, 10), strconv.FormatUint(maximum, 10)
	})
	m := budgetTestModel(t, policy, 8)
	ticket, err := m.ReserveCandidate(budgetTestSegment(1), CandidateWork{Segments: 1, WorkUnits: maximum}, maximum, maximum)
	if err != nil {
		t.Fatal(err)
	}
	if r, err := m.ReserveCandidate(budgetTestSegment(1), CandidateWork{WorkUnits: 1}, 0, 0); r != nil || err != ErrCandidateBudgetLimit {
		t.Fatal("outstanding work overflow")
	}
	if err := m.ChargeCandidate(ticket, "uncertain"); err != nil {
		t.Fatal(err)
	}
	for _, proposal := range []struct{ work, live, retained uint64 }{{1, 0, 0}, {0, 1, 0}, {0, 1, 1}} {
		if r, err := m.ReserveCandidate(budgetTestSegment(1), CandidateWork{WorkUnits: proposal.work}, proposal.live, proposal.retained); r != nil || err != ErrCandidateBudgetLimit {
			t.Fatal("charged work/capacity overflow")
		}
	}
	limited := budgetTestModel(t, budgetTestPolicy(t, nil), 1)
	if _, err := limited.ReserveCandidate(budgetTestSegment(1), CandidateWork{Segments: 1}, 0, 0); err != nil {
		t.Fatal(err)
	}
	for n := 2; n <= 6; n++ {
		if r, err := limited.ReserveCandidate(budgetTestSegment(n), CandidateWork{Segments: 1}, 0, 0); r != nil || err != ErrCandidateBudgetLimit || budgetTestView(t, limited).Segments != 1 {
			t.Fatal("model memory limit or refused segment allocation")
		}
	}
}

func TestCandidateBudgetZeroForeignAndJSONRefuse(t *testing.T) {
	p := budgetTestPolicy(t, nil)
	for _, policy := range []*ProviderPolicyCandidate{nil, {}} {
		if m, err := NewCandidateBudgetModel(policy, 1); m != nil || err != ErrConfiguration {
			t.Fatal("invalid policy")
		}
	}
	for _, n := range []int{-1, 0, 4097} {
		if m, err := NewCandidateBudgetModel(p, n); m != nil || err != ErrConfiguration {
			t.Fatal("implicit model bound")
		}
	}
	m := budgetTestModel(t, p, 4096)
	for _, zero := range []*CandidateBudgetModel{nil, {}, {owner: m, policy: m.policy, maxTickets: m.maxTickets, segments: m.segments}} {
		if r, err := zero.ReserveCandidate(budgetTestSegment(1), CandidateWork{Segments: 1}, 0, 0); r != nil || err != ErrCandidateBudget {
			t.Fatal("zero or copied owner")
		}
		if err := zero.ChargeCandidate(nil, "failed"); err != ErrCandidateBudget {
			t.Fatal("zero charge")
		}
		if _, err := zero.Diagnostic(); err != ErrCandidateBudget {
			t.Fatal("zero diagnostic")
		}
	}
	other := budgetTestModel(t, p, 4)
	foreign, err := other.ReserveCandidate(budgetTestSegment(1), CandidateWork{Segments: 1}, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, ticket := range []*CandidateBudgetTicket{nil, {}, foreign, {owner: m, index: -1}, {owner: m, index: 0}} {
		if err := m.ChargeCandidate(ticket, "failed"); err != ErrCandidateBudgetTicket {
			t.Fatal("invalid ticket owner/index")
		}
	}
	for _, value := range []any{m, &CandidateBudgetTicket{}} {
		if err := json.Unmarshal([]byte(`{"ready":true,"refund":100}`), value); err == nil {
			t.Fatal("JSON created private model state")
		}
	}
	if view := budgetTestView(t, m); view.Tickets != 0 || view.Charged != (CandidateWork{}) || view.Outstanding != (CandidateWork{}) {
		t.Fatal("rejected JSON changed model")
	}
}

func TestCandidateBudgetConcurrentReservationsAndCharges(t *testing.T) {
	policy := budgetTestPolicy(t, func(l map[string]any) {
		l["operation"].(map[string]any)["requests"] = "1"
		l["segment"].(map[string]any)["requests"] = "10"
		l["bootstrap"].(map[string]any)["requests"] = "10"
	})
	m := budgetTestModel(t, policy, 32)
	first, err := m.ReserveCandidate(budgetTestSegment(1), CandidateWork{Requests: 1, Segments: 1}, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	tickets := make(chan *CandidateBudgetTicket, 32)
	tickets <- first
	errors := make(chan error, 31)
	var wg sync.WaitGroup
	for range 31 {
		wg.Go(func() {
			ticket, err := m.ReserveCandidate(budgetTestSegment(1), CandidateWork{Requests: 1}, 0, 0)
			if ticket != nil {
				tickets <- ticket
			}
			errors <- err
		})
	}
	wg.Wait()
	close(tickets)
	close(errors)
	refused := 0
	for err := range errors {
		switch err {
		case nil:
		case ErrCandidateBudgetLimit:
			refused++
		default:
			t.Fatal(err)
		}
	}
	if view := budgetTestView(t, m); refused != 22 || view.Tickets != 10 || view.Outstanding.Requests != 10 || view.Charged.Requests != 0 {
		t.Fatal("concurrent reservation exceeded limit")
	}
	for ticket := range tickets {
		for range 2 {
			wg.Go(func() {
				if err := m.ChargeCandidate(ticket, "uncertain"); err != nil {
					t.Error(err)
				}
			})
		}
	}
	wg.Wait()
	if view := budgetTestView(t, m); view.Charged.Requests != 10 || view.Outstanding != (CandidateWork{}) || view.Segments != 1 {
		t.Fatal("concurrent charge doubled or refunded")
	}
}

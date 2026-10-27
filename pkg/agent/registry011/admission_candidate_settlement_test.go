package registry011

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strconv"
	"testing"
)

// Inert records only. These test clocks, hashes and proposed actual_work values
// are never submitted as owned measurements, custody or operating authority.
func ledgerTestClock(ticket *CandidateAdmissionTicket, delta int64) map[string]any {
	clock := ticket.intent.clock
	return map[string]any{"id": clock.ClockID, "epoch": clock.Epoch, "wall_ms": strconv.FormatInt(clock.WallMS+delta, 10), "monotonic_ms": strconv.FormatInt(clock.MonotonicMS+delta, 10), "uncertainty_ms": "0"}
}

func ledgerTestSettlementObject(ticket *CandidateAdmissionTicket, outcome string, delta int64) map[string]any {
	var work, retained any
	if outcome == "complete" {
		work = contentJSON(ticket.CanonicalBytes()).(map[string]any)["reservation"]
		retained = admissionTestDigest("a")
	}
	return map[string]any{"format": "sage-provider-budget-settlement-v2", "ticket_digest": ticket.digest, "outcome": outcome, "actual_work": work, "retained_root": retained, "clock_end": ledgerTestClock(ticket, delta)}
}

func ledgerTestSettlement(t *testing.T, ticket *CandidateAdmissionTicket, outcome string, delta int64) *CandidateAdmissionSettlement {
	t.Helper()
	s, err := ParseCandidateAdmissionSettlement(ticket, providerPolicyTestBytes(t, ledgerTestSettlementObject(ticket, outcome, delta)), 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestAdmissionCandidateSettlementOutcomesAndCopies(t *testing.T) {
	p, i, m := admissionTestTicketFixture(t)
	ticket := admissionTestTicket(t, p, i, 0, m)
	for _, outcome := range []string{"complete", "failed", "cancelled", "uncertain"} {
		t.Run(outcome, func(t *testing.T) {
			raw := providerPolicyTestBytes(t, ledgerTestSettlementObject(ticket, outcome, 20))
			s, err := ParseCandidateAdmissionSettlement(ticket, raw, 64<<10)
			if err != nil {
				t.Fatal(err)
			}
			d, err := s.Diagnostic()
			if err != nil || d.Digest != enrollmentTestHash("sage-provider-budget-settlement-v2", raw) || d.Ticket != ticket.digest || d.Outcome != outcome || d.CandidateClock.WallMS != 1000020 {
				t.Fatal(d, err)
			}
			if reflect.TypeOf(d.CandidateClock).AssignableTo(reflect.TypeOf(ClockSample{})) {
				t.Fatal("JSON clock became trusted input")
			}
			raw[0] = '!'
			copy := s.CanonicalBytes()
			copy[0] = '!'
			if bytes.Equal(copy, s.CanonicalBytes()) {
				t.Fatal("mutable settlement bytes")
			}
		})
	}
	for _, outcome := range []string{"failed", "cancelled", "uncertain"} {
		// Expired recovery charges in full; it grants no new admission.
		ledgerTestSettlement(t, ticket, outcome, 2000)
	}
	ledgerTestSettlement(t, ticket, "complete", 1000)
	m["kind"], m["method"], m["params_digest"] = "local-work", nil, nil
	m["reservation"].(map[string]any)["requests"] = "0"
	local := admissionTestTicket(t, p, i, 0, m)
	ledgerTestSettlement(t, local, "complete", 20)
}

func TestAdmissionCandidateSettlementClosedFields(t *testing.T) {
	p, i, m := admissionTestTicketFixture(t)
	ticket := admissionTestTicket(t, p, i, 0, m)
	for key := range ledgerTestSettlementObject(ticket, "complete", 20) {
		for _, mode := range []string{"missing", "wrong"} {
			t.Run(key+"/"+mode, func(t *testing.T) {
				s := ledgerTestSettlementObject(ticket, "complete", 20)
				if mode == "missing" {
					delete(s, key)
				} else {
					s[key] = "invalid"
				}
				if got, err := ParseCandidateAdmissionSettlement(ticket, providerPolicyTestBytes(t, s), 64<<10); err == nil || got != nil {
					t.Fatal("invalid settlement accepted")
				}
			})
		}
	}
	for _, mode := range []string{"extra", "null", "array", "noncanonical", "late-complete", "foreign-epoch", "foreign-id", "wall-back", "monotonic-back", "wall-drift", "monotonic-drift", "uncertainty", "segment-zero", "disk-zero", "work-zero", "request-zero", "wire-zero", "failed-actual", "failed-root"} {
		t.Run(mode, func(t *testing.T) {
			s := ledgerTestSettlementObject(ticket, "complete", 20)
			w := s["actual_work"].(map[string]any)
			clock := s["clock_end"].(map[string]any)
			var value any = s
			switch mode {
			case "extra":
				s["permit"] = true
			case "null":
				value = nil
			case "array":
				value = []any{}
			case "late-complete":
				s["clock_end"] = ledgerTestClock(ticket, 1001)
			case "foreign-epoch":
				clock["epoch"] = admissionTestDigest("b")
			case "foreign-id":
				clock["id"] = "foreign"
			case "wall-back":
				clock["wall_ms"] = "999999"
			case "monotonic-back":
				clock["monotonic_ms"] = "99"
			case "wall-drift":
				clock["wall_ms"] = "1000021"
			case "monotonic-drift":
				clock["monotonic_ms"] = "121"
			case "uncertainty":
				clock["uncertainty_ms"] = "1001"
			case "segment-zero":
				w["segments"] = "0"
			case "disk-zero":
				w["disk_bytes_written"] = "0"
			case "work-zero":
				w["work_units"] = "0"
			case "request-zero":
				w["requests"] = "0"
			case "wire-zero":
				w["wire_bytes"] = "0"
			case "failed-actual":
				s["outcome"], s["retained_root"] = "failed", nil
			case "failed-root":
				s["outcome"], s["actual_work"] = "failed", nil
			}
			raw := admissionTestBytes(t, value)
			if mode == "noncanonical" {
				raw = append(raw, '\n')
			}
			if got, err := ParseCandidateAdmissionSettlement(ticket, raw, 64<<10); err == nil || got != nil {
				t.Fatal("invalid settlement relation accepted", mode)
			}
		})
	}
	for name := range ledgerTestSettlementObject(ticket, "complete", 20)["actual_work"].(map[string]any) {
		t.Run("actual-over-reservation/"+name, func(t *testing.T) {
			s := ledgerTestSettlementObject(ticket, "complete", 20)
			s["actual_work"].(map[string]any)[name] = "2"
			if got, err := ParseCandidateAdmissionSettlement(ticket, providerPolicyTestBytes(t, s), 64<<10); err == nil || got != nil {
				t.Fatal("actual work exceeded reservation")
			}
		})
	}
}

func TestAdmissionCandidateSettlementClockAndArguments(t *testing.T) {
	p, i, m := admissionTestTicketFixture(t)
	ticket := admissionTestTicket(t, p, i, 0, m)
	object := ledgerTestSettlementObject(ticket, "complete", 20)
	raw := providerPolicyTestBytes(t, object)
	for _, bound := range []int{0, 64<<10 + 1, len(raw) - 1} {
		if got, err := ParseCandidateAdmissionSettlement(ticket, raw, bound); err == nil || got != nil {
			t.Fatal("invalid bound")
		}
	}
	for _, candidate := range []*CandidateAdmissionTicket{nil, {}} {
		if _, err := ParseCandidateAdmissionSettlement(candidate, raw, 64<<10); err == nil {
			t.Fatal("invalid ticket")
		}
	}
	if _, err := ParseCandidateAdmissionSettlement(ticket, nil, 64<<10); err == nil {
		t.Fatal("empty input")
	}
	var empty *CandidateAdmissionSettlement
	if empty.CanonicalBytes() != nil {
		t.Fatal("nil bytes")
	}
	if _, err := empty.Diagnostic(); err == nil {
		t.Fatal("nil diagnostic")
	}
	if _, err := (&CandidateAdmissionSettlement{}).Diagnostic(); err == nil {
		t.Fatal("zero diagnostic")
	}
	if json.Unmarshal([]byte(`{}`), &CandidateAdmissionSettlement{}) != ErrAdmissionCandidate {
		t.Fatal("JSON restored a settlement")
	}
	for _, field := range []string{"wall", "monotonic", "uncertainty", "after-uncertainty"} {
		before, after := i.clock, i.clock
		switch field {
		case "wall":
			before.WallMS = -1
		case "monotonic":
			before.MonotonicMS = -1
		case "uncertainty":
			before.UncertaintyMS = -1
		case "after-uncertainty":
			after.UncertaintyMS = -1
		}
		if candidateLedgerClockAdvance(before, after) {
			t.Fatal("negative candidate sample")
		}
	}
}

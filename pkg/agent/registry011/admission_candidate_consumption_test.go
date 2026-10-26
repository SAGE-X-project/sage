package registry011

import (
	"bytes"
	"encoding/json"
	"math"
	"reflect"
	"testing"
)

func admissionTestConsumptionFixture(t *testing.T) (*CandidateAdmissionTicket, map[string]any) {
	t.Helper()
	p, i, m := admissionTestTicketFixture(t)
	ticket := admissionTestTicket(t, p, i, 0, m)
	id := i.identity
	return ticket, map[string]any{"format": "sage-provider-budget-consumption-v1", "ticket_digest": ticket.digest, "namespace": id.Namespace, "instance_key": id.Instance, "source_id": id.SourceID, "generation": "0", "fence": "1", "sequence": "0", "previous_consumption": nil, "kind": ticket.kind, "input_digest": ticket.input, "clock_start": map[string]any{"id": i.clock.ClockID, "epoch": i.clock.Epoch, "wall_ms": "1000010", "monotonic_ms": "110", "uncertainty_ms": "0"}}
}

func admissionTestConsumption(t *testing.T, ticket *CandidateAdmissionTicket, m map[string]any) *CandidateConsumption {
	t.Helper()
	v, err := ParseCandidateConsumption(ticket, providerPolicyTestBytes(t, m), 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestAdmissionCandidateConsumptionBindingAndCopies(t *testing.T) {
	ticket, m := admissionTestConsumptionFixture(t)
	raw := providerPolicyTestBytes(t, m)
	v, err := ParseCandidateConsumption(ticket, raw, 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	d, err := v.Diagnostic()
	if err != nil || d.Digest != enrollmentTestHash("sage-provider-budget-consumption-v1", raw) || d.Ticket != ticket.digest || d.Previous != "" || d.Sequence != 0 || d.CandidateClock.WallMS != 1000010 {
		t.Fatal(d, err)
	}
	if reflect.TypeOf(v.clock).AssignableTo(reflect.TypeOf(ClockSample{})) || reflect.TypeOf(d.CandidateClock).AssignableTo(reflect.TypeOf(ClockSample{})) {
		t.Fatal("candidate clock became a trusted input")
	}
	raw[0] = '!'
	copy := v.CanonicalBytes()
	copy[0] = '!'
	if bytes.Equal(copy, v.CanonicalBytes()) {
		t.Fatal("mutable consumption bytes")
	}
	d.CandidateClock.WallMS = 0
	again, _ := v.Diagnostic()
	if again.CandidateClock.WallMS != 1000010 {
		t.Fatal("mutable clock diagnostic")
	}
}

func TestAdmissionCandidateConsumptionClosedFields(t *testing.T) {
	_, good := admissionTestConsumptionFixture(t)
	for key := range good {
		for _, mode := range []string{"missing", "wrong"} {
			t.Run(key+"/"+mode, func(t *testing.T) {
				ticket, m := admissionTestConsumptionFixture(t)
				if mode == "missing" {
					delete(m, key)
				} else {
					m[key] = "invalid"
				}
				if got, err := ParseCandidateConsumption(ticket, providerPolicyTestBytes(t, m), 64<<10); err == nil || got != nil {
					t.Fatal("invalid consumption accepted")
				}
			})
		}
	}
	for _, mode := range []string{"extra", "generation", "fence", "predecessor", "missing-predecessor", "null", "array", "noncanonical", "clock-id", "clock-epoch", "clock-wall-back", "clock-monotonic-back", "clock-wall-ahead", "clock-monotonic-ahead", "clock-after-deadline", "clock-underflow"} {
		t.Run(mode, func(t *testing.T) {
			ticket, m := admissionTestConsumptionFixture(t)
			clock := m["clock_start"].(map[string]any)
			var value any = m
			switch mode {
			case "extra":
				m["dispatch"] = true
			case "generation":
				m["generation"] = "1"
			case "fence":
				m["fence"] = "2"
			case "predecessor":
				m["previous_consumption"] = admissionTestDigest("a")
			case "missing-predecessor":
				m["sequence"] = "1"
			case "null":
				value = nil
			case "array":
				value = []any{}
			case "clock-id":
				clock["id"] = "foreign"
			case "clock-epoch":
				clock["epoch"] = admissionTestDigest("a")
			case "clock-wall-back":
				clock["wall_ms"] = "999999"
			case "clock-monotonic-back":
				clock["monotonic_ms"] = "99"
			case "clock-wall-ahead":
				clock["wall_ms"] = "1000011"
			case "clock-monotonic-ahead":
				clock["monotonic_ms"] = "111"
			case "clock-after-deadline":
				clock["wall_ms"], clock["monotonic_ms"] = "1001001", "1101"
			case "clock-underflow":
				clock["wall_ms"], clock["uncertainty_ms"] = "0", "1"
			}
			raw := admissionTestBytes(t, value)
			if mode == "noncanonical" {
				raw = append(raw, '\n')
			}
			if got, err := ParseCandidateConsumption(ticket, raw, 64<<10); err == nil || got != nil {
				t.Fatal("invalid consumption relation accepted", mode)
			}
		})
	}
}

func TestAdmissionCandidateConsumptionClockBoundaries(t *testing.T) {
	for _, mode := range []string{"original", "exact-deadline", "uncertainty-wall", "uncertainty-monotonic"} {
		t.Run(mode, func(t *testing.T) {
			ticket, m := admissionTestConsumptionFixture(t)
			clock := m["clock_start"].(map[string]any)
			switch mode {
			case "original":
				clock["wall_ms"], clock["monotonic_ms"] = "1000000", "100"
			case "exact-deadline":
				clock["wall_ms"], clock["monotonic_ms"] = "1001000", "1100"
			case "uncertainty-wall":
				clock["wall_ms"], clock["uncertainty_ms"] = "1000020", "10"
			case "uncertainty-monotonic":
				clock["monotonic_ms"], clock["uncertainty_ms"] = "120", "10"
			}
			admissionTestConsumption(t, ticket, m)
		})
	}
	ticket, _ := admissionTestConsumptionFixture(t)
	// Finite unit arithmetic boundary only; no operating clock is supplied.
	policy := contentJSON([]byte(ticket.policy)).(map[string]any)
	policy["limits"].(map[string]any)["operation"].(map[string]any)["deadline_ms"] = "18446744073709551615"
	if candidateConsumptionClock(ticket, CandidateClockDiagnostic{Epoch: ticket.intent.clock.Epoch, WallMS: 1000010, MonotonicMS: 110}, policy) {
		t.Fatal("operation deadline overflow")
	}
	policy["limits"].(map[string]any)["operation"].(map[string]any)["deadline_ms"] = "1"
	if candidateConsumptionClock(ticket, CandidateClockDiagnostic{Epoch: ticket.intent.clock.Epoch, WallMS: 1000010, MonotonicMS: 110}, policy) {
		t.Fatal("ticket extends operation duration")
	}
}

func TestAdmissionCandidateConsumptionSuccessors(t *testing.T) {
	ticket, m := admissionTestConsumptionFixture(t)
	first := admissionTestConsumption(t, ticket, m)
	p := &ProviderPolicyCandidate{raw: ticket.policy, policy: ticket.intent.identity.Policy}
	nextTicketObject := contentJSON(ticket.CanonicalBytes()).(map[string]any)
	nextTicketObject["sequence"], nextTicketObject["previous_ticket"] = "1", ticket.digest
	nextTicketObject["reservation"].(map[string]any)["segments"] = "0"
	nextTicket := admissionTestTicket(t, p, ticket.intent, 0, nextTicketObject)
	m["sequence"], m["previous_consumption"], m["ticket_digest"] = "1", first.digest, nextTicket.digest
	second := admissionTestConsumption(t, nextTicket, m)
	if CheckCandidateConsumptionSuccessor(first, second) != nil {
		t.Fatal("exact successor refused")
	}
	for _, mode := range []string{"nil-before", "nil-after", "zero", "sequence", "previous", "same-ticket", "foreign-intent", "overflow", "epoch", "wall-back", "monotonic-back"} {
		t.Run(mode, func(t *testing.T) {
			before, after := *first, *second
			left, right := &before, &after
			switch mode {
			case "nil-before":
				left = nil
			case "nil-after":
				right = nil
			case "zero":
				left = &CandidateConsumption{}
			case "sequence":
				after.sequence = 3
			case "previous":
				after.previous = admissionTestDigest("a")
			case "same-ticket":
				after.ticket = ticket
			case "foreign-intent":
				otherIntent := *ticket.intent
				otherIntent.digest = admissionTestDigest("a")
				otherTicket := *nextTicket
				otherTicket.intent = &otherIntent
				after.ticket = &otherTicket
			case "overflow":
				before.sequence = math.MaxUint64
			case "epoch":
				after.clock.Epoch = admissionTestDigest("a")
			case "wall-back":
				after.clock.WallMS--
			case "monotonic-back":
				after.clock.MonotonicMS--
			}
			if CheckCandidateConsumptionSuccessor(left, right) == nil {
				t.Fatal("invalid successor accepted")
			}
		})
	}
}

func TestAdmissionCandidateConsumptionArgumentsAndOpaqueState(t *testing.T) {
	ticket, m := admissionTestConsumptionFixture(t)
	raw := providerPolicyTestBytes(t, m)
	for _, bound := range []int{0, 64<<10 + 1, len(raw) - 1} {
		if got, err := ParseCandidateConsumption(ticket, raw, bound); err == nil || got != nil {
			t.Fatal("invalid byte bound")
		}
	}
	for _, value := range []*CandidateAdmissionTicket{nil, {}} {
		if _, err := ParseCandidateConsumption(value, raw, 64<<10); err == nil {
			t.Fatal("invalid ticket")
		}
	}
	if _, err := ParseCandidateConsumption(ticket, nil, 64<<10); err == nil {
		t.Fatal("empty input")
	}
	var empty *CandidateConsumption
	if empty.CanonicalBytes() != nil {
		t.Fatal("nil bytes")
	}
	if _, err := empty.Diagnostic(); err == nil {
		t.Fatal("nil diagnostic")
	}
	if _, err := (&CandidateConsumption{}).Diagnostic(); err == nil {
		t.Fatal("zero diagnostic")
	}
	if err := json.Unmarshal([]byte(`{}`), &CandidateConsumption{}); err != ErrAdmissionCandidate {
		t.Fatal("JSON restored consumption state")
	}
}

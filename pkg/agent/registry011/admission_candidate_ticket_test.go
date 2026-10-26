package registry011

import (
	"bytes"
	"encoding/json"
	"math"
	"strconv"
	"testing"
)

// All identities, clocks and resource amounts here are inert unit fixtures.
// No parsed record is a durable reservation or permission to perform work.
func admissionTestTicketFixture(t *testing.T) (*ProviderPolicyCandidate, *CandidateAdmissionIntent, map[string]any) {
	return admissionTestTicketPolicyFixture(t, nil)
}

func admissionTestTicketPolicyFixture(t *testing.T, object map[string]any) (*ProviderPolicyCandidate, *CandidateAdmissionIntent, map[string]any) {
	t.Helper()
	p, c, a, m := admissionTestIntentFixture(t, object)
	i, err := ParseCandidateAdmissionIntent(p, c, a, providerPolicyTestBytes(t, m), 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	w := map[string]any{}
	for _, name := range []string{"requests", "wire_bytes", "decoded_bytes", "blocks", "transactions", "frames", "logs", "proofs", "segments", "disk_bytes_written", "work_units"} {
		w[name] = "1"
	}
	id := i.identity
	return p, i, map[string]any{"format": "sage-provider-budget-ticket-v2", "admission_digest": i.digest, "namespace": id.Namespace, "instance_key": id.Instance, "source_id": id.SourceID, "generation": "0", "fence": "1", "sequence": "0", "previous_ticket": nil, "segment": candidateAdmissionSegment(i.digest, 0), "kind": "rpc-capture", "input_digest": admissionTestDigest("4"), "method": "eth_getBlockByNumber", "params_digest": admissionTestDigest("5"), "reservation": w, "live_disk_reserved": "100", "retained_bytes_reserved": "90", "deadline_wall_ms": strconv.FormatUint(i.deadline, 10)}
}

func admissionTestTicket(t *testing.T, p *ProviderPolicyCandidate, i *CandidateAdmissionIntent, segment uint64, m map[string]any) *CandidateAdmissionTicket {
	t.Helper()
	v, err := ParseCandidateAdmissionTicket(p, i, segment, providerPolicyTestBytes(t, m), 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestAdmissionCandidateTicketBindingAndCopies(t *testing.T) {
	p, i, m := admissionTestTicketFixture(t)
	raw := providerPolicyTestBytes(t, m)
	v, err := ParseCandidateAdmissionTicket(p, i, 0, raw, 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	d, err := v.Diagnostic()
	if err != nil || d.Digest != enrollmentTestHash("sage-provider-budget-ticket-v2", raw) || d.Admission != i.digest || d.Sequence != 0 || d.Segment != 0 || d.LiveReservedUpper != 100 || d.RetainedReservedUpper != 90 || d.Reservation.Segments != 1 || d.Method != "eth_getBlockByNumber" {
		t.Fatal(d, err)
	}
	raw[0] = '!'
	copy := v.CanonicalBytes()
	copy[0] = '!'
	if bytes.Equal(copy, v.CanonicalBytes()) {
		t.Fatal("mutable ticket bytes")
	}
	d.Reservation.WorkUnits = 0
	again, _ := v.Diagnostic()
	if again.Reservation.WorkUnits != 1 {
		t.Fatal("mutable reservation diagnostic")
	}
	for _, method := range []string{"eth_chainId", "web3_clientVersion", "eth_getBlockByNumber", "eth_getBlockByHash", "eth_getCode", "eth_call", "eth_getBlockReceipts", "debug_getRawBlock", "debug_getRawReceipts", "debug_traceBlockByHash", "debug_traceTransaction"} {
		m["method"] = method
		admissionTestTicket(t, p, i, 0, m)
	}
	m["kind"], m["method"], m["params_digest"] = "local-work", nil, nil
	m["reservation"].(map[string]any)["requests"] = "0"
	local := admissionTestTicket(t, p, i, 0, m)
	if local.kind != "local-work" || local.method != "" || local.params != "" {
		t.Fatal("local work retained an RPC method")
	}
}

func TestAdmissionCandidateTicketClosedFields(t *testing.T) {
	_, _, good := admissionTestTicketFixture(t)
	for key := range good {
		for _, mode := range []string{"missing", "wrong"} {
			t.Run(key+"/"+mode, func(t *testing.T) {
				p, i, m := admissionTestTicketFixture(t)
				if mode == "missing" {
					delete(m, key)
				} else {
					m[key] = "invalid"
				}
				if got, err := ParseCandidateAdmissionTicket(p, i, 0, providerPolicyTestBytes(t, m), 64<<10); err == nil || got != nil {
					t.Fatal("invalid ticket accepted")
				}
			})
		}
	}
	for _, mode := range []string{"extra", "generation", "fence", "predecessor", "missing-predecessor", "deadline-expired", "deadline-bootstrap", "retained-over-live", "retained-zero-rpc", "retained-zero-local", "live-cap", "retained-cap", "segment-cap", "segment-digest", "segment-first", "segment-count", "disk-zero", "work-zero", "rpc-requests", "rpc-wire", "write-method", "local-requests", "local-method", "local-params", "null", "array", "noncanonical"} {
		t.Run(mode, func(t *testing.T) {
			object := providerPolicyTestObject()
			if mode == "retained-cap" {
				object["limits"].(map[string]any)["retained_bytes"] = "50"
			}
			p, i, m := admissionTestTicketPolicyFixture(t, object)
			w := m["reservation"].(map[string]any)
			segment := uint64(0)
			var value any = m
			switch mode {
			case "extra":
				m["permit"] = true
			case "generation":
				m["generation"] = "1"
			case "fence":
				m["fence"] = "2"
			case "predecessor":
				m["previous_ticket"] = admissionTestDigest("8")
			case "missing-predecessor":
				m["sequence"] = "1"
			case "deadline-expired":
				m["deadline_wall_ms"] = "1000000"
			case "deadline-bootstrap":
				m["deadline_wall_ms"] = strconv.FormatUint(i.deadline+1, 10)
			case "retained-over-live":
				m["retained_bytes_reserved"] = "101"
			case "retained-zero-rpc", "retained-zero-local":
				m["retained_bytes_reserved"] = "0"
				if mode == "retained-zero-local" {
					m["kind"], m["method"], m["params_digest"], w["requests"] = "local-work", nil, nil, "0"
				}
			case "live-cap":
				m["live_disk_reserved"] = "1001"
			case "retained-cap":
				m["retained_bytes_reserved"] = "51"
			case "segment-cap":
				segment = 1000
				m["segment"] = candidateAdmissionSegment(i.digest, segment)
			case "segment-digest":
				m["segment"] = admissionTestDigest("a")
			case "segment-first":
				segment = 1
				m["segment"] = candidateAdmissionSegment(i.digest, segment)
			case "segment-count":
				w["segments"] = "2"
			case "disk-zero":
				w["disk_bytes_written"] = "0"
			case "work-zero":
				w["work_units"] = "0"
			case "rpc-requests":
				w["requests"] = "2"
			case "rpc-wire":
				w["wire_bytes"] = "0"
			case "write-method":
				m["method"] = "eth_sendRawTransaction"
			case "local-requests", "local-method", "local-params":
				m["kind"], m["method"], m["params_digest"] = "local-work", nil, nil
				w["requests"] = "0"
				switch mode {
				case "local-requests":
					w["requests"] = "1"
				case "local-method":
					m["method"] = "eth_chainId"
				case "local-params":
					m["params_digest"] = admissionTestDigest("6")
				}
			case "null":
				value = nil
			case "array":
				value = []any{}
			}
			raw := admissionTestBytes(t, value)
			if mode == "noncanonical" {
				raw = append(raw, '\n')
			}
			if got, err := ParseCandidateAdmissionTicket(p, i, segment, raw, 64<<10); err == nil || got != nil {
				t.Fatal("invalid ticket relation accepted", mode)
			}
		})
	}
}

func TestAdmissionCandidateTicketEveryWorkDimension(t *testing.T) {
	_, _, good := admissionTestTicketFixture(t)
	for key := range good["reservation"].(map[string]any) {
		for _, mode := range []string{"missing", "invalid", "over-limit"} {
			t.Run(key+"/"+mode, func(t *testing.T) {
				p, i, m := admissionTestTicketFixture(t)
				w := m["reservation"].(map[string]any)
				switch mode {
				case "missing":
					delete(w, key)
				case "invalid":
					w[key] = "01"
				case "over-limit":
					w[key] = "1001"
				}
				if got, err := ParseCandidateAdmissionTicket(p, i, 0, providerPolicyTestBytes(t, m), 64<<10); err == nil || got != nil {
					t.Fatal("invalid work accepted")
				}
			})
		}
	}
	if _, ok := candidateAdmissionWork(nil); ok {
		t.Fatal("null work")
	}
	if _, ok := candidateAdmissionWorkLimit(nil); ok {
		t.Fatal("null limit")
	}
}

func TestAdmissionCandidateTicketSuccessors(t *testing.T) {
	p, i, m := admissionTestTicketFixture(t)
	first := admissionTestTicket(t, p, i, 0, m)
	m["sequence"], m["previous_ticket"] = "1", first.digest
	m["reservation"].(map[string]any)["segments"] = "0"
	second := admissionTestTicket(t, p, i, 0, m)
	if err := CheckCandidateTicketSuccessor(first, second); err != nil {
		t.Fatal(err)
	}
	m["sequence"], m["previous_ticket"], m["segment"] = "2", second.digest, candidateAdmissionSegment(i.digest, 1)
	m["reservation"].(map[string]any)["segments"] = "1"
	third := admissionTestTicket(t, p, i, 1, m)
	if err := CheckCandidateTicketSuccessor(second, third); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"nil-before", "nil-after", "zero", "sequence", "previous", "same-segment-recount", "advance-no-count", "segment-skip", "foreign-intent", "overflow"} {
		t.Run(mode, func(t *testing.T) {
			before, after := *first, *second
			left, right := &before, &after
			switch mode {
			case "nil-before":
				left = nil
			case "nil-after":
				right = nil
			case "zero":
				left = &CandidateAdmissionTicket{}
			case "sequence":
				after.sequence = 3
			case "previous":
				after.previous = admissionTestDigest("a")
			case "same-segment-recount":
				after.work[8] = 1
			case "advance-no-count":
				after.segment = 1
			case "segment-skip":
				after.segment, after.work[8] = 2, 1
			case "foreign-intent":
				other := *i
				other.digest = admissionTestDigest("a")
				after.intent = &other
			case "overflow":
				before.sequence = math.MaxUint64
			}
			if CheckCandidateTicketSuccessor(left, right) == nil {
				t.Fatal("invalid successor accepted")
			}
		})
	}
}

func TestAdmissionCandidateTicketArgumentsAndOpaqueState(t *testing.T) {
	p, i, m := admissionTestTicketFixture(t)
	raw := providerPolicyTestBytes(t, m)
	for _, bound := range []int{0, 64<<10 + 1, len(raw) - 1} {
		if got, err := ParseCandidateAdmissionTicket(p, i, 0, raw, bound); err == nil || got != nil {
			t.Fatal("invalid byte bound")
		}
	}
	for _, policy := range []*ProviderPolicyCandidate{nil, {}} {
		if _, err := ParseCandidateAdmissionTicket(policy, i, 0, raw, 64<<10); err == nil {
			t.Fatal("invalid policy")
		}
	}
	for _, intent := range []*CandidateAdmissionIntent{nil, {}} {
		if _, err := ParseCandidateAdmissionTicket(p, intent, 0, raw, 64<<10); err == nil {
			t.Fatal("invalid intent")
		}
	}
	if _, err := ParseCandidateAdmissionTicket(p, i, 0, nil, 64<<10); err == nil {
		t.Fatal("empty input")
	}
	foreign := *i
	foreign.identity.Policy = admissionTestDigest("a")
	if _, err := ParseCandidateAdmissionTicket(p, &foreign, 0, raw, 64<<10); err == nil {
		t.Fatal("foreign policy")
	}
	var empty *CandidateAdmissionTicket
	if empty.CanonicalBytes() != nil {
		t.Fatal("nil bytes")
	}
	if _, err := empty.Diagnostic(); err == nil {
		t.Fatal("nil diagnostic")
	}
	if _, err := (&CandidateAdmissionTicket{}).Diagnostic(); err == nil {
		t.Fatal("zero diagnostic")
	}
	if err := json.Unmarshal([]byte(`{}`), &CandidateAdmissionTicket{}); err != ErrAdmissionCandidate {
		t.Fatal("JSON restored ticket state")
	}
}

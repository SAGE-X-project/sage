package registry011

import (
	"math"
	"strconv"
)

// CandidateAdmissionTicket is a proposed bounded reservation record. It is not
// CandidateBudgetTicket, a durable reservation, a one-use permit or an RPC API.
// Parsing and predecessor comparison never debit an actual operating ledger.
type CandidateAdmissionTicket struct {
	raw, digest                                 string
	intent                                      *CandidateAdmissionIntent
	sequence, segment, live, retained, deadline uint64
	previous, kind, input, method, params       string
	work                                        candidateWorkVector
	policy                                      string
}

type CandidateTicketDiagnostic struct {
	Digest, Admission, Previous, Kind, Input, Method, Params                    string
	Sequence, Segment, LiveReservedUpper, RetainedReservedUpper, DeadlineWallMS uint64
	Reservation                                                                 CandidateWork
}

func (*CandidateAdmissionTicket) UnmarshalJSON([]byte) error { return ErrAdmissionCandidate }

// ParseCandidateAdmissionTicket validates only proposed TicketV2 data. Segment
// identity is derived from the immutable admission and explicit segment ordinal.
// Actual cumulative accounting, request bytes/params, worst-case estimator,
// custody, metadata costs and operation consumption remain unqualified.
func ParseCandidateAdmissionTicket(policy *ProviderPolicyCandidate, intent *CandidateAdmissionIntent, segment uint64, raw []byte, maximumBytes int) (*CandidateAdmissionTicket, error) {
	return parseCandidateAdmissionTicketObserved(policy, intent, segment, raw, maximumBytes, nil)
}

func parseCandidateAdmissionTicketObserved(policy *ProviderPolicyCandidate, intent *CandidateAdmissionIntent, segment uint64, raw []byte, maximumBytes int, meter *providerWorkMeter) (*CandidateAdmissionTicket, error) {
	if policy == nil || policy.raw == "" || intent == nil || intent.raw == "" || policy.policy != intent.identity.Policy || maximumBytes <= 0 || maximumBytes > 64<<10 {
		return nil, ErrConfiguration
	}
	if len(raw) == 0 || len(raw) > maximumBytes {
		return nil, ErrAdmissionCandidate
	}
	m := contentObject(providerReplayJSON(raw, meter), "format", "admission_digest", "namespace", "instance_key", "source_id", "generation", "fence", "sequence", "previous_ticket", "segment", "kind", "input_digest", "method", "params_digest", "reservation", "live_disk_reserved", "retained_bytes_reserved", "deadline_wall_ms")
	id := intent.identity
	if m == nil || m["format"] != "sage-provider-budget-ticket-v2" || m["admission_digest"] != intent.digest || m["namespace"] != id.Namespace || m["instance_key"] != id.Instance || m["source_id"] != id.SourceID || !providerCandidateDigest(m["input_digest"]) {
		return nil, ErrAdmissionCandidate
	}
	generation, a := providerCandidateUint(m["generation"], false)
	fence, b := providerCandidateUint(m["fence"], true)
	sequence, c := providerCandidateUint(m["sequence"], false)
	live, d := providerCandidateUint(m["live_disk_reserved"], true)
	// Every ticket also retains consumption/ACK metadata, including local work.
	retained, e := providerCandidateUint(m["retained_bytes_reserved"], true)
	deadline, f := providerCandidateUint(m["deadline_wall_ms"], true)
	if !a || !b || !c || !d || !e || !f || generation != id.Generation || fence != intent.fence || deadline > intent.deadline || deadline <= uint64(intent.clock.WallMS-intent.clock.UncertaintyMS) || retained > live {
		return nil, ErrAdmissionCandidate
	}
	if sequence == 0 && m["previous_ticket"] != nil || sequence != 0 && !providerCandidateDigest(m["previous_ticket"]) {
		return nil, ErrAdmissionCandidate
	}
	p, policyOK := providerReplayJSON([]byte(policy.raw), meter).(map[string]any)
	if !policyOK {
		return nil, ErrAdmissionCandidate
	}
	limits := p["limits"].(map[string]any)
	segments, _ := providerCandidateUint(limits["bootstrap"].(map[string]any)["segments"], true)
	liveLimit, _ := providerCandidateUint(limits["live_disk_bytes"], true)
	retainedLimit, _ := providerCandidateUint(limits["retained_bytes"], true)
	segmentDigest := candidateAdmissionSegmentObserved(intent.digest, segment, meter)
	if segment >= segments || live > liveLimit || retained > retainedLimit || segmentDigest == "" || m["segment"] != segmentDigest {
		return nil, ErrAdmissionCandidate
	}
	work, ok := candidateAdmissionWork(m["reservation"])
	if !ok || work[8] > 1 || work[9] == 0 || work[10] == 0 {
		return nil, ErrAdmissionCandidate
	}
	op, ok := candidateAdmissionWorkLimit(limits["operation"])
	if !ok || !candidateFits(candidateBudgetTotals{}, work, op) {
		return nil, ErrAdmissionCandidate
	}
	if sequence == 0 && (segment != 0 || work[8] != 1) {
		return nil, ErrAdmissionCandidate
	}
	kind := contentString(m["kind"])
	switch kind {
	case "rpc-capture":
		if work[0] != 1 || work[1] == 0 || !candidateAdmissionReadMethod(contentString(m["method"])) || !providerCandidateDigest(m["params_digest"]) {
			return nil, ErrAdmissionCandidate
		}
	case "local-work":
		if work[0] != 0 || m["method"] != nil || m["params_digest"] != nil {
			return nil, ErrAdmissionCandidate
		}
	default:
		return nil, ErrAdmissionCandidate
	}
	digest := providerReplayDigest("sage-provider-budget-ticket-v2", raw, meter)
	retainedRaw := providerReplayString(raw, meter)
	if digest == "" || retainedRaw == "" {
		return nil, ErrAdmissionCandidate
	}
	return &CandidateAdmissionTicket{retainedRaw, digest, intent, sequence, segment, live, retained, deadline, contentString(m["previous_ticket"]), kind, contentString(m["input_digest"]), contentString(m["method"]), contentString(m["params_digest"]), work, policy.raw}, nil
}

func candidateAdmissionSegment(admission string, index uint64) string {
	return candidateAdmissionSegmentObserved(admission, index, nil)
}

func candidateAdmissionSegmentObserved(admission string, index uint64, meter *providerWorkMeter) string {
	raw := providerReplayMarshal(map[string]string{"admission_digest": admission, "index": strconv.FormatUint(index, 10)}, 4096, meter)
	if len(raw) == 0 {
		return ""
	}
	return providerReplayDigest("sage-provider-budget-segment-v1", raw, meter)
}

func candidateAdmissionWork(value any) (candidateWorkVector, bool) {
	names := [...]string{"requests", "wire_bytes", "decoded_bytes", "blocks", "transactions", "frames", "logs", "proofs", "segments", "disk_bytes_written", "work_units"}
	m := contentObject(value, names[:]...)
	var work candidateWorkVector
	if m == nil {
		return work, false
	}
	for i, name := range names {
		n, ok := providerCandidateUint(m[name], false)
		if !ok {
			return candidateWorkVector{}, false
		}
		work[i] = n
	}
	return work, true
}
func candidateAdmissionWorkLimit(value any) (candidateWorkVector, bool) {
	names := []string{"requests", "wire_bytes", "decoded_bytes", "blocks", "transactions", "frames", "logs", "proofs", "segments", "disk_bytes_written", "work_units", "deadline_ms"}
	m := contentObject(value, names...)
	if m == nil {
		return candidateWorkVector{}, false
	}
	work := make(map[string]any, len(names)-1)
	for _, name := range names[:len(names)-1] {
		work[name] = m[name]
	}
	return candidateAdmissionWork(work)
}
func candidateAdmissionReadMethod(method string) bool {
	switch method {
	case "eth_chainId", "web3_clientVersion", "eth_getBlockByNumber", "eth_getBlockByHash", "eth_getCode", "eth_call", "eth_getBlockReceipts", "debug_getRawBlock", "debug_getRawReceipts", "debug_traceBlockByHash", "debug_traceTransaction":
		return true
	}
	return false
}

// CheckCandidateTicketSuccessor checks an exact data chain, not admission or
// debit. Segment ordinals may stay or advance exactly1; bootstrap cannot reset.
func CheckCandidateTicketSuccessor(before, after *CandidateAdmissionTicket) error {
	if before == nil || after == nil || before.raw == "" || after.raw == "" || before.intent.digest != after.intent.digest || before.sequence == math.MaxUint64 || after.sequence != before.sequence+1 || after.previous != before.digest {
		return ErrAdmissionCandidate
	}
	if after.segment == before.segment {
		if after.work[8] != 0 {
			return ErrAdmissionCandidate
		}
		return nil
	}
	if before.segment == math.MaxUint64 || after.segment != before.segment+1 || after.work[8] != 1 {
		return ErrAdmissionCandidate
	}
	return nil
}
func (t *CandidateAdmissionTicket) CanonicalBytes() []byte {
	if t == nil {
		return nil
	}
	return []byte(t.raw)
}
func (t *CandidateAdmissionTicket) Diagnostic() (CandidateTicketDiagnostic, error) {
	if t == nil || t.raw == "" {
		return CandidateTicketDiagnostic{}, ErrAdmissionCandidate
	}
	return CandidateTicketDiagnostic{t.digest, t.intent.digest, t.previous, t.kind, t.input, t.method, t.params, t.sequence, t.segment, t.live, t.retained, t.deadline, t.work.diagnostic()}, nil
}

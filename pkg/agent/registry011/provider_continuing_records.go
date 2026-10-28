package registry011

import "strconv"

// Format a next TicketV2 under the caller's NEW metadata observer. This does
// not hold P, consume G, retain originals, or confer dispatch permission.
// The concrete owner must perform those separate steps in the closed order.
func providerContinuingTicketData(policy *ProviderPolicyCandidate, intent *CandidateAdmissionIntent, previous *CandidateAdmissionTicket, segment uint64, kind, input, method, params string, reservation providerOperationReservation, clock CandidateClockDiagnostic, meter *providerWorkMeter) (*CandidateAdmissionTicket, error) {
	if policy == nil || intent == nil || previous == nil || previous.intent == nil || previous.intent.digest != intent.digest ||
		previous.sequence == ^uint64(0) || previous.segment == ^uint64(0) || segment < previous.segment || segment > previous.segment+1 ||
		!lowerHash(previous.digest) || !lowerHash(input) || policy.policy != intent.identity.Policy ||
		!candidateLedgerClockAdvance(intent.clock, clock) || candidateLedgerExpired(clock, intent.deadline) ||
		meter == nil || meter.guardian == nil || meter.guardian.phase != providerGuardianRecorded || !meter.guardian.fits(meter) {
		return nil, errProviderIntegration
	}
	total, err := providerFrontierWorkTotal(reservation.execution, reservation.remaining)
	segments := uint64(0)
	if segment == previous.segment+1 {
		segments = 1
	}
	if err != nil || total != reservation.work || reservation.live == 0 || reservation.live != reservation.retained || total[8] != segments ||
		reservation.remaining[0] != 0 || reservation.remaining[1] != 0 || reservation.remaining[8] != 0 {
		return nil, errProviderIntegration
	}
	var methodValue, paramsValue any
	switch kind {
	case "rpc-capture":
		if total[0] != 1 || total[1] == 0 || !candidateAdmissionReadMethod(method) || !lowerHash(params) {
			return nil, errProviderIntegration
		}
		methodValue, paramsValue = method, params
	case "local-work":
		if total[0] != 0 || total[1] != 0 || method != "" || params != "" {
			return nil, errProviderIntegration
		}
	default:
		return nil, errProviderIntegration
	}
	p, ok := providerReplayJSON(providerReplayPhaseBytes(policy.raw, meter), meter).(map[string]any)
	limits, limitsOK := p["limits"].(map[string]any)
	operation, operationOK := limits["operation"].(map[string]any)
	duration, durationOK := providerCandidateUint(operation["deadline_ms"], true)
	deadline, deadlineOK := providerUpperSum(uint64(clock.WallMS-clock.UncertaintyMS), duration)
	segmentDigest := candidateAdmissionSegmentObserved(intent.digest, segment, meter)
	if !ok || !limitsOK || !operationOK || !durationOK || !deadlineOK || !lowerHash(segmentDigest) {
		return nil, errProviderIntegration
	}
	id := intent.identity
	raw := providerReplayCanonical(map[string]any{"format": "sage-provider-budget-ticket-v2", "admission_digest": intent.digest, "namespace": id.Namespace, "instance_key": id.Instance, "source_id": id.SourceID, "generation": strconv.FormatUint(id.Generation, 10), "fence": strconv.FormatUint(intent.fence, 10), "sequence": strconv.FormatUint(previous.sequence+1, 10), "previous_ticket": previous.digest, "segment": segmentDigest, "kind": kind, "input_digest": input, "method": methodValue, "params_digest": paramsValue, "reservation": candidateLedgerWorkObject(reservation.work), "live_disk_reserved": strconv.FormatUint(reservation.live, 10), "retained_bytes_reserved": strconv.FormatUint(reservation.retained, 10), "deadline_wall_ms": strconv.FormatUint(min(deadline, intent.deadline), 10)}, 64<<10, meter)
	return parseCandidateAdmissionTicketObserved(policy, intent, segment, raw, 64<<10, meter)
}

func providerContinuingConsumptionData(ticket *CandidateAdmissionTicket, previous *CandidateConsumption, clock CandidateClockDiagnostic, meter *providerWorkMeter) (*CandidateConsumption, error) {
	if ticket == nil || ticket.intent == nil || ticket.sequence == 0 || previous == nil || previous.ticket == nil || previous.ticket.intent == nil ||
		previous.sequence == ^uint64(0) || previous.ticket.intent.digest != ticket.intent.digest || !lowerHash(previous.digest) ||
		meter == nil || meter.integration == nil || meter.closed || meter.failed ||
		!candidateLedgerClockAdvance(previous.clock, clock) || !candidateLedgerClockAdvance(ticket.intent.clock, clock) || candidateLedgerExpired(clock, ticket.deadline) {
		return nil, errProviderIntegration
	}
	id := ticket.intent.identity
	raw := providerReplayCanonical(map[string]any{"format": "sage-provider-budget-consumption-v1", "ticket_digest": ticket.digest, "namespace": id.Namespace, "instance_key": id.Instance, "source_id": id.SourceID, "generation": strconv.FormatUint(id.Generation, 10), "fence": strconv.FormatUint(ticket.intent.fence, 10), "sequence": strconv.FormatUint(previous.sequence+1, 10), "previous_consumption": previous.digest, "kind": ticket.kind, "input_digest": ticket.input, "clock_start": providerDiagnosticClockObject(clock)}, candidateLedgerMetadataLimit, meter)
	return parseCandidateConsumptionObserved(ticket, raw, candidateLedgerMetadataLimit, meter)
}

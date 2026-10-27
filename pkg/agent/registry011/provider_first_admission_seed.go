package registry011

import (
	"math"
	"strconv"
)

// A historical replay view may read the exact same immutable roots/config as
// its active concrete owner. It never substitutes its provisional maps for
// that owner or installs a new meter, current pin or retaining principal.
func providerAdmissionOriginalOwner(s *providerCustodyStore, m *providerWorkMeter) *providerCustodyStore {
	if s == nil || m == nil || m.owner != m || m.closed || m.failed || s.activeMeter.Load() != m {
		return nil
	}
	var owner *providerCustodyStore
	if m.scan != nil && m.integration == nil {
		owner = m.scan.store
	} else if m.integration != nil && m.scan == nil {
		owner = m.integration.store
	}
	if !owner.valid() || owner.activeMeter.Load() != m || owner.custody != s.custody || owner.evidence != s.evidence || owner.clock != s.clock || owner.config != s.config {
		return nil
	}
	return owner
}

// Construct immutable proposed I bytes with the current observer. Inputs and
// output remain candidate DATA: only the future concrete first transaction
// can bind them to its original F/pins and complete prepaid E1 persistence.
// Nil-meter pure DATA construction cannot select an owner or acknowledge IO.
func providerFirstAdmissionIntentDataObserved(policy *ProviderPolicyCandidate, context *ProviderCandidateContext, anchor *CandidateEnrollmentRoot, authority string, clock CandidateClockDiagnostic, meter *providerWorkMeter) (*CandidateAdmissionIntent, error) {
	if policy == nil || policy.raw == "" || len(policy.raw) > 64<<10 || context == nil || anchor == nil || !lowerHash(authority) || len(clock.ClockID) == 0 || len(clock.ClockID) > 256 || !lowerHash(clock.Epoch) || !candidateLedgerClockAdvance(clock, clock) || clock.WallMS < clock.UncertaintyMS {
		return nil, errProviderIntegration
	}
	id, err := context.Identity()
	if err != nil || id.Policy != policy.policy || anchor.identity != id || anchor.sequence != 0 || anchor.budget != "" || anchor.status != "enrolling" || meter != nil && !meter.preflight(providerEncodeEvent, 64<<10) {
		return nil, errProviderIntegration
	}
	for _, digest := range []string{id.Policy, id.Namespace, id.Instance, id.SourceID, id.Provider, anchor.digest} {
		if !lowerHash(digest) {
			return nil, errProviderIntegration
		}
	}
	p, ok := providerReplayJSON(providerReplayPhaseBytes(policy.raw, meter), meter).(map[string]any)
	limits, limitsOK := p["limits"].(map[string]any)
	bootstrap, bootstrapOK := limits["bootstrap"].(map[string]any)
	duration, durationOK := providerCandidateUint(bootstrap["deadline_ms"], true)
	deadline, deadlineOK := providerUpperSum(uint64(clock.WallMS-clock.UncertaintyMS), duration)
	if !ok || !limitsOK || !bootstrapOK || !durationOK || !deadlineOK || deadline > math.MaxInt64 || !providerCandidateLimits(limits) || !providerCandidateClock(p["clock"]) {
		return nil, errProviderIntegration
	}
	limitsRaw := providerReplayMarshal(limits, 64<<10, meter)
	limitsDigest := providerReplayDigest("", limitsRaw, meter)
	joined := providerReplayJoin([]string{authority, id.Instance}, meter)
	admissionID := providerReplayDigest("sage-provider-admission-id-v1", providerReplayPhaseBytes(joined, meter), meter)
	if len(limitsRaw) == 0 || limitsDigest == "" || joined == "" || admissionID == "" {
		return nil, errProviderIntegration
	}
	raw := providerReplayCanonical(map[string]any{"format": "sage-provider-admission-intent-v1", "policy": id.Policy, "namespace": id.Namespace, "instance_key": id.Instance, "source_id": id.SourceID, "generation": "0", "fence": strconv.FormatUint(anchor.fence, 10), "admission_id": admissionID, "enrollment_anchor_root": anchor.digest, "prior_published_checkpoint": nil, "initial_clock": providerDiagnosticClockObject(clock), "deadline_wall_ms": strconv.FormatUint(deadline, 10), "limits_digest": limitsDigest, "authority_metadata_root": authority}, 64<<10, meter)
	return parseCandidateAdmissionIntentObserved(policy, context, anchor, raw, 64<<10, meter)
}

// Format proposed first-ticket DATA from a separate prospective quote. This
// does not establish that quote's complete work/physical qualification, hold
// resources, consume a request, or authorize IO. The future concrete owner must
// derive its own quote and bind the original pre-E1 sample before calling it.
func providerFirstAdmissionTicketDataObserved(policy *ProviderPolicyCandidate, intent *CandidateAdmissionIntent, request *providerIntegrationRequest, reservation providerOperationReservation, clock CandidateClockDiagnostic, meter *providerWorkMeter) (*CandidateAdmissionTicket, error) {
	if policy == nil || policy.raw == "" || len(policy.raw) > 64<<10 || intent == nil || intent.raw == "" || len(intent.raw) > 64<<10 || !lowerHash(intent.digest) || policy.policy != intent.identity.Policy || request == nil || len(request.method) > 64 || len(request.params) > 8<<10 || len(request.body) > 16<<10 || len(request.head) > 512 || len(request.wire) > 16<<10 || !lowerHash(request.inputDigest) || !lowerHash(request.paramsDigest) || !candidateLedgerClockAdvance(intent.clock, clock) || candidateLedgerExpired(clock, intent.deadline) {
		return nil, errProviderIntegration
	}
	id := intent.identity
	for _, digest := range []string{id.Policy, id.Namespace, id.Instance, id.SourceID} {
		if !lowerHash(digest) {
			return nil, errProviderIntegration
		}
	}
	total, err := providerFrontierWorkTotal(reservation.execution, reservation.remaining)
	if err != nil || total != reservation.work || reservation.live == 0 || reservation.retained != reservation.live || total[0] != 1 || total[8] != 1 || total[1] <= uint64(len(request.wire)) || reservation.remaining[0] != 0 || reservation.remaining[1] != 0 || reservation.remaining[8] != 0 || meter != nil && !meter.preflight(providerEncodeEvent, 64<<10) {
		return nil, errProviderIntegration
	}
	for coordinate := 3; coordinate <= 7; coordinate++ {
		if total[coordinate] != 0 {
			return nil, errProviderIntegration
		}
	}
	if !request.firstFinalizedObserved(meter) {
		return nil, errProviderIntegration
	}
	input, ok := providerIntegrationObjectDigestObserved("request", providerReplayPhaseBytes(request.wire, meter), meter)
	params := providerReplayDigest("sage-provider-params-v1", providerReplayPhaseBytes(request.params, meter), meter)
	if !ok || input != request.inputDigest || params == "" || params != request.paramsDigest {
		return nil, errProviderIntegration
	}
	p, policyOK := providerReplayJSON(providerReplayPhaseBytes(policy.raw, meter), meter).(map[string]any)
	limits, limitsOK := p["limits"].(map[string]any)
	operation, operationOK := limits["operation"].(map[string]any)
	duration, durationOK := providerCandidateUint(operation["deadline_ms"], true)
	deadline, deadlineOK := providerUpperSum(uint64(clock.WallMS-clock.UncertaintyMS), duration)
	if !policyOK || !limitsOK || !operationOK || !durationOK || !deadlineOK || deadline > math.MaxInt64 || !providerCandidateLimits(limits) || !providerCandidateClock(p["clock"]) {
		return nil, errProviderIntegration
	}
	segment := candidateAdmissionSegmentObserved(intent.digest, 0, meter)
	if segment == "" {
		return nil, errProviderIntegration
	}
	raw := providerReplayCanonical(map[string]any{"format": "sage-provider-budget-ticket-v2", "admission_digest": intent.digest, "namespace": id.Namespace, "instance_key": id.Instance, "source_id": id.SourceID, "generation": "0", "fence": strconv.FormatUint(intent.fence, 10), "sequence": "0", "previous_ticket": nil, "segment": segment, "kind": "rpc-capture", "input_digest": input, "method": request.method, "params_digest": params, "reservation": candidateLedgerWorkObject(reservation.work), "live_disk_reserved": strconv.FormatUint(reservation.live, 10), "retained_bytes_reserved": strconv.FormatUint(reservation.retained, 10), "deadline_wall_ms": strconv.FormatUint(min(deadline, intent.deadline), 10)}, 64<<10, meter)
	return parseCandidateAdmissionTicketObserved(policy, intent, 0, raw, 64<<10, meter)
}

// The first typed admit must contain exactly one semantically referenced
// binding. E0 is the already independently replayed prefix boundary, not the
// later live HEAD or producer maps. Original scan controls/chunks supply the
// seed before the first ticket append. No ACK or dispatch follows this DATA.
func (r *providerCustodyReplay) admissionScanSeed(s *providerCustodyStore) (*providerAdmissionControlData, error) {
	if r == nil || s == nil || r.intent == nil || r.anchor == nil || r.ledger != nil || r.frontier == nil || r.frontier.body != r || r.frontier.owner != r.frontier || r.frontier.meter != r.meter || r.frontier.indexes == nil || s.config.bounds.firstAdmission == nil || s.sequence != 1 || len(s.objects) == 0 || len(s.objects) > s.config.bounds.maximumObjects {
		return nil, errProviderIntegration
	}
	owner := providerAdmissionOriginalOwner(s, r.meter)
	if owner == nil {
		return nil, errProviderIntegration
	}
	var digest string
	for key, ref := range s.objects {
		if !r.meter.event(providerCompareEvent, uint64(len(ref.kind))) {
			return nil, errProviderIntegration
		}
		if ref.kind == "scan-admission-binding" {
			want, ok := providerReplayRefKey(ref, r.meter)
			if digest != "" || !ok || key != want {
				return nil, errProviderIntegration
			}
			digest = ref.digest
		}
	}
	raw, readErr := s.object("scan-admission-binding", digest)
	x, codecErr := newProviderScanCodec(s.config, r.meter)
	if readErr != nil || codecErr != nil {
		return nil, errProviderIntegration
	}
	binding, err := x.parseAdmissionBinding(raw)
	f := r.frontier.current
	frontier, frontierErr := f.bytesObserved(r.meter)
	target := providerScanTarget{setup: f.setup, custody: s.tip, frontier: binding.frontier, objects: s.objectsRoot, enrollment: s.enrollment}
	targetRaw, targetErr := x.target(target)
	if err != nil || frontierErr != nil || targetErr != nil || f.sequence != 1 || f.commit != s.tip || f.enrollment != s.enrollment || f.enrollment != r.anchor.digest || f.objects != s.objectsRoot || f.admission != "" || f.budget != "" || f.checkpoint != "" || f.initial != providerClockDiagnostic(s.setupOrigin) || binding.e0 != s.tip || binding.setup != f.setup || binding.frontier != x.hash("sage-provider-custody-frontier-v1", frontier) || binding.target != x.hash("sage-provider-scan-target-v1", targetRaw) || binding.admission != r.intent.digest || binding.clock != r.intent.clock || binding.deadline != r.intent.deadline {
		return nil, errProviderIntegration
	}
	// The historical immutable high-water is selected by this verified E0
	// boundary. Reading HEAD here would incorrectly select a later prefix.
	h, _, highwaterErr := s.originalHighwaterObserved(1, r.meter)
	if highwaterErr != nil || h.ordinal != 1 || h.sequence != 1 || h.commit != binding.e0 || h.frontier != binding.frontier || h.setup != binding.setup || h.fence != r.intent.fence {
		return nil, errProviderIntegration
	}
	controls, err := replayProviderAdmissionControlData(owner, r.meter, binding)
	if err != nil {
		return nil, errProviderIntegration
	}
	receipts, err := replayProviderAdmissionReceipts(owner, r.meter, binding)
	if err != nil || receipts.records != h.ordinal+1 {
		return nil, errProviderIntegration
	}
	metadata, ok := providerUpperSum(controls.metadata, receipts.metadata)
	common, commonOK := providerUpperSum(metadata, r.frontier.preI.metadata, r.frontier.indexes.schemaBytes)
	if !ok || !commonOK || common > s.config.bounds.continuation.maximumMetadataBytes || common > s.config.bounds.continuation.pageMetadataBytes {
		return nil, errProviderIntegration
	}
	return &providerAdmissionControlData{controls.seed, metadata}, nil
}

package registry011

// This is the coordinator's proposed immutable delta, never a retained E1,
// complete prospective resource plan, custody acknowledgment or dispatch token.
// Its maps must not be installed as the independent custodian's evidence.
type providerFirstAdmissionProposal struct {
	intent                  *CandidateAdmissionIntent
	request                 *providerIntegrationRequest
	ticket                  *CandidateAdmissionTicket
	root                    *CandidateEnrollmentRoot
	ledger                  *CandidateAdmissionLedger
	indexes                 *providerLedgerIndexes
	delta                   *providerIndexDelta
	binding                 providerScanAdmissionBinding
	fresh                   []providerIntegrationObject
	inventory               *providerIntegrationInventory
	claim, commit, frontier []byte
	after                   providerFrontierHighwater
}

func (*providerFirstAdmissionProposal) UnmarshalJSON([]byte) error { return errProviderIntegration }

// Construct the complete first-admit proposal with the current observer and
// newly derived coordinator maps. The prospective operation quote and clock
// argument remain DATA; this helper does not qualify them, spend a physical
// claim or write any file. A future live coordinator must first compare its
// complete F work/RAM/clock plan and independently derive the future-only quote.
// The caller holds the original store mutex throughout all original reads.
func providerFirstAdmissionProposalData(f *providerFirstAdmissionOwner, input *providerScanReplayReader, reservation providerOperationReservation, clock CandidateClockDiagnostic) (*providerFirstAdmissionProposal, error) {
	return providerFirstAdmissionProposalWithRequestData(f, input, reservation, clock, nil)
}

// Only the live future owner can supply the once-observed fixed request before
// I. The original DATA-only entry constructs it at its original position.
func providerFirstAdmissionProposalWithRequestData(f *providerFirstAdmissionOwner, input *providerScanReplayReader, reservation providerOperationReservation, clock CandidateClockDiagnostic, request *providerIntegrationRequest) (*providerFirstAdmissionProposal, error) {
	if f == nil || f.store == nil || f.state != providerFirstAdmissionStarted || !f.fits(candidateWorkVector{}) || input == nil || !input.valid() || input.control != f.control || input.meter != f.meter || input.selected != f.mainPin || input.view.sequence != 1 || input.replay == nil || input.replay.owner != input.replay || input.replay.body == nil || input.replay.setupOrdinal != 0 || input.replay.body.anchor == nil || input.replay.body.intent != nil || input.replay.body.ledger != nil || input.replay.current.admission != "" || input.replay.current.budget != "" || !candidateLedgerClockAdvance(providerClockDiagnostic(f.inspected), clock) || candidateLedgerExpired(clock, f.attempt.deadline) {
		return nil, errProviderIntegration
	}
	s, m, body := f.store, f.meter, input.replay.body
	if s.config.bounds.maximumEntries < 14 || len(input.view.objects) != 7 || s.config.bounds.maximumObjects-len(input.view.objects) < 14 {
		return nil, errProviderIntegration
	}
	i, err := providerFirstAdmissionIntentDataObserved(s.config.policy, body.context, body.anchor, f.mainPin.commit, providerClockDiagnostic(f.origin), m)
	var requestErr error
	if request == nil {
		request, requestErr = newProviderIntegrationRequestObserved("eth_getBlockByNumber", []any{"finalized", false}, m)
	} else if f.operation == nil || f.operation.request != request || !f.operation.selectedLocked() || reservation != f.operation.plan.reservation {
		return nil, errProviderIntegration
	}
	if err != nil || requestErr != nil {
		return nil, errProviderIntegration
	}
	ticket, err := providerFirstAdmissionTicketDataObserved(s.config.policy, i, request, reservation, clock, m)
	x, codecErr := newProviderScanCodec(s.config, m)
	total, totalOK := providerScanUpperSum(providerScanOpenUpper(s.config), f.scanPin.held)
	if err != nil || codecErr != nil || !totalOK {
		return nil, errProviderIntegration
	}
	binding := providerScanAdmissionBinding{e0: f.mainPin.commit, frontier: f.mainPin.frontier, setup: f.mainPin.setup, attempt: f.scanPin.intent, target: f.attempt.target, lastChunk: f.lastChunk, admission: i.digest, ordinal: f.scanPin.ordinal, deadline: f.attempt.deadline, live: total.retained, clock: providerClockDiagnostic(f.origin), open: providerScanOpenUpper(s.config), held: f.scanPin.held, total: total}
	// The exact retained receipt head/highwater name the selected Hk, not the
	// predecessor HEAD named by the attempt. Re-encode that immutable Hk here.
	highwater, highwaterErr := x.highwater(f.scanPin)
	head, headErr := x.head(f.scanPin)
	binding.head, binding.highwater = x.hash(providerScanHeadDomain, head), x.hash(providerScanHighwaterDomain, highwater)
	bound, bindingErr := x.admissionBinding(binding)
	if highwaterErr != nil || headErr != nil || bindingErr != nil {
		return nil, errProviderIntegration
	}
	controls, err := replayProviderAdmissionControlData(s, m, binding)
	receipts, receiptErr := replayProviderAdmissionReceipts(s, m, binding)
	if err != nil || receiptErr != nil || receipts.records != f.records {
		return nil, errProviderIntegration
	}
	metadata, metadataOK := providerUpperSum(controls.metadata, receipts.metadata, input.replay.preI.metadata)
	if !metadataOK {
		return nil, errProviderIntegration
	}
	ledger, err := newProviderContinuingLedgerSeeded(s.config.policy, i, ticket, *s.config.bounds.continuation, m, &controls.seed)
	if err != nil {
		return nil, errProviderIntegration
	}
	id := s.config.identity
	rootRaw := providerReplayCanonical(map[string]any{"format": "sage-provider-enrollment-v2", "namespace": id.Namespace, "instance_key": id.Instance, "policy": id.Policy, "source_id": id.SourceID, "generation": "0", "sequence": "1", "previous_root": body.anchor.digest, "status": "enrolling", "checkpoint": nil, "budget_root": ledger.current.digest, "transition_root": nil, "fence": "1"}, candidateLedgerMetadataLimit, m)
	root, err := parseCandidateEnrollmentRootObserved(body.context, rootRaw, candidateLedgerMetadataLimit, m)
	if err != nil {
		return nil, errProviderIntegration
	}
	indexes, _, err := newProviderLedgerIndexesObserved(id, 1, *s.config.bounds.continuation, m)
	if err != nil {
		return nil, errProviderIntegration
	}
	indexes.meter = m
	anchor, anchorErr := newProviderIntegrationObjectObserved("enrollment", providerReplayPhaseBytes(body.anchor.raw, m), candidateLedgerMetadataLimit, m)
	joined := providerReplayJoin([]string{s.config.anchor, id.Instance, id.Policy, id.SourceID}, m)
	enroll := providerReplayDigest("sage-provider-enroll-operation-v1", providerReplayPhaseBytes(joined, m), m)
	initial, err := indexes.prepare("enroll", enroll, 1, anchor, body.anchor, nil)
	if anchorErr != nil || err != nil || indexes.apply(initial) != nil || indexes.heads != input.replay.current.indexes {
		return nil, errProviderIntegration
	}
	prior := input.replay.preI
	prior.metadata = metadata
	if indexes.bindRetainedLedger(ledger, prior) != nil {
		return nil, errProviderIntegration
	}
	fresh := make([]providerIntegrationObject, 0, 14)
	for _, item := range []providerIntegrationObjectInput{{"admission", providerReplayPhaseBytes(i.raw, m)}, {"scan-admission-binding", bound}, {"request", providerReplayPhaseBytes(request.wire, m)}, {"params", providerReplayPhaseBytes(request.params, m)}, {"ticket", providerReplayPhaseBytes(ticket.raw, m)}, {"budget", providerReplayPhaseBytes(ledger.current.raw, m)}, {"enrollment", rootRaw}} {
		object, objectErr := newProviderIntegrationObjectObserved(item.kind, item.raw, 64<<10, m)
		if objectErr != nil {
			return nil, errProviderIntegration
		}
		fresh = append(fresh, object)
	}
	for _, node := range ledger.current.nodes {
		object, objectErr := newProviderIntegrationObjectObserved("ledger-node", providerReplayPhaseBytes(node, m), candidateLedgerMetadataLimit, m)
		if objectErr != nil {
			return nil, errProviderIntegration
		}
		fresh = append(fresh, object)
	}
	delta, err := indexes.prepare("admit", ticket.digest, 2, fresh[4], root, ledger)
	if err != nil || len(delta.mutations) != 3 {
		return nil, errProviderIntegration
	}
	heads := delta.previous
	for _, mutation := range delta.mutations {
		fresh = append(fresh, mutation.object)
		heads[mutation.chain] = mutation.object.digest
	}
	if len(fresh) != 14 {
		return nil, errProviderIntegration
	}
	inventory, err := newProviderFrontierInventoryObserved(input.view.objectsRoot, fresh, s.config.bounds.maximumEntries, min(s.config.bounds.maximumStoredBytes, 64<<20), m)
	previous, previousErr := input.replay.current.bytesObserved(m)
	link := providerFrontierLink{identity: id, fence: 1, sequence: 1, commit: f.mainPin.commit, setup: f.mainPin.setup, enrollment: body.anchor.digest, objects: input.view.objectsRoot, frontier: providerReplayDigest("sage-provider-custody-frontier-v1", previous, m)}
	claim, claimErr := providerFrontierIntentDiagnosticObserved(link, root, inventory, "admit", ticket.digest, clock, s.config.bounds, m)
	commit := providerFrontierCommitObserved(claim, m)
	if err != nil || previousErr != nil || claimErr != nil || len(commit) == 0 {
		return nil, errProviderIntegration
	}
	frontier := input.replay.current
	frontier.sequence, frontier.previous, frontier.commit = 2, link.frontier, providerReplayDigest("sage-provider-custody-commit-v2", commit, m)
	frontier.objects, frontier.enrollment, frontier.indexes = inventory.digest, root.digest, heads
	frontier.admission, frontier.budget, frontier.clock = i.digest, ledger.current.digest, clock
	raw, err := frontier.bytesObserved(m)
	after := f.mainPin
	after.ordinal, after.sequence, after.previous = 2, 2, providerReplayDigest("sage-provider-custody-highwater-v1", providerReplayPhaseBytes(f.mainRaw, m), m)
	after.commit, after.frontier = frontier.commit, providerReplayDigest("sage-provider-custody-frontier-v1", raw, m)
	if err != nil || !after.validObserved(s.config, m) || m.failed {
		return nil, errProviderIntegration
	}
	return &providerFirstAdmissionProposal{intent: i, request: request, ticket: ticket, root: root, ledger: ledger, indexes: indexes, delta: delta, binding: binding, fresh: fresh, inventory: inventory, claim: claim, commit: commit, frontier: raw, after: after}, nil
}

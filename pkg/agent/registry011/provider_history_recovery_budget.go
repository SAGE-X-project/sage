package registry011

func (m *providerHistoryRecoveryInitialModel) applyBody(record providerHistoryRecoveryLegacyRecord, root *CandidateEnrollmentRoot, setup providerSetupReference) (providerIntegrationObject, error) {
	r, body := m.recovery, m.frontier.body
	if record.kind == "enroll" {
		claim, a := m.object("setup", setup.claim)
		commit, b := m.object("setup-commit", setup.commit)
		if a != nil || b != nil || !providerReplayEqual(commit, providerSetupCommitObserved(claim, r.meter), r.meter) || body.anchor != nil || root.budget != "" {
			return providerIntegrationObject{}, errProviderIntegration
		}
		body.anchor = root
		return newProviderIntegrationObjectObserved("enrollment", providerReplayPhaseBytes(root.raw, r.meter), 64<<10, r.meter)
	}
	if body.anchor == nil || root.budget == "" {
		return providerIntegrationObject{}, errProviderIntegration
	}
	budget, err := m.object("budget", root.budget)
	if err != nil {
		return providerIntegrationObject{}, errProviderIntegration
	}
	var change *CandidateAdmissionLedgerChange
	entryKind := ""
	switch record.kind {
	case "admit":
		entryKind = "ticket"
		value, ok := providerReplayJSON(budget, r.meter).(map[string]any)
		if !ok || value["format"] != "sage-provider-budget-root-v2" || body.intent != nil || body.ledger != nil {
			return providerIntegrationObject{}, errProviderIntegration
		}
		raw, err := m.object("admission", contentString(value["admission_digest"]))
		intent, parseErr := parseCandidateAdmissionIntentObserved(body.policy, body.context, body.anchor, raw, 64<<10, r.meter)
		original, originalOK := providerReplayJSON(raw, r.meter).(map[string]any)
		if err != nil || parseErr != nil || !originalOK || original["authority_metadata_root"] != record.before.commit || intent.clock != r.initialDescriptor.attempt.clock {
			return providerIntegrationObject{}, errProviderIntegration
		}
		body.intent = intent
		seed, err := m.scanSeed()
		if err != nil {
			return providerIntegrationObject{}, errProviderIntegration
		}
		raw, err = m.object("ticket", record.operation)
		ticket, parseErr := parseCandidateAdmissionTicketObserved(body.policy, body.intent, 0, raw, 64<<10, r.meter)
		if err != nil || parseErr != nil || ticket.kind != "rpc-capture" || ticket.sequence != 0 || m.request(ticket) != nil {
			return providerIntegrationObject{}, errProviderIntegration
		}
		body.ledger, err = newProviderContinuingLedgerSeeded(body.policy, body.intent, ticket, *r.config.bounds.continuation, r.meter, seed)
		if err != nil {
			return providerIntegrationObject{}, errProviderIntegration
		}
		change, err = body.ledger.Current()
		if err != nil {
			return providerIntegrationObject{}, errProviderIntegration
		}
	case "consume", "settle":
		entryKind = "consumption"
		if record.kind == "settle" {
			entryKind = "settlement"
		}
		raw, err := m.object(entryKind, record.operation)
		value, ok := providerReplayJSON(raw, r.meter).(map[string]any)
		if err != nil || !ok || body.ledger == nil {
			return providerIntegrationObject{}, errProviderIntegration
		}
		state, exists := body.ledger.tickets[contentString(value["ticket_digest"])]
		if !exists || state.ticket == nil {
			return providerIntegrationObject{}, errProviderIntegration
		}
		if record.kind == "consume" {
			consumption, parseErr := parseCandidateConsumptionObserved(state.ticket, raw, 64<<10, r.meter)
			if parseErr != nil {
				return providerIntegrationObject{}, errProviderIntegration
			}
			change, err = body.ledger.AppendCandidateConsumption(consumption)
		} else {
			settlement, parseErr := parseCandidateAdmissionSettlementObserved(state.ticket, raw, 64<<10, r.meter)
			if parseErr != nil || settlement.outcome != "complete" || m.exchange(settlement) != nil || body.firstExchange != "" {
				return providerIntegrationObject{}, errProviderIntegration
			}
			body.firstExchange = settlement.retained
			change, err = body.ledger.AppendCandidateSettlement(settlement)
		}
		if err != nil {
			return providerIntegrationObject{}, errProviderIntegration
		}
	default:
		return providerIntegrationObject{}, errProviderIntegration
	}
	if change == nil || change.digest != root.budget || !providerReplayEqual(budget, providerReplayPhaseBytes(change.raw, r.meter), r.meter) {
		return providerIntegrationObject{}, errProviderIntegration
	}
	for _, original := range change.nodes {
		node := providerReplayPhaseBytes(original, r.meter)
		digest := providerReplayDigest("sage-provider-ledger-node-v1", node, r.meter)
		raw, err := m.object("ledger-node", digest)
		if err != nil || !providerReplayEqual(raw, node, r.meter) {
			return providerIntegrationObject{}, errProviderIntegration
		}
	}
	raw, err := m.object(entryKind, record.operation)
	if err != nil {
		return providerIntegrationObject{}, errProviderIntegration
	}
	return newProviderIntegrationObjectObserved(entryKind, raw, 64<<10, r.meter)
}

func (m *providerHistoryRecoveryInitialModel) scanSeed() (*providerLedgerInitialCharge, error) {
	r, body, scan := m.recovery, m.frontier.body, m.prefix.scan
	raw, err := m.object(providerAdmissionBindingKind(r.config.bounds), scan.bindingOriginal.digest)
	x, codecErr := newProviderScanCodec(r.config, r.meter)
	if err != nil || codecErr != nil || body.intent == nil || body.anchor == nil || body.ledger != nil || m.through != 2 {
		return nil, errProviderIntegration
	}
	binding, err := x.parseAdmissionBinding(raw)
	f, h := m.frontier.current, m.prefix.records[0].after
	frontier, frontierErr := f.bytesObserved(r.meter)
	highwater, highwaterErr := h.bytesObserved(r.config, r.meter)
	target := providerScanTarget{setup: f.setup, custody: f.commit, frontier: binding.frontier, objects: f.objects, enrollment: f.enrollment}
	targetRaw, targetErr := x.target(target)
	if err != nil || binding != scan.binding || frontierErr != nil || highwaterErr != nil || targetErr != nil || f.sequence != 1 || f.enrollment != body.anchor.digest ||
		f.admission != "" || f.budget != "" || f.checkpoint != "" || f.initial != body.intent.clock || binding.e0 != f.commit || binding.setup != f.setup ||
		binding.frontier != x.hash("sage-provider-custody-frontier-v1", frontier) || binding.target != x.hash("sage-provider-scan-target-v1", targetRaw) ||
		binding.admission != body.intent.digest || binding.clock != body.intent.clock || binding.deadline != body.intent.deadline ||
		h.ordinal != 1 || h.sequence != 1 || h.commit != binding.e0 || h.frontier != binding.frontier || h.setup != binding.setup || h.fence != body.intent.fence ||
		scan.result.records != h.ordinal+1 || h != r.initialDescriptor.scanMain ||
		!providerReplayEqual(highwater, providerReplayPhaseBytes(r.initialDescriptor.scanMainRaw, r.meter), r.meter) ||
		!providerReplayEqual(targetRaw, providerReplayPhaseBytes(r.initialDescriptor.targetRaw, r.meter), r.meter) {
		return nil, errProviderIntegration
	}
	total, sumOK := providerScanUpperSum(providerScanOpenUpper(r.config), binding.held)
	if !sumOK || total != binding.total || total.retained != binding.live || !m.initialCapacity(total) {
		return nil, errProviderIntegration
	}
	metadata, metadataOK := providerUpperSum(scan.controlBytes, scan.result.metadata)
	common, commonOK := providerUpperSum(metadata, m.frontier.preI.metadata, m.frontier.indexes.schemaBytes)
	if !metadataOK || !commonOK || common > r.config.bounds.continuation.maximumMetadataBytes || common > r.config.bounds.continuation.pageMetadataBytes {
		return nil, errProviderIntegration
	}
	body.scanMetadata = metadata
	return &providerLedgerInitialCharge{total.work, total.retained}, nil
}

// Same whole-hold capacity rule as the live history path, with a separately
// parsed policy value. This DATA check does not create a retaining principal.
func (m *providerHistoryRecoveryInitialModel) initialCapacity(preI providerScanUpper) bool {
	r := m.recovery
	if r.config.bounds.guardian != nil || r.config.bounds.history == nil || !r.config.bounds.history.valid(r.config.bounds) ||
		!providerHistoryUpperWithin(preI, providerHistoryPreIUpper(r.config.bounds)) {
		return false
	}
	held, ok := providerHistoryWholeHold(r.config.bounds)
	p, policyOK := providerReplayJSON(providerReplayPhaseBytes(m.frontier.body.policy.raw, r.meter), r.meter).(map[string]any)
	limits, limitsOK := p["limits"].(map[string]any)
	work, a := candidateAdmissionWorkLimit(limits["bootstrap"])
	live, b := providerCandidateUint(limits["live_disk_bytes"], true)
	retained, c := providerCandidateUint(limits["retained_bytes"], true)
	if !ok || !policyOK || !limitsOK || !a || !b || !c || held.retained > live || held.retained > retained || held.retained > r.config.bounds.maximumStoredBytes {
		return false
	}
	for coordinate, value := range held.work {
		if value > work[coordinate] {
			return false
		}
	}
	return true
}

func (m *providerHistoryRecoveryInitialModel) oldMaterials(record providerHistoryRecoveryLegacyRecord) error {
	f, r := m.frontier, m.recovery
	for index, head := range f.indexes.heads {
		kind := "index-mutation"
		if f.indexes.counts[index] == 0 {
			kind = "index-empty"
		}
		if _, err := m.object(kind, head); err != nil {
			return errProviderIntegration
		}
	}
	if f.body.anchor != nil {
		if m.exactObject(providerIntegrationObject{kind: "enrollment", digest: f.body.anchor.digest, raw: f.body.anchor.raw}) != nil {
			return errProviderIntegration
		}
	}
	ledger := f.body.ledger
	if ledger == nil {
		return nil
	}
	if !ledger.valid() || f.body.intent == nil || f.body.anchor == nil {
		return errProviderIntegration
	}
	for _, original := range []providerIntegrationObject{{kind: "admission", digest: f.body.intent.digest, raw: f.body.intent.raw},
		{kind: "budget", digest: ledger.current.digest, raw: ledger.current.raw}} {
		if m.exactObject(original) != nil {
			return errProviderIntegration
		}
	}
	for _, head := range ledger.current.heads {
		if _, err := m.object("ledger-node", head); err != nil {
			return errProviderIntegration
		}
	}
	entryKind := "consumption"
	if record.kind == "settle" {
		entryKind = "settlement"
	}
	raw, err := m.object(entryKind, record.operation)
	v, ok := providerReplayJSON(raw, r.meter).(map[string]any)
	state, exists := ledger.tickets[contentString(v["ticket_digest"])]
	if err != nil || !ok || !exists || state.ticket == nil || m.exactObject(providerIntegrationObject{kind: "ticket", digest: state.ticket.digest, raw: state.ticket.raw}) != nil || m.request(state.ticket) != nil {
		return errProviderIntegration
	}
	if state.consumption != nil && m.exactObject(providerIntegrationObject{kind: "consumption", digest: state.consumption.digest, raw: state.consumption.raw}) != nil {
		return errProviderIntegration
	}
	if record.kind == "consume" && ledger.lastConsumption != nil && m.exactObject(providerIntegrationObject{kind: "consumption", digest: ledger.lastConsumption.digest, raw: ledger.lastConsumption.raw}) != nil {
		return errProviderIntegration
	}
	return nil
}

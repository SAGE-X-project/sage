package registry011

import "context"

// Completed original spools can exist before any S record. Only the compact
// independently parsed result digests are retained by this DATA witness.
type providerHistoryRecoveryCompletedMembers struct {
	owner    *providerHistoryRecoveryCompletedMembers
	recovery *providerHistoryRecoveryOwner
	group    *providerHistoryRecoveryGroupModel
	pass     int
	count    int
	results  [providerHistoryGroupMembers]string
	complete bool
}

func (*providerHistoryRecoveryCompletedMembers) UnmarshalJSON([]byte) error {
	return errProviderIntegration
}

func (m *providerHistoryRecoveryCompletedMembers) selectedLocked(r *providerHistoryRecoveryOwner) bool {
	return m != nil && m.owner == m && r != nil && m.recovery == r && m.pass >= 0 && m.pass < len(r.completedMemberModels) &&
		r.completedMemberModels[m.pass] == m && m.complete && m.count == r.completed && m.count >= 0 && m.count <= len(m.results) &&
		m.group != nil && m.group == r.groupModels[m.pass] && m.group.selectedLocked(r) &&
		r.catalogBefore != nil && r.catalogBefore.selectedLocked(r)
}

func (r *providerHistoryRecoveryOwner) replayCompletedMemberPayloadsCandidate(ctx context.Context, pass int) (*providerHistoryRecoveryCompletedMembers, error) {
	if r == nil || r.store == nil {
		return nil, errProviderIntegration
	}
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	if !r.activeLocked(r.meter) || pass < 0 || pass >= len(r.completedMemberModels) || r.completedMemberModels[pass] != nil ||
		r.catalogBefore == nil || !r.catalogBefore.selectedLocked(r) || r.catalogAfter != nil || r.catalogResult != nil ||
		r.groupModels[pass] == nil || !r.groupModels[pass].selectedLocked(r) ||
		pass == 1 && (r.completedMemberModels[0] == nil || !r.completedMemberModels[0].selectedLocked(r)) {
		return nil, errProviderIntegration
	}
	accepted := false
	defer func() {
		if !accepted {
			r.stopLocked(true)
		}
	}()
	if !r.inputReadyLocked(ctx) || r.completed < 0 || r.completed > providerHistoryGroupMembers {
		return nil, errProviderIntegration
	}
	files, ok := providerUpperProduct(uint64(r.completed), 2)
	if !ok || r.inputFiles > r.bounds.maximumFiles || files > r.bounds.maximumFiles-r.inputFiles || !r.meter.event(providerCopyEvent, uint64(r.completed+1)) {
		return nil, errProviderIntegration
	}
	g := r.groupModels[pass]
	roots := ""
	if r.completed > 0 {
		if g.memberCheckpoint == nil || g.memberCheckpoint != r.consumption || g.memberCheckpoint.frame.phase != "consume" ||
			g.memberTicket == nil || g.memberConsumption == nil || g.memberTicket.table.count < r.completed ||
			g.memberTicket.table.stage != g.memberCheckpoint.frame.stage {
			return nil, errProviderIntegration
		}
		policy, policyOK := providerReplayJSON(providerReplayBytes(g.initial.frontier.body.policy.raw, r.meter), r.meter).(map[string]any)
		provider, providerOK := policy["provider"].(map[string]any)
		roots = contentString(provider["tls_roots_artifact"])
		if !policyOK || !providerOK || !lowerHash(roots) {
			return nil, errProviderIntegration
		}
	}
	out := &providerHistoryRecoveryCompletedMembers{recovery: r, group: g, pass: pass, count: r.completed}
	out.owner = out
	for i := range r.completed {
		sealed := r.members[i]
		if sealed == nil || sealed.capture == nil || sealed.capture.intent == nil || sealed.capture.intent.consumption != g.memberCheckpoint ||
			sealed.capture.intent.stage != g.memberTicket.table.stage || sealed.capture.intent.ordinal != i {
			return nil, errProviderIntegration
		}
		member := &g.memberTicket.table.members[i]
		intent, capture := sealed.capture.intent, sealed.capture
		if intent.member.capture != member.capture || !providerReplayTextEqual(intent.member.class, member.class, r.meter) ||
			!providerHistoryGroupRequestEqualObserved(&intent.member.request, &member.request, r.meter) {
			return nil, errProviderIntegration
		}
		response, err := r.readSelectedInputLocked(ctx, providerHistoryRecoveryInputSelection{checkpoint: g.memberCheckpoint, kind: providerHistoryRecoveryMemberResponse, index: i})
		if err != nil || response == nil || response.state != providerHistoryRecoveryWholeInput {
			return nil, errProviderIntegration
		}
		tls, err := r.readSelectedInputLocked(ctx, providerHistoryRecoveryInputSelection{checkpoint: g.memberCheckpoint, kind: providerHistoryRecoveryMemberTLS, index: i})
		if err != nil || tls == nil || tls.state != providerHistoryRecoveryWholeInput || uint64(len(response.raw)) != capture.responseBytes || uint64(len(tls.raw)) != capture.tlsBytes ||
			!providerReplayTextEqual(providerReplayDigest("", response.raw, r.meter), capture.response, r.meter) ||
			!providerReplayTextEqual(providerReplayDigest("", tls.raw, r.meter), capture.tls, r.meter) {
			return nil, errProviderIntegration
		}
		parsed, err := parseProviderHistoryMemberOriginalsObserved(providerReplayString(response.raw, r.meter), providerReplayString(tls.raw, r.meter),
			g.memberTicket, g.memberConsumption, i, roots, r.parent.tls, r.meter)
		if err != nil || !providerReplayTextEqual(parsed.digest, sealed.result, r.meter) {
			return nil, errProviderIntegration
		}
		out.results[i] = providerHistoryRecoveryCloneObserved(parsed.digest, r.meter)
		if out.results[i] == "" {
			return nil, errProviderIntegration
		}
	}
	if !r.inputReadyLocked(ctx) {
		return nil, errProviderIntegration
	}
	out.complete, r.completedMemberModels[pass], accepted = true, out, true
	return out, nil
}

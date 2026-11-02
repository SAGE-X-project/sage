package registry011

import (
	"bufio"
	"bytes"
	"context"
)

// An inspection of every original planned tail slot, including absent slots.
// Partial bytes and an unacknowledged complete capture grant no authority.
type providerHistoryRecoveryTailPayloads struct {
	owner    *providerHistoryRecoveryTailPayloads
	recovery *providerHistoryRecoveryOwner
	group    *providerHistoryRecoveryGroupModel
	members  *providerHistoryRecoveryCompletedMembers
	pending  *providerHistoryRecoveryCheckpoint
	attempt  *providerHistoryRecoveryMemberAttempt
	pass     int
	data     providerHistoryRecoveryTailData
	complete bool
}

func (*providerHistoryRecoveryTailPayloads) UnmarshalJSON([]byte) error {
	return errProviderIntegration
}

func (m *providerHistoryRecoveryTailPayloads) selectedLocked(r *providerHistoryRecoveryOwner) bool {
	return m != nil && m.owner == m && r != nil && m.recovery == r && m.pass >= 0 && m.pass < len(r.tailModels) && r.tailModels[m.pass] == m &&
		m.complete && m.data.count >= 0 && m.data.count <= len(m.data.entries) && m.group == r.groupModels[m.pass] && m.group != nil && m.group.selectedLocked(r) &&
		m.members == r.completedMemberModels[m.pass] && m.members != nil && m.members.selectedLocked(r) && m.pending == r.pending && m.attempt == r.memberAttempt
}

func (r *providerHistoryRecoveryOwner) replayTailPayloadsCandidate(ctx context.Context, pass int) (*providerHistoryRecoveryTailPayloads, error) {
	if r == nil || r.store == nil {
		return nil, errProviderIntegration
	}
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	if !r.activeLocked(r.meter) || pass < 0 || pass >= len(r.tailModels) || r.tailModels[pass] != nil ||
		r.catalogBefore == nil || !r.catalogBefore.selectedLocked(r) || r.catalogAfter != nil || r.catalogResult != nil ||
		r.completedMemberModels[pass] == nil || !r.completedMemberModels[pass].selectedLocked(r) ||
		pass == 1 && (r.tailModels[0] == nil || !r.tailModels[0].selectedLocked(r)) {
		return nil, errProviderIntegration
	}
	accepted := false
	defer func() {
		if !accepted {
			r.stopLocked(true)
		}
	}()
	if !r.inputReadyLocked(ctx) {
		return nil, errProviderIntegration
	}
	var files uint64
	if r.pending != nil {
		if r.pending.inventory.count < 1 || r.pending.inventory.count > len(r.pending.inventory.entries) {
			return nil, errProviderIntegration
		}
		files = uint64(7 + r.pending.inventory.count)
	}
	if r.memberAttempt != nil {
		files += 2 // The count is at most 8+7+2, before any native operation.
	}
	if r.inputFiles > r.bounds.maximumFiles || files > r.bounds.maximumFiles-r.inputFiles ||
		!r.meter.event(providerCopyEvent, providerHistoryRecoveryTailInputs+1) {
		return nil, errProviderIntegration
	}
	out := &providerHistoryRecoveryTailPayloads{recovery: r, group: r.groupModels[pass], members: r.completedMemberModels[pass],
		pending: r.pending, attempt: r.memberAttempt, pass: pass}
	out.owner = out
	out.data.unacknowledged = r.pending != nil || r.memberAttempt != nil
	if r.pending != nil && out.pendingOriginals(ctx) != nil || r.memberAttempt != nil && out.attemptOriginals(ctx) != nil ||
		out.data.count != int(files) || !r.meter.event(providerCopyEvent, providerHistoryRecoveryTailInputs+1) || !r.inputReadyLocked(ctx) {
		return nil, errProviderIntegration
	}
	// The private value seal does not alias the returned DATA array. C1 must
	// compare both independent passes against these original observation seals.
	out.complete, r.tailSeals[pass], r.tailModels[pass], accepted = true, out.data, out, true
	return out, nil
}

func (m *providerHistoryRecoveryTailPayloads) pendingOriginals(ctx context.Context) error {
	r, c, whole := m.recovery, m.pending, true
	for i := range c.inventory.count {
		ref := c.inventory.entries[i]
		domain, ok := providerHistoryStoredDomain(ref.kind)
		if !ok {
			return errProviderIntegration
		}
		_, full, err := m.read(ctx, providerHistoryRecoveryTailPlan{checkpoint: c, kind: providerHistoryRecoveryObject, index: i, bytes: ref.bytes, domain: domain, digest: ref.digest})
		if err != nil {
			return errProviderIntegration
		}
		whole = whole && full
	}
	for _, item := range []struct {
		kind providerHistoryRecoveryFileKind
		raw  string
	}{
		{providerHistoryRecoveryInventory, c.inventory.raw}, {providerHistoryRecoveryIntent, c.frame.intent},
		{providerHistoryRecoveryPending, c.frame.intent}, {providerHistoryRecoveryCommit, c.frame.commit},
		{providerHistoryRecoveryFrontier, c.frame.frontier}, {providerHistoryRecoveryHighwater, c.head},
		{providerHistoryRecoveryHeadPending, c.head},
	} {
		optional := item.kind == providerHistoryRecoveryHeadPending
		_, full, err := m.read(ctx, providerHistoryRecoveryTailPlan{checkpoint: c, kind: item.kind, raw: item.raw, bytes: uint64(len(item.raw)), optional: optional})
		if err != nil {
			return errProviderIntegration
		}
		if !optional {
			whole = whole && full
		}
	}
	if whole && m.group.physical != c || !whole && m.group.physical != r.confirmed {
		return errProviderIntegration
	}
	return nil
}

func (m *providerHistoryRecoveryTailPayloads) attemptOriginals(ctx context.Context) error {
	r, g, n := m.recovery, m.group, m.attempt
	if g.memberCheckpoint == nil || g.memberCheckpoint != n.consumption || n.consumption != r.consumption || g.memberTicket == nil || g.memberConsumption == nil ||
		n.ordinal < 0 || n.ordinal >= g.memberTicket.table.count || n.ordinal >= providerHistoryGroupMembers || n.stage != g.memberTicket.table.stage ||
		n.member.capture != g.memberTicket.table.members[n.ordinal].capture || !providerReplayTextEqual(n.member.class, g.memberTicket.table.members[n.ordinal].class, r.meter) ||
		!providerHistoryGroupRequestEqualObserved(&n.member.request, &g.memberTicket.table.members[n.ordinal].request, r.meter) {
		return errProviderIntegration
	}
	capture := r.memberCapture
	written := n.attempt.files
	var originals [2]*providerHistoryRecoveryInput
	var full [2]bool
	for i, kind := range [2]providerHistoryRecoveryFileKind{providerHistoryRecoveryMemberResponse, providerHistoryRecoveryMemberTLS} {
		plan := providerHistoryRecoveryTailPlan{checkpoint: g.memberCheckpoint, kind: kind, index: n.ordinal, mustAbsent: capture == nil || i > written}
		if capture != nil {
			plan.bytes, plan.digest = capture.responseBytes, capture.response
			if i == 1 {
				plan.bytes, plan.digest = capture.tlsBytes, capture.tls
			}
			plan.mustWhole = i < written
		}
		var err error
		originals[i], full[i], err = m.read(ctx, plan)
		if err != nil {
			return errProviderIntegration
		}
	}
	if capture == nil {
		return nil // No original spool writer was prepared; both must be absent.
	}
	policy, ok := providerReplayJSON(providerReplayBytes(g.initial.frontier.body.policy.raw, r.meter), r.meter).(map[string]any)
	provider, providerOK := policy["provider"].(map[string]any)
	roots := contentString(provider["tls_roots_artifact"])
	if !ok || !providerOK || !lowerHash(roots) {
		return errProviderIntegration
	}
	if full[0] && full[1] {
		result, err := parseProviderHistoryMemberOriginalsObserved(providerReplayString(originals[0].raw, r.meter), providerReplayString(originals[1].raw, r.meter),
			g.memberTicket, g.memberConsumption, n.ordinal, roots, r.parent.tls, r.meter)
		if err != nil {
			return errProviderIntegration
		}
		m.data.attemptResult = providerHistoryRecoveryCloneObserved(result.digest, r.meter)
		if m.data.attemptResult == "" {
			return errProviderIntegration
		}
		return nil // A parsed capture still has no original member completion/ACK.
	}
	m.data.incomplete = true
	if full[0] {
		reader := bufio.NewReader(bytes.NewReader(originals[0].raw))
		if _, _, _, _, err := readProviderHTTPObserved(reader, n.member.capture, r.meter, uint64(len(originals[0].raw))); err != nil {
			return errProviderIntegration
		}
	}
	// The original sequential writer cannot have a whole TLS file without
	// the earlier acknowledged whole response: files0 requires absent TLS,
	// while files1/2 require a whole response before reading the TLS slot.
	return nil
}
